# Spec Delta

## Purpose

Permite a la capa de servicios del middleware consumir por HTTP el CRUD de
terceros (listar, obtener, crear y actualizar), propagando la cancelación del
contexto y entregando un resultado con el formato de respuesta estándar de la
OAS que distingue el estado del CRUD del resultado observado por el consumidor.

## ADDED Requirements

### Requirement: Formato de respuesta estándar de la OAS
El servicio SHALL entregar el cuerpo como la estructura `Response` de la OAS,
con los campos `Success bool`, `Status int`, `Message string`, `Data any` y
`Metadata *Metadata` (opcional, omitido con `omitempty`). `Metadata` contiene
la paginación (`total`, `limit`, `offset`, `currentPage`, `totalPages`,
`hasNextPage`, `hasPreviousPage`) y SHALL poblarse únicamente en respuestas de
`GetAll`; en `GetOne`, `Post` y `Put` SHALL quedar nil.

#### Scenario: Cuerpo del CRUD conforme a Response
- **WHEN** el CRUD responde un cuerpo con la forma `Response` de la OAS
- **THEN** el servicio entrega ese `Response` con sus campos `Success`, `Status`, `Message` y `Data`

#### Scenario: Metadata solo en GetAll
- **WHEN** la respuesta proviene de `GetOne`, `Post` o `Put`
- **THEN** el campo `Metadata` del `Response` entregado es nil

#### Scenario: Metadata de paginación en GetAll
- **WHEN** el CRUD responde a `GetAll` con el bloque de paginación
- **THEN** el `Response` entregado incluye `Metadata` con los campos de paginación poblados

### Requirement: Resolución de la URL base desde la configuración
El servicio SHALL leer la URL base del CRUD de terceros desde la clave de
configuración `TercerosCrudHost`, cuyo valor se expande desde la variable de
entorno `TERCEROS_CRUD_SERVICE` y tiene la forma `http://host:puerto/v1`
(esquema, host con puerto o dominio, y prefijo de versión).

#### Scenario: Host ausente
- **WHEN** `TercerosCrudHost` está vacío o no está definido
- **THEN** el método no realiza ninguna petición HTTP
- **AND** retorna `ok = false`, `statusCode = 500` y un `Response` con `Success = false`, `Message` con el detalle del error y `Data` vacío

#### Scenario: Host inválido
- **WHEN** `TercerosCrudHost` no es una URL analizable
- **THEN** el método no realiza ninguna petición HTTP
- **AND** retorna `ok = false`, `statusCode = 500` y un `Response` con `Success = false`, `Message` con el detalle del error y `Data` vacío

### Requirement: Construcción de la URL conservando el prefijo de ruta
El servicio SHALL unir la URL base y el `endpoint` relativo produciendo una
única barra de separación y conservando cualquier prefijo de ruta de la base,
sin importar si la base termina en barra o el endpoint empieza con barra.

#### Scenario: Base con barra final y endpoint con barra inicial
- **WHEN** la base es `https://crud.example/api/v1/` y el endpoint es `/tercero/15`
- **THEN** la URL solicitada es `https://crud.example/api/v1/tercero/15`

#### Scenario: Base sin barra final y endpoint sin barra inicial
- **WHEN** la base es `https://crud.example/api/v1` y el endpoint es `tercero/15`
- **THEN** la URL solicitada es `https://crud.example/api/v1/tercero/15`

### Requirement: Listado con parámetros de consulta opcionales
`GetAll` SHALL aceptar parámetros de consulta opcionales y codificarlos en la
URL. El llamador no concatena el query string; pasa `nil` cuando no hay
filtros o un conjunto de parámetros que admite valores repetidos.

#### Scenario: Listado sin filtros
- **WHEN** se invoca `GetAll` con `endpoint = "/tercero/"` y parámetros `nil`
- **THEN** se realiza una petición `GET` a `/tercero/` sin query string

#### Scenario: Listado con filtros, espacios y caracteres especiales
- **WHEN** se invoca `GetAll` con un parámetro cuyo valor contiene espacios y acentos (por ejemplo `nombre:Juan Pérez`)
- **THEN** el valor se codifica de forma segura en la URL (espacios y caracteres especiales escapados)

#### Scenario: Listado con valores repetidos para un mismo parámetro
- **WHEN** se invoca `GetAll` con un parámetro que tiene varios valores (por ejemplo `sortby=nombre` y `sortby=fecha`)
- **THEN** la URL incluye el parámetro repetido una vez por cada valor

### Requirement: Obtención de un recurso
`GetOne` SHALL ejecutar una petición `GET` sobre el `endpoint` indicado para
obtener un único recurso del CRUD.

#### Scenario: Obtención por endpoint
- **WHEN** se invoca `GetOne` con `endpoint = "/tercero/15"`
- **THEN** se realiza una petición `GET` a `/tercero/15`

### Requirement: Creación y actualización con cuerpo JSON
`Post` y `Put` SHALL serializar el `payload` a JSON, enviarlo en el cuerpo de
la petición con la cabecera `Content-Type: application/json`, y ejecutar su
verbo HTTP respectivo (`POST` para crear, `PUT` para actualizar).

