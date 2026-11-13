# Spec Delta

## MODIFIED Requirements

### Requirement: Coherencia de configuración lectura/escritura
El sistema SHALL garantizar, por construcción del árbol efectivo compuesto, que los
roles de escritura (`writable`, `immutable`), de lectura (`readable`) y la nulabilidad
de cada campo provienen de la misma tabla de campos tipada por objeto, de modo que no
exista un bloque de escritura que contradiga la contraparte de lectura ni una fuente
de nulabilidad paralela. El cargador de composición SHALL abortar el inicio cuando una
relación escribible del árbol efectivo no declare una propagación de llave válida.

#### Scenario: Relación de escritura sin contraparte de lectura
- **WHEN** un objeto declara sus campos en la tabla de campos tipada unificada
- **THEN** el rol de lectura y los roles de escritura de cada campo provienen de la
  misma definición, sin un bloque de escritura separado que pueda declarar una
  relación escribible inexistente en el árbol de lectura

#### Scenario: Propagación de llave ausente o inválida
- **WHEN** una relación escribible del árbol efectivo no declara un modo de
  propagación de llave válido
- **THEN** el API no inicia y reporta el objeto o la referencia afectada

#### Scenario: La nulabilidad proviene de la tabla de campos tipada
- **WHEN** un objeto declara sus campos en la tabla de campos tipada unificada
- **THEN** la nulabilidad de cada campo proviene de esa misma definición y se proyecta
  al árbol efectivo compuesto, sin una fuente de nulabilidad separada del bloque de
  escritura

## ADDED Requirements

### Requirement: Rechazo de null en campo escalar no nulable antes de escribir
El sistema SHALL rechazar con `400`, antes de cualquier escritura, un campo escalar
escribible presente con valor `null` cuya nulabilidad en el árbol efectivo compuesto lo
declara no nulable, identificando el campo y sin materializar ninguna escritura. La
nulabilidad usada en la decisión SHALL provenir del árbol efectivo compuesto, no de una
fuente paralela derivada por convención de nombres.

#### Scenario: Null en un escalar no nulable se rechaza sin escribir
- **WHEN** el cuerpo presenta `null` en un campo escalar escribible que el árbol
  efectivo declara no nulable
- **THEN** el sistema responde `400` identificando el campo y no envía ninguna
  escritura a los CRUD dueños

#### Scenario: Null en un escalar nulable no se rechaza por esta validación
- **WHEN** el cuerpo presenta `null` en un campo escalar escribible que el árbol
  efectivo declara nulable
- **THEN** esta validación no rechaza el campo y la escritura continúa

#### Scenario: La decisión usa la nulabilidad del árbol efectivo compuesto
- **WHEN** el motor evalúa un `null` escalar de una entidad anidada del árbol
- **THEN** resuelve la nulabilidad del campo por la tabla de campos del objeto que el
  árbol efectivo asigna a esa entidad, no por el nombre de la clave del cuerpo
