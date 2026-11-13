# api-response-contract Specification

## Purpose

Define el contrato de respuesta uniforme del API en formato OAS plano más una extensión opcional `MetaData` para avisos de expansión y el total de valores posibles de las listas, alineado con el envoltorio estándar del ecosistema (utils_oas).

## Requirements
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
constante por cada nuevo origen de aviso. Los orígenes de fallo durante la
materialización se expresan como valores de `Kind` (`dependency_failure`,
`timeout`) sobre este mismo conjunto, sin introducir un segundo eje de
clasificación del error.

#### Scenario: Todo Kind emitido tiene su constante
- **WHEN** el sistema emite un aviso con un `Kind`
- **THEN** ese `Kind` corresponde a una constante declarada en el código y no
  existe ningún `Kind` emitido sin una constante que lo respalde

#### Scenario: El fallo de dependencia se distingue por Kind
- **WHEN** la cascada se detiene por un fallo de una dependencia remota
- **THEN** el aviso correspondiente lleva `Kind: dependency_failure` y no un eje de
  clasificación distinto de `Kind`

#### Scenario: El tiempo de espera se distingue por Kind
- **WHEN** la cascada se detiene porque una dependencia remota agota su tiempo de
  espera
- **THEN** el aviso correspondiente lleva `Kind: timeout`, distinto de
  `dependency_failure` y de `field_normalization`

#### Scenario: La taxonomía de fallo no agrega un segundo eje
- **WHEN** el sistema clasifica un fallo de materialización como dependencia o
  tiempo de espera
- **THEN** la clasificación se expresa únicamente mediante el valor de `Kind` del
  aviso, sin un campo discriminador adicional paralelo a `Kind`

### Requirement: Mapa de códigos de estado de la escritura
El sistema SHALL mapear el resultado de una escritura `POST`/`PATCH /v1/proveedor`
a un código de estado según el momento del fallo en la línea de tiempo de la
escritura: `200` cascada completa sin avisos; `206` materialización con avisos o
detenida a mitad de camino; `400` validación previa a toda escritura; `502`/`504`
fallo de dependencia en el pre-pass de solo lectura antes de materializar nada.

#### Scenario: Cascada completa sin avisos
- **WHEN** la cascada materializa todo el subárbol sin canonizar campos ni detenerse
- **THEN** la respuesta usa `Status: "200"` y omite el bloque `MetaData.Errors`

#### Scenario: Fallo de dependencia en el pre-pass previo a toda escritura
- **WHEN** una dependencia remota es inalcanzable o agota su tiempo de espera
  durante el pre-pass de solo lectura, antes de materializar ninguna entidad
- **THEN** la respuesta usa `Status: "502"` para una dependencia inalcanzable o
  `Status: "504"` para un tiempo de espera agotado, y no incluye ningún `Data`
  parcial porque no se materializó nada

#### Scenario: La validación previa no usa 206
- **WHEN** una validación del árbol completo determina un `400` antes de cualquier
  escritura
- **THEN** la respuesta usa `Status: "400"` y nunca `206`, porque no existe ningún
  subárbol materializado que reportar

### Requirement: Respuesta 206 de materialización detenida a mitad de camino
El sistema SHALL responder `206` cuando la cascada, tras materializar al menos una
entidad, se detiene por un fallo o un tiempo de espera de una dependencia remota
(stop-abort), devolviendo en `Data` el subárbol ya materializado y en
`MetaData.Errors` una entrada que nombra el punto de parada y su causa, sin
continuar materializando los nodos restantes.

#### Scenario: Stop-abort por fallo de dependencia a mitad de cascada
- **WHEN** la cascada materializa una o más entidades y luego un CRUD dueño falla al
  materializar el siguiente nodo
- **THEN** la respuesta usa `Status: "206"`, `Data` incluye los identificadores de
  las entidades ya materializadas y `MetaData.Errors` incluye una entrada con
  `Path`, `Service`, `Message` y `Kind` que nombra el nodo donde se detuvo

#### Scenario: Stop-abort por tiempo de espera a mitad de cascada
- **WHEN** la cascada materializa una o más entidades y luego un CRUD dueño agota su
  tiempo de espera al materializar el siguiente nodo
- **THEN** la respuesta usa `Status: "206"`, `MetaData.Errors` lleva una entrada con
  `Kind: timeout` que nombra el nodo, y los nodos posteriores no se intentan

#### Scenario: Los nodos posteriores al punto de parada no se materializan
- **WHEN** la cascada se detiene en un nodo por un fallo de dependencia
- **THEN** ninguna entidad posterior en el orden topológico se envía a su CRUD
  dueño, y `Data` refleja únicamente lo materializado hasta el punto de parada
