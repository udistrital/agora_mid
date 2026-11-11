# Tasks

## 1. Tabla de campos tipada unificada por objeto

- [x] 1.1 Escribir pruebas (en rojo) del modelo de tabla de campos tipada por objeto: un campo lleva `type` y banderas `readable`/`writable`/`immutable`/`required`; verificar que se parsea y que un campo sin `type` produce error de modelo. Verificación: `go test ./config/...` falla en estas pruebas antes de implementar
- [x] 1.2 Escribir pruebas (en rojo) de validación de coherencia por campo: `immutable:true` con `writable:false` aborta; `required:true` con `writable:false` aborta; nombrar objeto y campo. Verificación: las pruebas fallan antes de implementar
- [x] 1.3 Implementar el tipo de tabla de campos tipada y su parseo en `config/relaciones.go` (o archivo nuevo del paquete `config`) hasta pasar 1.1. Verificación: `go test ./config/...` pasa esas pruebas
- [x] 1.4 Implementar la validación de coherencia por campo hasta pasar 1.2. Verificación: `go test ./config/...` pasa esas pruebas
- [x] 1.4a Escribir pruebas (en rojo) del vocabulario cerrado de tipos: se aceptan `boolean`, `integer`, `jsonb`, `numeric`, `string`, `timestamp`, `uuid`; un tipo fuera del conjunto aborta el arranque nombrando objeto, campo y tipo. Verificación: `go test ./config/...` falla antes de implementar
- [x] 1.4b Implementar la validación del vocabulario cerrado de tipos hasta pasar 1.4a. Verificación: `go test ./config/...` pasa esas pruebas
- [x] 1.5 Escribir pruebas (en rojo) de la proyección de la tabla tipada a la vista de lectura (campos `readable`) y a la matriz de escritura (campos `writable`/`immutable`), equivalentes a `allowed_fields` y `writable_fields`/`immutable_fields` actuales para un objeto de ejemplo. Verificación: las pruebas fallan antes de implementar
- [x] 1.6 Implementar las proyecciones de lectura y escritura desde la tabla tipada hasta pasar 1.5. Verificación: `go test ./config/...` pasa esas pruebas

## 2. Objetos direccionables y referencia con semántica de arista

- [x] 2.1 Escribir pruebas (en rojo) del modelo de objeto direccionable (definición con nombre único + tabla de campos + referencias) y de la referencia de servicio que porta la arista (`direction`, `cardinality`, `key_propagation`, `policy`, `depth`, `mapped_name`). Verificación: `go test ./config/...` falla en estas pruebas
- [x] 2.2 Escribir pruebas (en rojo) de que un mismo objeto referenciado por dos servicios/referenciadores con aristas distintas produce nodos de árbol con aristas independientes, y de que un nombre de objeto duplicado aborta el arranque nombrando el nombre duplicado. Verificación: las pruebas fallan antes de implementar
- [x] 2.3 Implementar el modelo de objeto direccionable y de referencia con arista en `config/relaciones.go` hasta pasar 2.1 y 2.2. Verificación: `go test ./config/...` pasa esas pruebas
- [x] 2.4 Escribir pruebas (en rojo) de la llave lógica de servicio del objeto: se deriva del nombre a `snake_case` tratando las mayúsculas consecutivas como una sola sigla (`HTTPServer` → `http_server`), y una llave explícita en el objeto prevalece sobre la derivación. Verificación: `go test ./config/...` falla antes de implementar
- [x] 2.5 Implementar la llave lógica de servicio (campo explícito del objeto + derivación `snake_case` con siglas como una palabra) y su uso al componer el nodo del árbol efectivo hasta pasar 2.4. Verificación: `go test ./config/...` pasa esas pruebas

## 3. Cargador de composición y aplanado del árbol efectivo

