# Tasks

## 1. Prueba roja de integración (TDD: RED)

- [x] 1.1 Añadir en `test/integration_test.go` una prueba que haga `PATCH /v1/proveedor` sobre un proveedor existente (con `Id`) reenviando un `TerceroId` incompatible con el `tercero` resuelto, y que afirme `rec.Code == 400`; verificar ejecutándola con `go test ./test/ -run <NombrePrueba>` y confirmar que FALLA por la aserción de "cero escrituras" (no por compilación ni por el 400), observando el fallo en la salida
- [x] 1.2 En la misma prueba, afirmar con `peticionesPorMetodo(capturas, "tercero")` y `peticionesPorMetodo(capturas, "proveedor")` que `post == 0 && patch == 0` para ambos recursos (GET permitidos); verificar que la prueba sigue roja porque hoy `tercero` recibe una escritura antes del 400, citando el conteo observado

## 2. Separar recuperación de materialización en el contrato (TDD: RED→GREEN)

- [x] 2.1 Añadir una prueba de unidad en `services/proveedor_write_test.go` (o archivo nuevo de resolver) que verifique que la nueva operación de recuperación del `WriteResolver` resuelve el `Id` existente usando solo GET (sin POST/PATCH); ejecutar con `go test ./services/ -run <NombrePrueba>` y confirmar que falla porque la operación no existe aún
- [x] 2.2 Extender el contrato `WriteResolver` en `helper/write_engine.go` con una operación de recuperación por lectura y implementarla en `EntityWriteResolver` (`services/write_resolver.go`) reutilizando la lógica de `resolverExistente`, sin que ejecute escrituras; verificar que la prueba 2.1 pasa y que `go build ./...` compila

## 3. Reordenar materializarInbound a recuperar → conciliar → materializar (TDD: RED→GREEN)

- [x] 3.1 Añadir una prueba de unidad en `helper/write_engine_test.go` que, con un spy de resolver, verifique que ante un conflicto de `fk_inbound` en entidad existente no se invoca `Materialize` de la entidad referenciada ni del nodo y se devuelve `*ErrorEscritura`; ejecutar con `go test ./helper/ -run <NombrePrueba>` y confirmar que falla (hoy se materializa antes de conciliar)
- [x] 3.2 Reordenar `materializarInbound` para recuperar el `Id` autoritativo por lectura, conciliar la FK local del nodo con `conciliarCampoDerivado` y materializar solo si no hay conflicto; verificar que la prueba 3.1 pasa
- [x] 3.3 Ajustar las pruebas `fk_inbound` existentes en `helper/write_engine_test.go` (canonización de entidad nueva 206, propagación de tipo numérico) a la nueva semántica sin cambiar su intención, y verificar que todas pasan con `go test ./helper/`

## 4. Fallo de recuperación bajo on_relation_error (TDD: RED→GREEN)

- [x] 4.1 Añadir una prueba en `helper/write_engine_test.go` que, con un resolver cuya recuperación falla, verifique que en modo tolerante se agrega un aviso y la cascada continúa, y en modo estricto se aborta con `*errorFatalEscritura`; ejecutar con `go test ./helper/ -run <NombrePrueba>` y confirmar que falla
- [x] 4.2 Enrutar el fallo de la recuperación por `manejarErrorRelacion` igual que un fallo de escritura; verificar que la prueba 4.1 pasa

## 5. Verificación de integración y regresión

- [x] 5.1 Confirmar que la prueba roja del grupo 1 ahora pasa (400 con cero POST/PATCH en `tercero` y `proveedor`) ejecutándola de nuevo
- [x] 5.2 Auditar el resto de `helper/write_engine.go` para confirmar que ningún otro punto escribe antes de una validación que pueda dar `400` (conflictos descendentes incluidos); documentar el hallazgo en el resumen de aplicación
- [x] 5.3 Ejecutar la suite completa con `go test ./...` y `go build ./...` y confirmar que todo pasa sin errores ni advertencias; reportar por nombre cualquier fallo observado
