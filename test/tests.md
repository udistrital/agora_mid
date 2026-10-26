# Documentación de la suite de pruebas del subárbol del proveedor

Este documento describe las pruebas de integración y de extremo a extremo (e2e)
del middleware de escritura del proveedor, ubicadas en el paquete `test/`. Cada
prueba ejercita el stack completo **controlador → servicio → motor de escritura →
write-resolvers** contra CRUD dueños simulados, sin red real ni base de datos.

## Actores

- **Cliente del API**: envía el cuerpo anidado del subárbol por `POST`/`PATCH
  /v1/proveedor` y el autor de la modificación por la cabecera `X-Autor-Id`.
- **Middleware (MID)**: el router de `/v1/proveedor`, el `ProveedorWriteService` y
  el `WriteEngine` genérico cableados con la configuración declarativa real
  (`config/relaciones_proveedor.yaml`, cargada desde `../config`).
- **CRUD dueños simulados**: un único servidor HTTP de prueba (`CRUDSimulado`,
  en `fixtures.go`) que imita a Terceros CRUD y a Gestión de Personas CRUD.
  Responde `{"Data":{"Id":n}}` en `POST`/`PATCH` (con un Id autoincremental por
  recurso y un rango distinto por entidad) y `{"Data":[...]}` en `GET`, y captura
  cada petición (método, path, query, cuerpo) para las aserciones.

## Fixtures y mocks compartidos (`fixtures.go`)

- `CRUDSimulado` / `NuevoCRUDSimulado(t, existentes)`: servidor CRUD simulado;
  `existentes` precarga registros que una búsqueda por clave natural o por Id debe
  devolver (para idempotencia y continuación tras creación parcial).
- `routerProveedor(t, crud)`: arma el router real de `/v1/proveedor` (GET →
  lectura, POST/PATCH → escritura) sobre el servicio de escritura real cableado
  contra el CRUD simulado, cargando el YAML de producción. Equivale a
  `routerProveedorConBlacklist(t, crud, nil)`.
- `routerProveedorConBlacklist(t, crud, blacklist)`: como `routerProveedor` pero
  inyecta una lista negra global (`config.GlobalBlacklist`) en el servicio de
  lectura, para ejercitar el recorte y el rechazo por lista negra sobre la
  configuración real. Una `blacklist` nil equivale a "sin supresión". El router
  además despacha `POST /v1/proveedor/consulta` al handler de lectura por cuerpo
  (`Consultar`), replicando el enrutado del `main`.
- `ejecutar(t, router, metodo, cuerpo)` / `ejecutarConsulta(t, router, id, fields)`:
  envían, respectivamente, una petición con cuerpo a `POST`/`PATCH /v1/proveedor`
  o un `POST /v1/proveedor/consulta` con cuerpo JSON `{id, fields}` (lectura por
  cuerpo), y decodifican el envoltorio de respuesta.
- `cuerpoProveedorNatural()`: cuerpo de **creación** válido de un proveedor
  natural (proveedor raíz + tercero con datos de identificación y FK del tipo de
  contribuyente + rama natural + contacto).
- `cuerpoProveedorNaturalUpdate(provID, terceroID, diID)`: cuerpo de
  **reconciliación** (`PATCH`) que identifica las entidades por su Id y omite los
  inmutables.
- `cuerpoProveedorNaturalConTerceroID(terceroID, incluir)`: cuerpo de **creación**
  de un proveedor natural nuevo que varía la FK ascendente reenviada `TerceroId`;
  `incluir=false` omite el campo (ausencia), `incluir=true` lo agrega con el valor
  dado (incluido `nil` para un `null` explícito). Ejercita la matriz de la FK
  ascendente ausente/null/valor extremo a extremo.
- `CRUDSimulado.FallarPostDe(recurso, status)`: configura el mock para que el
  `POST` de un recurso responda con un status HTTP de error (p. ej. `503`),
  simulando un CRUD dueño caído para ejercitar la política de fallo tolerado
  (`notify`) de una relación.
- Rangos de Id por entidad en el mock: proveedor `100+`, tercero `200+`,
  datos_identificacion `300+`, proveedor_natural `400+`, contacto `600+`. Permiten
  afirmar de qué CRUD provino cada Id materializado.

## Pruebas de integración (`integration_test.go`)

### TestIntegracionCreacionCompleta

- **Propósito**: verificar que un `POST /v1/proveedor` materializa el subárbol
  completo en cascada, respeta la propagación de llave `fk_inbound` del tercero y
  **canoniza** la FK ascendente reenviada desactualizada en una entidad nueva.
