# Tasks

## 1. Configuración declarativa de escritura (bloque `write`)

- [ ] 1.1 Extender el esquema de configuración de relaciones con un bloque `write` namespaced por relación (campos escribibles, inmutables, clave natural, modo de propagación de llave `fk`/`shared_pk`, políticas por verbo); verificar con `go build ./...`
- [ ] 1.2 Declarar el bloque `write` del subárbol del proveedor en `config/relaciones_proveedor.yaml` (tercero, datos_identificacion, proveedor, proveedor_natural/juridico, contacto, cuenta_bancaria); tercero y datos_identificacion creatable/updatable y NO deletable; catálogos solo referencia por Id
- [ ] 1.3 Validar en el loader la coherencia lectura/escritura (toda relación `write` existe en el árbol; propagación de llave válida) y abortar el arranque con mensaje que identifique archivo y relación; verificar con pruebas de config coherente e incoherente

## 2. Motor de escritura declarativo (`WriteEngine` genérico en helper)

- [ ] 2.1 Implementar el recorrido topológico del árbol (padres antes que hijos) y la propagación de llave del padre al hijo según el modo declarado (`fk` y `shared_pk`); verificar con pruebas de ambos modos
- [ ] 2.2 Implementar el aplanado de parámetros anidados a FK (`{X:{Id:n}}` -> `XId:n`) y el rechazo `400` de campos no escribibles o inmutables sin materializar nada; verificar con pruebas de aplanado y de rechazo
- [ ] 2.3 Implementar el reporte tolerante/estricto espejo de lectura (bloqueante vs no bloqueante, conservar lo materializado, avisos sin datos sensibles); verificar con `go test ./helper/...`
- [ ] 2.4 Garantizar la frontera de capas: `WriteEngine` genérico en `helper/` sin nombrar entidades de dominio; verificar con grep de que `helper/` no referencia tablas ni rutas `/v1/x`

## 3. Write-resolvers del núcleo (en services) e idempotencia

- [ ] 3.1 Definir el write-resolver fino reutilizable (endpoint real + upsert por clave natural/`Id`) y el cableado del árbol escribible del proveedor en `services/`; verificar con `go build ./...`
- [ ] 3.2 Implementar los write-resolvers del núcleo con su idempotencia (proveedor por `tercero_id`; datos_identificacion por `numero`+`tipo_documento_id`; detalle por PK compartida; colecciones por `Id` del cuerpo) e inyección del autor de modificación desde el contexto; verificar con pruebas por entidad
- [ ] 3.3 Ubicar las reglas de negocio irreductibles en el servicio/write-resolver (consistencia natural/jurídico, inmutables) y no en el motor; verificar con `go test ./services/...`

## 4. ProveedorService de escritura y controlador POST/PATCH

- [ ] 4.1 Implementar el servicio de escritura que arma el árbol desde el cuerpo anidado, invoca el `WriteEngine` con el registro de write-resolvers y devuelve los identificadores materializados; verificar con prueba de creación completa
- [ ] 4.2 Implementar el controlador `POST`/`PATCH /v1/proveedor`: mapear resultado a `200` (éxito), `206` + `MetaData.Errors` (parcial), `400` (campo/estructura inválida) y `502/504` (fallo bloqueante); verificar con pruebas por situación
- [ ] 4.3 Registrar las rutas de escritura de `/v1/proveedor` en `main.go`; verificar que el API arranca con el bloque `write` cargado y las rutas responden

## 5. Retiro BREAKING de los flujos estáticos de persona

- [ ] 5.1 Eliminar `RegistrarPersonaNatural/Juridica`, `ActualizarPersona` y sus ayudantes de `services/`, y los DTOs compuestos (`dto/registro_compuesto.go`, `dto/actualizar_compuesto.go`); verificar con `go build ./...` y que no queden referencias
- [ ] 5.2 Retirar las rutas `POST/PUT/GET /v1/persona_*` de `main.go` y el controlador asociado; verificar que el API arranca sin ellas
- [ ] 5.3 Actualizar o retirar los tests que dependían de los flujos estáticos; verificar con `go test ./...`

## 6. Pruebas de integración y e2e

- [ ] 6.1 Añadir pruebas de integración locales en `/test` (router + CRUD simulados) para creación y reconciliación (`PATCH`) del subárbol, incluyendo el escenario de continuación tras creación parcial
- [ ] 6.2 Añadir al menos una prueba e2e de un flujo completo de creación/actualización del subárbol del proveedor; centralizar fixtures/mocks compartidos
- [ ] 6.3 Ejecutar `go build ./...` y `go test ./...` en verde sobre todo el proyecto

## 7. Estrategia de DELETE (diferida, al final)

- [ ] 7.1 Validar con el equipo la estrategia de baja (probable soft-delete `activo=false`, alcance de la cascada, tratamiento de `tercero`); registrar la decisión en design.md antes de implementar
- [ ] 7.2 Implementar `DELETE /v1/proveedor` según la estrategia acordada, con su política de cascada y reporte; verificar con pruebas de baja total y parcial
- [ ] 7.3 Documentar en Swagger el contrato público de escritura (`POST`/`PATCH`/`DELETE /v1/proveedor`, cuerpo anidado, respuesta y `MetaData.Errors`); verificar que la definición refleja el contrato
