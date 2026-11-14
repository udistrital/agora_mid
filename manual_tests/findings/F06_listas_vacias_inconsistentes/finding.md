# F06: el CRUD responde distinto a una lista vacía

Estado: abierto. Sin evidencia vigente. La observación original (13/11) no se conservó y no se repitió en la corrida 4.
Solución: propuesta. Ver `## Propuesta`.
Gravedad: media. Clasificación propuesta, ver utils/README.md.

## Explicación
En la observación original, con las tablas vacías, `proveedor_natural` devolvió 200 con `Data: []` y `proveedor` y `contacto` devolvieron 404. El contrato de `collection-read` pide `[]` para una lista vacía.

## Propuesta
Revalidar con el mismo GET sobre tablas vacías. Si se confirma, hacer que `GetAll` del CRUD responda 200 con `Data: []` cuando no hay filas (ver F07).

## Reverificación (14/11, 15:5x)

Con filtros que no devuelven filas, `GET /v1/contacto`, `GET /v1/proveedor_natural` y `GET /v1/proveedor` responden 200 con `Data: []`. El 404 de este hallazgo no aparece con esos filtros. Solo se reproduciría con tablas vacías, que no se vaciaron en esta revisión.
