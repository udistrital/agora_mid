# caracterizacion-proveedor-natural Specification

## Purpose

Expone los endpoints del **dominio 2** para registrar y actualizar parcialmente
la caracterización de una persona natural:

```text
POST /v1/proveedores/natural/caracterizacion
PUT  /v1/proveedores/natural/caracterizacion/{terceroId}
PUT  /v1/proveedores/natural/caracterizacion/proveedor/{proveedorId}
```

El spec complementa `registro-proveedor-natural` (dominio 1). No crea el
tercero ni su identificación: los recibe ya materializados en `terceros_crud`.
Tampoco administra la cuenta bancaria (dominio 3) ni los documentos.

## Alcance

| Incluido | Detalle |
|---|---|
| `POST /v1/proveedores/natural/caracterizacion` | Crea o reutiliza el proveedor y escribe las secciones enviadas. |
| `PUT /v1/proveedores/natural/caracterizacion/{terceroId}` | Actualiza por tercero; exige proveedor existente. |
| `PUT /v1/proveedores/natural/caracterizacion/proveedor/{proveedorId}` | Actualiza por proveedor; evita resolver por tercero. |
| `InfoComplementariaTercero` | Caracterización por `(código, grupo)`; opciones de selección única. |
| `ProveedorNatural` | Datos propios de `proveedor_natural`. |
| `PerfilProveedor` | Perfil por `CodigoPerfil` contra `Grupo_5`. |
| `Contacto` | Contactos del proveedor por canal y propósito. |
| `ActividadEconomicaProveedor` | Actividades CIIU. |
| `DescripcionPortafolio` y `Consentimiento` | Campos de `proveedor`. |
| Dirección DIAN | Objeto `DIRECCIÓN` y `LUGAR_RESIDENCIA`. |
| Actualización parcial | Solo escribe lo enviado; no borra lo omitido. |
| Fallo parcial | `206` con `MetaData.Errors` e Ids ya materializados. |
| Catálogo `PROVEEDOR_NATURAL` | Grupo y filas creadas en `terceros_crud` por API. |

| Excluido | Motivo |
|---|---|
| `tercero` y `datos_identificacion` | Dominio 1 (`registro-proveedor-natural`). |
| `cuenta_bancaria` y contacto de pago | Dominio 3. |
| Documentos (renta, RUT, RUP) | Gestor documental / MF. |
| Autenticación, ownership y estado del proveedor | Fases posteriores. |
| `PATCH` y `DELETE` | No se usan. |
| Swagger / OpenAPI | Se deja para una etapa posterior. |

## Payload de referencia

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

## Requirements

### Requirement: Contrato de los endpoints

El sistema SHALL exponer `POST /v1/proveedores/natural/caracterizacion`,
`PUT /v1/proveedores/natural/caracterizacion/{terceroId}` y
`PUT /v1/proveedores/natural/caracterizacion/proveedor/{proveedorId}`.

El controlador SHALL:

- Leer y decodificar el JSON.
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
exigir que sea de tipo natural (`MID_TIPO_PROVEEDOR_NATURAL`, por defecto `1`).

El MID SHALL responder `409` cuando:

- El proveedor existente es de otro tipo.
- El proveedor está inactivo.
- Hay más de un proveedor activo para el tercero.

#### Scenario: Proveedor reutilizado

- **WHEN** el tercero ya tiene un proveedor natural activo
- **THEN** el `POST` reutiliza ese proveedor
- **AND** no crea un segundo registro

#### Scenario: Tipo incorrecto

- **WHEN** el proveedor del tercero no es natural
- **THEN** el MID responde `409`

#### Scenario: Proveedores duplicados

- **WHEN** hay más de un proveedor activo para el tercero
- **THEN** el MID responde `409`
- **AND** no elige uno de forma arbitraria

### Requirement: Info complementaria por código y grupo

Cada ítem de `InfoComplementariaTercero` SHALL traer exactamente uno de:

- `InfoComplementariaId` (compatibilidad), o
- `CodigoInfoComplementaria` + `CodigoGrupoInfoComplementaria`.

