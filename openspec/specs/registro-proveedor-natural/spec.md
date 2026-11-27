# registro-proveedor-natural Specification

## Purpose

Expone `POST /v1/proveedores/natural` y `PUT /v1/proveedores/natural/{terceroId}` para registrar y actualizar el tercero persona natural y su identificación activa en `terceros_crud`.

Este spec cubre únicamente el **dominio 1**: `tercero` y `datos_identificacion`. No crea ni consulta `gestion_proveedores_crud`; la caracterización, el proveedor, la cuenta bancaria y los documentos quedan en specs posteriores.

## Alcance

| Incluido | Detalle |
|---|---|
| `POST /v1/proveedores/natural` | Alta o continuación del tercero y su identificación. |
| `PUT /v1/proveedores/natural/{terceroId}` | Actualización del tercero y su identificación. |
| `TipoContribuyenteId` | Recibido en `DatosBasicos`, validado contra `P_NATURAL`. |
| `NombreCompleto` | Derivado por el MID. |
| `DigitoVerificacion` | Obligatorio para NIT (entero 0–9); opcional para otros tipos. |
| Envelope y errores | `Success`, `Status` string, `Message`, `Data`, `MetaData.Errors`; `customerror` y `utils_oas` v2. |
| Logs | Etapa fallida, causa técnica y número de documento enmascarado. |

| Excluido | Motivo |
|---|---|
| `gestion_proveedores_crud` | Se implementa en el spec de caracterización. |
| `ProveedorId`, `TipoProveedor`, `proveedor_natural` | Dominio 2. |
| `info_complementaria_tercero` | Dominio 2. |
| `cuenta_bancaria` | Dominio 3. |
| Documentos | Los gestiona el MF. |
| Autenticación, ownership, estado | Fases posteriores. |
| `PATCH`, `DELETE` | No se usan. |

## Payload de referencia

### `POST /v1/proveedores/natural`

```json
{
  "DatosBasicos": {
    "PrimerNombre": "Ana",
    "SegundoNombre": "",
    "PrimerApellido": "Pérez",
    "SegundoApellido": "",
    "FechaNacimiento": "1990-05-10T00:00:00Z",
    "LugarOrigen": 149,
    "TipoContribuyenteId": 1
  },
  "DatosIdentificacion": {
    "Numero": "1012345678",
    "TipoDocumentoId": 3,
    "CiudadExpedicion": 149,
    "FechaExpedicion": "2010-01-01T00:00:00Z",
    "DigitoVerificacion": 0
  }
}
```

### `PUT /v1/proveedores/natural/{terceroId}`

Mismo cuerpo. Puede ser parcial. No lleva `TipoProveedor` ni `TerceroId` en el body.

## Requirements

### Requirement: Contrato de los endpoints

El sistema SHALL exponer `POST /v1/proveedores/natural` y `PUT /v1/proveedores/natural/{terceroId}`.

El controlador SHALL:

- Leer y decodificar el JSON.
- Validar los campos obligatorios.
- Propagar el `context.Context` a la capa de servicios.
- Construir la respuesta con el envelope estándar del MID.
- Responder sin invocar a `terceros_crud` cuando la petición es inválida.

El `PUT` SHALL recibir `terceroId` como último segmento de la ruta. No se permiten variables intermedias.

`DatosIdentificacion.Numero` SHALL tratarse como string durante todo el flujo, conservando ceros iniciales. La grafía `DatosIdentificacion` SHALL conservarse en entrada y respuesta.

#### Scenario: POST válido

- **WHEN** llega un `POST /v1/proveedores/natural` con `DatosBasicos` y `DatosIdentificacion` válidos
- **THEN** el controlador valida la petición, propaga el contexto y responde con el envelope del MID

#### Scenario: PUT válido

- **WHEN** llega un `PUT /v1/proveedores/natural/101` con un cuerpo válido
- **THEN** el controlador toma `101` como `TerceroId`, propaga el contexto y responde con el envelope del MID

#### Scenario: JSON inválido

- **WHEN** el cuerpo no es un JSON válido
- **THEN** el controlador responde `400` sin invocar a `terceros_crud`

#### Scenario: Ruta con variable intermedia

