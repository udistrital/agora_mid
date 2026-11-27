# Dominio 2 — Caracterización de persona natural

Endpoints del MID para registrar y actualizar parcialmente la caracterización de
una persona natural.

```text
POST /v1/proveedores/natural/caracterizacion
PUT  /v1/proveedores/natural/caracterizacion/{terceroId}
PUT  /v1/proveedores/natural/caracterizacion/proveedor/{proveedorId}
```

- El parámetro de ruta va al final.
- Solo se usan `POST` y `PUT`.
- `POST` crea el proveedor si no existe; `PUT` exige que exista.
- Para los pasos 2 y 4 se recomienda el `PUT` por `proveedorId`: evita resolver
  el proveedor por `TerceroId` y elimina ambigüedades. El `PUT` por `terceroId`
  se conserva por compatibilidad.
- En el `PUT` por `proveedorId`, el `TerceroId` del body es opcional; si llega,
  debe coincidir con el tercero del proveedor.
- Cada llamada escribe solo las secciones que llegan. No elimina ni sobrescribe
  lo omitido.
- El envelope sigue `utils_oas` v2: `Success`, `Status` string, `Message`,
  `Data` y `MetaData.Errors`.

## 1. Regla de oro: códigos, no Ids

Los Ids de `terceros_crud` cambian entre pruebas y producción; los códigos de
abreviación no. Por eso `InfoComplementariaTercero` **no recibe
`InfoComplementariaId`**:

```json
{
  "CodigoInfoComplementaria": "MASCULINO",
  "CodigoGrupoInfoComplementaria": "Grupo_6"
}
```

- `CodigoInfoComplementaria`: código de la fila.
- `CodigoGrupoInfoComplementaria`: código del grupo. Es obligatorio cuando se
  usa `CodigoInfoComplementaria`, porque hay códigos repetidos entre grupos
  (`NO APLICA`, `NO CONTESTA`, `EPS`).
- El MID resuelve `(código, grupo)` contra el catálogo del entorno y usa el Id
  local internamente.

`InfoComplementariaId` se mantiene solo por compatibilidad con clientes
antiguos. El MF nuevo no lo usa.

## 2. Selectores de opción única

Estos campos se guardan como **una fila por opción**, con `Dato` vacío. Al
cambiar la opción, el MID desactiva la opción anterior del mismo grupo y
reactiva o crea la nueva.

| Campo del MF | Grupo | Opciones |
|---|---|---|
| `genero` | `Grupo_6` | `FEMENINO`, `MASCULINO`, `NO APLICA`, `INTERSEXUAL` |
| `grupoEtnico` | `Grupo_3` | `AFRODESCENDIENTES`, `INDIGENAS`, `RAIZALES`, `ROM`, `NO APLICA` |
| `estadoCivil` | `Grupo_2` | `SOLTERO`, `CASADO`, `UNION LIBRE`, `VIUDO`, `DIVORCIADO` |
| `seReconoceComo` | `IDENTIDAD_GENERO` | `CISGENERO`, `TRANSFEMENINO`, `TRANSMASCULINO`, `NO_CONTESTA` |
| `orientacionSexual` | `ORIENTACION_SEXUAL` | `BISEXUAL`, `HETEROSEXUAL`, `HOMOSEXUAL`, `NO_CONTESTA` |
| `discapacidad` | `Grupo_1` | `FISICA`, `SENSORIAL`, `AUDITIVA`, `VISUAL`, `PSIQUICA`, `MENTAL`, `NO APLICA` |

Payload:

```json
"InfoComplementariaTercero": [
  { "CodigoInfoComplementaria": "MASCULINO", "CodigoGrupoInfoComplementaria": "Grupo_6" },
  { "CodigoInfoComplementaria": "SOLTERO", "CodigoGrupoInfoComplementaria": "Grupo_2" },
  { "CodigoInfoComplementaria": "NO APLICA", "CodigoGrupoInfoComplementaria": "Grupo_3" },
  { "CodigoInfoComplementaria": "CISGENERO", "CodigoGrupoInfoComplementaria": "IDENTIDAD_GENERO" },
  { "CodigoInfoComplementaria": "HETEROSEXUAL", "CodigoGrupoInfoComplementaria": "ORIENTACION_SEXUAL" },
  { "CodigoInfoComplementaria": "NO APLICA", "CodigoGrupoInfoComplementaria": "Grupo_1" }
]
```

