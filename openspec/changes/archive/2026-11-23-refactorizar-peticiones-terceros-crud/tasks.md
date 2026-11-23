# Tasks

## 1. Dependencia y limpieza del cliente propio
- [x] 1.1 Agregar `github.com/udistrital/utils_oas/v2 v2.0.2` con `go get` y verificar que `go.mod`/`go.sum` la fijan y que `go build ./utils/... ./services/...` resuelve el import de `request`.
- [x] 1.2 En `utils/http_client.go` retirar `Ejecutar`, `doRequest`, `respuestaError`, `clienteHTTP`, `leerTimeout`, `timeoutPorDefecto` y `variableTimeout`; conservar la unión de URL como única función exportada `UnirURL`. Verificar que `go build ./utils/...` compila y que `UnirURL` sigue cubierto por `TestUnirURL`.
- [x] 1.3 Conservar `utils.Response`/`utils.Metadata` en `utils/response.go` y mantener `TestResponseMetadataOmitempty`. Verificar con `go test ./utils/...` que la prueba de serialización sigue pasando.
- [x] 1.4 Retirar `HTTP_CLIENT_TIMEOUT` de toda referencia activa (configuración, despliegue, documentación y pruebas). Verificar con `grep -r HTTP_CLIENT_TIMEOUT` que no quedan referencias activas.

## 2. Nuevo contrato del servicio y mapeo de resultados
- [x] 2.1 Reescribir `GetAll(ctx, endpoint, queryParams, target)` usando `utils.UnirURL` + `url.Parse`, fusionando la consulta previa con `queryParams` (repetidos, `Encode()` una vez) y llamando `request.GetWithContext(ctx, u.String(), target)`. Verificar con `httptest.NewServer` que conserva prefijo, consulta previa, valores repetidos y caracteres especiales, y que decodifica una lista en `*[]map[string]any` devolviendo `(true, status, nil)` (cubre "Listado con target y parámetros de consulta opcionales").
- [x] 2.2 Reescribir `GetOne(ctx, endpoint, target)` con `request.GetWithContext`. Verificar con `httptest.NewServer` que `endpoint="tercero/15"` produce `GET` sobre la URL con prefijo y decodifica en `*map[string]any` devolviendo `(true, status, nil)` (cubre "Obtención de un recurso con target").
- [x] 2.3 Reescribir `Post(ctx, endpoint, payload, target)` y `Put(ctx, endpoint, payload, target)` con `request.PostWithContext`/`request.PutWithContext`. Verificar con `httptest.NewServer` que Post envía el payload una sola vez y decodifica la respuesta 201 en un `target` distinto del objeto enviado, y que un payload no serializable genera cero solicitudes y devuelve `(false, 500, errPublico)` (cubre "Creación y actualización con target").
- [x] 2.4 Implementar el mapeo común de `(status, err)` de la librería a `(ok, statusCode, err)`: éxito 200–226, rechazo HTTP, status 0 → 500, y reconocimiento del 204/EOF como `(true, 204, nil)`. Verificar con `httptest.NewServer` que un 204 vacío en `Put` devuelve `(true, 204, nil)` y que un rechazo HTTP conserva el estado remoto devolviendo error público (cubre "Rango de éxito 200 a 226", "Respuesta 204 sin contenido" y "Rechazo HTTP del CRUD").
- [x] 2.5 Propagar `context.Context` a la librería en las cuatro operaciones. Verificar con prueba que un contexto cancelado antes de completar devuelve `(false, 500, errPublico)` (cubre "Propagación del contexto de cancelación").

## 3. Error público, target y logs
- [x] 3.1 Construir el `errPublico` neutro por etapa y asegurar que `target` solo es confiable con `ok=true`. Verificar con `httptest.NewServer` que un JSON truncado o incompatible conserva el estado recibido, devuelve `ok=false` y que `err.Error()` no revela host/puerto/URL (cubre "Respuesta no decodificable en el target", "Contrato de salida con target y error" y "Error público neutro").
- [x] 3.2 Verificar la precondición del `target`: un `target` nil o no puntero ante respuesta exitosa devuelve `(false, status, errPublico)` registrando la causa (cubre "Precondición del target").
- [x] 3.3 Registrar la causa técnica con `beego/v2/core/logs`: una sola entrada por fallo, con operación/método/etapa/host/puerto/estado/causa, excluyendo credenciales, tokens, filtros y payloads, extrayendo la causa cuando el error trae la URL. Verificar con `httptest.NewServer` que una conexión rechazada devuelve `(false, 500, errPublico)`, identifica el host en el log y lo mantiene fuera del mensaje público, y que se escribe una sola entrada (cubre "Registro de la causa técnica", "Fallo sin respuesta HTTP" y "Error público neutro").

## 4. Verificación de integración
- [x] 4.1 Ejecutar `go build ./utils/... ./services/...`, `go test ./utils/... ./services/...` y `go vet ./utils/... ./services/...` y verificar que compila, todas las pruebas pasan y vet está limpio.
