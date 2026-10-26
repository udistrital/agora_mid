# Proposal

## Why

El `Data` devuelto por una escritura contiene llaves relacionales que el MID
calcula durante la materialización. Si el cliente reenvía ese resultado completo,
la validación actual rechaza esas llaves por no estar en `writable_fields`, aunque
sean valores producidos por el propio MID. Esto obliga al cliente a limpiar una
respuesta antes de reutilizarla y hace fallar reintentos idempotentes.

## What Changes

- Reconocer los campos derivados de la configuración de relaciones como campos
  internos de escritura, sin agregarlos a `writable_fields`.
- Aceptar un campo relacional reenviado cuando está ausente, es `null` o coincide
  con la llave autoritativa calculada por el motor.
- Tratar `null` como ausencia y completar el campo con la llave autoritativa.
- Para una entidad nueva, canonizar un campo relacional desactualizado con la
  llave autoritativa y devolver un aviso en `MetaData.Errors`.
- Para una entidad existente con un campo relacional incompatible, no reasignar
  silenciosamente la entidad a otro padre; rechazar el cuerpo cuando el conflicto
  no sea compatible con la relación actual.
- Conservar los campos relacionales canonizados en el payload enviado al CRUD
  dueño y en el `Data` devuelto, preservando su tipo numérico.
- Mantener la validación estructural completa antes de las llamadas externas y
  distinguir sus errores `400` de los avisos de normalización y de los fallos
  remotos tolerados `206`.
- No añadir un `GET` previo genérico: los conflictos de entidades existentes se
  resolverán de forma conservadora sin introducir una llamada adicional por
  entidad.
- Añadir pruebas unitarias, de integración y e2e para reenvío de `Data`,
  coincidencia, `null`, canonización, conflictos de entidades existentes y
  avisos.
- Actualizar la documentación del contrato de respuesta, la API publicada y la
  documentación de pruebas.

## Capabilities

### New Capabilities

<!-- No se introduce una capacidad independiente; el comportamiento pertenece
     al motor de escritura y al contrato de respuestas existentes. -->

### Modified Capabilities

- `declarative-write-engine`: reconocer, validar y canonizar campos derivados de
  relaciones durante la escritura.
- `api-response-contract`: distinguir avisos de normalización de campos
  derivados dentro de `MetaData.Errors` y sus respuestas `206`.
- `proveedor-write`: permitir reenviar el subárbol materializado completo sin
  eliminar llaves relacionales generadas por el MID.
- `test-documentation`: documentar los nuevos casos de reenvío, canonización y
  conflicto.
- `openapi-publication`: publicar los tipos de aviso y la forma contractual del
  subárbol de escritura actualizado.

## Impact

- `helper/write_engine.go` y sus pruebas: validación en dos fases, propagación y
  construcción de payloads internos.
- `responses` y el controlador de proveedor: nuevo tipo de aviso y respuesta
  parcial cuando una entidad nueva se canoniza.
- `services/write_resolver.go`: conservar y materializar los campos relacionales
  generados, sin introducir una consulta GET previa genérica.
- `test/integration_test.go`, `test/e2e_test.go`, fixtures y `test/tests.md`.
- `API.md` y el contrato OpenAPI publicado.
- No cambia la modularización de YAML ni el camino de expansión de lectura.
