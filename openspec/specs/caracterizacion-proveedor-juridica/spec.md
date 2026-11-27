# caracterizacion-proveedor-juridica Specification

## Purpose

Expone los endpoints del **dominio 2** para registrar y actualizar parcialmente
la caracterización de una persona jurídica:

```text
POST /v1/proveedores/juridica/caracterizacion
PUT  /v1/proveedores/juridica/caracterizacion/{terceroId}
PUT  /v1/proveedores/juridica/caracterizacion/proveedor/{proveedorId}
```

El spec complementa `registro-proveedor-juridica` (dominio 1). No crea el
tercero ni su identificación: los recibe ya materializados en `terceros_crud`.
Tampoco administra la cuenta bancaria (dominio 3), el capital, el correo de
pagos ni los documentos (gestor documental).

## Alcance

| Incluido | Detalle |
|---|---|
| `POST /v1/proveedores/juridica/caracterizacion` | Crea o reutiliza el proveedor y escribe las secciones enviadas. |
| `PUT /v1/proveedores/juridica/caracterizacion/{terceroId}` | Actualiza por tercero; exige proveedor existente. |
| `PUT /v1/proveedores/juridica/caracterizacion/proveedor/{proveedorId}` | Actualiza por proveedor; evita resolver por tercero. |
| `Representante` y `representacion` | Representante legal existente, natural y con proveedor natural activo. |
| `ProveedorJuridico` | Columnas de `proveedor_juridico`, incluidas procedencia, tipo legal y régimen. |
| `InfoComplementariaTercero` | Caracterización por `(código, grupo)`; campos sin columna propia. |
| `PerfilProveedor` | Opcional; el formulario jurídico actual no lo envía. |
| `Contacto` | Correo de notificaciones, teléfono y asesor comercial. |
| `ActividadEconomicaProveedor` | Actividades CIIU. |
| `DescripcionPortafolio` y `Consentimiento` | Campos de `proveedor`. |
| Dirección DIAN | Objeto `DIRECCIÓN` y coherencia con la ciudad de origen. |
| Actualización parcial | Solo escribe lo enviado; no borra lo omitido. |
| Fallo parcial | `206` con `MetaData.Errors` e Ids ya materializados. |
| Catálogo `PROVEEDOR_JURIDICO` | Grupo y filas creadas en `terceros_crud` por API. |

| Excluido | Motivo |
|---|---|
| `tercero` y `datos_identificacion` | Dominio 1 (`registro-proveedor-juridica`). |
| Creación del representante | Debe existir por dominio 1 natural y tener proveedor natural activo. |
| `cuenta_bancaria`, capital y correo de pagos | Dominio 3. |
| Documentos (RUT, RUP, renta) | Gestor documental / MF. |
| Autenticación, ownership y estado del proveedor | Fases posteriores. |
| `PATCH` y `DELETE` | No se usan. |
| `NombreComercial`, `MatriculaMercantil`, `FechaConstitucion`, `FechaRenovacion`, `ReportaBeneficiosFinales`, `CotizaBolsa`, `RequiereRevisorFiscal` | No existen en el formulario actual ni en `proveedor_juridico`. |
| Swagger / OpenAPI | Se deja para una etapa posterior. |

## Payload de referencia

