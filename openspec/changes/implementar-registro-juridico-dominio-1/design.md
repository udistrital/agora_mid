# Design

## Context

El MID debe registrar y actualizar un tercero persona jurídica y su
identificación activa en `terceros_crud`. El dominio 1 no crea proveedor ni
escribe en `gestion_proveedores_crud`.

El dominio natural ya resolvió el motor declarativo, la identificación por
documento activo, el envelope `utils_oas` v2 y el tratamiento de `[]`/`[{}]`.
Este cambio reutiliza esa base y solo agrega la variante jurídica.

## Goals / Non-Goals

**Goals:**

- Exponer `POST /v1/proveedores/juridica` y `PUT /v1/proveedores/juridica/{terceroId}`
  con `terceroId` al final.
- Recibir `RazonSocial`, `LugarOrigen` opcional y `TipoContribuyenteId`
  obligatorio; rechazar nombres de persona natural.
- Persistir `NombreCompleto = RazonSocial`, normalizada.
- Validar `TipoContribuyenteId` contra `P_JURIDICA`.
- Reutilizar el núcleo de `TerceroWriteService` sin duplicar la lógica de
  identificación ni de clasificación.
- Mantener el envelope y los estados del dominio 1.

**Non-Goals:**

- `ProveedorWriteService`, `proveedor`, `proveedor_juridico`, representante legal.
- `info_complementaria_tercero`, contactos, perfil, cuenta bancaria y documentos.
- Autenticación, ownership, `estado_proveedor`, `PATCH` y `DELETE`.
- Cambios en los CRUD de referencia o en los YAML de `config/terceros`.

## Decisions

### Reutilización del motor base

Se reutiliza el mismo `TerceroWriteService` del dominio natural. La raíz sigue
siendo `Tercero` y el único hijo escribible es `DatosIdentificacion`.

No se crea un servicio jurídico paralelo ni se usa `ProveedorWriteService`.

### Núcleo neutral

`EscribirTercero` y `EscribirTerceroJuridico` delegan en un método privado
`escribir` que recibe `entradaEscrituraTercero`:

- `tipoContribuyente`
- `variante` (`natural` / `juridica`)
- `natural *DatosBasicosTercero`
- `juridica *DatosBasicosJuridico`
- `identificacion`

El flujo compartido valida el tipo esperado, resuelve la identificación, valida
el tercero existente, materializa y clasifica. Solo `construirCuerpo` cambia
según la variante.

### RazonSocial y NombreCompleto

`tercero` no tiene columna `RazonSocial`. Por eso:

- El alta envía `NombreCompleto = normalizar(RazonSocial)`.
- El `PUT` actualiza `NombreCompleto` con la nueva `RazonSocial`.
- El `POST` que reutiliza un tercero existente no reescribe el nombre.
- La respuesta proyecta `RazonSocial` desde `NombreCompleto` y omite los campos
  de nombre natural vacíos.

No se agrega `RazonSocial` a `objetos_tercero.yaml` ni se envía al CRUD.

### LugarOrigen

Se envía en el alta y en el `PUT` cuando llega. El `POST` que reutiliza un
tercero existente lo conserva.

### DigitoVerificacion

Permanece opcional. El MID no conoce el mapa de tipos de documento que identifica
un NIT y no se hardcodea. Se escribe cuando llega.

### Identidad inmutable en PUT

La identificación activa de un tercero existente no se reasigna. Antes de
escribir, `TerceroWriteService` compara `Numero` y `TipoDocumentoId` del payload
con la identificación activa:

- Si difieren, responde `409` con `ErrIdentificacionInmutable` y no escribe.
- Si el nuevo documento pertenece a otro tercero, responde `409` con
  `ErrIdentificacionDeOtroTercero`.
- Si coinciden, actualiza `CiudadExpedicion`, `FechaExpedicion` y
  `DigitoVerificacion`.

La regla se aplica a las variantes natural y jurídica porque comparten el mismo
núcleo de identificación.

### Validaciones de entrada

Antes de leer o escribir, el servicio valida:

