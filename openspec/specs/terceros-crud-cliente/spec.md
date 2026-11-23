# terceros-crud-cliente Specification

## Purpose

Permite a la capa de servicios del middleware consumir por HTTP el CRUD de
terceros (listar, obtener, crear y actualizar), propagando la cancelación del
contexto y entregando un resultado con el formato de respuesta estándar de la
OAS que distingue el estado del CRUD del resultado observado por el consumidor.

## Requirements

### Requirement: Propagación del contexto de cancelación
Cada método SHALL recibir `context.Context` como primer parámetro y
propagarlo a la librería, de modo que la cancelación y los deadlines del
llamador cancelen la petición en curso.

#### Scenario: Contexto cancelado
- **WHEN** el `context.Context` recibido se cancela antes de completar la petición
- **THEN** la petición se cancela
- **AND** el servicio devuelve `(false, 500, errPublico)`

### Requirement: Listado con target y parámetros de consulta opcionales
`GetAll` SHALL recibir `target any` (puntero provisto por el llamador) y
parámetros de consulta opcionales. SHALL construir la URL con `UnirURL`,
analizarla con `url.Parse`, combinar la consulta existente con `queryParams`
conservando valores repetidos y codificando una sola vez con `url.Values.Encode()`,
y enviar la URL completa a `request.GetWithContext(ctx, u.String(), target)`.
Con `nil` o filtros vacíos SHALL conservar la consulta original.

#### Scenario: Listado sin filtros
- **WHEN** se invoca `GetAll` con `endpoint = "tercero"` y parámetros `nil`
- **THEN** se realiza una petición `GET` que conserva el prefijo de ruta de la base sin query string añadido

#### Scenario: Listado con filtros, espacios y caracteres especiales
- **WHEN** se invoca `GetAll` con un parámetro cuyo valor contiene espacios y acentos (por ejemplo `nombre:Juan Pérez`)
- **THEN** el valor se codifica de forma segura en la URL una sola vez (espacios y caracteres especiales escapados)

#### Scenario: Listado con valores repetidos y consulta previa
- **WHEN** el `endpoint` ya trae una consulta y `queryParams` aporta un parámetro con varios valores (por ejemplo `sortby=nombre` y `sortby=fecha`)
- **THEN** la URL conserva la consulta previa y añade el parámetro repetido una vez por cada valor

#### Scenario: Decodificación en el target del llamador
- **WHEN** el CRUD responde una lista JSON compatible con el `target` (por ejemplo `*[]map[string]any`)
- **THEN** la librería decodifica la respuesta en `target` y el servicio devuelve `(true, statusRemoto, nil)`

### Requirement: Obtención de un recurso con target
`GetOne` SHALL recibir `target any` y ejecutar `request.GetWithContext(ctx, urlCompleta, target)`
sobre el `endpoint` indicado, conservando el prefijo de ruta de la base.

#### Scenario: Obtención por endpoint
- **WHEN** se invoca `GetOne` con `endpoint = "tercero/15"` y un `target` compatible (por ejemplo `*map[string]any`)
- **THEN** se realiza una petición `GET` sobre la URL que conserva el prefijo y, en éxito, la respuesta se decodifica en `target` devolviendo `(true, statusRemoto, nil)`

### Requirement: Creación y actualización con target
`Post` y `Put` SHALL recibir `payload any` y `target any`, serializar el
`payload` a JSON con `Content-Type: application/json`, y ejecutar su verbo
respectivo mediante `request.PostWithContext(ctx, urlCompleta, payload, target)`
y `request.PutWithContext(ctx, urlCompleta, payload, target)`. SHALL realizar
una sola petición por invocación, también cuando falle la decodificación de la
respuesta.

#### Scenario: Creación de un recurso
- **WHEN** se invoca `Post` con un `payload` serializable y un `target` distinto del objeto enviado
- **THEN** se envía una sola petición `POST` con el JSON en el cuerpo y, ante 201, la respuesta se decodifica en `target` devolviendo `(true, 201, nil)`

#### Scenario: Actualización de un recurso
- **WHEN** se invoca `Put` con un `payload` serializable y un `target`
- **THEN** se envía una sola petición `PUT` con el JSON en el cuerpo y, en éxito, la respuesta se decodifica en `target` devolviendo `(true, statusRemoto, nil)`

#### Scenario: Payload no serializable
- **WHEN** el `payload` no puede serializarse a JSON
- **THEN** no se realiza ninguna petición HTTP
- **AND** el servicio devuelve `(false, 500, errPublico)`

### Requirement: Contrato de salida con target y error
Las operaciones `GetAll`, `GetOne`, `Post` y `Put` SHALL devolver
`(ok bool, statusCode int, err error)` y entregar los datos en el `target`
provisto por el llamador. El servicio SHALL pasar `target` directamente a la
librería, sin añadir indirección. El llamador SHALL consumir `target`
únicamente cuando `ok` es true.

#### Scenario: Éxito entrega datos en target y ok true
- **WHEN** una operación tiene éxito HTTP y la respuesta se decodifica
- **THEN** el servicio devuelve `ok = true`, el `statusCode` remoto y `err = nil`
- **AND** los datos quedan disponibles en `target`

#### Scenario: Fallo deja target no confiable
- **WHEN** una operación devuelve `ok = false`
- **THEN** el llamador no consume `target`, que puede contener datos parciales

### Requirement: Rango de éxito 200 a 226
El servicio SHALL considerar exitoso un estado HTTP en el rango 200–226, el
mismo que aplica `utils_oas/v2/request`. Un estado fuera de ese rango es un
rechazo del CRUD.