- **WHEN** se intenta una ruta como `/v1/proveedores/natural/101/identificacion`
- **THEN** la ruta no existe y no se ejecuta ningún servicio

### Requirement: Payload de entrada

`DatosBasicos` SHALL ser obligatorio y SHALL contener:

- `PrimerNombre` string obligatorio.
- `SegundoNombre` string opcional.
- `PrimerApellido` string obligatorio.
- `SegundoApellido` string opcional.
- `FechaNacimiento` fecha RFC3339 opcional.
- `LugarOrigen` entero opcional.
- `TipoContribuyenteId` entero obligatorio.

`DatosIdentificacion` SHALL ser obligatorio y SHALL contener:

- `Numero` string obligatorio.
- `TipoDocumentoId` entero obligatorio.
- `CiudadExpedicion` entero opcional.
- `FechaExpedicion` fecha RFC3339 opcional.
- `DigitoVerificacion` numérico opcional.

El cuerpo NO SHALL aceptar:

- `TipoProveedor`.
- `NombreCompleto`.
- `Activo`.
- `Id`.
- `TerceroId`.
- Campos de caracterización, cuenta bancaria o documentos.

#### Scenario: Petición con campos obligatorios

- **WHEN** llegan `PrimerNombre`, `PrimerApellido`, `TipoContribuyenteId`, `Numero` y `TipoDocumentoId`
- **THEN** la petición se considera válida y continúa el flujo

#### Scenario: Falta `TipoContribuyenteId`

- **WHEN** `DatosBasicos.TipoContribuyenteId` está ausente o es cero
- **THEN** el MID responde `400` y no escribe

#### Scenario: Falta `Numero` o `TipoDocumentoId`

- **WHEN** `DatosIdentificacion.Numero` está vacío o `TipoDocumentoId` es cero
- **THEN** el MID responde `400` y no escribe

#### Scenario: `TipoProveedor` en el cuerpo

- **WHEN** el cuerpo incluye `TipoProveedor`
- **THEN** el MID responde `400` porque ese campo pertenece al dominio 2

#### Scenario: Número con ceros iniciales

- **WHEN** `DatosIdentificacion.Numero` es `"0023456789"`
- **THEN** el valor viaja como string sin perder los ceros iniciales en consultas, escrituras y respuesta

### Requirement: Validaciones de formato, longitud y catálogos

Antes de leer o escribir, el MID SHALL validar:

- `PrimerNombre` y `PrimerApellido` obligatorios.
- Cada nombre o apellido con máximo 100 caracteres.
- `NombreCompleto` derivado con máximo 255 caracteres.
- `FechaNacimiento` RFC3339 y no futura.
- `Numero` recortado, de 1 a 50 caracteres, sin `,`, `:`, espacios internos ni caracteres de control.
- Si el tipo es NIT, `Numero` solo dígitos y `DigitoVerificacion` obligatorio.
- `DigitoVerificacion` entero entre 0 y 9 cuando venga.
- `TipoDocumentoId` existente y activo en `tipo_documento`.
- `LugarOrigen` y `CiudadExpedicion` no negativos; si son mayores que cero, existentes y activos en `ubicaciones_crud`.
- `FechaExpedicion` RFC3339 y no futura.
- Rechazo de datos de cola, `Content-Type` distinto de `application/json` y cuerpos mayores a 64 KB.

#### Scenario: Nombre demasiado largo

- **WHEN** un nombre o apellido supera 100 caracteres o el `NombreCompleto` derivado supera 255
- **THEN** el MID responde `400` y no escribe

#### Scenario: Número con coma o dos puntos

- **WHEN** `Numero` contiene `,` o `:`
- **THEN** el MID responde `400` y no consulta ni escribe

#### Scenario: NIT sin dígito de verificación

- **WHEN** el tipo es NIT y `DigitoVerificacion` está ausente
- **THEN** el MID responde `400`

#### Scenario: Tipo de documento inexistente

- **WHEN** `TipoDocumentoId` no existe o está inactivo
- **THEN** el MID responde `400`

#### Scenario: Ubicación inexistente

- **WHEN** `LugarOrigen` o `CiudadExpedicion` es mayor que cero y no existe o está inactivo
- **THEN** el MID responde `400` y no escribe

#### Scenario: Fecha futura

