# Spec Delta

## Purpose

Expone `POST /v1/proveedores/natural` para registrar un proveedor persona
natural, orquestando el tercero, su identificación y el proveedor sobre los
CRUD de `terceros_crud` y `gestion_proveedores_crud`, y devolviendo el
envoltorio de respuesta del middleware con el estado HTTP que corresponde a si
hubo creación o solo actualización.

## ADDED Requirements

### Requirement: Contrato del endpoint de registro
El sistema SHALL exponer `POST /v1/proveedores/natural`. El controlador SHALL
leer y validar la petición (`TipoProveedor`, `DatosBasicos`,
`DatosIdentificacion`), propagar su `context.Context` a la capa de
coordinación y construir la respuesta HTTP. `DatosIdentificacion.Numero` SHALL
tratarse como string durante todo el flujo, conservando ceros iniciales. La
grafía `DatosIdentificacion` SHALL conservarse en entrada y respuesta.

#### Scenario: Petición válida
- **WHEN** llega un `POST /v1/proveedores/natural` con `TipoProveedor`, `DatosBasicos` y `DatosIdentificacion` completos
- **THEN** el controlador valida la petición, propaga el contexto a la coordinación y responde con el envoltorio del middleware

#### Scenario: Petición inválida
- **WHEN** la petición no se puede decodificar o faltan campos obligatorios
- **THEN** el controlador responde con el contrato de error del middleware sin invocar a los CRUD

#### Scenario: Numero con ceros iniciales
- **WHEN** `DatosIdentificacion.Numero` es `"0023456789"`
- **THEN** el valor viaja como string sin perder los ceros iniciales en consultas, escrituras y respuesta

### Requirement: Consulta de la identificación activa
La coordinación SHALL consultar `datos_identificacion` en `terceros_crud` con
`GetAll` combinando en una sola query los filtros `Numero`,
`TipoDocumentoId` y `Activo=true` con la sintaxis de filtros del CRUD. SHALL
comprobar primero que la consulta tuvo éxito; una consulta fallida es un error
y no autoriza crear registros.

#### Scenario: Consulta fallida
- **WHEN** la consulta de `datos_identificacion` devuelve `ok = false`
- **THEN** la coordinación detiene el flujo, no ejecuta ninguna escritura y responde con el contrato de error identificando la etapa

#### Scenario: Consulta exitosa
- **WHEN** la consulta devuelve `ok = true`
- **THEN** la coordinación interpreta la lista para decidir si hay identificación activa

### Requirement: Interpretación de la lista de identificación
La coordinación SHALL interpretar `[]` y `[{}]` (lista vacía o con un primer
elemento que es un objeto vacío) como ausencia de identificación activa. SHALL
considerar coincidencia solo cuando el primer elemento es un objeto no vacío.
SHALL distinguir el `Id` del registro de identificación del `TerceroId.Id`
(la persona). Ante varias coincidencias activas SHALL usar la primera en el
orden devuelto por el CRUD.

#### Scenario: Lista vacía o con objeto vacío
- **WHEN** la respuesta es `[]` o `[{}]`
- **THEN** la coordinación trata el caso como ausencia de identificación activa

#### Scenario: Una coincidencia activa
- **WHEN** la lista contiene un primer objeto no vacío
- **THEN** la coordinación toma su `Id` como identificación seleccionada y su `TerceroId.Id` como tercero

#### Scenario: Varias coincidencias activas
- **WHEN** la lista contiene más de un objeto no vacío
- **THEN** la coordinación usa el primero en el orden devuelto
- **AND** registra un warning con los filtros de la query, la cantidad de coincidencias y el `Id` seleccionado, enmascarando el número de documento según la convención del proyecto

### Requirement: Actualización de la identificación existente
Cuando hay identificación activa, la coordinación SHALL construir el payload de
actualización a partir del registro recuperado, reemplazando únicamente los
campos presentes en `DatosIdentificacion` de la petición y conservando su `Id`,
la relación con el tercero y los campos requeridos por el CRUD. SHALL ejecutar
un único `Put` sobre `datos_identificacion` por el `Id` seleccionado y guardar
la respuesta efectiva del `Put` en `Data.DatosIdentificacion`.

#### Scenario: Actualización por Id de identificación
- **WHEN** existe una identificación activa con `Id = N`
- **THEN** se ejecuta exactamente un `Put` sobre `datos_identificacion` por el `Id = N`
- **AND** el payload conserva `Id`, la relación con el tercero y los campos requeridos, reemplazando solo los campos enviados
- **AND** la respuesta del `Put` se guarda en `Data.DatosIdentificacion`

### Requirement: Rama A — tercero e identificación existen
Cuando existe identificación activa (y por tanto `TerceroId.Id`), la
coordinación SHALL consultar el tercero con `GetOne` por `TerceroId.Id`,
construir el payload de `Put` a partir del tercero recuperado reemplazando
únicamente `FechaNacimiento` con `DatosBasicos.FechaNacimiento`, conservar los
nombres y apellidos del tercero recuperado aunque difieran de la petición,
actualizar el tercero por su `Id` y guardar la respuesta del `Put` en
`Data.DatosBasicos`.

