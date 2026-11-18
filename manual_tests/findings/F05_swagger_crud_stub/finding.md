# F05: el swagger del CRUD es un stub de Beego

Estado: cerrado_sin_solucion. No se puede usar como contrato.
Solución: propuesta. Ver `## Propuesta`.
Gravedad: baja. Clasificación propuesta, ver utils/README.md.

## Explicación
`/swagger/swagger.json` del CRUD declara `title: "beego Test API"`, con descripción de plantilla. Tiene 24 rutas y 12 modelos que sí corresponden al servicio, pero el documento no describe el servicio.

## Propuesta
Regenerar la documentación del CRUD (`bee generate docs`) a partir de los comentarios `@Title` y `@router`, y reiniciar el proceso. No se ejecutó: el cambio es del repositorio del CRUD.

## Decisión
Cerrado sin solución por el usuario (17/11/2026, 20:4x). El swagger lo genera el repositorio del CRUD, que no se modifica desde este repositorio.
