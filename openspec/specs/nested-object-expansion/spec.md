# nested-object-expansion Specification

## Purpose

Permite que cualquier servicio del API resuelva y entregue entidades relacionadas que viven en otros CRUD como objetos anidados, mediante configuración declarativa, sin que el MID se convierta en dueño de los datos.

## Requirements

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
El sistema SHALL aceptar un parámetro `fields` con rutas separadas por comas (por
ejemplo `Id,Tercero.Id,Tercero.NombreCompleto`) y resolver cada ruta según su
último segmento: una ruta que termina en una relación entrega esa relación
completa (su subárbol expandido por configuración hasta la profundidad efectiva);
una ruta que termina en un campo escalar entrega solo ese escalar. Los segmentos
intermedios se materializan únicamente como contenedores (esqueleto) del camino
solicitado, sin sus demás escalares ni sus relaciones no solicitadas. El sistema
SHALL recortar de cada entidad los campos y relaciones no alcanzados por ninguna
ruta. Cuando varias rutas alcanzan la misma relación, la precedencia es **por
relación**: si alguna ruta la nombra como hoja, esa relación se entrega completa
y las demás rutas sobre ella quedan subsumidas; la precedencia de una relación no
afecta a sus relaciones hermanas, que conservan su propio tratamiento (completo o
parcial).

#### Scenario: Selección de campos base y anidados
- **WHEN** la solicitud incluye `fields=Id,Tercero.Id,Tercero.NombreCompleto`
- **THEN** la respuesta incluye el campo base `Id` y la relación `Tercero` con sus
  campos `Id` y `NombreCompleto`

#### Scenario: Proyección estricta de escalares seleccionados
- **WHEN** `fields` selecciona un subconjunto de campos escalares de una relación
  (por ejemplo `Tercero.NombreCompleto`)
- **THEN** la entidad anidada se entrega solo con los campos escalares
  seleccionados en ese nivel, sin los demás escalares (incluido el `Id` cuando no
  se seleccionó) ni relaciones no solicitadas, y sin aviso en `MetaData`

#### Scenario: Relación terminal se expande completa
- **WHEN** `fields` nombra una relación como último segmento de una ruta (por
  ejemplo `Tercero`)
- **THEN** la respuesta entrega esa relación con su subárbol expandido por
  configuración (sus escalares y sus relaciones hasta la profundidad efectiva),
  como si no hubiera selección por debajo de ella

#### Scenario: Precedencia de la selección más amplia por relación
- **WHEN** `fields` incluye una relación como hoja y además escalares o
  descendientes de esa misma relación (por ejemplo `Tercero` junto a `Tercero.Id`
  o `Tercero.NombreCompleto`)
- **THEN** la relación se entrega completa (su subárbol por configuración) y esos
  escalares o descendientes quedan subsumidos

#### Scenario: La precedencia no afecta a relaciones hermanas
- **WHEN** `fields` nombra una relación como hoja (completa) y otra relación
  hermana solo por un descendiente (por ejemplo `Tercero` y `Perfil.Nombre`)
- **THEN** la primera se entrega completa y la hermana se entrega como esqueleto
  con solo lo solicitado; una no subsume a la otra

#### Scenario: Dos ramas de la misma relación coexisten
- **WHEN** `fields` nombra dos descendientes de la misma relación (por ejemplo
  `Tercero.DatosIdentificacion` y `Tercero.Perfil`)
- **THEN** el padre se entrega como esqueleto conteniendo ambas ramas solicitadas

#### Scenario: Objeto padre intermedio como esqueleto
- **WHEN** `fields` solicita un descendiente a través de uno o más padres (por
  ejemplo `Tercero.DatosIdentificacion.Numero`)
- **THEN** los objetos padre del camino (`Tercero`, `DatosIdentificacion`) se
  materializan solo como contenedores del descendiente solicitado, sin sus demás
  escalares ni sus relaciones no solicitadas

#### Scenario: Relación hermana no solicitada se omite
- **WHEN** `fields` selecciona una rama de un padre (por ejemplo
  `Tercero.DatosIdentificacion`) y ese padre tiene otras relaciones no nombradas
- **THEN** esas relaciones hermanas no se consultan ni se incluyen en la respuesta

#### Scenario: Sin fields no se recorta
- **WHEN** la solicitud no incluye `fields` para un nivel
- **THEN** la entidad conserva los campos que entrega el resolver, sin recorte
  proyectivo

