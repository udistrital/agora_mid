# Dominio 2 — Caracterización de persona jurídica

Endpoints del MID para registrar y actualizar parcialmente la caracterización de
una persona jurídica.

```text
POST /v1/proveedores/juridica/caracterizacion
PUT  /v1/proveedores/juridica/caracterizacion/{terceroId}
PUT  /v1/proveedores/juridica/caracterizacion/proveedor/{proveedorId}
```

- El parámetro de ruta va al final.
- Solo se usan `POST` y `PUT`.
- `POST` crea el proveedor si no existe; `PUT` exige que exista.
- Para el paso 2 y el paso 4 se recomienda el `PUT` por `proveedorId`.
- Cada llamada escribe solo las secciones que llegan. No elimina lo omitido.
- `ProveedorJuridico` y `Representante` son secciones completas: si llegan, deben
  traer todos sus obligatorios. Para actualizar un solo campo de esas secciones,
  envíe la sección completa.
- El envelope sigue `utils_oas` v2: `Success`, `Status` string, `Message`,
  `Data` y `MetaData.Errors`.

## 1. Regla de oro: códigos, no Ids

Los Ids cambian entre pruebas y producción; los códigos de abreviación no. Por
eso:

- `InfoComplementariaTercero` usa `CodigoInfoComplementaria` +
  `CodigoGrupoInfoComplementaria`.
- `PerfilProveedor` usa `CodigoPerfil`.
- `ProveedorJuridico` acepta `CodigoProcedencia`, `CodigoTipoConformacion` y
  `CodigoRegimenContributivo`, además de los Ids por compatibilidad.
- `Representante` acepta `CodigoTipoDocumento` y `CodigoCargo`, además de los
  Ids.

Si llegan Id y código, deben coincidir.

## 2. Orden de las llamadas

```text
1. POST /v1/proveedores/juridica                 → tercero + NIT
2. POST /v1/proveedores/juridica/caracterizacion → dominio 2
3. POST /v1/proveedores/cuenta-bancaria/{id}     → dominio 3
4. Gestor documental                             → RUT, RUP, renta
```

Antes del paso 2, el representante legal debe existir:

```text
POST /v1/proveedores/natural
POST /v1/proveedores/natural/caracterizacion
```

El dominio 2 jurídico no crea al representante ni su `proveedor_natural`.

## 3. Representante legal

```json
"Representante": {
  "CodigoTipoDocumento": "CC",
  "Numero": "1099887001",
  "CodigoCargo": "PROFESIONAL",
  "PrimerNombre": "Carlos",
  "SegundoNombre": "Eduardo",
  "PrimerApellido": "Rodriguez",
  "SegundoApellido": "Patiño"
}
```

Reglas:

- Exactamente `RepresentanteId` o (`TipoDocumentoId`/`CodigoTipoDocumento` +
  `Numero`).
- `PA` se acepta como alias de `PAS`.
- El tercero debe estar activo y ser `P_NATURAL`.
- Debe tener `proveedor` activo de tipo natural y `proveedor_natural` activo.
- `CargoId` o `CodigoCargo` es obligatorio; se resuelve contra `Grupo_5`.
- El MID escribe el cargo en `InfoComplementariaTercero` con código
  `CARGO_REPRESENTANTE`; el cliente no debe enviarlo allí.
- Los nombres son opcionales. Si no coinciden con el tercero resuelto, el MID
  usa el tercero del catálogo y emite un aviso `name_mismatch`.
- Si el representante no existe: `400`.
- Si no es natural o no tiene proveedor natural activo: `409`.

La relación se guarda en `representacion` con clave
`ProveedorId + RepresentanteId`. Al cambiar de representante, la relación
anterior queda `Activo=false`.

## 4. `ProveedorJuridico`

```json
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
}
```

Obligatorios al enviar la sección:

| Campo | Destino | Regla |
|---|---|---|
| `ProcedenciaEmpresaId` / `CodigoProcedencia` | `ProcedenciaEmpresaId` | Grupo `PROCEDENCIA`. |
| `TipoConformacionId` / `CodigoTipoConformacion` | `TipoConformacionId` | Tipo `TIPO_CONFORMACION` (23 tipos legales). |
| `RegimenContributivo` / `CodigoRegimenContributivo` | `RegimenContributivo` | Tipo `REG_CONT`. |
| `ExclusividadProducto` | `ExclusividadProducto` | Booleano. |
| `Pyme` | `Pyme` | Booleano. |
| `RegistroMercantil` | `RegistroMercantil` | Booleano. |
| `SujetoRetencion` | `SujetoRetencion` | Booleano. |
| `AgenteRetenedor` | `AgenteRetenedor` | Booleano. |
| `ResponsableIca` | `ResponsableIca` | Booleano. |
| `ResponsableIva` | `ResponsableIva` | Booleano. |

