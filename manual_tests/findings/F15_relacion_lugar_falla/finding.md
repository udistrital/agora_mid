# F15: fallan las relaciones a `lugar`

Estado: resuelto. Causa original (14/11, 15:50) obsoleta: el servicio de ubicaciones responde desde la URL actual de `.env` (ver `## Verificación en vivo (14/11/2026, 16:40)` y `## Cambio aplicado`).
Solución: ninguna. Sin corrección desde el MID. Ver `## Propuesta`.
Gravedad: media. Clasificación propuesta, ver utils/README.md.

## Explicación
Las relaciones a `lugar` (`ProveedorNatural.PaisNacimiento`, `Tercero.DatosIdentificacion.CiudadExpedicionDetalle`, `Tercero.LugarOrigenDetalle`) quedan vacías y generan `kind: relation`.

## Causa confirmada (14/11/2026)
La variable `AGORA_MID_UBICACIONES_CRUD_URL` apunta a un servicio que rechaza la conexión:
```shell
curl -s -m 20 -o /dev/null -w 'code=%{http_code} exit=%{exitcode}\n' "${AGORA_MID_UBICACIONES_CRUD_URL}/v1/lugar/?limit=1"
# code=000 exit=7 (Connection refused)
```

## Propuesta
No se corrige desde el MID. Esperar a que el servicio de ubicaciones responda, o apuntar a un CRUD de ubicaciones disponible. Repetir la comprobación antes de cada corrida.

## Verificación en vivo (14/11/2026, 16:40)

La URL de `AGORA_MID_UBICACIONES_CRUD_URL` cambió después de la causa original. El valor de la variable no se registra aquí.

- GET directo: `GET ${AGORA_MID_UBICACIONES_CRUD_URL}/v1/lugar/?limit=1` responde `code=200` y devuelve registros de lugar (por ejemplo `Id 1, Nombre "Afganistán"`).
- MID: se arrancó el binario con el procedimiento de 3.2 (puerto 8080, log en `/tmp/agora_mid_run.log`). `POST /v1/proveedor/consulta` con `requests/mid_proveedor_consulta_1.json` responde `206` con 17 avisos `depth` y ningún aviso `relation`.
- `Data.PaisNacimiento` se expande como objeto (`Id 1`, `Nombre "Afganistán"`, `TipoLugarId 1`). Antes quedaba vacío.
- `Data.Tercero.DatosIdentificacion[0].CiudadExpedicion` sigue como escalar (96), y `Data.Tercero.LugarOrigen` es 0. No se revisó si el proveedor de prueba tiene esos datos cargados.

Conclusión: la falla por conexión rechazada ya no ocurre. Queda abierto solo porque el hallazgo depende del servicio externo. Los avisos `depth` que aún aparecen pertenecen a F14, no a este hallazgo.

## Cambio aplicado

Sin cambio en el repositorio. La URL de `AGORA_MID_UBICACIONES_CRUD_URL` cambió en el ambiente después de la causa original. La corrección no tocó código del MID.

**Archivos modificados:** ninguno del repositorio.

**Verificación:** `POST /v1/proveedor/consulta {"id":1}` (17/11/2026, 19:21) no devuelve avisos `kind: relation`. `Data.PaisNacimiento` viene expandido. Los avisos `depth` restantes pertenecen a F14.
