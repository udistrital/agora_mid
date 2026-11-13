# Spec Delta

## MODIFIED Requirements

### Requirement: Expansión declarativa de relaciones
El sistema SHALL expandir relaciones hacia otros CRUD únicamente a partir del árbol
efectivo compuesto por servicio (ver `config-composition`), cargado y aplanado al
arranque, y nunca a partir de endpoints o URLs provistos por el cliente. El expansor
consume el árbol ya compuesto y no resuelve referencias de objetos por sí mismo.

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
- **WHEN** un archivo de configuración está ausente, es inválido o declara una relación inconsistente
- **THEN** el API no inicia y reporta un mensaje que identifica el archivo y la relación

### Requirement: Selección de campos por rutas
El sistema SHALL aceptar un parámetro `fields` con rutas separadas por comas (por
ejemplo `Id,Tercero.Id,Tercero.NombreCompleto`) y resolver cada ruta según su
último segmento: una ruta que termina en una relación entrega esa relación
completa (su subárbol expandido por configuración hasta la profundidad efectiva);
una ruta que termina en un campo escalar entrega solo ese escalar. Los campos
seleccionables de cada objeto son los que la tabla de campos tipada unificada
(ver `config-composition`) marca con el rol `readable`.

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
- **THEN** se recuperan los campos marcados `readable` en la tabla de campos tipada
  del objeto y las relaciones permitidas hasta la profundidad efectiva

#### Scenario: Campo no permitido rechaza la expansión
- **WHEN** `fields` incluye una ruta fuera de cualquier campo marcado `readable` o una
  ruta vacía o duplicada
- **THEN** el sistema responde `400 Bad Request`, no inicia la expansión e
  identifica la ruta inválida

### Requirement: Id implícito en las entidades expandidas
El sistema SHALL conservar el `Id` de toda entidad expandida de forma implícita,
sin exigir que la tabla de campos tipada del objeto marque `Id` como `readable`, de
modo que toda entidad anidada sea identificable y seleccionable por defecto.

#### Scenario: Id presente sin declararlo en allowed_fields
- **WHEN** una relación se expande y la tabla de campos de su objeto no marca `Id`
  como `readable`
- **THEN** la entidad anidada se entrega con su `Id` de todas formas

#### Scenario: Id seleccionable aunque no esté en allowed_fields
- **WHEN** `fields` incluye la ruta del `Id` de una relación (por ejemplo
  `Tercero.Id`) y la tabla de campos de ese objeto no marca `Id` como `readable`
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
de la respuesta de expansión y gana sobre el rol `readable` de la tabla de campos
tipada y sobre el `Id` implícito.

#### Scenario: Campo vetado se suprime pese a allowed_fields
- **WHEN** un campo está marcado `readable` en la tabla de campos de un objeto y
  también figura en la lista negra para su `related_api` y `related_table`
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