- **WHEN** `FechaNacimiento` o `FechaExpedicion` es posterior al momento actual
- **THEN** el MID responde `400`

### Requirement: Registros inactivos

El MID SHALL consultar las identificaciones incluyendo las inactivas para detectarlas antes de escribir.

- Si el tercero existente está inactivo, `POST` y `PUT` SHALL responder `409`.
- Si la identificación está inactiva, `POST` y `PUT` SHALL responder `409`.
- El MID NO SHALL reactivar automáticamente un tercero ni una identificación.
- Si el tercero no tiene ninguna identificación (ni activa ni inactiva), el `PUT` SHALL crear una.

#### Scenario: Tercero inactivo

- **WHEN** el documento resuelve a un tercero con `Activo: false`
- **THEN** el MID responde `409`

#### Scenario: Identificación inactiva

- **WHEN** la identificación del tercero tiene `Activo: false`
- **THEN** el MID responde `409`

#### Scenario: Tercero sin identificaciones

- **WHEN** el tercero existe y no tiene identificaciones
- **THEN** el `PUT` crea la identificación y responde `201`

### Requirement: Validación de `TipoContribuyenteId`

El `TipoContribuyenteId` SHALL ser obligatorio y SHALL corresponder a `P_NATURAL` para este endpoint.

El MID SHALL resolver el ID esperado de `P_NATURAL` así:

1. Usar `MID_TIPO_CONTRIBUYENTE_NATURAL` si está configurado.
2. Si no, consultar `terceros_crud`:
   ```text
   GET /v1/tipo_contribuyente?query=CodigoAbreviacion:P_NATURAL,Activo:true&sortby=Id&order=asc
   ```
3. Tomar el primer registro no vacío.
4. Cachear el ID esperado en memoria.

Si el ID recibido no coincide con el esperado, el MID SHALL responder `400` y no SHALL escribir.

Si el ID esperado no se encuentra, el MID SHALL fallar antes de escribir con un error de configuración/catálogo (`500`). Si la consulta al catálogo falla por dependencia, SHALL responder `502` o `504`.

En actualización, si el `TipoContribuyenteId` recibido no coincide con el del tercero existente, el MID SHALL responder `409`.

#### Scenario: ID correcto

- **WHEN** el payload envía el ID de `P_NATURAL`
- **THEN** el MID lo acepta y lo envía a `tercero` como `{ "Id": n }`

#### Scenario: ID de otra variante

- **WHEN** el payload envía el ID de `P_JURIDICA` en `/natural`
- **THEN** el MID responde `400` y no escribe

#### Scenario: ID inexistente en el catálogo

- **WHEN** no se encuentra `P_NATURAL` activo en `tipo_contribuyente`
- **THEN** el MID responde `500` antes de escribir

#### Scenario: Catálogo no disponible

- **WHEN** la consulta a `tipo_contribuyente` falla
- **THEN** el MID responde `502` o `504` y no escribe

#### Scenario: Tipo distinto al existente

- **WHEN** el tercero existente tiene `P_NATURAL` y el payload envía otro `TipoContribuyenteId`
- **THEN** el MID responde `409` y no modifica el tercero

### Requirement: Consulta de la identificación activa

En `POST`, la coordinación SHALL consultar `datos_identificacion` en `terceros_crud` con:

- `Numero`
- `TipoDocumentoId`
- `Activo:true`
- `sortby=Id`
- `order=asc`

SHALL comprobar primero que la consulta tuvo éxito. Una consulta fallida es un error y no autoriza crear registros.

SHALL interpretar `[]` y `[{}]` como ausencia de identificación activa. SHALL considerar coincidencia solo cuando el primer elemento es un objeto no vacío. Ante varias coincidencias activas SHALL usar la primera y emitir un aviso en `MetaData.Errors`.

#### Scenario: Consulta fallida

- **WHEN** la consulta de `datos_identificacion` devuelve `ok = false`
- **THEN** la coordinación detiene el flujo, no ejecuta ninguna escritura y responde `502` o `504`

#### Scenario: Lista vacía o con objeto vacío

- **WHEN** la respuesta es `[]` o `[{}]`
- **THEN** la coordinación trata el caso como ausencia de identificación activa y crea el tercero

#### Scenario: Una coincidencia activa