- **Actores**: cliente del API, middleware, CRUD simulados (sin precarga).
- **Entradas**: `POST`, cuerpo `cuerpoProveedorNatural()` (reenvía
  `TerceroId=25`), cabecera `X-Autor-Id: 9`.
- **Salidas esperadas**:
  - `206` con envoltorio `Success: true`, `Status: "206"`, porque la FK
    ascendente reenviada (`TerceroId=25`) difiere del Id autoritativo del tercero
    recién materializado: una entidad nueva canoniza y avisa.
  - `MetaData.Errors` incluye una entrada con `Kind: field_normalization`.
  - El proveedor raíz materializa `Id=101`.
  - El tercero (FK ascendente) materializa su propio `Id=201`, distinto del
    proveedor, y el proveedor lo referencia por `TerceroId=201` (valor
    autoritativo canonizado) como **valor numérico** (no cadena de texto).
  - El `ProveedorNatural` (PK compartida) no tiene Id propio: hereda el del
    proveedor como **valor numérico** (`ProveedorNatural.Id == 101`).
  - Cada `Contacto` (FK descendente) materializa su propio `Id` (rango 600) y
    referencia al proveedor por `ProveedorId=101` como **valor numérico** (no
    cadena de texto).
  - El tercero se crea **antes** que el proveedor (orden `fk_inbound`).
  - Un único `POST /v1/proveedor/`.
  - La rama jurídica, podada para un proveedor natural, nunca se materializa.
  - El autor `9` se inyecta en los `POST` desde la cabecera, no desde el cuerpo.

### TestIntegracionReconciliacionPatchIdempotente

- **Propósito**: verificar que un `PATCH` reconcilia sin duplicar cuando el
  proveedor ya existe por su `tercero_id`.
- **Actores**: cliente del API, middleware, CRUD simulados con el proveedor
  precargado (`GET /v1/proveedor` → `{Id:101, TerceroId:25}`).
- **Entradas**: `PATCH`, cuerpo `cuerpoProveedorNaturalUpdate(101, 201, 301)`.
- **Salidas esperadas**:
  - `200`; `Data.Id == 101` (se reutiliza el proveedor existente).
  - Cero `POST` de proveedor y exactamente un `PATCH` de proveedor.

### TestIntegracionContinuacionTrasCreacionParcial

- **Propósito**: verificar la continuación tras una creación parcial: lo ya
  creado se reutiliza y solo se crea lo faltante.
- **Actores**: cliente del API, middleware, CRUD simulados con proveedor
  (`Id:101`) y datos_identificacion (`Id:301`) precargados.
- **Entradas**: `PATCH`, cuerpo `cuerpoProveedorNaturalUpdate(101, 201, 301)`.
- **Salidas esperadas**:
  - `200`; `Data.Id == 101`.
  - Proveedor: 0 `POST`, 1 `PATCH`.
  - datos_identificacion existente: 0 `POST`, 1 `PATCH` (no se duplica).
  - Contacto faltante: 1 `POST` (lo ausente sí se crea).

### TestIntegracionFieldVetadoSeRechazaComoNoPermitido

- **Propósito**: verificar la **negación tipo `allowed_fields` por lista negra**
  en la ruta de **lectura por cuerpo** (`POST /v1/proveedor/consulta`): pedir en
  `fields` un campo que sí está declarado en la configuración real
  (`allowed_fields` de la relación) pero que la lista negra global veta se rechaza
  igual que un campo no permitido, sin distinguir el caso (política anti-sondeo) y
  sin consultar el CRUD dueño (fallo temprano).
- **Actores**: cliente del API (`POST /v1/proveedor/consulta` con cuerpo
  `{id, fields}`), middleware con una lista negra global inyectada, CRUD simulados
  (sin precarga).
- **Entradas**:
  - Lista negra: `terceros → tercero → [NombreCompleto]` (veta un campo que el
    YAML real sí lista en `allowed_fields` de `Tercero`).
  - Caso A: `POST /v1/proveedor/consulta` con `{id:101, fields:["Tercero.NombreCompleto"]}`
    sobre el router con la lista negra inyectada (`routerProveedorConBlacklist`).
  - Caso B (contraste): `POST /v1/proveedor/consulta` con
    `{id:101, fields:["Tercero.CampoInexistente"]}` sobre el router sin lista negra
    (`routerProveedor`), un campo realmente no permitido.
