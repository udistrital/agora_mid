# Proposal

## Why

El contrato OpenAPI del API se mantiene hoy como un `swagger.yml` autorado a mano,
del que se deriva el JSON por reserialización. A medida que la configuración se
vuelve la fuente de verdad del comportamiento —objetos direccionables con tabla de
campos tipada, árbol efectivo por servicio y el contrato de escritura/respuesta de
los cambios previos— mantener el YAML a mano lo deja a la deriva respecto del
comportamiento real. Este cambio invierte la fuente de verdad: la configuración
genera el `swagger.yml` de forma determinística, y se retira la autoría asistida por
IA como fuente del contrato. Además, actualiza la narrativa legible de la suite de
pruebas al árbol del modelo #58.

## What Changes

- **BREAKING (fuente de verdad)** La configuración pasa a ser la fuente del contrato
  OpenAPI. `swagger.yml` deja de autorarse a mano y se convierte en un artefacto
  **generado** (una caché versionada, igual que `swagger.json` hoy).
- **Cadena determinística de tres etapas, regenerada en cada arranque.** La
  configuración genera `swagger.yml`; el `swagger.yml` deriva `swagger.json` por la
  lógica de reserialización 1:1 existente, sin cambios. Ambas etapas regeneran al
  arrancar y escriben solo si el contenido difiere (idempotentes en disco).
- **Determinismo de la etapa config -> YAML.** El generador produce una salida
  byte-estable (orden estable de claves y colecciones) para que un arranque sin
  cambios de configuración no reescriba `swagger.yml`.
- **Mapeo de tipo de almacenamiento -> tipo/formato OpenAPI.** El generador fija el
  mapeo del vocabulario cerrado de tipos de almacenamiento (`boolean`, `integer`,
  `jsonb`, `numeric`, `string`, `timestamp`, `uuid`) a su tipo y formato OpenAPI.
  Este mapeo lo dejó explícitamente abierto el cambio que introdujo la tabla tipada.
- **Fallo de fuente desplazado a la configuración.** La fuente que aborta el
  arranque pasa a ser la configuración: una configuración válida garantiza que la
  generación de `swagger.yml` no falle; el aborto por configuración inválida ya lo
  provee el cambio de composición de objetos tipados.
- **Narrativa de la suite de pruebas actualizada al modelo #58.** `test/tests.md`
  describe los flujos vigentes del subárbol #58 (registro natural/jurídico,
  reconciliación, continuación tras `206`, normalización, conflicto). Las pruebas
  ejecutables no viven en este cambio; viven en los cambios que prueban su
  comportamiento.

## Capabilities

### New Capabilities
<!-- Ninguna. -->

### Modified Capabilities
- `openapi-publication`: La fuente de verdad del contrato pasa del `swagger.yml`
  autorado a la configuración; `swagger.yml` se genera desde la configuración de
  forma determinística en cada arranque; se fija el mapeo tipo de almacenamiento ->
  tipo/formato OpenAPI; el fallo explícito de fuente se desplaza a la configuración.
  La derivación `swagger.yml` -> `swagger.json` y la publicación en ambos formatos no
  cambian.
- `test-documentation`: La narrativa de `test/tests.md` se actualiza al árbol del
  modelo #58 y a los flujos de respuesta de los cambios previos (incluido el `206`
  de stop-abort y la reconciliación), sin describir pruebas inexistentes.

## Impact

- **Artefactos**: `swagger/swagger.yml` deja de ser fuente y pasa a ser salida
  generada (caché versionada); `swagger/swagger.json` sigue derivándose de él sin
  cambios; `test/tests.md` se reescribe a la suite #58.
- **Código** (fase apply): se añade un generador `config -> swagger.yml`
  determinístico que corre antes de la derivación existente `swagger.yml ->
  swagger.json`; ambos regeneran en cada arranque escribiendo solo si difieren.
- **Comportamiento observable**: el contrato publicado en `/swagger` (YAML y JSON) y
  en `API.md` refleja la configuración vigente, incluidos los tipos de campo
  mapeados a OpenAPI y los endpoints de escritura con sus códigos de respuesta.
- **Dependencia de planeación**: se apila sobre `config-composicion-objetos-tipados`
  (tabla de campos tipada y vocabulario de tipos) y sobre
  `absorber-modelo-58-proveedor`, `control-escritura-por-relacion` y
  `contrato-respuesta-fallo-parcial` (comportamiento a documentar). Orden de apply
  `0 -> A -> B -> C -> D`.