Opcionales:

- `CiudadOrigenId`.
- `PaisOrigenId`.
- `CodigoPostal` (máximo 16).

### 4.1. Tipo de conformación legal vs modalidad de la cuenta

`proveedor_juridico.tipo_conformacion_id` es NOT NULL y corresponde a los 23
tipos de entidad legal de `TIPO_CONFORMACION`:

```text
SAS, SOC_LTDA, SOC_ANONIMA, SOC_COM_ACCIONES, SOC_COM_SIMPLE,
EMP_UNIPERSONAL, FUND_ANIMO_LUCRO, ONG, ENT_ESTATAL,
ESP, ESE, EPS, IPS, ARS, APC, INST_ASEGURADORA, INST_FINANCIERA,
INV_CIENTIFICA, ORG_MULTILATERAL, OTRAS_ECON_SOLID, UNIV_PRIVADA,
UNIV_PUBLICA, OTRO
```

El formulario actual **no captura** este campo. El MF deberá agregar un selector
de `TIPO_CONFORMACION` y enviar `CodigoTipoConformacion`. Sin él, el MID responde
`400` y no crea el detalle.

La opción “Tipo de Conformación” del paso 2 (Individual/Principal,
Conjunta/Mancomunada, Consorcio/UT) es otra cosa: se guarda en
`InfoComplementariaTercero` con código `TIPO_CONF_CUENTA`:

| Opción del formulario | Código |
|---|---|
| Individual / Principal | `INDIVIDUAL_PRINCIPAL` |
| Conjunta / Mancomunada | `CONJUNTA_MANCOMUNADA` |
| Consorcio / Unión Temporal | `CONSORCIO_UT` |

### 4.2. Régimen contributivo

| Opción del formulario | Código |
|---|---|
| Común / Ordinario | `COMUN_ORDINARIO` |
| Régimen Simple de Tributación | `REGIMEN_SIMPLE_RST` |
| Régimen Tributario Especial | `REGIMEN_TRIB_ESP` |
| Entidad de Derecho Público | `ENTIDAD_DER_PUBLICO` |

### 4.3. Booleanos tributarios

| Campo del formulario | Destino | Regla |
|---|---|---|
| `responsableIca` | `ResponsableIca` | `false` solo para “No Sujeto / Exento”; `true` para los demás. |
| `responsableIva` | `ResponsableIva` | `true` solo para “Responsable de IVA”; `false` para “No Responsable” y “Exento”. |
| `clasificacionPyme` | `Pyme` | `true` para Micro/Pequeña/Mediana; `false` para Gran Empresa. |
| `exclusividadImportacion` | `ExclusividadProducto` | `true` para las opciones “Sí”; `false` para “No”. |
| `registroMercantilActivo` | `RegistroMercantil` | `true` para matrícula activa; `false` para exenta. |
| `sujetoRetencion` | `SujetoRetencion` | Booleano directo. |
| `agenteRetenedor` | `AgenteRetenedor` | Booleano directo. |

El detalle exacto de ICA, IVA, tamaño y exclusividad no se almacena; solo el
booleano. `CalidadGranContribuyente` no tiene columna booleana y se guarda como
código en `CALIDAD_GRAN_CONTRIB`:

| Opción | Código |
|---|---|
| No | `NO` |
| Sí - Calificado por Resolución DIAN | `SI_CALIFICADO` |
| Autorretenedor por Resolución | `AUTORRETENEDOR` |

## 5. Grupo `PROVEEDOR_JURIDICO`

Campos que no existen en `gestion_proveedores_crud` y se crearon en
`terceros_crud`:

| Código | Nombre | `Dato` |
|---|---|---|
| `SITIO_WEB` | Sitio web corporativo | string http/https |
| `ACTIVIDAD_PRINCIPAL` | Actividad económica principal | Id CIIU |
| `CARGO_REPRESENTANTE` | Cargo del representante | Id `Grupo_5` |
| `TIPO_CONF_CUENTA` | Modalidad de la cuenta | código de 3 opciones |
| `CALIDAD_GRAN_CONTRIB` | Calidad DIAN | código de 3 opciones |

`CARGO_REPRESENTANTE` lo genera el MID desde `Representante.Cargo`; no se envía
en `InfoComplementariaTercero`.

