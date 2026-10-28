# Design

## Context

Ver `proposal.md` y los deltas de `declarative-write-engine`,
`api-response-contract` y `proveedor-write`. El motor ya valida todo el árbol
antes de materializarlo, propaga llaves después de esa validación y permite
avisos tolerantes para fallos de relaciones remotas. El problema es que el
payload reenviado mezcla campos propios con campos derivados que el motor debe
reconocer, y el aplanado actual elimina esos campos antes de llamar al resolver.

### Secuenciación y fusión con `expansion-id-implicito-lista-negra`

Este cambio se aplica **después** de `expansion-id-implicito-lista-negra`. Ese
cambio establece el conjunto de tipos de aviso (`relation`, `depth`, `cycle`,
`generic_resolver`) y el invariante "todo `Kind` emitido tiene una constante
declarada". Este cambio añade un quinto origen, `field_normalization`, por lo que
reformula ese invariante a su forma **abierta**: el conjunto de `Kind` no se fija
de forma cerrada, sino que se extiende con una constante por cada nuevo origen.
Al sincronizar `api-response-contract`, debe prevalecer la formulación abierta (no
la lista cerrada de cuatro) e incorporar `field_normalization` como quinta
constante; la prueba del invariante se actualiza en consecuencia (no enumera un
conjunto cerrado, sino que verifica el respaldo por constante).

## Goals / Non-Goals

**Goals:**

- Mantener `writable_fields` como la lista de campos controlables por el
  cliente.
- Reconocer campos derivados únicamente mediante `RelationConfig` y
  `key_propagation`.
- Tratar ausencia y `null` como falta de valor derivado.
- Comparar valores reenviados con la llave autoritativa cuando esté disponible.
- Canonizar una relación desactualizada solo para una entidad nueva y reportarlo
  como aviso diferenciado.
- Conservar la llave autoritativa en el payload del resolver y en `Data`.
- Evitar un GET previo genérico y no mover silenciosamente entidades existentes.

**Non-Goals:**

- No cambiar la modularización ni el formato de los archivos YAML.
- No modificar el flujo de expansión de lectura.
- No convertir arbitrariamente campos no declarados en campos relacionales.
- No agregar transacciones distribuidas ni rollback entre CRUD dueños.
- No permitir reasignar una entidad existente entre padres como efecto implícito
  de reenviar un payload.

## Decisions

### Separar campos de entrada y campos derivados

La validación inicial seguirá tratando `writable_fields` como la autorización de
campos propios del cliente. El índice de relaciones calculará además los campos
derivados:

- `fk`: `related_field` del hijo;
- `shared_pk`: `Id` del detalle;
- `fk_inbound`: `local_field` del nodo propietario.

Solo esos nombres, en el nivel y relación donde están declarados, obtendrán el
tratamiento especial. Un campo desconocido seguirá produciendo `ErrorEscritura`.

Alternativa descartada: agregar automáticamente las FKs a `writable_fields`.
Eso confundiría campos generados por el motor con campos que el cliente puede
controlar y debilitaría la validación de configuración.

### Validación en dos fases

La validación estructural se mantiene completa antes de cualquier llamada
externa. Durante ella se registra si un campo derivado apareció y se acepta
`null` como ausencia. No se intenta resolver todavía una llave que depende de
materializar un padre.

Después de cada materialización, el motor obtiene la llave autoritativa y aplica
esta regla. El disparador del conflicto es la presencia de un `Id` propio en la
entidad hija (entidad existente); una entidad sin `Id` nunca entra en conflicto:
sigue la vía de creación/reconciliación idempotente ya existente (`resolverExistente`
decide crear o reutilizar por `Id` o clave natural con la llave del padre ya
propagada):

```text
ausente o null       -> completar sin aviso
igual                -> conservar sin aviso
diferente + sin Id   -> crear/reconciliar con la llave autoritativa (canoniza) y avisar
diferente + con Id   -> error, sin reasignar silenciosamente
```

La comparación de conflicto solo tiene sentido cuando la entidad trae su propio
`Id` (entidad existente que podría estar siendo reasignada). Sin `Id` no hay
entidad previa que reasignar. En `shared_pk`, donde el campo derivado es el
propio `Id`, esto se resuelve igual: sin `Id` propio se propaga el del padre y se
crea; con `Id` propio, debe coincidir con el del padre o es conflicto.

Cuando la llave del padre ya está disponible en el cuerpo, la comparación se
realiza antes de la llamada externa correspondiente. Cuando el padre necesita
materialización, la comparación se realiza inmediatamente después de obtener su
Id y antes de enviar el payload del hijo.

**Inmutabilidad y campos derivados.** La regla de inmutabilidad protege contra el
*cambio* de un campo tras la creación, no contra su mera presencia. Cuando un
campo inmutable es además un campo derivado de una relación (por ejemplo una FK
que es a la vez `immutable_field` y `local_field` de un `fk_inbound`), reenviarlo
con el valor que coincide con la llave autoritativa no constituye una
modificación y no se rechaza: la pre-validación no lo rechaza por inmutable y la
conciliación posterior decide según la matriz (coincide conserva; difiere en
entidad existente es conflicto `400`; difiere en entidad nueva canoniza). Un campo
inmutable **no derivado** conserva el rechazo `400` al enviarse con un valor en
una actualización.

### Canonización sin GET previo

