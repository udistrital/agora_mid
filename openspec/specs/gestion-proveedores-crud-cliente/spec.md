# gestion-proveedores-crud-cliente Specification

## Purpose
Permite a la capa de servicios del middleware consumir por HTTP el CRUD de
gestión de proveedores (listar, obtener, crear y actualizar), propagando la
cancelación del contexto, decodificando la respuesta en el `target` del
llamador y devolviendo `(ok, statusCode, err)` con el mismo contrato que el
service de terceros.

## Requirements

### Requirement: Resolución de la URL base de proveedores
El service SHALL leer la URL base del CRUD de gestión de proveedores desde la
clave de configuración `GestionProveedoresCrudHost`, cuyo valor se expande
desde la variable de entorno `GESTION_PROVEEDORES_CRUD_SERVICE` y tiene la
forma `http://host:puerto/v1`.

#### Scenario: Host de proveedores ausente o inválido
- **WHEN** `GestionProveedoresCrudHost` está vacío o no produce una URL analizable
- **THEN** el método no realiza ninguna petición HTTP
- **AND** devuelve `(false, 500, errPublico)` registrando la causa con el service identificado como proveedores

### Requirement: Aislamiento del host por service
Cada service SHALL usar su propio host de configuración. Una invocación al
service de proveedores SHALL dirigirse únicamente al host de proveedores, sin
alcanzar el host de terceros, y viceversa.

#### Scenario: La petición llega solo al servidor de proveedores
- **WHEN** se invoca una operación del service de proveedores con hosts distintos configurados para terceros y proveedores
- **THEN** la petición llega exclusivamente al host de proveedores

### Requirement: Listado de proveedores con target y filtros opcionales
`GetAll` SHALL recibir `target any` y parámetros de consulta opcionales
(`url.Values`). SHALL construir la URL con la utilidad compartida conservando
el prefijo de la URL base y los parámetros previos del endpoint, combinar los
filtros conservando valores repetidos y codificando una sola vez, y ejecutar
`request.GetWithContext(ctx, urlCompleta, target)`.

#### Scenario: Listado sin filtros
- **WHEN** se invoca `GetAll` con `endpoint = "proveedor"` y parámetros `nil`
- **THEN** se realiza una petición `GET` que conserva el prefijo de ruta de la base sin query string añadido

#### Scenario: Listado con filtros repetidos y caracteres especiales
- **WHEN** se invoca `GetAll` con filtros que incluyen valores repetidos y caracteres especiales (por ejemplo `sortby=nombre`, `sortby=fecha`, `query=razon:ACME S.A.`)
- **THEN** la URL conserva los parámetros previos del endpoint, repite cada valor y escapa los caracteres especiales una sola vez

#### Scenario: Decodificación de la lista en el target
- **WHEN** el CRUD responde una lista JSON compatible con `target` (por ejemplo `*[]map[string]any`)
- **THEN** la librería decodifica en `target` y el service devuelve `(true, statusRemoto, nil)`

### Requirement: Obtención de un proveedor con target
`GetOne` SHALL recibir `target any` y el `endpoint` del recurso individual (la
ruta que identifica un proveedor, por ejemplo `proveedor/15`, siguiendo la
convención de terceros), y ejecutar `request.GetWithContext(ctx, urlCompleta, target)`.

#### Scenario: Obtención por endpoint del recurso individual
- **WHEN** se invoca `GetOne` con `endpoint = "proveedor/15"` y un `target` compatible (por ejemplo `*map[string]any`)
- **THEN** se realiza una petición `GET` sobre la URL que conserva el prefijo y, en éxito, la respuesta se decodifica en `target` devolviendo `(true, statusRemoto, nil)`

### Requirement: Creación y actualización de proveedores con target
`Post` y `Put` SHALL recibir `payload any` y `target any`, serializar el
`payload` a JSON con `Content-Type: application/json`, y ejecutar
`request.PostWithContext` y `request.PutWithContext` respectivamente. SHALL
realizar una sola petición por invocación, también cuando falle la
decodificación de la respuesta. El `target` de respuesta es distinto del
`payload` enviado.

#### Scenario: Creación de un proveedor
- **WHEN** se invoca `Post` con un `payload` serializable y un `target` distinto del objeto enviado
- **THEN** se envía una sola petición `POST` con el JSON en el cuerpo y, ante 201, la respuesta se decodifica en `target` devolviendo `(true, 201, nil)`

#### Scenario: Actualización de un proveedor
- **WHEN** se invoca `Put` con un `payload` serializable y un `target`
- **THEN** se envía una sola petición `PUT` con el JSON en el cuerpo y, en éxito, la respuesta se decodifica en `target` devolviendo `(true, statusRemoto, nil)`

#### Scenario: Payload no serializable
- **WHEN** el `payload` no puede serializarse a JSON
- **THEN** no se realiza ninguna petición HTTP
- **AND** el service devuelve `(false, 500, errPublico)`

### Requirement: Propagación del contexto de cancelación
Cada método SHALL recibir `context.Context` como primer parámetro y
propagarlo a la librería, de modo que la cancelación y los deadlines del
llamador cancelen la petición en curso.

#### Scenario: Contexto cancelado
- **WHEN** el `context.Context` recibido se cancela antes de completar la petición
- **THEN** la petición se cancela
- **AND** el service devuelve `(false, 500, errPublico)`

### Requirement: Traducción de resultados compartida con terceros
El service SHALL aplicar la misma traducción de resultados que terceros,
usando las utilidades HTTP compartidas: éxito 200–226 con JSON decodificado,
204 sin contenido reconociendo el EOF de decodificación, rechazo HTTP,
respuesta no decodificable, y fallo sin respuesta remota.

#### Scenario: Éxito con JSON válido
- **WHEN** el CRUD responde un estado 200–226 con cuerpo JSON compatible con `target`
- **THEN** el service devuelve `(true, statusRemoto, nil)` con los datos en `target`

#### Scenario: Respuesta 204 sin contenido
- **WHEN** el CRUD responde `204` sin cuerpo
- **THEN** el service reconoce el EOF de decodificación y devuelve `(true, 204, nil)`

#### Scenario: Rechazo HTTP del CRUD
- **WHEN** el CRUD responde un estado fuera de 200–226 (por ejemplo 400, 404 o 500)
- **THEN** el service devuelve `(false, statusRemoto, errPublico)` sin poblar `target`

#### Scenario: Respuesta no decodificable en el target
- **WHEN** el CRUD responde con éxito HTTP pero el cuerpo está truncado o su tipo no encaja en `target`
- **THEN** el service conserva el status recibido y devuelve `(false, statusRemoto, errPublico)` registrando la causa de decodificación

#### Scenario: Fallo sin respuesta remota
- **WHEN** la conexión es rechazada o el transporte falla antes de recibir una respuesta
- **THEN** el service devuelve `(false, 500, errPublico)`

### Requirement: Error público neutro y registro de la causa
El service SHALL devolver errores públicos neutros (sin host, puerto ni URL
interna en `err.Error()`) y registrar la causa técnica con los logs de Beego
en una sola entrada por fallo, identificando el service como proveedores e
incluyendo etapa, host y puerto cuando estén disponibles y el estado remoto.
El registro SHALL excluir credenciales, tokens, valores de filtros y payloads.

#### Scenario: El log identifica el service y el error omite el host
- **WHEN** una operación de proveedores falla en cualquier etapa (transporte, rechazo HTTP o decodificación)
- **THEN** se escribe una sola entrada de log que identifica `servicio=gestion_proveedores`, la etapa y el host
- **AND** `err.Error()` no contiene host, puerto ni URL interna
