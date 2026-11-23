# Proposal

## Why

El middleware ya tiene services para consumir `terceros_crud` y
`gestion_proveedores_crud`, pero no expone ninguna operación pública que los
orqueste. Registrar un proveedor persona natural hoy exige coordinar a mano
varias llamadas (consultar/crear tercero, su identificación y el proveedor).
Este cambio expone `POST /v1/proveedores/natural`, que recibe los datos de la
persona y resuelve en una sola operación el tercero, su identificación y el
proveedor, reutilizando los services existentes.

## What Changes

- Se añade el endpoint público `POST /v1/proveedores/natural` con su
  controlador Beego: lee y valida la petición, propaga su `context.Context` y
  construye la respuesta HTTP con el envoltorio del middleware
  (`Status`, `Message`, `Data`).
- Se añade una capa de coordinación (orquestación) que ejecuta el flujo
  usando los services existentes (`GetAll`, `GetOne`, `Post`, `Put` de
  terceros y proveedores) con `target` y `(ok, statusCode, err)`.
- El flujo resuelve en orden: (1) la identificación activa por
  `Numero`+`TipoDocumentoId`+`Activo` en `datos_identificacion`; (2) el
  tercero (actualizar, crear, o crear solo la identificación para un tercero
  existente); (3) el proveedor por tercero+`TipoProveedor`+`Activo`.
- `DatosIdentificacion.Numero` se trata como string en todo el flujo
  (conserva ceros iniciales). Se conserva la grafía `DatosIdentificacion` en
  entrada y respuesta.
- La respuesta devuelve 201 si se creó al menos uno de los tres registros, y
  200 si todos existían (aunque se hayan actualizado). Un fallo aplica el
  contrato de error del middleware e identifica la etapa en logs, con el host
  de los CRUD fuera del error público.

Fuera de alcance: proveedor persona jurídica; otros sub-recursos de
`gestion_proveedores_crud` (contacto, cuenta bancaria, documento, etc.);
cambios en los contratos de los CRUD.

## Capabilities

### New Capabilities
- `registro-proveedor-natural`: registro de un proveedor persona natural por
  `POST /v1/proveedores/natural`, orquestando tercero, identificación y
  proveedor sobre los CRUD existentes, con reglas de existencia/creación y el
  envoltorio de respuesta del middleware.

### Modified Capabilities
<!-- Ninguna. Los services terceros-crud-cliente y gestion-proveedores-crud-cliente
     se reutilizan sin cambiar su contrato. -->

## Impact

- **Código nuevo**: controlador del endpoint (`controllers/`), capa de
  coordinación del registro, y los modelos de entrada/salida del endpoint
  (`DatosBasicos`, `DatosIdentificacion`, `Data`). Registro de la ruta en
  `routers/router.go` bajo el namespace `/v1`.
- **Reutiliza**: `services` de terceros y proveedores (sin cambios), y los
  contratos en `docs/contratos/terceros_crud` y
  `docs/contratos/gestion_proveedores_crud`.
- **Configuración**: usa los hosts ya existentes `TercerosCrudHost` y
  `GestionProveedoresCrudHost`.
- **Decisión de negocio pendiente** (ver design.md - Open Questions): cómo
  distinguir "crear tercero" (rama B) de "tercero existente sin identificación
  activa" (rama C) cuando la búsqueda por identificación devuelve `[]`/`[{}]`
  y la petición no trae `TerceroId`. Debe resolverse antes de Apply.
