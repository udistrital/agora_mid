# Spec Delta

## ADDED Requirements

### Requirement: El contrato publicado describe los endpoints de escritura del proveedor
El sistema SHALL describir en el contrato OpenAPI publicado y en `API.md` los
endpoints de escritura `POST` y `PATCH /v1/proveedor`, incluyendo el cuerpo
anidado y las respuestas `200`, `206` con `MetaData.Errors`, `400` y `502/504`,
de modo que la documentación pública sea coherente con el comportamiento
implementado.

#### Scenario: El contrato incluye los endpoints de escritura
- **WHEN** se consulta el contrato OpenAPI publicado o `API.md`
- **THEN** ambos describen `POST` y `PATCH /v1/proveedor` con su cuerpo anidado y
  los códigos de respuesta `200`, `206`, `400` y `502/504`

#### Scenario: El contrato no describe endpoints retirados
- **WHEN** se consulta el contrato publicado tras retirar los flujos estáticos
- **THEN** no describe las rutas `/v1/persona_*` eliminadas

### Requirement: La interfaz de documentación carga el contrato local
El sistema SHALL servir la interfaz de documentación en `/swagger` cargando el
contrato propio derivado (`./swagger.json`), y nunca un contrato remoto de
demostración, de modo que la interfaz muestre el contrato vigente del API.

#### Scenario: La interfaz muestra el contrato del API
- **WHEN** un usuario abre la interfaz de documentación en `/swagger`
- **THEN** la interfaz carga el contrato local derivado del YAML fuente y muestra
  los endpoints del API, no un contrato de ejemplo externo