#### Scenario: Creación de un recurso
- **WHEN** se invoca `Post` con un payload serializable a JSON
- **THEN** se realiza una petición `POST` con el JSON en el cuerpo y `Content-Type: application/json`

#### Scenario: Actualización de un recurso
- **WHEN** se invoca `Put` con un payload serializable a JSON
- **THEN** se realiza una petición `PUT` con el JSON en el cuerpo y `Content-Type: application/json`

#### Scenario: Payload no serializable
- **WHEN** el `payload` no puede serializarse a JSON
- **THEN** el método no realiza ninguna petición HTTP
- **AND** retorna `ok = false`, `statusCode = 500` y un `Response` con `Success = false`, `Message` con el detalle del error y `Data` vacío

### Requirement: Propagación del contexto de cancelación
Cada método SHALL recibir `context.Context` como primer parámetro y asociarlo
a la petición HTTP, de modo que la cancelación del contexto cancele la
petición en curso.

#### Scenario: Contexto cancelado
- **WHEN** el `context.Context` recibido se cancela antes de completar la petición
- **THEN** la petición HTTP se cancela
- **AND** el método retorna `ok = false`, `statusCode = 500` y un `Response` con `Success = false`, `Message` con el detalle del error y `Data` vacío

### Requirement: Resultado exitoso del CRUD
El servicio SHALL retornar `ok = true` cuando el CRUD responde con un estado
HTTP entre 200 y 299 y el cuerpo se lee correctamente, devolviendo siempre el
`statusCode` recibido del CRUD. El cuerpo JSON conforme a `Response` se entrega
como tal; una respuesta sin cuerpo (por ejemplo, 204) se entrega como un
`Response` con `Success = true`, el `statusCode` recibido y `Data` vacío.

#### Scenario: Respuesta 2xx con cuerpo Response
- **WHEN** el CRUD responde con un estado entre 200 y 299 y un cuerpo JSON conforme a `Response`
- **THEN** el método retorna `ok = true`, el `statusCode` recibido y el `Response` decodificado

#### Scenario: Respuesta 204 sin cuerpo
- **WHEN** el CRUD responde `204` con cuerpo vacío
- **THEN** el método retorna `ok = true`, `statusCode = 204` y un `Response` con `Success = true` y `Data` vacío

### Requirement: Resultado de error del CRUD
El servicio SHALL retornar `ok = false` cuando el CRUD responde con un estado
HTTP fuera del rango 200 a 299, devolviendo el `statusCode` recibido. El cuerpo
se entrega como el `Response` decodificado cuando es JSON conforme a `Response`;
cuando el cuerpo está vacío o no encaja en `Response`, el servicio construye un
`Response` con `Success = false`, `Message` con el detalle del error y `Data`
vacío.

#### Scenario: Error con cuerpo Response
- **WHEN** el CRUD responde `400`, `404` o `500` con un cuerpo JSON conforme a `Response`
- **THEN** el método retorna `ok = false`, el `statusCode` recibido y el `Response` decodificado

#### Scenario: Error con cuerpo vacío o no conforme a Response
- **WHEN** el CRUD responde con un estado de error y el cuerpo está vacío o no encaja en `Response`
- **THEN** el método retorna `ok = false`, el `statusCode` recibido y un `Response` con `Success = false`, `Message` con el detalle del error y `Data` vacío

### Requirement: Fallo de transporte sin respuesta HTTP
El servicio SHALL retornar `ok = false`, `statusCode = 500` y un `Response` con
`Success = false`, `Message` con el detalle del error y `Data` vacío cuando no
se obtiene ninguna respuesta HTTP del CRUD.

#### Scenario: Fallo de conexión
- **WHEN** la petición no logra establecer conexión o el transporte falla antes de recibir cabeceras
- **THEN** el método retorna `ok = false`, `statusCode = 500` y un `Response` con `Success = false`, `Message` con el detalle del error y `Data` vacío

### Requirement: Fallo al leer el cuerpo de la respuesta
El servicio SHALL retornar `ok = false`, el `statusCode` recibido y un
`Response` con `Success = false`, `Message` con el detalle del error y `Data`
vacío cuando recibe las cabeceras de la respuesta pero la lectura del cuerpo
falla, de modo que el llamador conserva el estado conocido sin interpretar un
cuerpo incompleto como válido.

#### Scenario: Cabeceras recibidas y lectura interrumpida
- **WHEN** el CRUD responde con un estado pero la lectura del cuerpo falla (conexión interrumpida o cuerpo truncado)
- **THEN** el método retorna `ok = false`, el `statusCode` recibido y un `Response` con `Success = false`, `Message` con el detalle del error y `Data` vacío

### Requirement: Timeout configurable del cliente HTTP
El cliente HTTP compartido SHALL aplicar un timeout configurable por variable
de entorno, con un valor por defecto de 30 segundos cuando no se configura.

#### Scenario: Timeout por defecto
- **WHEN** no se configura el timeout del cliente HTTP
- **THEN** el cliente aplica un timeout de 30 segundos

#### Scenario: Timeout configurado
- **WHEN** se configura el timeout mediante la variable de entorno `HTTP_CLIENT_TIMEOUT`
- **THEN** el cliente aplica el valor configurado