## 6. Dirección DIAN

```json
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
```

Obligatorios: `DepartamentoId`, `CiudadId`, `ViaId`, `DetalleVia`,
`NumeroCruce`, `PlacaPuerta`, `Direccion`.

Opcionales: `PaisId`, `InteriorId`, `Via`, `Interior`, `DetalleInterior`.

`ProveedorJuridico.CiudadOrigenId` y `DIRECCIÓN.CiudadId` deben coincidir si
llegan ambos. Lo mismo para `PaisOrigenId` y `DIRECCIÓN.PaisId`. Si se omiten en
`ProveedorJuridico`, el MID los toma de `DIRECCIÓN`.

## 7. Contactos

| Campo | Canal | Propósito |
|---|---|---|
| `correoNotificaciones` | `EMAIL` | `NOTIFICACIONES` |
| `telefonoAlterno` | `TEL_MOVIL` | `COMERCIAL` |
| `extension` | — | `Extension` del teléfono |
| `contactoComercial` + `telefonoAsesor` | `TEL_MOVIL` | `ASESOR` |
| `sitioWeb` | — | Info `SITIO_WEB` |
| `correoPagos` | — | Dominio 3 |

La clave es `ProveedorId + TipoCanalContactoId + Proposito`. Los canales se
resuelven en runtime contra `MED_COMUNICACION`; no se hardcodean.

## 8. Actividades CIIU

- `ActividadEconomicaProveedor` recibe los Ids de CIIU.
- `ACTIVIDAD_PRINCIPAL` va en `InfoComplementariaTercero` con el Id CIIU.
- Si en la misma petición llegan actividades y principal, la principal debe estar
  entre las actividades.
- Las actividades omitidas no se eliminan.

## 9. Descripción y consentimiento

- `DescripcionPortafolio`: máximo 2000 caracteres; el MF envía texto plano.
- `Consentimiento`: por defecto `true`; si se omite en una actualización, se
  conserva.
- El campo `Activo` del proveedor se conserva.

## 10. Mapa completo del formulario

### Paso 1

| Campo | Destino |
|---|---|
| `nit` | Dominio 1: `DatosIdentificacion.Numero` |
| `digitoVerificacion` | Dominio 1: `DigitoVerificacion` |
| `nitConfirmacion` | Solo cliente |
| `procedencia` | `ProveedorJuridico.CodigoProcedencia` |
| `razonSocial` | Dominio 1: `DatosBasicos.RazonSocial` |
| `tipoDocumentoRepresentante` | `Representante.CodigoTipoDocumento` |
| `documentoRepresentante` | `Representante.Numero` |
| `cargoRepresentante` | `Representante.CodigoCargo` |
| `primerNombre` y demás nombres | Validación opcional del representante |
| `regimenContributivo` | `ProveedorJuridico.CodigoRegimenContributivo` |
| `sujetoRetencion` | `ProveedorJuridico.SujetoRetencion` |
| `agenteRetenedor` | `ProveedorJuridico.AgenteRetenedor` |
| `responsableIca` | `ProveedorJuridico.ResponsableIca` |
| `responsableIva` | `ProveedorJuridico.ResponsableIva` |
| `clasificacionPyme` | `ProveedorJuridico.Pyme` |
| `registroMercantilActivo` | `ProveedorJuridico.RegistroMercantil` |
| `exclusividadImportacion` | `ProveedorJuridico.ExclusividadProducto` |
| `calidadGranContribuyente` | Info `CALIDAD_GRAN_CONTRIB` |
| `departamento` | `DIRECCIÓN.DepartamentoId` |
| `ciudad` | `ProveedorJuridico.CiudadOrigenId` + `DIRECCIÓN.CiudadId` |
| `pais` | `ProveedorJuridico.PaisOrigenId` + `DIRECCIÓN.PaisId` |
| `tipoVia` | `DIRECCIÓN.ViaId` |
| `detalleVia` | `DIRECCIÓN.DetalleVia` |
| `numeroCruce` | `DIRECCIÓN.NumeroCruce` |
| `placaPuerta` | `DIRECCIÓN.PlacaPuerta` |
| `interiorTipo` | `DIRECCIÓN.InteriorId` |
| `detalleInterior` | `DIRECCIÓN.DetalleInterior` |
| `Direccion` | `DIRECCIÓN.Direccion` |
| `correoNotificaciones` | `Contacto` `EMAIL` / `NOTIFICACIONES` |
| `correoNotificacionesConfirmacion` | Solo cliente |
| `extension` | `Contacto.Extension` |
| `telefonoAlterno` | `Contacto` `TEL_MOVIL` / `COMERCIAL` |
| `sitioWeb` | Info `SITIO_WEB` |
| `contactoComercial` | `Contacto.NombreContacto` del asesor |
| `telefonoAsesor` | `Contacto` `TEL_MOVIL` / `ASESOR` |

