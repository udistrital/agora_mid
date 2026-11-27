# registro-proveedor-juridica Specification

## Purpose

Expone `POST /v1/proveedores/juridica` y `PUT /v1/proveedores/juridica/{terceroId}`
para registrar y actualizar el tercero persona jurídica y su identificación activa
en `terceros_crud`.

Este spec cubre únicamente el **dominio 1**: `tercero` y `datos_identificacion`.
No crea ni consulta `gestion_proveedores_crud`; el proveedor, la caracterización,
el representante legal, la cuenta bancaria y los documentos quedan en specs
posteriores.

## Alcance

| Incluido | Detalle |
|---|---|
| `POST /v1/proveedores/juridica` | Alta o continuación del tercero y su identificación. |
| `PUT /v1/proveedores/juridica/{terceroId}` | Actualización del tercero y su identificación. |
| `DatosBasicos.RazonSocial` | Campo propio de la persona jurídica; se persiste en `tercero.NombreCompleto`. |
| `DatosBasicos.LugarOrigen` | Opcional; se actualiza en el `PUT` cuando llega. |
| `TipoContribuyenteId` | Recibido en `DatosBasicos`, validado contra `P_JURIDICA`. |
| `NombreCompleto` | Derivado por el MID a partir de `RazonSocial`. |
| `DigitoVerificacion` | Obligatorio para NIT (entero 0–9); en otros tipos es opcional. |
| Envelope y errores | `Success`, `Status` string, `Message`, `Data`, `MetaData.Errors`; `customerror` y `APIResponse`. |
| Logs | Etapa fallida, causa técnica y número de documento enmascarado. |

| Excluido | Motivo |
|---|---|
| `gestion_proveedores_crud` | Dominio 2. |
| `ProveedorId`, `TipoProveedor`, `proveedor_juridico` | Dominio 2. |
| Representante legal y `representacion` | Dominio 2. |
| `info_complementaria_tercero` | Dominio 2. |
| `cuenta_bancaria` | Dominio 3. |
| Documentos | Los gestiona el MF. |
| Autenticación, ownership, estado | Fases posteriores. |
| `PATCH`, `DELETE` | No se usan. |

## Payload de referencia

### `POST /v1/proveedores/juridica`

```json
{
  "DatosBasicos": {
    "RazonSocial": "ACME S.A.S.",
    "LugarOrigen": 149,
    "TipoContribuyenteId": 2
  },
  "DatosIdentificacion": {
    "Numero": "900123456",
    "TipoDocumentoId": 7,
    "CiudadExpedicion": 149,
    "FechaExpedicion": "2015-01-01T00:00:00Z",
    "DigitoVerificacion": 7
  }
}
```

### `PUT /v1/proveedores/juridica/{terceroId}`

Mismo cuerpo. No lleva `TipoProveedor`, `TerceroId` ni `NombreCompleto` en el body.
El `PUT` es completo: `RazonSocial`, `TipoContribuyenteId`, `Numero` y
`TipoDocumentoId` siguen siendo obligatorios.

## Requirements

### Requirement: Contrato de los endpoints

El sistema SHALL exponer `POST /v1/proveedores/juridica` y
`PUT /v1/proveedores/juridica/{terceroId}`.

El controlador SHALL:

- Leer y decodificar el JSON rechazando campos desconocidos.
- Validar los campos obligatorios.
- Propagar el `context.Context` a la capa de servicios.
- Construir la respuesta con el envelope estándar del MID.
- Responder sin invocar a `terceros_crud` cuando la petición es inválida.

El `PUT` SHALL recibir `terceroId` como último segmento de la ruta. No se permiten
variables intermedias.

`DatosIdentificacion.Numero` SHALL tratarse como string durante todo el flujo,
conservando ceros iniciales. La grafía `DatosIdentificacion` SHALL conservarse en
entrada y respuesta.

#### Scenario: POST válido

