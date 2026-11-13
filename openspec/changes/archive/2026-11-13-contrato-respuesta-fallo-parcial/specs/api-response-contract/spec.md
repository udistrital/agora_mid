# Spec Delta

## ADDED Requirements

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

## MODIFIED Requirements

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
