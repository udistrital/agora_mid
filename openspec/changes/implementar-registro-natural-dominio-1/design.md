# Design

## Context

El MID debe registrar y actualizar un tercero persona natural y su identificación activa en `terceros_crud`. El dominio 1 no crea proveedor ni escribe en `gestion_proveedores_crud`.

La rama contiene un flujo monolítico previo en `services/registro_proveedor_natural.go`. Ese flujo no se usa en este cambio; se deja como legado hasta implementar el dominio 2. La implementación del dominio 1 se apoya en el motor declarativo que vive en la rama `base_engine` del mismo repositorio.

## Goals / Non-Goals

**Goals:**

- Exponer `POST /v1/proveedores/natural` y `PUT /v1/proveedores/natural/{terceroId}` con `terceroId` al final.
- Reutilizar `clients`, `config`, `helper.Write`, `WriteResolver` y `EntityWriteResolver` del proyecto base.
- Crear un `TerceroWriteService` con raíz `Tercero` y la relación `DatosIdentificacion`.
- Resolver `TipoContribuyenteId` contra `P_NATURAL` y derivar `NombreCompleto`.
- Enviar y actualizar `DigitoVerificacion`.
- Resolver la identidad por documento activo con `sortby=Id&order=asc`.
- Devolver el envelope `utils_oas` v2 con `Success`, `Status` string, `Message`, `Data` y `MetaData.Errors`.
- Reportar fallos parciales con `206` e IDs materializados.

**Non-Goals:**

- `ProveedorWriteService`, `proveedor`, `proveedor_natural`, `proveedor_juridico`, contactos, cuenta bancaria y documentos.
- Autenticación, ownership, `estado_proveedor`, `PATCH` y `DELETE`.
- Cambios en los CRUD de referencia.

## Decisions

### Reutilización del motor base

El dominio 1 usa el mismo stack del proyecto base, pero con una raíz distinta:

- `clients.HTTPClient` para transporte.
- `config.LoadRelaciones` para componer el árbol.
- `helper.Write` como motor declarativo.
- `services.EntityWriteResolver` para materializar `Tercero` y `DatosIdentificacion`.

Se crea `services/tercero_write.go` con `TerceroWriteService`, análogo a `ProveedorWriteService` pero con raíz `Tercero`.

`ProveedorWriteService` no se usa en este dominio. Su raíz es `Proveedor` y escribiría un proveedor que el dominio 1 no debe crear. Queda reservado para el dominio 2.

### Configuración del servicio tercero

Se agrega `config/terceros/relaciones_tercero.yaml`:

- `service: tercero`
- `raiz: Tercero`
- `write.policy`: creatable/updatable, sin delete.
- Una única referencia escribible: `DatosIdentificacion` con `fk`, `creatable`, `updatable`, `deletable:false`.

El objeto `Tercero` conserva sus campos escribibles y el objeto `DatosIdentificacion` declara:

- `natural_key: [Numero, TipoDocumentoId]`.
- `Activo` como campo escribible para que el MID lo inyecte en el alta.
- `DigitoVerificacion` con la grafía correcta (se corrigió `DigitoVerficacion`).

### TipoContribuyenteId

El payload envía `DatosBasicos.TipoContribuyenteId` como entero. El MID resuelve `P_NATURAL` al arranque con `TercerosService.ResolverTipoContribuyente` y valida que el ID recibido coincida. Si no coincide, responde `400`; si el tercero existente tiene otro tipo, `409`.

El campo es plano en el payload y escribible en el objeto `Tercero`; no requiere transformación adicional. El motor lo envía al CRUD como parte del cuerpo.

### NombreCompleto

El MID deriva `NombreCompleto` en el alta a partir de nombres y apellidos. En actualización no reenvía nombres ni `NombreCompleto`: el resolver recupera el registro y lo mezcla, conservando los valores existentes.

### Activo y actualización con PUT

