# Tasks

## 1. Centralizar utilidades HTTP en `services/http_utils.go`
- [x] 1.1 Crear `services/http_utils.go` (paquete `services`) y trasladar allí la única implementación de `unirURL` (desde `utils/http_client.go`), `construirURL`, `traducir`, `esEOFDecodificacion`, `etapaSinRespuesta`, `registrarLogs`, `hostPuerto` y el tipo `errPublico`, retirándolas de `services/terceros.go`. Verificar con `grep` que cada utilidad tiene una sola definición y que `go build ./services/...` compila.
- [x] 1.2 Eliminar `utils/http_client.go` y reubicar `TestUnirURL` a `services/http_utils_test.go` llamando `unirURL`. Verificar que no quedan referencias a `utils.UnirURL` ni a `utils/http_client.go` y que `go test ./services/...` cubre la unión de URL.
- [x] 1.3 Parametrizar la identidad del service en las utilidades compartidas: pasar un identificador de service (por ejemplo `servicio string`) a `registrarLogs` y `traducir`, y neutralizar el texto del host vacío en `construirURL`. Verificar que el log puede reflejar `servicio=terceros` o `servicio=gestion_proveedores` sin duplicar funciones.
- [x] 1.4 Retirar `var registrar = registrarLogs`: reemplazar los usos por llamadas directas a `registrarLogs`. Verificar con `grep` que no queda la variable ni reasignaciones, y que ninguna otra variable global mutable la sustituye.

## 2. Adaptar el service de terceros a las utilidades compartidas
- [x] 2.1 Actualizar `services/terceros.go` para usar las utilidades compartidas con su identidad (`servicio=terceros`) y sus mensajes públicos, sin volver a declarar utilidades. Verificar que `go build ./services/...` compila y que `services/terceros.go` ya no define ninguna de las utilidades trasladadas.
- [x] 2.2 Ejecutar las pruebas de terceros y verificar que siguen pasando tras el traslado (`go test ./services/... -run 'Terceros|GetAll|GetOne|Post|Put|Decodificacion|Rechazo|Conexion|Target|Payload|Contexto'` o el conjunto equivalente), confirmando que no cambió el comportamiento observable.

## 3. Crear el service de gestion_proveedores_crud
- [x] 3.1 Crear `services/gestion_proveedores.go` con `hostProveedores()` leyendo `GestionProveedoresCrudHost`, y `GetAll`, `GetOne`, `Post`, `Put` con las firmas de terceros (`ctx`, `endpoint`, `url.Values`/`payload`, `target`) que delegan en las utilidades compartidas y en `utils_oas/v2/request`, identificando el service como `gestion_proveedores`. Verificar con `go build ./services/...` que compila.
- [x] 3.2 Verificar con `httptest.NewServer` que cada uno de los cuatro métodos usa el verbo, la ruta y el target esperados, y que `GetOne` consulta el recurso individual por endpoint (por ejemplo `proveedor/15`) frente a `GetAll` que lista (cubre "Obtención de un proveedor con target" y "Listado de proveedores con target y filtros opcionales"). Incluir un ejemplo de `GetOne` y otro de `GetAll` que muestren la diferencia.

## 4. Aislamiento, filtros y escrituras (pruebas)
- [x] 4.1 Verificar con dos `httptest.NewServer` distintos (terceros y proveedores) que una llamada a cada service llega exclusivamente a su propio servidor (cubre "Aislamiento del host por service").
- [x] 4.2 Verificar que `GetAll` de proveedores conserva el prefijo, los parámetros previos del endpoint, los valores repetidos y los caracteres especiales (cubre "Listado de proveedores con target y filtros opcionales").
- [x] 4.3 Verificar que `Post` y `Put` de proveedores envían el payload una sola vez, decodifican la respuesta en un `target` distinto del enviado, y que un payload no serializable genera cero solicitudes devolviendo `(false, 500, errPublico)` (cubre "Creación y actualización de proveedores con target").

## 5. Tratamiento de resultados y logs compartidos (pruebas)
- [x] 5.1 Verificar que proveedores aplica el mismo tratamiento que terceros para: 204 vacío → `(true, 204, nil)`, rechazo HTTP → `(false, statusRemoto, errPublico)`, JSON incompatible → `(false, statusRemoto, errPublico)`, y fallo de conexión → `(false, 500, errPublico)` (cubre "Traducción de resultados compartida con terceros").
- [x] 5.2 Verificar que ante un fallo el log identifica `servicio=gestion_proveedores` y el host, mientras que `err.Error()` omite host, puerto y URL; y que el log excluye credenciales, tokens, payloads y valores de filtros (cubre "Error público neutro y registro de la causa"). Usar el mecanismo de captura de logs sin variable global mutable.
- [x] 5.3 Verificar la propagación de cancelación en proveedores: un contexto cancelado antes de completar devuelve `(false, 500, errPublico)` (cubre "Propagación del contexto de cancelación").

## 6. Verificación de integración
- [x] 6.1 Ejecutar `go build ./utils/... ./services/...`, `go test ./utils/... ./services/...` y `go vet ./utils/... ./services/...` y verificar que compila, todas las pruebas (terceros y proveedores) pasan y vet está limpio.
