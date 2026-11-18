# proveedor-read Specification

## Purpose

Define la lectura del proveedor como recurso raíz por identificador, entregando sus entidades relacionadas del núcleo como objetos anidados mediante selección de campos y configuración declarativa, sin que el MID sea dueño de los datos.

## Requirements
### Requirement: Lectura del proveedor por identificador
El sistema SHALL exponer la lectura del proveedor mediante `POST /v1/proveedor/consulta`, recibiendo el identificador del proveedor en el campo `id` del cuerpo JSON y devolviendo el recurso en el envoltorio de respuesta estándar, consultando el proveedor base contra el CRUD dueño. La URL de la consulta es estable y no transporta la selección de campos, de modo que el gateway no deba manejar query strings extensas.

#### Scenario: Consulta por id de proveedor
- **WHEN** se solicita `POST /v1/proveedor/consulta` con un cuerpo JSON cuyo `id` de proveedor existe
- **THEN** la respuesta incluye el proveedor base en `Data` dentro del envoltorio OAS plano con `Status: "200"`

#### Scenario: Identificador inválido o ausente
- **WHEN** el cuerpo omite el `id` o entrega un valor no positivo
- **THEN** el sistema responde `400 Bad Request` sin consultar el CRUD

#### Scenario: Cuerpo JSON malformado
- **WHEN** la solicitud no entrega un objeto JSON válido como cuerpo
- **THEN** el sistema responde `400 Bad Request` sin consultar el CRUD

#### Scenario: Proveedor inexistente
- **WHEN** el `id` solicitado no corresponde a ningún proveedor
- **THEN** el sistema responde con un estado de no encontrado y no inventa un recurso

#### Scenario: El tercero no es punto de entrada
- **WHEN** se desea obtener el tercero asociado
- **THEN** se alcanza únicamente de forma anidada mediante `fields` (por ejemplo `Tercero.Id`) y no existe una ruta de lectura propia por `tercero_id`

### Requirement: Expansión del núcleo del proveedor por selección de campos
El sistema SHALL expandir, a partir del proveedor raíz, las entidades relacionadas del núcleo según el campo `fields` del cuerpo JSON y la configuración declarativa del servicio, anidando cada relación bajo su `mapped_name`. El campo `fields` es un arreglo JSON de rutas canónicas (por ejemplo `["Id","Tercero.Id","Tercero.NombreCompleto"]`) que se validan estrictamente contra la lista permitida antes de consultar el CRUD; no se normalizan variantes de mayúsculas, espacios ni caracteres.

#### Scenario: Expansión de relaciones del núcleo
- **WHEN** el cuerpo incluye `fields` que seleccionan rutas del núcleo (tercero, datos de identificación, tipo de contribuyente, detalle natural o jurídico, perfil, contactos, cuentas bancarias)
- **THEN** la respuesta anida cada relación seleccionada bajo su `mapped_name` respetando su cardinalidad

#### Scenario: Ciudades expandidas con su tipo
- **WHEN** una entidad del núcleo referencia una ciudad u otro lugar de ubicaciones y `fields` selecciona esa relación
- **THEN** el lugar se expande bajo su `mapped_name` entregando su identificador y su nombre, y su tipo de lugar anidado con su nombre, de un salto

#### Scenario: Ciudades sin expansión
- **WHEN** una entidad del núcleo referencia una ciudad de ubicaciones y `fields` no selecciona esa relación
- **THEN** la ciudad se entrega como identificador plano, sin expandirse, igual que cualquier relación no seleccionada

#### Scenario: Campo fuera del núcleo permitido
- **WHEN** `fields` solicita una ruta no declarada en la configuración del proveedor
- **THEN** el sistema responde `400 Bad Request` identificando la ruta inválida, sin consultar el CRUD dueño ni iniciar la expansión

#### Scenario: Nombre de campo no canónico
- **WHEN** `fields` incluye una ruta con una variante no canónica (diferencias de mayúsculas, espacios o caracteres) de un campo permitido
- **THEN** el sistema responde `400 Bad Request` identificando la ruta inválida y no normaliza ni reinterpreta el nombre

#### Scenario: Validación de `fields` antes de consultar el CRUD
- **WHEN** la solicitud entrega un `fields` inválido (ruta vacía, segmento vacío, duplicado o ruta no declarada)
- **THEN** el sistema valida `fields` contra la configuración declarativa y responde `400 Bad Request` antes de consultar el proveedor base, de modo que un `fields` inválido no produce ninguna llamada al CRUD dueño

