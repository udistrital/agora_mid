# Spec Delta

## MODIFIED Requirements

### Requirement: Envoltorio de respuesta OAS plano
El sistema SHALL responder con un envoltorio OAS plano que incluya `Success` (booleano), `Status` (cadena con el código HTTP), `Message` (cadena) y `Data` (carga útil), en todos los verbos del API. Una escritura que solo canoniza campos derivados mantiene la misma forma del envoltorio.

#### Scenario: Respuesta exitosa de lectura
- **WHEN** una consulta de lectura se resuelve correctamente
- **THEN** la respuesta incluye `Success: true`, `Status: "200"`, un `Message` y el recurso en `Data`

#### Scenario: Respuesta de escritura
- **WHEN** una operación de escritura (`POST` o `PATCH`) se completa sin fallos parciales
- **THEN** la respuesta usa el mismo envoltorio plano `Success/Status/Message/Data` sin el bloque `MetaData`

#### Scenario: Respuesta de escritura con materialización parcial
- **WHEN** una operación de escritura tolerante materializa solo parte del subárbol por un fallo de relación no bloqueante
- **THEN** la respuesta usa `Status: "206"`, incluye `MetaData.Errors` con los avisos y `Data` con los identificadores ya materializados

#### Scenario: Escritura con campo derivado canonizado
- **WHEN** una escritura de entidad nueva corrige un campo relacional reenviado
- **THEN** la respuesta usa `Status: "206"`, conserva el subárbol canonizado en `Data` y expone el aviso en `MetaData.Errors`

#### Scenario: Compatibilidad del envoltorio con el ecosistema
- **WHEN** el controlador de errores estándar y los controladores de negocio emiten respuestas
- **THEN** ambos usan la misma raíz plana `Success/Status/Message/Data` con `Status` como cadena

### Requirement: Extensión MetaData para avisos y total de listas
El sistema SHALL incluir un bloque `MetaData` opcional que puede llevar una colección opcional `Errors` con los avisos de expansión y normalización, y un campo opcional `Length` con el total de valores posibles de un `Data` de tipo lista. Cada campo del bloque es opcional y se incluye únicamente cuando aplica; `MetaData` se omite cuando no aporta ningún dato. Cada aviso de `Errors` SHALL declarar su origen mediante un `Kind` respaldado por una constante de tipo de aviso declarada en el código; el conjunto de valores de `Kind` no se fija de forma cerrada en el contrato, sino que se extiende declarando una nueva constante por cada nuevo origen de aviso, de modo que toda respuesta de aviso posible esté asociada a una constante `Kind`.

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
