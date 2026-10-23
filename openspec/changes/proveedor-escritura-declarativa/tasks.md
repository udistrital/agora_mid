# Tasks

## 1. Configuración declarativa de escritura (bloque `write`)

- [x] 1.1 Extender el esquema de configuración de relaciones con un bloque `write` namespaced por relación (campos escribibles, inmutables, clave natural, modo de propagación de llave `fk`/`shared_pk`, políticas por verbo); verificar con `go build ./...`
- [x] 1.2 Declarar el bloque `write` del subárbol del proveedor en `config/relaciones_proveedor.yaml` (tercero, datos_identificacion, proveedor, proveedor_natural/juridico, contacto, cuenta_bancaria); tercero y datos_identificacion creatable/updatable y NO deletable; catálogos solo referencia por Id
- [x] 1.3 Validar en el loader la coherencia lectura/escritura (toda relación `write` existe en el árbol; propagación de llave válida) y abortar el arranque con mensaje que identifique archivo y relación; verificar con pruebas de config coherente e incoherente

## 2. Motor de escritura declarativo (`WriteEngine` genérico en helper)

- [x] 2.1 Implementar el recorrido topológico del árbol (padres antes que hijos) y la propagación de llave del padre al hijo según el modo declarado (`fk` y `shared_pk`); verificar con pruebas de ambos modos
- [x] 2.2 Implementar el aplanado de parámetros anidados a FK (`{X:{Id:n}}` -> `XId:n`) y el rechazo `400` de campos no escribibles o inmutables sin materializar nada; verificar con pruebas de aplanado y de rechazo
- [x] 2.3 Implementar el reporte tolerante/estricto espejo de lectura (bloqueante vs no bloqueante, conservar lo materializado, avisos sin datos sensibles); verificar con `go test ./helper/...`
- [x] 2.4 Garantizar la frontera de capas: `WriteEngine` genérico en `helper/` sin nombrar entidades de dominio; verificar con grep de que `helper/` no referencia tablas ni rutas `/v1/x`
- [x] 2.5 Añadir al motor el modo de propagación `fk_inbound`: materializar la entidad referenciada por el nodo (p. ej. `tercero`) antes que el nodo y propagar su `Id` a la FK del nodo (`TerceroId`); ajustar el orden topológico para resolver las relaciones `fk_inbound` antes del nodo que las referencia; verificar con pruebas del nuevo modo

## 3. Write-resolvers del núcleo (en services) e idempotencia

- [x] 3.1 Definir el write-resolver fino reutilizable (endpoint real + upsert por clave natural/`Id`) y el cableado del árbol escribible del proveedor en `services/`; verificar con `go build ./...`
- [x] 3.2 Implementar los write-resolvers del núcleo con su idempotencia (proveedor por `tercero_id`; datos_identificacion por `numero`+`tipo_documento_id`; detalle por PK compartida; colecciones por `Id` del cuerpo) e inyección del autor de modificación desde el contexto; verificar con pruebas por entidad
- [x] 3.3 Ubicar las reglas de negocio irreductibles en el servicio/write-resolver (consistencia natural/jurídico, inmutables) y no en el motor; verificar con `go test ./services/...`
- [x] 3.4 Corregir en `config/relaciones_proveedor.yaml` la propagación de `Tercero`: de `shared_pk` a `fk_inbound` (FK `proveedor.TerceroId → tercero.Id`), y actualizar el comentario que la describía como PK compartida; verificar con el loader de coherencia y `go build ./...`

## 4. ProveedorService de escritura y controlador POST/PATCH

- [x] 4.1 Implementar el servicio de escritura que arma el árbol desde el cuerpo anidado, invoca el `WriteEngine` con el registro de write-resolvers y devuelve los identificadores materializados; verificar con prueba de creación completa
- [x] 4.2 Implementar el controlador `POST`/`PATCH /v1/proveedor`: mapear resultado a `200` (éxito), `206` + `MetaData.Errors` (parcial), `400` (campo/estructura inválida) y `502/504` (fallo bloqueante); verificar con pruebas por situación
- [x] 4.3 Registrar las rutas de escritura de `/v1/proveedor` en `main.go`; verificar que el API arranca con el bloque `write` cargado y las rutas responden

