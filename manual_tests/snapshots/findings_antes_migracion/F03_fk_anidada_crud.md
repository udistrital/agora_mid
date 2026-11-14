# F03: el CRUD devuelve FK como objeto anidado y el motor de expansión no las compara

Estado: corregido en el árbol de trabajo (sin commit). Prueba `helper/flatten_fk_crud_test.go` en verde.

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
