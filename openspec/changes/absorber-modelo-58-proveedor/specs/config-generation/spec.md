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

### Requirement: Esquemas con CRUD que generan objetos
El generador SHALL generar objetos únicamente para las tablas de los esquemas que
tienen un CRUD en la configuración: `proveedores` (CRUD `gestion_personas_crud`),
`terceros`, `parametros` y `ubicaciones`. Las tablas de cualquier otro esquema no
generan objetos.

#### Scenario: Las tablas de un esquema con CRUD generan objetos con ese CRUD
- **WHEN** el DBML declara una tabla del esquema `ubicaciones`
- **THEN** el generador emite un objeto con `related_api` igual a `ubicaciones`

#### Scenario: Las tablas de otro esquema no generan objetos
- **WHEN** el DBML declara una tabla de un esquema sin CRUD
- **THEN** el generador no emite ningún objeto para esa tabla

### Requirement: Sidecars de intención
El generador SHALL tomar la intención que la estructura del DBML no expresa de
cinco sidecars agrupados por capacidad, cuyas entradas se direccionan como
`esquema.tabla` o `esquema.tabla.columna`: `expand` (FK que se expanden como
aristas), `write` (política CRUD por objeto, con `natural_key` opcional),
`fk_inbound` (aristas cuya llave la porta el padre), `blacklist` (columnas que no
llegan a la configuración) y `service` (parámetros de servicio por tabla raíz).

#### Scenario: Una FK no listada en expand ni en fk_inbound queda como identificador plano
- **WHEN** una FK entre tablas no aparece en el sidecar `expand` ni en el sidecar
  `fk_inbound`
- **THEN** el generador la emite como campo identificador plano y no como arista
  expandible

#### Scenario: La política de escritura por objeto proviene del sidecar write
- **WHEN** el sidecar `write` declara `creatable`, `updatable` o `deletable` para
  un objeto no raíz
- **THEN** la política se emite sobre las aristas que apuntan al objeto y no sobre el
  objeto, y una `natural_key` declarada se emite como la clave natural del objeto

#### Scenario: Una columna en blacklist no llega a la configuración
- **WHEN** una columna aparece en el sidecar `blacklist`
- **THEN** el generador la omite del objeto generado, sin eliminarla del DBML

#### Scenario: Una entrada de expand o fk_inbound fija la profundidad de su arista
- **WHEN** una entrada de `expand` o `fk_inbound` es un mapa con `depth`
- **THEN** la arista generada lleva esa profundidad, y una entrada de cadena simple
  usa la profundidad por omisión

#### Scenario: Una FK en expand y en fk_inbound prevalece como fk_inbound
- **WHEN** una misma FK aparece en el sidecar `expand` y en el sidecar `fk_inbound`
- **THEN** el generador la emite solo como arista `fk_inbound`, con la profundidad de
  la entrada de `fk_inbound`

### Requirement: Direccionamiento de una FK por su columna portadora
El generador SHALL resolver una entrada de `expand` o `fk_inbound` contra la columna
que porta la FK (la columna del lado hijo de la relación), y SHALL rechazar una
entrada que nombre otra columna como no correspondiente a una FK declarada.

#### Scenario: La entrada nombra la columna portadora de la FK
- **WHEN** una entrada de `expand` nombra la columna del hijo que porta la FK
- **THEN** el generador la resuelve como esa FK declarada

#### Scenario: La entrada nombra el lado referenciado y falla
- **WHEN** una entrada de `expand` nombra la columna del lado referenciado de una FK
  en lugar de la columna portadora
- **THEN** el generador falla identificando la entrada como no correspondiente a una
  FK declarada

### Requirement: Generación del archivo de servicio
El generador SHALL emitir un archivo de servicio `relaciones_<servicio>.yaml` por
cada entrada del sidecar `service`, con el nombre del servicio, la raíz, la política
de escritura de la raíz, los parámetros del servicio y las referencias de la raíz.
El archivo se escribe en el subdirectorio del CRUD que almacena la raíz.