- **WHEN** la lista contiene un primer objeto no vacío
- **THEN** la coordinación toma su `Id` y su `TerceroId.Id` para actualizar la identificación y el tercero

#### Scenario: Varias coincidencias activas

- **WHEN** la lista contiene más de un objeto no vacío
- **THEN** la coordinación usa el primero con `sortby=Id&order=asc`
- **AND** emite `MetaData.Errors` con `kind: "ambiguous_match"`
- **AND** enmascara el número de documento en los logs

### Requirement: Alta de tercero e identificación

Cuando no exista identificación activa en `POST`, o cuando el `PUT` reciba un `TerceroId` sin identificación activa, el MID SHALL crear el registro faltante.

Alta de `tercero`:

- `NombreCompleto` derivado de nombres y apellidos, sin espacios duplicados.
- `PrimerNombre`, `SegundoNombre`, `PrimerApellido`, `SegundoApellido`.
- `FechaNacimiento` si viene.
- `LugarOrigen` si viene.
- `Activo: true`.
- `TipoContribuyenteId: { "Id": n }`.

Alta de `datos_identificacion`:

- `TerceroId: { "Id": n }`.
- `TipoDocumentoId: { "Id": n }`.
- `Numero`.
- `DigitoVerificacion` si viene.
- `CiudadExpedicion` si viene.
- `FechaExpedicion` si viene.
- `Activo: true`.

La identificación SHALL crearse con el `Id` devuelto por el alta del tercero. No se SHALL usar `Id = 0` ni deducir identidad por nombres.

#### Scenario: Tercero e identificación nuevos

- **WHEN** no hay identificación activa y el alta del tercero responde con `Id = 101`
- **THEN** se crea la identificación con `TerceroId: { "Id": 101 }`
- **AND** `Data.TerceroId` es `101`
- **AND** `Data.DatosIdentificacionId` es el `Id` devuelto por la identificación
- **AND** la respuesta es `201`

#### Scenario: Falla el alta del tercero

- **WHEN** el `POST` del tercero falla
- **THEN** no se intenta crear la identificación
- **AND** la respuesta es `502` o `504`

#### Scenario: Falla la identificación después del tercero

- **WHEN** el tercero se creó correctamente y el `POST` de la identificación falla
- **THEN** la respuesta es `206`
- **AND** `Data.TerceroId` y `Data.DatosBasicos` contienen lo materializado
- **AND** `MetaData.Errors` describe el fallo de `DatosIdentificacion`

### Requirement: Actualización de tercero e identificación

En `POST` con identificación activa y en `PUT` con `TerceroId` existente, el MID SHALL actualizar los registros existentes sin reemplazar los campos que no llegaron.

`tercero`:

- Conservar `PrimerNombre`, `SegundoNombre`, `PrimerApellido`, `SegundoApellido` y `NombreCompleto` del registro recuperado.
- Reemplazar `FechaNacimiento` solo si viene.
- Conservar `LugarOrigen` en actualización.
- Conservar `TipoContribuyenteId` y validarlo contra la ruta y el valor existente.
- Conservar `Activo`.

`datos_identificacion`:

- Conservar `Id`, `TerceroId`, `Activo` y `DocumentoSoporte`.
- Conservar `Numero` y `TipoDocumentoId`; si difieren de la identificación activa, responder `409`.
- Reemplazar `CiudadExpedicion`, `FechaExpedicion` y `DigitoVerificacion` solo cuando vienen.
- Ejecutar un único `PUT` por el `Id` de la identificación seleccionada.

En `PUT`:

- Si el `TerceroId` no existe, responder `404`.
- Buscar la identificación por `TerceroId + Activo:true`, con `sortby=Id&order=asc`.
- Si no hay identificación activa, crearla.
- Si hay varias, usar la primera y emitir aviso.
- Si el `Numero` o el `TipoDocumentoId` del payload difieren de la identificación activa, responder `409` y no escribir.
- Si el `TerceroId` del cuerpo no coincide con el de la ruta, responder `400`.

#### Scenario: POST con identificación existente

- **WHEN** la identificación activa existe
- **THEN** se actualiza la identificación por su `Id`
- **AND** se actualiza el tercero conservando nombres y reemplazando solo `FechaNacimiento` si viene
- **AND** la respuesta es `200` si no hubo creaciones

