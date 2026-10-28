# Spec Delta

## MODIFIED Requirements

### Requirement: Extensión MetaData para avisos y total de listas
El sistema SHALL incluir un bloque `MetaData` opcional que puede llevar una colección opcional `Errors` con los avisos de expansión y un campo opcional `Length` con el total de valores posibles de un `Data` de tipo lista. Cada campo del bloque es opcional y se incluye únicamente cuando aplica; `MetaData` se omite cuando no aporta ningún dato. El bloque `Errors` se incluye solo cuando existe al menos un aviso de expansión (error de relación tolerado, reducción de profundidad, ciclo o uso del resolver genérico). La supresión de un campo por la lista negra global no genera ningún aviso.

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
- **WHEN** se registran avisos de distinto origen
- **THEN** cada entrada de `MetaData.Errors` identifica su `Kind` exactamente entre `relation`, `depth`, `cycle` o `generic_resolver`

#### Scenario: Todo tipo de aviso emitido tiene una constante declarada
- **WHEN** el sistema emite un aviso de expansión con un `Kind`
- **THEN** ese `Kind` corresponde a una constante de tipo de aviso declarada, incluido `generic_resolver`, y no existe un `Kind` emitido sin constante

#### Scenario: La lista negra no introduce un tipo de aviso
- **WHEN** la lista negra global suprime un campo de la respuesta de expansión
- **THEN** no se agrega ninguna entrada a `MetaData.Errors` ni se introduce un nuevo valor de `Kind`
