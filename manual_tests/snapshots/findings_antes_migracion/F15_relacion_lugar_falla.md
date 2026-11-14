# F15: fallan las relaciones a `lugar`

Estado: abierto. Causa confirmada: dependencia externa no disponible.

## Explicación
Las relaciones a `lugar` (`ProveedorNatural.PaisNacimiento`, `Tercero.DatosIdentificacion.CiudadExpedicionDetalle`, `Tercero.LugarOrigenDetalle`) quedan vacías y generan `kind: relation`.

## Causa confirmada (14/11/2026)
`AGORA_MID_UBICACIONES_CRUD_URL` apunta a `pruebasapi.intranetoas.udistrital.edu.co:8085`. Ese puerto rechaza la conexión:
```shell
curl -s -m 20 -o /dev/null -w 'code=%{http_code} exit=%{exitcode}\n' "http://pruebasapi.intranetoas.udistrital.edu.co:8085/v1/lugar/?limit=1"
# code=000 exit=7 (Connection refused)
```

## Propuesta
No se corrige desde el MID. Esperar a que el servicio de ubicaciones responda, o apuntar a un CRUD de ubicaciones disponible. Repetir la comprobación antes de cada corrida.
