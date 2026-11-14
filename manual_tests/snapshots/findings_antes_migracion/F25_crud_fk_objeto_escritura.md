# F25: el CRUD exige ProveedorId y ContactoId como objeto en contacto y cuenta_bancaria

Estado: corregido en el árbol de trabajo (solución rápida, sin commit).

## Explicación
`POST /v1/contacto/` con `ProveedorId: 1` responde 400. Con `ProveedorId: {"Id": 1}` responde 201. El motor enviaba el entero, así que los hijos fallaban.

## Solución aplicada
- `services/write_resolver.go`: `envolverFKRelacional` convierte `ProveedorId` y `ContactoId` a `{"Id": n}` en una copia que se envía al CRUD. El payload interno sigue con enteros.
- `services/fk_objeto_crud_test.go`: prueba nueva.
- `test/cascada_proveedor_test.go`: tres aserciones leen la FK con `fkEntero` (acepta entero u objeto).

## Cambio aplicado

**Archivos modificados:**

- `services/write_resolver.go`.
- `test/cascada_proveedor_test.go`.
- `services/fk_objeto_crud_test.go` (nuevo).

**Verificación:** `go test ./services/...` pasa. La prueba `services/fk_objeto_crud_test.go` cubre el envoltorio de `ProveedorId` y `ContactoId`. El paquete `test/` solo falla por F04.