- **Salidas esperadas**:
  - Ambos casos: `400` con envoltorio `Success: false`, `Status: "400"`.
  - El mensaje del campo vetado usa la misma plantilla "no está permitido" que el
    no permitido y cita la ruta solicitada (`Tercero.NombreCompleto`), sin revelar
    que el campo existe ni que está vetado.
  - Ninguna petición al CRUD de `proveedor`: la ruta vetada se rechaza antes de
    consultar el CRUD dueño.

### TestIntegracionFKAscendenteAusente200

- **Propósito**: verificar que, con la FK ascendente (`TerceroId`) **ausente** en
  un proveedor nuevo, el motor la completa con el Id autoritativo del tercero y
  responde `200` sin aviso.
- **Actores**: cliente del API, middleware, CRUD simulados (sin precarga).
- **Entradas**: `POST`, cuerpo `cuerpoProveedorNaturalConTerceroID(nil, false)`
  (sin `TerceroId`).
- **Salidas esperadas**:
  - `200`; `MetaData.Errors` ausente o vacío (completar una FK ausente no genera
    aviso).
  - `Data.TerceroId` se completa, como **valor numérico**, con el Id del tercero
    materializado.

### TestIntegracionFKAscendenteNull200

- **Propósito**: verificar que una FK ascendente reenviada como `null` se trata
  como ausencia, se completa con el Id autoritativo y responde `200` sin aviso.
- **Actores**: cliente del API, middleware, CRUD simulados (sin precarga).
- **Entradas**: `POST`, cuerpo `cuerpoProveedorNaturalConTerceroID(nil, true)`
  (`TerceroId: null`).
- **Salidas esperadas**:
  - `200`; sin avisos.
  - `Data.TerceroId` completado numérico con el Id del tercero materializado.

### TestIntegracionReenvioDataCoincide200

- **Propósito**: verificar el reenvío **idempotente** del `Data` materializado:
  reenviar por `PATCH` el subárbol con todas sus llaves relacionales, cuando la FK
  ascendente coincide con el Id autoritativo, responde `200` sin aviso y reutiliza
  las entidades por Id sin duplicarlas.
- **Actores**: cliente del API, middleware, CRUD simulados con el proveedor
  precargado (`GET /v1/proveedor` → `{Id:101, TerceroId:201}`).
- **Entradas**: `PATCH`, cuerpo que reenvía proveedor y tercero con Id propio y
  `TerceroId == tercero.Id == 201`.
- **Salidas esperadas**:
  - `200`; `MetaData.Errors` ausente o vacío (reenvío coincidente, sin aviso).
  - `Data.TerceroId == 201` como **valor numérico**.
  - Idempotencia: 0 `POST` de proveedor y 1 `PATCH` (se reutiliza, no se
    duplica).

### TestIntegracionConflictoFKAscendenteExistente400

- **Propósito**: verificar que una FK ascendente que **difiere** en una entidad
  existente (proveedor con Id propio) es un conflicto `fk_inbound`: se rechaza con
  `400` sin reasignar ni materializar la operación incompatible.
- **Actores**: cliente del API, middleware, CRUD simulados con el proveedor
  precargado (`{Id:101, TerceroId:201}`).
- **Entradas**: `PATCH`, cuerpo con proveedor `Id:101` y `TerceroId=999` divergente
  del tercero referenciado (`Id:201`).
- **Salidas esperadas**:
  - `400` con envoltorio `Success: false`, `Status: "400"`.
  - No se materializa el proveedor en conflicto: 0 `POST` y 0 `PATCH` de
    proveedor (sin reasignación silenciosa).

### TestIntegracionFalloRelacionToleradaNotify206

- **Propósito**: verificar la política de fallo **tolerado** (`notify`): cuando el
  CRUD de una relación no bloqueante responde error, el motor conserva lo ya
  materializado y responde `206` con un aviso `Kind: relation`, sin exponer el
  error interno del CRUD.
- **Actores**: cliente del API, middleware, CRUD simulados con el `POST` de
  `contacto` forzado a `503` (`FallarPostDe("contacto", 503)`).
- **Entradas**: `POST`, cuerpo `cuerpoProveedorNaturalConTerceroID(nil, false)`
  (FK ascendente ausente, para aislar el aviso de relación del de normalización).
- **Salidas esperadas**:
  - `206`; `MetaData.Errors` incluye una entrada con `Kind: relation`.
  - El aviso no expone el cuerpo interno del CRUD.
  - El proveedor sí se materializa pese al fallo del contacto (creación parcial):
    1 `POST` de proveedor.

## Prueba e2e (`e2e_test.go`)

### TestE2EProveedorCreacionYActualizacion

