# Design

## Context

Ver `proposal.md` (Why) y los requisitos en
`specs/declarative-write-engine/spec.md`. Estado actual relevante: el motor
`helper/write_engine.go` es genérico sobre el árbol de relaciones (orden
topológico, propagación de llave, inmutabilidad, idempotencia por clave natural).
La política de verbos `WriteVerbPolicy` (`creatable`/`updatable`/`deletable`)
existe en `config/relaciones.go` y se fija en fixtures de prueba, pero el motor no
la consulta en ningún punto. El cambio `config-composicion-objetos-tipados`
(Cambio 0) reforma la configuración a objetos direccionables con árbol efectivo
compuesto y lleva la `policy` a la arista/referencia; este cambio se apila sobre
ese árbol efectivo.

El spec vigente ya tiene un pre-pass de recuperación de solo lectura previo a la
escritura (requisitos "Validación antes de cualquier escritura" y "La recuperación
de validación honra la política de error") y un escenario de entidad existente con
relación incompatible. La restricción que da forma al enfoque: no inventar una ruta
de error ni un recorrido nuevos; los dos controles se deciden dentro del pre-pass
que ya existe.

## Goals / Non-Goals

**Goals:**
- Hacer efectiva la política por relación (`creatable`/`updatable`/`deletable`)
  sobre la acción materializada de cada arista del árbol efectivo.
- Dar al motor una ruta de materialización de borrado genérica, gateada por
  `deletable` y dirigida por el árbol compuesto (nunca inferida por el motor).
- Rechazar el reparent de un hijo existente antes de cualquier escritura, como
  regla explícita de pertenencia.
- Garantizar que los controles se evalúan sobre el árbol completo antes de la
  primera escritura, con identidad de error por fragmento.

**Non-Goals:**
- No introducir un endpoint DELETE de controlador: la baja se materializa solo
  para un nodo que el árbol compuesto marca, por la ruta de escritura existente.
- No decidir *qué* nodo se marca para borrado ni la lógica de negocio que lo
  motiva: eso es del servicio llamante, agnóstico a este cambio.
- No definir el contrato de respuesta ni el manejo de fallo parcial *durante* la
  materialización; eso es de un cambio posterior. Aquí la promesa es de validación
  total previa a toda escritura, no de transacción distribuida.
- No cambiar la forma de configuración: se consume la `policy` por arista que ya
  define el Cambio 0.

## Decisions

### Decisión 1: Ambos controles se deciden en el pre-pass de solo lectura ya existente, sobre el árbol completo
El motor resuelve cada nodo (por `Id` o clave natural, solo lecturas GET) y retiene
su estado para todo el árbol antes de materializar la primera escritura. Con el
estado resuelto clasifica la acción prevista de cada arista (crear si no resuelve a
un registro existente, actualizar si sí) y aplica el gate de política y la regla de
no-reparent. Cualquier violación en cualquier nodo produce `400` sin haber escrito
nada.

Por qué: es la única forma honesta de cumplir "un `400` nunca deja escrituras
parciales". El spec ya exige un pre-pass de solo lectura y una validación previa a
la escritura; extender su alcance a los dos controles nuevos reutiliza esa fase en
vez de abrir una ruta de error paralela. Clasificar la acción requiere conocer si la
entidad existe, lo que el pre-pass ya determina.

Alternativa considerada: gatear nodo por nodo a medida que el motor escribe en orden
topológico. Rechazada: una violación en un nodo tardío dejaría escritos los nodos
previos, contradiciendo la garantía de cero escrituras parciales ante `400`.

### Decisión 2: La acción gateada es la materializada, no el verbo HTTP
`creatable` gatea la creación de un registro nuevo, `updatable` la actualización de
uno existente y `deletable` la eliminación de un nodo que el árbol compuesto marca,
donde "nuevo" vs "existente" lo decide la resolución del pre-pass (incluida la
idempotencia por clave natural) y "marcado para borrado" lo decide el árbol
compuesto, no si el endpoint fue POST o PATCH.

Por qué: el endpoint es un upsert idempotente; el mismo POST puede crear unas
entidades del árbol y reconciliar otras ya existentes. Atar el gate al verbo HTTP
daría un control sin sentido por arista. Atarlo a la acción materializada es lo que
el control por relación pretende.

Alternativa considerada: mapear `creatable`/`updatable` a POST/PATCH del endpoint
raíz. Rechazada: un PATCH que toca un subárbol puede legítimamente crear un hijo
nuevo, y un POST puede reconciliar un padre existente; el verbo no describe la acción
por arista.

### Decisión 3: `deletable` es un control vivo; decisión de negocio en el servicio, mecanismo genérico en el motor
`deletable` se consulta y gatea la eliminación. El motor aporta el **mecanismo**
genérico: materializar la baja de un nodo que el árbol efectivo compuesto marca
para eliminación, gateada por `deletable`, por la ruta de escritura existente. La
**decisión** de qué nodo se marca para borrado es del servicio llamante (lógica de
negocio), que compone el árbol que entrega al motor; el motor nunca infiere una
eliminación a partir de nodos ausentes.

Por qué: el servicio entrega el árbol completo con la intención ya expresada, de
modo que el borrado pasa por el mismo motor gateado y pre-validado que crear y
actualizar, en lugar de abrir un segundo camino de escritura fuera del motor.
Separar decisión (servicio) de mecanismo (motor) mantiene al motor agnóstico del
dominio y da a `deletable` un enforcement real.

Alternativa considerada: dejar `deletable` declarado y no consultado, sin ruta de
borrado en el motor. Rechazada: deja a un consumidor con una transición de baja sin
mecanismo genérico, forzando un segundo camino de escritura en el servicio.

Alternativa considerada: que el motor infiera eliminaciones por diferencia contra
el estado actual del CRUD. Rechazada: convierte una omisión en un borrado silencioso
y contradice la semántica de actualización parcial (omitir no es eliminar).

### Decisión 4: Identidad de error por fragmento; interacción de los dos controles
Cuando un nodo infringe a la vez el gate de política y la regla de no-reparent, el
error nombra el objeto o la arista responsable (identidad por fragmento, coherente
con el Cambio 0), sin prescribir aquí la forma del sobre de respuesta. Un nodo
existente apuntado a otro padre bajo una arista `updatable: false` es tanto un
intento de reparent como una actualización no permitida; cualquiera de las dos
causas es suficiente para el `400` previo a la escritura y el mensaje identifica el
fragmento.

Por qué: la forma del sobre de error (y el fallo parcial durante la materialización)
es territorio del cambio posterior de contrato de respuesta; fijar aquí solo la
identidad por fragmento evita pintar a ese cambio en una esquina y mantiene este
acotado a los dos controles.

## Risks / Trade-offs

- [Retener el árbol resuelto completo antes de escribir aumenta el trabajo de
  lectura previa] → El spec ya exige recuperación de solo lectura para validar
  inmutables y campos derivados; el pre-pass ya recorre el árbol, de modo que
  clasificar la acción y gatear no añade un recorrido nuevo, solo decisiones sobre el
  estado ya recuperado.
- [Clasificar crear vs actualizar puede divergir de la idempotencia por clave
  natural] → La clasificación usa la misma resolución (Id + clave natural) que la
  idempotencia existente, de modo que el gate y el upsert ven el mismo "existe / no
  existe"; las pruebas cubren una arista `updatable: false` sobre una entidad que
  resuelve por clave natural.
- [Confundir la garantía de cero escrituras parciales con transaccionalidad] → La
  garantía es solo que ninguna escritura empieza si la validación falla; un fallo del
  CRUD durante la materialización no está cubierto aquí y pertenece al cambio de
  contrato de respuesta. El spec lo delimita explícitamente.
- [El gate por acción podría rechazar un reenvío sin cambios de una entidad bajo
  `updatable: false`] → La clasificación distingue actualización con cambios de
  reconciliación sin cambios; un reenvío idéntico de un derivado o de campos no
  modificados no cuenta como actualización gateada. Las pruebas cubren el reenvío sin
  cambios como caso permitido.
- [El borrado podría convertirse en una ruta de escritura silenciosa fuera del
  gate] → El motor solo materializa la baja de un nodo que el árbol compuesto marca,
  evalúa `deletable` en el mismo pre-pass que los demás controles y comparte la
  garantía de cero-escrituras-ante-`400`; nunca infiere eliminaciones por diferencia
  contra el CRUD. Las pruebas cubren un borrado no permitido (`400` sin escrituras) y
  la ausencia de inferencia de borrado.
- [El fallo parcial de una transición que combina borrado y creación] → La
  ordenación y la recuperación de una transición de varios pasos son del servicio
  llamante y del cambio posterior de contrato de respuesta; este cambio solo
  garantiza que el gate de borrado se evalúa antes de cualquier escritura.

## Migration Plan

- Despliegue: cambio de comportamiento del motor que consume la `policy` por arista
  ya presente en el árbol efectivo del Cambio 0; no hay migración de datos ni de
  configuración más allá de declarar la política deseada por arista.
- Rollback: al no consultar la política, el motor vuelve al comportamiento previo
  (política inerte); no hay estado persistido que deshacer, solo respuestas `400`
  que dejan de emitirse.
