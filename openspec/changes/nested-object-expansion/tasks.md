# Tasks

## 1. Contrato de respuesta (Forma 3)

- [x] 1.1 Definir los tipos del envoltorio OAS plano (`Success`, `Status` string, `Message`, `Data`) y el bloque opcional `MetaData` con `Errors []RelationError` opcional y `Length *int` opcional (total de valores posibles de una lista, sin `limit`/`offset`) en la capa de respuesta; verificar con `go build ./...`
- [x] 1.2 Reemplazar `respuestaRegistro` y `respuestaOperacion` por el nuevo envoltorio en `controllers/registro_persona.go` para todos los verbos; verificar que POST/PUT/GET serializan la raíz plana y que `MetaData` se omite sin avisos
- [x] 1.3 Agregar pruebas de controlador que verifiquen serialización de `Success/Status/Message/Data`, omisión de `MetaData` sin avisos y presencia de `MetaData.Errors` con `Status: "206"` cuando hay avisos; verificar con `go test ./controllers/...`

## 2. Configuración declarativa de relaciones

- [x] 2.1 Definir tipos `RelationConfig`/`ServiceConfig` y promover `gopkg.in/yaml.v3` a dependencia directa; verificar con `go mod tidy && go build ./...`
- [x] 2.2 Implementar la carga de `config/relaciones_<servicio>.yaml` al arranque en `config/`, resolviendo `related_service` contra las URLs de `conf/app.conf`; verificar que un archivo válido se carga en una prueba
- [x] 2.3 Implementar validación (nombres no vacíos, `related_api` presente y existente en URLs, `related_service` no vacío como nombre lógico, `direction`/`cardinality` válidos, `depth` positivo, `mapped_name` único por servicio) que aborta el arranque con mensaje que identifica archivo y relación; verificar con pruebas de config inválida en `config/`

## 3. Resolver sobre HTTPClient

- [x] 3.1 Definir la interfaz `Resolver` y un adaptador sobre `clients.HTTPClient` que respete `context.Context`, codifique parámetros desde `query_template` y reutilice `decodeAPIList` para envelope OAS y lista directa; verificar con `go build ./...`
- [x] 3.2 Implementar el resolver genérico que decodifica y entrega `Data` sin transformar solo para relaciones declaradas; verificar con prueba que confirma payload sin transformación y aviso `generic_resolver`
- [x] 3.3 Agregar pruebas unitarias del resolver con mocks HTTP para respuestas enveloped y directas, timeouts y cancelación por contexto; verificar con `go test ./...` del paquete del resolver

## 4. Helper de expansión

- [x] 4.1 Implementar el parseo de `fields` en rutas, rechazo de rutas vacías/duplicadas/no permitidas y construcción del árbol de selección; verificar con pruebas de rutas válidas e inválidas
- [x] 4.2 Implementar el expansor recursivo: dedupe de llaves, consulta por lotes, mapeo por `related_field`, inserción en `mapped_name`, `null` para `one` sin resultado y `[]` para `many`, caché por solicitud; verificar con pruebas de `one` encontrada/no encontrada y `many` con cero/uno/varios
- [x] 4.3 Implementar límites, detección de ciclos y profundidad efectiva (`min(global restante, depth)`), con avisos `depth` y `cycle`; verificar con pruebas de ciclo, límite de IDs, `max_depth` y reducción por config hija
- [x] 4.4 Implementar política `on_relation_error` (`notify` conserva principal + aviso `relation`; `fail` aborta) y errores de entrada/configuración como error normal; verificar con pruebas de ambos modos
- [x] 4.5 Agregar pruebas unitarias de selección de campos base/anidados e inclusión automática de padres; verificar con `go test ./helper/...`

## 5. Integración en controladores de lectura

- [ ] 5.1 Leer `fields` de la query string en la lectura de `controllers/registro_persona.go`, construir `ExpandOptions` e invocar el helper sin implementar recursión en el controlador; verificar con prueba de `fields=Id,Tercero.Id,Tercero.NombreCompleto`
- [ ] 5.2 Mapear resultados a códigos HTTP (`400` fields inválido, `206` expansión parcial tolerada, `502/504` en modo `fail`) y serializar avisos en `MetaData.Errors`; verificar con pruebas por cada situación
- [ ] 5.3 Agregar pruebas de controlador para `fields` ausente, campo inválido, rutas duplicadas, preservación de la llave local y serialización de `null`/`[]`/`MetaData.Errors`; verificar con `go test ./controllers/...`

## 6. Configuración inicial y documentación

- [ ] 6.1 Crear los archivos de relaciones iniciales para `terceros`, `parametros` y `gestion_personas_crud` en `config/`; verificar que el API arranca con ellos cargados
- [ ] 6.2 Documentar el contrato público (Forma 3 y parámetro `fields`) en Swagger; verificar que la definición refleja `Success/Status/Message/Data` y `MetaData.Errors`

## 7. Verificación integral

- [ ] 7.1 Ejecutar `go test ./...` y `go build ./...` en verde sobre todo el proyecto, cubriendo helper, resolver, config y controladores
