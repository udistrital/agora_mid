# Tasks

## 1. Pre-pass de resolución y retención sobre el árbol completo

- [ ] 1.1 Escribir la prueba (con marcador de rojo) de que, ante una violación en un nodo posterior del recorrido, el motor no envía ninguna escritura a ningún CRUD (inspección de peticiones = cero escrituras) porque resuelve y retiene el estado de todo el árbol antes de materializar; la prueba debe fallar honestamente contra el motor vigente, que escribe los subárboles `fk_inbound` antes de validar las ramas `fk`/`shared_pk`, y confirmar que falla
- [ ] 1.2 Escribir la prueba (con marcador de rojo) de que la recuperación previa devuelve los campos del registro existente (no solo su `Id`), usando únicamente lecturas (GET), para que la clasificación y el no-reparent dispongan del estado almacenado, y confirmar que falla
- [ ] 1.3 Implementar en `helper/write_engine.go` el pre-pass de solo lectura que recorre el árbol una sola vez, recupera y retiene el estado de cada nodo antes de la primera escritura, incorporando el camino `fk_inbound` existente a ese único recorrido; ampliar el contrato de recuperación del `WriteResolver` para devolver el registro existente y actualizar `services/write_resolver.go` para recuperar y devolver sus campos por lecturas GET; y verificar que las pruebas 1.1–1.2 pasan y se retiran sus marcadores

## 2. Clasificación de acción sobre el estado retenido

- [ ] 2.1 Escribir la prueba (con marcador de rojo) que, sobre el árbol resuelto y retenido por el pre-pass, verifica que cada nodo se clasifica como creación cuando no resuelve a un registro existente y como actualización cuando sí (por `Id` o por clave natural), y confirmar que falla antes de implementar
- [ ] 2.2 Escribir la prueba (con marcador de rojo) que distingue una actualización con cambios de una reconciliación/reenvío sin cambios de campos no modificados o derivados, confirmando que el reenvío sin cambios no se clasifica como actualización, y confirmar que falla
- [ ] 2.3 Implementar en `helper/write_engine.go` la clasificación de acción (crear vs actualizar) usando la misma resolución (Id + clave natural) que la idempotencia existente y comparando contra los campos del registro retenido, y verificar que las pruebas 2.1–2.2 pasan y se retiran sus marcadores

## 3. Gate de política de escritura por relación

- [ ] 3.1 Escribir la prueba (con marcador de rojo) de que una arista con `creatable: false` que recibe una entidad inexistente produce `400` nombrando el objeto/arista y no envía ninguna petición a los CRUD (inspección de peticiones), y confirmar que falla
- [ ] 3.2 Escribir la prueba (con marcador de rojo) de que una arista con `updatable: false` que recibe una entidad existente con cambios produce `400` nombrando el objeto/arista y no envía ninguna petición, incluido el caso en que la entidad resuelve por clave natural, y confirmar que falla
- [ ] 3.3 Escribir la prueba (con marcador de rojo) de que una arista `creatable: true` crea y una `updatable: true` actualiza con normalidad, y de que el reenvío sin cambios bajo `updatable: false` no se rechaza, y confirmar que falla
- [ ] 3.4 Implementar en `helper/write_engine.go` el gate de política que consulta `creatable`/`updatable` de la arista del árbol efectivo contra la acción clasificada, y verificar que las pruebas 3.1–3.3 pasan y se retiran sus marcadores

## 4. Gate y materialización de eliminación por relación

- [ ] 4.1 Escribir la prueba (con marcador de rojo) de que un nodo que el árbol compuesto marca para eliminación bajo una arista `deletable: false` produce `400` nombrando el objeto/arista y no envía ninguna petición a los CRUD, y confirmar que falla
- [ ] 4.2 Escribir la prueba (con marcador de rojo) de que un nodo marcado para eliminación bajo una arista `deletable: true` materializa la baja contra el CRUD dueño por la ruta de escritura existente (inspección de la petición enviada), y confirmar que falla
- [ ] 4.3 Escribir la prueba (con marcador de rojo) de que el motor no infiere ninguna eliminación cuando el árbol no marca nodos para borrado, aunque falten nodos respecto al estado del CRUD, y confirmar que falla
- [ ] 4.4 Implementar en `helper/write_engine.go` la clasificación de la acción de eliminación para un nodo marcado por el árbol compuesto, el gate por `deletable` y la materialización de la baja por la ruta de escritura existente, sin endpoint DELETE de controlador, y verificar que las pruebas 4.1–4.3 pasan y se retiran sus marcadores

## 5. Regla de no-reparent / pertenencia

- [ ] 5.1 Escribir la prueba (con marcador de rojo) de que un nodo con `Id` propio llegado bajo un padre cuya llave propagada difiere de la asociación actual del registro existente retenido produce `400` nombrando el objeto/arista y no envía ninguna petición, y confirmar que falla
- [ ] 5.2 Escribir la prueba (con marcador de rojo) de que un nodo con `Id` propio bajo el padre que ya lo posee, con llave propagada coincidente, no se trata como reparent y continúa con normalidad, y confirmar que falla
- [ ] 5.3 Actualizar la prueba del escenario "Entidad existente con relación incompatible" para que el rechazo ocurra antes de cualquier escritura por aplicación de la regla de no-reparent (marcador de rojo si cambia el comportamiento esperado), y confirmar que falla
- [ ] 5.4 Implementar en `helper/write_engine.go` la regla de no-reparent dentro del pre-pass, comparando la llave propagada por la arista contra la asociación actual del registro existente retenido, y verificar que las pruebas 5.1–5.3 pasan y se retiran sus marcadores

## 6. Garantía de validación total previa a toda escritura

- [ ] 6.1 Escribir la prueba (con marcador de rojo) de que una violación de política (crear/actualizar/eliminar) o de no-reparent en un nodo posterior del recorrido produce `400` sobre el árbol completo sin que se haya enviado ninguna creación, actualización ni eliminación a ningún CRUD (inspección de peticiones = cero escrituras), y confirmar que falla
- [ ] 6.2 Escribir la prueba de regresión (con marcador de rojo si aplica) que confirma que el gate de política, el gate de eliminación y el no-reparent se evalúan dentro del mismo pre-pass de solo lectura que inmutables y campos derivados, sin introducir un recorrido ni una ruta de error adicionales, y confirmar que falla
- [ ] 6.3 Verificar que todos los controles se evalúan sobre el árbol completo antes de la primera escritura reutilizando el estado retenido por el pre-pass, y verificar que las pruebas 6.1–6.2 pasan y se retiran sus marcadores

## 7. Pruebas de regresión y verificación de integración

- [ ] 7.1 Confirmar como cobertura/regresión que los escenarios vigentes de campo inmutable incompatible y campo de escritura no permitido siguen rechazándose antes de cualquier escritura tras los cambios, con las pruebas existentes en verde
- [ ] 7.2 Ejecutar `go test ./...` y `go build ./...` y verificar que toda la suite pasa sin marcadores de rojo remanentes
