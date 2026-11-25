# Proposal

## Why

El registro de proveedores necesita separarse en dominios. El primer dominio es el registro de la persona natural: `tercero` y `datos_identificacion`. Hoy el endpoint monolítico mezcla tercero, identificación y proveedor, y no incorpora las correcciones necesarias (`TipoContribuyenteId`, `NombreCompleto`, `DigitoVerificacion`, `PUT`, `sortby` y envelope `utils_oas` v2).

Este cambio deja funcionando el dominio 1 de forma atómica y reutiliza el motor declarativo del proyecto base.

## What Changes

- Se expone `POST /v1/proveedores/natural` para registrar o continuar el tercero y su identificación.
- Se expone `PUT /v1/proveedores/natural/{terceroId}` para actualizar el tercero y su identificación.
- El payload recibe `TipoContribuyenteId` en `DatosBasicos` y lo valida contra `P_NATURAL`.
- El MID deriva `NombreCompleto`.
- Se envía y actualiza `DigitoVerificacion`.
- La consulta de identificación usa `sortby=Id&order=asc`.
- Se reutiliza el `base_engine`: `clients`, `config`, `helper.Write`, `EntityWriteResolver` y un nuevo `TerceroWriteService`.
- `ProveedorWriteService` no se usa en este dominio; queda reservado para la caracterización.
- Se adopta el envelope `utils_oas` v2: `Success`, `Status` string, `Message`, `Data` y `MetaData.Errors`.
- Se corrigen los clientes del CRUD para actualizar con `PUT` (no `PATCH`) y para envolver las FK de `terceros_crud` como objetos `{Id:n}`.

Fuera de alcance: proveedor, `ProveedorWriteService`, caracterización, cuenta bancaria, documentos, autenticación, estado y `DELETE`.

## Capabilities

### New Capabilities

- `registro-proveedor-natural`: registro y actualización del tercero persona natural y su identificación activa, con envelope `utils_oas` v2.

### Modified Capabilities

Ninguna capacidad existente cambia fuera de `registro-proveedor-natural`.

## Impact

- **Código nuevo:** `clients`, `config`, `helper`, `responses` y resolvers de escritura portados desde `base_engine`; `TerceroWriteService`; controlador del dominio 1; DTOs; envelope.
- **Código modificado:** `main.go`, `routers/router.go`, `utils/response.go`, configuraciones YAML de `terceros`.
- **Configuración:** `config.Load` acepta las claves actuales (`TercerosCrudHost`, `GestionProveedoresCrudHost`, `ParamtrosCrudHost`, `UbicacionesCrudHost`) además de las claves `*_crud_url`.
- **Pruebas:** pruebas del servicio con un `terceros_crud` simulado y pruebas del controlador con un escritor fake. `go build`, `go test` y `go vet` pasan.
- **Fuera de alcance:** `gestion_proveedores_crud` y `ProveedorWriteService` quedan para el dominio 2.
