# Proposal

## Why

Hoy la escritura del proveedor vive en flujos estáticos cableados a mano
(`RegistrarPersonaNatural/Juridica`, `ActualizarPersona` y los DTOs compuestos),
que mapean campo por campo contra cada CRUD y hay que reescribir cada vez que el
esquema cambia. El esquema de proveedores está en movimiento (y es posible que
absorba el de terceros), así que necesitamos que la escritura sea tan
declarativa y config-driven como ya lo es la lectura: cambiar un archivo de
configuración, no un servicio, cuando el modelo evolucione.

## What Changes

- Nuevos endpoints de escritura sobre la raíz `proveedor`, con mapeo transparente
  `MÉTODO /v1/<tabla>` y cuerpo anidado:
  - `POST /v1/proveedor`: crea el subárbol del proveedor.
  - `PATCH /v1/proveedor`: reconcilia (upsert) el subárbol; crea lo ausente,
    actualiza lo presente, nunca borra. Es idempotente y sirve como ruta de
    continuación tras una creación parcial.
  - `DELETE /v1/proveedor`: se incorpora **al final** del plan; su estrategia
    (previsiblemente baja lógica vía `activo=false`) queda **diferida** hasta
    validación con el equipo. No se implementa junto con POST/PATCH.
- Nuevo motor de escritura declarativo, espejo del motor de expansión de lectura:
  recorre el árbol de relaciones en orden topológico (padres antes que hijos),
  propaga la llave del padre al hijo (FK o PK compartida), aplana la entrada
  anidada de parámetros (`{X:{Id:n}}` -> FK plana `XId: n`) y delega cada entidad
  a un *write-resolver* fino. La lógica de negocio irreductible (idempotencia por
  clave natural, consistencia de tipo natural/jurídico, inmutables) vive en el
  servicio/write-resolver, no en el motor genérico.
- El archivo declarativo por servicio gana un bloque `write` namespaced por
  relación (campos escribibles, inmutables, clave natural para upsert, modo de
  propagación de llave, políticas por verbo). El loader valida **coherencia entre
  los bloques de lectura y escritura**: toda relación del bloque `write` existe en
  el árbol y es consistente con su contraparte de lectura.
- Semántica de fallo espejo de la lectura: relaciones bloqueantes vs. no
  bloqueantes, creación parcial tolerada y avisos en `MetaData.Errors`, con los
  identificadores ya materializados devueltos para permitir el reintento
  idempotente. El cliente envía siempre el `Id` de parámetros/catálogos; el MID
  nunca crea ni modifica catálogos.
- Alcance fase 1 (Corte B): subárbol escribible completo que cruza ambos CRUD
  dueños — `tercero` y `datos_identificacion` (Terceros CRUD; se crean y modifican,
  **no** se eliminan), `proveedor`, `proveedor_natural`/`proveedor_juridico`,
  `contacto[]` y `cuenta_bancaria[]` (gestión de personas CRUD). Catálogos
  (`tipo_contribuyente`, `perfil`) solo se referencian por `Id`.
- **BREAKING**: se retiran por completo los flujos estáticos de persona y sus
  rutas: `RegistrarPersonaNatural/Juridica`, `ActualizarPersona` y sus ayudantes
  en `services/`, los DTOs compuestos (`dto/registro_compuesto.go`,
  `dto/actualizar_compuesto.go`) y las rutas `POST/PUT/GET /v1/persona_*`. La
  escritura declarativa sobre `/v1/proveedor` las reemplaza en su totalidad. No
  hay cliente productivo que dependa del contrato anterior.
- Se añaden pruebas de integración locales (router + CRUD simulados) y al menos
  una prueba e2e de un flujo completo de creación/actualización del subárbol.
- Se actualiza la documentación pública del contrato para describir los endpoints
  de escritura: `swagger/swagger.yml` (y su `swagger/swagger.json` derivado) y
  `API.md` ganan `POST`/`PATCH /v1/proveedor` (cuerpo anidado, respuestas `200`,
  `206` con `MetaData.Errors`, `400`, `502/504`) y retiran el contrato de los
  flujos `/v1/persona_*` eliminados. `README.md` no se modifica.

