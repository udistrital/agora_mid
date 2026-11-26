# Tasks

## 1. Contrato y DTO

- [x] 1.1 Crear `models.RegistroTerceroJuridico` con `DatosBasicos` y `DatosIdentificacion`.
- [x] 1.2 Crear `models.DatosBasicosJuridico` con `RazonSocial`, `LugarOrigen` y `TipoContribuyenteId`.
- [x] 1.3 Reutilizar `models.DatosIdentificacionTercero` para `Numero`, `TipoDocumentoId`, `CiudadExpedicion`, `FechaExpedicion` y `DigitoVerificacion`.
- [x] 1.4 Probar que `RazonSocial` viaja en el DTO y que `Numero` conserva ceros iniciales.

## 2. Servicio del dominio 1 jurídico

- [x] 2.1 Agregar `TipoContribuyenteJuridica` y el parámetro correspondiente en `NewTerceroWriteService`.
- [x] 2.2 Extraer `entradaEscrituraTercero` y el método privado `escribir`.
- [x] 2.3 Agregar `EscribirTerceroJuridico` sobre el mismo núcleo.
- [x] 2.4 Mapear `RazonSocial` a `NombreCompleto` con normalización de espacios.
- [x] 2.5 Actualizar `NombreCompleto` y `LugarOrigen` en el `PUT`; conservarlos en el `POST` de reutilización.
- [x] 2.6 Validar `TipoContribuyenteId` contra `P_JURIDICA` (`400`) y el tipo del tercero existente (`409`).
- [x] 2.7 Reutilizar la resolución de identificación, el manejo de `[]`/`[{}]` y la clasificación `Creados`/`Reutilizados`/`Actualizados`.
- [x] 2.8 Agregar los errores `ErrTipoContribuyenteJuridicoInvalido` y `ErrTipoContribuyenteJuridicoNoConfigurado`.
- [x] 2.9 Rechazar con `409` el cambio de `Numero` o `TipoDocumentoId` en el `PUT` de un tercero existente.

## 3. Controladores, rutas y arranque

- [x] 3.1 Extraer los helpers compartidos a `controllers/tercero_comun.go`.
- [x] 3.2 Refactorizar `ProveedorNaturalController` para delegar en los helpers sin cambiar su contrato.
- [x] 3.3 Implementar `POST /v1/proveedores/juridica`.
- [x] 3.4 Implementar `PUT /v1/proveedores/juridica/{terceroId}` con el parámetro al final.
- [x] 3.5 Rechazar campos desconocidos, incluidos nombres naturales y `TipoProveedor`.
- [x] 3.6 Validar `RazonSocial`, `TipoContribuyenteId`, `Numero` y `TipoDocumentoId`.
- [x] 3.7 Proyectar `DatosBasicos` con `RazonSocial` y omitir nombres naturales vacíos.
- [x] 3.8 Registrar las rutas en `routers/router.go`.
- [x] 3.9 Resolver `P_JURIDICA` en `main.go` y configurar ambos controladores.

## 4. Pruebas y verificación

- [x] 4.1 Probar el alta jurídica: `NombreCompleto`, `Activo`, FK como objeto, `DigitoVerificacion`, sin nombres naturales.
- [x] 4.2 Probar el `POST` con documento existente: reutiliza tercero e identificación y conserva `NombreCompleto`.
- [x] 4.3 Probar el `PUT`: actualiza `RazonSocial` y `LugarOrigen`, conserva `Activo` y campos no enviados.
- [x] 4.4 Probar el `PUT` sin identificación activa: crea la identificación.
- [x] 4.5 Probar `TipoContribuyenteId` natural (`400`), documento de otro tercero (`409`) y tercero de otra variante (`409`).
- [x] 4.6 Probar el controlador: `201`, `200`, `400`, `404`, envelope y proyección de `RazonSocial`.
- [x] 4.7 Probar `[{}]`: alta correcta y `PUT` inexistente con `404`.
- [x] 4.8 Probar que un `PUT` con `Numero` o `TipoDocumentoId` distinto responde `409` sin escribir.
- [x] 4.9 Ejecutar `gofmt`, `go build ./...`, `go test ./...` y `go vet ./...`.
- [x] 4.10 Validar el flujo E2E con `terceros_crud` simulado: `POST 201`, `POST 200`, `PUT 200`, `PUT 404`, `400` por tipo y por campo.

## 5. Documentación OpenSpec

- [x] 5.1 Crear `openspec/specs/registro-proveedor-juridica/spec.md`.
- [x] 5.2 Crear el change con proposal, design, tasks y `.openspec.yaml`.

## 6. Endurecimiento de validaciones e integridad

- [x] 6.1 Validar longitudes reales: `RazonSocial` 255, `Numero` 50, nombres 100, `NombreCompleto` 255.
- [x] 6.2 Normalizar `Numero`, nombres y `RazonSocial`; rechazar `,`, `:`, espacios y control en `Numero`.
- [x] 6.3 Exigir NIT y `DigitoVerificacion` (0–9) en jurídica; NIT solo dígitos.
- [x] 6.4 Validar `TipoDocumentoId` contra `tipo_documento` en natural.
- [x] 6.5 Validar `LugarOrigen` y `CiudadExpedicion` contra `ubicaciones_crud` (existencia y `Activo`).
- [x] 6.6 Rechazar fechas futuras; solo RFC3339.
- [x] 6.7 Detectar terceros e identificaciones inactivas y responder `409`.
- [x] 6.8 Serializar por `Numero + TipoDocumentoId` dentro de la instancia y avisar `duplicate_match`.
- [x] 6.9 Rechazar JSON con cola, `Content-Type` inválido y cuerpos mayores a 64 KB.
- [x] 6.10 Normalizar las FK de respuesta a `{ "Id": n }`.
- [x] 6.11 No informar `DatosIdentificacion` creada con `Id: 0` en `206`.
- [x] 6.12 Tipar errores HTTP y mapear `4xx` del CRUD; no exponer URLs ni números en logs.
- [x] 6.13 Agregar pruebas de longitudes, formatos, catálogos, fechas, inactivos, concurrencia y `206`.