Los CRUD de la OAS no exponen `PATCH` y el MID prohíbe ese verbo. `EntityWriteResolver` se ajustó para:

- Actualizar con `PUT`.
- Leer el registro existente antes de actualizar y mezclar el payload sobre él, para no borrar campos ausentes (`Activo`, nombres, fechas no enviadas).
- Convertir las FK que `terceros_crud` espera como objeto (`TipoContribuyenteId`, `TipoDocumentoId`, `TerceroId`) antes de enviar.
- Aplanar esas mismas FK al leerlas, para que el motor compare llaves escalares.

En el alta, `TerceroWriteService` inyecta `Activo:true` en `tercero` y `datos_identificacion`.

### Resolución de identidad

`TerceroWriteService` resuelve antes de invocar el motor:

- **POST:** busca `datos_identificacion` por `Numero + TipoDocumentoId + Activo:true`, con `sortby=Id&order=asc`. Si existe, reutiliza el `TerceroId` y el `Id` de la identificación. Si hay varias, usa la primera y agrega un aviso `ambiguous_match`. Las respuestas `[]` y `[{}]` se tratan como ausencia de identificación.
- **PUT:** valida que el `TerceroId` exista, busca su identificación activa, y verifica que el nuevo documento no pertenezca a otro tercero. Si pertenece a otro, responde `409`. Si no hay identificación activa, la crea. Las respuestas `[]` y `[{}]` se tratan como ausencia.

### Respuesta y estado HTTP

El controlador arma `Data` con:

- `TerceroId`
- `DatosBasicos` (sin la colección `DatosIdentificacion`)
- `DatosIdentificacion`
- `DatosIdentificacionId`
- `Resultado` con `Creados`, `Reutilizados`, `Actualizados`

Estados:

- `201` si hubo al menos una creación.
- `200` si solo hubo actualización/reutilización.
- `206` si el motor reporta avisos de materialización parcial.
- `400`, `404`, `409`, `500`, `502`, `504` según el error.

### utils_oas v2

`main.go` registra `customerror`, `apiStatusLib` y `security`. `utils/response.go` expone `APIResponse` con `Status` string y helpers `WriteSuccess`, `WriteSuccessWithWarnings`, `WritePartial`, `WriteError` y `AbortWithError`.

`customerror` cubre `400/401/404/409/500/501/502/509`; `206` y `504` usan el mismo `APIResponse` desde los helpers locales.

### Configuración de URLs

`config.Load` lee `conf/app.conf` y acepta tanto las claves `*_crud_url` del proyecto base como las claves actuales `TercerosCrudHost`, `GestionProveedoresCrudHost`, `ParamtrosCrudHost` y `UbicacionesCrudHost`.

## Risks / Trade-offs

- **Port del motor base:** se portó el mínimo necesario. `config` conserva solo el cargador y la configuración de `terceros`; `helper` conserva solo el motor de escritura. Los tests y el catálogo de proveedor/parámetros/ubicaciones se retiraron del dominio 1 y se recuperarán cuando se implementen los dominios 2 y 3.
- **Flujo monolítico legado:** `services/registro_proveedor_natural.go` sigue en el repositorio sin estar cableado. Se retirará o reemplazará al implementar el dominio 2.
- **Sin transacción distribuida:** un fallo después de una escritura se reporta con `206` y los IDs ya materializados; no hay rollback.
- **Actualización idempotente:** el motor omite escrituras cuando no hay cambios. `Resultado` describe la intención del flujo, no un diff campo a campo.
- **`TipoContribuyenteId` al arranque:** si no se resuelve `P_NATURAL`, el MID no inicia. Es una falla de configuración/catálogo, no una escritura parcial.

## Open Questions

Ninguna para el dominio 1. Para el dominio 2 queda pendiente:

- Cómo invocar `ProveedorWriteService` con `TerceroId` y podar la rama `Tercero`/`DatosIdentificacion` para no reescribir el dominio 1.