#### Scenario: Estado dentro del rango
- **WHEN** el CRUD responde un estado entre 200 y 226 con cuerpo compatible con `target`
- **THEN** el servicio devuelve `(true, statusRemoto, nil)`

#### Scenario: Estado fuera del rango
- **WHEN** el CRUD responde un estado mayor que 226 o menor que 200
- **THEN** el servicio lo trata como rechazo y devuelve `(false, statusRemoto, errPublico)`

### Requirement: Respuesta 204 sin contenido
El servicio SHALL devolver `(true, 204, nil)` ante un `204` sin cuerpo,
reconociendo específicamente el EOF que la librería envuelve como error de
decodificación al intentar decodificar un cuerpo vacío.

#### Scenario: Respuesta 204 sin cuerpo
- **WHEN** el CRUD responde `204` sin contenido
- **THEN** el servicio reconoce el EOF de decodificación como ausencia de datos y devuelve `(true, 204, nil)`
- **AND** el llamador usa el status para identificar que no hay datos en `target`

### Requirement: Rechazo HTTP del CRUD
El servicio SHALL devolver `(false, statusRemoto, errPublico)` cuando el CRUD
rechaza la petición con un estado fuera del rango 200–226. La librería devuelve
el error de estado antes de decodificar el cuerpo, por lo que `target` no se
puebla.

#### Scenario: Rechazo HTTP conserva el estado remoto
- **WHEN** el CRUD responde con un estado fuera de 200–226 (por ejemplo 400, 404 o 500)
- **THEN** el servicio devuelve `(false, statusRemoto, errPublico)` sin decodificar el cuerpo en `target`

### Requirement: Fallo sin respuesta HTTP
El servicio SHALL devolver `(false, 500, errPublico)` cuando no se obtiene una
respuesta HTTP: configuración inválida, fallo al construir la URL, error de
serialización del payload, o fallo de transporte. El middleware asigna 500
cuando la librería devuelve status 0.

#### Scenario: Configuración de host ausente o inválida
- **WHEN** `TercerosCrudHost` está vacío o no produce una URL analizable
- **THEN** no se realiza ninguna petición y el servicio devuelve `(false, 500, errPublico)`

#### Scenario: Fallo de transporte
- **WHEN** la conexión es rechazada o el transporte falla antes de recibir una respuesta
- **THEN** la librería devuelve status 0 y el servicio devuelve `(false, 500, errPublico)`

### Requirement: Respuesta no decodificable en el target
El servicio SHALL devolver `(false, statusRemoto, errPublico)` cuando la
respuesta tiene éxito HTTP pero no puede decodificarse en `target` (JSON
truncado, tipo incompatible o fallo de lectura). El error público SHALL indicar
que la respuesta no pudo procesarse. Una decodificación fallida puede dejar
datos parciales en `target`; el llamador SHALL descartarlos. Esto aplica
también a escrituras (`Post`/`Put`), donde la decodificación puede fallar tras
aplicar el cambio en el CRUD.

#### Scenario: JSON truncado o incompatible con el target
- **WHEN** el CRUD responde con éxito HTTP pero el cuerpo está truncado o su tipo no encaja en `target`
- **THEN** el servicio conserva el status recibido y devuelve `(false, statusRemoto, errPublico)`
- **AND** registra un log con la causa de decodificación

### Requirement: Precondición del target
El `target` SHALL ser un puntero no nil a un valor compatible con el JSON
esperado. Un `target` nil o que no sea puntero es una precondición incumplida:
la librería devuelve un error de decodificación que el servicio traduce a
`(false, statusRemoto, errPublico)`.

#### Scenario: Target nil o no puntero
- **WHEN** se invoca una operación con un `target` nil o que no es puntero y el CRUD responde con éxito
- **THEN** la decodificación falla y el servicio devuelve `(false, statusRemoto, errPublico)` registrando la causa

### Requirement: Error público neutro
Los errores devueltos por el servicio SHALL tener un mensaje útil y neutro
para el llamador (por ejemplo, "No fue posible conectar con el servicio de
terceros" o "No fue posible procesar la respuesta del servicio de terceros").
El host, el puerto y la URL interna SHALL quedar fuera de `err.Error()` y de
cualquier respuesta HTTP pública. Si el error original contiene la URL
completa, el servicio SHALL extraer la causa antes de exponerla.

#### Scenario: El mensaje público no revela detalles internos
- **WHEN** una operación falla por conexión rechazada, rechazo HTTP o decodificación
- **THEN** `err.Error()` no contiene host, puerto ni URL interna
- **AND** el mensaje describe la etapa del fallo en términos neutros

### Requirement: Registro de la causa técnica
El servicio SHALL registrar la causa técnica con `github.com/beego/beego/v2/core/logs`
(o el wrapper que use ese paquete), con una sola entrada por fallo. El registro
SHALL incluir operación, método, etapa, host y puerto cuando estén disponibles,
estado remoto y causa concreta, distinguiendo timeout, cancelación, conexión
rechazada, serialización, rechazo HTTP y decodificación. El registro SHALL
excluir credenciales, tokens, valores de filtros y payloads.

#### Scenario: Una entrada de log por fallo con la causa distinguida
- **WHEN** una operación falla en cualquier etapa (transporte, rechazo HTTP o decodificación)
- **THEN** se escribe exactamente una entrada de log con la etapa y la causa concreta
- **AND** la entrada no contiene credenciales, tokens, valores de filtros ni payloads