### Requirement: Orquestación por servicio de negocio
El sistema SHALL orquestar la lectura mediante un servicio de negocio que trae el proveedor base, construye las opciones de expansión, provee el mapeo de cada relación a su resolver e invoca el expansor; la lógica de negocio reside en el servicio y no en los resolvers.

#### Scenario: El servicio provee el mapeo relación a resolver
- **WHEN** el servicio inicia la expansión de un proveedor
- **THEN** inyecta el mapeo de relaciones a resolvers específicos y delega el recorrido al expansor

#### Scenario: Procesos de negocio en el servicio
- **WHEN** la lectura requiere un proceso adicional de negocio (por ejemplo decidir la rama natural o jurídica, o conservar el objeto para una posible modificación)
- **THEN** ese proceso ocurre en el servicio y no dentro de un resolver

### Requirement: Métodos permitidos en el endpoint de consulta
El sistema SHALL aceptar únicamente el método `POST` en `/v1/proveedor/consulta` y SHALL responder `405 Method Not Allowed` a cualquier otro método, anunciando `POST` como método permitido.

#### Scenario: Método no permitido en la consulta
- **WHEN** se solicita `GET`, `PUT` o `DELETE` sobre `/v1/proveedor/consulta`
- **THEN** el sistema responde `405 Method Not Allowed` e indica `POST` como método permitido, sin consultar el CRUD

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

#### Scenario: Listado mixto de proveedores naturales y jurídicos
- **WHEN** el listado devuelve en la misma página proveedores con
  `TipoRegistroId` de persona natural y de persona jurídica
- **THEN** cada proveedor natural expande su rama `ProveedorNatural` y cada
  proveedor jurídico expande su rama `ProveedorJuridico`, según el
  `TipoRegistroId` de cada registro, sin aplicar a toda la página la rama de un
  único tipo

### Requirement: Métodos permitidos en el endpoint de listado
El sistema SHALL aceptar únicamente el método `POST` en `/v1/proveedor/consulta/lista` y SHALL responder `405 Method Not Allowed` a cualquier otro método, anunciando `POST` como método permitido.

#### Scenario: Método no permitido en el listado
- **WHEN** se solicita `GET`, `PUT` o `DELETE` sobre `/v1/proveedor/consulta/lista`
- **THEN** el sistema responde `405 Method Not Allowed` e indica `POST` como método permitido, sin consultar el CRUD

### Requirement: Expansión del árbol extendido del proveedor
El sistema SHALL expandir, a partir del proveedor raíz y según `fields`, las
entidades del modelo extendido: `estado_proveedor`, `documento`,
`perfil_proveedor`, `titulo_proveedor_natural`, `actividad_economica_proveedor`,
`representacion` y `persona_juridica_tipo_entidad`, cada una anidada bajo su
`mapped_name` con su cardinalidad declarada.

#### Scenario: Expansión de colecciones del modelo extendido
- **WHEN** `fields` selecciona rutas de colecciones del modelo extendido (por
  ejemplo `PerfilProveedor`, `Documento` o `EstadoProveedor`)
- **THEN** la respuesta anida cada colección bajo su `mapped_name` como lista,
  respetando su cardinalidad

### Requirement: Aplanado de la especialización en la lectura del proveedor
El sistema SHALL proyectar los campos de la especialización del proveedor
(`proveedor_natural` o `proveedor_juridico`, según `tipo_proveedor`) dentro del
objeto proveedor en lugar de anidarlos bajo un `mapped_name`, de un solo salto y
solo en la dirección proveedor->especialización.

#### Scenario: Los campos de la especialización aparecen en el proveedor
- **WHEN** se expande un proveedor y su `tipo_proveedor` indica persona natural o
  jurídica
- **THEN** los campos de la especialización correspondiente aparecen al nivel del
  objeto proveedor, no bajo un objeto anidado

#### Scenario: Listado mixto aplana la especialización de cada registro
- **WHEN** un listado devuelve en la misma página proveedores de persona natural y
  de persona jurídica
- **THEN** cada registro aplana los campos de su propia especialización según su
  `tipo_proveedor`, sin aplicar a toda la página la rama de un único tipo