Cuando se usa `CodigoInfoComplementaria`, el grupo SHALL ser obligatorio. Los
Ids cambian entre ambientes; el par `(código, grupo)` es estable.

El MID SHALL resolver el par contra `terceros_crud` y usar el Id local
internamente. La respuesta SHALL incluir `CodigoInfoComplementaria` y
`CodigoGrupoInfoComplementaria`.

El flujo natural SHALL NOT usar la extensión `DESCRIPCIÓN`. Un código
inexistente o un grupo que no coincide SHALL responder `400`.

#### Scenario: Código inexistente

- **WHEN** llega `CodigoInfoComplementaria = "NO_EXISTE"` con un grupo
- **THEN** el MID responde `400`
- **AND** no guarda el valor en `DESCRIPCIÓN`

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

#### Scenario: `Dato` en campos normales

- **WHEN** el ítem es un campo normal y no trae `Dato`
- **THEN** el MID responde `400`
- **AND** valida el tipo del `Dato` según el código

### Requirement: Selectores de opción única

Los siguientes grupos SHALL tratarse como selección única:

| Grupo | Campo |
|---|---|
| `Grupo_1` | `discapacidad` |
| `Grupo_2` | `estadoCivil` |
| `Grupo_3` | `grupoEtnico` |
| `Grupo_6` | `genero` |
| `IDENTIDAD_GENERO` | `seReconoceComo` |
| `ORIENTACION_SEXUAL` | `orientacionSexual` |

Para estos campos el `Dato` SHALL ir vacío o ausente. El MID SHALL:

- Aceptar una sola opción por grupo por petición.
- Desactivar las demás filas activas del mismo grupo para el tercero.
- No desactivar las filas `DOC_DISCAPACIDAD` ni `DOC_POBLACION`.
- Crear, reutilizar o reactivar la fila de la opción seleccionada.
- Rechazar con `400` una opción inactiva en el catálogo.

#### Scenario: Cambio de opción

- **WHEN** el tercero tiene `MASCULINO` activo y llega `FEMENINO`
- **THEN** `MASCULINO` queda `Activo=false`
- **AND** `FEMENINO` queda activo
- **AND** `Femenino` se crea si no existía o se reactiva si estaba inactivo

#### Scenario: Repetición idempotente

- **WHEN** llega la misma opción que ya está activa
- **THEN** el MID reutiliza la fila
- **AND** no crea duplicados

#### Scenario: Dos opciones del mismo grupo

- **WHEN** llegan dos opciones del mismo grupo en la misma petición
- **THEN** el MID responde `400`

#### Scenario: `Dato` en una opción

- **WHEN** una opción llega con un `Dato` distinto de vacío o `null`
- **THEN** el MID responde `400`

### Requirement: `ProveedorNatural`

El MID SHALL escribir los campos de `proveedor_natural` que lleguen. En
actualización, los campos ausentes SHALL conservar su valor.

| Campo | Regla |
|---|---|
| `PaisNacimientoId` | Lugar activo en `ubicaciones_crud`; obligatorio al crear el detalle |
| `Cargo` | Máximo 150 caracteres, sin caracteres de control |
| `MedicinaPrepagada`, `ResponsableIva`, `ResideColombia`, `PensionVoluntaria`, `Pensionado`, `Dependiente*` | Booleanos |
| `ValorUvtPrepagada`, `ValorAfc`, `ValorPensionVoluntaria` | `>= 0` |
| `InteresViviendaAfc` | Entre 0 y 1 |
| `ExperienciaLaboral`, `ExperienciaProfesional` | Enteros `>= 0` |

#### Scenario: Creación sin país de nacimiento

- **WHEN** se crea el detalle de `proveedor_natural` sin `PaisNacimientoId`
- **THEN** el MID responde `400`

#### Scenario: Actualización parcial

- **WHEN** llega solo `ExperienciaLaboral`
- **THEN** el MID actualiza ese campo
- **AND** conserva `PaisNacimientoId` y los demás campos existentes

### Requirement: `PerfilProveedor`

El perfil SHALL enviarse por `CodigoPerfil` y resolverse contra `Grupo_5`.
`PerfilId` se conserva por compatibilidad; si llegan ambos, SHALL coincidir.

