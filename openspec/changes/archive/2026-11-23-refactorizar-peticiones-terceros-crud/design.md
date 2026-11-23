# Design

## Context

Ver `proposal.md` - Why. El servicio de terceros (`services/terceros.go`) y su
soporte (`utils/http_client.go`) hoy implementan a mano la petición: un
`*http.Client` compartido con timeout configurable (`HTTP_CLIENT_TIMEOUT`),
`doRequest`/`Ejecutar` que construyen la URL, serializan el payload, ejecutan y
decodifican el cuerpo en un `utils.Response` fijo. El único consumidor de las
operaciones son sus propias pruebas; no hay controladores ni otros llamadores.

La librería `github.com/udistrital/utils_oas/v2/request` (rama v2, fijada a
**v2.0.2**) expone `GetWithContext`, `PostWithContext`, `PutWithContext`
`(ctx, url, [body,] target) (int, error)`. Hechos verificados en su código
(`request/ctx_request.go`):

- Rango de éxito: `status < 200 || status > http.StatusIMUsed` es error.
  `StatusIMUsed = 226`, es decir **éxito 200–226**.
- Devuelve el error de estado **antes** de decodificar; en rechazo HTTP no
  toca `target`.
- Tras un 2xx siempre llama `json.NewDecoder(resp.Body).Decode(target)`. Con
  cuerpo vacío (204) el decoder devuelve `io.EOF`, envuelto como
  `request.ErrResponseDecode`.
- En fallo de transporte / creación de request / serialización devuelve
  `(0, err)`.
- Integra `xray` y reenvía `Authorization` desde el contexto; su cliente por
  defecto usa `Timeout: 30s` (no configurable).

## Goals / Non-Goals

**Goals:**
- Migrar `GetAll`, `GetOne`, `Post`, `Put` a la librería con el contrato
  `(ctx, endpoint, [payload,] target) (ok, statusCode, err)`.
- Error público neutro + un log con la causa técnica por fallo.
- Conservar `UnirURL` y la combinación de query params; retirar el resto del
  cliente propio y `HTTP_CLIENT_TIMEOUT`.

**Non-Goals:**
- Modelos concretos de tercero (`models/`), controladores, rutas y DELETE.
- Reintentos, cache o circuit breaker.
- Cambiar el mecanismo de `Authorization` (lo maneja la librería vía contexto).

## Decisions

### Firmas y paso de `target`
```go
func GetAll(ctx context.Context, endpoint string, queryParams url.Values, target any) (ok bool, statusCode int, err error)
func GetOne(ctx context.Context, endpoint string, target any)                        (ok bool, statusCode int, err error)
func Post(ctx context.Context, endpoint string, payload any, target any)             (ok bool, statusCode int, err error)
func Put(ctx context.Context, endpoint string, payload any, target any)              (ok bool, statusCode int, err error)
```
`target` se pasa **directamente** a la librería. El llamador ya entregó el
puntero; `&target` añadiría indirección sobre la interfaz. Decisión del usuario:
GetOne se migra al mismo estilo por coherencia.

### Mapeo de resultado de la librería
La librería devuelve `(status, err)`. El servicio traduce:

| Situación | Detección | Salida del servicio |
|---|---|---|
| Éxito 200–226, decodifica | `err == nil` | `(true, status, nil)` |
| 204 sin cuerpo | `status == 204` y `errors.Is(err, ErrResponseDecode)` envolviendo EOF | `(true, 204, nil)` |
| Rechazo HTTP | `status` fuera de 200–226 (err de estado) | `(false, status, errPublico)` |
| No decodificable | `errors.Is(err, ErrResponseDecode)` con status 2xx y no-EOF | `(false, status, errPublico)` |
| Sin respuesta (transporte/URL/serialización) | `status == 0` | `(false, 500, errPublico)` |

El 500 lo asigna el servicio cuando la librería reporta status 0.

### Rango de éxito 200–226
Se adopta el rango de la librería (200–226), más estrecho que el 200–299
anterior. Decisión del usuario: las APIs de la OAS no devuelven estados
mayores. Se documenta como cambio de contrato (era 200–299).

### Reconocer el 204/EOF
Como la librería intenta decodificar siempre, un 204 legítimo llega como
`ErrResponseDecode` envolviendo `io.EOF`. El servicio distingue ese caso:
`status == 204` (o cuerpo vacío con EOF) → éxito sin datos. Se prueba contra
v2.0.2 en la implementación.

### Construcción de la URL
Se conserva `UnirURL(base, endpoint)` (exportada; el servicio la usa desde
otro paquete). `GetAll` analiza la URL unida con
`url.Parse`, funde la consulta previa del endpoint con `queryParams`
(conservando repetidos) y codifica una vez con `Encode()`. Se envía
`u.String()` a `GetWithContext`. `GetOne`/`Post`/`Put` envían la URL unida sin
query añadido.

### Error público y logs
El servicio no propaga el `err` de la librería tal cual (puede contener la URL
completa). Construye un `errPublico` neutro por etapa ("No fue posible conectar
con el servicio de terceros", "No fue posible procesar la respuesta del
servicio de terceros", etc.) y registra una sola entrada con
`beego/v2/core/logs`: operación, método, etapa, host y puerto (cuando estén),
estado remoto y causa concreta, distinguiendo timeout, cancelación, conexión
rechazada, serialización, rechazo HTTP y decodificación. Excluye credenciales,
tokens, valores de filtros y payloads. Si el `err` original trae la URL, se
extrae la causa antes de loguear.

### Limpieza de `utils/http_client.go`
Inventario actual y destino:

| Símbolo | Destino |
|---|---|
| `UnirURL` | **conservar** y exportar (responsabilidad vigente; la usa el servicio) |
| `Ejecutar`, `doRequest` | retirar (sustituidos por la librería) |
| `respuestaError` | retirar (ya no se construye `Response` de error) |
| `clienteHTTP`, `leerTimeout`, `timeoutPorDefecto`, `variableTimeout` | retirar (timeout lo da la librería) |
| `utils.Response`, `utils.Metadata` (`response.go`) | **conservar** (posible `target`) |

### `target` nil o no puntero
Precondición del llamador. La librería devuelve error de `Decode` con nil o
no-puntero; el servicio lo traduce a `(false, status, errPublico)` y lo
registra. Especialmente relevante en escrituras: un POST/PUT puede haber
aplicado el cambio en el CRUD aunque la decodificación falle; el status permite
al llamador saber que la escritura ocurrió.

## Risks / Trade-offs

- [Pérdida del cuerpo de error del CRUD] → La librería no decodifica el cuerpo
  en rechazos. Mitigación: los consumidores usan `statusCode`; no había
  consumidores productivos que dependieran del cuerpo de error.
- [204 tratado como error por la librería] → Si no se reconoce el EOF, un 204
  devolvería `ok=false`. Mitigación: detección explícita de 204/EOF, con prueba
  dedicada contra v2.0.2.
- [Datos parciales en `target`] → Una decodificación fallida puede dejar `target`
  a medias. Mitigación: contrato explícito de no consumir `target` cuando
  `ok=false`; documentado en spec.
- [Fuga de URL interna en el error] → El `err` de la librería incluye la URL.
  Mitigación: nunca se propaga directo; se extrae la causa y se construye un
  mensaje neutro.

## Open Questions

Ninguna.
