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

## 1. Cuerpo

```json
{
  "TerceroId": 101,
  "InfoComplementariaTercero": [
    { "CodigoInfoComplementaria": "GENERO", "Dato": 12 },
    { "CodigoInfoComplementaria": "HIJOS", "Dato": true },
    { "CodigoInfoComplementaria": "NUMERO_HIJOS", "Dato": 2 },
    { "CodigoInfoComplementaria": "DIRECCIÓN", "Dato": {
        "DepartamentoId": 5,
        "CiudadId": 149,
        "Via": "Calle",
        "DetalleVia": "10-20",
        "NumeroCruce": "10",
        "PlacaPuerta": "20"
    }},
    { "CodigoInfoComplementaria": "LUGAR_RESIDENCIA", "Dato": 149 }
  ],
  "ProveedorNatural": {
    "PaisNacimientoId": 1,
    "MedicinaPrepagada": true,
    "ResponsableIva": true,
    "ResideColombia": true,
    "PensionVoluntaria": false,
    "Pensionado": false,
    "ExperienciaLaboral": 60,
    "ExperienciaProfesional": 36
  },
  "PerfilProveedor": [{ "PerfilId": 5 }],
  "Contacto": [
    {
      "TipoCanalContactoId": 1,
      "Proposito": "NOTIFICACIONES",
      "ValorEspecifico": "ana@correo.co",
      "NombreContacto": "Ana Pérez"
    }
  ],
  "ActividadEconomicaProveedor": [{ "ActividadEconomicaId": 10 }],
  "DescripcionPortafolio": "Servicios de consultoría",
  "Consentimiento": true
}
```

## 2. `InfoComplementariaTercero`

Cada entrada debe traer **exactamente uno** de:

- `InfoComplementariaId` numérico, o
- `CodigoInfoComplementaria` (código del catálogo).

`Dato` es obligatorio y acepta string, número, booleano, objeto, arreglo o
`null`.

El MID resuelve el código contra `info_complementaria`:

- Si existe y está activo, escribe una fila propia.
- Si no existe, guarda el valor en una fila de extensión bajo `DESCRIPCIÓN`, con
  el código como clave. Las actualizaciones mezclan las claves y conservan las
  que no se envían.
- Si existe pero está inactivo, rechaza con `400`.

### Códigos de campo recomendados para el MF

| Campo del MF | Código sugerido | Valor de `Dato` |
|---|---|---|
| `genero` | `GENERO` | Id de la opción de `Grupo_6` |
| `grupoEtnico` | `GRUPO_ETNICO` | Id de `Grupo_3` |
| `seReconoceComo` | `SE_RECONOCE_COMO` | Id de `IDENTIDAD_GENERO` |
| `orientacionSexual` | `ORIENTACION_SEXUAL` | Id de `ORIENTACION_SEXUAL` |
| `estadoCivil` | `ESTADO_CIVIL` | Id de `Grupo_2` |
| `discapacidad` | `DISCAPACIDAD` | Id de `Grupo_1` |
| `esMigrante` | `MIGRANTE` | booleano |
| `victimaConflictoArmado` | `VICTIMA_CONFLICTO` | booleano |
| `tieneHijos` | `HIJOS` | booleano |
| `numeroHijos` | `NUMERO_HIJOS` | entero |
| `esCabezaFamilia` | `CABEZA_FAMILIA` | booleano |
| `esPep` | `PERSONA_PEP` | booleano |
| `personasACargo` | `PERSONAS_A_CARGO` | booleano |
| `declaranteRenta` | `DECLARANTE_RENTA` | booleano |
| `cuentaAfc` | `CUENTA_AFC` | booleano |
| `eps` | `EPS` | Id de parámetro |
| `afp` | `AFP` | Id de parámetro |
| `ccf` | `CCF` | Id de parámetro |
| `sitioWebPortafolio` | `SITIO_WEB` | string |
| `autorizacionGiro` | `AUTORIZACION_GIRO` | booleano |
| `aplicaRup` | `APLICA_RUP` | booleano |
| `actividadCiiuPrincipalId` | `ACTIVIDAD_PRINCIPAL` | Id de parámetro CIIU |
| `paisExpedicion` | `PAIS_EXPEDICION` | Id de lugar |
| `departamentoExpedicion` | `DEPARTAMENTO_EXPEDICION` | Id de lugar |
| Dirección completa | `DIRECCIÓN` | objeto JSON |
| Ciudad de residencia | `LUGAR_RESIDENCIA` | Id de lugar |

Los códigos que ya existen en `terceros_crud` se escriben en su fila propia. Los
que no existen se guardan en la extensión `DESCRIPCIÓN`. No es necesario crear
catálogos nuevos para que el flujo funcione.

Para los códigos conocidos, el MID valida el tipo y el catálogo: las opciones de
género, etnia, identidad, orientación, estado civil y discapacidad se comprueban
contra su grupo; los lugares contra `ubicaciones_crud`; EPS, AFP, CCF y la
actividad principal contra `parametros_crud`; y los booleanos contra su tipo.

Si un código tiene varios registros activos, el MID usa el de menor `Id` y emite
un aviso `ambiguous_match`. Para fijar uno concreto sin tocar `terceros_crud`,
se puede configurar:

```bash
MID_INFOCOMP_DESCRIPCION=53
MID_INFOCOMP_MIGRANTE=444
MID_INFOCOMP_EPS=308
```

El mismo mecanismo `MID_INFOCOMP_<CODIGO>` sirve para los demás códigos
conocidos. Si el código no existe, el valor sigue yendo a la extensión
`DESCRIPCIÓN`.

> No uses el Id de la opción seleccionada como `InfoComplementariaId` para
> campos de valor único (género, estado civil, etc.). Usa el código del campo y
> envía el Id de la opción en `Dato`; así las actualizaciones reemplazan el valor
> sin dejar filas antiguas activas.

## 3. `ProveedorNatural`

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

## 4. `PerfilProveedor`

- `PerfilId` debe existir, estar activo y pertenecer a `Grupo_5`.
- La clave es `ProveedorId + PerfilId`.
- No se eliminan perfiles omitidos.

## 5. `Contacto`

- `TipoCanalContactoId` debe existir y estar activo en `parametros_crud`.
- `Proposito` se normaliza a mayúsculas, máximo 50, sin `,` ni `:`.
- `ValorEspecifico` máximo 100, sin `,` ni `:`.
- `NombreContacto` máximo 100; `Extension` máximo 10.
- La clave es `ProveedorId + TipoCanalContactoId + Proposito`.
- Si se necesitan varios contactos del mismo canal y propósito, enviar `Id`.

## 6. `ActividadEconomicaProveedor`

- `ActividadEconomicaId` debe existir, estar activo y ser de tipo `CIIU`.
- La clave es `ProveedorId + ActividadEconomicaId`.
- No se eliminan actividades omitidas.

## 7. Reglas de actualización

- `POST` crea el proveedor si no existe y lo reutiliza si existe.
- `PUT` responde `404` si el proveedor no existe.
- El proveedor existente debe ser de tipo natural; si no, `409`.
- Si hay más de un proveedor activo para el tercero, `409`.
- Los registros inactivos responden `409` y no se reactivan.
- `Consentimiento` se crea en `true` y se conserva salvo que se envíe.
- `DescripcionPortafolio` solo se actualiza si se envía.
- `Activo` se conserva.
- Los objetos `Dato` se mezclan por clave; los escalares se reemplazan.
- Un fallo de una sección no impide las demás: se responde `206` con
  `MetaData.Errors` y los IDs ya materializados.

## 8. Códigos de respuesta

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
