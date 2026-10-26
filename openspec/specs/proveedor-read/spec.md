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

#### Scenario: Ciudades sin expansión
- **WHEN** una entidad del núcleo referencia una ciudad de ubicaciones
- **THEN** la ciudad se entrega como identificador plano sin expandirse

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
