# Spec Delta

## Purpose

Genera los archivos de configuración de relaciones del middleware a partir del
modelo DBML y un conjunto de sidecars de intención, de forma determinista y
verificable, para que la configuración no se escriba a mano.

## ADDED Requirements

### Requirement: Generación de configuración desde el DBML
El generador SHALL producir las definiciones de objeto y las referencias de
servicio a partir del modelo DBML, derivando de la estructura las tablas como
objetos, las columnas como campos tipados y las FK entre tablas como aristas
candidatas. El generador NUNCA SHALL ejecutarse como parte del arranque del API.

#### Scenario: Una tabla del DBML produce un objeto con sus campos tipados
- **WHEN** el generador procesa una tabla del DBML con sus columnas
- **THEN** emite un objeto direccionable cuyo nombre es la tabla en PascalCase y
  cuyos campos son las columnas en PascalCase con su tipo de almacenamiento

#### Scenario: El generador no participa del arranque
- **WHEN** el API arranca
- **THEN** interpreta solo los archivos de configuración ya generados y nunca lee
  el modelo DBML

### Requirement: Sidecars de intención
El generador SHALL tomar la intención que la estructura del DBML no expresa de
cuatro sidecars agrupados por capacidad, cuyas entradas se direccionan como
`esquema.tabla` o `esquema.tabla.columna`: `expand` (FK que se expanden como
aristas), `write` (política CRUD por objeto, con `natural_key` opcional),
`fk_inbound` (aristas cuya llave la porta el padre) y `blacklist` (columnas que no
llegan a la configuración).

#### Scenario: Una FK no listada en expand queda como identificador plano
- **WHEN** una FK entre tablas no aparece en el sidecar `expand`
- **THEN** el generador la emite como campo identificador plano y no como arista
  expandible

#### Scenario: La política de escritura por objeto proviene del sidecar write
- **WHEN** el sidecar `write` declara `creatable`, `updatable` o `deletable` para
  un objeto
- **THEN** el objeto generado lleva esa política, y una `natural_key` declarada se
  emite como la clave natural del objeto

#### Scenario: Una columna en blacklist no llega a la configuración
- **WHEN** una columna aparece en el sidecar `blacklist`
- **THEN** el generador la omite del objeto generado, sin eliminarla del DBML

### Requirement: Salida determinista
El generador SHALL producir una salida estable byte a byte para un mismo DBML y
los mismos sidecars, con un orden fijo de objetos, de campos dentro de cada objeto
y de entradas de cada lista, de modo que una regeneración no introduzca ruido en la
revisión.

#### Scenario: Regenerar sin cambios no altera los archivos
- **WHEN** el generador se ejecuta dos veces sobre el mismo DBML y los mismos
  sidecars
- **THEN** los archivos producidos son idénticos byte a byte entre ambas
  ejecuciones

### Requirement: Validación de los sidecars contra el DBML
El generador SHALL rechazar una entrada de sidecar que no resuelva contra el
modelo DBML: una entrada de `expand` o `fk_inbound` que no corresponde a una FK
declarada, o una columna de `blacklist` o `natural_key` que no existe en su tabla.
El error SHALL identificar la entrada responsable.

#### Scenario: Entrada de expand sin FK correspondiente falla
- **WHEN** el sidecar `expand` nombra una FK que el DBML no declara
- **THEN** el generador falla identificando la entrada, en lugar de ignorarla en
  silencio

#### Scenario: Columna inexistente en blacklist falla
- **WHEN** el sidecar `blacklist` nombra una columna que no existe en su tabla
- **THEN** el generador falla identificando la entrada
