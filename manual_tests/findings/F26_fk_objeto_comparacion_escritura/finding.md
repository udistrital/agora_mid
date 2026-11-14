# F26: la comparación de llaves en escritura no entiende FK como objeto

Estado: aceptado.
Solución: aplicada_sin_commit. Corregido en el árbol de trabajo (solución rápida, sin commit). Ver `## Cambio aplicado`.
Gravedad: alta. Clasificación propuesta, ver utils/README.md.

## Explicación
Al actualizar un contacto, el motor compara `aCadena(objeto)` con la llave del padre y rechaza con 400 "la llave relacional reenviada difiere", aunque coincidan.

## Solución aplicada
`helper/expand.go`: `aCadena` toma el `Id` de un mapa que lo tenga. Prueba: `helper/acadena_objeto_test.go`.

## Cambio aplicado

**Archivos modificados:**

- `helper/expand.go`.
- `helper/acadena_objeto_test.go` (nuevo).

**Verificación:** `go test ./helper/...` pasa. La prueba `helper/acadena_objeto_test.go` cubre la comparación de FK como objeto.

## Decisión

Aprobado. La comparación de FK como objeto en escritura se mantiene. Cubierto por `helper/acadena_objeto_test.go`. Sin verificación en vivo en esta revisión. El diseño definitivo de la normalización de FK (junto con F03 y F25) queda como nota de mantenimiento.
