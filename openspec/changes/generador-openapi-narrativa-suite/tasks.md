# Tasks

## 1. Mapeo de tipo de almacenamiento a tipo OpenAPI

- [ ] 1.1 Escribir pruebas unitarias que afirmen el mapeo de cada tipo del vocabulario cerrado (`boolean`, `integer`, `numeric`, `string`, `uuid`, `timestamp`, `jsonb`) a su `type`/`format` OpenAPI, incluido que `integer` se describe como entero para llaves; confirmar que fallan antes de implementar el mapeo.
- [ ] 1.2 Implementar la función de mapeo tipo de almacenamiento -> tipo/formato OpenAPI y verificar que las pruebas de 1.1 pasan para todo el vocabulario.

## 2. Generador config -> swagger.yml determinístico

- [ ] 2.1 Escribir pruebas que, a partir de una configuración de prueba (objetos tipados + árbol efectivo), afirmen que el generador produce un `swagger.yml` que describe los objetos, campos (con el mapeo de tipos) y endpoints esperados; confirmar que fallan antes de implementar el generador.
- [ ] 2.2 Escribir una prueba de determinismo que afirme que generar dos veces desde la misma configuración produce bytes idénticos, con orden estable de claves y colecciones; confirmar que falla antes de implementar el orden estable.
- [ ] 2.3 Implementar el generador `config -> swagger.yml` con salida byte-estable y verificar que las pruebas de 2.1 y 2.2 pasan.

## 3. Integración de la cadena en el arranque

- [ ] 3.1 Escribir una prueba que afirme la regeneración escribe-si-cambio de `swagger.yml`: configuración cambiada actualiza el archivo; configuración sin cambios lo deja intacto y no reescribe `swagger.json`; confirmar que falla antes de anteponer la etapa.
- [ ] 3.2 Anteponer en `main` la generación `config -> swagger.yml` antes de la derivación `generarSwaggerJSON` existente, conservando esa derivación sin cambios, y verificar que las pruebas de 3.1 pasan.
- [ ] 3.3 Escribir y verificar una prueba de que una configuración válida produce un `swagger.yml` generable y que el aborto de arranque de la derivación a JSON se conserva ante un YAML generado inválido.

## 4. Narrativa de la suite de pruebas

- [ ] 4.1 Reescribir `test/tests.md` para describir los flujos del subárbol #58 (registro natural, registro jurídico con rechazo de ramas combinadas) con entradas y salidas esperadas, y verificar que cada flujo documentado corresponde a una prueba existente de la suite.
- [ ] 4.2 Añadir a `test/tests.md` la narrativa del contrato de respuesta (`206` de stop-abort con `Kind`, reintento idempotente por `Id`/clave natural y transición de estado por `estado_id`) y verificar que describe las pruebas vigentes sin describir pruebas inexistentes.

## 5. Verificación de integración de la publicación del contrato

- [ ] 5.1 Ejecutar `go test ./...` y `go build ./...` y verificar que la suite pasa, cubriendo el generador, el mapeo de tipos, la regeneración en arranque y la coherencia del contrato publicado en `/swagger`.
- [ ] 5.2 Verificar con `openspec validate generador-openapi-narrativa-suite --strict` que el cambio valida sin advertencias.

## Workflow follow-up

- Archivar el cambio tras satisfacer los requisitos de revisión del proyecto.
- Verificar el resultado archivado.
