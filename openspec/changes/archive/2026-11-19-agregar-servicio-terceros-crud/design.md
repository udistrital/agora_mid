# Design

## Context

Ver `proposal.md` - Why. El repositorio es un esqueleto de Beego v2: no existe
capa de servicios ni cliente HTTP en Go (`services/`, `controllers/` y
`models/` están vacíos; no hay usos de `net/http`, `os.Getenv` ni
`web.AppConfig`). La clave `TercerosCrudHost` ya está declarada en
`conf/app.conf` y se expande desde la variable de entorno
`TERCEROS_CRUD_SERVICE`. El contrato del CRUD vive en
`docs/contratos/terceros_crud/swagger.yml` (`basePath: /v1`): confirma los
recursos (`tercero`, `datos_identificacion`, etc.), que los GET-all aceptan
query params (`query`, `fields`, `sortby`, `order`, `limit`, `offset`), que
POST crea (201) y PUT actualiza por id (200).

Restricción del proyecto (`openspec/config.yaml`): los controladores Beego
pasan su `context.Context` a la capa de servicios y los clientes HTTP
conservan ese contexto.

## Goals / Non-Goals

**Goals:**
- Un paquete `utils/` con el cliente HTTP compartido, el helper de unión de
  URL y la ejecución/decodificación de la petición, reutilizable por futuros
  servicios CRUD.
- Un servicio `services/terceros.go` con `GetAll`, `GetOne`, `Post` y `Put`,
  todos con contrato de salida uniforme `(ok bool, statusCode int, body Response)`,
  usando el formato de respuesta estándar de la OAS.
- Pruebas deterministas con `httptest.NewServer`, sin un CRUD en ejecución.

**Non-Goals:**
- Controladores, rutas públicas y verbos `DELETE` o variantes especiales del
  contrato.
- Reintentos, backoff, cache o circuit breaker.
- Tipado fuerte de los modelos de tercero (el `Data` del `Response` se maneja
  como `any`).

## Decisions

### Firmas con `context.Context` como primer parámetro
Las cuatro firmas son:

```go
func GetAll(ctx context.Context, endpoint string, queryParams url.Values) (ok bool, statusCode int, body Response)
func GetOne(ctx context.Context, endpoint string)                        (ok bool, statusCode int, body Response)
func Post(ctx context.Context, endpoint string, payload any)             (ok bool, statusCode int, body Response)
func Put(ctx context.Context, endpoint string, payload any)              (ok bool, statusCode int, body Response)
```

Rationale: la convención del proyecto exige propagar el contexto del
controlador. Se usa `http.NewRequestWithContext`. Alternativa descartada:
firmas sin `ctx`, que impediría cancelar la petición al cancelarse la
solicitud entrante.

### `queryParams url.Values` en lugar de `map[string]any`
`url.Values` (`map[string][]string`) resuelve de forma nativa los valores
repetidos y la codificación de espacios y caracteres especiales vía
`.Encode()`. Un `map[string]any` no soporta valores repetidos sin trabajo
extra. El llamador pasa `nil` cuando no hay filtros.

### Cuerpo de salida con el formato `Response` de la OAS
El cuerpo se tipa con la estructura de respuesta estándar de la OAS (la misma
de `gestion_contractual_crud/controllers/base.go`), definida en `utils/`:

```go
type Response struct {
    Success  bool      `json:"Success"`
    Status   int       `json:"Status"`
    Message  string    `json:"Message"`
    Data     any       `json:"Data"`
    Metadata *Metadata `json:"Metadata,omitempty"`
}

type Metadata struct {
    Total           int64 `json:"total"`
    Limit           int64 `json:"limit"`
    Offset          int64 `json:"offset"`
    CurrentPage     int64 `json:"currentPage"`
    TotalPages      int64 `json:"totalPages"`
    HasNextPage     bool  `json:"hasNextPage"`
    HasPreviousPage bool  `json:"hasPreviousPage"`
}
```

