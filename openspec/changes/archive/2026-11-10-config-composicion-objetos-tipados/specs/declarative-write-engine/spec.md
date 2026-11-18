# Spec Delta

## MODIFIED Requirements

### Requirement: Escritura declarativa dirigida por configuración
El sistema SHALL ejecutar la escritura en cascada únicamente a partir del árbol
efectivo compuesto por servicio (ver `config-composition`), cargado y aplanado al
arranque, y nunca a partir de endpoints, tablas o rutas provistos por el cliente. El
motor consume el árbol ya compuesto y no resuelve referencias de objetos por sí
mismo. Los campos que el cliente puede enviar para una entidad son los que la tabla
de campos tipada unificada del objeto marca con el rol `writable`; los campos
derivados de una relación se reconocen según la configuración aunque no lleven el rol
`writable`.

#### Scenario: La estructura de escritura proviene de la configuración
- **WHEN** se procesa un cuerpo de escritura anidado
- **THEN** el motor usa el árbol de relaciones y las políticas de escritura
  declaradas en configuración para decidir qué entidades materializar y en qué
  CRUD, sin aceptar destinos del cliente

#### Scenario: Campo de escritura no permitido se rechaza
- **WHEN** el cuerpo incluye un campo que no lleva el rol `writable` para una
  relación, o intenta cambiar el valor de un campo con rol `immutable` que no es un
  campo derivado reenviado
- **THEN** el sistema responde `400 Bad Request` identificando el campo inválido y
  no materializa ninguna escritura

#### Scenario: Campo derivado relacional reenviado se reconoce
- **WHEN** el cuerpo incluye un campo derivado por `fk`, `shared_pk` o `fk_inbound`
  que corresponde a una relación declarada
- **THEN** el motor no lo rechaza como campo desconocido y lo valida contra la
  llave autoritativa de la relación

#### Scenario: Campo inmutable derivado reenviado coincidente
- **WHEN** el cuerpo reenvía un campo que lleva a la vez el rol `immutable` y es
  derivado de una relación, con el valor que coincide con la llave autoritativa
- **THEN** el sistema no lo rechaza por inmutable y lo concilia como campo
  derivado, sin materializar un cambio prohibido

### Requirement: Coherencia de configuración lectura/escritura
El sistema SHALL garantizar, por construcción del árbol efectivo compuesto, que los
roles de escritura (`writable`, `immutable`) y de lectura (`readable`) provienen de la
misma tabla de campos tipada por objeto, de modo que no exista un bloque de escritura
que contradiga la contraparte de lectura. El cargador de composición SHALL abortar el
inicio cuando una relación escribible del árbol efectivo no declare una propagación de
llave válida.

#### Scenario: Relación de escritura sin contraparte de lectura
- **WHEN** un objeto declara sus campos en la tabla de campos tipada unificada
- **THEN** el rol de lectura y los roles de escritura de cada campo provienen de la
  misma definición, sin un bloque de escritura separado que pueda declarar una
  relación escribible inexistente en el árbol de lectura

#### Scenario: Propagación de llave ausente o inválida
- **WHEN** una relación escribible del árbol efectivo no declara un modo de
  propagación de llave válido
- **THEN** el API no inicia y reporta el objeto o la referencia afectada
