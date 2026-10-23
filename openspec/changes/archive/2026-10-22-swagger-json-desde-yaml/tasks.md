# Tasks

## 1. Generador determinístico YAML→JSON

- [x] 1.1 Implementar la función que lee `swagger/swagger.yml`, lo decodifica con `yaml.v3` y lo reserializa a JSON indentado con un salto final; verificar con una prueba de que el JSON generado es válido y su modelo de datos es equivalente al del YAML
- [x] 1.2 Garantizar salida determinística (indentación fija); verificar con una prueba de que dos generaciones sobre el mismo YAML producen bytes idénticos
- [x] 1.3 Escribir el JSON solo cuando su contenido difiera del archivo publicado; verificar con una prueba de que una segunda generación sin cambios no reescribe el archivo

## 2. Fallo explícito ante contrato fuente ausente o inválido

- [x] 2.1 Devolver un error identificable cuando el YAML fuente no existe o no parsea; verificar con pruebas de YAML inexistente y de YAML inválido que afirmen el error
- [x] 2.2 Abortar el arranque del API (`log.Fatal`) si la generación falla, antes de registrar la ruta estática; verificar que el flujo de `main.go` invoca el generador y detiene el arranque ante error

## 3. Publicación coherente del contrato

- [x] 3.1 Invocar el generador al arranque en `main.go` y publicar el directorio `swagger/` en la ruta estática `/swagger`; verificar que `GET /swagger/swagger.yml` y `GET /swagger/swagger.json` responden el mismo contrato con el servicio arriba
- [x] 3.2 Confirmar que Swagger UI carga sin faltar el recurso por defecto; verificar accediendo a `/swagger/` y comprobando que la interfaz obtiene `swagger.json`

## 4. Verificación integral

- [x] 4.1 Ejecutar `go build ./...` y `go test ./...` en verde, cubriendo el paquete que contiene el generador y su prueba