#### Scenario: Actualización de la fecha de nacimiento conservando nombres
- **WHEN** el tercero existe y se recupera por `TerceroId.Id`
- **THEN** se ejecuta un `Put` del tercero que reemplaza solo `FechaNacimiento`
- **AND** se conservan `PrimerNombre`, `SegundoNombre`, `PrimerApellido` y `SegundoApellido` del tercero recuperado
- **AND** la respuesta del `Put` se guarda en `Data.DatosBasicos`

### Requirement: Rama B — crear el tercero cuando no hay identificación activa
Cuando la consulta de identificación devuelve `[]` o `[{}]`, la coordinación
SHALL crear un tercero nuevo con `Post` y, con el `Id` devuelto, crear su
identificación. No SHALL intentar localizar un tercero preexistente por otra
clave ni usar `GET /tercero/identificacion`.

#### Scenario: Alta de tercero e identificación asociada
- **WHEN** la consulta de identificación devuelve `[]` o `[{}]`
- **THEN** se crea el tercero con `Post` y su respuesta completa queda en `Data.DatosBasicos`
- **AND** la identificación se crea con el `Id` devuelto por el alta del tercero
- **AND** la respuesta del alta de la identificación queda en `Data.DatosIdentificacion`

#### Scenario: No se localiza un tercero preexistente sin identificación activa
- **WHEN** no hay identificación activa
- **THEN** la coordinación no consulta `GET /tercero/identificacion` ni deduce identidad por otra vía
- **AND** procede siempre a crear el tercero

### Requirement: No operar con Id de tercero cero o no confirmado
La coordinación SHALL obtener el `Id` del tercero solo de una respuesta
efectiva: `TerceroId.Id` de una identificación activa, o el `Id` devuelto por
el alta del tercero. SHALL evitar cualquier consulta o escritura con un `Id`
cero o no confirmado, y SHALL evitar deducir identidad por coincidencia de
nombres. Una referencia incompleta es un error, no una autorización para crear
registros.

#### Scenario: Id de tercero ausente o cero
- **WHEN** un paso requiere el `Id` del tercero pero no hay uno confirmado por una respuesta efectiva
- **THEN** la coordinación no ejecuta consultas ni escrituras con `Id = 0`
- **AND** trata la situación como error sin deducir identidad por nombres

### Requirement: Resolución del proveedor
Con el `Id` definitivo del tercero, la coordinación SHALL consultar
`gestion_proveedores_crud` por la combinación de tercero, `TipoProveedor` y
`Activo=true`. Si existe un proveedor activo, SHALL reutilizar su `Id` sin
ejecutar ninguna escritura en `gestion_proveedores_crud`. Si la consulta
confirma que falta, SHALL crear el proveedor asociado al tercero y al tipo
recibido, tomando el `Id` de la respuesta de creación. En ambas ramas SHALL
incorporar el `Id` en `Data.ProveedorId`.

#### Scenario: Proveedor activo existente
- **WHEN** existe un proveedor activo para el tercero y `TipoProveedor`
- **THEN** se reutiliza su `Id` y no se ejecuta ninguna escritura en `gestion_proveedores_crud`
- **AND** el `Id` queda en `Data.ProveedorId`

#### Scenario: Proveedor ausente
- **WHEN** la consulta confirma que no hay proveedor activo
- **THEN** se crea el proveedor asociado al tercero y al tipo recibido
- **AND** el `Id` de la respuesta de creación queda en `Data.ProveedorId`

### Requirement: Composición de la respuesta y estado HTTP
La coordinación SHALL devolver el envoltorio del middleware con `Status`,
`Message` y `Data`, siguiendo la convención del proyecto para el tipo de
`Status` y el texto de `Message`. `Data` SHALL contener `TipoProveedor` (el
recibido), `DatosBasicos` (respuesta del tercero), `DatosIdentificacion`
(respuesta de la identificación) y `ProveedorId`. SHALL responder HTTP 201 si
al menos uno de los tres registros fue creado, y HTTP 200 si todos existían,
aunque se hayan ejecutado actualizaciones.

#### Scenario: Todos existían (solo actualizaciones)
- **WHEN** el tercero, la identificación y el proveedor ya existían
- **THEN** la respuesta es HTTP 200 con `Data` compuesto por las respuestas efectivas

#### Scenario: Al menos una creación
- **WHEN** se creó al menos uno de los tres registros
- **THEN** la respuesta es HTTP 201

#### Scenario: Una escritura previa no fuerza 201 en flujo incompleto
- **WHEN** una operación necesaria falla después de una escritura exitosa
- **THEN** la respuesta aplica el contrato de error del middleware y no es 201

### Requirement: Manejo de errores y logs del flujo
Ante el fallo de una operación necesaria, la coordinación SHALL aplicar el
contrato de error del middleware e identificar en logs la etapa fallida,
manteniendo el host de los CRUD fuera del error público. Una escritura previa
exitosa no SHALL repetirse automáticamente.

#### Scenario: Fallo tras un alta previa
- **WHEN** una operación falla después de un alta exitosa
- **THEN** se registra la etapa fallida en logs
- **AND** se devuelve el error sin repetir automáticamente la escritura confirmada
- **AND** el host de los CRUD no aparece en el error público
