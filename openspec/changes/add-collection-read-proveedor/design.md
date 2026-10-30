# Design

## Context

Ver `proposal.md` (sección Why) para la motivación. Estado actual relevante:

- La lectura puntual del proveedor (`proveedor-read`) vive en
  `services/proveedor.go` (`ProveedorService.LeerProveedor`): resuelve el proveedor
  base por id contra el CRUD de gestión de personas, valida y parsea `fields` con
  `helper.ParsearFields`, poda la rama de detalle según el tipo de registro e invoca
  `helper.Expand` con el registro raíz como mapa.
- La expansión de un único registro ya existe en `helper/expand.go`
  (`helper.Expand`, `helper.ExpandResult`, `helper.ExpandOptions`) y en los resolvers
  del núcleo (`services/resolvers_nucleo.go`), gobernada por la configuración
  declarativa `config/relaciones_proveedor.yaml` (`config.ServiceConfig`).
- El controlador `controllers/proveedor.go` ya expone `POST /v1/proveedor/consulta`
  (lectura puntual) y `POST`/`PATCH /v1/proveedor` (escritura), y mapea errores del
  servicio al envoltorio de respuesta estándar (`responses`).
- `proveedorBase` ya traslada `query`, `limit` y `offset` al CRUD dueño para traer un
  único registro, de modo que el patrón de passthrough de paginación al CRUD ya está
  probado en el repositorio.
- El CRUD dueño acepta `query=col:val,col2:val2`, `limit` y `offset`, y entrega el
  conteo total del conjunto en el campo `Total` de su envoltorio OAS plano
  (`Data`/`Message`/`Status`/`Total`/`Success`), hermano de `Data`.
- El decodificador de listas actual (`services/api_response.go`, `decodeAPIList`)
  extrae solo `Data` del envoltorio y descarta el resto; capturar `Total` es un punto
  concreto de implementación.

## Goals / Non-Goals

**Goals:**
- Un motor de listado genérico en `helper/` que reutilice `helper.Expand` por
  elemento, sin acoplarse al esquema del proveedor.
- Que la lógica de negocio (recurso raíz, campos filtrables, poda de rama) viva en el
  servicio, igual que en la lectura puntual; el motor y los resolvers no deciden
  esquema.
- Consumir el `Total` del envoltorio del CRUD y exponerlo en `MetaData.Total` de la
  respuesta de listado del MID.

**Non-Goals:**
- No se implementan filtros de segundo nivel (relaciones de otros CRUD), que exigirían
  sub-consultas, intersección y un total costoso. Un filtro de segundo nivel se
  rechaza con `400`.
- No se aborda el esquema de cotizaciones ni la sugerencia/confirmación de invitados.

## Decisions

### Decisión 1: Motor de listado genérico que reutiliza la expansión por elemento
Se añade a `helper/` un motor de listado (por ejemplo `ListCollection`) que recibe la
lista de registros raíz ya traída del CRUD y, por cada uno, invoca `helper.Expand` con
las mismas opciones de selección y profundidad que la lectura puntual, acumulando los
avisos de expansión de todos los elementos.

- **Por qué:** la expansión de un registro ya está probada y es agnóstica al esquema;
  el listado es “expandir N veces”. Reutilizarla evita duplicar la lógica de selección,
  recorte, lista negra y política de error de relación.
- **Alternativa descartada:** un expansor de lista independiente que reimplemente la
  selección. Se descarta por duplicación y riesgo de divergencia de semántica con la
  lectura puntual.

### Decisión 2: El servicio trae la página del CRUD; el motor solo expande
El servicio de listado (en `services/`) construye el `query` de primer nivel a partir
de los filtros validados contra los campos filtrables declarados, traslada `limit`/
`offset` al CRUD dueño, decodifica la lista de registros raíz y la pasa al motor de
listado. La validación de `fields` y de filtros ocurre en el servicio ANTES de
consultar el CRUD, igual que en la lectura puntual.

- **Por qué:** mantiene la frontera ya establecida (negocio en el servicio, mecanismo
  en `helper/`). El passthrough de paginación reutiliza el patrón de `proveedorBase`.
- **Trade-off:** el servicio conoce qué campos son filtrables. Se declara en la
  configuración del servicio para no cablear el esquema en código.