## 5. Retiro BREAKING de los flujos estáticos de persona

- [x] 5.1 Eliminar `RegistrarPersonaNatural/Juridica`, `ActualizarPersona` y sus ayudantes de `services/`, y los DTOs compuestos (`dto/registro_compuesto.go`, `dto/actualizar_compuesto.go`); verificar con `go build ./...` y que no queden referencias
- [x] 5.2 Retirar las rutas `POST/PUT/GET /v1/persona_*` de `main.go` y el controlador asociado; verificar que el API arranca sin ellas
- [x] 5.3 Actualizar o retirar los tests que dependían de los flujos estáticos; verificar con `go test ./...`
- [x] 5.4 Eliminar la capa estática residual que quedó sin llamadores de producción tras retirar los orquestadores: `RegistrarOReutilizar` (`services/registro_tercero.go`) y sus ayudantes (`crearIdentificacion`, `filtrarVacios`, `parseFecha*`, `ErrorRegistroParcial`), `CrearOReutilizarProveedor` (`services/registro_proveedor.go`) con `validarProveedorExistente` y `ErrTipoProveedorIncompatible`, todo `services/gestion_personas.go` (`GestionPersonasService`, `BuscarProveedorPorTercero`), el DTO `dto/datos_terceros.go` (`DatosTercerosDTO`, único consumidor del paquete `dto/`), y las funciones muertas `BuscarPorDocumento`/`identificacionResponse`/`ErrTerceroAmbiguo` (`services/terceros.go`) y `ResolverTipoDocumento` (`services/catalogos.go`); conservar `TercerosService` y `ResolverTipoContribuyente`, que `main.go` sigue usando para resolver `P_NATURAL` al arranque; verificar con `go build ./...` y grep de que no quedan referencias
- [x] 5.5 Retirar los tests de la capa estática residual (`registro_tercero_test.go`, `registro_tercero_idempotencia_test.go`, `registro_proveedor_test.go`, `gestion_personas_test.go`, `TestBuscarPorDocumento` en `terceros_test.go`); verificar con `go test ./...`

## 6. Pruebas de integración y e2e

- [ ] 6.1 Añadir pruebas de integración locales en `/test` (router + CRUD simulados) para creación y reconciliación (`PATCH`) del subárbol, incluyendo el escenario de continuación tras creación parcial
- [ ] 6.2 Añadir al menos una prueba e2e de un flujo completo de creación/actualización del subárbol del proveedor; centralizar fixtures/mocks compartidos
- [ ] 6.3 Ejecutar `go build ./...` y `go test ./...` en verde sobre todo el proyecto
- [ ] 6.4 Documentar la suite en `test/tests.md`: por cada prueba de integración y e2e, su propósito, actores (cliente del API, MID, CRUD dueños simulados), entradas dadas (verbo, cuerpo, estado precargado) y salidas esperadas (código, envoltorio, interacciones con los CRUD); verificar que cubre los flujos de creación, reconciliación y continuación tras creación parcial

## 7. Documentación pública del contrato de escritura

- [ ] 7.1 Actualizar `swagger/swagger.yml` con `POST`/`PATCH /v1/proveedor`: cuerpo anidado del subárbol, respuestas `200`, `206` con `MetaData.Errors`, `400` y `502/504`; verificar que `swagger/swagger.json` se deriva sin error
- [ ] 7.2 Actualizar `API.md` con los endpoints de escritura `POST`/`PATCH /v1/proveedor` (parámetros, cuerpo, códigos de respuesta y reglas) y retirar el contrato de las rutas `/v1/persona_*`; no modificar `README.md`
- [ ] 7.3 Verificar la coherencia del contrato publicado con el comportamiento implementado (endpoints de escritura presentes, `/v1/persona_*` ausentes)
- [x] 7.4 Corregir `swagger/index.html` para que la interfaz de `/swagger` cargue `./swagger.json` (contrato local) en vez del contrato de demostración `petstore`, y añadir la excepción `!swagger/index.html` en `.gitignore` para versionar el arreglo; `README.md` no se modifica
