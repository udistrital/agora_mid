# F03: el CRUD devuelve FK como objeto anidado y el motor de expansión no las compara

Estado: en_decision. Prueba `helper/flatten_fk_crud_test.go` en verde.
Solución: aplicada_sin_commit. Corregido en el árbol de trabajo (sin commit). Ver `## Cambio aplicado`.
Gravedad: alta. Clasificación propuesta, ver utils/README.md.

## Explicación
El CRUD devuelve `Contacto.ProveedorId`, `CuentaBancaria.ProveedorId` y `CuentaBancaria.ContactoId` como objeto completo. El motor compara la llave del padre con `aCadena(objeto)`, que produce `map[...]`, y la expansión devuelve `[]` aunque haya datos.

## Solución aplicada
`helper/expand.go`: `aplanarFKAnidadas` reemplaza cada objeto con clave `Id` por ese `Id`, y se llama en `resolver` después de `res.Resolve`.

Verificado el 13/11/2026 con `POST /v1/proveedor/consulta {"id":1}`: `Contacto[0].ProveedorId = 1`. Esa respuesta no se conservó en el árbol de respuestas actual.

## Riesgo
Se aplana cualquier mapa con clave `Id`. Hoy el CRUD solo anida objetos en FK. Si un CRUD devuelve un objeto legítimo con `Id`, se pierde.

## Cambio aplicado

**Archivos modificados:**

- `helper/expand.go`.
- `helper/flatten_fk_crud_test.go` (nuevo).

**Verificación:** `go test ./helper/...` pasa. La prueba `helper/flatten_fk_crud_test.go` cubre el aplanado de las FK.

## Manejo de error (14/11, 15:15)

Una FK que llega como objeto sin `Id`, o con `Id` nulo, no se puede aplanar. El campo se entrega sin cambios y la expansión emite un aviso `fk_no_aplanable` (origen expansión) en `MetaData`. No falla la respuesta. La prueba `helper/flatten_fk_sin_id_test.go` cubre los tres casos: sin `Id`, `Id` nulo y `Id` válido sin aviso. El caso de paso normal no cambió.

**Verificación en vivo (14/11, 15:20):** MID reemplazado (PID 87903, `/tmp/agora_mid_run_f03`). `POST /v1/proveedor/consulta {"id":1}` devuelve `Contacto[0].ProveedorId = 1` y no aparece `fk_no_aplanable`. El log no tiene errores.

## Notas de especificación

Nuevo tipo de aviso `fk_no_aplanable` con origen expansión. Las especificaciones `nested-object-expansion` y `api-response-contract` enumeran los tipos de aviso, así que ajustarlas al sincronizar. No se modificó el spec desde estas pruebas.
