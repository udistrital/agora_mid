# Spec Delta

## ADDED Requirements

### Requirement: Listado del proveedor como colección
El sistema SHALL exponer el listado de proveedores mediante `POST /v1/proveedor/consulta/lista`, recibiendo en el cuerpo JSON el parámetro `query` (lista de condiciones de filtro de primer nivel), `fields` (lista de rutas a expandir), `limit` y `offset`, y devolviendo en `Data` la lista de proveedores expandidos dentro del envoltorio de respuesta estándar. La URL es estable y no transporta la selección de campos ni los filtros en la query string.

#### Scenario: Listado de proveedores
- **WHEN** se solicita `POST /v1/proveedor/consulta/lista` con un cuerpo JSON válido
- **THEN** la respuesta incluye en `Data` la lista de proveedores dentro del envoltorio OAS plano con `Status: "200"`

#### Scenario: El endpoint recibe y aplica los parámetros del cuerpo
- **WHEN** el cuerpo incluye `query`, `fields`, `limit` y `offset`
- **THEN** el sistema aplica cada parámetro recibido con su valor (filtra por `query`, expande según `fields` y pagina con `limit`/`offset`), sin ignorar ninguno

#### Scenario: Cuerpo JSON malformado en el listado
- **WHEN** la solicitud no entrega un objeto JSON válido como cuerpo
- **THEN** el sistema responde `400 Bad Request` sin consultar el CRUD dueño

### Requirement: Filtros de primer nivel del proveedor
El sistema SHALL aceptar en `query` condiciones `Campo[__operador]:valor` únicamente sobre columnas filtrables de la tabla proveedor (primer nivel), validar el operador contra el conjunto soportado por el ORM del CRUD y la coherencia del valor, y trasladar las válidas al `query` del CRUD dueño combinadas con AND. Una condición inválida se rechaza con `400`.

#### Scenario: Condición por columna del proveedor
- **WHEN** `query` incluye una condición sobre un campo filtrable del proveedor (por ejemplo `TerceroId:15` o `Id__in:15|16|17`)
- **THEN** la respuesta entrega solo los proveedores que la satisfacen, trasladando la condición al CRUD dueño

#### Scenario: Condición por un campo de relación en el listado
- **WHEN** `query` incluye una condición sobre un campo de una relación del proveedor (por ejemplo `Tercero.NombreCompleto:...`)
- **THEN** el sistema responde `400 Bad Request` indicando que el campo no es filtrable, sin consultar el CRUD dueño

#### Scenario: Operador o valor inválido en una condición
- **WHEN** `query` incluye una condición con un operador no soportado o un valor incoherente con su operador
- **THEN** el sistema responde `400 Bad Request` identificando la condición inválida, sin consultar el CRUD dueño

### Requirement: Paginación del listado del proveedor
El sistema SHALL delegar la paginación del listado de proveedores al CRUD dueño trasladando `limit` y `offset`, aplicando valores por defecto y un máximo configurado cuando se omiten o exceden, y exponer en `MetaData.Total` el total del conjunto que el CRUD entrega en su campo `Total`.

#### Scenario: Página de proveedores
- **WHEN** el cuerpo incluye `limit` y `offset` válidos
- **THEN** la respuesta entrega a lo sumo `limit` proveedores a partir de `offset`, delegando la ventana al CRUD dueño

#### Scenario: Listado con total del conjunto
- **WHEN** se lista proveedores y el CRUD dueño entrega el conteo total en su campo `Total`
- **THEN** la respuesta expone ese total en `MetaData.Total`, reflejando el conjunto filtrado completo y no el tamaño de la página

### Requirement: Expansión de cada proveedor listado
El sistema SHALL expandir el subárbol de cada proveedor del listado según `fields` y la configuración declarativa del servicio, con la misma semántica de selección y profundidad que la lectura puntual del proveedor.

#### Scenario: Listado con expansión del núcleo
- **WHEN** el cuerpo incluye `fields` que seleccionan rutas del núcleo del proveedor
- **THEN** cada proveedor de `Data` se expande y recorta según `fields`, anidando las relaciones seleccionadas bajo su `mapped_name`

#### Scenario: fields inválido en el listado del proveedor
- **WHEN** el cuerpo entrega un `fields` inválido (ruta vacía, duplicada o no permitida)
- **THEN** el sistema responde `400 Bad Request` identificando la ruta inválida, antes de consultar el CRUD dueño

### Requirement: Métodos permitidos en el endpoint de listado
El sistema SHALL aceptar únicamente el método `POST` en `/v1/proveedor/consulta/lista` y SHALL responder `405 Method Not Allowed` a cualquier otro método, anunciando `POST` como método permitido.

#### Scenario: Método no permitido en el listado
- **WHEN** se solicita `GET`, `PUT` o `DELETE` sobre `/v1/proveedor/consulta/lista`
- **THEN** el sistema responde `405 Method Not Allowed` e indica `POST` como método permitido, sin consultar el CRUD