- `RazonSocial` normalizada, 1–255, sin caracteres de control.
- Nombres naturales de máximo 100 y `NombreCompleto` derivado de máximo 255.
- `Numero` recortado, 1–50, sin `,`, `:`, espacios ni control; NIT solo dígitos.
- `DigitoVerificacion` obligatorio para NIT y entero 0–9.
- `TipoDocumentoId` existente y activo para natural; exactamente NIT para jurídica.
- `LugarOrigen` y `CiudadExpedicion` no negativos; si son mayores que cero, activos en `ubicaciones_crud`.
- Fechas RFC3339 no futuras.
- JSON de un único objeto, `application/json` y máximo 64 KB.

Los datos inválidos devuelven `400` mediante `ValidationError`. Los fallos de
catálogo devuelven `502/504` y no autorizan escrituras.

### Registros inactivos

Las consultas de identificación incluyen las inactivas. Si el tercero o la
identificación están inactivos, el flujo responde `409` y no reactiva ni crea
registros alternos. Un tercero sin ninguna identificación puede recibir una nueva
en `PUT`.

### Catálogos y NIT

`main.go` resuelve `NIT` con `MID_TIPO_DOCUMENTO_NIT` o el catálogo. El servicio
valida el tipo de documento y las ubicaciones con `CatalogosService`, que cachea
por Id y propaga los errores de dependencia.

### Concurrencia

`TerceroWriteService` usa un mutex por `variante + Numero + TipoDocumentoId`.
Esto evita duplicados dentro de una instancia. Tras un alta se consulta el
documento y, si hay varias identificaciones activas, se emite `duplicate_match`.
La solución definitiva para varias instancias requiere un índice único en
`terceros_crud`.

### Respuesta consistente

Las FK de respuesta se normalizan a `{ "Id": n }`. En un `206`, la identificación
solo aparece en `Creados` si tiene un `Id` real; un fallo parcial no informa una
entidad creada con `Id: 0`.

### TipoContribuyenteId

`main.go` resuelve `P_JURIDICA` con `MID_TIPO_CONTRIBUYENTE_JURIDICA` o, si no
está configurado, con `TercerosService.ResolverTipoContribuyente`. El ID se pasa
al servicio. Si no coincide con el payload, `400`; si el tercero existente tiene
otro tipo, `409`.

Se extrae un helper en `main.go` para resolver ambos tipos sin duplicar la lógica
de variable de entorno/catálogo.

### Controladores compartidos

Se extraen helpers a `controllers/tercero_comun.go`:

- `decodificarCuerpo`
- `leerTerceroID`
- `escribirRespuestaTercero`
- `responderErrorTercero`
- `dataDeResultado` y utilidades de mapas

El controlador natural delega en ellos sin cambiar su contrato. El controlador
jurídico agrega validación y proyección propias.

### Respuesta y estado HTTP

`Data` contiene `TerceroId`, `DatosBasicos`, `DatosIdentificacion`,
`DatosIdentificacionId` y `Resultado`.

Estados:

- `201` si hubo al menos una creación.
- `200` si solo hubo actualización/reutilización.
- `206` si el motor reporta materialización parcial.
- `400`, `404`, `409`, `500`, `502`, `504` según el error.

### utils_oas v2

Se mantiene `customerror` para errores y `APIResponse` con `Status` string para
éxito/parcial. El transporte usa `clients.HTTPClient`; no se introduce
`utils_oas/v2/request`.

## Risks / Trade-offs

- **Refactor del controlador natural:** se mitiga delegando en helpers y
  conservando mensajes/firmas; las pruebas existentes siguen pasando.
- **`P_JURIDICA` al arranque:** si no se resuelve, el MID no inicia. Se documenta
  `MID_TIPO_CONTRIBUYENTE_JURIDICA` y el error es explícito.
- **`RazonSocial` en actualización:** se actualiza en `PUT`. Si el negocio prefiere
  conservarla, solo cambia una rama de `construirCuerpo`.
- **Sin transacción distribuida:** un fallo después de una escritura se reporta
  con `206` y los IDs materializados.
- **DV para NIT:** queda opcional; una validación por tipo de documento requiere
  el catálogo y se difiere.

## Open Questions

Ninguna para el dominio 1 jurídico. Para el dominio 2 queda pendiente la poda de
la rama `Tercero`/`DatosIdentificacion` al invocar `ProveedorWriteService`.
