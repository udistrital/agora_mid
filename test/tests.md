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
- Rangos de Id por entidad en el mock: proveedor `100+`, tercero `200+`,
  datos_identificacion `300+`, proveedor_natural `400+`, contacto `600+`. Permiten
  afirmar de qué CRUD provino cada Id materializado.

## Pruebas de integración (`integration_test.go`)

### TestIntegracionCreacionCompleta

- **Propósito**: verificar que un `POST /v1/proveedor` materializa el subárbol
  completo en cascada y respeta la propagación de llave `fk_inbound` del tercero.
- **Actores**: cliente del API, middleware, CRUD simulados (sin precarga).
- **Entradas**: `POST`, cuerpo `cuerpoProveedorNatural()`, cabecera
  `X-Autor-Id: 9`.
- **Salidas esperadas**:
  - `200` con envoltorio `Success: true`, `Status: "200"`.
  - El proveedor raíz materializa `Id=101`.
  - El tercero (FK ascendente) materializa su propio `Id=201`, distinto del
    proveedor, y el proveedor lo referencia por `TerceroId=201` como **valor
    numérico** (no cadena de texto).
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
  - Fase 1: `200`; el proveedor y el tercero materializan Ids propios distintos;
    `proveedor.TerceroId == tercero.Id` (FK ascendente) como **valor numérico**
    (no cadena de texto); datos_identificacion materializa su Id.
  - Fase 2: `200`; opera sobre el mismo proveedor (`Data.Id == provID`).
  - Conteos a lo largo de las dos fases, por entidad con Id estable: proveedor,
    tercero y datos_identificacion con exactamente 1 `POST` (crear) y 1 `PATCH`
    (actualizar), sin duplicados.

## Cómo ejecutar

```bash
go test ./test/...
```
