# Proposal

## Why

El middleware ya consume el CRUD de terceros con un service que delega en
`utils_oas/v2/request`. Falta el service equivalente para el CRUD de
`gestion_proveedores_crud`, cuya URL base ya está declarada
(`GestionProveedoresCrudHost`). Además, las utilidades HTTP que hoy viven en
`services/terceros.go` (construcción de URL, traducción de resultados, logs)
no están compartidas: crear proveedores copiándolas duplicaría lógica. Este
cambio añade el service de proveedores y centraliza esas utilidades en una
sola implementación reutilizable por ambos.

## What Changes

- Se crea el service `gestion_proveedores_crud` en el paquete `services` con
  `GetAll`, `GetOne`, `Post` y `Put`, con las mismas firmas y convenciones que
  terceros: `context.Context`, `target any`, retorno `(ok, statusCode, err)`,
  filtros por `url.Values`, y la ejecución vía `utils_oas/v2/request`. Lee su
  URL base de `GestionProveedoresCrudHost`.
- Se centralizan las utilidades HTTP compartidas en un nuevo archivo
  `services/http_utils.go` (paquete `services`), con una única implementación
  de `unirURL`, `construirURL`, `traducir`, `esEOFDecodificacion`,
  `etapaSinRespuesta`, `registrarLogs` y `hostPuerto`. Ambos services las usan.
- Se elimina `utils/http_client.go`: la unión de URL (`UnirURL`) se traslada a
  `services/http_utils.go` como `unirURL` privada, porque ya no cruza paquetes.
  Las structs `utils.Response`/`utils.Metadata` se conservan en
  `utils/response.go` (siguen siendo un posible `target`).
- Se parametriza por service lo que hoy está fijo a terceros: el nombre del
  service para el log (`servicio=terceros`/`servicio=gestion_proveedores`), los
  mensajes públicos y la clave/host de configuración. Un fallo de proveedores
  queda identificado como tal en los logs, con el host fuera del error público.
- Se retira la indirección `var registrar = registrarLogs`: las operaciones
  llaman `registrarLogs` directamente y las pruebas capturan los logs sin una
  variable global mutable.

Fuera de alcance: rutas públicas y controladores; definir modelos concretos de
proveedor; el verbo DELETE.

## Capabilities

### New Capabilities
- `gestion-proveedores-crud-cliente`: consumo por HTTP del CRUD de gestión de
  proveedores desde la capa de servicios (GetAll, GetOne, Post, Put), con el
  mismo contrato de terceros (target, `(ok, statusCode, err)`, error público
  neutro y registro de la causa) y usando las utilidades HTTP compartidas.

### Modified Capabilities
<!-- Ninguna. El comportamiento observable de terceros no cambia; la
     centralización de utilidades y la relocalización de UnirURL son
     refactors internos sin efecto en su contrato. -->

## Impact

- **Código nuevo**: `services/gestion_proveedores.go` y
  `services/gestion_proveedores_test.go`.
- **Código movido/centralizado**: `services/http_utils.go` (nuevo; concentra
  las utilidades compartidas y `unirURL`). `services/terceros.go` deja de
  contener esas utilidades y pasa a usar las compartidas, parametrizando su
  identidad de service.
- **Código eliminado**: `utils/http_client.go` (y su test se reubica/ajusta a
  `services`).
- **Configuración**: usa `GestionProveedoresCrudHost` (ya presente en
  `conf/app.conf`, expandida desde `GESTION_PROVEEDORES_CRUD_SERVICE`). No se
  agrega configuración nueva.
- **Dependencias**: sin cambios; se reutiliza `utils_oas/v2` ya presente.
- **Pruebas**: las de terceros siguen pasando tras el traslado; se añaden las
  de proveedores y una prueba de aislamiento (cada service llega solo a su
  servidor de prueba).
