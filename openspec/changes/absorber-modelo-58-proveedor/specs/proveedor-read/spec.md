# Spec Delta

## ADDED Requirements

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

## MODIFIED Requirements

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