El MID SHALL responder `400` cuando el código no existe, está inactivo, no
pertenece a `Grupo_5` o el perfil está repetido.

La clave SHALL ser `ProveedorId + PerfilId`. Los perfiles omitidos SHALL NOT
eliminarse.

#### Scenario: Perfil por código

- **WHEN** llega `CodigoPerfil = "PROFESIONAL"`
- **THEN** el MID resuelve el Id local de `Grupo_5`
- **AND** escribe `perfil_proveedor` con ese Id

#### Scenario: Perfil fuera del grupo

- **WHEN** el código resuelve a un registro de otro grupo
- **THEN** el MID responde `400`

#### Scenario: Perfil repetido

- **WHEN** el mismo perfil llega dos veces en la petición
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
(`MID_TIPO_PARAMETRO_CIIU`, por defecto `CIIU`).

La clave SHALL ser `ProveedorId + ActividadEconomicaId`. Las actividades
omitidas SHALL NOT eliminarse. La actividad principal SHALL ir aparte en
`ACTIVIDAD_PRINCIPAL`.

#### Scenario: Actividad no CIIU

- **WHEN** el Id no pertenece al tipo CIIU
- **THEN** el MID responde `400`

#### Scenario: Actividad repetida

- **WHEN** la misma actividad llega dos veces
- **THEN** el MID responde `400`

#### Scenario: Actividad principal

- **WHEN** llega `ACTIVIDAD_PRINCIPAL` con un Id CIIU
- **THEN** el MID lo valida contra el tipo CIIU
- **AND** lo guarda en el grupo `PROVEEDOR_NATURAL`

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

`LUGAR_RESIDENCIA` SHALL ser un Id de lugar y SHALL coincidir con
`DIRECCIÓN.CiudadId` cuando ambos se envíen.

La dirección SHALL enviarse completa en cada actualización; el MID valida las
claves obligatorias antes de mezclar.

#### Scenario: Dirección incompleta

- **WHEN** falta una clave obligatoria
- **THEN** el MID responde `400`

#### Scenario: Lugar inactivo

- **WHEN** un Id de la dirección no existe o está inactivo
- **THEN** el MID responde `400`

#### Scenario: Ciudad incoherente

- **WHEN** `DIRECCIÓN.CiudadId` y `LUGAR_RESIDENCIA` difieren
- **THEN** el MID responde `400`

#### Scenario: Dirección válida

- **WHEN** el objeto trae todas las claves y lugares válidos
- **THEN** el MID lo guarda en `DIRECCIÓN`
- **AND** guarda `LUGAR_RESIDENCIA` como Id de ciudad

### Requirement: Códigos de los campos del grupo `PROVEEDOR_NATURAL`

El MID SHALL resolver los siguientes códigos dentro del grupo
`PROVEEDOR_NATURAL` de `terceros_crud`:

| Código | Nombre | `Dato` |
|---|---|---|
| `SITIO_WEB` | Sitio web o portafolio | string http/https |
| `AUTORIZACION_GIRO` | Autorización de giro | booleano |
| `APLICA_RUP` | Aplica RUP | booleano |
| `ACTIVIDAD_PRINCIPAL` | Actividad económica principal | Id CIIU |
| `AFP` | Administradora de fondos de pensiones | Id `EFP` |
| `CCF` | Caja de compensación familiar | Id `CCF` |
| `PAIS_EXPEDICION` | País de expedición | Id de lugar |
| `DEPTO_EXPEDICION` | Departamento de expedición | Id de lugar |

`DEPTO_EXPEDICION` SHALL usarse porque `codigo_abreviacion` es `varchar(20)`.
`DEPARTAMENTO_EXPEDICION` se acepta por compatibilidad, pero no existe como
código en el catálogo.

#### Scenario: `SITIO_WEB` inválido

- **WHEN** el valor no es una URL http/https
- **THEN** el MID responde `400`

#### Scenario: `AFP` o `CCF` de otro tipo

- **WHEN** el Id no pertenece al tipo `EFP` o `CCF`
- **THEN** el MID responde `400`

