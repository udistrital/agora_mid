# Design

## Context

Ver `proposal.md` - Why. El service de terceros (`services/terceros.go`,
paquete `services`) ya implementa `GetAll`, `GetOne`, `Post`, `Put` delegando
en `utils_oas/v2/request` y contiene, mezcladas, las utilidades HTTP:
`construirURL`, `traducir`, `esEOFDecodificacion`, `etapaSinRespuesta`,
`registrarLogs`, `hostPuerto`, el tipo `errPublico`, los mensajes públicos y el
seam `var registrar = registrarLogs`. La unión de URL vive aparte en
`utils/http_client.go` como `UnirURL` (exportada, otro paquete).

El contrato de terceros (fijado en la capacidad `terceros-crud-cliente`):
`(ctx, endpoint[, payload], target) (ok, statusCode, err)`, filtros por
`url.Values`, éxito 200–226, 204/EOF como éxito, rechazo HTTP sin decodificar,
error público neutro y un log por fallo. El CRUD de proveedores ya tiene su
host configurado (`GestionProveedoresCrudHost`).

## Goals / Non-Goals

**Goals:**
- Service de proveedores con el mismo contrato y convenciones que terceros.
- Una sola implementación de cada utilidad HTTP, compartida por ambos services.
- Parametrizar la identidad del service (log + mensajes + host) sin duplicar.

**Non-Goals:**
- Rutas, controladores, modelos concretos de proveedor y DELETE.
- Cambiar el comportamiento observable de terceros.
- Reintroducir una variable global mutable para los logs.

## Decisions

### Ubicación: `services/http_utils.go` en el paquete `services`
Las utilidades dependen de `request.ErrResponseDecode`, del tipo `errPublico`
y de los mensajes públicos: todo del dominio `services`. Moverlas a `utils`
arrastraría esas dependencias y arriesgaría un ciclo de imports. Por eso el
archivo compartido es `services/http_utils.go`, en el mismo paquete que ambos
services. `utils/http_client.go` se elimina y `UnirURL` se traslada a
`services/http_utils.go`; como ya no cruza paquetes, vuelve a ser privada
(`unirURL`). `utils.Response`/`utils.Metadata` permanecen en `utils/response.go`.

### Visibilidad: privadas (mismo paquete)
Ambos services viven en `services`, así que las utilidades no necesitan
exportarse. Se conservan nombres privados. No se crea `ConstruirURL` ni
wrappers con el nombre anterior.

### Tabla de funciones: nombre, ubicación final, visibilidad, llamadores

| Función (actual) | Ubicación final | Visibilidad | Llamadores |
|---|---|---|---|
| `UnirURL` (en `utils/http_client.go`) | `services/http_utils.go` como `unirURL` | privada | `construirURL` |
| `construirURL` (en `terceros.go`) | `services/http_utils.go` | privada | `GetAll/GetOne/Post/Put` de terceros y proveedores |
| `traducir` (en `terceros.go`) | `services/http_utils.go` | privada | las 4 operaciones de ambos services |
| `esEOFDecodificacion` (en `terceros.go`) | `services/http_utils.go` | privada | `traducir` |
| `etapaSinRespuesta` (en `terceros.go`) | `services/http_utils.go` | privada | `traducir` |
| `registrarLogs` (en `terceros.go`) | `services/http_utils.go` | privada | `construirURL`, `traducir` (ambos services) |
| `hostPuerto` (en `terceros.go`) | `services/http_utils.go` | privada | `registrarLogs`, pruebas |
| `errPublico` (tipo, en `terceros.go`) | `services/http_utils.go` | privada | ambos services |

`utils/http_client.go` queda eliminado; su test `TestUnirURL` se reubica a
`services` (p. ej. `http_utils_test.go`) llamando `unirURL`.

### Parametrización por service
Hoy `construirURL` y los mensajes están fijos a terceros (`"TercerosCrudHost
vacío"`, `servicio=terceros`). Para compartir sin duplicar, lo específico se
pasa como dato:

- `construirURL(base, endpoint, queryParams)` ya es neutro salvo el texto del
  error de host vacío; se vuelve neutro ("URL base no configurada") o se le
  pasa el nombre de la clave. El host lo provee cada service (`hostTerceros()` /
  `hostProveedores()`).
- `registrarLogs` y `traducir` reciben el identificador del service (por
  ejemplo un parámetro `servicio string`: `"terceros"` / `"gestion_proveedores"`)
  para que el log lo refleje. El mensaje público neutro también se elige por
  service, manteniendo el host fuera de `err.Error()`.
- Alternativa considerada: un struct `clienteCRUD{servicio, host, msgs...}` con
  métodos. Es más idiomático si crecen los services; para dos, pasar el
  identificador como parámetro a las utilidades compartidas es suficiente y
  menos intrusivo. Decisión: parámetro `servicio` por ahora; el struct queda
  como evolución futura si se añaden más CRUD.

### Decisión sobre `var registrar = registrarLogs`
Se **retira** la indirección. Hoy `registrar` solo reenvía a `registrarLogs`;
su único motivo era permitir que las pruebas capturaran las entradas
sustituyendo la variable global. Las operaciones pasan a llamar `registrarLogs`
directamente. Para conservar la cobertura de "una sola entrada por fallo",
"identifica el service" y "omite el host del error público", las pruebas usan
un mecanismo de captura de logs que no dependa de una variable global mutable:
capturar la salida del logger de Beego (adaptador/buffer) o verificar el efecto
observable del fallo (retorno `(ok, status, err)` y ausencia de host en
`err.Error()`). No se introduce otra variable global mutable equivalente.

Qué se perdería y por qué es aceptable: el seam de variable permitía asertar el
contenido exacto del log con mínima fricción. Al retirarlo, las aserciones de
log se hacen sobre la salida real del logger (más fiel al comportamiento de
producción) o sobre el efecto observable. No se pierde cobertura de requisitos;
se cambia el punto de observación.

## Risks / Trade-offs

- [Capturar el log de Beego en pruebas es más laborioso que sustituir una
  variable] → Mitigación: registrar un adaptador de logs con buffer en el
  arranque de la prueba, o asertar el efecto observable (status/err sin host).
  Se elige el que dé cobertura con menos fricción en la implementación.
- [Parametrizar por `servicio string` dispersa el identificador en las firmas
  internas] → Mitigación: es privado y acotado a `services`; si crecen los
  services se migra al struct `clienteCRUD`.
- [El traslado de utilidades podría romper las pruebas de terceros] → Mitigación:
  mover sin cambiar comportamiento, correr las pruebas de terceros tras el
  traslado y verificar una sola definición de cada utilidad.

## Open Questions

Ninguna.