```json
{
  "TerceroId": 320522,
  "Representante": {
    "CodigoTipoDocumento": "CC",
    "Numero": "1099887001",
    "CodigoCargo": "PROFESIONAL"
  },
  "InfoComplementariaTercero": [
    {
      "CodigoInfoComplementaria": "TIPO_CONF_CUENTA",
      "CodigoGrupoInfoComplementaria": "PROVEEDOR_JURIDICO",
      "Dato": "INDIVIDUAL_PRINCIPAL"
    },
    {
      "CodigoInfoComplementaria": "CALIDAD_GRAN_CONTRIB",
      "CodigoGrupoInfoComplementaria": "PROVEEDOR_JURIDICO",
      "Dato": "NO"
    },
    {
      "CodigoInfoComplementaria": "SITIO_WEB",
      "CodigoGrupoInfoComplementaria": "PROVEEDOR_JURIDICO",
      "Dato": "https://acme-juridica.example"
    },
    {
      "CodigoInfoComplementaria": "ACTIVIDAD_PRINCIPAL",
      "CodigoGrupoInfoComplementaria": "PROVEEDOR_JURIDICO",
      "Dato": 7778
    },
    {
      "CodigoInfoComplementaria": "DIRECCIÓN",
      "CodigoGrupoInfoComplementaria": "Grupo_10",
      "Dato": {
        "PaisId": 48,
        "DepartamentoId": 260,
        "CiudadId": 824,
        "ViaId": 128058,
        "Via": "CALLE",
        "DetalleVia": "10",
        "NumeroCruce": "20",
        "PlacaPuerta": "30",
        "InteriorId": null,
        "Interior": "",
        "DetalleInterior": "",
        "Direccion": "CALLE 10 # 20 - 30"
      }
    }
  ],
  "ProveedorJuridico": {
    "CodigoProcedencia": "NACIONAL",
    "CiudadOrigenId": 824,
    "PaisOrigenId": 48,
    "CodigoTipoConformacion": "SAS",
    "CodigoRegimenContributivo": "COMUN_ORDINARIO",
    "SujetoRetencion": true,
    "AgenteRetenedor": false,
    "ResponsableIca": true,
    "ResponsableIva": true,
    "Pyme": true,
    "RegistroMercantil": true,
    "ExclusividadProducto": false
  },
  "Contacto": [
    {
      "TipoCanalContactoId": 8419,
      "Proposito": "NOTIFICACIONES",
      "ValorEspecifico": "notificaciones@acme-juridica.co",
      "NombreContacto": "ACME JURIDICA SAS"
    },
    {
      "TipoCanalContactoId": 8421,
      "Proposito": "COMERCIAL",
      "ValorEspecifico": "3186749920",
      "Extension": "104"
    },
    {
      "TipoCanalContactoId": 8421,
      "Proposito": "ASESOR",
      "ValorEspecifico": "3186749921",
      "NombreContacto": "Carlos Rodriguez"
    }
  ],
  "ActividadEconomicaProveedor": [
    { "ActividadEconomicaId": 7778 }
  ],
  "DescripcionPortafolio": "Servicios de consultoría y desarrollo de software",
  "Consentimiento": true
}
```

`CARGO_REPRESENTANTE` no se envía en `InfoComplementariaTercero`: el MID lo
genera a partir de `Representante.CargoId` o `CodigoCargo`.

## Requirements

### Requirement: Contrato de los endpoints

El sistema SHALL exponer `POST /v1/proveedores/juridica/caracterizacion`,
`PUT /v1/proveedores/juridica/caracterizacion/{terceroId}` y
`PUT /v1/proveedores/juridica/caracterizacion/proveedor/{proveedorId}`.

El controlador SHALL:

- Leer y decodificar el JSON rechazando campos desconocidos.
- Validar las secciones enviadas.
- Propagar el `context.Context` a la capa de servicios.
- Construir la respuesta con el envelope `utils_oas` v2:
  `Success`, `Status` string, `Message`, `Data` y `MetaData.Errors`.
- No invocar dependencias externas cuando la petición es inválida.

El parámetro de ruta SHALL ir al final. `POST` SHALL crear el proveedor si no
existe y reutilizarlo si existe. `PUT` SHALL responder `404` si el proveedor no
existe.

En el `PUT` por `proveedorId`, `TerceroId` SHALL ser opcional; si llega, SHALL
coincidir con el tercero del proveedor.

#### Scenario: POST válido

- **WHEN** llega un `POST` con `TerceroId` y al menos una sección válida
- **THEN** el MID materializa las secciones y responde `201` si creó al menos
  una entidad
- **AND** responde `200` si solo actualizó o reutilizó
- **AND** devuelve `TerceroId`, `ProveedorId` y las entidades en `Data`

#### Scenario: PUT sin proveedor

- **WHEN** llega un `PUT` para un tercero o proveedor que no existe
- **THEN** el MID responde `404`
- **AND** no crea el proveedor

#### Scenario: Ruta con parámetro al final

- **WHEN** se registra la ruta del `PUT`
- **THEN** el identificador viaja como último segmento
- **AND** no se aceptan variables intermedias

### Requirement: Resolución y tipo del proveedor

El MID SHALL resolver el proveedor por `TerceroId` o por `ProveedorId` y SHALL
exigir que sea de tipo jurídico (`MID_TIPO_PROVEEDOR_JURIDICO`, por defecto `2`).

El MID SHALL responder `409` cuando:

- El proveedor existente es de otro tipo.
- El proveedor está inactivo.
- Hay más de un proveedor activo para el tercero.

#### Scenario: Proveedor reutilizado

- **WHEN** el tercero ya tiene un proveedor jurídico activo
- **THEN** el `POST` reutiliza ese proveedor
- **AND** no crea un segundo registro

#### Scenario: Tipo incorrecto

- **WHEN** el proveedor del tercero no es jurídico
- **THEN** el MID responde `409`

#### Scenario: Proveedores duplicados

