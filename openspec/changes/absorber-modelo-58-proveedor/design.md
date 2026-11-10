# Design

## Context

Ver `proposal.md` (Why) y los requisitos en `specs/`. Estado actual relevante: el
árbol del proveedor en `config/relaciones_proveedor.yaml` es un marcador de
posición escrito antes de #58; su discriminante (`TipoRegistroId`), varias
relaciones (`Perfil` 1-1 anidado) y las listas de campos de las especializaciones no
corresponden al modelo extendido. El motor de escritura y el de expansión son
genéricos sobre el árbol; este cambio los alimenta con el árbol #58 sobre los objetos
direccionables y la tabla de campos tipada de `config-composicion-objetos-tipados`, y
se apoya en `control-escritura-por-relacion` (política por relación, no-reparent y la
capacidad genérica de eliminación dirigida por el árbol).

Hoy la autoría se toma del contexto de la petición y se inyecta en cada escritura
como un campo que el cuerpo del cliente no puede fijar. Este cambio invierte esa
fuente.

## Goals / Non-Goals

**Goals:**
- Árbol efectivo del proveedor fiel al modelo extendido, en lectura, escritura y
  expansión, por entidad.
- Especialización (`proveedor_natural`/`proveedor_juridico`) como proyección
  aplanada por PK compartida, de un salto, en la dirección proveedor->
  especialización.
- Autoría por cuerpo (`UsuarioModificacionId`) y rechazo de los demás campos de
  auditoría.
- Transición de `estado_proveedor` con baja del activo previo, expresada por el
  servicio y materializada por la capacidad genérica de eliminación del motor.

**Non-Goals:**
- Proyección inversa (expandir desde `proveedor_juridico` y traer los campos del
  proveedor): fuera de alcance.
- Materializar el tercero representante como subárbol de escritura:
  `representacion.representante_id` es una FK plana a profundidad 0; igual para
  `cuenta_bancaria.contacto_id`.
- Definir el sobre de respuesta o el manejo de fallo parcial *durante* la
  materialización (incluida la recuperación de una transición de estado a medio
  camino): pertenece al cambio posterior de contrato de respuesta.
- El mecanismo de la tabla de campos tipada y de objetos direccionables: lo provee
  `config-composicion-objetos-tipados`; aquí solo se puebla.

## Decisions

### Decisión 1: La especialización es una proyección aplanada sobre la arista de PK compartida existente, no una primitiva de herencia
`proveedor_natural`/`proveedor_juridico` ya se unen al proveedor por PK compartida
(`proveedor.id = especializacion.id`). Lo nuevo es una bandera de proyección en esa
arista: al leer, los campos de la especialización se aplanan en el proveedor; al
escribir, un cuerpo plano se enruta por PK compartida al CRUD dueño de la
especialización. La dirección es proveedor->especialización y de un solo salto.

Por qué: no se necesita una primitiva de herencia. La arista de PK compartida ya
expresa "misma identidad, campos en dos CRUD"; añadir la proyección aplanada es la
diferencia mínima para que "un proveedor jurídico es un proveedor" se lea y escriba
como un solo objeto. Limitar a un salto y una dirección evita la doble declaración
(proveedor pide especialización y especialización pide proveedor) que sería un ciclo
de composición.

Alternativa considerada: objeto de especialización anidado bajo un `mapped_name`
(como el marcador actual). Rechazada: contradice "es un proveedor"; obliga al cliente
a conocer la rama y parte el contrato en dos niveles.

Alternativa considerada: proyección bidireccional (también desde la
especialización). Rechazada y marcada como no-goal: introduce una doble arista que el
detector de ciclos de composición trataría como ciclo, y issue-65 solo ejercita
`/v1/proveedor`.

### Decisión 2: La autoría se toma del cuerpo; los demás campos de auditoría se rechazan
`UsuarioModificacionId` es un campo escribible por entidad con auditoría y su valor
viene del cuerpo. `FechaCreacion`, `FechaModificacion` y `Activo` no son escribibles
y se rechazan con `400` si llegan.

Por qué: el método de servicio no entrega un encabezado de autoría al backend, de
modo que el cuerpo es la única fuente viable del autor. Mantener los demás campos de
auditoría como administrados por el sistema evita que el cliente falsee marcas de
tiempo o el estado activo.

### Decisión 3: La transición de estado es decisión del servicio, mecanismo del motor
El servicio de escritura del proveedor resuelve el estado activo vigente y compone el
árbol que entrega al motor marcando el estado previo para baja cuando el `estado_id`
entrante difiere; si coincide, no marca nada (no-op). El motor materializa la baja
por su capacidad genérica de eliminación gateada por `deletable`. La ordenación es
crear el nuevo estado y dar de baja el previo después.

Por qué: la regla de "un solo estado activo" es lógica de negocio del proveedor, pero
el borrado pasa por el mismo motor gateado y pre-validado que el resto de la cascada,
sin un segundo camino de escritura. Crear antes de dar de baja deja, ante un fallo
intermedio, el proveedor con el nuevo estado activo (y quizá el previo aún activo),
situación recuperable por reconciliación, en vez de un proveedor sin estado activo.

Alternativa considerada: dar de baja primero y crear después. Rechazada: un fallo tras
la baja dejaría al proveedor sin estado activo.

### Decisión 4: Identidad por `estado_id` para la idempotencia de la transición
La transición compara el `estado_id` entrante con el del estado activo vigente;
coincidencia -> no-op, diferencia -> crear + dar de baja.

Por qué: `estado_id` es el valor de catálogo que identifica el estado; reenviar la
misma intención no debe acumular filas de historia ni romper el reenvío idempotente
del resto del árbol.

## Risks / Trade-offs

- [El cliente puede afirmar el autor de la modificación] → Aceptado por decisión del
  proyecto: el método de servicio no permite que un encabezado de autoría llegue al
  backend, por lo que el cuerpo es la única fuente; los demás campos de auditoría
  siguen administrados por el sistema.
- [La transición de estado combina creación y baja: fallo parcial a medio camino] →
  La ordenación crear-antes-de-dar-de-baja acota el peor caso a "estado previo aún
  activo" (recuperable), no a "sin estado activo"; la recuperación y el sobre de
  respuesta del fallo parcial pertenecen al cambio posterior de contrato de respuesta.
- [El aplanado de la especialización podría colisionar con el detector de ciclos de
  composición] → La proyección es de un salto y en una sola dirección; no se declara
  la arista inversa, de modo que no hay ciclo de composición.
- [El árbol reescrito diverge del marcador actual] → Las pruebas de lectura y
  escritura del árbol #58 se escriben primero y fallan contra el marcador actual,
  cubriendo las entidades nuevas, el aplanado de la especialización y el enrutado por
  PK compartida.
- [`cuenta_bancaria.tipo_cuenta_bancaria_id` es un campo fantasma en el diccionario]
  → Se omite del modelo efectivo; la cuenta usa `tipo_cuenta_id`.

## Migration Plan

- Configuración: `config/relaciones_proveedor.yaml` se reescribe al árbol #58 en la
  forma de objetos direccionables + referencias; se puebla la tabla de campos tipada
  por objeto. No hay migración de datos.
- Código: se retira el camino de autoría por contexto de petición y se adopta el campo
  de cuerpo `UsuarioModificacionId`; el servicio de escritura compone la transición de
  estado sobre el motor genérico.
- Rollback: revertir la configuración y el servicio restituye el contrato previo; no
  hay estado persistido propio del MID que deshacer.
