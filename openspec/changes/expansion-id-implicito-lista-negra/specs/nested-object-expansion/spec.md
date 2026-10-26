# Spec Delta

## MODIFIED Requirements

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

## ADDED Requirements

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