- **WHEN** hay más de un proveedor activo para el tercero
- **THEN** el MID responde `409`
- **AND** no elige uno de forma arbitraria

### Requirement: Representante legal existente

El `Representante` SHALL identificarse por `RepresentanteId` o por
(`TipoDocumentoId` o `CodigoTipoDocumento`) + `Numero`. `PA` SHALL aceptarse como
alias de `PAS`.

El MID SHALL resolver un `tercero` activo y SHALL exigir que su
`TipoContribuyenteId` sea `P_NATURAL` (`MID_TIPO_CONTRIBUYENTE_NATURAL`).

El MID SHALL exigir que el representante tenga un `proveedor` activo de tipo
natural y un `proveedor_natural` activo. El MID no crea al representante ni su
proveedor natural.

Si el representante no existe, el MID SHALL responder `400` indicando que se
registre por `/v1/proveedores/natural`. Si existe pero no es natural o no tiene
proveedor natural activo, SHALL responder `409`.

#### Scenario: Representante por documento

- **WHEN** llega `CodigoTipoDocumento = "CC"` y `Numero = "1099887001"`
- **THEN** el MID resuelve la identificación activa
- **AND** obtiene el `TerceroId` del representante
- **AND** valida su proveedor natural activo

#### Scenario: Representante inexistente

- **WHEN** el documento no corresponde a un tercero activo
- **THEN** el MID responde `400`
- **AND** no escribe ninguna sección

#### Scenario: Representante sin proveedor natural

- **WHEN** el tercero natural no tiene `proveedor_natural` activo
- **THEN** el MID responde `409`
- **AND** no crea la representación

#### Scenario: Documento incoherente con RepresentanteId

- **WHEN** llegan `RepresentanteId` y un documento que pertenece a otro tercero
- **THEN** el MID responde `400`

#### Scenario: Nombres distintos

- **WHEN** los nombres enviados no coinciden con el tercero resuelto
- **THEN** el MID usa el tercero del catálogo
- **AND** agrega un aviso `name_mismatch` sin bloquear

### Requirement: Cargo del representante

El `Representante` SHALL traer `CargoId` o `CodigoCargo`, resuelto contra
`Grupo_5` (`MID_GRUPO_CARGO`). Si llegan ambos, SHALL coincidir.

El MID SHALL escribir el cargo como una fila de
`info_complementaria_tercero` con código `CARGO_REPRESENTANTE` y
`Dato` = Id del cargo.

El cliente SHALL NOT enviar `CARGO_REPRESENTANTE` dentro de
`InfoComplementariaTercero`; si lo hace, el MID SHALL responder `400`.

#### Scenario: Cargo por código

- **WHEN** llega `CodigoCargo = "PROFESIONAL"`
- **THEN** el MID resuelve el Id local de `Grupo_5`
- **AND** escribe `CARGO_REPRESENTANTE` con ese Id

#### Scenario: Cargo fuera del grupo

- **WHEN** el código o Id no pertenece a `Grupo_5`
- **THEN** el MID responde `400`

### Requirement: Ciclo de vida de la representación

El MID SHALL crear o reutilizar una fila de `representacion` con clave
`ProveedorId + RepresentanteId`.

Si el representante cambia, el MID SHALL desactivar las representaciones activas
anteriores y SHALL crear o reactivar la del nuevo representante.

Si la representación ya está activa, el MID SHALL reutilizarla y no crear
duplicados.

#### Scenario: Cambio de representante

- **WHEN** el proveedor tenía activo al representante A y llega el B
- **THEN** la representación de A queda `Activo=false`
- **AND** la representación de B queda `Activo=true`

#### Scenario: Repetición idempotente

- **WHEN** llega el mismo representante que ya está activo
- **THEN** el MID reutiliza la misma fila
- **AND** no crea una segunda representación

#### Scenario: Representación inactiva

- **WHEN** existe una fila inactiva del mismo representante
- **THEN** el MID la reactiva
- **AND** no crea otra

### Requirement: `ProveedorJuridico`

`ProveedorJuridico` SHALL escribirse en `proveedor_juridico` con
`Id = ProveedorId`. Cuando la sección llega, SHALL validarse completa porque
alimenta columnas NOT NULL y una relación uno a uno.

Campos obligatorios:

- `ProcedenciaEmpresaId` o `CodigoProcedencia`.
- `TipoConformacionId` o `CodigoTipoConformacion`.
- `RegimenContributivo` o `CodigoRegimenContributivo`.
- `ExclusividadProducto`.
- `Pyme`.
- `RegistroMercantil`.
- `SujetoRetencion`.
- `AgenteRetenedor`.
- `ResponsableIca`.
- `ResponsableIva`.

Campos opcionales:

