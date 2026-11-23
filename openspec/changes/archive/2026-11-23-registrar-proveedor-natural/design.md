# Design

## Context

Ver `proposal.md` - Why. El middleware tiene `services/terceros.go` y
`services/gestion_proveedores.go` con el contrato
`(ctx, endpoint[, payload], target) (ok, statusCode, err)`, y las utilidades
compartidas en `services/http_utils.go`. No hay controladores ni rutas
públicas: `routers/router.go` solo declara el namespace `/v1` vacío. Este
cambio añade el primer controlador real.

Contratos externos (en `docs/contratos/`):

- `terceros_crud` — `models.DatosIdentificacion`: `Numero` (string),
  `TipoDocumentoId` (objeto `TipoDocumento`), `CiudadExpedicion` (int64),
  `FechaExpedicion` (datetime), `Activo` (bool), `Id` (int64),
  `TerceroId` (objeto `Tercero` anidado). Confirma que `TerceroId.Id` es la
  persona y el `Id` del registro es la identificación.
- `terceros_crud` — `models.Tercero`: `PrimerNombre`, `SegundoNombre`,
  `PrimerApellido`, `SegundoApellido`, `FechaNacimiento` (datetime),
  `Activo`, `Id`, `NombreCompleto`.
- `gestion_proveedores_crud` — `models.Proveedor`: `TerceroId` (int64 plano),
  `TipoProveedor` (int64), `Activo` (bool), `Id` (int64). Endpoints `/proveedor/`
  (GetAll, Post) y `/proveedor/{id}` (GetOne, Put).

Ambos CRUD son Beego con el filtro estándar `query=col1:v1,col2:v2`
(ver descripción del parámetro `query` en los swagger). `Numero` es string en
el contrato, lo que respalda conservar ceros iniciales.

## Goals / Non-Goals

**Goals:**
- Un controlador delgado (valida, propaga contexto, arma respuesta) y una capa
  de coordinación que ejecuta el flujo con los services existentes.
- Reglas deterministas de existencia/creación y estado HTTP 200/201.
- Errores públicos neutros (host de CRUD oculto) y logs por etapa.

**Non-Goals:**
- Proveedor jurídico y demás sub-recursos de proveedores.
- Transaccionalidad distribuida / rollback entre CRUD (no lo ofrecen los CRUD).
- Cambiar el contrato de los services.

## Decisions

### Ubicación de la coordinación
El controlador Beego (`controllers/`) queda delgado: decodifica y valida la
petición, pasa `ctx` a una función/paquete de coordinación, y traduce el
resultado al envoltorio del middleware. La coordinación (p. ej.
`services/registro_proveedor_natural.go` o un paquete dedicado) concentra el
orden de llamadas. Decisión de ubicación exacta a confirmar en Apply según la
convención que se adopte para orquestadores; el spec no depende de ella.

### Filtros de la consulta de identificación
`GetAll` sobre `datos_identificacion` con
`query=Numero:<numero>,TipoDocumentoId:<tipo>,Activo:true` (sintaxis estándar
del CRUD). El `target` es un `*[]map[string]any` para poder distinguir `[]`,
`[{}]` y un objeto no vacío. De un objeto no vacío se leen `Id` (identificación)
y `TerceroId.Id` (persona).

### Interpretación de la lista (regla del usuario)
- `[]` o `[{}]` → ausencia de identificación activa.
- primer elemento objeto no vacío → coincidencia; se usa la primera.
- varias → warning con filtros, cantidad e `Id` seleccionado, enmascarando el
  número de documento según la convención del proyecto.

### Construcción de payloads de actualización
Se parte del registro recuperado (no de cero) y se reemplazan solo los campos
enviados, conservando `Id`, relaciones y campos requeridos. Para el tercero
(rama A/C) se reemplaza únicamente `FechaNacimiento`, conservando nombres y
apellidos del tercero recuperado.

### Estado HTTP
Se lleva un marcador de "hubo al menos una creación" (identificación, tercero
o proveedor creados vía `Post`). 201 si el marcador es true al final; 200 si
todas fueron lecturas/actualizaciones. Un fallo posterior a una escritura no
produce 201: se devuelve el error del middleware.

### Contratos usados (resumen)

| Paso | CRUD | Operación | Endpoint | Filtros / payload |
|---|---|---|---|---|
| Buscar identificación | terceros | GetAll | `datos_identificacion` | `query=Numero:..,TipoDocumentoId:..,Activo:true` |
| Actualizar identificación | terceros | Put | `datos_identificacion/{id}` | registro recuperado + campos enviados |
| Crear identificación | terceros | Post | `datos_identificacion` | `DatosIdentificacion` + `TerceroId` |
| Leer tercero | terceros | GetOne | `tercero/{id}` | — |
| Actualizar tercero | terceros | Put | `tercero/{id}` | recuperado + `FechaNacimiento` |
| Crear tercero | terceros | Post | `tercero` | `DatosBasicos` |
| Buscar proveedor | proveedores | GetAll | `proveedor` | `query=TerceroId:..,TipoProveedor:..,Activo:true` |
| Crear proveedor | proveedores | Post | `proveedor` | `TerceroId`, `TipoProveedor`, `Activo` |