#### Scenario: PUT de tercero existente

- **WHEN** el `TerceroId` de la ruta existe
- **THEN** se actualizan el tercero y su identificación activa
- **AND** la respuesta es `200` si no hubo creaciones

#### Scenario: PUT de tercero inexistente

- **WHEN** el `TerceroId` de la ruta no existe
- **THEN** la respuesta es `404` y no se escribe

#### Scenario: PUT sin identificación activa

- **WHEN** el tercero existe pero no tiene identificación activa
- **THEN** se crea la identificación para ese tercero
- **AND** la respuesta es `201`

#### Scenario: Varias identificaciones del tercero

- **WHEN** el tercero tiene más de una identificación activa
- **THEN** se usa la primera con `sortby=Id&order=asc`
- **AND** se emite `MetaData.Errors` con `kind: "ambiguous_match"`

### Requirement: Identidad inmutable y no reasignación entre terceros

La identificación activa de un tercero existente SHALL ser inmutable en `PUT`. El MID NO SHALL cambiar `Numero` ni `TipoDocumentoId` de una identificación ya asociada al tercero.

Antes de escribir, el MID SHALL comparar el `Numero` y el `TipoDocumentoId` del payload con la identificación activa del tercero. Si alguno difiere, SHALL responder `409` y no SHALL escribir.

Además, antes de crear o actualizar una identificación, el MID SHALL consultar `datos_identificacion` por el nuevo `Numero + TipoDocumentoId + Activo:true`.

Si la consulta devuelve una identificación cuyo `TerceroId.Id` es distinto del tercero que se está actualizando, el MID SHALL responder `409` y no SHALL mover ni modificar la identificación de otro tercero.

#### Scenario: Número ya usado por otro tercero

- **WHEN** el nuevo `Numero + TipoDocumentoId` pertenece a otro `TerceroId`
- **THEN** el MID responde `409`
- **AND** no modifica la identificación original ni la del otro tercero

#### Scenario: Número distinto y libre

- **WHEN** el `PUT` envía un `Numero` distinto al de la identificación activa
- **THEN** el MID responde `409` con `"el número y el tipo de documento no se pueden cambiar en la actualización"`
- **AND** no modifica la identificación del tercero

#### Scenario: Tipo de documento distinto

- **WHEN** el `PUT` envía el mismo `Numero` pero otro `TipoDocumentoId`
- **THEN** el MID responde `409`
- **AND** no modifica la identificación del tercero

#### Scenario: Misma identidad

- **WHEN** el `PUT` envía el mismo `Numero` y el mismo `TipoDocumentoId`
- **THEN** el MID actualiza los demás campos permitidos de la identificación
- **AND** conserva `Numero` y `TipoDocumentoId`

### Requirement: Composición de la respuesta y estado HTTP

La respuesta SHALL usar el envelope:

```json
{
  "Success": true,
  "Status": "201",
  "Message": "Tercero e identificación creados",
  "Data": {
    "TerceroId": 101,
    "DatosBasicos": { "Id": 101, "PrimerNombre": "Ana" },
    "DatosIdentificacion": { "Id": 140, "Numero": "1012345678" },
    "DatosIdentificacionId": 140,
    "Resultado": {
      "Creados": [
        { "Entidad": "Tercero", "Id": 101 },
        { "Entidad": "DatosIdentificacion", "Id": 140 }
      ],
      "Reutilizados": [],
      "Actualizados": []
    }
  }
}
```

`Data` SHALL contener:

- `TerceroId`.
- `DatosBasicos` (respuesta efectiva del tercero).
- `DatosIdentificacion` (respuesta efectiva de la identificación).
- `DatosIdentificacionId`.
- `Resultado` con `Creados`, `Reutilizados` y `Actualizados`.

`Status` SHALL ser string. `Success` SHALL ser `true` para `200/201` y `false` para `206/4xx/5xx`. `MetaData.Errors` SHALL omitirse cuando no haya avisos.

El estado HTTP SHALL ser:

- `201` si se creó al menos un registro.
- `200` si solo hubo lectura y actualización.
- `206` si hubo un fallo parcial después de materializar al menos un registro.
- `400` por validación.
- `404` por `PUT` con `TerceroId` inexistente.
- `409` por conflicto de tipo o de identificación.
- `500` por configuración/catálogo.
- `502` por fallo de dependencia antes de escribir.
- `504` por timeout antes de escribir.

#### Scenario: Todo nuevo

- **WHEN** se crean tercero e identificación
- **THEN** la respuesta es `201` con `Resultado.Creados` completo

#### Scenario: Todo existente

- **WHEN** el tercero y la identificación existen y solo se actualizan
- **THEN** la respuesta es `200` con `Resultado.Actualizados`

#### Scenario: Fallo parcial

- **WHEN** se crea el tercero y falla la identificación
- **THEN** la respuesta es `206`, `Success` es `false` y `MetaData.Errors` describe el fallo
- **AND** `Resultado.Creados` solo incluye entidades con `Id` real
- **AND** `DatosIdentificacionId` es `0` si la identificación no se materializó

### Requirement: Formato de respuesta con `utils_oas` v2

El MID SHALL usar el estándar de `utils_oas` v2 para el armado de respuestas:

- `customerror.CustomErrorController` para errores.
- `APIResponse` con `Success`, `Status` string, `Message`, `Data` y `MetaData`.
- Helpers `writeSuccess`, `writePartial` y `abortWithError`.
- `apiStatusLib` para el health check.
- `security` para headers.
- `request` para las llamadas HTTP a `terceros_crud`.

`main.go` SHALL registrar:

```go
apistatus.Init()
security.SetSecurityHeaders()
beego.ErrorController(&customerror.CustomErrorController{})
```

Los códigos que `customerror` no define (`206` y `504`) SHALL emitirse con el mismo `APIResponse` desde un helper local.

#### Scenario: Respuesta exitosa

- **WHEN** la operación termina con `200` o `201`
- **THEN** el cuerpo tiene `Success`, `Status` string, `Message` y `Data`

#### Scenario: Respuesta de error

- **WHEN** la operación termina con `400`, `404`, `409`, `500` o `502`
- **THEN** el cuerpo tiene el formato de `customerror`: `Success: false`, `Status` string, `Message` y `Data`

#### Scenario: Respuesta parcial

- **WHEN** la operación termina con `206`
- **THEN** el cuerpo usa el mismo `APIResponse` con `Success: false`, `Status: "206"` y `MetaData.Errors`

### Requirement: Manejo de errores y logs

Ante el fallo de una operación necesaria, la coordinación SHALL:

- Registrar la etapa fallida y la causa técnica en logs.
- Mantener el host, puerto y URL interna de los CRUD fuera del error público.
- No repetir automáticamente una escritura confirmada.
- Enmascarar el número de documento en los logs.

#### Scenario: Fallo de dependencia

- **WHEN** una consulta o escritura a `terceros_crud` falla antes de materializar registros
- **THEN** la respuesta es `502` o `504`
- **AND** el mensaje público no expone el host del CRUD
- **AND** el log identifica la etapa

#### Scenario: Fallo después de una escritura

- **WHEN** una escritura falla después de otra exitosa
- **THEN** se devuelve `206` con lo materializado
- **AND** no se repite automáticamente la escritura exitosa

#### Scenario: Documento en logs

- **WHEN** se registra una identificación o una ambigüedad
- **THEN** el número de documento aparece enmascarado, dejando visibles como máximo los últimos cuatro caracteres

### Requirement: Integración con el proyecto base

La implementación SHALL reutilizar los servicios del proyecto base (`base_engine`) en lugar de crear una segunda ruta de escritura.

Para este dominio:

- SHALL reutilizar `clients.HTTPClient` para el transporte HTTP.
- SHALL reutilizar el cargador de `config` y los YAML de objetos de `terceros`.
- SHALL reutilizar `helper.Write`, `WriteResolver` y `WriteResolverRegistry`.
- SHALL reutilizar `services.EntityWriteResolver` para `Tercero` y `DatosIdentificacion`.
- SHALL crear un `TerceroWriteService` con raíz `Tercero` y la relación `DatosIdentificacion`, análogo a `ProveedorWriteService` pero sin raíz `Proveedor`.
- NO SHALL usar `ProveedorWriteService` en este dominio, porque su raíz es `Proveedor` y crearía un proveedor que este endpoint no debe crear. `ProveedorWriteService` se reserva para el dominio de caracterización.