Reglas:

- `Dato` se omite o va vacío; cualquier otro valor responde `400`.
- Solo se puede enviar una opción por grupo en cada llamada.
- Si la opción estaba inactiva, se reactiva; no se responde `409`.
- Las filas `DOC_*` del mismo grupo no se desactivan.

## 3. Campos con fila propia

| Campo del MF | Código | Grupo | `Dato` |
|---|---|---|---|
| `declaranteRenta` | `DECLARANTE_RENTA` | `Grupo_9` | booleano |
| `cuentaAfc` | `CUENTA_AFC` | `Grupo_9` | booleano |
| `esMigrante` | `MIGRANTE` | `Grupo_9` | booleano |
| `victimaConflictoArmado` | `VICTIMA_CONFLICTO` | `Grupo_9` | booleano |
| `tieneHijos` | `HIJOS` | `Grupo_9` | booleano |
| `numeroHijos` | `NUMERO_HIJOS` | `Grupo_9` | entero |
| `esCabezaFamilia` | `CABEZA_FAMILIA` | `Grupo_9` | booleano |
| `esPep` | `PERSONA_PEP` | `Grupo_9` | booleano |
| `personasACargo` | `PERSONAS_A_CARGO` | `Grupo_9` | booleano |
| `eps` | `EPS` | `V` | Id parámetro tipo `EPS` |
| Dirección | `DIRECCIÓN` | `Grupo_10` | objeto DIAN |
| Ciudad de residencia | `LUGAR_RESIDENCIA` | `Grupo_10` | Id de lugar |

`MIGRANTE` tiene dos filas activas en `Grupo_9` (444 y 452). El MID usa la de
menor Id y emite un aviso `ambiguous_match`. Es el único caso de ambigüedad
conocido.

## 4. Grupo `PROVEEDOR_NATURAL`

Campos que no existen en `gestion_proveedores_crud` y que se crearon en un grupo
propio de `terceros_crud`:

| Código | Nombre | `Dato` |
|---|---|---|
| `SITIO_WEB` | Sitio web o portafolio | string http/https |
| `AUTORIZACION_GIRO` | Autorización de giro | booleano |
| `APLICA_RUP` | Aplica RUP | booleano |
| `ACTIVIDAD_PRINCIPAL` | Actividad económica principal | Id parámetro CIIU |
| `AFP` | Administradora de fondos de pensiones | Id parámetro tipo `EFP` |
| `CCF` | Caja de compensación familiar | Id parámetro tipo `CCF` |
| `PAIS_EXPEDICION` | País de expedición | Id de lugar |
| `DEPTO_EXPEDICION` | Departamento de expedición | Id de lugar |

`DEPTO_EXPEDICION` se usa en lugar de `DEPARTAMENTO_EXPEDICION` porque
`codigo_abreviacion` es `varchar(20)`. El caso `DEPARTAMENTO_EXPEDICION` se
mantiene aceptado por compatibilidad.

## 5. Qué vive en `gestion_proveedores_crud` y no se duplica

| Campo del MF | Destino |
|---|---|
| `paisNacimiento` | `proveedor_natural.pais_nacimiento_id` |
| `experienciaLaboralTotal` | `proveedor_natural.experiencia_laboral` |
| `experienciaProfesional` | `proveedor_natural.experiencia_profesional` |
| `medicinaPrepagada` | `proveedor_natural.medicina_prepagada` |
| `responsableIva` | `proveedor_natural.responsable_iva` |
| `resideFiscalmenteColombia` | `proveedor_natural.reside_colombia` |
| `aportesPensionVoluntaria` | `proveedor_natural.pension_voluntaria` |
| `pensionadoAsignacionRetiro` | `proveedor_natural.pensionado` |
| `perfilRolPrincipal` | `perfil_proveedor` |
| `correoElectronico`, `telefonoCelular`, `otroContacto`, `extensionTelefonica` | `contacto` |
| `contactoEmergencia`, `telefonoContactoEmergencia` | `contacto` propósito `EMERGENCIA` |
| `actividadesCiiu` | `actividad_economica_proveedor` |
| `portafolioCapacidadEntrega` | `proveedor.descripcion_portafolio` |
| `consentimientoExpreso` | `proveedor.consentimiento` |
| `entidadBancaria`, `tipoCuenta`, `numeroCuentaBancaria`, `capitalAutorizadoPatrimonio`, `correoNotificacionesTesoreria` | dominio 3, cuenta bancaria |

