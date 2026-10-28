# Proposal

## Why

El contrato OpenAPI del API vive en `swagger/swagger.yml`, pero Swagger UI busca
por defecto `swagger.json`; sin ese archivo la interfaz falla al cargar. Mantener
a mano un `swagger.json` paralelo introduciría una segunda fuente de verdad que
se desincroniza. Se necesita que `swagger.json` exista siempre y derive del YAML
de forma determinística, con el YAML como única fuente.

## What Changes

- Nuevo generador que, al arranque del API, deriva `swagger/swagger.json` desde
  `swagger/swagger.yml` mediante una reserialización 1:1 (Swagger 2.0 comparte
  modelo de datos entre YAML y JSON).
- La generación es determinística (la misma entrada produce los mismos bytes) e
  idempotente en disco: solo reescribe `swagger.json` cuando su contenido
  cambió respecto al derivado del YAML.
- El arranque falla de forma explícita (no inicia el API) si el YAML no existe o
  no es un documento válido, en lugar de publicar un contrato ausente o
  corrupto.
- La definición queda publicada como recurso estático bajo `/swagger`, de modo
  que `GET /swagger/swagger.yml` y `GET /swagger/swagger.json` entregan el mismo
  contrato y Swagger UI lo encuentra con su nombre por defecto.

## Capabilities

### New Capabilities
- `openapi-publication`: publicación del contrato OpenAPI del API, incluyendo la
  derivación determinística de `swagger.json` a partir del `swagger.yml` fuente y
  la exposición coherente de ambos formatos en una ruta estática.

### Modified Capabilities
<!-- Ninguna: las capabilities de comportamiento del dominio (proveedor-read,
     nested-object-expansion, api-response-contract) no cambian sus requisitos. -->

## Impact

- **Código**: un generador en el paquete `main` (`swagger_gen.go`) invocado desde
  `main.go` antes de registrar la ruta estática; publicación con
  `beego.SetStaticPath("/swagger", "swagger")`.
- **Artefactos servidos**: `swagger/swagger.yml` (fuente, versionada) y
  `swagger/swagger.json` (derivado en cada arranque).
- **Dependencias**: reutiliza `gopkg.in/yaml.v3` (ya presente) y `encoding/json`
  de la biblioteca estándar; no añade dependencias nuevas.
- **Despliegue**: la imagen o entorno de ejecución debe incluir el directorio
  `swagger/` junto al binario para que la ruta estática resuelva.
- **Rutas afectadas**: alta de `/swagger` (estático). No altera los endpoints de
  dominio existentes.
