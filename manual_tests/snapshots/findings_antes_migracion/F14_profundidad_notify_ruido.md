# F14: el middleware reporta "profundidad reducida" para casi todas las relaciones

Estado: abierto. Es ruido en `MetaData.Errors`, no un error de datos.

## Explicación
`POST /v1/proveedor/consulta {"id":1}` responde 206 con 17 avisos `kind: depth` ("profundidad reducida a N por configuración de la relación") y 3 `kind: relation` (F15). Los avisos `depth` aparecen en relaciones que sí se resolvieron. El 206 hace que la respuesta parezca degradada.

## Propuesta
Decidir con el dueño de la especificación (`nested-object-expansion`): no emitir `depth` cuando la relación se resuelve completa, o dejar el aviso y documentar que el 206 no implica pérdida de datos.