Ajustes obligatorios del proyecto base para este dominio:

1. El `EntityWriteResolver` SHALL actualizar con `PUT`, no con `PATCH`, porque los CRUD de la OAS no exponen `PATCH` y el contrato del MID prohíbe ese verbo.
2. El campo `DigitoVerficacion` de `config/terceros/objetos_datos_identificacion.yaml` SHALL corregirse a `DigitoVerificacion`.
3. El `TipoContribuyenteId` plano del payload SHALL mapearse a la forma anidada que espera el motor (`TipoContribuyente: { "Id": n }`) o la configuración SHALL aceptar el campo plano.
4. El resultado del motor (`WriteResult.Data` y `WriteResult.Warnings`) SHALL mapearse al `APIResponse` de `utils_oas` v2, con `MetaData.Errors` para los avisos.
5. La raíz `Tercero` SHALL resolverse por `Id` en `PUT` y por `Numero + TipoDocumentoId + Activo:true` en `POST`.
6. La identificación SHALL resolverse por `Numero + TipoDocumentoId + Activo:true`, con `sortby=Id&order=asc`.

#### Scenario: Dominio 1 no crea proveedor

- **WHEN** se ejecuta `POST /v1/proveedores/natural`
- **THEN** el flujo usa el motor con raíz `Tercero`
- **AND** no invoca `ProveedorWriteService`
- **AND** no escribe en `gestion_proveedores_crud`

#### Scenario: Reutilización del motor base

- **WHEN** se implementa el dominio 1
- **THEN** el servicio usa `clients.HTTPClient`, `config`, `EntityWriteResolver` y `helper.Write` del proyecto base
- **AND** no se duplica la lógica de upsert ni de propagación de llaves

#### Scenario: Actualización con PUT

- **WHEN** el motor actualiza un `tercero` o una `datos_identificacion` existente
- **THEN** la llamada HTTP usa `PUT`
- **AND** no se envía `PATCH`

#### Scenario: `DigitoVerificacion`

- **WHEN** el payload envía `DatosIdentificacion.DigitoVerificacion`
- **THEN** el campo se escribe en la columna `digito_verificacion`
- **AND** la configuración no usa la grafía incorrecta `DigitoVerficacion`

#### Scenario: Avisos del motor

- **WHEN** el motor devuelve `WriteResult.Warnings`
- **THEN** el controlador responde `206` si corresponde
- **AND** los avisos se publican en `MetaData.Errors`

## Non-Goals

- No se implementa `gestion_proveedores_crud`.
- No se devuelve `ProveedorId`.
- No se acepta `TipoProveedor`.
- No se implementan `info_complementaria_tercero`, `proveedor_natural`, `perfil_proveedor`, `contacto`, `cuenta_bancaria` ni `documento`.
- No se implementan autenticación, ownership ni `estado_proveedor`.
- No se implementan `PATCH` ni `DELETE`.
- No se modifican los contratos de `terceros_crud`.

## Notas

- La implementación reutiliza el `base_engine` mediante un `TerceroWriteService` con raíz `Tercero`.
- `ProveedorWriteService` queda reservado para el dominio 2 (caracterización).
- El dominio 2 se especifica en `openspec/specs/caracterizacion-proveedor-natural/spec.md`.
- El spec reemplaza el alcance anterior de `registro-proveedor-natural`, que incluía la creación del proveedor.
- `DigitoVerificacion` es obligatorio para NIT y debe ser un entero entre 0 y 9.
- `TipoDocumentoId` se valida contra `tipo_documento`; no se hardcodean IDs.
- `LugarOrigen` y `CiudadExpedicion` se validan contra `ubicaciones_crud` por existencia y `Activo`; no se exige un tipo de lugar específico porque los datos existentes usan IDs de país en `CiudadExpedicion`.
- Los terceros y las identificaciones inactivas responden `409`; no se reactivan automáticamente.
- El MID serializa las operaciones por `Numero + TipoDocumentoId` dentro de la instancia y emite un aviso `duplicate_match` si detecta duplicados.
- Las FK de respuesta se normalizan a `{ "Id": n }`.