`MED_PREPAGADA`, `RESPONSABLE_IVA`, `RESIDE_FIS_COL` y `PENSION_VOL` existen en
`info_complementaria`, pero el MID **no los escribe**: su fuente es
`proveedor_natural`. `PENSIONADO` no tiene fila en `info_complementaria`.

## 6. `ProveedorNatural`

Campos opcionales con punteros. Al crear el detalle, `PaisNacimientoId` es
obligatorio. En actualización, los campos ausentes conservan su valor.

| Campo | Regla |
|---|---|
| `PaisNacimientoId` | Lugar activo en `ubicaciones_crud`; obligatorio al crear |
| `Cargo` | Máximo 150 caracteres |
| `MedicinaPrepagada`, `ResponsableIva`, `ResideColombia`, `PensionVoluntaria`, `Pensionado`, `Dependiente*` | Booleanos |
| `ValorUvtPrepagada`, `ValorAfc`, `ValorPensionVoluntaria` | `>= 0` |
| `InteresViviendaAfc` | Entre 0 y 1 |
| `ExperienciaLaboral`, `ExperienciaProfesional` | Enteros `>= 0` |

## 7. `PerfilProveedor`

El perfil también se envía por código, porque vive en `info_complementaria`
(`Grupo_5`):

```json
"PerfilProveedor": [{ "CodigoPerfil": "PROFESIONAL" }]
```

- `CodigoPerfil` se resuelve contra `Grupo_5`.
- `PerfilId` se mantiene por compatibilidad; si llegan ambos, deben coincidir.
- La clave es `ProveedorId + PerfilId`.
- No se eliminan perfiles omitidos.

## 8. `Contacto`

- `TipoCanalContactoId` debe existir, estar activo y pertenecer a
  `MID_TIPO_PARAMETRO_CANAL_CONTACTO` (por defecto `MED_COMUNICACION`).
- `Proposito` se normaliza a mayúsculas, máximo 50, sin `,` ni `:`.
- `ValorEspecifico` máximo 100, sin `,` ni `:`.
- `NombreContacto` máximo 100; `Extension` máximo 10.
- La clave es `ProveedorId + TipoCanalContactoId + Proposito`.
- Si se necesitan varios contactos del mismo canal y propósito, enviar `Id`.

Mapeo recomendado del formulario natural:

| Campo | Canal | Propósito |
|---|---|---|
| `correoElectronico` | `EMAIL` | `NOTIFICACIONES` |
| `telefonoCelular` | `TEL_MOVIL` | `COMERCIAL` |
| `otroContacto` | `TEL_FIJO` | `COMERCIAL` |
| `extensionTelefonica` | — | `Extension` del contacto `TEL_FIJO` |
| `telefonoContactoEmergencia` | `TEL_MOVIL` | `EMERGENCIA` |
| `correoNotificacionesTesoreria` | `EMAIL` | dominio 3 |

## 9. `ActividadEconomicaProveedor`

- `ActividadEconomicaId` debe existir, estar activo y ser de tipo CIIU.
- La clave es `ProveedorId + ActividadEconomicaId`.
- No se eliminan actividades omitidas.
- La actividad principal va aparte en `ACTIVIDAD_PRINCIPAL`.

## 10. Dirección DIAN

`DIRECCIÓN` guarda un objeto con la nomenclatura DIAN:

```json
{
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
```

- Obligatorios: `DepartamentoId`, `CiudadId`, `ViaId`, `DetalleVia`,
  `NumeroCruce`, `PlacaPuerta`, `Direccion`.
- Opcionales: `PaisId`, `InteriorId`, `Via`, `Interior`, `DetalleInterior`.
- El MID valida que los Ids de lugar existan y estén activos.
- `LUGAR_RESIDENCIA` debe coincidir con `DIRECCIÓN.CiudadId`.
- La dirección vieja `{city, address, country, department}` no se usa en este
  flujo.

