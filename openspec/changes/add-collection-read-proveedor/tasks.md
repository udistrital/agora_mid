# Tasks

## 1. Soporte de configuración para campos filtrables

- [x] 1.1 Añadir una prueba en `config/relaciones_proveedor_test.go` (o el test del loader) que falle si la configuración del recurso raíz no admite declarar los campos filtrables de primer nivel; ejecutar `go test ./config/` y confirmar que FALLA porque el soporte no existe aún
- [x] 1.2 Extender la configuración del servicio para declarar los campos filtrables de primer nivel y su carga en `config/relaciones_loader.go`, declararlos en `config/relaciones_proveedor.yaml` con un comentario que indique que solo admiten primer nivel, y verificar que la prueba 1.1 pasa y `go build ./...` compila

## 2. Prueba de integración primero (roja)

- [ ] 2.1 Escribir en `test/integration_test.go` la prueba de integración del listado contra CRUD simulados: cuerpo con `query` (incluyendo una condición de igualdad y una `__in` separada por `|`), `fields` y paginación, afirmando la lista expandida, que las condiciones se trasladan al `query` del CRUD combinadas con AND, que `MetaData.Total` refleja el `Total` entregado por el CRUD (y no el tamaño de la página) y las interacciones con los CRUD; incluir un caso límite de condición inválida (operador no soportado o campo de relación) que responde `400` sin consultar el CRUD; ejecutar `go test ./test/` y confirmar que FALLA por ausencia de la funcionalidad (no por una aserción trivial), reportando el fallo observado

## 3. Decodificación del total y motor de listado genérico en helper

- [ ] 3.1 Escribir en `services/api_response_test.go` una prueba que verifique que la decodificación del envoltorio de lista del CRUD captura el campo `Total` además de `Data`; ejecutar `go test ./services/` y confirmar que FALLA porque `decodeAPIList` hoy descarta `Total`
- [ ] 3.2 Escribir en `helper/` (nuevo archivo de test del listado) las pruebas del motor: expande cada elemento reutilizando la expansión por elemento, acumula los avisos de todos los elementos, propaga el total del conjunto hacia la salida, devuelve lista vacía ante cero registros y, bajo política tolerante, conserva un elemento con su rama fallida como `null`/`[]` con aviso sin abortar el listado; ejecutar `go test ./helper/` y confirmar que FALLAN porque el motor no existe
- [ ] 3.3 Extender la decodificación del envoltorio del CRUD en `services/api_response.go` para capturar `Total`, e implementar el motor de listado en `helper/` reutilizando `helper.Expand` por elemento con las mismas `ExpandOptions` (selección y profundidad), acumulando avisos, honrando la política de error de relación y exponiendo el total del conjunto en su resultado; verificar que las pruebas 3.1 y 3.2 pasan y `go build ./...` compila

## 4. Servicio de listado del proveedor

- [ ] 4.1 Escribir en `services/proveedor_test.go` las pruebas del caso de uso cubriendo entradas, salidas y casos límite: parseo y validación de cada condición de `query` (`Campo[__operador]:valor`) ANTES de consultar el CRUD —campo filtrable válido con igualdad implícita y con operador explícito, campo no filtrable, condición de ruta de relación de segundo nivel, operador fuera del conjunto beego, `__in` con lista vacía o sin separador `|`, operador escalar con varios valores, `isnull` con valor no booleano— afirmando que las válidas se trasladan al `query` del CRUD combinadas con AND y las inválidas devuelven error sin llamar al CRUD; más traslado de `limit`/`offset`, defaults, máximo y valores inválidos, y propagación del `Total` del CRUD al resultado; ejecutar `go test ./services/` y confirmar que FALLAN porque el caso de uso no existe
- [ ] 4.2 Implementar el caso de uso de listado en `services/proveedor.go` que parsea y valida `query` (campo filtrable + operador del conjunto beego `exact/iexact/contains/icontains/gt/gte/lt/lte/startswith/istartswith/endswith/iendswith/in/isnull` + coherencia valor↔operador, con `|` como separador de `__in`), construye el `query` del CRUD, traslada `limit`/`offset`, decodifica la lista de proveedores raíz con su `Total` e invoca el motor de listado; verificar que las pruebas 4.1 pasan y `go build ./...` compila

## 5. Endpoint de listado en el controlador

- [ ] 5.1 Escribir en `controllers/proveedor_test.go` las pruebas del endpoint `POST /v1/proveedor/consulta/lista` cubriendo entradas, salidas y casos límite: con un espía del servicio, afirmar que el handler decodifica `query`, `fields`, `limit` y `offset` del cuerpo y los pasa al servicio con los valores recibidos (incluido el caso de parámetros omitidos que aplican defaults y el cuerpo vacío); `200` con lista en `Data` y `MetaData.Total` igual al total del CRUD; `400` por cuerpo malformado o por el error de validación del servicio (`query`/`fields` inválido) sin consultar el CRUD; y `405` para métodos distintos de `POST` anunciando `POST`; ejecutar `go test ./controllers/` y confirmar que FALLAN porque el endpoint no existe
- [ ] 5.2 Implementar el handler del listado en `controllers/proveedor.go` que decodifica el cuerpo (`query`, `fields`, `limit`, `offset`) y lo pasa al servicio, registrar la ruta `POST /v1/proveedor/consulta/lista` y exponer el total del conjunto en `MetaData.Total` del envoltorio de respuesta, mapeando los errores del servicio sin exponer mensajes internos de los CRUD; verificar que las pruebas 5.1 y la prueba de integración 2.1 pasan y `go build ./...` compila

## 6. Publicación del contrato OpenAPI (openapi-publication)

- [ ] 6.1 Describir el endpoint `POST /v1/proveedor/consulta/lista` en el YAML fuente `swagger/swagger.yml` (cuerpo con `query` como lista de condiciones `Campo[__operador]:valor` de primer nivel con los operadores soportados y `|` como separador de `__in`, `fields`, `limit`, `offset`; respuestas `200`, `206`, `400`, `405`, `502/504`; `Data` como lista de proveedores expandidos; `MetaData.Total` con el total del conjunto) y documentarlo en `API.md`, de modo que ambos sean coherentes con el comportamiento implementado
- [ ] 6.2 Verificar que el JSON derivado (`swagger/swagger.json`) refleja el YAML vigente de forma determinística tras el arranque del API (regeneración idempotente) y que la interfaz de `/swagger` carga el contrato local con el nuevo endpoint; comprobar que el contrato no introduce rutas inexistentes

## 7. Documentación de la suite de pruebas (test-documentation)

- [ ] 7.1 Documentar en `test/tests.md` la prueba de integración del listado (2.1) con su propósito, actores (cliente del API, middleware y CRUD dueños simulados), entradas (verbo, cuerpo con filtros/`fields`/paginación y estado precargado de los CRUD) y salidas esperadas (código de estado, envoltorio de respuesta, `MetaData.Total` e interacciones con los CRUD), manteniendo el documento coherente con la suite vigente

## 8. Verificación de integración y regresión

- [ ] 8.1 Ejecutar `go test ./...` y `go build ./...` y confirmar que todo pasa sin errores ni advertencias, reportando por nombre cualquier fallo observado
