# F06: el CRUD responde distinto a una lista vacía

Estado: cerrado_sin_solucion. Sin evidencia vigente. La observación original (13/11) no se conservó y no se repitió en la corrida 4.
Solución: propuesta. Ver `## Propuesta`.
Gravedad: media. Clasificación propuesta, ver utils/README.md.

## Explicación
En la observación original, con las tablas vacías, `proveedor_natural` devolvió 200 con `Data: []` y `proveedor` y `contacto` devolvieron 404. El contrato de `collection-read` pide `[]` para una lista vacía.

## Propuesta
Revalidar con el mismo GET sobre tablas vacías. Si se confirma, hacer que `GetAll` del CRUD responda 200 con `Data: []` cuando no hay filas (ver F07).

## Reverificación (14/11, 15:5x)

Con filtros que no devuelven filas, `GET /v1/contacto`, `GET /v1/proveedor_natural` y `GET /v1/proveedor` responden 200 con `Data: []`. El 404 de este hallazgo no aparece con esos filtros. Solo se reproduciría con tablas vacías, que no se vaciaron en esta revisión.

## Verificación con tablas vacías (17/11/2026, 20:3x)

Procedimiento: respaldo `snapshots/proveedores_antes_f06_17nov.sql`, `TRUNCATE` de las cinco tablas de `proveedores` en `agora_db`, GET de solo lectura en el CRUD, y restauración del respaldo. Los conteos volvieron a la línea base (proveedor 2, natural 1, jurídico 1, contacto 1, cuenta 1).

Resultados con todas las tablas vacías:
- `GET /v1/proveedor/?limit=5`, `/v1/proveedor/?fields=Id&limit=5`, `/v1/proveedor_natural/?limit=5`, `/v1/contacto/?limit=5`, `/v1/cuenta_bancaria/?limit=5` y `/v1/proveedor_juridico/?limit=5`: todos responden 200 con `Data: []`.
- `GET /v1/proveedor/1` y `/v1/proveedor_juridico/2` (registro inexistente): 404 con "Error service GetOne".
- `POST /v1/proveedor/consulta {"id":1}` en el MID: 404 "No existe un proveedor con el id solicitado".

Conclusión: la inconsistencia de listas vacías descrita en `## Explicación` no se reproduce. Las listas responden 200 con `[]`, y solo los registros individuales dan 404. Queda pendiente decidir el estado: la explicación original ya no aplica.

## Decisión
Cerrado sin solución por el usuario (17/11/2026, 20:4x). La inconsistencia no se reproduce: con tablas vacías, todas las listas responden 200 con `Data: []` (ver `## Verificación con tablas vacías`). No hay cambio en este repositorio. No se marca como resuelto, porque no hay cambio aplicado.