## 11. Reglas de actualización

- `POST` crea el proveedor si no existe y lo reutiliza si existe.
- `PUT` responde `404` si el proveedor no existe.
- El proveedor existente debe ser de tipo natural; si no, `409`.
- Si hay más de un proveedor activo para el tercero, `409`.
- Los registros inactivos de campos normales responden `409`; las opciones se
  reactivan.
- `Consentimiento` se crea en `true` y se conserva salvo que se envíe.
- `DescripcionPortafolio` solo se actualiza si se envía.
- `Activo` se conserva.
- Los objetos `Dato` se mezclan por clave; los escalares se reemplazan.
  Para `DIRECCIÓN` se debe enviar el objeto completo: la validación DIAN exige
  las claves obligatorias antes de mezclar.
- Un código desconocido responde `400`; no cae en `DESCRIPCIÓN`.
- Un fallo de una sección no impide las demás: se responde `206` con
  `MetaData.Errors` y los IDs ya materializados.

## 12. Códigos de respuesta

| Código | Caso |
|---|---|
| `201` | Al menos una creación |
| `200` | Solo actualizaciones o reutilización |
| `206` | Fallo parcial |
| `400` | Validación |
| `404` | Tercero o proveedor inexistente en `PUT` |
| `409` | Inactivo, tipo incorrecto, múltiples proveedores o conflicto |
| `500` | Catálogo no configurado |
| `502/504` | Dependencia caída o timeout |

## 13. Ejemplo completo

```json
{
  "TerceroId": 320521,
  "InfoComplementariaTercero": [
    { "CodigoInfoComplementaria": "MASCULINO", "CodigoGrupoInfoComplementaria": "Grupo_6" },
    { "CodigoInfoComplementaria": "SOLTERO", "CodigoGrupoInfoComplementaria": "Grupo_2" },
    { "CodigoInfoComplementaria": "NO APLICA", "CodigoGrupoInfoComplementaria": "Grupo_3" },
    { "CodigoInfoComplementaria": "CISGENERO", "CodigoGrupoInfoComplementaria": "IDENTIDAD_GENERO" },
    { "CodigoInfoComplementaria": "HETEROSEXUAL", "CodigoGrupoInfoComplementaria": "ORIENTACION_SEXUAL" },
    { "CodigoInfoComplementaria": "NO APLICA", "CodigoGrupoInfoComplementaria": "Grupo_1" },
    { "CodigoInfoComplementaria": "DECLARANTE_RENTA", "CodigoGrupoInfoComplementaria": "Grupo_9", "Dato": false },
    { "CodigoInfoComplementaria": "CUENTA_AFC", "CodigoGrupoInfoComplementaria": "Grupo_9", "Dato": false },
    { "CodigoInfoComplementaria": "MIGRANTE", "CodigoGrupoInfoComplementaria": "Grupo_9", "Dato": false },
    { "CodigoInfoComplementaria": "VICTIMA_CONFLICTO", "CodigoGrupoInfoComplementaria": "Grupo_9", "Dato": false },
    { "CodigoInfoComplementaria": "HIJOS", "CodigoGrupoInfoComplementaria": "Grupo_9", "Dato": true },
    { "CodigoInfoComplementaria": "NUMERO_HIJOS", "CodigoGrupoInfoComplementaria": "Grupo_9", "Dato": 1 },
    { "CodigoInfoComplementaria": "CABEZA_FAMILIA", "CodigoGrupoInfoComplementaria": "Grupo_9", "Dato": false },
    { "CodigoInfoComplementaria": "PERSONA_PEP", "CodigoGrupoInfoComplementaria": "Grupo_9", "Dato": false },
    { "CodigoInfoComplementaria": "PERSONAS_A_CARGO", "CodigoGrupoInfoComplementaria": "Grupo_9", "Dato": true },
    { "CodigoInfoComplementaria": "EPS", "CodigoGrupoInfoComplementaria": "V", "Dato": 429 },
    { "CodigoInfoComplementaria": "SITIO_WEB", "CodigoGrupoInfoComplementaria": "PROVEEDOR_NATURAL", "Dato": "https://juanprueba.example" },
    { "CodigoInfoComplementaria": "AUTORIZACION_GIRO", "CodigoGrupoInfoComplementaria": "PROVEEDOR_NATURAL", "Dato": true },
    { "CodigoInfoComplementaria": "APLICA_RUP", "CodigoGrupoInfoComplementaria": "PROVEEDOR_NATURAL", "Dato": false },
    { "CodigoInfoComplementaria": "ACTIVIDAD_PRINCIPAL", "CodigoGrupoInfoComplementaria": "PROVEEDOR_NATURAL", "Dato": 7778 },
    { "CodigoInfoComplementaria": "AFP", "CodigoGrupoInfoComplementaria": "PROVEEDOR_NATURAL", "Dato": 7353 },
    { "CodigoInfoComplementaria": "CCF", "CodigoGrupoInfoComplementaria": "PROVEEDOR_NATURAL", "Dato": 7358 },
    { "CodigoInfoComplementaria": "PAIS_EXPEDICION", "CodigoGrupoInfoComplementaria": "PROVEEDOR_NATURAL", "Dato": 48 },
    { "CodigoInfoComplementaria": "DEPTO_EXPEDICION", "CodigoGrupoInfoComplementaria": "PROVEEDOR_NATURAL", "Dato": 260 },
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
    },
    { "CodigoInfoComplementaria": "LUGAR_RESIDENCIA", "CodigoGrupoInfoComplementaria": "Grupo_10", "Dato": 824 }
  ],
  "ProveedorNatural": {
    "PaisNacimientoId": 48,
    "MedicinaPrepagada": true,
    "ResponsableIva": true,
    "ResideColombia": true,
    "PensionVoluntaria": false,
    "Pensionado": false,
    "ExperienciaLaboral": 60,
    "ExperienciaProfesional": 36
  },
  "PerfilProveedor": [{ "CodigoPerfil": "PROFESIONAL" }],
  "Contacto": [
    {
      "TipoCanalContactoId": 8419,
      "Proposito": "NOTIFICACIONES",
      "ValorEspecifico": "juan.prueba@example.com",
      "NombreContacto": "Juan Prueba"
    }
  ],
  "ActividadEconomicaProveedor": [{ "ActividadEconomicaId": 7778 }],
  "DescripcionPortafolio": "Servicios de consultoría",
  "Consentimiento": true
}
```