- `CiudadOrigenId`.
- `PaisOrigenId`.
- `CodigoPostal` (máximo 16, sin caracteres de control).

Si falta un obligatorio, el MID SHALL responder `400` y no escribir nada.

#### Scenario: Alta completa

- **WHEN** llega una sección con todos los obligatorios
- **THEN** el MID crea `proveedor_juridico` con `Id = ProveedorId`
- **AND** responde con los Ids y códigos resueltos

#### Scenario: Sección incompleta

- **WHEN** falta `Pyme` o cualquiera de los booleanos
- **THEN** el MID responde `400`
- **AND** no crea el proveedor

#### Scenario: Detalle inactivo

- **WHEN** existe `proveedor_juridico` con `Activo=false`
- **THEN** el MID responde `206`
- **AND** publica el conflicto en `MetaData.Errors`
- **AND** conserva las demás secciones

#### Scenario: Actualización parcial de otras secciones

- **WHEN** una actualización no envía `ProveedorJuridico`
- **THEN** el MID conserva el detalle existente

### Requirement: Procedencia, tipo legal y régimen contributivo

`CodigoProcedencia` SHALL resolverse contra el grupo `PROCEDENCIA`
(`MID_GRUPO_PROCEDENCIA`). `ProcedenciaEmpresaId` SHALL validarse contra el mismo
grupo. Si llegan ambos, SHALL coincidir.

`CodigoTipoConformacion` SHALL resolverse contra el tipo de parámetro
`TIPO_CONFORMACION` (`MID_TIPO_PARAMETRO_TIPO_CONFORMACION`). Corresponde a los
tipos de entidad legal (`SAS`, `SOC_LTDA`, `SOC_ANONIMA`, `EMP_UNIPERSONAL`,
`FUND_ANIMO_LUCRO`, etc.).

`CodigoRegimenContributivo` SHALL resolverse contra `REG_CONT`
(`MID_TIPO_PARAMETRO_REGIMEN_CONTRIBUTIVO`): `COMUN_ORDINARIO`,
`REGIMEN_SIMPLE_RST`, `REGIMEN_TRIB_ESP` o `ENTIDAD_DER_PUBLICO`.

Los Ids SHALL validarse contra el mismo tipo. Si llegan Id y código, SHALL
coincidir. Un código inexistente o inactivo SHALL responder `400`.

`TIPO_CONF_CUENTA` NO SHALL usarse para `TipoConformacionId`: la modalidad de la
cuenta (Individual/Conjunta/Consorcio) se guarda aparte en
`InfoComplementariaTercero`.

#### Scenario: Tipo legal por código

- **WHEN** llega `CodigoTipoConformacion = "SAS"`
- **THEN** el MID resuelve el Id local de `TIPO_CONFORMACION`
- **AND** escribe `TipoConformacionId`

#### Scenario: Régimen por código

- **WHEN** llega `CodigoRegimenContributivo = "COMUN_ORDINARIO"`
- **THEN** el MID resuelve el Id local de `REG_CONT`
- **AND** escribe `RegimenContributivo`

#### Scenario: Procedencia inexistente

- **WHEN** el código no existe en `PROCEDENCIA`
- **THEN** el MID responde `400`

#### Scenario: Id y código incoherentes

- **WHEN** llegan `TipoConformacionId` y `CodigoTipoConformacion` con valores distintos
- **THEN** el MID responde `400`

### Requirement: Booleanos tributarios y de tamaño

El formulario jurídico envía selecciones de varios valores que
`proveedor_juridico` guarda como booleanos. El MF SHALL aplicar estas reglas:

| Campo del formulario | Destino | Regla |
|---|---|---|
| `responsableIca` | `ResponsableIca` | `false` solo para “No Sujeto / Exento”; `true` para los demás. |
| `responsableIva` | `ResponsableIva` | `true` solo para “Responsable de IVA”; `false` para “No Responsable” y “Exento”. |
| `clasificacionPyme` | `Pyme` | `true` para Micro/Pequeña/Mediana; `false` para Gran Empresa. |
| `exclusividadImportacion` | `ExclusividadProducto` | `true` para las opciones “Sí”; `false` para “No”. |
| `registroMercantilActivo` | `RegistroMercantil` | `true` para matrícula activa; `false` para exenta. |
| `sujetoRetencion` | `SujetoRetencion` | Booleano directo. |
| `agenteRetenedor` | `AgenteRetenedor` | Booleano directo. |

El detalle exacto de ICA, IVA, tamaño y exclusividad NO se almacena en este
dominio; solo el booleano. `CalidadGranContribuyente` no tiene columna booleana y
se guarda como código en `CALIDAD_GRAN_CONTRIB`.

