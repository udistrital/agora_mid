# Documentación de la suite de pruebas del subárbol del proveedor

Este documento describe las pruebas de integración y de extremo a extremo (e2e)
del middleware del proveedor, ubicadas en el paquete `test/`. Las pruebas
recorren el stack completo **cliente → controlador → servicio → motor** contra CRUD
dueños simulados, sin red real ni base de datos. Cubren tanto la lectura de
colección como la escritura del subárbol del proveedor.

## Actores

- **Cliente del API**: envía cuerpos JSON para consultar el listado o materializar
  el subárbol por `POST`/`PATCH /v1/proveedor`, y el autor de la modificación
  por la cabecera `X-Autor-Id` cuando corresponde.
- **Middleware (MID)**: los routers y controladores del proveedor, los servicios
  de lectura y escritura, y los motores genéricos cableados con la configuración
  declarativa real (`config/relaciones_proveedor.yaml`, cargada desde `../config`).
- **CRUD dueños simulados**: un único servidor HTTP de prueba (`CRUDSimulado`,
  en `fixtures.go`) que imita a Terceros CRUD y a Gestión de Personas CRUD.
  Responde `{"Data":{"Id":n}}` en `POST`/`PATCH` y `{"Data":[...]}` en `GET`,
  y captura cada petición (método, path, query, cuerpo) para las aserciones.

## Fixtures y mocks compartidos (`fixtures.go`)

- `CRUDSimulado` / `NuevoCRUDSimulado(t, existentes)`: servidor CRUD simulado;
  `existentes` precarga registros que una búsqueda por clave natural o por Id debe
  devolver (para idempotencia y continuación tras creación parcial).
- `routerProveedor(t, crud)`: arma el router real de `/v1/proveedor` y de las
  consultas del proveedor sobre los servicios reales cableados contra el CRUD
  simulado, cargando el YAML de producción. Equivale a
  `routerProveedorConBlacklist(t, crud, nil)`.
- `routerProveedorConBlacklist(t, crud, blacklist)`: como `routerProveedor` pero
  inyecta una lista negra global (`config.GlobalBlacklist`) en el servicio de
  lectura, para ejercitar el recorte y el rechazo por lista negra sobre la
  configuración real. Una `blacklist` nil equivale a "sin supresión". El router
  despacha `POST /v1/proveedor/consulta` a `Consultar` y
  `POST /v1/proveedor/consulta/lista` a `ConsultarLista`, replicando el enrutado
  de `main`.
- `ejecutar(t, router, metodo, cuerpo)` / `ejecutarConsulta(t, router, id, fields)` /
  `ejecutarListaProveedor(t, router, cuerpo)`: envían, respectivamente, una
  petición con cuerpo a `POST`/`PATCH /v1/proveedor`, una consulta puntual por
  `POST /v1/proveedor/consulta` o un listado por
  `POST /v1/proveedor/consulta/lista`, y decodifican el envoltorio de respuesta.
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

### TestIntegracionListadoProveedor

- **Propósito**: verificar de extremo a extremo el listado de proveedores con filtros
  de primer nivel, expansión mediante `fields`, paginación delegada, propagación
  del total del conjunto y rechazo temprano de filtros sobre relaciones.
- **Actores**:
  - **Cliente del API**: envía `POST /v1/proveedor/consulta/lista` con el cuerpo
    JSON del listado.
  - **Middleware (MID)**: `ConsultarLista`, `ProveedorService.ListarProveedores`,
    el decodificador del envoltorio del CRUD y el motor de expansión de la
    configuración real de proveedor.
  - **CRUD dueños simulados**: `CRUDSimulado`, un servidor HTTP que representa al
    CRUD de Gestión de Personas y al CRUD de Terceros, devuelve listas precargadas
    y captura las peticiones recibidas.
- **Entradas**:
  - Estado precargado en el CRUD simulado:
    - `GET /v1/proveedor`: dos proveedores, con `Id` 101 y 102, ambos con
      `TerceroId=15` y `TipoRegistroId=1`.
    - `GET /v1/tercero`: el tercero 15 con `NombreCompleto="Ana Pérez"`.
  - Solicitud válida:
    - Verbo y ruta: `POST /v1/proveedor/consulta/lista`.
    - `query`: `[`\`TerceroId:15\``, `\`Id__in:101|102\``]`; la primera
      condición usa igualdad implícita y la segunda usa una lista `__in` separada
      por `|`.
    - `fields`: `[`\`Tercero.NombreCompleto\``].
    - Paginación: `limit=1`, `offset=1`.
  - Solicitud inválida independiente:
    - `query`: `[`\`Tercero.NombreCompleto:Ana\``], que intenta filtrar por un
      campo de relación no permitido.
- **Salidas esperadas**:
  - Solicitud válida:
    - Código HTTP `200`, envoltorio con `Success: true` y `Status: "200"`.
    - `Data` contiene un solo proveedor, porque la página solicita un elemento
      desde el desplazamiento 1.
    - Cada proveedor de la página incluye `Tercero.NombreCompleto` expandido.
    - `MetaData.Total` es `2`, tomado del total informado por el CRUD y no del
      tamaño de la página (`1`). El envoltorio puede conservar avisos
      informativos de profundidad en `MetaData.Errors` sin cambiar el estado
      exitoso del listado.
    - El CRUD de proveedores recibe una consulta GET con
      `query=TerceroId:15,Id__in:101|102`; la ventana `limit`/`offset` se aplica
      en el CRUD simulado. El MID consulta el CRUD de terceros para expandir
      `Tercero` y no realiza escrituras.
  - Solicitud inválida:
    - Código HTTP `400` con respuesta no exitosa.
    - El middleware rechaza el filtro antes de consultar el CRUD de proveedores;
      el simulador no registra una lectura del recurso raíz.

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

### TestIntegracionConflictoFKAscendenteNoDejaEscriturasParciales

- **Propósito**: verificar el invariante **"validar antes de escribir"** para
  `fk_inbound`: un conflicto de FK ascendente en un proveedor existente responde
  `400` sin dejar **escrituras parciales** en ningún CRUD dueño. A diferencia de
  `TestIntegracionConflictoFKAscendenteExistente400` (que solo verifica el
  proveedor), esta prueba afirma además que la entidad referenciada (`tercero`) no
  recibe ninguna escritura: el motor recupera su Id autoritativo por lectura y
  concilia la FK local **antes** de materializar.
- **Actores**: cliente del API, middleware, CRUD simulados con el proveedor
  precargado (`GET /v1/proveedor` → `{Id:101, TerceroId:201}`) y el tercero
  precargado (`GET /v1/tercero` → `{Id:201, NombreCompleto:"Ana Pérez"}`).
- **Entradas**: `PATCH`, cuerpo con proveedor `Id:101` y `TerceroId=999` divergente
  del tercero referenciado (`Id:201`).
- **Salidas esperadas**:
  - `400` con envoltorio `Success: false`, `Status: "400"`.
  - Cero escrituras en `tercero`: 0 `POST` y 0 `PATCH` (solo `GET` permitidos).
  - Cero escrituras en `proveedor`: 0 `POST` y 0 `PATCH` (solo `GET` permitidos).

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
| **Diferencia en entidad existente** | conflicto, `400`, sin reasignar | `TestWriteFKDerivadoDiferenteExistente`, `TestWriteSharedPKDiferenteExistente`, `TestWriteInboundDiferenteExistente` | `campo inválido 400` (en `TestProveedorHandlerEscribir`) | `TestIntegracionConflictoFKAscendenteExistente400`, `TestIntegracionConflictoFKAscendenteNoDejaEscriturasParciales` |
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
