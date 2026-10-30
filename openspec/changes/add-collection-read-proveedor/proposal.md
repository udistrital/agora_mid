# Proposal

## Why

Hoy el MID solo lee un proveedor por identificador (`POST /v1/proveedor/consulta`);
no existe una lectura de colección que liste recursos con filtros y paginación. Se
necesita un mecanismo de listado genérico y agnóstico al esquema —reutilizable por
servicios futuros como cotizaciones sin tocar el motor— y validarlo sobre el único
recurso con esquema real hoy: el proveedor.

## What Changes

- Introducir un motor de lectura de colección declarativo y agnóstico al esquema:
  lista un recurso raíz, aplica filtros de primer nivel (columnas de la tabla raíz)
  por passthrough al `query` del CRUD dueño, delega `limit`/`offset` al CRUD y
  expande el subárbol de cada elemento reutilizando la expansión de objetos anidados
  existente (selección por `fields` y `max_depth`).
- Exponer el listado del proveedor como primera instancia del motor, mediante un
  endpoint de consulta de colección que recibe filtros de primer nivel, `fields`,
  `limit` y `offset` en el cuerpo JSON y devuelve la lista en el envoltorio de
  respuesta estándar.
- Exponer el total del conjunto: el CRUD entrega el conteo total en el campo `Total`
  de su envoltorio OAS plano (hermano de `Data`); el MID lo consume y lo expone en
  `MetaData.Total` de su respuesta de listado, de modo que el consumidor conozca el
  tamaño del conjunto filtrado más allá de la página devuelta.
- Fuera de alcance (deuda explícita, no se aborda aquí): filtros de segundo nivel
  (campos de relaciones en otros CRUD, que exigirían intersección y total costoso);
  el esquema de cotizaciones (consulta de solicitud, listado real con sus filtros de
  vigencia/estado/dependencia/responsable); la sugerencia de proveedores para
  invitados (requiere proyección a `proveedor_id` y deduplicación, no es un listado
  directo sobre la tabla puente); y la confirmación de la lista de invitados.

## Capabilities

### New Capabilities
- `collection-read`: lectura de colección genérica y declarativa de un recurso raíz:
  filtrado de primer nivel por passthrough al CRUD dueño, paginación delegada
  (`limit`/`offset`) con el total del conjunto tomado del CRUD y expuesto en
  `MetaData.Total`, y expansión del subárbol de cada elemento reutilizando la
  expansión de objetos anidados. El recurso y sus campos filtrables se declaran por
  configuración; el motor no conoce ningún esquema.

### Modified Capabilities
- `proveedor-read`: añadir la lectura del proveedor como colección (listado con
  filtros de primer nivel y paginación), además de la lectura puntual por
  identificador ya existente.

## Impact

- Código: nuevo motor de listado en `helper/` que reutiliza `helper/expand.go` y los
  resolvers; nuevo servicio/caso de uso de listado de proveedor en `services/`;
  nuevo endpoint en el controlador `controllers/proveedor.go`; posible parámetro de
  configuración declarativa para los campos filtrables de primer nivel en
  `config/relaciones_proveedor.yaml`.
- APIs del MID: nuevo endpoint de consulta de colección del proveedor. La lectura
  puntual existente no cambia.
- CRUD dueños: se consume el `query`, `limit`, `offset` y el `Total` del envoltorio
  del CRUD de gestión de personas; sin cambios en esos servicios.
- Sin impacto en dependencias ni en la configuración de escritura existente.