#### Scenario: PYME

- **WHEN** el MF envía `Pyme = true`
- **THEN** el MID lo guarda en `proveedor_juridico.Pyme`

#### Scenario: Gran empresa

- **WHEN** el formulario indica “No - Gran Empresa”
- **THEN** el MF envía `Pyme = false`

#### Scenario: Calidad de gran contribuyente

- **WHEN** el formulario indica “Autorretenedor por Resolución”
- **THEN** el MF envía `CALIDAD_GRAN_CONTRIB` con `Dato = "AUTORRETENEDOR"`

### Requirement: Info complementaria por código y grupo

Cada ítem de `InfoComplementariaTercero` SHALL traer exactamente uno de:

- `InfoComplementariaId` (compatibilidad), o
- `CodigoInfoComplementaria` + `CodigoGrupoInfoComplementaria`.

Cuando se usa `CodigoInfoComplementaria`, el grupo SHALL ser obligatorio. Los
Ids cambian entre ambientes; el par `(código, grupo)` es estable.

El MID SHALL resolver el par contra `terceros_crud` y usar el Id local
internamente. La respuesta SHALL incluir `CodigoInfoComplementaria` y
`CodigoGrupoInfoComplementaria`.

Un código inexistente o un grupo que no coincide SHALL responder `400`. Un
código repetido o dos opciones del mismo grupo SHALL responder `400`.

#### Scenario: Código inexistente

- **WHEN** llega `CodigoInfoComplementaria = "NO_EXISTE"`
- **THEN** el MID responde `400`
- **AND** no escribe el valor

#### Scenario: Grupo faltante

- **WHEN** llega `CodigoInfoComplementaria` sin
  `CodigoGrupoInfoComplementaria`
- **THEN** el MID responde `400`

#### Scenario: Grupo incorrecto

- **WHEN** el código existe, pero no pertenece al grupo enviado
- **THEN** el MID responde `400`

#### Scenario: Código repetido

- **WHEN** el mismo par `(código, grupo)` llega dos veces en la misma petición
- **THEN** el MID responde `400`

### Requirement: Códigos del grupo `PROVEEDOR_JURIDICO`

El MID SHALL resolver los siguientes códigos dentro del grupo
`PROVEEDOR_JURIDICO` (`MID_GRUPO_PROVEEDOR_JURIDICO`):

| Código | Nombre | `Dato` |
|---|---|---|
| `SITIO_WEB` | Sitio web corporativo | string http/https |
| `ACTIVIDAD_PRINCIPAL` | Actividad económica principal | Id CIIU |
| `CARGO_REPRESENTANTE` | Cargo del representante | Id `Grupo_5` (generado por el MID) |
| `TIPO_CONF_CUENTA` | Modalidad de la cuenta | `INDIVIDUAL_PRINCIPAL`, `CONJUNTA_MANCOMUNADA` o `CONSORCIO_UT` |
| `CALIDAD_GRAN_CONTRIB` | Calidad DIAN | `NO`, `SI_CALIFICADO` o `AUTORRETENEDOR` |

`TIPO_CONF_CUENTA` y `CALIDAD_GRAN_CONTRIB` se validan contra esos conjuntos
cerrados. `SITIO_WEB` SHALL ser una URL http/https. `ACTIVIDAD_PRINCIPAL` SHALL
ser un CIIU activo.

#### Scenario: `SITIO_WEB` inválido

- **WHEN** el valor no es una URL http/https
- **THEN** el MID responde `400`

#### Scenario: `TIPO_CONF_CUENTA` inválido

- **WHEN** el valor no es una de las tres modalidades
- **THEN** el MID responde `400`

#### Scenario: `CALIDAD_GRAN_CONTRIB` inválida

- **WHEN** el valor no es `NO`, `SI_CALIFICADO` o `AUTORRETENEDOR`
- **THEN** el MID responde `400`

#### Scenario: `ACTIVIDAD_PRINCIPAL` no CIIU

- **WHEN** el Id no pertenece al tipo CIIU
- **THEN** el MID responde `400`

### Requirement: `Contacto`

Cada contacto SHALL validar:

- `TipoCanalContactoId` positivo, activo y del tipo
  `MID_TIPO_PARAMETRO_CANAL_CONTACTO` (`MED_COMUNICACION`).
- `Proposito` obligatorio, normalizado a mayúsculas, máximo 50, sin `,` ni `:`.
- `ValorEspecifico` obligatorio, máximo 100, sin `,` ni `:`.
- `NombreContacto` máximo 100 y `Extension` máximo 10.

