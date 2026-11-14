# F00: problemas resueltos y configuración de arranque

Estado: resuelto.

## R1: arranque abortado por la lista negra
`config/global_blacklist.yaml` tenía un bloque `ANY/ANY/ANY` que el validador (`config/blacklist.go`) rechaza. Se quitó el bloque (cambio sin commit). Consecuencia abierta: la auditoría ya no se oculta, ver F04.

## R2: URLs del `.env` con sufijo `/v1/`
El código agrega `/v1/tipo_contribuyente` a la base de terceros. Con el sufijo quedaba `/v1/v1/`. Se quitó el sufijo de las cuatro URLs `*_CRUD_URL` en `.env`. El `.env` no está versionado, así que no hay diff.

## Cambio aplicado

**Archivos modificados:**

- `config/global_blacklist.yaml` (R1).
- `.env` (R2, no versionado).

**Verificación:** `go test ./config/...` pasa. El arranque del MID del 14/11/2026 (12:03) termina sin el error de lista negra.
