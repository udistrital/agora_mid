# Design

## Context

Ver proposal.md — Why. Estado actual relevante (verificado en el repo):

- El helper de expansión (`helper/expand.go`, `seleccion.go`), el resolver
  genérico (`helper/resolver.go` con `HTTPResolver`, `generic_resolver.go`), la
  carga/validación de relaciones (`config/relaciones.go`, `relaciones_loader.go`)
  y el contrato de respuesta (`responses/response.go`) ya existen y están en
  verde. Son la fundación sincronizada desde el cambio archivado
  `nested-object-expansion`.
- `HTTPResolver.Resolve` ya resuelve la URL base por `related_api`, codifica las
  llaves en `query` con `query_template` y decodifica envelope OAS o lista
  directa. No transforma datos.
- `config.LoadRelaciones` carga `config/relaciones_*.yaml`, valida y aborta el
  arranque ante configuración inválida.
- El controlador actual (`controllers/registro_persona.go`) sirve POST/PUT/GET en
  un path vía switch por método, y la integración de lectura con `fields` nunca
  se implementó. La composición manual previa (`componerProveedor`, DTOs
  compuestos de 7387a47) queda obsoleta.
- Modelo (DBML): `proveedor.tercero_id` es UNIQUE (1-1 proveedor↔tercero);
  `proveedor_natural.id`/`proveedor_juridico.id` comparten PK con `proveedor`;
  `contacto`/`cuenta_bancaria` cuelgan de `proveedor_id` (1-N);
  `datos_identificacion` cuelga de `tercero_id`.

## Goals / Non-Goals

**Goals:**
- Un único endpoint raíz `/v1/proveedor` que expande el núcleo vía `fields` y
  `config/relaciones_proveedor.yaml`.
- Separación estricta de capas: servicio = negocio/orquestación; resolver =
  acceso a datos sin negocio; helper = recorrido y dispatch.
- Reutilizar la fundación existente sin reescribir transporte ni decodificación.

**Non-Goals:**
- No se expone endpoint raíz por `tercero_id` ni por otras entidades en fase 1.
- No se expanden ubicaciones (ciudades quedan como ID plano).
- No se crea un proxy genérico; la raíz la trae el servicio, no el cliente.
- No se accede directo a Postgres: el MID sigue siendo middleware HTTP.

## Decisions

- **Dispatch relación→resolver por convención + inyección (sobre declaración en
  YAML)**: el helper gana un registro `map[nombreRelación]Resolver` que el
  `ProveedorService` inyecta; si una relación no tiene entrada, cae al genérico.
  Alternativa descartada: nombrar el resolver en el YAML — acopla la
  configuración a nombres de structs Go y contamina el contrato declarativo.
- **Resolvers específicos como envoltorios finos de `HTTPResolver`**: cada uno
  fija endpoint y campo de consulta real (por `id` propio, `tercero_id`,
  `proveedor_id` o PK compartida) y reutiliza el transporte/decodificación.
  Alternativa descartada: implementar `Resolve` desde cero por entidad —
  duplicaría transporte, timeouts y manejo de envelope.
- **Frontera `helper/` ↔ `services/` (solo primitivas reutilizables en helper)**:
  `helper/` contiene exclusivamente componentes transversales reutilizables que
  NO nombran entidades de dominio (`EntityResolver`, `NewEntityResolver`, el
  expansor y la selección). El cableado específico del proveedor
  (`NewResolversNucleo` y las constantes `ServicioX`/`endpointX`, que fijan qué
  entidades del núcleo existen y su endpoint/campo de consulta) vive en
  `services/`, no en `helper/`. Alinea con el lineamiento OATI (helpers =
  reutilizable entre servicios; lo específico de un servicio en services) y con
  el Goal de separación estricta de capas. Alternativa descartada: dejar el
  cableado del núcleo en `helper/` — acopla una primitiva transversal a un
  servicio concreto.
- **`ProveedorService` como única capa de negocio**: trae el proveedor base por
  `id` de `gestion_personas_crud`, arma `ExpandOptions(fields)`, registra el
  mapeo e invoca el helper. La decisión de rama natural/jurídico (por
  `tipo_registro_id` → `tipo_contribuyente`) y el marcado/conservación del objeto
  viven aquí, no en los resolvers.
- **Un solo controlador `/v1/proveedor`**: lee `id` y `fields` de la query,
  delega al servicio y serializa con el contrato OAS plano + `MetaData.Errors`.
- **Eliminación de la composición manual de 7387a47**: `componerProveedor`, el
  `MarshalJSON` a medida de `PersonaIntegradaResultado` y los DTOs compuestos se
  retiran; el shape de salida lo produce el helper.
- **Validación de `fields` antes de tocar el CRUD (fallar temprano) con un solo
  parseo**: la validación de `fields` contra la configuración declarativa ocurre
  en la entrada de la lectura, antes de traer el proveedor base; un `fields`
  inválido responde `400` sin ninguna llamada al CRUD dueño, igual que el `id`
  inválido. El parseo de `fields` se consolida en una sola fuente de verdad (un
  único `ParsearFields` con el mismo conjunto `allowed`) en lugar de parsear una
  vez en el servicio con `allowed=nil` sobre la config completa y otra vez en
  `Expand` con `allowedTopLevel` sobre la config podada. Alternativa descartada:
  validar dentro de `Expand` tras la lectura — consume una llamada de red
  evitable y deja dos nociones de validez divergentes para el mismo `fields`.

## Risks / Trade-offs

- [Eliminar la composición manual rompe el contrato previo y sus tests] →
  retirar en conjunto código + tests asociados dentro de este cambio; documentar
  el nuevo contrato.
- [Entidades con llave de consulta no trivial (datos_identificacion por
  tercero_id, natural/juridico por PK compartida)] → encapsular el campo de
  consulta en el resolver específico de cada entidad y cubrir con pruebas.
- [N+1 al expandir listas (contacto, cuenta_bancaria)] → reutilizar la
  deduplicación, agrupación y caché por solicitud ya presentes en el helper.
- [Latencia/fallos por CRUD adicionales] → timeouts y `context.Context` del
  `HTTPResolver`; política `on_relation_error` en la config del proveedor.
- [Fuga de datos en avisos] → los resolvers no construyen mensajes con URLs ni
  credenciales; se mantiene la regla de solo ruta lógica.
- [El cableado del núcleo quedó en `helper/` (`NewResolversNucleo`) violando la
  frontera de capas] → reubicarlo en `services/` (tarea 8.1); invariante
  verificable: `helper/` no referencia nombres de tablas/entidades de dominio ni
  rutas `/v1/x` (comprobable por grep en revisión/CI).

## Migration Plan

1. Añadir el dispatch relación→resolver con fallback al genérico en el helper,
   sin romper el uso actual del genérico.
2. Implementar los resolvers específicos del núcleo y el `ProveedorService`.
3. Crear `config/relaciones_proveedor.yaml` y registrar la ruta `/v1/proveedor`.
4. Eliminar la composición manual (`componerProveedor`, DTOs compuestos) y sus
   tests; migrar lo que aún sea útil.
5. Rollback: desregistrar la ruta `/v1/proveedor`; la fundación (helper,
   resolver genérico, config) permanece intacta.

## Open Questions

- Valores concretos de `max_depth`, `max_related_ids` y `on_relation_error` para
  `relaciones_proveedor.yaml` (ajustables sin cambiar specs ni tareas).
- Endpoints y nombres de recurso exactos de cada CRUD para fijar `endpoint` y
  `query_template` por relación (se confirman al implementar cada resolver).
