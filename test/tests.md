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
  contra el CRUD simulado, cargando el YAML de producción.
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
    proveedor, y el proveedor lo referencia por `TerceroId=201`.
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
    `proveedor.TerceroId == tercero.Id` (FK ascendente); datos_identificacion
    materializa su Id.
  - Fase 2: `200`; opera sobre el mismo proveedor (`Data.Id == provID`).
  - Conteos a lo largo de las dos fases, por entidad con Id estable: proveedor,
    tercero y datos_identificacion con exactamente 1 `POST` (crear) y 1 `PATCH`
    (actualizar), sin duplicados.

## Cómo ejecutar

```bash
go test ./test/...
```
