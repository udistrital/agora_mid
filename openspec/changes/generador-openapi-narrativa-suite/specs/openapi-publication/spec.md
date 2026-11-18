# Spec Delta

## ADDED Requirements

### Requirement: Derivación determinística de swagger.yml desde la configuración
El sistema SHALL generar el documento `swagger.yml` a partir de la configuración
(los objetos direccionables y su tabla de campos tipada, y el árbol efectivo por
servicio) de forma determinística, de modo que la configuración sea la fuente de
verdad del contrato OpenAPI y `swagger.yml` sea un artefacto derivado y no autorado
a mano.

#### Scenario: swagger.yml refleja la configuración vigente
- **WHEN** el sistema genera `swagger.yml` a partir de la configuración
- **THEN** el documento describe los objetos, campos y endpoints de la
  configuración vigente, sin agregar ni omitir contenido autorado aparte

#### Scenario: Salida byte-estable para una configuración sin cambios
- **WHEN** el sistema genera `swagger.yml` dos veces a partir de la misma
  configuración sin cambios
- **THEN** ambas generaciones producen exactamente el mismo contenido, con orden
  estable de claves y de colecciones

### Requirement: Regeneración de swagger.yml en cada arranque
El sistema SHALL regenerar `swagger.yml` desde la configuración al iniciar el API y
SHALL actualizar el archivo publicado solo cuando su contenido difiera del derivado
de la configuración vigente, de forma análoga a la regeneración del JSON.

#### Scenario: La configuración cambió desde el último arranque
- **WHEN** el API inicia y la configuración produce un `swagger.yml` distinto del
  archivo publicado
- **THEN** el sistema actualiza `swagger.yml` para reflejar la configuración
  vigente, y a continuación la derivación existente actualiza `swagger.json`

#### Scenario: La configuración no cambió desde el último arranque
- **WHEN** el API inicia y la configuración produce un `swagger.yml` idéntico al
  archivo publicado
- **THEN** el sistema deja `swagger.yml` sin modificar y, por tanto, tampoco
  reescribe `swagger.json`

### Requirement: Mapeo de tipo de almacenamiento a tipo OpenAPI
El sistema SHALL mapear cada tipo del vocabulario cerrado de almacenamiento de la
tabla de campos tipada (`boolean`, `integer`, `jsonb`, `numeric`, `string`,
`timestamp`, `uuid`) a un tipo y formato OpenAPI determinados, de modo que el
esquema publicado describa cada campo con el tipo que corresponde a su
almacenamiento.

#### Scenario: Cada tipo de almacenamiento tiene un tipo OpenAPI determinado
- **WHEN** el generador encuentra un campo con un tipo del vocabulario cerrado
- **THEN** el esquema OpenAPI del campo usa el tipo y formato fijados para ese tipo
  de almacenamiento, de forma consistente en todo el documento

#### Scenario: Las llaves numéricas se describen como enteros
- **WHEN** el generador describe un campo de tipo `integer` que representa una llave
  foránea o una PK compartida
- **THEN** el esquema del campo usa el tipo entero de OpenAPI, coherente con las
  llaves materializadas numéricas del contrato de escritura

## MODIFIED Requirements

### Requirement: Fallo explícito ante un contrato fuente ausente o inválido
El sistema SHALL abortar el arranque del API con un error identificable cuando la
fuente del contrato no permita generar un documento válido, en lugar de publicar un
contrato ausente o corrupto. La fuente del contrato es la configuración: el aborto
ante una configuración ausente o inválida lo provee la composición de la
configuración al arranque, y una configuración válida garantiza que la generación
de `swagger.yml` produzca un documento válido.

#### Scenario: YAML fuente ausente
- **WHEN** el API inicia y la configuración fuente del contrato está ausente, de
  modo que no puede generarse ningún `swagger.yml`
- **THEN** el API no inicia y reporta un error que identifica la configuración
  fuente ausente, sin publicar un contrato derivado

#### Scenario: YAML fuente inválido
- **WHEN** el API inicia y la configuración fuente del contrato es inválida, de modo
  que no puede generarse un `swagger.yml` válido
- **THEN** el API no inicia y reporta un error que identifica la configuración
  fuente inválida como causa

#### Scenario: Configuración inválida aborta el arranque antes de publicar
- **WHEN** el API inicia y la configuración fuente no es válida
- **THEN** el API no inicia y reporta un error que identifica la configuración como
  causa, sin publicar un contrato derivado de una fuente inválida

#### Scenario: La derivación a JSON falla ante un YAML generado inválido
- **WHEN** el API inicia y el `swagger.yml` generado no es un documento válido para
  la derivación a JSON
- **THEN** el API no inicia y reporta un error que identifica el archivo fuente de
  la derivación como causa
