# Spec Delta

## ADDED Requirements

### Requirement: Nulabilidad del campo derivada del DBML
El generador SHALL derivar la nulabilidad de cada campo del atributo `[not null]` de
su columna en el DBML y emitirla en la tabla de campos tipada del objeto, junto a las
demás banderas de rol. Un campo cuya columna declara `[not null]`, o que es la llave
primaria, SHALL emitirse como no nulable; cualquier otro campo SHALL emitirse como
nulable. La nulabilidad es un hecho de base de datos independiente de la obligación del
cliente (`required`), que este generador no deriva.

#### Scenario: Una columna sin `[not null]` produce un campo nulable
- **WHEN** el generador procesa una columna del DBML que no declara `[not null]` ni es
  llave primaria
- **THEN** emite el campo correspondiente como nulable en la tabla de campos tipada

#### Scenario: Una columna con `[not null]` produce un campo no nulable
- **WHEN** el generador procesa una columna del DBML que declara `[not null]`
- **THEN** emite el campo correspondiente como no nulable en la tabla de campos tipada

#### Scenario: La llave primaria se emite como no nulable
- **WHEN** el generador procesa la columna de llave primaria de una tabla
- **THEN** emite el campo correspondiente como no nulable, aunque la columna no repita
  el atributo `[not null]`

#### Scenario: La salida con nulabilidad es determinista
- **WHEN** el generador se ejecuta dos veces sobre el mismo DBML y los mismos sidecars
- **THEN** la nulabilidad emitida por campo es idéntica byte a byte entre ambas
  ejecuciones, en la posición fija del campo dentro del objeto
