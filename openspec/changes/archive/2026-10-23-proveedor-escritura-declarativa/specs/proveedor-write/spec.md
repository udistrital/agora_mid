# Spec Delta

## Purpose

Define la escritura del proveedor como recurso raíz mediante cuerpo anidado,
creando y reconciliando en cascada su subárbol del núcleo a través de los CRUD
dueños, sin que el MID sea dueño de los datos.

## ADDED Requirements

### Requirement: Creación del subárbol del proveedor
El sistema SHALL exponer `POST /v1/proveedor` que recibe un cuerpo anidado y crea
el subárbol del proveedor en cascada a través de los CRUD dueños, devolviendo los
identificadores materializados en el envoltorio estándar.

#### Scenario: Creación completa exitosa
- **WHEN** se envía un `POST /v1/proveedor` con un subárbol válido y todas las
  entidades se crean correctamente
- **THEN** la respuesta usa el envoltorio plano con `Status: "200"` y `Data`
  incluye los identificadores materializados del proveedor y de sus entidades
  anidadas

#### Scenario: Catálogos y parámetros solo por Id
- **WHEN** el cuerpo referencia una entidad de parámetros o catálogo
  (por ejemplo `TipoContribuyente` o `Perfil`)
- **THEN** el sistema toma únicamente su `Id` y nunca crea ni modifica el catálogo

### Requirement: Reconciliación idempotente por PATCH
El sistema SHALL exponer `PATCH /v1/proveedor` que reconcilia el subárbol: crea lo
ausente, actualiza lo presente y nunca borra, de modo que reenviar el mismo cuerpo
produce el mismo estado final.

#### Scenario: Continuación tras creación parcial
- **WHEN** un `POST`/`PATCH` previo dejó el subárbol parcialmente creado y se
  reenvía el mismo cuerpo con `PATCH`
- **THEN** las entidades ya existentes se reutilizan por su clave natural o `Id` y
  solo se crean o actualizan las faltantes, sin duplicar registros

#### Scenario: Colección sin clave natural
- **WHEN** un elemento de una colección (por ejemplo `Contacto` o
  `CuentaBancaria`) se envía sin `Id`
- **THEN** el elemento se crea; un elemento con `Id` presente se actualiza

### Requirement: Cascada entre CRUD dueños con tercero no eliminable
El sistema SHALL ejecutar la escritura del subárbol a través de los CRUD dueños
(terceros y gestión de personas), creando y modificando `tercero` y
`datos_identificacion` pero sin eliminarlos.

#### Scenario: Tercero se crea y modifica pero no se elimina
- **WHEN** una operación de escritura requiere crear o actualizar el `tercero`
- **THEN** el sistema lo crea o actualiza en el CRUD de terceros, y ninguna
  operación de escritura elimina un `tercero`

#### Scenario: Reporte de creación parcial
- **WHEN** la cascada falla en una relación tolerada después de materializar otras
- **THEN** el recurso se conserva, la respuesta usa `Status: "206"` con
  `MetaData.Errors` y `Data` incluye los identificadores ya materializados para
  permitir el reintento

