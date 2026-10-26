# Tasks

## 1. Endpoint de consulta POST

- [x] 1.1 Agregar el manejador `Consultar` en `controllers/proveedor.go` que decodifica el cuerpo JSON `{id, fields}`, rechaza con `400` el cuerpo no-JSON o `id` ausente/no positivo, acepta `fields` como arreglo de strings, valida cada ruta reutilizando la validación existente e invoca `LeerProveedor(ctx, id, fields)`; verificar con una prueba de tabla que cubra `200`, `id` inválido/ausente (`400`) y cuerpo malformado (`400`)
- [x] 1.2 Registrar la ruta `POST /v1/proveedor/consulta` en `main.go` (nueva ruta separada del raíz `/v1/proveedor`) y responder `405` con `Allow: POST` a `GET`/`PUT`/`DELETE` sobre `/consulta`; verificar con pruebas que `POST` enruta a `Consultar` y que otros métodos devuelven `405`
- [x] 1.3 Portar los casos de selección de campos de la lectura a solicitudes basadas en cuerpo (campo permitido anidado, campo no permitido, nombre no canónico, ruta vacía/duplicada), verificando `400` antes de consultar el CRUD para `fields` inválidos y `206`+`MetaData` cuando hay advertencias de expansión; verificar con `go test ./controllers/...`

## 2. Publicación del contrato

- [x] 2.1 Agregar `POST /v1/proveedor/consulta` a `swagger/swagger.yml` con el esquema del cuerpo (`id` entero positivo, `fields` arreglo de strings) y el envoltorio de respuesta (`200`/`206`/`400`/`405`); verificar que el arranque regenera `swagger/swagger.json` sin error y que el path aparece en el JSON derivado

## 3. Retiro del GET de lectura (grupo final)

- [ ] 3.1 Eliminar el manejador `Leer` y la rama de lectura en `enrutarProveedor` (`main.go`) de modo que `GET`/`PUT`/`DELETE` sobre `/v1/proveedor` respondan `405` con `Allow: POST, PATCH`; verificar con una prueba que `GET /v1/proveedor` devuelve `405`
- [ ] 3.2 Retirar el path `GET /v1/proveedor` de `swagger/swagger.yml`; verificar que el JSON derivado ya no publica la lectura por GET
- [ ] 3.3 Eliminar las pruebas específicas del GET de lectura y ejecutar `go build ./...` y `go test ./...` para confirmar que no quedan referencias al manejador `Leer` ni al GET de lectura

## Workflow follow-up

- Confirmar en ambiente real que WSO2 acepta y reenvía el cuerpo JSON sin transformaciones y validar el límite de longitud de URL vigente (verificación de despliegue, fuera del alcance de la especificación).
- Actualizar por separado el cambio en curso `proveedor-escritura-declarativa` para retirar la especificación de DELETE.
- Ejecutar `openspec validate proveedor-read-post-consulta` antes de sincronizar y archivar el cambio.
