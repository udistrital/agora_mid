# Spec Delta

## Purpose

Publica el contrato OpenAPI del API en los formatos que consumen las herramientas, derivando el JSON desde el YAML fuente de forma determinística para que ambos describan siempre el mismo contrato.

## ADDED Requirements

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
