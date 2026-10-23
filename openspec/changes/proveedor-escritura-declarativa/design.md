# Design

## Context

Ver proposal.md — Why. Estado actual relevante (verificado en el repo y el
modelo `agora_modelo_extendido.dbml`):

- La lectura ya es declarativa: `helper/expand.go` recorre el árbol de relaciones,
  `EntityResolver` encapsula el origen real de cada entidad (endpoint + campo de
  consulta) y la configuración vive en `config/relaciones_proveedor.yaml`. La
  escritura se diseña como su espejo.
- Los flujos de escritura actuales son estáticos y cableados a mano
  (`services/registro_persona.go`, `services/consulta_actualizacion.go`,
  `dto/registro_compuesto.go`, `dto/actualizar_compuesto.go`). Este cambio los
  retira por completo (BREAKING) y los reemplaza por el motor declarativo.
- Invariantes de negocio hoy cableadas que deben sobrevivir: crear-o-reutilizar
  idempotente (no duplicar proveedor por tercero), resolución de tipo por código
  del cliente ya no aplica (el cliente envía `Id`), inmutabilidad de
  numero/tipo documental, consistencia natural/jurídico.
- Modelo (DBML): `proveedor.tercero_id` es UNIQUE y es una **FK** a `tercero.id`
  (1-1; el `tercero` tiene su propio `Id` y el proveedor lo referencia, no
  comparten PK); `proveedor_natural.id` y
  `proveedor_juridico.id` comparten PK con `proveedor` (Ref `-`);
  `contacto`/`cuenta_bancaria` cuelgan de `proveedor_id` (FK, 1-N);
  `datos_identificacion` cuelga de `tercero_id`; la unicidad real del tercero
  vive en `datos_identificacion (numero + tipo_documento_id)`. Toda tabla lleva
  campos de auditoría (`activo`, `fecha_*`, `autor_modificacion_id`).
- Frontera de capas (lineamiento OATI y cambio `proveedor-nested-resolvers`):
  `helper/` solo primitivas reutilizables que no nombran dominio; el cableado
  específico del proveedor vive en `services/`.

## Goals / Non-Goals

**Goals:**
- Escritura declarativa espejo de la lectura: `POST`/`PATCH` sobre `/v1/proveedor`
  con cuerpo anidado, dirigida por configuración.
- Portabilidad de esquema: un esquema nuevo = cambios en el YAML (y a lo sumo un
  write-resolver acotado), no reescritura de servicios.
- Reutilizar el árbol de relaciones y el transporte/decodificación existentes.
- Reemplazar totalmente los flujos estáticos de persona (Corte B: incluye
  `tercero` y `datos_identificacion`).

**Non-Goals:**
- No convertir el MID en dueño de los datos ni en un ORM genérico.
- No escribir catálogos/parámetros: el cliente envía su `Id`.
- No acceder directo a Postgres: el MID sigue siendo middleware HTTP.
- No incluir entidades de fase 2 (perfil_financiero, actividad_economica, etc.).

## Decisions

- **Postura 2 — motor declarativo estructural + write-resolvers para lo
  irreductible**: un `WriteEngine` genérico (en `helper/`) recorre el árbol en
  orden topológico y delega cada entidad a un write-resolver fino (en `services/`),
  espejo de `EntityResolver`. Lo estructural (árbol, propagación de llave, aplanado
  de parámetros, upsert por clave) es configuración; las reglas de dominio
  irreductibles (consistencia natural/jurídico, inmutables, validación de código de
  tipo) viven en el servicio/write-resolver, no en el motor. Alternativa
  descartada: motor 100% genérico con un DSL que modele toda regla — frágil y
  tiende a reinventar un ORM. Alternativa descartada: servicios por entidad como
  hoy — no avanza la portabilidad.
- **Un solo YAML por servicio con bloque `write` namespaced por relación**: lectura
  y escritura comparten el árbol; el bloque `write` declara campos escribibles,
  inmutables, clave natural (upsert), modo de propagación de llave y políticas por
  verbo. El loader valida coherencia lectura/escritura al arranque. Alternativa
  descartada: archivo de escritura separado — duplica el árbol y arriesga
  desincronía, en contra de la premisa de portabilidad.
- **Mapeo transparente `MÉTODO /v1/<tabla>` con anidación**: `proveedor` es solo la
  primera tabla configurada; el patrón se repite para futuras raíces
  (perfil_financiero, etc.) añadiendo configuración. Los casos que requieran
  control explícito tendrán su propio controlador/servicio sin romper el motor.