No se añadirá un GET genérico para cada entidad con `Id`. El caso de reenvío
normal tiene la misma llave y sigue el camino actual de upsert por `Id` o clave
natural. Si una entidad nueva trae un valor relacional desactualizado, se
sobrescribe con la llave autoritativa y se agrega un aviso de normalización.

Si una entidad existente trae una llave incompatible, el motor devuelve
`ErrorEscritura` antes de enviar el PATCH de esa entidad. Así se evita una
reasignación silenciosa, aunque no se intenta demostrar mediante una consulta
adicional si el estado remoto podría ser compatible.

Alternativa descartada: GET previo. Aumentaría latencia y no garantizaría
atomicidad frente a cambios concurrentes; una validación condicional del CRUD
dueño sería una capacidad futura independiente.

### Conservación del payload interno

Tras conciliar la llave del padre con el campo derivado del hijo (comparándola,
no solo propagándola), el aplanado conservará explícitamente los campos internos
permitidos por la relación. Conservará únicamente:

- campos propios aceptados;
- `Id`;
- FKs de parámetros anidados declaradas;
- el campo derivado de la relación que el motor acaba de validar o generar.

Las relaciones anidadas continuarán excluyéndose del payload plano. La llave
autoritativa se asignará con su valor nativo, sin convertirla a texto.

### Avisos de normalización

Se añadirá el `Kind: field_normalization` para distinguir una corrección de campo
derivado de `Kind: relation`, que seguirá reservado para fallos remotos tolerados.
El aviso incluirá ruta y servicio, pero no valores sensibles ni detalles del CRUD.
Una normalización exitosa produce `206`; una coincidencia produce `200` sin
aviso. `field_normalization` es un nuevo origen de aviso y, por el invariante abierto de
`api-response-contract`, se registra como una constante de tipo de aviso más. Las
constantes de `Kind` y el motor que las emite se mantienen en un catálogo único
que es la fuente de verdad del conjunto; la lista de tipos declarados se deriva de
ese catálogo en lugar de mantenerse en paralelo, de modo que toda constante
registrada queda asociada a su origen por construcción. El invariante "toda
constante declarada tiene una ruta de emisión que la respalda" se verifica de
forma repartida por motor (expansión y escritura), no enumerando un conjunto
cerrado.

Alternativa descartada: que los emisores recuperen el `Kind` a emitir desde el
catálogo. Cada punto de emisión referencia su constante directamente para que el
compilador garantice el símbolo y se conserve el amarre uno a uno entre sitio de
emisión y constante; el catálogo sirve para validar y enumerar, no para decidir
qué se emite.

### `fk_inbound`, `shared_pk` y `fk`

Los tres modos usan la misma política de comparación contra la llave autoritativa,
pero no comparten un único punto de comparación en el flujo:

- `fk` y `shared_pk` (descendentes): el padre se materializa primero; la llave
  autoritativa (su `Id`) existe antes de tocar al hijo. La comparación ocurre en
  el flujo padre->hijo, al propagar la llave al hijo.
  - `fk`: compara `related_field` del hijo con el `Id` del padre.
  - `shared_pk`: compara `Id` del detalle con el `Id` del padre.
- `fk_inbound` (ascendente): el campo derivado (`local_field`) es un campo del
  propio nodo, y su valor autoritativo es el `Id` de la entidad referenciada, que
  solo se conoce tras materializarla en el pre-pass. La comparación ocurre en ese
  pre-pass, inmediatamente antes de fijar `local_field` en el nodo.
  - `fk_inbound`: compara `local_field` del nodo con el `Id` de la entidad
    referenciada materializada.

En `fk_inbound`, un nodo con `Id` propio (entidad existente) cuyo `local_field`
reenviado difiera del `Id` de la entidad referenciada materializada es un
conflicto: no se resuelve sobrescribiendo el campo local, sino que produce
`ErrorEscritura` (`400`), coherente con la regla de conflicto de entidad
existente. Un nodo sin `Id` sigue la vía normal: se fija `local_field` con la
llave autoritativa.

## Risks / Trade-offs

- [Un campo derivado desactualizado puede producir `206` en una creación] ->
  el `Data` devuelto siempre contiene el valor autoritativo y el aviso identifica
  la normalización.
- [Un conflicto existente se rechaza sin verificar el estado remoto] ->
  evita reasignaciones silenciosas y deja una futura validación atómica del CRUD
  como evolución separada.
- [La validación posterior puede ocurrir después de materializar un padre] ->
  se conserva la política existente de creación parcial; se valida antes de la
  llamada del hijo y se documenta que no hay rollback distribuido.
- [Un nuevo `Kind` requiere mantener sincronizados respuestas y OpenAPI] ->
  se agregan pruebas de contrato y actualización de documentación en la misma
  implementación.

## Migration Plan

1. Añadir la representación de campos derivados y la validación de coherencia en
   el motor.
2. Conservar esos campos en payloads planos y agregar el aviso de normalización.
3. Cubrir primero pruebas unitarias del motor y después integración/e2e.
4. Actualizar `API.md`, `test/tests.md` y el contrato OpenAPI.
5. Ejecutar validación, build y suites focalizadas; luego la suite completa.
6. Rollback: revertir el cambio de motor y del contrato; los clientes volverían
   a necesitar eliminar campos derivados antes de reenviar respuestas.