#### Scenario: `DEPTO_EXPEDICION`

- **WHEN** el MF envía `DEPTO_EXPEDICION`
- **THEN** el MID lo resuelve en `PROVEEDOR_NATURAL`
- **AND** valida el Id contra `ubicaciones_crud`

### Requirement: Fuente de verdad y campos que no se duplican

El contrato del MF SHALL usar `proveedor_natural` como fuente de verdad para:

- `PaisNacimientoId`
- `ExperienciaLaboral`
- `ExperienciaProfesional`
- `MedicinaPrepagada`
- `ResponsableIva`
- `ResideColombia`
- `PensionVoluntaria`
- `Pensionado`

El MF SHALL usar `proveedor` para `DescripcionPortafolio` y `Consentimiento`,
`perfil_proveedor` para el perfil, `contacto` para los contactos y
`actividad_economica_proveedor` para las actividades.

El MF SHALL NOT enviar `MED_PREPAGADA`, `RESPONSABLE_IVA`, `RESIDE_FIS_COL` ni
`PENSION_VOL` dentro de `InfoComplementariaTercero`; esos valores viajan en
`ProveedorNatural`. Si llegaran, el MID los trataría como un campo normal, pero
no serían la fuente de verdad.

Para `discapacidad`, el MF SHALL enviar la opción del `Grupo_1`; SHALL NOT usar
la fila `DISCAPACIDAD` (303) de `INFO_BASICA`.

#### Scenario: Construcción de los campos tributarios

- **WHEN** el MF arma la caracterización
- **THEN** envía `MedicinaPrepagada`, `ResponsableIva`, `ResideColombia` y
  `PensionVoluntaria` en `ProveedorNatural`
- **AND** omite `MED_PREPAGADA`, `RESPONSABLE_IVA`, `RESIDE_FIS_COL` y
  `PENSION_VOL` de `InfoComplementariaTercero`

#### Scenario: Discapacidad

- **WHEN** el MF envía una opción de `Grupo_1`
- **THEN** el MID la guarda como fila de opción
- **AND** el MF no usa la fila 303 de `INFO_BASICA`

### Requirement: Actualización parcial y fallo parcial

Cada llamada SHALL escribir solo las secciones que llegan. Los campos omitidos
SHALL NOT borrarse ni sobrescribirse. Los objetos `Dato` SHALL mezclarse por
clave; los escalares SHALL reemplazarse.

Un fallo en una sección SHALL NOT impedir las demás. El MID SHALL responder
`206` con `MetaData.Errors` y los Ids ya materializados.

Un aviso `ambiguous_match` SHALL NOT marcar la respuesta como parcial ni como
fallo.

#### Scenario: Actualización de una sola sección

- **WHEN** llega solo `ProveedorNatural`
- **THEN** el MID actualiza esa sección
- **AND** conserva `InfoComplementariaTercero`, `PerfilProveedor`, `Contacto` y
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
crear duplicados de proveedor, `proveedor_natural`, perfil, contacto,
actividad, opción o fila de información complementaria.

#### Scenario: Segundo POST idéntico

- **WHEN** se repite el mismo `POST`
- **THEN** el MID responde `200`
- **AND** todas las entidades aparecen como `Reutilizados`
- **AND** no se crean filas nuevas

#### Scenario: Misma opción

- **WHEN** se repite la misma opción de un grupo
- **THEN** el MID reutiliza la fila activa
- **AND** no la duplica

### Requirement: Códigos de respuesta

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

#### Scenario: Validación

- **WHEN** el payload no cumple el contrato
- **THEN** el MID responde `400`
- **AND** no escribe en los CRUD

#### Scenario: Conflicto de proveedor

- **WHEN** el proveedor es de otro tipo, está inactivo o hay duplicados
- **THEN** el MID responde `409`

### Requirement: Configuración

El MID SHALL usar:

- `MID_TIPO_PROVEEDOR_NATURAL` (por defecto `1`).
- `MID_TIPO_PARAMETRO_CIIU` (por defecto `CIIU`).
- `MID_TIPO_PARAMETRO_CANAL_CONTACTO` (por defecto vacío; en el entorno actual
  `MED_COMUNICACION`).