La clave SHALL ser `ProveedorId + TipoCanalContactoId + Proposito`. Si se
necesitan varios contactos con la misma clave, SHALL enviarse `Id`. Los
contactos omitidos SHALL NOT eliminarse.

Mapeo recomendado:

| Campo | Canal | Propósito |
|---|---|---|
| `correoNotificaciones` | `EMAIL` | `NOTIFICACIONES` |
| `telefonoAlterno` | `TEL_MOVIL` | `COMERCIAL` |
| `extension` | — | `Extension` del teléfono |
| `contactoComercial` + `telefonoAsesor` | `TEL_MOVIL` | `ASESOR` |
| `sitioWeb` | — | Info `SITIO_WEB` |
| `correoPagos` | — | Dominio 3 |

#### Scenario: Canal de otro tipo

- **WHEN** `TipoCanalContactoId` existe pero no es de `MED_COMUNICACION`
- **THEN** el MID responde `400`

#### Scenario: Contacto repetido

- **WHEN** dos contactos de la petición comparten canal y propósito
- **THEN** el MID responde `400`

#### Scenario: Actualización de contacto

- **WHEN** llega un contacto con la misma clave y un valor nuevo
- **THEN** el MID actualiza el contacto existente
- **AND** conserva `NombreContacto` si no se envía

### Requirement: `ActividadEconomicaProveedor`

Cada actividad SHALL existir, estar activa y ser de tipo CIIU
(`MID_TIPO_PARAMETRO_CIIU`).

La clave SHALL ser `ProveedorId + ActividadEconomicaId`. Las actividades
omitidas SHALL NOT eliminarse. La actividad principal SHALL ir aparte en
`ACTIVIDAD_PRINCIPAL`.

Si en la misma petición llegan actividades y `ACTIVIDAD_PRINCIPAL`, la principal
SHALL estar entre las actividades enviadas.

#### Scenario: Actividad no CIIU

- **WHEN** el Id no pertenece al tipo CIIU
- **THEN** el MID responde `400`

#### Scenario: Actividad repetida

- **WHEN** la misma actividad llega dos veces
- **THEN** el MID responde `400`

#### Scenario: Principal fuera de la lista

- **WHEN** `ACTIVIDAD_PRINCIPAL` no está entre las actividades enviadas
- **THEN** el MID responde `400`

### Requirement: `DescripcionPortafolio` y `Consentimiento`

`DescripcionPortafolio` SHALL tener máximo 2000 caracteres, sin caracteres de
control distintos de salto de línea, retorno de carro y tabulación.

`Consentimiento` SHALL crearse en `true` y conservarse salvo que se envíe. El
campo `Activo` del proveedor SHALL conservarse.

#### Scenario: Descripción larga

- **WHEN** la descripción supera 2000 caracteres
- **THEN** el MID responde `400`

#### Scenario: Descripción omitida

- **WHEN** una actualización no envía `DescripcionPortafolio`
- **THEN** el MID conserva el valor existente

### Requirement: Dirección DIAN

`DIRECCIÓN` SHALL guardar un objeto con la nomenclatura DIAN. Las claves
obligatorias son:

- `DepartamentoId`
- `CiudadId`
- `ViaId`
- `DetalleVia`
- `NumeroCruce`
- `PlacaPuerta`
- `Direccion`

Las claves opcionales son `PaisId`, `InteriorId`, `Via`, `Interior` y
`DetalleInterior`.

El MID SHALL validar que los Ids de lugar existan y estén activos. `ViaId`,
`InteriorId`, `PaisId`, `DepartamentoId` y `CiudadId` SHALL ser Ids de
`ubicaciones_crud`.

Si `ProveedorJuridico.CiudadOrigenId` y `DIRECCIÓN.CiudadId` llegan, SHALL
coincidir. Lo mismo para `PaisOrigenId` y `DIRECCIÓN.PaisId`. Si
`CiudadOrigenId` o `PaisOrigenId` se omiten, el MID SHALL tomarlos de
`DIRECCIÓN`.

La dirección SHALL enviarse completa en cada actualización; el MID valida las
claves obligatorias antes de mezclar.

#### Scenario: Dirección incompleta

- **WHEN** falta una clave obligatoria
- **THEN** el MID responde `400`

#### Scenario: Lugar inactivo

- **WHEN** un Id de la dirección no existe o está inactivo
- **THEN** el MID responde `400`

#### Scenario: Ciudad incoherente

- **WHEN** `ProveedorJuridico.CiudadOrigenId` y `DIRECCIÓN.CiudadId` difieren
- **THEN** el MID responde `400`

#### Scenario: Ciudad tomada de la dirección

