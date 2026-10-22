# Tasks

## 1. Dispatch relación→resolver en el helper

- [x] 1.1 Extender el expansor para aceptar un registro inyectado `map[string]Resolver` (por nombre de relación) y despachar cada relación a su resolver específico, cayendo al genérico cuando no haya entrada; verificar con `go build ./...`
- [x] 1.2 Emitir el aviso `generic_resolver` solo cuando una relación usa el fallback genérico, y ninguno cuando usa un resolver específico; verificar con prueba unitaria de ambos caminos
- [x] 1.3 Agregar pruebas del dispatch: relación con resolver específico (no toca el genérico), relación sin resolver (usa genérico + aviso), y que el expansor no conoce el mapeo salvo por inyección; verificar con `go test ./helper/...`

## 2. Resolvers específicos del núcleo (envoltorios finos sobre HTTPResolver)

- [x] 2.1 Definir un envoltorio fino reutilizable sobre `HTTPResolver` que fije endpoint y campo de consulta por entidad, reutilizando transporte, timeouts y decodificación existentes; verificar con `go build ./...`
- [x] 2.2 Implementar los resolvers del núcleo (tercero por id; datos_identificacion por tercero_id; tipo_contribuyente; proveedor_natural/juridico por PK compartida; perfil; contacto por proveedor_id; cuenta_bancaria por proveedor_id) sin lógica de negocio; verificar con pruebas que confirman el campo de consulta correcto por entidad
- [x] 2.3 Agregar pruebas con mocks HTTP de que cada resolver anida el payload sin transformar (sin marcar, filtrar ni decidir ramas) y respeta cancelación por contexto; verificar con `go test ./...` del paquete de resolvers

## 3. ProveedorService (orquestación y negocio)

- [x] 3.1 Implementar `ProveedorService` que trae el proveedor base por `id` desde `gestion_personas_crud` y devuelve error diferenciado para id inválido y proveedor inexistente; verificar con pruebas de id válido, inválido e inexistente
- [x] 3.2 Construir `ExpandOptions` desde `fields`, registrar el mapeo relación→resolver e invocar el expansor; ubicar aquí la decisión de rama natural/jurídico (por tipo_registro) y la conservación del objeto; verificar con prueba de expansión de un proveedor con tercero y detalle
- [x] 3.3 Agregar pruebas de que la lógica de negocio reside en el servicio y no en los resolvers (ramas natural/jurídico, ciudades sin expandir como ID plano); verificar con `go test ./services/...`

## 4. Configuración declarativa del proveedor

- [x] 4.1 Crear `config/relaciones_proveedor.yaml` con el árbol del núcleo (tercero→datos_identificacion, tipo_contribuyente; natural/juridico→perfil; contacto; cuenta_bancaria), con `max_depth`, `max_related_ids` y `on_relation_error`; verificar que el loader existente lo carga y valida sin error
- [x] 4.2 Verificar que la configuración no nombra implementaciones de código y que una relación mal declarada aborta el arranque con mensaje que identifica archivo y relación; verificar con prueba de config inválida en `config/`

## 5. Controlador y ruta /v1/proveedor

- [x] 5.1 Implementar `controllers/proveedor.go` que lee `id` y `fields` de la query, delega a `ProveedorService` y serializa con el contrato OAS plano, omitiendo `MetaData` sin avisos; verificar con prueba de lectura exitosa
- [x] 5.2 Mapear resultados a códigos HTTP (`400` id/fields inválido, `200` ok, no encontrado, `206` expansión parcial tolerada, `502/504` en modo `fail`) y serializar avisos en `MetaData.Errors`; verificar con pruebas por cada situación
- [x] 5.3 Registrar la ruta `/v1/proveedor` en `main.go`; verificar que el API arranca con `relaciones_proveedor.yaml` cargado y la ruta responde

## 6. Validación de `fields` temprana y parseo único (fallar temprano)

- [ ] 6.1 Mover la validación de `fields` (contra la configuración declarativa del proveedor) por delante de la consulta al proveedor base en `ProveedorService.LeerProveedor`, de modo que un `fields` inválido devuelva `*helper.ErrorFields` sin invocar el CRUD dueño; verificar con prueba de que un `fields` inválido no produce ninguna llamada HTTP de lectura base
- [ ] 6.2 Consolidar el parseo de `fields` en una sola fuente de verdad (un único `ParsearFields` con el mismo conjunto `allowed`), eliminando el doble parseo con `allowed=nil` en el servicio y `allowedTopLevel` en `Expand`; verificar con prueba de que la validez de un `fields` es consistente entre la entrada y la expansión
- [ ] 6.3 Cubrir el escenario del spec "Validación de `fields` antes de consultar el CRUD" con prueba de controlador/servicio que afirme `400` y cero llamadas al CRUD ante rutas vacías, segmentos vacíos, duplicados y rutas no declaradas; verificar con `go test ./services/... ./controllers/...`

## 7. Eliminación de la composición manual (7387a47)

- [ ] 7.1 Eliminar `componerProveedor` y el `MarshalJSON` a medida de `PersonaIntegradaResultado` en `services/consulta_actualizacion.go`, migrando lo que siga siendo necesario; verificar con `go build ./...`
- [ ] 7.2 Eliminar los DTOs compuestos (`dto/registro_compuesto.go`, `dto/actualizar_compuesto.go`) y sus `UnmarshalJSON`, junto con sus tests asociados; verificar con `go build ./...` y que no queden referencias
- [ ] 7.3 Actualizar o retirar los tests de servicios/controladores que dependían del contrato compuesto manual; verificar con `go test ./...`

## 8. Verificación integral y documentación

- [ ] 8.1 Documentar el contrato público de `GET /v1/proveedor` (parámetros `id` y `fields`, forma de respuesta y `MetaData.Errors`) en Swagger; verificar que la definición refleja el contrato
- [ ] 8.2 Ejecutar `go build ./...` y `go test ./...` en verde sobre todo el proyecto, cubriendo helper, resolvers, servicio, config y controlador
