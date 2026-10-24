# Spec Delta

## MODIFIED Requirements

### Requirement: Envoltorio de respuesta OAS plano
El sistema SHALL responder con un envoltorio OAS plano que incluya `Success` (booleano), `Status` (cadena con el código HTTP), `Message` (cadena) y `Data` (carga útil), en todos los verbos del API.

#### Scenario: Respuesta exitosa de lectura
- **WHEN** una consulta de lectura se resuelve correctamente
- **THEN** la respuesta incluye `Success: true`, `Status: "200"`, un `Message` y el recurso en `Data`

#### Scenario: Respuesta de escritura
- **WHEN** una operación de escritura (`POST` o `PATCH`) se completa sin fallos parciales
- **THEN** la respuesta usa el mismo envoltorio plano `Success/Status/Message/Data` sin el bloque `MetaData`

#### Scenario: Respuesta de escritura con materialización parcial
- **WHEN** una operación de escritura tolerante materializa solo parte del subárbol por un fallo de relación no bloqueante
- **THEN** la respuesta usa `Status: "206"`, incluye `MetaData.Errors` con los avisos y `Data` con los identificadores ya materializados

#### Scenario: Compatibilidad del envoltorio con el ecosistema
- **WHEN** el controlador de errores estándar y los controladores de negocio emiten respuestas
- **THEN** ambos usan la misma raíz plana `Success/Status/Message/Data` con `Status` como cadena