- **WHEN** llega un `POST /v1/proveedores/juridica` con `DatosBasicos` y
  `DatosIdentificacion` válidos
- **THEN** el controlador valida la petición, propaga el contexto y responde con el
  envelope del MID

#### Scenario: PUT válido

- **WHEN** llega un `PUT /v1/proveedores/juridica/101` con un cuerpo válido
- **THEN** el controlador toma `101` como `TerceroId`, propaga el contexto y responde
  con el envelope del MID

#### Scenario: JSON inválido o campo desconocido

- **WHEN** el cuerpo no es JSON válido o incluye un campo no permitido
- **THEN** el controlador responde `400` sin invocar a `terceros_crud`

#### Scenario: Ruta con variable intermedia

- **WHEN** se intenta una ruta como `/v1/proveedores/juridica/101/identificacion`
- **THEN** la ruta no existe y no se ejecuta ningún servicio

### Requirement: Payload de entrada jurídico

`DatosBasicos` SHALL ser obligatorio y SHALL contener:

- `RazonSocial` string obligatorio y no vacío.
- `LugarOrigen` entero opcional.
- `TipoContribuyenteId` entero obligatorio.

`DatosIdentificacion` SHALL ser obligatorio y SHALL contener:

- `Numero` string obligatorio.
- `TipoDocumentoId` entero obligatorio.
- `CiudadExpedicion` entero opcional.
- `FechaExpedicion` fecha RFC3339 opcional.
- `DigitoVerificacion` numérico opcional.

El cuerpo NO SHALL aceptar:

- `PrimerNombre`, `SegundoNombre`, `PrimerApellido`, `SegundoApellido`.
- `FechaNacimiento`.
- `NombreCompleto`.
- `TipoProveedor`.
- `Activo`, `Id`, `TerceroId`.
- Campos de representación, caracterización, cuenta bancaria o documentos.

#### Scenario: Petición con campos obligatorios

- **WHEN** llegan `RazonSocial`, `TipoContribuyenteId`, `Numero` y `TipoDocumentoId`
- **THEN** la petición se considera válida y continúa el flujo

#### Scenario: Falta `RazonSocial`

- **WHEN** `DatosBasicos.RazonSocial` está ausente o solo contiene espacios
- **THEN** el MID responde `400` y no escribe

#### Scenario: Falta `TipoContribuyenteId`

- **WHEN** `DatosBasicos.TipoContribuyenteId` está ausente o es cero
- **THEN** el MID responde `400` y no escribe

#### Scenario: Falta `Numero` o `TipoDocumentoId`

- **WHEN** `DatosIdentificacion.Numero` está vacío o `TipoDocumentoId` es cero
- **THEN** el MID responde `400` y no escribe

#### Scenario: Nombres naturales en el cuerpo

- **WHEN** el cuerpo incluye `PrimerNombre`, `PrimerApellido` o `FechaNacimiento`
- **THEN** el MID responde `400` porque esos campos pertenecen a persona natural

#### Scenario: `TipoProveedor` en el cuerpo

- **WHEN** el cuerpo incluye `TipoProveedor`
- **THEN** el MID responde `400` porque ese campo pertenece al dominio 2

#### Scenario: Número con ceros iniciales

- **WHEN** `DatosIdentificacion.Numero` es `"00900123456"`
- **THEN** el valor viaja como string sin perder los ceros iniciales en consultas,
  escrituras y respuesta

### Requirement: Validaciones de formato, longitud y catálogos

Antes de leer o escribir, el MID SHALL validar:

`DatosBasicos.RazonSocial`:

- Normalizar espacios.
- Longitud entre 1 y 255 caracteres después de normalizar.
- Rechazar caracteres de control.

`DatosBasicos` natural:

- `PrimerNombre` y `PrimerApellido` obligatorios.
- Cada nombre o apellido con máximo 100 caracteres.
- `NombreCompleto` derivado con máximo 255 caracteres.
- `FechaNacimiento` RFC3339 y no futura.

