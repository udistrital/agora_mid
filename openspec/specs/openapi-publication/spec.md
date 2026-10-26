# openapi-publication Specification

## Purpose

Publica el contrato OpenAPI del API en los formatos que consumen las herramientas, derivando el JSON desde el YAML fuente de forma determinística para que ambos describan siempre el mismo contrato.

## Requirements

### Requirement: Derivación determinística de JSON desde el YAML fuente
El sistema SHALL derivar el documento OpenAPI en formato JSON a partir del documento YAML fuente mediante una reserialización 1:1, de modo que el YAML sea la única fuente de verdad y el JSON represente exactamente el mismo modelo de datos.

#### Scenario: El JSON representa el mismo contrato que el YAML
- **WHEN** el sistema deriva el documento JSON a partir del YAML fuente
- **THEN** el JSON resultante es un documento válido cuyo modelo de datos es equivalente al del YAML, sin agregar, omitir ni reinterpretar campos

#### Scenario: Salida determinística
- **WHEN** el sistema deriva el documento JSON dos veces a partir del mismo YAML sin cambios
- **THEN** ambas derivaciones producen exactamente el mismo contenido

### Requirement: Regeneración idempotente en cada arranque
El sistema SHALL regenerar el documento JSON derivado al iniciar el API y SHALL actualizar el archivo publicado solo cuando su contenido difiera del derivado del YAML vigente.

#### Scenario: El YAML cambió desde el último arranque
- **WHEN** el API inicia y el YAML fuente produce un JSON distinto del archivo JSON publicado
- **THEN** el sistema actualiza el archivo JSON publicado para reflejar el YAML vigente

#### Scenario: El YAML no cambió desde el último arranque
- **WHEN** el API inicia y el YAML fuente produce un JSON idéntico al archivo JSON publicado
- **THEN** el sistema deja el archivo JSON publicado sin modificar

### Requirement: Fallo explícito ante un contrato fuente ausente o inválido
El sistema SHALL abortar el arranque del API con un error identificable cuando el YAML fuente no exista o no sea un documento válido, en lugar de publicar un contrato ausente o corrupto.

#### Scenario: YAML fuente ausente
- **WHEN** el API inicia y el YAML fuente no existe
- **THEN** el API no inicia y reporta un error que identifica el archivo fuente

#### Scenario: YAML fuente inválido
- **WHEN** el API inicia y el YAML fuente no es un documento válido
- **THEN** el API no inicia y reporta un error que identifica el archivo fuente como causa

### Requirement: Publicación coherente del contrato en ambos formatos
El sistema SHALL publicar el contrato OpenAPI como recurso accesible por HTTP en formato YAML y en formato JSON, de modo que una herramienta que solicite el JSON con su nombre por defecto obtenga el mismo contrato descrito por el YAML.

#### Scenario: El contrato está disponible en ambos formatos
- **WHEN** un cliente solicita el documento del contrato en formato YAML o en formato JSON
- **THEN** el sistema entrega, en cada formato, el mismo contrato OpenAPI vigente

#### Scenario: La interfaz de documentación encuentra el JSON por defecto
- **WHEN** una interfaz de documentación de OpenAPI solicita el documento JSON por su nombre por defecto
- **THEN** el documento existe y describe el contrato vigente, de modo que la interfaz carga sin faltar el recurso

### Requirement: El contrato publicado describe los endpoints de escritura del proveedor
El sistema SHALL describir en el contrato OpenAPI publicado y en `API.md` los
endpoints de escritura `POST` y `PATCH /v1/proveedor`, incluyendo el cuerpo
anidado y las respuestas `200`, `206` con `MetaData.Errors`, `400` y `502/504`,
de modo que la documentación pública sea coherente con el comportamiento
implementado, incluyendo el tipo de las llaves, la forma real del `Data`
materializado y la reutilización de sus campos relacionales derivados.

#### Scenario: El contrato incluye los endpoints de escritura
- **WHEN** se consulta el contrato OpenAPI publicado o `API.md`
- **THEN** ambos describen `POST` y `PATCH /v1/proveedor` con su cuerpo anidado y
  los códigos de respuesta `200`, `206`, `400` y `502/504`

#### Scenario: El contrato no describe endpoints retirados
- **WHEN** se consulta el contrato publicado tras retirar los flujos estáticos
- **THEN** no describe las rutas `/v1/persona_*` eliminadas

#### Scenario: El contrato describe las llaves materializadas como números
- **WHEN** se consulta el esquema del `Data` materializado de escritura en el
  contrato publicado
- **THEN** las llaves foráneas (`TerceroId`, `ProveedorId`) y las PK compartidas
  se describen con tipo entero, coherentes con el comportamiento implementado

#### Scenario: El contrato describe el Data materializado de forma fiel
- **WHEN** se consulta el esquema de respuesta de `POST`/`PATCH /v1/proveedor`
- **THEN** el esquema del `Data` refleja el subárbol materializado real que el API
  devuelve, incluyendo las entidades anidadas y el campo de auditoría
  `AutorModificacionId`, y no un subconjunto reducido de identificadores

#### Scenario: El contrato describe la normalización de campos derivados
- **WHEN** se consulta la documentación del endpoint de escritura
- **THEN** se documenta que una entidad nueva puede canonizar una llave relacional
  y responder `206` con un aviso en `MetaData.Errors`

#### Scenario: El contrato describe el tipo de aviso
- **WHEN** se consulta el esquema de errores de `MetaData`
- **THEN** el contrato distingue el aviso de normalización de un fallo remoto de
  relación mediante `Kind`

#### Scenario: El contrato describe el reenvío del Data
- **WHEN** se consulta el ejemplo o esquema de `Data` materializado
- **THEN** se muestra que las llaves relacionales e identificadores materializados
  pueden aparecer en un cuerpo reenviado de `PATCH`

### Requirement: La interfaz de documentación carga el contrato local
El sistema SHALL servir la interfaz de documentación en `/swagger` cargando el
contrato propio derivado (`./swagger.json`), y nunca un contrato remoto de
demostración, de modo que la interfaz muestre el contrato vigente del API.

#### Scenario: La interfaz muestra el contrato del API
- **WHEN** un usuario abre la interfaz de documentación en `/swagger`
- **THEN** la interfaz carga el contrato local derivado del YAML fuente y muestra
  los endpoints del API, no un contrato de ejemplo externo
