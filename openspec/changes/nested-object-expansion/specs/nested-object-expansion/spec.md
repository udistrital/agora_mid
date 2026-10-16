# Spec Delta

## Purpose

Permite que cualquier servicio del API resuelva y entregue entidades relacionadas que viven en otros CRUD como objetos anidados, mediante configuración declarativa, sin que el MID se convierta en dueño de los datos.

## ADDED Requirements

### Requirement: Expansión declarativa de relaciones
El sistema SHALL expandir relaciones hacia otros CRUD únicamente a partir de configuración versionada por servicio, cargada al arranque, y nunca a partir de endpoints o URLs provistos por el cliente.

#### Scenario: Relación singular encontrada
- **WHEN** se expande una relación con `cardinality: one` y el CRUD devuelve un registro coincidente por `related_field`
- **THEN** el objeto relacionado se inserta bajo `mapped_name` conservando la llave local original

#### Scenario: Relación singular sin resultado
- **WHEN** se expande una relación `one` y el CRUD no devuelve coincidencia
- **THEN** `mapped_name` se entrega como `null` y no se inventan propiedades

#### Scenario: Relación múltiple con cero, uno o varios resultados
- **WHEN** se expande una relación con `cardinality: many`
- **THEN** `mapped_name` siempre es una lista; vacía `[]` si no hay resultados y con todos los elementos cuando los hay

#### Scenario: Configuración inválida impide el arranque
- **WHEN** un archivo de relaciones está ausente, es inválido o declara una relación inconsistente
- **THEN** el API no inicia y reporta un mensaje que identifica el archivo y la relación

### Requirement: Distinción entre API externa y servicio lógico
El sistema SHALL distinguir, en la configuración de cada relación, entre el campo `related_api` que identifica el servicio externo (la API que resuelve la URL base contra `conf/app.conf`, con valores `terceros`, `parametros` o `gestion_personas_crud`) y el campo `related_service` que nombra el servicio o recurso lógico relacionado del dominio y no resuelve URLs.

#### Scenario: related_api resuelve la URL base
- **WHEN** una relación declara `related_api: terceros`
- **THEN** la URL base de la consulta se resuelve a partir de la API `terceros` configurada en `conf/app.conf`, independientemente del valor de `related_service`

#### Scenario: related_api obligatorio y válido
- **WHEN** una relación omite `related_api`, lo deja vacío o declara una API que no existe en la configuración de URLs
- **THEN** el API no inicia y reporta un mensaje que identifica el archivo y la relación con el valor de `related_api` inválido

#### Scenario: related_service no resuelve URLs
- **WHEN** una relación declara un `related_service` como nombre lógico distinto de cualquier API
- **THEN** la validación solo exige que `related_service` no esté vacío y no lo compara contra las URLs externas

### Requirement: Selección de campos por rutas
El sistema SHALL aceptar un parámetro `fields` con rutas separadas por comas (por ejemplo `Id,Tercero.Id,Tercero.NombreCompleto`) y devolver solo esas rutas más los objetos padre necesarios para materializarlas.

#### Scenario: Selección de campos base y anidados
- **WHEN** la solicitud incluye `fields=Id,Tercero.Id,Tercero.NombreCompleto`
- **THEN** la respuesta incluye el campo base `Id` y la relación `Tercero` con sus campos `Id` y `NombreCompleto`

#### Scenario: Inclusión automática de padres
- **WHEN** `fields` solicita un descendiente como `Tercero.InfoComplementariaTercero`
- **THEN** la respuesta incluye el objeto padre `Tercero` necesario para materializar el descendiente

#### Scenario: Ausencia de fields expande por configuración
- **WHEN** la solicitud no incluye `fields`
- **THEN** se recuperan los `allowed_fields` configurados y las relaciones permitidas hasta la profundidad efectiva

#### Scenario: Campo no permitido rechaza la expansión
- **WHEN** `fields` incluye una ruta fuera de cualquier `allowed_fields` o una ruta vacía o duplicada
- **THEN** el sistema responde `400 Bad Request`, no inicia la expansión e identifica la ruta inválida

### Requirement: Límites, ciclos y profundidad efectiva
El sistema SHALL aplicar límites de profundidad e IDs por relación, detectar ciclos y calcular la profundidad efectiva de cada rama como el mínimo entre el límite global restante y la profundidad declarada por la relación.

#### Scenario: Expansión recursiva hasta max_depth
- **WHEN** una rama alcanza el límite de profundidad global configurado
- **THEN** la rama no se expande por encima de ese límite

#### Scenario: Configuración hija con menor profundidad
- **WHEN** una configuración hija declara una profundidad menor que el límite restante
- **THEN** la rama continúa con el valor menor y se agrega una notificación de tipo `depth` a los avisos con su `path`

#### Scenario: Ciclo detectado
- **WHEN** el recorrido detecta un ciclo de relaciones (por ejemplo `Persona -> Tercero -> Persona`)
- **THEN** la rama se detiene, no es un error HTTP y se registra un aviso de tipo `cycle`

#### Scenario: Deduplicación y caché por solicitud
- **WHEN** varios registros comparten la misma llave en una misma solicitud
- **THEN** las llaves se deduplican, las consultas se agrupan y los resultados se reutilizan dentro de esa solicitud

### Requirement: Reporte tolerante de errores de relación
El sistema SHALL honrar la política `on_relation_error` configurada: en modo `notify` conservar el recurso principal y reportar avisos; en modo `fail` abortar toda la expansión.

#### Scenario: Error de relación tolerado (notify)
- **WHEN** un CRUD relacionado no está disponible y `on_relation_error=notify`
- **THEN** el recurso principal se conserva, la rama fallida usa `null` o `[]` y se agrega un aviso no sensible de tipo `relation`

#### Scenario: Error de relación estricto (fail)
- **WHEN** un CRUD relacionado no está disponible y `on_relation_error=fail`
- **THEN** toda la expansión se aborta con un error de puerta de enlace

#### Scenario: Resolver genérico sin adaptador específico
- **WHEN** una relación declarada y validada no tiene resolver especializado
- **THEN** el resolver genérico inserta su `Data` sin transformar y se agrega un aviso de tipo `generic_resolver`

#### Scenario: Mensajes de error no exponen datos sensibles
- **WHEN** se reporta cualquier aviso o error de relación
- **THEN** el mensaje identifica la ruta lógica de la relación pero no incluye URLs internas, credenciales ni cuerpos remotos