`DatosIdentificacion.Numero`:

- Recortar espacios externos.
- Longitud entre 1 y 50 caracteres.
- Rechazar `,`, `:`, espacios internos y caracteres de control.
- Si el tipo es NIT, solo dígitos.

`DatosIdentificacion.TipoDocumentoId`:

- Debe existir y estar activo en `tipo_documento` para persona natural.
- Para persona jurídica debe ser exactamente `NIT`.

`DatosIdentificacion.DigitoVerificacion`:

- Obligatorio cuando el tipo es NIT.
- Entero entre 0 y 9.

Fechas:

- Solo RFC3339.
- No futuras.
- No se impone fecha mínima.

`LugarOrigen` y `CiudadExpedicion`:

- Si son negativos, `400`.
- Si son mayores que cero, deben existir y estar activos en `ubicaciones_crud`.
- Si son cero o ausentes, no se validan.

El cuerpo SHALL rechazar datos de cola, `Content-Type` distinto de
`application/json` y cuerpos mayores a 64 KB.

#### Scenario: Razón social vacía

- **WHEN** `RazonSocial` está ausente o solo contiene espacios
- **THEN** el MID responde `400`

#### Scenario: Razón social demasiado larga

- **WHEN** `RazonSocial` supera 255 caracteres después de normalizar
- **THEN** el MID responde `400` y no escribe

#### Scenario: Número con coma o dos puntos

- **WHEN** `Numero` contiene `,` o `:`
- **THEN** el MID responde `400` y no consulta ni escribe

#### Scenario: Número de NIT con letras

- **WHEN** el tipo es NIT y `Numero` contiene letras
- **THEN** el MID responde `400`

#### Scenario: NIT sin dígito de verificación

- **WHEN** el tipo es NIT y `DigitoVerificacion` está ausente
- **THEN** el MID responde `400`

#### Scenario: Dígito de verificación fuera de rango

- **WHEN** `DigitoVerificacion` es negativo, mayor que 9 o no entero
- **THEN** el MID responde `400`

#### Scenario: Tipo de documento inexistente

- **WHEN** `TipoDocumentoId` no existe o está inactivo en `tipo_documento`
- **THEN** el MID responde `400`

#### Scenario: Jurídica con tipo distinto de NIT

- **WHEN** `/juridica` recibe `TipoDocumentoId` distinto de `NIT`
- **THEN** el MID responde `400`

#### Scenario: Ubicación inexistente

- **WHEN** `LugarOrigen` o `CiudadExpedicion` es mayor que cero y no existe o
  está inactivo en `ubicaciones_crud`
- **THEN** el MID responde `400` y no escribe

#### Scenario: Fecha futura

- **WHEN** `FechaNacimiento` o `FechaExpedicion` es posterior al momento actual
- **THEN** el MID responde `400`

#### Scenario: JSON con datos de cola

- **WHEN** el cuerpo contiene dos objetos JSON
- **THEN** el MID responde `400`

#### Scenario: Content-Type inválido

- **WHEN** el `Content-Type` no es `application/json`
- **THEN** el MID responde `400`

#### Scenario: Cuerpo demasiado grande

- **WHEN** el cuerpo supera 64 KB
- **THEN** el MID responde `400` y no escribe

### Requirement: Registros inactivos

El MID SHALL consultar las identificaciones incluyendo las inactivas para
detectarlas antes de escribir.

- Si el tercero existente está inactivo, `POST` y `PUT` SHALL responder `409`.
- Si la identificación está inactiva, `POST` y `PUT` SHALL responder `409`.
- El MID NO SHALL reactivar automáticamente un tercero ni una identificación.
- Si el tercero no tiene ninguna identificación (ni activa ni inactiva), el `PUT`
  SHALL crear una.

#### Scenario: Tercero inactivo

- **WHEN** el documento resuelve a un tercero con `Activo: false`
- **THEN** el MID responde `409` con `"el tercero está inactivo..."`