- **Propósito**: ejercitar un flujo e2e completo del subárbol sobre un único
  router y CRUD simulado: crear (`POST`) y luego actualizar (`PATCH`) reutilizando
  los identificadores materializados que devolvió la creación, como lo haría un
  cliente real que continúa sobre lo ya creado.
- **Actores**: cliente del API, middleware, CRUD dueños simulados.
- **Entradas**:
  - Fase 1: `POST`, cuerpo `cuerpoProveedorNatural()`, `X-Autor-Id: 9`.
  - Fase 2: `PATCH`, cuerpo `cuerpoProveedorNaturalUpdate(provID, terceroID,
    diID)` con los Ids devueltos por la fase 1.
- **Salidas esperadas**:
  - Fase 1: `206` (la FK ascendente reenviada `TerceroId=25` se canoniza en la
    entidad nueva, con aviso `field_normalization`); el proveedor y el tercero
    materializan Ids propios distintos; `proveedor.TerceroId == tercero.Id` (FK
    ascendente) como **valor numérico** (no cadena de texto); datos_identificacion
    materializa su Id.
  - Fase 2: `200`; opera sobre el mismo proveedor (`Data.Id == provID`).
  - Conteos a lo largo de las dos fases, por entidad con Id estable: proveedor,
    tercero y datos_identificacion con exactamente 1 `POST` (crear) y 1 `PATCH`
    (actualizar), sin duplicados.

## Matriz de comportamiento de campos relacionales derivados

Cada fila es un caso de la conciliación de un campo derivado (FK descendente,
PK compartida o FK ascendente) y la respuesta observable. La columna de cada
nivel indica qué prueba lo cubre: **unitaria** en `helper/write_engine_test.go`
(motor, con `resolverFalso`), **controlador** en
`controllers/proveedor_test.go` (mapeo a envoltorio y código HTTP) e
**integración/e2e** en `test/` (observable extremo a extremo vía router y CRUD
simulado).

| Caso | Resultado | Unitaria (`helper/`) | Controlador | Integración/e2e (`test/`) |
| --- | --- | --- | --- | --- |
| Campo derivado **ausente** | completa con la llave autoritativa, `200` sin aviso | `TestWriteFKDerivadoAusente`, `TestWriteSharedPKAusente`, `TestWriteInboundAusente` | — | `TestIntegracionFKAscendenteAusente200` |
| Campo derivado **`null`** | se trata como ausente, completa, `200` sin aviso | `TestWriteFKDerivadoNull`, `TestWriteInboundNull` | — | `TestIntegracionFKAscendenteNull200` |
| Campo derivado **coincide** | conserva, `200` sin aviso | `TestWriteFKDerivadoCoincide`, `TestWriteSharedPKCoincide`, `TestWriteInboundCoincide` | `campo derivado coincidente 200 sin aviso` (en `TestProveedorHandlerEscribir`) | `TestIntegracionReenvioDataCoincide200` |
| **Diferencia en entidad nueva** | canoniza con la llave autoritativa, `206` + `Kind: field_normalization` | `TestWriteFKDerivadoDiferenteNuevo`, `TestWriteInboundDiferenteNuevo` | `canonización de campo derivado 206 field_normalization` (en `TestProveedorHandlerEscribir`) | `TestIntegracionCreacionCompleta`, e2e fase 1 |
| **Diferencia en entidad existente** | conflicto, `400`, sin reasignar | `TestWriteFKDerivadoDiferenteExistente`, `TestWriteSharedPKDiferenteExistente`, `TestWriteInboundDiferenteExistente` | `campo inválido 400` (en `TestProveedorHandlerEscribir`) | `TestIntegracionConflictoFKAscendenteExistente400` |
| **Fallo de relación tolerado** (`notify`) | conserva lo materializado, `206` + `Kind: relation` | `TestWriteTolerante` | `materialización parcial 206` (en `TestProveedorHandlerEscribir`) | `TestIntegracionFalloRelacionToleradaNotify206` |
| **Fallo de relación bloqueante** (`fail`) | aborta, `502`/`504` | `TestWriteEstricto` | `fallo bloqueante 502`, `fallo bloqueante por timeout 504` (en `TestProveedorHandlerEscribir`) | — (cubierto por unitaria del motor y prueba del controlador) |

Las normalizaciones y conflictos de campos derivados aplican a los tres modos de
propagación (`fk`, `shared_pk`, `fk_inbound`); las filas listan las pruebas
unitarias por modo donde aplica. El caso `fail` no se repite en integración: lo
cubren la prueba unitaria del motor y la del controlador que mapea el error fatal
al código HTTP.

## Cómo ejecutar

```bash
go test ./test/...
```