- **WHEN** se omite `CiudadOrigenId` y llega `DIRECCIÓN.CiudadId`
- **THEN** el MID usa la ciudad de la dirección
- **AND** la guarda en `CiudadOrigenId`

### Requirement: Actualización parcial y fallo parcial

Cada llamada SHALL escribir solo las secciones que llegan. Los campos omitidos
SHALL NOT borrarse ni sobrescribirse. Los objetos `Dato` SHALL mezclarse por
clave; los escalares SHALL reemplazarse.

`ProveedorJuridico` y `Representante` son la excepción: si llegan, se validan
completos. Para actualizar solo un campo de esas secciones, el cliente SHALL
enviar la sección completa.

Un fallo en una sección SHALL NOT impedir las demás. El MID SHALL responder
`206` con `MetaData.Errors` y los Ids ya materializados.

Un aviso `ambiguous_match` o `name_mismatch` SHALL NOT marcar la respuesta como
parcial ni como fallo.

#### Scenario: Actualización de una sola sección

- **WHEN** llega solo `DescripcionPortafolio`
- **THEN** el MID actualiza esa sección
- **AND** conserva `ProveedorJuridico`, `Representacion`, `Contacto` y
  `ActividadEconomicaProveedor`

#### Scenario: Fallo de una dependencia

- **WHEN** una sección falla y otra se materializa
- **THEN** el MID responde `206`
- **AND** publica el error en `MetaData.Errors`
- **AND** devuelve los Ids de lo que sí se escribió

#### Scenario: Aviso de ambigüedad

- **WHEN** un código tiene varias filas activas y el MID usa la de menor Id
- **THEN** agrega un aviso `ambiguous_match`
- **AND** la respuesta sigue siendo `200` o `201`

### Requirement: Idempotencia

Repetir el mismo payload SHALL reutilizar las entidades existentes y SHALL NOT
crear duplicados de proveedor, `proveedor_juridico`, representación, perfil,
contacto, actividad, opción o fila de información complementaria.

#### Scenario: Segundo POST idéntico

- **WHEN** se repite el mismo `POST`
- **THEN** el MID responde `200`
- **AND** todas las entidades aparecen como `Reutilizados`
- **AND** no se crean filas nuevas

#### Scenario: Misma representación

- **WHEN** se repite el mismo representante
- **THEN** el MID reutiliza la representación activa
- **AND** no desactiva ni duplica filas

### Requirement: Códigos de respuesta

| Código | Caso |
|---|---|
| `201` | Al menos una creación |
| `200` | Solo actualizaciones o reutilización |
| `206` | Fallo parcial |
| `400` | Validación, representante inexistente, código inválido, sección incompleta |
| `404` | Tercero o proveedor inexistente en `PUT` |
| `409` | Tercero o proveedor de otro tipo, inactivo, duplicado, representante sin proveedor natural |
| `500` | Catálogo no configurado |
| `502/504` | Dependencia caída o timeout |

#### Scenario: Validación

- **WHEN** el payload no cumple el contrato
- **THEN** el MID responde `400`
- **AND** no escribe en los CRUD

#### Scenario: Conflicto de proveedor

- **WHEN** el proveedor es de otro tipo, está inactivo o hay duplicados
- **THEN** el MID responde `409`

### Requirement: Configuración

El MID SHALL usar:

- `MID_TIPO_PROVEEDOR_JURIDICO` (por defecto `2`).
- `MID_TIPO_PROVEEDOR_NATURAL` (por defecto `1`, para el representante).
- `MID_TIPO_CONTRIBUYENTE_JURIDICA` y `MID_TIPO_CONTRIBUYENTE_NATURAL`.
- `MID_TIPO_PARAMETRO_CIIU`.
- `MID_TIPO_PARAMETRO_CANAL_CONTACTO`.
- `MID_TIPO_PARAMETRO_REGIMEN_CONTRIBUTIVO` (por defecto `REG_CONT`).
- `MID_TIPO_PARAMETRO_TIPO_CONFORMACION` (por defecto `TIPO_CONFORMACION`).
- `MID_GRUPO_PROCEDENCIA` (por defecto `PROCEDENCIA`).
- `MID_GRUPO_CARGO` (por defecto `Grupo_5`).
- `MID_GRUPO_PROVEEDOR_JURIDICO` (por defecto `PROVEEDOR_JURIDICO`).
- `TERCEROS_CRUD_SERVICE`, `GESTION_PROVEEDORES_CRUD_SERVICE`,
  `PARAMETROS_CRUD_SERVICE` y `UBICACIONES_CRUD_SERVICE`.