#### Scenario: Inclusión automática de padres
- **WHEN** `fields` solicita un descendiente como `Tercero.InfoComplementariaTercero`
- **THEN** la respuesta incluye el objeto padre `Tercero` como contenedor
  necesario para materializar el descendiente

#### Scenario: Ausencia de fields expande por configuración
- **WHEN** la solicitud no incluye `fields`
- **THEN** se recuperan los `allowed_fields` configurados y las relaciones
  permitidas hasta la profundidad efectiva

#### Scenario: Campo no permitido rechaza la expansión
- **WHEN** `fields` incluye una ruta fuera de cualquier `allowed_fields` o una
  ruta vacía o duplicada
- **THEN** el sistema responde `400 Bad Request`, no inicia la expansión e
  identifica la ruta inválida

### Requirement: Id implícito en las entidades expandidas
El sistema SHALL conservar el `Id` de toda entidad expandida de forma implícita,
sin exigir que la configuración lo declare en `allowed_fields`, de modo que toda
entidad anidada sea identificable y seleccionable por defecto.

#### Scenario: Id presente sin declararlo en allowed_fields
- **WHEN** una relación se expande y su `allowed_fields` no incluye `Id`
- **THEN** la entidad anidada se entrega con su `Id` de todas formas

#### Scenario: Id seleccionable aunque no esté en allowed_fields
- **WHEN** `fields` incluye la ruta del `Id` de una relación (por ejemplo
  `Tercero.Id`) y `allowed_fields` de esa relación no lista `Id`
- **THEN** la ruta se acepta como válida y el `Id` se incluye, sin rechazarla como
  campo no permitido

### Requirement: Omisión del Id por selección de campos
El sistema SHALL permitir que una selección explícita por `fields` omita el `Id`
de una relación: cuando ninguna ruta de `fields` para esa relación menciona el
`Id`, el `Id` no se incluye para esa solicitud, de forma silenciosa y local a la
solicitud. No existe sintaxis de exclusión.

#### Scenario: fields sin Id omite el Id de esa rama
- **WHEN** `fields` selecciona campos de una relación (por ejemplo
  `Tercero.NombreCompleto`) pero no incluye `Tercero.Id`
- **THEN** la entidad anidada se entrega sin su `Id`, sin aviso en `MetaData`

#### Scenario: Ausencia total de fields conserva el Id implícito
- **WHEN** la solicitud no incluye `fields`
- **THEN** la expansión por configuración conserva el `Id` implícito de cada
  entidad expandida

### Requirement: Lista negra global de campos de expansión
El sistema SHALL cargar al arranque una lista negra global de campos desde el
archivo único `config/global_blacklist.yaml`, cuyas entradas se expresan de forma
jerárquica `related_api -> related_table -> [field]` con la palabra reservada
`ANY` como comodín en cualquier nivel. La lista negra suprime cada campo listado
de la respuesta de expansión y gana sobre `allowed_fields` y sobre el `Id`
implícito.

#### Scenario: Campo vetado se suprime pese a allowed_fields
- **WHEN** un campo está declarado en `allowed_fields` de una relación y también
  figura en la lista negra para su `related_api` y `related_table`
- **THEN** el campo se omite de la entidad expandida

#### Scenario: Id implícito vetado se suprime
- **WHEN** el `Id` de una entidad está en la lista negra para su `related_api` y
  `related_table`
- **THEN** la entidad se expande sin su `Id`, de forma silenciosa y sin aviso en
  `MetaData`

#### Scenario: Comodín ANY por nivel
- **WHEN** una entrada usa `ANY` en el nivel de `related_api`, de `related_table`
  o de campo
- **THEN** la supresión aplica a cualquier valor de ese nivel que coincida con los
  demás niveles de la entrada

### Requirement: Aplicación de la lista negra sobre la salida de todo resolver
El sistema SHALL aplicar la lista negra como un recorte del motor de expansión
sobre la salida de todo resolver, específico o genérico por igual, de modo que
ningún resolver —incluido el genérico que inserta el registro sin transformar—
pueda filtrar un campo vetado.

#### Scenario: El resolver genérico obedece la lista negra
- **WHEN** una relación se resuelve con el resolver genérico y su registro
  contiene un campo vetado