### Paso 2

| Campo | Destino |
|---|---|
| `banco` | Dominio 3 |
| `tipoCuenta` | Dominio 3 |
| `tipoConformacion` | Info `TIPO_CONF_CUENTA` |
| `numeroCuenta` | Dominio 3 |
| `numeroCuentaConfirmacion` | Solo cliente |
| `capitalAutorizado` | Dominio 3 |
| `correoPagos` | Dominio 3 |
| `correoPagosConfirmacion` | Solo cliente |

### Paso 3

| Campo | Destino |
|---|---|
| `rutArchivo`, `tieneRup`, `rupArchivo` | Gestor documental |

### Paso 4

| Campo | Destino |
|---|---|
| `actividadesCiiu` | `ActividadEconomicaProveedor` |
| `actividadCiiuPrincipalId` | Info `ACTIVIDAD_PRINCIPAL` |
| `descripcionServicios` | `DescripcionPortafolio` |
| `aceptaTratamiento` | `Consentimiento` |

## 11. Errores

| Código | Caso |
|---|---|
| `201` | Al menos una creación |
| `200` | Solo actualización o reutilización |
| `206` | Fallo parcial con `MetaData.Errors` |
| `400` | Validación, representante inexistente, código inválido, sección incompleta |
| `404` | Tercero o proveedor inexistente en `PUT` |
| `409` | Tercero/proveedor de otro tipo, inactivo, duplicado, representante sin proveedor natural |
| `500` | Catálogo no configurado |
| `502/504` | Dependencia caída o timeout |

## 12. Ejemplo mínimo

```json
{
  "TerceroId": 320522,
  "Representante": {
    "CodigoTipoDocumento": "CC",
    "Numero": "1099887001",
    "CodigoCargo": "PROFESIONAL"
  },
  "ProveedorJuridico": {
    "CodigoProcedencia": "NACIONAL",
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
  "DescripcionPortafolio": "Servicios corporativos",
  "Consentimiento": true
}
```

## 13. Notas de despliegue

- Crear el grupo `PROVEEDOR_JURIDICO` y sus 5 filas en `terceros_crud` por API.
- Verificar que existan:
  - `PROCEDENCIA`.
  - `Grupo_5`.
  - `MED_COMUNICACION` con los 12 canales.
  - `REG_CONT` con sus 4 regímenes.
  - `TIPO_CONFORMACION` con los 23 tipos legales.
  - El catálogo CIIU.
- No hardcodear Ids de CIIU, canales, tipos de documento, cargos ni lugares.
- El MF debe agregar el campo de tipo de entidad legal y registrar al
  representante con su proveedor natural antes de llamar al dominio 2 jurídico.

## 14. Auditoría de modificación

Todas las escrituras del MID contra `gestion_proveedores_crud` envían
`UsuarioModificacionId` con el `TerceroId` del proveedor jurídico:

| Tabla | Valor |
|---|---|
| `proveedor` | `TerceroId` del proveedor jurídico |
| `proveedor_juridico` | `TerceroId` del proveedor jurídico |
| `representacion` | `TerceroId` del proveedor jurídico |
| `contacto` | `TerceroId` del proveedor jurídico |
| `actividad_economica_proveedor` | `TerceroId` del proveedor jurídico |
| `perfil_proveedor` | `TerceroId` del proveedor jurídico |
| `cuenta_bancaria` | `TerceroId` del proveedor jurídico |

Para `320522`, el valor es `320522`, no el `TerceroId` del representante.

Reglas:

- El valor se envía tanto en `POST` como en `PUT`. El CRUD no lo llena solo y,
  en los `PUT`, lo sobrescribe con el valor recibido.
- El cliente no envía `UsuarioModificacionId`; el MID lo administra.
- Los registros históricos que quedaron en `0` se actualizan en la primera
  escritura del MID.
- En una fase con autenticación, el valor deberá salir del usuario autenticado y
  el `TerceroId` del proveedor quedará como respaldo.
- `info_complementaria_tercero` vive en `terceros_crud` y no usa este campo de
  `gestion_proveedores_crud`.