## Capabilities

### New Capabilities
- `proveedor-write`: escritura del proveedor como raíz mediante `POST`/`PATCH`
  (y `DELETE` diferido) con cuerpo anidado, que crea y reconcilia en cascada el
  subárbol del núcleo a través de los CRUD dueños, sin que el MID sea dueño de los
  datos, reportando la creación parcial y los identificadores materializados.
- `declarative-write-engine`: motor de escritura declarativo reutilizable, espejo
  del de expansión, que recorre el árbol de relaciones en orden topológico,
  propaga la llave del padre (FK o PK compartida), aplana la entrada anidada de
  parámetros a FK y delega cada entidad a un write-resolver, aplicando las
  políticas de escritura declaradas por configuración.

### Modified Capabilities
- `api-response-contract`: la respuesta de escritura deja de ser siempre "sin
  `MetaData`"; una operación de escritura con creación o actualización parcial
  tolerada responde `Status: "206"` e incluye `MetaData.Errors` con los avisos,
  igual que la lectura. Una escritura completamente exitosa sigue usando el
  envoltorio plano sin `MetaData`.
- `openapi-publication`: el contrato publicado (YAML fuente + `API.md`) describe
  los endpoints de escritura `POST`/`PATCH /v1/proveedor` y permanece coherente
  con el comportamiento implementado.

## Impact

- **Código nuevo**: controlador(es) de escritura de `/v1/proveedor`; motor de
  escritura en `helper/` (genérico); write-resolvers y cableado del árbol
  escribible del proveedor en `services/`; bloque `write` y su validación en el
  loader de configuración.
- **Código eliminado (BREAKING)**: flujos estáticos de registro/actualización de
  persona en `services/`, DTOs compuestos en `dto/`, y las rutas `/v1/persona_*`.
- **Config**: el YAML de relaciones del proveedor gana el bloque `write`
  namespaced; el loader valida coherencia lectura/escritura.
- **Endpoints afectados**: alta de `POST`/`PATCH` (y `DELETE` diferido)
  `/v1/proveedor`; baja de `POST/PUT/GET /v1/persona_*`.
- **CRUD dueños**: la cascada cruza Terceros CRUD y gestión de personas CRUD en
  una sola operación; sin transacción distribuida, la recuperación se apoya en
  idempotencia por clave natural y en los identificadores devueltos.
- **Pruebas**: unitarias del motor, write-resolvers, servicio, loader y
  controlador; integración local (router + CRUD simulados) y e2e de flujo
  completo.
- **Documentación pública**: `swagger/swagger.yml` + `API.md` describen el
  contrato de escritura y reflejan la baja de `/v1/persona_*`; `README.md` queda
  intacto.
- **Alcance**: Corte B (incluye `tercero`/`datos_identificacion`). `DELETE` y su
  estrategia de baja quedan diferidos. Fase 2 (ya visible en el modelo):
  `perfil_financiero`, `actividad_economica_proveedor`, `representacion`,
  `informacion_financiera`, `documento`, `declaracion`, con sus propios
  controladores/servicios explícitos.

## Riesgos y preguntas abiertas (para el diseño)

- Clave natural de `tercero` para el upsert: el modelo no declara `unique` en
  `tercero`; la unicidad real vive en `datos_identificacion`
  (`numero` + `tipo_documento_id`). El design debe fijar cómo "identificar el
  tercero para continuar".
- Ambigüedad del modelo entre `proveedor.tercero_id [unique]` (FK) y el `Ref`
  `tercero.id - proveedor.id` (PK compartida). El bloque `write` declara la
  propagación de llave de forma explícita en lugar de inferirla; se marca para
  validación con el equipo.
- Campo transversal de auditoría (`autor_modificacion_id`/usuario) que toda
  escritura debe poblar desde el contexto de autenticación.