#### Scenario: El archivo de servicio nombra la raíz y su servicio
- **WHEN** el generador procesa una entrada del sidecar `service` cuya clave es la
  tabla raíz
- **THEN** emite `relaciones_<servicio>.yaml` con `service` igual al nombre declarado
  y `raiz` igual al objeto de esa tabla en PascalCase

#### Scenario: Los parámetros del servicio provienen del sidecar service
- **WHEN** el sidecar `service` declara `max_depth`, `on_relation_error`,
  `max_related_ids` y `filterable_fields` para una raíz
- **THEN** el archivo de servicio generado lleva `max_depth`, `on_relation_error` y
  `max_related_ids` con los valores declarados, y `filterable_fields` con las columnas
  declaradas en PascalCase, tal como el motor de lectura las direcciona

#### Scenario: Una entrada de service que no resuelve contra el DBML falla
- **WHEN** el sidecar `service` nombra como clave una tabla que el DBML no declara, o
  un `filterable_fields` que no es columna de la tabla raíz
- **THEN** el generador falla identificando la entrada

### Requirement: Las aristas y la política de la raíz viven en el archivo de servicio
El generador SHALL trasladar las aristas cuya entidad padre es la raíz de un servicio
al archivo de servicio de esa raíz, emitir el objeto raíz con su lista de referencias
vacía, y emitir la política de escritura de la raíz del sidecar `write` en
`write.policy` del archivo de servicio. Las aristas de objetos que no son raíz
permanecen en su objeto.

#### Scenario: Una arista de la raíz se emite en el archivo de servicio
- **WHEN** una arista expandible tiene como padre la raíz de un servicio
- **THEN** la arista aparece en las referencias del archivo de servicio y el objeto
  raíz se emite con su lista de referencias vacía

#### Scenario: La política de la raíz aparece en write.policy
- **WHEN** el sidecar `write` declara política para la tabla raíz de un servicio
- **THEN** el archivo de servicio lleva esa política en `write.policy`, el objeto raíz
  no lleva política, y la `natural_key` de la raíz permanece en el objeto raíz

#### Scenario: Las aristas de un objeto no raíz no cambian de lugar
- **WHEN** una arista expandible tiene como padre un objeto que no es raíz de ningún
  servicio
- **THEN** la arista permanece en las referencias de ese objeto

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

#### Scenario: Regenerar elimina la salida generada anterior
- **WHEN** el generador se ejecuta sobre una salida que ya contiene archivos generados
- **THEN** elimina antes de escribir los YAML que llevan el encabezado de generado,
  después las carpetas que quedan vacías, y escribe la salida nueva

#### Scenario: Regenerar conserva los archivos ajenos al generador
- **WHEN** la salida contiene un YAML sin el encabezado de generado
- **THEN** el generador no lo elimina

#### Scenario: Una construcción fallida no altera la salida previa
- **WHEN** la construcción de objetos o de servicios falla
- **THEN** la salida previa queda intacta

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

#### Scenario: Profundidad no positiva falla
- **WHEN** una entrada de mapa en `expand` o `fk_inbound` declara `depth` menor que 1
- **THEN** el generador falla identificando la entrada

#### Scenario: Entrada de servicio sin nombre falla
- **WHEN** una entrada del sidecar `service` no declara `service`
- **THEN** el generador falla identificando la entrada

#### Scenario: Raíz de servicio fuera de un CRUD mapeado falla
- **WHEN** una entrada del sidecar `service` nombra como raíz una tabla de un esquema
  sin CRUD
- **THEN** el generador falla identificando la entrada

#### Scenario: Natural_key no escribible falla
- **WHEN** el sidecar `write` declara como `natural_key` una columna de `id` o de
  auditoría
- **THEN** el generador falla identificando la entrada

#### Scenario: Columna en blacklist y en expand o fk_inbound falla
- **WHEN** una misma columna aparece en el sidecar `blacklist` y en `expand` o
  `fk_inbound`
- **THEN** el generador falla identificando la entrada
