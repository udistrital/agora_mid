# Spec Delta

## MODIFIED Requirements

### Requirement: El contrato publicado describe los endpoints de escritura del proveedor
El sistema SHALL describir en el contrato OpenAPI publicado y en `API.md` los endpoints de escritura `POST` y `PATCH /v1/proveedor`, incluyendo el cuerpo anidado y las respuestas `200`, `206` con `MetaData.Errors`, `400` y `502/504`, de modo que la documentación pública sea coherente con el comportamiento implementado, incluyendo el tipo de las llaves, la forma real del `Data` materializado y la reutilización de sus campos relacionales derivados.

#### Scenario: El contrato incluye los endpoints de escritura
- **WHEN** se consulta el contrato OpenAPI publicado o `API.md`
- **THEN** ambos describen `POST` y `PATCH /v1/proveedor` con su cuerpo anidado y los códigos de respuesta `200`, `206`, `400` y `502/504`

#### Scenario: El contrato no describe endpoints retirados
- **WHEN** se consulta el contrato publicado tras retirar los flujos estáticos
- **THEN** no describe las rutas `/v1/persona_*` eliminadas

#### Scenario: El contrato describe las llaves materializadas como números
- **WHEN** se consulta el esquema del `Data` materializado de escritura en el contrato publicado
- **THEN** las llaves foráneas (`TerceroId`, `ProveedorId`) y las PK compartidas se describen con tipo entero, coherentes con el comportamiento implementado

#### Scenario: El contrato describe el Data materializado de forma fiel
- **WHEN** se consulta el esquema de respuesta de `POST`/`PATCH /v1/proveedor`
- **THEN** el esquema del `Data` refleja el subárbol materializado real que el API devuelve, incluyendo las entidades anidadas y el campo de auditoría `AutorModificacionId`, y no un subconjunto reducido de identificadores

#### Scenario: El contrato describe la normalización de campos derivados
- **WHEN** se consulta la documentación del endpoint de escritura
- **THEN** se documenta que una entidad nueva puede canonizar una llave relacional y responder `206` con un aviso en `MetaData.Errors`

#### Scenario: El contrato describe el tipo de aviso
- **WHEN** se consulta el esquema de errores de `MetaData`
- **THEN** el contrato distingue el aviso de normalización de un fallo remoto de relación mediante `Kind`

#### Scenario: El contrato describe el reenvío del Data
- **WHEN** se consulta el ejemplo o esquema de `Data` materializado
- **THEN** se muestra que las llaves relacionales e identificadores materializados pueden aparecer en un cuerpo reenviado de `PATCH`
