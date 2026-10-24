# Proposal

## Why

Los servicios del API necesitan entregar respuestas con objetos relacionados
(anidados) que viven en otros CRUD, sin que el MID se convierta en dueño de los
datos ni duplique lógica de resolución en cada controlador. Hoy cada servicio
consulta un recurso a la vez y devuelve datos planos; no existe una forma
reutilizable de expandir relaciones ni de reportar errores parciales de forma
tolerante. Esto implementa el contrato definido en
udistrital/agora_documentacion#29.

## What Changes

- Nueva capacidad de expansión de objetos anidados: un helper reutilizable en
  `helper/` que orquesta la recursividad, construye el árbol de selección,
  detecta ciclos, aplica límites (profundidad, IDs por relación) y delega cada
  consulta a un `Resolver` por servicio. Los controladores no implementan
  recursión; solo construyen opciones, invocan el helper y serializan.
- Configuración declarativa por servicio en `config/relaciones_<servicio>.yaml`,
  cargada y validada al arranque. Un archivo ausente, inválido o inconsistente
  impide el arranque del API.
- Resolución mediante `Resolver` sobre `clients.HTTPClient` (especializado por
  servicio y genérico para relaciones declaradas sin adaptador propio).
- Parámetro `fields` en los controladores de lectura para seleccionar rutas
  (`Id,Tercero.Id,Tercero.NombreCompleto`), validadas contra `allowed_fields`
  jerárquico; un campo no permitido produce `400 Bad Request`.
- **BREAKING**: migración del contrato de respuesta de todo el API al formato
  OAS estándar plano más una extensión `MetaData.Errors`. La forma actual
  `{success, data, error}` (campos en minúscula) se reemplaza por
  `{Success, Status, Message, Data, MetaData?}`, con `Status` como string y
  `MetaData.Errors[]` presente solo cuando hay avisos de expansión (errores de
  relación tolerados, reducción de profundidad, ciclos o uso del resolver
  genérico). Esto alinea el API con `utils_oas` (customerror/errorctrl) que ya
  emite esa raíz plana en los errores de Beego.

## Capabilities

### New Capabilities
- `nested-object-expansion`: resolución y expansión recursiva de relaciones
  configuradas entre CRUD, con selección por `fields`, límites, detección de
  ciclos, caché por solicitud y reporte tolerante de errores de relación.
- `api-response-contract`: contrato de respuesta uniforme del API en formato OAS
  plano (`Success/Status/Message/Data`) con la extensión opcional
  `MetaData.Errors` para avisos de expansión.

### Modified Capabilities
<!-- No hay specs existentes en el proyecto; todas las capacidades son nuevas. -->

## Impact

- **Código nuevo**: paquete `helper/` (árbol de selección, expansor recursivo,
  tipos `RelationConfig`/`ExpandOptions`/`RelationError`/`ExpandResult`,
  `Resolver`).
- **Config**: `config/config.go` gana carga y validación de archivos
  `relaciones_<servicio>.yaml` (usando `gopkg.in/yaml.v3`, ya presente en
  go.sum) y pasa a abortar el arranque ante configuración inválida. Las URLs
  base siguen viniendo de `conf/app.conf`.
- **Controladores**: `controllers/registro_persona.go` lee `fields` de la query
  string en lectura e invoca el helper. Se reemplazan `respuestaRegistro` y
  `respuestaOperacion` por el contrato Forma 3 en todos los verbos.
- **Clientes**: `clients.HTTPClient` se envuelve como `Resolver`; se reutiliza
  `decodeAPIList`/`decodeAPI` que ya decodifica envelope OAS y lista directa.
- **Endpoints afectados**: `/v1/registro/persona-natural` y
  `/v1/registro/persona-juridica` (GET con expansión; POST/PUT migran de forma
  de respuesta). BREAKING para clientes que leen `success`/`data`/`error`.
- **Dependencias**: promover `gopkg.in/yaml.v3` de indirect a directa.
- **Pruebas**: unitarias del helper y de controlador; regresión del nuevo
  contrato de respuesta.
- **Documentación**: actualizar Swagger con el contrato público y `fields`.
