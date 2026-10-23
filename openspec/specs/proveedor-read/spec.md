# proveedor-read Specification

## Purpose

Define la lectura del proveedor como recurso raíz por identificador, entregando sus entidades relacionadas del núcleo como objetos anidados mediante selección de campos y configuración declarativa, sin que el MID sea dueño de los datos.

## Requirements

### Requirement: Lectura del proveedor por identificador
El sistema SHALL exponer una lectura del proveedor que recibe el identificador del proveedor y devuelve el recurso en el envoltorio de respuesta estándar, consultando el proveedor base contra el CRUD dueño.

#### Scenario: Consulta por id de proveedor
- **WHEN** se solicita la lectura con un `id` de proveedor existente
- **THEN** la respuesta incluye el proveedor base en `Data` dentro del envoltorio OAS plano con `Status: "200"`

#### Scenario: Identificador inválido o ausente
- **WHEN** la solicitud omite el `id` o entrega un valor no positivo
- **THEN** el sistema responde `400 Bad Request` sin consultar el CRUD

#### Scenario: Proveedor inexistente
- **WHEN** el `id` solicitado no corresponde a ningún proveedor
- **THEN** el sistema responde con un estado de no encontrado y no inventa un recurso

#### Scenario: El tercero no es punto de entrada
- **WHEN** se desea obtener el tercero asociado
- **THEN** se alcanza únicamente de forma anidada mediante `fields` (por ejemplo `Tercero.Id`) y no existe una ruta de lectura propia por `tercero_id`

### Requirement: Expansión del núcleo del proveedor por selección de campos
El sistema SHALL expandir, a partir del proveedor raíz, las entidades relacionadas del núcleo según el parámetro `fields` y la configuración declarativa del servicio, anidando cada relación bajo su `mapped_name`.

#### Scenario: Expansión de relaciones del núcleo
- **WHEN** la solicitud incluye `fields` que seleccionan rutas del núcleo (tercero, datos de identificación, tipo de contribuyente, detalle natural o jurídico, perfil, contactos, cuentas bancarias)
- **THEN** la respuesta anida cada relación seleccionada bajo su `mapped_name` respetando su cardinalidad

#### Scenario: Ciudades sin expansión
- **WHEN** una entidad del núcleo referencia una ciudad de ubicaciones
- **THEN** la ciudad se entrega como identificador plano sin expandirse

#### Scenario: Campo fuera del núcleo permitido
- **WHEN** `fields` solicita una ruta no declarada en la configuración del proveedor
- **THEN** el sistema responde `400 Bad Request` identificando la ruta inválida, sin consultar el CRUD dueño ni iniciar la expansión

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