#### Scenario: Identificación inactiva

- **WHEN** el documento o la identificación del tercero tiene `Activo: false`
- **THEN** el MID responde `409` con `"la identificación está inactiva..."`

#### Scenario: Tercero sin identificaciones

- **WHEN** el tercero existe y no tiene identificaciones
- **THEN** el `PUT` crea la identificación y responde `201`

### Requirement: Validación de `TipoContribuyenteId`

El `TipoContribuyenteId` SHALL ser obligatorio y SHALL corresponder a `P_JURIDICA`
para este endpoint.

El MID SHALL resolver el ID esperado de `P_JURIDICA` así:

1. Usar `MID_TIPO_CONTRIBUYENTE_JURIDICA` si está configurado.
2. Si no, consultar `terceros_crud`:
   ```text
   GET /v1/tipo_contribuyente?query=CodigoAbreviacion:P_JURIDICA,Activo:true
   ```
3. Tomar el primer registro no vacío y activo.
4. Cachear el ID esperado en memoria.

Si el ID recibido no coincide con el esperado, el MID SHALL responder `400` y no
SHALL escribir.

Si el ID esperado no se encuentra, el MID SHALL fallar antes de escribir con un
error de configuración/catálogo (`500`). Si la consulta al catálogo falla por
dependencia, SHALL responder `502` o `504`.

En actualización, si el `TipoContribuyenteId` del tercero existente no es
`P_JURIDICA`, el MID SHALL responder `409`.

#### Scenario: ID correcto

- **WHEN** el payload envía el ID de `P_JURIDICA`
- **THEN** el MID lo acepta y lo envía a `tercero` como `{ "Id": n }`

#### Scenario: ID de otra variante

- **WHEN** el payload envía el ID de `P_NATURAL` en `/juridica`
- **THEN** el MID responde `400` y no escribe

#### Scenario: ID inexistente en el catálogo

- **WHEN** no se encuentra `P_JURIDICA` activo en `tipo_contribuyente`
- **THEN** el MID responde `500` antes de escribir

#### Scenario: Catálogo no disponible

- **WHEN** la consulta a `tipo_contribuyente` falla
- **THEN** el MID responde `502` o `504` y no escribe

#### Scenario: Tercero existente de otra variante

- **WHEN** el tercero de la ruta tiene `TipoContribuyenteId = P_NATURAL`
- **THEN** el MID responde `409` y no modifica el tercero

### Requirement: Consulta de la identificación activa

En `POST`, la coordinación SHALL consultar `datos_identificacion` en
`terceros_crud` con:

- `Numero`
- `TipoDocumentoId`
- `Activo:true`
- `sortby=Id`
- `order=asc`

SHALL comprobar primero que la consulta tuvo éxito. Una consulta fallida es un
error y no autoriza crear registros.

SHALL interpretar `[]` y `[{}]` como ausencia de identificación activa. SHALL
considerar coincidencia solo cuando el primer elemento es un objeto con `Id`
válido. Ante varias coincidencias activas SHALL usar la primera y emitir un aviso
en `MetaData.Errors`.

#### Scenario: Consulta fallida

- **WHEN** la consulta de `datos_identificacion` falla
- **THEN** la coordinación detiene el flujo, no ejecuta ninguna escritura y responde
  `502` o `504`

#### Scenario: Lista vacía o con objeto vacío

- **WHEN** la respuesta es `[]` o `[{}]`
- **THEN** la coordinación trata el caso como ausencia de identificación activa y
  crea el tercero

#### Scenario: Una coincidencia activa

- **WHEN** la lista contiene un primer objeto con `Id` válido
- **THEN** la coordinación toma su `Id` y su `TerceroId.Id` para actualizar la
  identificación y el tercero

#### Scenario: Varias coincidencias activas

