# Proposal

## Why

El middleware necesita consumir por HTTP el CRUD de terceros, pero hoy no
existe ninguna capa de servicios ni cliente HTTP en el código Go: los
paquetes `services/`, `controllers/` y `models/` están vacíos y la clave
`TercerosCrudHost` solo está declarada en `conf/app.conf` sin ningún
consumidor. Sin esta capa, los futuros controladores no tienen forma de
leer ni escribir terceros en el CRUD externo.

## What Changes

- Se introduce un paquete compartido `utils/` con un cliente HTTP reutilizable
  (timeout configurable por entorno, por defecto 30s), un helper para unir
  host y endpoint conservando el prefijo de ruta, y la ejecución y
  decodificación de la petición. Será reutilizable por otros servicios CRUD
  (por ejemplo, el futuro servicio de proveedores).
- Se crea el servicio de terceros en `services/terceros.go` con cuatro
  métodos que consumen el CRUD de terceros leyendo la URL base desde la clave
  de configuración `TercerosCrudHost`:
  - `GetAll(ctx, endpoint, queryParams)`: lista con parámetros de consulta
    opcionales.
  - `GetOne(ctx, endpoint)`: obtiene un recurso.
  - `Post(ctx, endpoint, payload)`: crea un recurso enviando JSON.
  - `Put(ctx, endpoint, payload)`: actualiza un recurso enviando JSON.
- Cada método recibe `context.Context` como primer parámetro para propagar la
  cancelación, conforme a la convención del proyecto (el controlador pasa su
  contexto a la capa de servicios).
- El contrato de salida uniforme `(ok bool, statusCode int, body Response)` usa
  el formato de respuesta estándar de la OAS (`Response` con `Success`,
  `Status`, `Message`, `Data` y `Metadata` opcional); `Metadata` solo se
  puebla en `GetAll`. Distingue el resultado del CRUD del resultado que recibe
  el consumidor.
- Se añaden pruebas con `httptest.NewServer`, sin depender de un CRUD en
  ejecución.

Fuera de alcance: controladores, rutas públicas del middleware, y los verbos
`DELETE` y variantes especiales del contrato (quedan para cambios futuros).

## Capabilities

### New Capabilities
- `terceros-crud-cliente`: consumo por HTTP del CRUD de terceros desde la capa
  de servicios del middleware (construcción de la petición, propagación de
  contexto, y contrato de salida que distingue el estado del CRUD del
  resultado entregado al consumidor).

### Modified Capabilities
<!-- No hay capacidades existentes; el proyecto no tiene specs todavía. -->

## Impact

- **Código nuevo**: `utils/` (cliente HTTP, helper de URL y las structs
  `Response`/`Metadata` del formato OAS), `services/terceros.go` y sus pruebas
  (`utils/*_test.go`, `services/terceros_test.go`).
- **Configuración**: consume la clave `TercerosCrudHost` de `conf/app.conf`
  (ya presente, expandida desde la variable de entorno `TERCEROS_CRUD_SERVICE`;
  forma esperada `http://host:puerto/v1`) y la variable de entorno
  `HTTP_CLIENT_TIMEOUT` para el timeout del cliente HTTP (por defecto 30s).
- **Dependencias**: solo librería estándar (`net/http`, `net/url`,
  `encoding/json`, `context`) y `web.AppConfig` de Beego v2, ya presente.
  Las pruebas usan `httptest` y `goconvey`, ya en `go.mod`.
- **Sin cambios de ruta pública**: no se registran controladores ni rutas en
  `routers/router.go`.
