# Tasks

> Decisión de negocio resuelta (design.md - Decisión de negocio): la fuente es
> `datos_identificacion`. `[]`/`[{}]` → crear tercero (rama B); objeto no vacío
> → leer tercero por `GET tercero/{id}` con `TerceroId.Id` (rama A). No hay
> rama C ni uso de `tercero/identificacion`.

## 1. Modelos y ruta del endpoint
- [x] 1.1 Definir los modelos de entrada/salida del endpoint (`TipoProveedor`, `DatosBasicos`, `DatosIdentificacion`, `Data`) con `Numero` como string y la grafía `DatosIdentificacion`. Verificar con una prueba de (de)serialización que `Numero` con ceros iniciales se conserva y que las claves mantienen su grafía.
- [x] 1.2 Registrar la ruta `POST /v1/proveedores/natural` en `routers/router.go` bajo el namespace `/v1` apuntando al nuevo controlador. Verificar con una prueba de routing (beego `ServeHTTP`) que la ruta resuelve al handler.

## 2. Controlador
- [x] 2.1 Implementar el controlador: decodifica y valida la petición, propaga `ctx` a la coordinación y arma el envoltorio del middleware (`Status`, `Message`, `Data`) según la convención del proyecto. Verificar con pruebas que una petición válida invoca la coordinación y que una inválida responde el contrato de error sin llamar a los CRUD (cubre "Contrato del endpoint de registro").
- [x] 2.2 Verificar que el host de los CRUD nunca aparece en la respuesta pública ante error (cubre "Manejo de errores y logs del flujo").

## 3. Coordinación del flujo (requiere decisión B/C resuelta)
- [x] 3.1 Implementar la consulta de identificación con `GetAll` sobre `datos_identificacion` (`Numero`, `TipoDocumentoId`, `Activo=true`), comprobando primero `ok`. Verificar con `httptest` que una consulta fallida detiene el flujo sin escrituras (cubre "Consulta de la identificación activa").
- [x] 3.2 Implementar la interpretación de la lista: `[]`/`[{}]` como ausencia; primer objeto no vacío como coincidencia; distinguir `Id` de `TerceroId.Id`; ante varias, usar la primera y emitir warning con filtros, cantidad e `Id`, enmascarando el documento. Verificar con `httptest` los tres casos (cubre "Interpretación de la lista de identificación").
- [x] 3.3 Implementar la actualización de identificación: payload desde el registro recuperado, reemplazando solo los campos enviados, un único `Put` por el `Id`, respuesta en `Data.DatosIdentificacion`. Verificar con `httptest` que se emite exactamente un `Put` y se preservan `Id`/relación/campos requeridos (cubre "Actualización de la identificación existente").
- [x] 3.4 Implementar la rama A (tercero existe): `GetOne` por `TerceroId.Id`, `Put` reemplazando solo `FechaNacimiento` y conservando nombres/apellidos recuperados, respuesta en `Data.DatosBasicos` (cubre "Rama A").
- [x] 3.5 Implementar la rama B (crear tercero) cuando la identificación es `[]`/`[{}]`: `Post` del tercero → `Data.DatosBasicos`; crear identificación con el `Id` devuelto → `Data.DatosIdentificacion`. Sin uso de `tercero/identificacion` ni localización por otra clave (cubre "Rama B — crear el tercero cuando no hay identificación activa").
- [x] 3.6 Implementar la guarda de `Id` de tercero: obtener el `Id` solo de `TerceroId.Id` o del alta; ninguna consulta ni escritura con `Id = 0`; referencia incompleta es error; sin deducción por nombres. Verificar con prueba (cubre "No operar con Id de tercero cero o no confirmado").
- [x] 3.7 Implementar la resolución del proveedor: `GetAll` por tercero+`TipoProveedor`+`Activo=true`; reutilizar `Id` con cero escrituras si existe, o `Post` si falta; `Id` en `Data.ProveedorId`. Verificar con `httptest` ambas ramas, confirmando cero escrituras en la de reutilización (cubre "Resolución del proveedor").

## 4. Composición de respuesta, estado HTTP y matriz de pruebas
- [x] 4.1 Implementar la composición de `Data` (`TipoProveedor`, `DatosBasicos`, `DatosIdentificacion`, `ProveedorId`) y la regla de estado: 201 si hubo al menos una creación, 200 si todo existía; un fallo posterior a una escritura no produce 201 (cubre "Composición de la respuesta y estado HTTP").
- [x] 4.2 Montar las pruebas de flujo con **dos** `httptest.NewServer` (uno por CRUD), apuntando `TercerosCrudHost` y `GestionProveedoresCrudHost` a cada uno. Verificar la matriz de casos, registrando por servidor las llamadas recibidas (método, ruta, cuerpo):

  | # | Caso | Llamadas terceros | Llamadas proveedores | `Data` | HTTP |
  |---|---|---|---|---|---|
  | 1 | Todos existen | GetAll ident, Put ident, GetOne tercero, Put tercero | GetAll proveedor (0 escrituras) | ident+tercero actualizados, nombres conservados, ProveedorId reutilizado | 200 |
  | 2 | Varias identificaciones activas | GetAll ident (>1), Put ident (por Id del primero), GetOne+Put tercero | GetAll proveedor | warning emitido; un solo Put ident | 200/201 según proveedor |
  | 3 | Solo falta el proveedor | GetAll ident, Put ident, GetOne+Put tercero | GetAll proveedor, Post proveedor (1 vez) | ProveedorId nuevo | 201 |
  | 4 | Búsqueda `[]`/`[{}]` → rama B | GetAll ident, Post tercero, Post ident | GetAll proveedor | sin consultas/escrituras con Id 0; tercero creado | 201 |
  | 5 | Tercero nuevo (idéntico a 4, verifica asociación por Id del alta) | GetAll ident, Post tercero, Post ident | GetAll proveedor, Post proveedor | ident+proveedor asociados al Id del alta | 201 |
  | 7a | Falla consulta inicial | GetAll ident (falla) | — (0) | — | error, 0 escrituras |
  | 7b | Falla tras un alta | Post tercero (ok), luego fallo | según punto de fallo | etapa en log | error, sin repetir la escritura confirmada, no 201 |

  (cubre los escenarios de identificación, ramas A/B/C, proveedor, estado HTTP y errores).
- [x] 4.3 Verificar que ante fallo se registra la etapa en logs y el host de los CRUD queda fuera del error público (cubre "Manejo de errores y logs del flujo").

## 5. Verificación de integración
- [x] 5.1 Ejecutar `go build ./...` (o los paquetes afectados si el esqueleto aún no compila completo), `go test` de los paquetes del cambio y `go vet`, verificando compilación, pruebas en verde y vet limpio.