- [x] 3.1 Escribir pruebas (en rojo) de que el cargador resuelve referencias y produce un árbol efectivo por servicio en la forma que consumen los motores, con los campos ya proyectados a vistas de lectura y escritura. Verificación: `go test ./config/...` falla en estas pruebas
- [x] 3.2 Escribir pruebas (en rojo) de integridad de composición al arranque: ciclo de referencias entre objetos aborta nombrando el fragmento que cierra el ciclo; referencia colgante aborta nombrando el objeto ausente y quién lo referencia; profundidad de composición excedida aborta nombrando la referencia; objeto huérfano se reporta por su nombre y no entra a ningún árbol efectivo. Verificación: las pruebas fallan antes de implementar
- [x] 3.3 Implementar el cargador de composición (resolución + aplanado del árbol efectivo) en `config/relaciones_loader.go` hasta pasar 3.1. Verificación: `go test ./config/...` pasa esas pruebas
- [x] 3.4 Implementar la validación de integridad de composición con identidad de error por fragmento hasta pasar 3.2. Verificación: `go test ./config/...` pasa esas pruebas
- [x] 3.5 Escribir prueba (en rojo) que distingue la integridad de composición (falla de arranque) del ciclo de datos de expansión en tiempo de petición (aviso `cycle`, sin error HTTP), confirmando que el comportamiento de expansión en ruta no cambia. Verificación: la prueba falla antes de implementar y pasa después

## 4. Reescritura de la configuración del proveedor preservando el árbol efectivo

- [x] 4.1 Escribir prueba de equivalencia (en rojo) que compara el árbol efectivo compuesto del servicio `proveedor` contra el árbol de relaciones esperado (equivalente al inline actual), nodo a nodo, incluyendo propagación de llave y campos de lectura/escritura por objeto. Verificación: `go test ./config/...` falla antes de reescribir la configuración
- [x] 4.2 Reescribir `config/relaciones_proveedor.yaml` de árbol inline a definiciones de objeto direccionables + referencias de servicio, preservando el árbol efectivo. Verificación: la prueba 4.1 pasa
- [x] 4.3 Actualizar los comentarios inline de la configuración del proveedor para documentar la forma de objetos direccionables + referencias y la tabla de campos tipada. Verificación: revisión de que cada objeto y referencia lleva comentario explicativo coherente con el nuevo formato

## 5. Equivalencia de comportamiento de lectura y escritura del proveedor

- [x] 5.1 Escribir pruebas de equivalencia de lectura (en rojo) que ejercitan la expansión del proveedor (sin `fields`, con `fields`, con lista negra, Id implícito) sobre el árbol efectivo compuesto y verifican salida idéntica a la del comportamiento vigente. Verificación: `go test ./...` falla antes de que los motores consuman el árbol compuesto
- [x] 5.2 Escribir pruebas de equivalencia de escritura (en rojo) que ejercitan la cascada del proveedor (natural, jurídico, contacto, cuenta bancaria, propagación de llave, campo inmutable derivado, idempotencia por clave natural) sobre el árbol efectivo compuesto y verifican comportamiento idéntico. Verificación: `go test ./...` falla antes de conectar el árbol compuesto
- [x] 5.3 Conectar los motores de `helper/expand.go` y `helper/write_engine.go` para que consuman el árbol efectivo compuesto por el cargador, sin cambiar su lógica. Verificación: las pruebas 5.1 y 5.2 pasan y la suite completa `go test ./...` queda verde
- [x] 5.4 Ejecutar `go build ./...` y `go test ./...` completos y confirmar que no hay regresiones en ningún paquete. Verificación: ambos comandos terminan sin error

## 6. Documentación del formato de configuración

- [x] 6.1 Crear `config/README.md` en español (nuevo; el README raíz no se modifica para esto) que documente el formato de configuración: objetos direccionables, tabla de campos tipada unificada con su vocabulario cerrado de almacenamiento, referencia de servicio con semántica de arista, y la validación de integridad de composición al arranque (qué aborta el arranque y qué se reporta como huérfano). Verificación: el README describe el formato tal como lo cargan las pruebas de los grupos 1-4, y cada ejemplo YAML del README se parsea sin error con el cargador
