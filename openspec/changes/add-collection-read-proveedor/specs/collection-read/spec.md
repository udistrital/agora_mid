# Spec Delta

## Purpose

Define la lectura de colección de un recurso raíz de forma genérica y declarativa: lista registros aplicando filtros de primer nivel contra el CRUD dueño, delega la paginación y expande el subárbol de cada elemento, sin que el MID conozca ningún esquema concreto ni sea dueño de los datos.

## ADDED Requirements

### Requirement: Listado declarativo de un recurso raíz
El sistema SHALL listar un recurso raíz configurado de forma declarativa por servicio, devolviendo los registros en el envoltorio de respuesta estándar como una lista en `Data`, sin que el motor conozca el esquema del recurso: el recurso y sus campos listables se declaran en configuración.

#### Scenario: Listado sin filtros
- **WHEN** se solicita el listado de un recurso raíz sin filtros
- **THEN** la respuesta incluye en `Data` la lista de registros del recurso dentro del envoltorio estándar con `Status: "200"`

#### Scenario: Listado vacío
- **WHEN** el recurso raíz no tiene registros que satisfagan la solicitud
- **THEN** la respuesta entrega `Data` como una lista vacía `[]` y no inventa registros

### Requirement: Filtrado de primer nivel mediante query declarativo
El sistema SHALL aceptar filtros en un parámetro `query` que es una lista de condiciones con la forma `Campo[__operador]:valor`, donde `Campo` es una columna de la tabla raíz (primer nivel) declarada como filtrable y `operador` es un sufijo opcional. El sistema SHALL trasladar las condiciones válidas al parámetro `query` del CRUD dueño combinándolas con AND, sin consultar otros CRUD para filtrar.

#### Scenario: Condición de igualdad implícita
- **WHEN** `query` incluye una condición sin operador (por ejemplo `TerceroId:15`)
- **THEN** el motor la interpreta como igualdad y la traslada al `query` del CRUD dueño

#### Scenario: Condición con operador explícito
- **WHEN** `query` incluye una condición con operador (por ejemplo `Id__in:15|16|17`)
- **THEN** el motor traslada la condición con su operador al `query` del CRUD dueño y devuelve solo los registros que la satisfacen

#### Scenario: Varias condiciones se combinan con AND
- **WHEN** `query` incluye más de una condición
- **THEN** el motor exige que cada registro satisfaga todas las condiciones (conjunción) al trasladarlas al CRUD dueño

#### Scenario: Condición sobre un campo no permitido
- **WHEN** `query` incluye una condición sobre un campo no declarado como filtrable para el recurso
- **THEN** el sistema responde `400 Bad Request` identificando el campo inválido, sin consultar el CRUD dueño

#### Scenario: Condición de segundo nivel no soportada
- **WHEN** `query` incluye una condición cuya ruta atraviesa una relación de otro CRUD (por ejemplo `Tercero__NombreCompleto:...` o `Tercero.NombreCompleto:...`)
- **THEN** el sistema responde `400 Bad Request` indicando que el campo no es filtrable, sin consultar el CRUD dueño

### Requirement: Validación estricta de operadores de filtro
El sistema SHALL validar cada condición de `query` ANTES de consultar el CRUD dueño: el operador, cuando está presente, SHALL pertenecer al conjunto soportado por el ORM del CRUD (`exact`, `iexact`, `contains`, `icontains`, `gt`, `gte`, `lt`, `lte`, `startswith`, `istartswith`, `endswith`, `iendswith`, `in`, `isnull`), y la forma del valor SHALL ser coherente con el operador. Una condición malformada se rechaza con `400` sin consultar el CRUD.

#### Scenario: Operador no reconocido
- **WHEN** una condición usa un sufijo de operador que no pertenece al conjunto soportado
- **THEN** el sistema responde `400 Bad Request` identificando la condición inválida, sin consultar el CRUD dueño

#### Scenario: Valor incoherente con el operador de lista
- **WHEN** una condición usa el operador `in` con una lista vacía o un valor que no es una lista separada por `|`
- **THEN** el sistema responde `400 Bad Request` sin consultar el CRUD dueño

#### Scenario: Valor incoherente con un operador escalar
- **WHEN** una condición usa un operador escalar (por ejemplo `gt`) con una lista de varios valores, o `isnull` con un valor que no es booleano
- **THEN** el sistema responde `400 Bad Request` sin consultar el CRUD dueño

### Requirement: Paginación delegada al CRUD dueño
El sistema SHALL delegar la paginación al CRUD dueño trasladando `limit` y `offset` a su consulta, de modo que la ventana de registros se calcula en el CRUD y no en el MID.

#### Scenario: Paginación con limit y offset
- **WHEN** la solicitud incluye `limit` y `offset` válidos
- **THEN** el motor los traslada al CRUD dueño y la respuesta entrega a lo sumo `limit` registros a partir de `offset`

#### Scenario: Paginación por defecto
- **WHEN** la solicitud omite `limit` u `offset`
- **THEN** el motor aplica los valores por defecto configurados y acota el tamaño de la página a un máximo configurado

#### Scenario: Paginación inválida
- **WHEN** `limit` u `offset` no son enteros no negativos (o `limit` no es positivo)
- **THEN** el sistema responde `400 Bad Request` sin consultar el CRUD dueño

### Requirement: Total del conjunto en la paginación
El sistema SHALL tomar el total de registros del conjunto filtrado del campo `Total` del envoltorio que entrega el CRUD dueño y exponerlo en `MetaData.Total` de la respuesta de listado, de modo que el consumidor conozca el tamaño del conjunto más allá de la página devuelta.

#### Scenario: El listado expone el total del CRUD
- **WHEN** el CRUD dueño entrega el conteo total del conjunto filtrado en el campo `Total` de su envoltorio
- **THEN** la respuesta de listado del MID expone ese valor en `MetaData.Total`

#### Scenario: El total refleja el conjunto filtrado, no la página
- **WHEN** la página devuelta contiene menos registros que el total del conjunto
- **THEN** `MetaData.Total` refleja el tamaño del conjunto filtrado completo y no el número de elementos de la página

### Requirement: Expansión del subárbol de cada elemento listado
El sistema SHALL expandir el subárbol de cada registro listado reutilizando la expansión de objetos anidados: la selección por `fields` y el límite de profundidad se aplican por elemento igual que en la lectura puntual del recurso.

#### Scenario: Listado con selección de campos
- **WHEN** la solicitud incluye `fields` con rutas válidas del recurso
- **THEN** cada elemento de `Data` se expande y recorta según `fields`, anidando las relaciones bajo su `mapped_name`

#### Scenario: fields inválido en el listado
- **WHEN** la solicitud incluye un `fields` con una ruta vacía, duplicada o no permitida
- **THEN** el sistema responde `400 Bad Request` identificando la ruta inválida, sin consultar el CRUD dueño

#### Scenario: Error de relación durante la expansión del listado
- **WHEN** la expansión de una relación de un elemento falla y la política es tolerante
- **THEN** el elemento se conserva, la rama fallida usa `null` o `[]` y se agrega un aviso no sensible, sin abortar el listado completo