## 14. Notas de implementación

- El grupo `PROVEEDOR_NATURAL` y sus filas se crean en `terceros_crud` por API;
  no se modifica su código.
- Los Ids de los parámetros (`EPS`, `AFP`, `CCF`, CIIU) y de los lugares se
  resuelven contra el catálogo del entorno; no se hardcodean.
- El flujo natural ya no escribe la extensión `DESCRIPCIÓN`. Las claves viejas
  que queden allí (`GENERO`, `AFP`, `CCF`, etc.) son legado y no se leen; se
  pueden ignorar o limpiar con un script de migración por tercero.
- El MF debe enviar `CodigoGrupoInfoComplementaria` en todos los ítems que usan
  `CodigoInfoComplementaria`.

## 15. Auditoría de modificación

Todas las escrituras del MID contra `gestion_proveedores_crud` envían
`UsuarioModificacionId` con el `TerceroId` del proveedor:

| Tabla | Valor |
|---|---|
| `proveedor` | `TerceroId` del proveedor |
| `proveedor_natural` | `TerceroId` del proveedor |
| `contacto` | `TerceroId` del proveedor |
| `actividad_economica_proveedor` | `TerceroId` del proveedor |
| `perfil_proveedor` | `TerceroId` del proveedor |

El mismo criterio se aplica en la cuenta bancaria (dominio 3) y en la
representación (dominio 2 jurídico).

Reglas:

- El valor se envía tanto en `POST` como en `PUT`. El CRUD no lo llena solo y,
  en los `PUT`, lo sobrescribe con el valor recibido.
- El cliente no envía `UsuarioModificacionId`; el MID lo administra.
- En autoregistro, el actor es el mismo proveedor. En una fase con autenticación,
  el valor deberá salir del usuario autenticado y el `TerceroId` del proveedor
  quedará como respaldo.
- Los registros históricos que quedaron en `0` se actualizan en la primera
  escritura del MID.
- `info_complementaria_tercero` vive en `terceros_crud` y no usa este campo de
  `gestion_proveedores_crud`.