- **Entrada anidada de parámetros aplanada a FK; `Id` provisto por el cliente**:
  `{Tercero:{TipoContribuyente:{Id:2}}}` se escribe como `TipoContribuyenteId: 2`.
  La lectura hace el camino inverso (FK -> objeto anidado). El MID nunca resuelve
  catálogos en escritura.
- **Propagación de llave declarada: tres modos**: el modo se declara por relación
  y no se infiere del modelo. (1) **FK descendente** (`fk`): el padre se
  materializa primero y su `Id` fija la FK del hijo (`contacto`/`cuenta_bancaria`
  por `proveedor_id = proveedor.Id`). (2) **PK compartida** (`shared_pk`): el hijo
  usa como `Id` el del padre (`proveedor_natural`/`proveedor_juridico`,
  `Id = proveedor.Id`). (3) **FK ascendente** (`fk_inbound`): el `tercero` es dueño
  de su propio `Id` autogenerado y el proveedor raíz lo referencia por la FK
  `TerceroId`; por tanto el `tercero` se materializa **antes** que la raíz y su
  `Id` fluye hacia arriba a `proveedor.TerceroId`. Esto obliga a un orden
  topológico en el que las relaciones `fk_inbound` del nodo se resuelven antes de
  materializar el nodo que las referencia.
- **Idempotencia por clave natural donde el esquema la garantiza; `Id` en cuerpo
  para el resto**: `proveedor` por `tercero_id` (unique); `datos_identificacion`
  por `numero + tipo_documento_id`; colecciones sin clave natural usan `Id` del
  cuerpo (sin `Id` crea, con `Id` actualiza). Esto hace `PATCH` idempotente y es la
  ruta de continuación tras creación parcial.
- **Semántica de fallo espejo de la lectura**: relaciones bloqueantes vs. no
  bloqueantes, `on_relation_error` tolerante/estricto, creación parcial que
  conserva lo materializado y reporta `MetaData.Errors` con `Status: "206"`,
  devolviendo los `Id` materializados. Alternativa descartada: best-effort
  silencioso — deja huérfanos sin señal de reintento.
- **Auditoría transversal inyectada**: `autor_modificacion_id`/usuario se pueblan
  desde el contexto de autenticación en cada escritura, no desde el cuerpo.
- **Frontera helper/services**: el `WriteEngine` genérico vive en `helper/`; el
  cableado del árbol escribible del proveedor (write-resolvers, claves naturales,
  propagación concreta) vive en `services/`. Invariante: `helper/` no nombra
  entidades de dominio ni rutas `/v1/x`.

## Risks / Trade-offs

- [Cascada cruza dos CRUD sin transacción distribuida] → orden topológico que
  minimiza huérfanos + idempotencia por clave natural; `tercero` se crea pero no
  se elimina, por lo que un fallo posterior deja un tercero reutilizable en el
  reintento, no limpiable automáticamente. Se acepta y se reporta.
- [Clave natural de `tercero` no es columna unique] → se identifica por
  `datos_identificacion (numero + tipo_documento_id)`, igual que el flujo estático
  actual; a fijar en la implementación del write-resolver de tercero.
- [Ambigüedad del modelo: `proveedor.tercero_id [unique]` (FK) vs `Ref tercero.id -
  proveedor.id` (PK compartida)] → **resuelto: es una FK**
  (`proveedor.tercero_id → tercero.id`), no una PK compartida (el `Ref ... -` del
  DBML está mal y debe ser `tercero.id < proveedor.tercero_id`). Se modela con
  propagación `fk_inbound`: el `tercero` se crea primero y su `Id` se propaga a
  `proveedor.TerceroId`. El DBML se reporta al equipo del esquema para corregir el
  `Ref`.
- [Colecciones sin clave natural duplican en reintento ciego] → se exige `Id` en el
  cuerpo para actualizar; sin `Id` siempre crea. Riesgo acotado y documentado.
- [Fuga de datos en avisos] → los write-resolvers no construyen mensajes con URLs
  ni credenciales; solo ruta lógica, igual que la lectura.

## Open Questions

- Clave natural definitiva de `tercero` y comportamiento cuando el documento ya
  existe asociado a otro tercero (¿conflicto 409 o reutilización?).
- Endpoints y campos de escritura exactos de cada CRUD para fijar el bloque `write`
  por relación (se confirman al implementar cada write-resolver).
- Representación del autor de modificación en el contexto (fuente del
  `autor_modificacion_id`).