- **WHEN** la lista contiene más de un objeto válido
- **THEN** la coordinación usa el primero con `sortby=Id&order=asc`
- **AND** emite `MetaData.Errors` con `kind: "ambiguous_match"`

### Requirement: Alta de tercero e identificación jurídica

Cuando no exista identificación activa en `POST`, o cuando el `PUT` reciba un
`TerceroId` sin identificación activa, el MID SHALL crear el registro faltante.

Alta de `tercero`:

- `NombreCompleto` igual a `RazonSocial`, normalizada sin espacios duplicados.
- `LugarOrigen` si viene.
- `Activo: true`.
- `TipoContribuyenteId: { "Id": n }`.
- NO SHALL enviar `PrimerNombre`, `SegundoNombre`, `PrimerApellido`,
  `SegundoApellido` ni `FechaNacimiento`.

Alta de `datos_identificacion`:

- `TerceroId: { "Id": n }`.
- `TipoDocumentoId: { "Id": n }`.
- `Numero`.
- `DigitoVerificacion` si viene.
- `CiudadExpedicion` si viene.
- `FechaExpedicion` si viene.
- `Activo: true`.

La identificación SHALL crearse con el `Id` devuelto por el alta del tercero. No se
SHALL usar `Id = 0` ni deducir identidad por la razón social.

#### Scenario: Tercero e identificación nuevos

- **WHEN** no hay identificación activa y el alta del tercero responde con `Id = 101`
- **THEN** se crea la identificación con `TerceroId: { "Id": 101 }`
- **AND** `Data.TerceroId` es `101`
- **AND** `Data.DatosIdentificacionId` es el `Id` devuelto por la identificación
- **AND** la respuesta es `201`

#### Scenario: Razón social normalizada

- **WHEN** `RazonSocial` es `"  ACME   S.A.S.  "`
- **THEN** `tercero.NombreCompleto` es `"ACME S.A.S."`

#### Scenario: Falla el alta del tercero

- **WHEN** el `POST` del tercero falla
- **THEN** no se intenta crear la identificación
- **AND** la respuesta es `502` o `504`

#### Scenario: Falla la identificación después del tercero

- **WHEN** el tercero se creó correctamente y el `POST` de la identificación falla
- **THEN** la respuesta es `206`
- **AND** `Data.TerceroId` y `Data.DatosBasicos` contienen lo materializado
- **AND** `MetaData.Errors` describe el fallo de `DatosIdentificacion`

### Requirement: Actualización de tercero e identificación jurídica

En `POST` con identificación activa y en `PUT` con `TerceroId` existente, el MID
SHALL actualizar los registros existentes sin reemplazar los campos que no
llegaron.

`tercero`:

- En `PUT`, reemplazar `NombreCompleto` con `RazonSocial` normalizada.
- En `POST` que reutiliza un tercero existente, conservar `NombreCompleto`.
- En `PUT`, reemplazar `LugarOrigen` solo si viene.
- En `POST` que reutiliza un tercero existente, conservar `LugarOrigen`.
- Conservar `TipoContribuyenteId` y validarlo contra `P_JURIDICA` y el valor
  existente.
- Conservar `Activo`.
- Conservar los campos de nombre natural que existieran, sin enviarlos.

`datos_identificacion`:

- Conservar `Id`, `TerceroId`, `Activo` y `DocumentoSoporte`.
- Conservar `Numero` y `TipoDocumentoId`; si difieren de la identificación activa,
  responder `409`.
- Reemplazar `CiudadExpedicion`, `FechaExpedicion` y `DigitoVerificacion` solo
  cuando vienen.
- Ejecutar un único `PUT` por el `Id` de la identificación seleccionada.

En `PUT`:

- Si el `TerceroId` no existe, responder `404`.
- Buscar la identificación por `TerceroId + Activo:true`, con `sortby=Id&order=asc`.
- Si no hay identificación activa, crearla.
- Si hay varias, usar la primera y emitir aviso.
- Si el `Numero` o el `TipoDocumentoId` del payload difieren de la identificación
  activa, responder `409` y no escribir.
