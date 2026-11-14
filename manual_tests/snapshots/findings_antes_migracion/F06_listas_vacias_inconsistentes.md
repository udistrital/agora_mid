# F06: el CRUD responde distinto a una lista vacía

Estado: abierto, sin evidencia vigente. La observación original (13/11) no se conservó y no se repitió en la corrida 4.

## Explicación
En la observación original, con las tablas vacías, `proveedor_natural` devolvió 200 con `Data: []` y `proveedor` y `contacto` devolvieron 404. El contrato de `collection-read` pide `[]` para una lista vacía.

## Propuesta
Revalidar con el mismo GET sobre tablas vacías. Si se confirma, hacer que `GetAll` del CRUD responda 200 con `Data: []` cuando no hay filas (ver F07).
