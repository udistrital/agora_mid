# Tasks

## 1. Motor base y configuración

- [x] 1.1 Portar los paquetes `clients`, `config`, `helper` y `responses` desde `base_engine`.
- [x] 1.2 Portar `services/api_response.go`, `services/catalogos.go` y `services/write_resolver.go`.
- [x] 1.3 Crear `config/terceros/relaciones_tercero.yaml` con raíz `Tercero` y la relación escribible `DatosIdentificacion`.
- [x] 1.4 Corregir `DigitoVerficacion` a `DigitoVerificacion` en `config/terceros/objetos_datos_identificacion.yaml`.
- [x] 1.5 Declarar `natural_key: [Numero, TipoDocumentoId]` en `DatosIdentificacion`.
- [x] 1.6 Marcar `Activo` como escribible en `Tercero` y `DatosIdentificacion` para que el MID lo inyecte en el alta.
- [x] 1.7 Ajustar `config.Load` para aceptar las claves actuales de `conf/app.conf`.
- [x] 1.8 Reducir `config` y `helper` al mínimo del dominio 1: configuración de `Tercero`/`DatosIdentificacion` y solo el motor de escritura. Retirar tests y catálogo no usados.

## 2. Resolvers de escritura

- [x] 2.1 Cambiar `EntityWriteResolver` para actualizar con `PUT`.
- [x] 2.2 Leer el registro existente y mezclar el payload antes del `PUT` para conservar campos ausentes.
- [x] 2.3 Envolver como `{Id:n}` las FK de `terceros_crud` (`TipoContribuyenteId`, `TipoDocumentoId`, `TerceroId`).
- [x] 2.4 Aplanar esas FK al recuperar el registro, para que el motor compare llaves escalares.
- [x] 2.5 Agregar `sortby=Id&order=asc` a la búsqueda por clave natural.
- [x] 2.6 Crear `NewTerceroWriteResolvers` con raíz `Tercero` y resolver de `DatosIdentificacion`.

## 3. Servicio del dominio 1

- [x] 3.1 Crear `TerceroWriteService` con `helper.Write` y el registro de resolvers.
- [x] 3.2 Validar `TipoContribuyenteId` contra `P_NATURAL` resuelto al arranque.
- [x] 3.3 Resolver en `POST` la identificación por `Numero + TipoDocumentoId + Activo:true`, con orden determinista.
- [x] 3.4 Resolver en `PUT` el tercero por `TerceroId` y su identificación activa.
- [x] 3.5 Rechazar con `409` un documento que pertenezca a otro tercero.
- [x] 3.6 Derivar `NombreCompleto` en el alta y conservar nombres en actualización.
- [x] 3.7 Inyectar `Activo:true` en las altas.
- [x] 3.8 Construir `Resultado` (`Creados`, `Reutilizados`, `Actualizados`) y los IDs de respuesta.
- [x] 3.9 Propagar avisos de ambigüedad y de materialización parcial.
- [x] 3.10 Tratar `[]` y `[{}]` como ausencia de identificación y cubrir el caso con pruebas de alta y `PUT` inexistente.

## 4. Controladores, rutas y envelope

- [x] 4.1 Implementar `POST /v1/proveedores/natural`.
- [x] 4.2 Implementar `PUT /v1/proveedores/natural/{terceroId}` con el parámetro al final.
- [x] 4.3 Rechazar campos desconocidos, incluido `TipoProveedor`.
- [x] 4.4 Validar `TerceroId`, `TipoContribuyenteId`, `Numero` y `TipoDocumentoId`.
- [x] 4.5 Mapear los errores del servicio a `400/404/409/500/502/504`.
- [x] 4.6 Publicar `200/201/206` con el envelope `utils_oas` v2 y `MetaData.Errors` cuando aplique.
- [x] 4.7 Registrar `customerror`, `apiStatusLib` y `security` en `main.go`.
- [x] 4.8 Configurar el router con `POST` y `PUT`.

## 5. Pruebas y verificación

- [x] 5.1 Probar el alta con `terceros_crud` simulado: `Activo`, `NombreCompleto`, FK como objeto, `DigitoVerificacion` y IDs.
- [x] 5.2 Probar la actualización: conserva nombres, `Activo` y `TerceroId`; actualiza la identificación.
- [x] 5.3 Probar el rechazo de un documento de otro tercero.
- [x] 5.4 Probar el rechazo de `TipoContribuyenteId` inválido.
- [x] 5.5 Probar el controlador: `201`, `200`, `400`, `404`, envelope y mapeo de `TerceroId`.
- [x] 5.6 Ejecutar `go build ./...`, `go test ./...` y `go vet ./...`.
- [x] 5.7 Validar el flujo real con un `terceros_crud` simulado: `POST 201` y `PUT 200`.