- Si el tercero existente no es `P_JURIDICA`, responder `409`.

#### Scenario: POST con identificación existente

- **WHEN** la identificación activa existe
- **THEN** se actualiza la identificación por su `Id`
- **AND** el tercero conserva su `NombreCompleto` y su `LugarOrigen`
- **AND** la respuesta es `200` si no hubo creaciones

#### Scenario: POST de reutilización con otra razón social

- **WHEN** el `POST` envía una `RazonSocial` distinta para un documento existente
- **THEN** el MID responde `200`
- **AND** conserva el `NombreCompleto` y el `LugarOrigen` almacenados
- **AND** los cambios de nombre o lugar se realizan únicamente con `PUT`

#### Scenario: PUT de tercero existente

- **WHEN** el `TerceroId` de la ruta existe y es `P_JURIDICA`
- **THEN** se actualizan `NombreCompleto`, `LugarOrigen` (si viene) y la
  identificación activa
- **AND** la respuesta es `200` si no hubo creaciones

#### Scenario: PUT sin `LugarOrigen`

- **WHEN** el `PUT` no envía `LugarOrigen`
- **THEN** el valor existente se conserva

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

La identificación activa de un tercero existente SHALL ser inmutable en `PUT`.
El MID NO SHALL cambiar `Numero` ni `TipoDocumentoId` de una identificación ya
asociada al tercero.

Antes de escribir, el MID SHALL comparar el `Numero` y el `TipoDocumentoId` del
payload con la identificación activa del tercero. Si alguno difiere, SHALL
responder `409` y no SHALL escribir.

Además, antes de crear o actualizar una identificación, el MID SHALL consultar
`datos_identificacion` por el nuevo `Numero + TipoDocumentoId + Activo:true`.

Si la consulta devuelve una identificación cuyo `TerceroId.Id` es distinto del
tercero que se está actualizando, el MID SHALL responder `409` y no SHALL mover ni
modificar la identificación de otro tercero.

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
    "DatosBasicos": {
      "Id": 101,
      "RazonSocial": "ACME S.A.S.",
      "LugarOrigen": 149,
      "TipoContribuyenteId": { "Id": 2 },
      "Activo": true
    },
    "DatosIdentificacion": { "Id": 140, "Numero": "900123456" },
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

`Data.DatosBasicos` SHALL ser la proyección jurídica del tercero:

- `Id`.
- `RazonSocial` derivada de `NombreCompleto`.
- `LugarOrigen` si tiene valor.
- `TipoContribuyenteId` como objeto `{ "Id": n }`.
- `Activo`.
- NO SHALL incluir `NombreCompleto`, `PrimerNombre`, `SegundoNombre`,
  `PrimerApellido`, `SegundoApellido` ni `FechaNacimiento` cuando estén vacíos.

Las FK de `DatosBasicos` y `DatosIdentificacion` SHALL exponerse siempre como
`{ "Id": n }`, incluso cuando el motor reutiliza un registro sin cambios.

`Data` SHALL contener además:

- `TerceroId`.
- `DatosIdentificacion` (respuesta efectiva de la identificación).
- `DatosIdentificacionId`.
- `Resultado` con `Creados`, `Reutilizados` y `Actualizados`.

`Status` SHALL ser string. `Success` SHALL ser `true` para `200/201` y `false` para
`206/4xx/5xx`. `MetaData.Errors` SHALL omitirse cuando no haya avisos.

El estado HTTP SHALL ser:

- `201` si se creó al menos un registro.
- `200` si solo hubo lectura, reutilización y actualización.
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
- **THEN** la respuesta es `206`, `Success` es `false` y `MetaData.Errors` describe
  el fallo
- **AND** `Resultado.Creados` solo incluye entidades con `Id` real
- **AND** `DatosIdentificacionId` es `0` si la identificación no se materializó

