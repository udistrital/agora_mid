# F25: el CRUD exige ProveedorId y ContactoId como objeto en contacto y cuenta_bancaria

Estado: aceptado.
Solución: aplicada_sin_commit. Corregido en el árbol de trabajo (solución rápida, sin commit). Ver `## Cambio aplicado`.
Gravedad: alta. Clasificación propuesta, ver utils/README.md.

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

## Decisión

Aprobado. El envoltorio de `ProveedorId` y `ContactoId` como objeto se mantiene. Cubierto por las pruebas de `services/fk_objeto_crud_test.go`. Sin verificación en vivo en esta revisión. El diseño definitivo de la normalización de FK (junto con F03 y F26) queda como nota de mantenimiento.
