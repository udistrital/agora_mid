# Proposal

## Why

El gateway WSO2 impone límites y transformaciones sobre los parámetros que viajan en la URL (longitud máxima, codificación y normalización de caracteres por proxies/gateways). Una selección de campos extensa en el `GET /v1/proveedor?fields=...` puede superar el límite de URL o ser alterada al pasar por el gateway, fallando en despliegue aunque funcione en local. Se migra la lectura a un endpoint de consulta por cuerpo JSON para mantener la URL estable y manejar los nombres de campos como texto JSON.

## What Changes

- Se introduce la lectura del proveedor mediante `POST /v1/proveedor/consulta`, que recibe `id` y `fields` en el cuerpo JSON y conserva el contrato de respuesta OAS plano (200/206 con `MetaData`).
- `fields` viaja como arreglo JSON de rutas canónicas (por ejemplo `["Id","Tercero.Id","Tercero.NombreCompleto"]`); se valida `id` y cada ruta contra la lista permitida (allow-list estricto, nombres canónicos) antes de consultar el CRUD dueño. No se normalizan variantes: un nombre no canónico se rechaza con `400`.
- `POST /v1/proveedor/consulta` solo acepta POST; `GET`, `PUT` y `DELETE` sobre esa ruta responden `405 Method Not Allowed`.
- Se establece la convención `{recurso}/consulta` para la lectura por cuerpo POST, de modo que otros recursos gestionados por el middleware (por ejemplo terceros) sigan la misma forma.
- **BREAKING**: se elimina el verbo `GET /v1/proveedor` como ruta de lectura. La eliminación es el último grupo de trabajo del cambio, de modo que el microfrontend migre componente a componente contra el nuevo POST antes de retirar el GET. Tras la eliminación, `GET` (y `PUT`, `DELETE`) sobre `/v1/proveedor` responden `405`.
- Se actualiza la definición OpenAPI (`swagger/swagger.yml`), que WSO2 consume vía su configuración JSON de enrutamiento: se agrega el path `POST /v1/proveedor/consulta` y, en el grupo final, se retira `GET /v1/proveedor`.

## Capabilities

### New Capabilities
<!-- Ninguna capacidad nueva: se modifican capacidades existentes. -->

### Modified Capabilities
- `proveedor-read`: la lectura por identificador y la selección de campos migran del `GET` con query string a `POST /v1/proveedor/consulta` con `id` y `fields` en el cuerpo JSON; se agrega el manejo de métodos no permitidos (405) y se retira la ruta GET de lectura.

<!-- `openapi-publication` no cambia a nivel de requisitos: sus requisitos describen la derivación YAML->JSON, la idempotencia de arranque y la publicación en ambos formatos, todo independiente de los paths concretos. Agregar `POST /v1/proveedor/consulta` y retirar `GET /v1/proveedor` en `swagger/swagger.yml` es un detalle de contrato/implementación (ver Impact, design y tasks), no un cambio de comportamiento de esa capacidad. -->


## Impact

- **APIs**: nuevo `POST /v1/proveedor/consulta`; eliminación de `GET /v1/proveedor` como lectura; `405` para métodos no soportados. La escritura `POST`/`PATCH` sobre `/v1/proveedor` no cambia.
- **Código**: `controllers/proveedor.go` (nuevo manejador de consulta y decodificación/validación del cuerpo, retiro de `Leer`), enrutado en `main.go` (nueva ruta `/v1/proveedor/consulta`, ajuste del enrutado raíz tras retirar GET). La firma del servicio `LeerProveedor(ctx, id, fields)` se conserva.
- **Contrato/Gateway**: `swagger/swagger.yml` como fuente de verdad que WSO2 consume para enrutar.
- **Pruebas**: se portan las pruebas de tabla de la lectura a solicitudes basadas en cuerpo; se agregan casos de `405` y de cuerpo JSON malformado; se retiran las pruebas específicas del GET en el grupo final.
- **Consumidores**: el microfrontend co-desarrollado migra sus llamadas de lectura al POST antes de retirar el GET. Todo es local; no hay contrato publicado en producción.