### Requirement: Formato de respuesta con `utils_oas` v2

El MID SHALL usar el estándar de `utils_oas` v2 para el armado de respuestas:

- `customerror.CustomErrorController` para errores.
- `APIResponse` con `Success`, `Status` string, `Message`, `Data` y `MetaData`.
- Helpers `WriteSuccess`, `WriteSuccessWithWarnings`, `WritePartial`,
  `WriteError` y `AbortWithError`.
- `apiStatusLib` para el health check.
- `security` para headers.

El transporte hacia `terceros_crud` SHALL usar `clients.HTTPClient`, que es el
cliente del motor base del proyecto. No se usa `utils_oas/v2/request` en este
dominio.

`main.go` SHALL registrar:

```go
apistatus.Init()
security.SetSecurityHeaders()
beego.ErrorController(&customerror.CustomErrorController{})
```

Los códigos que `customerror` no define (`206` y `504`) SHALL emitirse con el mismo
`APIResponse` desde un helper local.

#### Scenario: Respuesta exitosa

- **WHEN** la operación termina con `200` o `201`
- **THEN** el cuerpo tiene `Success`, `Status` string, `Message` y `Data`

#### Scenario: Respuesta de error

- **WHEN** la operación termina con `400`, `404`, `409`, `500` o `502`
- **THEN** el cuerpo tiene el formato de `customerror`: `Success: false`, `Status`
  string, `Message` y `Data`

#### Scenario: Respuesta parcial

- **WHEN** la operación termina con `206`
- **THEN** el cuerpo usa el mismo `APIResponse` con `Success: false`,
  `Status: "206"` y `MetaData.Errors`

### Requirement: Manejo de errores y logs

Ante el fallo de una operación necesaria, la coordinación SHALL:

- Registrar la etapa fallida y la causa técnica en logs.
- Mantener el host, puerto y URL interna de los CRUD fuera del error público.
- No repetir automáticamente una escritura confirmada.
- Enmascarar el número de documento en los logs.

#### Scenario: Fallo de dependencia

- **WHEN** una consulta o escritura a `terceros_crud` falla antes de materializar
  registros
- **THEN** la respuesta es `502` o `504`
- **AND** el mensaje público no expone el host del CRUD
- **AND** el log identifica la etapa

#### Scenario: Fallo después de una escritura

- **WHEN** una escritura falla después de otra exitosa
- **THEN** se devuelve `206` con lo materializado
- **AND** no se repite automáticamente la escritura exitosa

#### Scenario: Documento en logs

- **WHEN** se registra una identificación o una ambigüedad
- **THEN** el número de documento aparece enmascarado, dejando visibles como máximo
  los últimos cuatro caracteres

### Requirement: Integración con el proyecto base

La implementación SHALL reutilizar los servicios del proyecto base (`base_engine`)
en lugar de crear una segunda ruta de escritura.

Para este dominio:

- SHALL reutilizar `clients.HTTPClient` para el transporte HTTP.
- SHALL reutilizar el cargador de `config` y los YAML de objetos de `terceros`.
- SHALL reutilizar `helper.Write`, `WriteResolver` y `WriteResolverRegistry`.
- SHALL reutilizar `services.EntityWriteResolver` para `Tercero` y
  `DatosIdentificacion`.
- SHALL reutilizar `TerceroWriteService` con raíz `Tercero` y la relación
  `DatosIdentificacion`.
- NO SHALL usar `ProveedorWriteService` en este dominio, porque su raíz es
  `Proveedor` y crearía un proveedor que este endpoint no debe crear.
  `ProveedorWriteService` se reserva para el dominio de caracterización.

Reglas de implementación:

1. `TerceroWriteService` SHALL exponer `EscribirTercero` (natural) y
   `EscribirTerceroJuridico` (jurídica) sobre el mismo núcleo de identificación y
   clasificación.
2. `RazonSocial` SHALL mapearse a `tercero.NombreCompleto`; NO SHALL enviarse al
   CRUD como campo propio.
