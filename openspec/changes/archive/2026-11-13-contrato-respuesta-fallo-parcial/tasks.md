# Tasks

## 1. Taxonomía de avisos de fallo por Kind

- [x] 1.1 Escribir pruebas unitarias que afirmen que el envoltorio emite un aviso con `Kind: dependency_failure` ante un fallo de dependencia y `Kind: timeout` ante un tiempo de espera, cada uno respaldado por una constante declarada; confirmar que las pruebas fallan porque las constantes aún no existen.
- [x] 1.2 Declarar las constantes de `Kind` `dependency_failure` y `timeout` junto a las existentes y verificar que las pruebas de 1.1 pasan y que no existe ningún `Kind` emitido sin constante.
- [x] 1.3 Escribir y verificar una prueba que afirme que la clasificación de fallo se expresa solo por el valor de `Kind`, sin un campo discriminador paralelo en `MetaData.Errors`.

## 2. Mapa de códigos de estado de la escritura

- [x] 2.1 Escribir pruebas de contrato que, sobre el motor de escritura con CRUD dueños simulados, afirmen: `200` en cascada completa sin avisos; `502` ante dependencia inalcanzable en el pre-pass; `504` ante tiempo de espera en el pre-pass; sin `Data` parcial en los dos últimos. Incluir dos casos de lectura caída en el pre-pass: una relación bajo política `fail` (o bloqueante) produce `502`/`504` sin materializar ninguna escritura; y la lectura de `estado_proveedor`, cuya política es `notify`, produce `206` con un aviso `Kind: dependency_failure`, omite ese nodo y materializa los demás. Confirmar que fallan antes de implementar el mapeo.
- [x] 2.1a Extender el CRUD simulado para inducir fallos no solo en `POST` sino también en la lectura (`GET`) del pre-pass y en la baja (`DELETE`), de modo que 2.1 y 3.1 puedan simular un CRUD dueño caído en el nodo y método exactos; verificar que el fallo inducido se observa en el método esperado y no en otros.
- [x] 2.2 Implementar en el pre-pass de solo lectura la detección de dependencia inalcanzable/timeout que responde `502`/`504` sin materializar nada, y verificar que las pruebas de 2.1 pasan.
- [x] 2.3 Escribir y verificar una prueba que afirme que una validación previa produce `400` y nunca `206`, delimitando la frontera con el control de escritura por relación.

## 3. Respuesta 206 de stop-abort a mitad de cascada

- [x] 3.1 Escribir pruebas de contrato que simulen un fallo de dependencia y un timeout en el nodo N tras materializar los nodos previos, y afirmen: `Status: "206"`, `Data` con los identificadores de los nodos materializados, una entrada en `MetaData.Errors` con `Path`/`Service`/`Message`/`Kind` que nombra el nodo N, y que los nodos posteriores a N no se enviaron a su CRUD. Confirmar que fallan antes de implementar el stop-abort.
- [x] 3.2 Implementar el stop-abort en la cascada: al fallar un nodo, detener el recorrido, no intentar los posteriores y construir el `206` con el subárbol parcial y el aviso tipado; verificar que las pruebas de 3.1 pasan.
- [x] 3.3 Escribir y verificar una prueba de regresión que confirme que una cascada completa con canonización sigue devolviendo `206` con `Kind: field_normalization`, sin que el nuevo stop-abort la altere.

## 4. Identidad de reconciliación para el reintento

- [x] 4.1 Escribir pruebas de integración que, tras un `206`, reenvíen el `Data` materializado y afirmen que cada entidad se re-encuentra por `Id` (reutilización/actualización, sin duplicar) y que una entidad sin `Id` pero con clave natural se re-encuentra por su clave natural, incluido el caso de creación en frío (una entidad sin `Id` ni FK ascendente, resuelta únicamente por su clave natural) que crea sin duplicar. Confirmar que fallan antes de implementar el re-find.
- [x] 4.2 Implementar el re-find por `Id` y clave natural en el reintento y verificar que las pruebas de 4.1 pasan sin crear duplicados.
- [x] 4.3 Escribir y verificar pruebas que afirmen la idempotencia limitada de colecciones sin clave estable: reenviar el `Data` no duplica elementos (traen `Id`); reenviar el cuerpo original puede duplicarlos, de acuerdo con la limitación declarada.
- [x] 4.4 Escribir y verificar una prueba del caso de transición de estado a medio camino: reintento re-encuentra el nuevo `estado_proveedor` por `estado_id` como no-op y re-intenta solo la baja pendiente del estado previo, sin crear un segundo estado.
- [x] 4.5 Escribir y verificar pruebas de la transición de estado en la escritura normal (no en el reintento), por precedencia: un `estado_proveedor` con el `Id` del activo se actualiza en sitio sin alta ni baja, aunque cambien sus campos; sin `Id` y con todos los campos iguales al activo es no-op; sin `Id` y con `EstadoId` igual pero `Observacion` distinta crea el nuevo estado y da de baja el previo. Reemplazar `TestIntegracionTransicionEstadoMismoEstadoEsNoOp` por el caso sin `Id` y campos iguales, y añadir el caso sin `Id` con `Observacion` distinta. Confirmar que fallan antes de implementar.

## 5. Distinción omisión / vacío / null

- [x] 5.1 Escribir pruebas unitarias que afirmen, por tipo de campo: campo ausente conserva el valor; `null` en escalar nullable lo fija a null; `null` en escalar no nullable responde `400` sin escrituras; `null` en FK relacional se trata como ausente y deriva la llave. Confirmar que fallan antes de implementar la distinción.
- [x] 5.2 Implementar la distinción omisión/null por tipo de campo en el motor de escritura y verificar que las pruebas de 5.1 pasan.
- [x] 5.3 Escribir y verificar pruebas que afirmen que una colección `[]` y una colección ausente conservan los elementos existentes y no materializan ninguna baja (el motor no infiere eliminaciones), incluido `EstadoProveedor: []` como caso concreto: una lista de estados vacía no da de baja el estado activo vigente.

## 6. Verificación de integración del contrato de fallo parcial

- [x] 6.1 Ejecutar `go test ./...` y `go build ./...` y verificar que la suite completa pasa, cubriendo el flujo end-to-end: `200`, `206` de canonización, `206` de stop-abort, `502`/`504` de pre-pass, `400` de validación, reintento idempotente y distinción omisión/vacío/null.
- [x] 6.2 Verificar con `openspec validate contrato-respuesta-fallo-parcial --strict` que el cambio valida sin advertencias.

## Workflow follow-up

- Archivar el cambio tras satisfacer los requisitos de revisión del proyecto.
- Verificar el resultado archivado.
