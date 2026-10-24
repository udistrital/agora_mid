# Spec Delta

## MODIFIED Requirements

### Requirement: El contrato publicado describe los endpoints de escritura del proveedor
El sistema SHALL describir en el contrato OpenAPI publicado y en `API.md` los
endpoints de escritura `POST` y `PATCH /v1/proveedor`, incluyendo el cuerpo
anidado y las respuestas `200`, `206` con `MetaData.Errors`, `400` y `502/504`,
de modo que la documentación pública sea coherente con el comportamiento
implementado, incluyendo el tipo de las llaves y la forma real del `Data`
materializado.

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