Los Ids de CIIU, canales, tipos de documento, cargos, lugares y parámetros SHALL
resolverse en tiempo de ejecución contra el catálogo del entorno. No SHALL
hardcodearse.

#### Scenario: Canal de contacto

- **WHEN** el MID valida un contacto
- **THEN** usa `MID_TIPO_PARAMETRO_CANAL_CONTACTO`
- **AND** exige que el parámetro pertenezca a ese tipo

#### Scenario: Régimen contributivo

- **WHEN** el MF envía un código de régimen
- **THEN** el MID lo resuelve contra `MID_TIPO_PARAMETRO_REGIMEN_CONTRIBUTIVO`

### Requirement: Auditoría de modificación

Toda escritura del MID contra `gestion_proveedores_crud` SHALL enviar
`UsuarioModificacionId` con el `TerceroId` del proveedor jurídico, tanto en
`POST` como en `PUT`. Aplica a `proveedor`, `proveedor_juridico`,
`representacion`, `contacto`, `actividad_economica_proveedor`,
`perfil_proveedor` y `cuenta_bancaria` (dominio 3).

Para `320522`, el valor SHALL ser `320522`, no el `TerceroId` del representante.

El cliente SHALL NOT enviar `UsuarioModificacionId`; el MID lo administra.

El CRUD no completa el campo por sí solo y, en los `PUT`, lo sobrescribe con el
valor recibido. Si un registro histórico tiene `0`, el MID lo actualiza en la
primera escritura.

#### Scenario: Alta jurídica con auditoría

- **WHEN** el MID crea el proveedor, `proveedor_juridico`, representación,
  contactos y actividades
- **THEN** todas las filas quedan con `UsuarioModificacionId = TerceroId` del
  proveedor jurídico

#### Scenario: Actualización jurídica con auditoría

- **WHEN** el MID actualiza un registro existente con `UsuarioModificacionId = 0`
- **THEN** la fila queda con `UsuarioModificacionId = TerceroId` del proveedor
  jurídico
- **AND** la operación conserva la idempotencia si el mismo tercero repite la
  petición

#### Scenario: Cambio de representante

- **WHEN** el MID desactiva una representación anterior y activa otra
- **THEN** ambas filas quedan con `UsuarioModificacionId = TerceroId` del
  proveedor jurídico

#### Scenario: Campo enviado por el cliente

- **WHEN** el cliente intenta enviar `UsuarioModificacionId`
- **THEN** el MID lo rechaza porque es un campo administrado

## Non-Goals

- No se implementa el dominio 1 (`tercero`, `datos_identificacion`).
- No se crea el representante ni su proveedor natural.
- No se implementa la cuenta bancaria, el capital ni el correo de pagos (dominio 3).
- No se gestionan documentos.
- No se implementa autenticación, ownership ni estado del proveedor.
- No se implementan `PATCH` ni `DELETE`.
- No se actualiza Swagger/OpenAPI en este cambio.
- No se modifica el MF.
- No se modifica el código de `terceros_crud` ni de `gestion_proveedores_crud`;
  solo se hacen peticiones HTTP para crear el grupo y las filas del catálogo.

## Notas

- El spec complementa `registro-proveedor-juridica` (dominio 1).
- `ProveedorJuridico.TipoConformacionId` corresponde a los 23 tipos de entidad
  legal de `TIPO_CONFORMACION`. El formulario actual no captura ese campo; el MF
  deberá agregarlo. La opción “Tipo de Conformación” del paso 2
  (Individual/Conjunta/Consorcio) se guarda aparte en `TIPO_CONF_CUENTA`.
- El representante debe existir por dominio 1 natural y tener proveedor natural
  activo por dominio 2 natural. El formulario jurídico no captura los datos de
  nacimiento del representante, así que el MF debe resolverlo antes.
- El grupo `PROVEEDOR_JURIDICO` y sus 5 filas se crean por API en
  `terceros_crud`. En pruebasapi quedaron con Ids propios del entorno; el
  contrato usa los códigos, no los Ids.
- `CALIDAD_GRAN_CONTRIB` no tiene columna booleana en `proveedor_juridico`; se
  guarda como código para no perder la opción “Autorretenedor”.
- El E2E se ejecutó con `TerceroId=320522` y `ProveedorId=6`: `POST` `201`,
  segundo `POST` `200` idempotente, 6 filas de información complementaria, 3
  contactos, 1 actividad, representante `320521` con proveedor natural `1`,
  representación `3` activa y cambio posterior a `320523` con proveedor natural
  `3`, que activó la representación `4` y desactivó la anterior.
- Los Ids de `InfoComplementariaTercero` que aparecen en la respuesta son del
  entorno; el consumidor no debe usarlos como contrato.
