# Design

## Context

Ver proposal.md — Why. Estado actual relevante del repo:

- Enrutado Beego mínimo: `main.go` registra `beego.Any(...)` + `adaptarHandler`
  que delega a handlers `net/http` planos. Un mismo path sirve POST/PUT/GET
  mediante un switch por `r.Method` en `controllers/registro_persona.go`.
- Respuestas actuales: `respuestaRegistro` y `respuestaOperacion` con campos en
  minúscula `{success, data, error}`, serializadas con `json.NewEncoder`.
- `clients.HTTPClient` ya respeta `context.Context`, lleva `Authorization` y
  expone `Get`. `services.decodeAPIList[T]` ya decodifica tanto el envelope OAS
  estándar (`Data`) como una lista JSON directa.
- `config.Load()` lee solo tres URLs de `conf/app.conf` de forma tolerante (no
  aborta ante ausencia).
- `utils_oas` (customerror/errorctrl) ya emite la raíz OAS plana
  `{Success, Status, Message, Data}` con `Status` como string.
- `gopkg.in/yaml.v3` ya está en go.sum (indirect).

## Goals / Non-Goals

**Goals:**
- Helper reutilizable e independiente de controladores que orqueste la
  expansión; los controladores solo arman opciones, invocan y serializan.
- Contrato de respuesta Forma 3 uniforme en todo el API.
- Reutilizar las fundaciones existentes (HTTPClient, decodeAPIList, yaml.v3).

**Non-Goals:**
- No se introduce un caché global ni persistencia; el caché es solo por
  solicitud.
- No se simula consistencia transaccional entre CRUD.
- No se crea un endpoint proxy genérico `/v1/recurso`; la integración ocurre en
  los endpoints de lectura existentes.

## Decisions

- **Contrato Forma 3 (raíz OAS plana + `MetaData.Errors`)**: se conserva la raíz
  `Success/Status/Message/Data` que ya emite `utils_oas`, y se añade `MetaData`
  solo para avisos de expansión. Alternativas descartadas: Forma 1 (anidar
  `Success/Status` dentro de `MetaData`, descartada por discrepar del estándar
  OAS plano del ecosistema y coexistir mal con los errores de Beego); híbrido
  `success/data/metadata` (descartado por el usuario a favor de migrar a `Data`).
- **Migración de alcance B2 (todo el API)**: todos los verbos migran al contrato
  Forma 3. Rationale: POST/PUT/GET comparten path vía `beego.Any`; migrar solo
  GET dejaría el mismo endpoint con dos formas según el verbo.
- **`Resolver` sobre `clients.HTTPClient`**: el helper delega cada consulta a un
  `Resolver` por `related_service`; HTTPClient se envuelve, no se reescribe.
  Resolver genérico solo para relaciones declaradas y validadas en config.
- **Config de relaciones en YAML cargada al arranque**: `config/` gana parser y
  validador con `gopkg.in/yaml.v3` (promovida a dependencia directa). Config
  inválida aborta el arranque; esto cambia la filosofía tolerante actual de
  `Load()`, de forma deliberada y acotada a los archivos de relaciones.
- **Profundidad efectiva = min(límite global restante, depth de la relación)**;
  reducción por config hija se notifica como aviso `depth`.

## Risks / Trade-offs

- [Cambio BREAKING del contrato de respuesta] → documentar en Swagger; acordar
  con consumidores; evaluar versión/bandera de activación por endpoint (ver
  Open Questions).
- [N+1 al expandir listas] → deduplicación de llaves, consultas agrupadas y
  caché por solicitud; límites de `max_depth` y `max_related_ids`.
- [Latencia y puntos de falla adicionales por cada CRUD] → timeouts, cancelación
  por `context.Context` y política `on_relation_error`.
- [Salto de paradigma de structs tipados a `map[string]any` en el helper] →
  aislar la manipulación dinámica dentro de `helper/`, mantener los servicios
  tipados existentes intactos.
- [Fuga de datos sensibles en avisos/errores] → mensajes solo con ruta lógica,
  nunca URLs internas, credenciales ni cuerpos remotos.

## Migration Plan

1. Introducir el contrato Forma 3 en la capa de respuesta y migrar los endpoints
   existentes, manteniendo los campos base de `Data`.
2. Añadir carga/validación de relaciones y el helper sin activarlo en
   endpoints hasta que la config esté lista.
3. Activar la expansión por endpoint, documentar en Swagger.
4. Rollback: desactivar la lectura con `fields` por endpoint; la raíz OAS plana
   permanece.

## Open Questions

- Versión o bandera de activación para endpoints cuyos clientes dependan
  estrictamente de la forma actual (decisión §9 del boceto; no cambia specs ni
  task breakdown, se puede resolver al integrar por endpoint).
- Confirmar nombres/rutas reales de recursos de cada CRUD y si aceptan consultas
  por múltiples IDs o requieren particionado por `max_related_ids`.
- Valores numéricos predeterminados de `max_depth`, `max_related_ids` y total de
  objetos por respuesta.