- `TERCEROS_CRUD_SERVICE`, `GESTION_PROVEEDORES_CRUD_SERVICE`,
  `PARAMETROS_CRUD_SERVICE` y `UBICACIONES_CRUD_SERVICE`.

Los Ids de `EPS`, `AFP`, `CCF`, CIIU, canales y lugares SHALL resolverse en
tiempo de ejecución contra el catálogo del entorno. No SHALL hardcodearse.

#### Scenario: Canal de contacto

- **WHEN** el MID valida un contacto
- **THEN** usa `MID_TIPO_PARAMETRO_CANAL_CONTACTO`
- **AND** exige que el parámetro pertenezca a ese tipo

#### Scenario: Ids de parámetros

- **WHEN** el MF envía un Id de EPS, AFP, CCF o CIIU
- **THEN** el MID lo valida contra el `parametros_crud` del entorno

### Requirement: Auditoría de modificación

Toda escritura del MID contra `gestion_proveedores_crud` SHALL enviar
`UsuarioModificacionId` con el `TerceroId` del proveedor, tanto en `POST` como
en `PUT`. Aplica a `proveedor`, `proveedor_natural`, `contacto`,
`actividad_economica_proveedor` y `perfil_proveedor`.

El cliente SHALL NOT enviar `UsuarioModificacionId`; el MID lo administra.

El CRUD no completa el campo por sí solo y, en los `PUT`, lo sobrescribe con el
valor recibido. Si un registro histórico tiene `0`, el MID lo actualiza en la
primera escritura.

#### Scenario: Alta con auditoría

- **WHEN** el MID crea un proveedor natural y sus secciones
- **THEN** todas las filas de `gestion_proveedores_crud` quedan con
  `UsuarioModificacionId = TerceroId`

#### Scenario: Actualización con auditoría

- **WHEN** el MID actualiza un registro existente con `UsuarioModificacionId = 0`
- **THEN** la fila queda con `UsuarioModificacionId = TerceroId`
- **AND** la operación conserva la idempotencia si el mismo tercero repite la
  petición

#### Scenario: Campo enviado por el cliente

- **WHEN** el cliente intenta enviar `UsuarioModificacionId`
- **THEN** el MID lo rechaza porque es un campo administrado

## Non-Goals

- No se implementa el dominio 1 (`tercero`, `datos_identificacion`).
- No se implementa la cuenta bancaria (dominio 3).
- No se gestionan documentos.
- No se implementa autenticación, ownership ni estado del proveedor.
- No se implementan `PATCH` ni `DELETE`.
- No se actualiza Swagger/OpenAPI en este cambio.
- No se modifica el MF.
- No se modifica el código de `terceros_crud`; solo se hacen peticiones HTTP
  para crear el grupo y las filas del catálogo.

## Notas

- El spec complementa `registro-proveedor-natural` (dominio 1).
- El grupo `PROVEEDOR_NATURAL` y sus 8 filas se crean por API en
  `terceros_crud`. En pruebasapi quedaron con Ids de grupo/fila propios del
  entorno; el contrato usa los códigos, no los Ids.
- `MIGRANTE` tiene dos filas activas en `Grupo_9` (444 y 452). El MID usa la de
  menor Id y emite `ambiguous_match`; es una condición conocida del catálogo.
- El flujo nuevo ya no escribe en `DESCRIPCIÓN`. Las claves viejas de
  `DESCRIPCIÓN` son legado y no se leen.
- La fila `DISCAPACIDAD` (303) de `INFO_BASICA` es legado; el flujo nuevo usa
  las opciones del `Grupo_1`.
- El E2E se ejecutó con `TerceroId=320521` y `ProveedorId=1`: `POST` `201`,
  segundo `POST` `200` idempotente, 26 filas de información complementaria, 4
  contactos, 2 actividades, perfil por código y dirección DIAN.
- Los Ids de `InfoComplementariaTercero` que aparecen en la respuesta son del
  entorno; el consumidor no debe usarlos como contrato.