### Decisión 3: Campos filtrables declarados en configuración
Los campos de primer nivel admitidos como filtro se declaran de forma explícita en la
configuración del servicio (por ejemplo una lista `filterable_fields` del recurso
raíz), y el servicio rechaza con `400` cualquier filtro fuera de esa lista, incluido
cualquier filtro con forma de ruta de relación (segundo nivel).

- **Por qué:** hace el filtrado agnóstico al esquema y auditable; otro servicio
  (cotizaciones) declarará su propio conjunto sin tocar el motor.
- **Alternativa descartada:** permitir cualquier columna de la raíz. Se descarta porque
  expondría columnas internas y acoplaría el MID al esquema del CRUD.

### Decisión 4: Gramática del `query` y validación estricta de operadores beego
El cuerpo del listado es un objeto JSON con `query` (lista de cadenas de condición),
`fields` (lista de rutas), `limit` y `offset`. Cada condición de `query` tiene la forma
`Campo[__operador]:valor`: un único segmento de campo de primer nivel, un sufijo de
operador opcional (ausente ⇒ `exact`) y un valor; para `__in` el valor es una lista
separada por `|` (por ejemplo `Id__in:15|16|17`). El servicio valida cada condición
ANTES de consultar el CRUD: el campo debe estar en los campos filtrables declarados, el
operador debe pertenecer al conjunto soportado por el ORM del CRUD dueño (beego:
`exact`, `iexact`, `contains`, `icontains`, `gt`, `gte`, `lt`, `lte`, `startswith`,
`istartswith`, `endswith`, `iendswith`, `in`, `isnull`) y la forma del valor debe ser
coherente con el operador (validación tipo (b)): `in` exige lista no vacía separada por
`|`, `isnull` exige booleano y los operadores escalares exigen un único valor. Una
condición malformada se rechaza con `400` sin consultar el CRUD. Las condiciones
válidas se trasladan al `query` del CRUD combinadas con AND, conservando el separador
`__` de operador que beego interpreta.

- **Por qué:** el `query` se pasa por passthrough al ORM beego del CRUD; validar la
  gramática de operadores en el MID atrapa el error con un `400` claro en vez de
  propagar un fallo opaco del CRUD. La propia documentación de beego muestra que un
  `__in` con lista vacía hace *panic* en el ORM, de modo que la validación tipo (b)
  previene un fallo en producción.
- **Forma del cuerpo:** análoga a la del `{id, fields}` de la consulta puntual, pero con
  `query`, `fields`, `limit` y `offset`; una condición de ruta de relación
  (`A.B:` o `A__B:`) no es de primer nivel y se rechaza.
- **Alternativa descartada:** filtros como campos sueltos sin operadores. Se descarta
  porque no cubriría los operadores (`in`, rangos) que el CRUD ya soporta.

### Decisión 5: Consumir el `Total` del CRUD y exponerlo en `MetaData.Total`
El servicio captura el campo `Total` del envoltorio que entrega el CRUD dueño (hoy
descartado por `decodeAPIList`) y el motor/servicio lo propaga al bloque `MetaData` de
la respuesta del MID como `MetaData.Total`, junto a los avisos de expansión existentes.

- **Por qué:** `MetaData` ya es el lugar del envoltorio del MID para metadatos de la
  respuesta (hoy `Errors` y `Length`); `Total` encaja ahí sin alterar la forma del
  envoltorio estándar ni replicar un campo hermano de `Data`.
- **Alternativa descartada:** exponer el total como campo hermano de `Data` imitando al
  CRUD. Se descarta porque cambiaría la forma del envoltorio estándar del MID.
- **Nota de implementación:** `decodeAPIList` hoy solo extrae `Data`; capturar `Total`
  exige extender la decodificación del envoltorio del CRUD para ese campo.

## Risks / Trade-offs

- [Expandir N elementos multiplica las consultas a los CRUD relacionados] → La
  deduplicación y el agrupamiento por solicitud ya existentes en la expansión acotan el
  costo; `limit` acota el número de elementos expandidos por página.
- [El `Total` del CRUD podría no coincidir con el número de elementos expandidos por
  página] → Es esperado: `MetaData.Total` describe el conjunto filtrado completo y la
  página a lo sumo `limit` elementos; son magnitudes distintas y así se documenta.
- [Un filtro de segundo nivel enviado por el cliente] → Se rechaza con `400` por no
  estar en los campos filtrables declarados; no se intenta resolver.

## Open Questions

Ninguna que afecte specs, enfoque o desglose de tareas.
