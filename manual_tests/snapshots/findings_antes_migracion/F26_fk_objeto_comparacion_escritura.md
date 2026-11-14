# F26: la comparación de llaves en escritura no entiende FK como objeto

Estado: corregido en el árbol de trabajo (solución rápida, sin commit).

## Explicación
Al actualizar un contacto, el motor compara `aCadena(objeto)` con la llave del padre y rechaza con 400 "la llave relacional reenviada difiere", aunque coincidan.

## Solución aplicada
`helper/expand.go`: `aCadena` toma el `Id` de un mapa que lo tenga. Prueba: `helper/acadena_objeto_test.go`.

## Cambio aplicado

**Archivos modificados:**

- `helper/expand.go`.
- `helper/acadena_objeto_test.go` (nuevo).

**Verificación:** `go test ./helper/...` pasa. La prueba `helper/acadena_objeto_test.go` cubre la comparación de FK como objeto.
