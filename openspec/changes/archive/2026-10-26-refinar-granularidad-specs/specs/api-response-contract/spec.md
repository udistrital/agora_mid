# Spec Delta

## MODIFIED Requirements

### Requirement: Extensión MetaData para avisos y total de listas
El sistema SHALL incluir un bloque `MetaData` opcional cuyo campo opcional
`Length` lleva el total de valores posibles de un `Data` de tipo lista. Cada campo
del bloque es opcional y se incluye únicamente cuando aplica; `MetaData` se omite
cuando no aporta ningún dato.

#### Scenario: Expansión parcial con relación tolerada
- **WHEN** una expansión en modo `notify` sufre un error de relación tolerado
- **THEN** la respuesta usa `Status: "206"` e incluye `MetaData.Errors` con una entrada que lleva `Path`, `Service`, `Message` y `Kind`

#### Scenario: Errors es opcional
- **WHEN** una respuesta incluye `MetaData` por otro motivo (por ejemplo `Length`) pero no genera avisos de expansión
- **THEN** el campo `Errors` se omite del bloque `MetaData`

#### Scenario: Total de lista en Length
- **WHEN** una consulta de lectura devuelve un `Data` de tipo lista
- **THEN** `MetaData.Length` refleja el número total de valores posibles sin considerar `limit` ni `offset`, y no el tamaño de la página devuelta

#### Scenario: Length es opcional
- **WHEN** `Data` no es una lista o no aplica un total
- **THEN** el campo `Length` se omite del bloque `MetaData`

#### Scenario: Respuesta sin MetaData aplicable lo omite
- **WHEN** una respuesta no genera avisos de expansión ni reporta un total de lista
- **THEN** el bloque `MetaData` se omite por completo de forma consistente

#### Scenario: Distinción de tipos de aviso
- **WHEN** el sistema emite cualquier aviso en `MetaData.Errors`
- **THEN** su `Kind` corresponde a una constante de tipo de aviso declarada en el código, y no existe ningún `Kind` emitido sin una constante que lo respalde

#### Scenario: Normalización de campo derivado
- **WHEN** el motor reemplaza un campo relacional desactualizado por la llave autoritativa
- **THEN** `MetaData.Errors` incluye una entrada con `Kind: field_normalization` y la respuesta usa `206`

#### Scenario: Campo derivado coincidente
- **WHEN** el valor reenviado coincide con la llave autoritativa
- **THEN** la respuesta usa `200` y omite `MetaData.Errors`

#### Scenario: Distinción entre normalización y fallo remoto
- **WHEN** una respuesta contiene avisos por normalización y por fallo remoto tolerado
- **THEN** cada entrada identifica su origen mediante un `Kind` distinto, sin exponer URLs, credenciales ni cuerpos remotos

#### Scenario: Todo tipo de aviso emitido tiene una constante declarada
- **WHEN** el sistema emite un aviso con un `Kind`
- **THEN** ese `Kind` corresponde a una constante de tipo de aviso declarada en el código y no existe ningún `Kind` emitido sin una constante que lo respalde

#### Scenario: La lista negra no introduce un tipo de aviso
- **WHEN** la lista negra global suprime un campo de la respuesta de expansión
- **THEN** no se agrega ninguna entrada a `MetaData.Errors` ni se introduce un nuevo valor de `Kind`

## ADDED Requirements

### Requirement: Colección Errors de avisos en MetaData
El sistema SHALL exponer en `MetaData` una colección opcional `Errors` que lleva
los avisos de expansión y de normalización cuando existen; `Errors` se omite
cuando la respuesta no genera ningún aviso.

#### Scenario: Errors lleva los avisos generados
- **WHEN** una respuesta genera uno o más avisos de expansión o de normalización
- **THEN** `MetaData.Errors` incluye una entrada por cada aviso, y se omite cuando
  no hay ningún aviso

### Requirement: Tipo de aviso Kind respaldado por constante declarada
El sistema SHALL exigir que cada aviso de `Errors` declare su origen mediante un
`Kind` respaldado por una constante de tipo de aviso declarada en el código; el
conjunto de valores de `Kind` es abierto y se extiende declarando una nueva
constante por cada nuevo origen de aviso.

#### Scenario: Todo Kind emitido tiene su constante
- **WHEN** el sistema emite un aviso con un `Kind`
- **THEN** ese `Kind` corresponde a una constante declarada en el código y no
  existe ningún `Kind` emitido sin una constante que lo respalde
