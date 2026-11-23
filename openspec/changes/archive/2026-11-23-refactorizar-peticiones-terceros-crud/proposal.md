# Proposal

## Why

El servicio de terceros hoy arma la petición a mano con un `*http.Client`
propio, decodifica el cuerpo en un `utils.Response` fijo y obliga al llamador
a navegar un mapa intermedio. La OAS provee `github.com/udistrital/utils_oas/v2/request`,
que estandariza las peticiones HTTP con contexto, integra trazas (xray),
reenvía el token de autorización y decodifica la respuesta directamente en el
tipo que declara el llamador. Migrar reduce código propio, alinea el middleware
con el resto de la OAS y deja que cada llamador reciba la respuesta ya tipada.

## What Changes

- **BREAKING**: cambia el contrato de `GetAll`, `GetOne`, `Post` y `Put`. En
  lugar de devolver `body utils.Response`, cada método recibe `target any`
  (puntero provisto por el llamador) y devuelve `(ok bool, statusCode int, err error)`.
  El llamador consume `target` solo cuando `ok` es true.
- Las peticiones pasan a `request.GetWithContext`, `request.PostWithContext` y
  `request.PutWithContext` de `utils_oas/v2` (v2.0.2). Se agrega la dependencia.
- El rango de éxito pasa de 200–299 a **200–226** (el que aplica la librería).
  Las APIs de la OAS no devuelven estados mayores.
- Se elimina el mapa intermedio y el retorno del cuerpo. La construcción de la
  URL conserva `UnirURL` y la combinación de query params; la ejecución y
  decodificación quedan en la librería.
- Se retira `HTTP_CLIENT_TIMEOUT` (variable, lectura, default y validaciones):
  el timeout lo aporta la librería y se documenta su relación con el contexto.
- Se retiran de `utils/http_client.go` las funciones sustituidas por la
  librería (`Ejecutar`, `doRequest`, `respuestaError`, el `*http.Client`
  compartido y `leerTimeout`). Se conserva `UnirURL`.
- Se conservan `utils.Response` y `utils.Metadata`: siguen siendo un tipo de
  dominio válido que un llamador puede pasar como `target` si el CRUD devuelve
  ese envelope.
- Los errores devueltos son públicos y neutros (sin host, puerto ni URL
  interna). La causa técnica se registra con `beego/v2/core/logs`.

Fuera de alcance: definir modelos concretos de tercero (`models/`), controladores
y rutas públicas, y el verbo DELETE.

## Capabilities

### New Capabilities
<!-- Ninguna. -->

### Modified Capabilities
- `terceros-crud-cliente`: cambia el contrato de salida de las operaciones
  (de `body utils.Response` a `target any` + `err error`), la fuente de la
  ejecución HTTP (librería `utils_oas/v2/request`), el rango de éxito
  (200–226), el manejo de errores (error público + log de la causa) y retira
  el timeout configurable del cliente propio.

## Impact

- **Código**: `services/terceros.go` (nuevas firmas y mapeo de resultados),
  `utils/http_client.go` (retiro de funciones y del cliente propio; conserva
  `UnirURL`), `services/terceros_test.go` y `utils/http_client_test.go`
  (pruebas actualizadas al nuevo contrato).
- **Dependencias**: se agrega `github.com/udistrital/utils_oas/v2 v2.0.2`
  (arrastra `request` y `xray`). `go.mod`/`go.sum` cambian.
- **Configuración/despliegue/documentación**: se retira `HTTP_CLIENT_TIMEOUT`
  de toda referencia activa. Se mantiene la clave `TercerosCrudHost`.
- **Consumidores**: el único uso actual de las operaciones son sus propias
  pruebas; no hay controladores ni otros llamadores productivos. El cambio de
  firma es incompatible para cualquier consumidor futuro que asuma el contrato
  anterior.