Reglas de construcción del `body`:
- 2xx con cuerpo JSON conforme a `Response` → `Response` decodificado.
- 2xx sin cuerpo (204) → `Response{Success: true, Status: <code>, Data: nil}`.
- Error del CRUD con cuerpo JSON conforme a `Response` → `Response` decodificado.
- Error del CRUD con cuerpo vacío o no conforme a `Response`, y todo caso sin
  respuesta interpretable (host ausente/inválido, payload no serializable,
  fallo de conexión, contexto cancelado, fallo al leer el cuerpo) →
  `Response{Success: false, Status: <code>, Message: <detalle del error>, Data: nil}`.

`Metadata` se puebla únicamente en `GetAll` (es la paginación que el CRUD
devuelve en el listado); en `GetOne`, `Post` y `Put` queda nil y se omite por
`omitempty`. Como el CRUD ya responde con este formato, en el camino feliz el
servicio decodifica directamente el cuerpo en `Response`, incluido su
`Metadata`.

Rationale: alinea el servicio con el formato de respuesta estándar de la OAS,
de modo que el consumidor del middleware recibe `Success`/`Status`/`Message`/
`Data`/`Metadata` sin reinterpretar `map[string]any`. En los casos sin cuerpo
interpretable el servicio sintetiza un `Response` de error (`Success=false`,
`Message` con el detalle) en lugar de devolver `nil`, para que el consumidor
siempre reciba la misma forma. Alternativas descartadas: `map[string]any`
(obligaría al consumidor a navegar claves) y `[]byte`/`any` (sin la estructura
estándar).

### Unión de host y endpoint
`strings.TrimRight(base, "/") + "/" + strings.TrimLeft(endpoint, "/")`. Se
descarta `url.ResolveReference` solo, porque con un endpoint absoluto
(`/tercero/15`) descartaría el prefijo de ruta de la base (`/api/v1`). La
construcción de la URL (incluido el query string) se hace tras unir las
partes y validar que la base es analizable.

### Función interna compartida en `utils/`
Los cuatro métodos delegan en un helper interno `doRequest(ctx, method,
endpoint, payload)` que: resuelve y valida la base, une la URL, serializa el
payload (si lo hay) y fija `Content-Type`, crea la petición con contexto, la
ejecuta con el cliente compartido, lee y decodifica el cuerpo, y calcula `ok`
con `statusCode >= 200 && statusCode <= 299`. `GetAll` añade el query string
antes de delegar. Reutilizable por el futuro servicio de proveedores
(`GestionProveedoresCrudHost`).

### Cliente HTTP y timeout
Un único `*http.Client` compartido en `utils/`, con `Timeout` leído de la
variable de entorno `HTTP_CLIENT_TIMEOUT` (default 30s). Un solo cliente
reutiliza conexiones (keep-alive) y evita crear uno por llamada.

## Risks / Trade-offs

- [Pérdida del cuerpo de error no conforme a `Response`] → Si el CRUD responde
  un error con un cuerpo que no encaja en `Response`, el servicio lo reemplaza
  por un `Response` sintético (`Success=false`, `Message` con el detalle) y no
  conserva el cuerpo original. Mitigación: el `statusCode` y `Message`
  bastan para que el consumidor decida; los CRUD de la OAS ya responden en
  formato `Response`, así que el caso no conforme es excepcional.
- [Mensaje de error sintético poco descriptivo] → En los casos sin respuesta
  interpretable el `Message` lo genera el servicio, no el CRUD. Mitigación:
  incluir en `Message` el detalle del error de transporte/serialización para
  no perder la causa.
- [Simular el fallo de lectura del cuerpo en pruebas] → Requiere forzar una
  respuesta con cuerpo truncado. Mitigación: en el test, responder con un
  `Content-Length` mayor que el cuerpo real o cerrar la conexión del
  `httptest.Server` tras enviar cabeceras.

## Open Questions

Ninguna.
