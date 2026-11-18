# Proposal

## Why

El control de escritura por relación garantiza que una validación fallida no deje
escrituras parciales (`400` sin efectos), pero no define qué ocurre cuando la
cascada ya comenzó a materializar y una dependencia remota falla o agota su tiempo
de espera a mitad de camino. Falta el contrato de respuesta para ese fallo parcial:
qué código se devuelve, qué queda en `Data`, cómo se nombra el punto de parada y
cómo un reintento vuelve a enlazar lo ya materializado sin duplicarlo. Este cambio
cierra esa frontera y fija la semántica de omisión/vacío/null del contrato de
escritura.

## What Changes

- **Mapa de códigos de estado de la escritura.** Se fija el significado de cada
  código sobre la línea de tiempo de la escritura:
  - `400`: validación previa a cualquier escritura (ya es responsabilidad del
    control por relación; aquí solo se referencia como frontera).
  - `502`/`504`: una dependencia remota es inalcanzable **durante el pre-pass de
    solo lectura**, antes de materializar nada; no hay subárbol parcial que
    entregar.
  - `200`: la cascada completa se materializó sin avisos.
  - `206`: la cascada se materializó con canonización de un campo derivado
    (`Kind: field_normalization`, ya existente).
  - `206`: la cascada **se detuvo a mitad de camino** por un fallo o timeout de una
    dependencia remota (stop-abort). `Data` lleva lo ya materializado y
    `MetaData.Errors` nombra el punto de parada y su causa.
- **Taxonomía de avisos por `Kind`.** La distinción validación / dependencia /
  timeout se expresa como valores de `Kind` respaldados por constantes declaradas,
  sobre el conjunto abierto existente de `MetaData.Errors`, sin introducir un
  segundo eje de clasificación.
- **Identidad de reconciliación por entidad.** Tras un `206`, un reintento vuelve a
  enlazar cada nodo ya materializado buscándolo por su `Id` propio y, en su defecto,
  por su clave natural. Las colecciones sin `Id` ni clave natural solo son
  idempotentes por la vía de reenvío del `Data` materializado (donde el elemento ya
  trae su `Id`); el reenvío del cuerpo original puede duplicarlas, y el contrato lo
  declara explícitamente. La transición de estado del proveedor usa `estado_id` como
  identidad de reconciliación.
- **Distinción omisión / vacío / null en el contrato de escritura.**
  - Ausente: "no hablo de este campo", se conserva el valor existente.
  - `null`: en un campo escalar nullable y escribible, se fija a null (se rechaza si
    no es nullable); en una FK relacional, se conserva la regla existente del motor
    (tratar como ausente y derivar la llave autoritativa).
  - Vacío (`""`, `[]`): es un valor literal; una colección vacía **nunca** dispara
    la eliminación de elementos existentes (el motor nunca infiere eliminaciones).
- **No se promete transacción distribuida.** El contrato es explícito en que la
  recuperación ante fallos parciales se apoya en reintento idempotente por
  reconciliación, no en atomicidad distribuida entre CRUD dueños.

## Capabilities

### New Capabilities
<!-- Ninguna. -->

### Modified Capabilities
- `api-response-contract`: Se añade el mapa de códigos de la escritura (pre-pass
  `502`/`504`, stop-abort `206`), el significado del `206` de fallo parcial durante
  la materialización, y la extensión de la taxonomía de `Kind` a
  `dependency_failure` y `timeout` sobre el conjunto abierto existente.
- `proveedor-write`: Se añade la identidad de reconciliación por entidad para el
  reintento tras `206` (re-find por `Id`/clave natural; colecciones sin clave solo
  idempotentes por reenvío del `Data`; `estado_id` para la transición de estado) y
  la distinción omisión/vacío/null del contrato de escritura, incluido que `[]`
  nunca elimina. Modifica la transición de estado del proveedor (requisito del cambio
  absorbido) con esta precedencia: `Id` presente actualiza en sitio; sin `Id` y con
  campos coincidentes es no-op; sin `Id` y con campos distintos crea el nuevo estado
  y da de baja el previo.
- `declarative-write-engine`: Se precisa el fallo de escritura en modo estricto: un
  fallo bloqueante antes de materializar detiene la cascada sin subárbol parcial,
  mientras que un fallo después de materializar al menos una entidad es un stop-abort
  que devuelve el subárbol parcial con un aviso tipado que nombra el nodo de parada.

## Impact

- **Comportamiento observable**: `POST`/`PATCH /v1/proveedor` adquieren un contrato
  de fallo parcial preciso: `206` con `Data` parcial y `MetaData.Errors` que nombra
  el punto de parada y su `Kind`; `502`/`504` reservados al fallo de dependencia del
  pre-pass previo a toda escritura.
- **Frontera con cambios existentes**: consume la garantía de "cero escrituras si la
  validación falla" y el pre-pass de solo lectura del control de escritura por
  relación, y los casos de reconciliación (incluida la transición de estado del
  proveedor) del cambio que absorbe el modelo #58.
- **Código** (fase apply): el envoltorio de respuesta y el motor de escritura emiten
  el `206` de stop-abort con el aviso tipado; se declaran las constantes de `Kind`
  `dependency_failure` y `timeout`; el motor aplica la distinción
  omisión/vacío/null por tipo de campo.
- **No-goal explícito**: ninguna transacción distribuida entre CRUD dueños.
- **Dependencia de planeación**: se apila sobre `control-escritura-por-relacion` y
  `absorber-modelo-58-proveedor`; orden de apply `0 -> A -> B -> C`.