- **THEN** el motor recorta el campo vetado antes de anidar el registro

#### Scenario: El resolver específico obedece la lista negra
- **WHEN** una relación se resuelve con un resolver específico y su registro
  contiene un campo vetado
- **THEN** el motor recorta el campo vetado antes de anidar el registro

### Requirement: Rechazo de un campo vetado solicitado por fields
El sistema SHALL tratar la solicitud explícita en `fields` de un campo vetado por
la lista negra con el mismo resultado que la solicitud de un campo no permitido,
reutilizando el rechazo genérico existente sin revelar que el campo existe ni que
está en la lista negra.

#### Scenario: Pedir un campo vetado en fields
- **WHEN** `fields` incluye la ruta de un campo que está en la lista negra
- **THEN** la solicitud se rechaza con el mismo resultado observable que una ruta
  no permitida, sin distinguir que el campo está vetado

### Requirement: Validación al arranque de la lista negra global
El sistema SHALL validar al arranque la lista negra global: una entrada con
componentes exactos (sin comodín) SHALL referenciar un `related_api`,
`related_table` o campo existente en el modelo conocido, y la entrada
`ANY/ANY/ANY` SHALL rechazarse. Un componente `ANY` no se verifica contra el
modelo.

#### Scenario: Componente exacto inexistente aborta el arranque
- **WHEN** una entrada nombra un `related_api`, `related_table` o campo exacto que
  no existe en el modelo conocido
- **THEN** el API no inicia y reporta un mensaje que identifica el archivo y la
  entrada inválida

#### Scenario: Entrada ANY/ANY/ANY prohibida
- **WHEN** una entrada usa `ANY` en los tres niveles a la vez
- **THEN** el API no inicia y reporta la entrada como prohibida

#### Scenario: Comodín no se verifica contra el modelo
- **WHEN** una entrada usa `ANY` en un nivel
- **THEN** ese nivel no se contrasta contra el modelo y no impide el arranque por
  inexistencia

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

### Requirement: Despacho a resolvers específicos por relación
El sistema SHALL permitir que cada relación declarada se resuelva mediante un resolver específico asociado por convención de nombre e inyectado por la capa de negocio, y SHALL usar el resolver genérico solo como respaldo cuando una relación no tiene resolver específico asociado.

#### Scenario: Relación con resolver específico asociado
- **WHEN** una relación declarada tiene un resolver específico asociado por su nombre
- **THEN** el expansor delega la consulta de esa relación a ese resolver específico y no al genérico

#### Scenario: Relación sin resolver específico cae al genérico
- **WHEN** una relación declarada no tiene resolver específico asociado
- **THEN** el expansor usa el resolver genérico y se agrega un aviso de tipo `generic_resolver`

#### Scenario: El mapeo se inyecta desde la capa de negocio
- **WHEN** la capa de negocio invoca la expansión
- **THEN** provee el mapeo de nombre de relación a resolver específico, y el expansor no conoce ese mapeo por sí mismo

#### Scenario: La configuración no nombra implementaciones de código
- **WHEN** se declara una relación en la configuración
- **THEN** la asociación con su resolver específico ocurre por convención de nombre en el código y la configuración no incluye nombres de implementaciones

### Requirement: Resolvers específicos como adaptadores de consulta
El sistema SHALL implementar los resolvers específicos como envoltorios finos sobre el cliente HTTP que fijan el endpoint y el campo de consulta real de cada entidad relacionada, reutilizando el transporte, los timeouts y la decodificación existentes, sin aplicar lógica de negocio.

#### Scenario: Consulta por el campo real de la entidad
- **WHEN** una entidad relacionada se consulta por un campo distinto de su propio identificador (por ejemplo por la llave del padre o por una llave primaria compartida)
- **THEN** su resolver específico fija ese campo de consulta y recupera los registros correspondientes

#### Scenario: Sin transformación de negocio en el resolver
- **WHEN** un resolver específico recupera los registros de su entidad
- **THEN** los anida tal como los entrega el CRUD, sin marcar, filtrar ni decidir ramas de negocio

#### Scenario: Reutilización del transporte existente
- **WHEN** un resolver específico ejecuta su consulta
- **THEN** reutiliza el cliente HTTP, los timeouts y la decodificación del envoltorio OAS o lista directa ya existentes, sin reimplementarlos