### Decisión sobre ramas B vs C (investigación)
La petición de ejemplo no trae `TerceroId`, y `[{}]` tampoco lo aporta. Busqué
una fuente confiable en el repositorio:

- `docs/contratos/terceros_crud/swagger.yml` expone
  `GET /tercero/identificacion?query=...&documento=...` (y
  `GET /tercero/identificacion/{id}`), que devuelve
  `models.DatosIdentificacionTercero { compuesto: string, tercero_id: int64 }`.
  Su descripción dice que consulta "en las tablas tercero y
  datos_identificacion". Es el único mecanismo documentado que podría devolver
  un `tercero_id` a partir del documento.

Limitación: el contrato **no documenta** si ese endpoint devuelve `tercero_id`
cuando el tercero existe pero **no tiene identificación activa** (la consulta
une ambas tablas y no especifica el filtro de `Activo`). Por tanto no puedo
afirmar que sea una fuente confiable para separar la rama C (tercero sin
identificación activa) de la rama B (crear tercero). No hay en el repositorio
otra clave de búsqueda documentada para localizar un tercero sin su
identificación activa.

Consecuencia para el diseño: sin una fuente confiable confirmada, el flujo
**no puede** decidir de forma determinista entre B y C a partir de `[]`/`[{}]`.
Esto queda como pregunta de negocio (abajo). El spec ya prohíbe deducir
identidad por nombres y operar con `Id = 0`, y trata una consulta fallida o
referencia incompleta como error.

## Ejemplos de respuesta

### 200 (todos existían; se actualizaron identificación y fecha de nacimiento)
```json
{
  "Status": 200,
  "Message": "Registro procesado",
  "Data": {
    "TipoProveedor": 1,
    "DatosBasicos": {
      "Id": 55,
      "PrimerNombre": "Carlos",
      "SegundoNombre": "Andrés",
      "PrimerApellido": "González",
      "SegundoApellido": "Pérez",
      "FechaNacimiento": "1990-05-15T00:00:00Z",
      "Activo": true
    },
    "DatosIdentificacion": {
      "Id": 88,
      "Numero": "1023456789",
      "TipoDocumentoId": 1,
      "CiudadExpedicion": 123,
      "FechaExpedicion": "2008-06-20T00:00:00Z",
      "TerceroId": { "Id": 55 },
      "Activo": true
    },
    "ProveedorId": 42
  }
}
```

### 201 (al menos una creación; aquí tercero, identificación y proveedor nuevos)
```json
{
  "Status": 201,
  "Message": "Registro creado",
  "Data": {
    "TipoProveedor": 1,
    "DatosBasicos": {
      "Id": 101,
      "PrimerNombre": "Carlos",
      "SegundoNombre": "Andrés",
      "PrimerApellido": "González",
      "SegundoApellido": "Pérez",
      "FechaNacimiento": "1990-05-15T00:00:00Z",
      "Activo": true
    },
    "DatosIdentificacion": {
      "Id": 140,
      "Numero": "1023456789",
      "TipoDocumentoId": 1,
      "CiudadExpedicion": 123,
      "FechaExpedicion": "2008-06-20T00:00:00Z",
      "TerceroId": { "Id": 101 },
      "Activo": true
    },
    "ProveedorId": 77
  }
}
```

El tipo exacto de `Status` (entero o string) y el texto de `Message` siguen la
convención del envoltorio del middleware; se fija en Apply al mirar el helper
de respuesta existente.

## Risks / Trade-offs

- [No hay transaccionalidad entre CRUD] → Un fallo tras un alta deja registros
  creados. Mitigación: no se reintenta automáticamente, se registra la etapa y
  se devuelve el error; la limpieza/compensación queda fuera de alcance.
- [Separar B de C sin fuente confiable] → Riesgo de crear terceros duplicados o
  de bloquear altas legítimas. Mitigación: se bloquea la decisión hasta
  resolver la pregunta de negocio; no se deduce identidad por nombres.
- [Formato real de los filtros/relaciones] → Si el CRUD espera la relación
  anidada (`TerceroId` objeto) en escritura, el payload debe respetarlo.
  Mitigación: construir payloads a partir del registro recuperado y validar
  contra los servidores `httptest` en las pruebas.

## Decisión de negocio (resuelta)

La fuente de verdad para resolver el tercero es `datos_identificacion` (vía
`GetAll`). No se usa `GET /tercero/identificacion`.

- Si la consulta de identificación activa devuelve `[]` o `[{}]` →
  **crear un tercero nuevo** (rama B). No se intenta localizar un tercero
  preexistente por otra clave.
- Si el primer elemento es un objeto no vacío → tomar `TerceroId.Id` de ese
  objeto y leer el tercero con `GET tercero/{id}` (rama A).

Consecuencia: la **rama C se elimina** del flujo. "Sin identificación activa"
siempre implica crear el tercero; no existe un camino de "tercero existente sin
identificación activa", porque no hay (ni se desea) una clave documentada para
localizarlo sin su identificación. La guarda de `Id = 0`/no confirmado sigue
vigente: solo se consulta/escribe con un `Id` de tercero obtenido de una
respuesta efectiva (de `TerceroId.Id` o del alta del tercero).

## Open Questions

Ninguna.