3. `RazonSocial` NO SHALL agregarse a `config/terceros/objetos_tercero.yaml`,
   porque `tercero` no tiene esa columna.
4. La actualización SHALL usar `PUT`; NO SHALL enviarse `PATCH`.
5. La configuración de `Tercero` y `DatosIdentificacion` SHALL reutilizarse sin
   cambios.
6. El resultado del motor (`WriteResult.Data` y `WriteResult.Warnings`) SHALL
   mapearse al `APIResponse`, con `MetaData.Errors` para los avisos.
7. El MID SHALL serializar las peticiones que compiten por el mismo
   `Numero + TipoDocumentoId` dentro de la instancia.
8. Tras un alta, el MID SHALL detectar identificaciones activas duplicadas y
   emitir un aviso `duplicate_match` sin borrar datos.
9. Las FK de respuesta SHALL normalizarse a `{ "Id": n }`.

#### Scenario: Dominio 1 no crea proveedor

- **WHEN** se ejecuta `POST /v1/proveedores/juridica`
- **THEN** el flujo usa el motor con raíz `Tercero`
- **AND** no invoca `ProveedorWriteService`
- **AND** no escribe en `gestion_proveedores_crud`

#### Scenario: Reutilización del motor base

- **WHEN** se implementa el dominio 1 jurídico
- **THEN** el servicio usa `clients.HTTPClient`, `config`,
  `EntityWriteResolver` y `helper.Write` del proyecto base
- **AND** no se duplica la lógica de upsert ni de propagación de llaves

#### Scenario: Actualización con PUT

- **WHEN** el motor actualiza un `tercero` o una `datos_identificacion` existente
- **THEN** la llamada HTTP usa `PUT`
- **AND** no se envía `PATCH`

#### Scenario: `RazonSocial` no se envía como columna

- **WHEN** el payload envía `DatosBasicos.RazonSocial`
- **THEN** el cuerpo enviado a `terceros_crud` contiene `NombreCompleto`
- **AND** no contiene la clave `RazonSocial`

#### Scenario: Avisos del motor

- **WHEN** el motor devuelve `WriteResult.Warnings`
- **THEN** el controlador responde `206` si corresponde
- **AND** los avisos se publican en `MetaData.Errors`

## Non-Goals

- No se implementa `gestion_proveedores_crud`.
- No se devuelve `ProveedorId`.
- No se acepta `TipoProveedor`.
- No se implementan representante legal, `representacion`,
  `info_complementaria_tercero`, `proveedor_juridico`, `perfil_proveedor`,
  `contacto`, `cuenta_bancaria` ni `documento`.
- No se implementan autenticación, ownership ni `estado_proveedor`.
- No se implementan `PATCH` ni `DELETE`.
- No se modifican los contratos ni los repositorios de `terceros_crud`.

## Notas

- `tercero` no tiene columna `RazonSocial`; el nombre legal se persiste en
  `NombreCompleto`.
- `DigitoVerificacion` es obligatorio para NIT y debe ser un entero entre 0 y 9.
- `TipoDocumentoId` se resuelve por catálogo; en jurídica debe ser `NIT`.
- `LugarOrigen` y `CiudadExpedicion` se validan contra `ubicaciones_crud` por
  existencia y `Activo`. No se exige un tipo de lugar específico porque los datos
  existentes usan IDs de país en `CiudadExpedicion`.
- Los terceros y las identificaciones inactivas responden `409`; no se reactivan
  automáticamente.
- `TerceroWriteService` se generaliza; no se crea un servicio jurídico paralelo.
- La configuración `config/terceros/*.yaml` se comparte con persona natural y no
  requiere cambios.
- `ProveedorWriteService` queda reservado para el dominio 2 (caracterización).
- El dominio 2 se especifica en
  `openspec/specs/caracterizacion-proveedor-juridica/spec.md`.
