# F00: problemas resueltos y configuración de arranque

Estado: resuelto.
Solución: aplicada_sin_commit. Ver `## R1` y `## R2`.
Gravedad: alta. Clasificación propuesta, ver utils/README.md.

## Explicación

Dos problemas de arranque y configuración. El arranque del MID abortaba porque el validador de la lista negra rechazaba el bloque `ANY/ANY` de `config/global_blacklist.yaml`: sus cuatro campos de auditoría no están en el modelo conocido. Las URLs de `.env` tenían el sufijo `/v1/`.

## R1: arranque abortado por la lista negra

`config/global_blacklist.yaml` declara el bloque `ANY/ANY` con `FechaCreacion`, `FechaModificacion`, `Activo` y `UsuarioModificacionId`. `config/blacklist.go` registra como advertencia los campos exactos que no están en el modelo conocido, en lugar de abortar. El arranque termina y registra una advertencia por campo. Los componentes `related_api`, `related_table` y el comodín en los tres niveles siguen fallando. La causa de fondo y la solución definitiva están en F04.

## R2: URLs del `.env` con sufijo `/v1/`

El código agrega `/v1/tipo_contribuyente` a la base de terceros. Con el sufijo quedaba `/v1/v1/`. Las cuatro URLs `*_CRUD_URL` de `.env` no llevan el sufijo. El `.env` no está versionado.

## Cambio aplicado

**Archivos modificados:**

- `config/global_blacklist.yaml` (R1).
- `config/blacklist.go` (R1).
- `.env` (R2, no versionado).

**Verificación:** `go test ./config/...` pasa. El arranque del MID del 14/11/2026 (15:40, PID 956) termina con cuatro advertencias de lista negra y sin error de arranque.
