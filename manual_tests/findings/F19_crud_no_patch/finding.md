# F19: el middleware enviaba PATCH al CRUD, que solo acepta PUT

Estado: resuelto. Verificado en vivo el 14/11/2026 contra el CRUD de 8081.
Solución: aplicada_sin_commit. Corregido en el árbol de trabajo (sin commit). Ver `## Cambio aplicado`.
Gravedad: alta. Clasificación propuesta, ver utils/README.md.

## Explicación

El middleware enviaba PATCH al CRUD de gestion_personas, que solo acepta PUT. Ver `## Causa`.

## Causa
- El CRUD de gestion_personas solo tiene `PUT /v1/<recurso>/{id}` (12 anotaciones `[put]`, ninguna `[patch]` en `gestion_personas_crud/controllers/`).
- `UpdateProveedorById` (y los demás `Update*`) escriben solo una lista fija de columnas con los valores del struct. Un PUT parcial deja en cero las columnas omitidas. Por eso la actualización envía el registro completo.

## Flujo aplicado (`services/write_resolver.go`, `Materialize`)
1. GET del registro por Id (`leerRegistroCompleto`). Si responde 404, se crea con POST y el Id propagado (PK compartida). Otro error aborta sin escribir. Si el registro llega sin campos, también se aborta.
2. `fusionarRegistro`: los valores recibidos se superponen al registro leído. Las FK con objeto completo se reducen a `{Id}`.
3. PUT con el registro completo.

Procedencia: la especificación (`declarative-write-engine`) fija que la recuperación usa solo GET. No describe el PUT ni la mezcla; esos pasos vienen de la indicación del responsable del cambio, con la justificación de la sección Causa.

Otros cambios: `Recuperar` devuelve `nil` con 404 (su documentación ya lo decía). El simulador del CRUD en `test/fixtures.go` responde GET por Id, guarda lo creado y actualizado, acepta PUT y responde 405 a PATCH (supuesto, no verificado contra el CRUD).

## Resultado en vivo (corrida 4)
Los PATCH que daban 502 ahora dan 200: U01, U11, U18 y U19. P06, P19, U05 y U08 pasan de 206 a 200. Casos que no coinciden: P11 (F22), U03 (F23), U13 (F28, regresión de este cambio). Detalle en `../../escritura/resultados.md`.

## Pendiente
- Un GET adicional por cada actualización.
- El PUT del CRUD con campos de auditoría y FK como objeto, verificado solo con los casos de la matriz.
- `TestIntegracionConsultaSinFieldsNoEntregaAuditoria` (F04) sigue en rojo, sin relación con este cambio.

## Cambio aplicado

**Archivos modificados:**

- `services/write_resolver.go`.
- `services/write_resolver_put_test.go` (nuevo).
- `services/write_resolver_test.go`.
- `services/proveedor_write_test.go`.
- `test/fixtures.go`.
- `test/integration_test.go`.
- `test/e2e_test.go`.
- `test/nulos_test.go`.
- `test/cascada_proveedor_test.go`.

**Verificación:** `go test ./services/...` pasa. En vivo contra el CRUD de 8081 (14/11/2026), los PATCH de U01, U11, U18 y U19 responden 200 (`escritura/resultados.md`). El paquete `test/` solo falla por F04.
