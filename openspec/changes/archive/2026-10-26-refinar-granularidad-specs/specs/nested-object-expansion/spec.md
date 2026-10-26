# Spec Delta

## MODIFIED Requirements

### Requirement: Selección de campos por rutas
El sistema SHALL aceptar un parámetro `fields` con rutas separadas por comas (por
ejemplo `Id,Tercero.Id,Tercero.NombreCompleto`) y resolver cada ruta según su
último segmento: una ruta que termina en una relación entrega esa relación
completa (su subárbol expandido por configuración hasta la profundidad efectiva);
una ruta que termina en un campo escalar entrega solo ese escalar.

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

### Requirement: Materialización de segmentos intermedios como esqueleto
El sistema SHALL materializar los segmentos intermedios de una ruta de `fields`
únicamente como contenedores (esqueleto) del camino solicitado, sin sus demás
escalares ni sus relaciones no solicitadas, incluyendo automáticamente los padres
necesarios para alcanzar el descendiente pedido.

#### Scenario: El camino intermedio se materializa sin sus demás campos
- **WHEN** `fields` alcanza un descendiente a través de uno o más segmentos
  intermedios que no se nombran como hoja
- **THEN** cada segmento intermedio se entrega solo como contenedor del camino,
  sin sus otros escalares ni sus relaciones no solicitadas

### Requirement: Recorte de campos y relaciones no alcanzados
El sistema SHALL recortar de cada entidad los campos y las relaciones que ninguna
ruta de `fields` alcanza, dejando en la respuesta únicamente lo que las rutas
seleccionaron.

#### Scenario: Lo no alcanzado por una ruta se recorta
- **WHEN** una entidad tiene campos o relaciones que ninguna ruta de `fields`
  nombra ni atraviesa
- **THEN** esos campos y relaciones se recortan de la respuesta y no se consultan

### Requirement: Precedencia por relación en la selección de campos
El sistema SHALL resolver la coincidencia de varias rutas sobre una misma
relación con precedencia **por relación**: si alguna ruta la nombra como hoja, esa
relación se entrega completa y las demás rutas sobre ella quedan subsumidas, sin
que esa precedencia afecte el tratamiento de sus relaciones hermanas.

#### Scenario: La ruta hoja subsume a las descendientes de su relación
- **WHEN** `fields` nombra una relación como hoja y, a la vez, escalares o
  descendientes de esa misma relación
- **THEN** la relación se entrega completa y esas rutas más específicas quedan
  subsumidas, mientras las relaciones hermanas conservan su propio tratamiento
