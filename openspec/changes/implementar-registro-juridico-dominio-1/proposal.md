# Proposal

## Why

El dominio 1 ya funciona para persona natural (`POST/PUT /v1/proveedores/natural`),
pero no existe la ruta equivalente para persona jurídica. El registro de una
entidad legal necesita el mismo flujo atómico de `tercero` y
`datos_identificacion`, con `RazonSocial` como nombre legal y `P_JURIDICA` como
tipo de contribuyente.

Este cambio replica la estructura del dominio natural sin duplicar el motor de
escritura ni la lógica de identificación.

## What Changes

- Se expone `POST /v1/proveedores/juridica` para registrar o continuar el tercero
  jurídico y su identificación.
- Se expone `PUT /v1/proveedores/juridica/{terceroId}` para actualizar el tercero
  y su identificación.
- `DatosBasicos` recibe `RazonSocial`, `LugarOrigen` opcional y
  `TipoContribuyenteId` obligatorio. No admite nombres de persona natural.
- El MID deriva `NombreCompleto = RazonSocial` y lo persiste en
  `tercero.NombreCompleto`, porque la tabla `tercero` no tiene columna
  `RazonSocial`.
- `TipoContribuyenteId` se valida contra `P_JURIDICA`, resuelto al arranque por
  `MID_TIPO_CONTRIBUYENTE_JURIDICA` o por catálogo.
- `TerceroWriteService` se generaliza con `EscribirTerceroJuridico` y un núcleo
  neutral compartido con natural.
- Se reutilizan `clients.HTTPClient`, `config`, `helper.Write`,
  `EntityWriteResolver` y los YAML de `config/terceros`, sin cambios en la
  configuración.
- Se mantiene el envelope `utils_oas` v2: `Success`, `Status` string, `Message`,
  `Data` y `MetaData.Errors`.

Fuera de alcance: proveedor, `ProveedorWriteService`, caracterización,
representante legal, cuenta bancaria, documentos, autenticación, estado y
`DELETE`.

## Capabilities

### New Capabilities

- `registro-proveedor-juridica`: registro y actualización del tercero persona
  jurídica y su identificación activa, con envelope `utils_oas` v2.

### Modified Capabilities

- `registro-proveedor-natural`: se refactoriza su controlador y su servicio para
  compartir helpers, sin cambiar su contrato observable.

## Impact

- **Código nuevo:** DTO jurídico, controlador jurídico, helpers compartidos de
  controlador y pruebas.
- **Código modificado:** `services/tercero_write.go`, `controllers/proveedor_natural.go`,
  `main.go`, `routers/router.go`, `services/tercero_write_test.go`.
- **Configuración:** ninguna. `config/terceros/*.yaml` se comparte con natural y
  ya declara los campos necesarios.
- **Variables de entorno:** `MID_TIPO_CONTRIBUYENTE_JURIDICA` opcional para evitar
  la consulta de catálogo al arranque.
- **Pruebas:** servicio con CRUD simulado, controlador con escritor fake y E2E con
  el CRUD simulado. `go build`, `go test` y `go vet` pasan.
