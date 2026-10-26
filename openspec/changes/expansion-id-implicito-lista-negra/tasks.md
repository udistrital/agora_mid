# Tasks

## 1. Constantes de tipos de aviso e invariante

- [x] 1.1 En `helper/expand.go` añadir la constante `KindGenericResolver = "generic_resolver"` junto a `KindRelation/KindDepth/KindCycle` y hacer que `AvisoGenericResolver` use la constante; verificar con `go test ./helper/...` que el aviso emitido conserva `Kind: "generic_resolver"`
- [x] 1.2 Agregar una prueba en `helper/` que afirme el invariante "todo `Kind` emitido tiene una constante declarada": recorre el conjunto de constantes de aviso ({relation, depth, cycle, generic_resolver}) y verifica que cada ruta de emisión usa una de ellas y que no hay ningún `Kind` emitido fuera del conjunto; verificar con `go test ./helper/...`

## 2. Proyección estricta de escalares por fields

- [x] 2.1 En `helper/expand.go` recortar los escalares del registro devuelto por el resolver antes de anidarlo, de modo que una expansión con selección de `fields` para un nivel entregue SOLO los campos escalares seleccionados en ese nivel (más las relaciones hijas materializadas), tratando los segmentos intermedios como contenedores esqueleto del camino solicitado; verificar con una prueba en `helper/expand_test.go` que `fields=Tercero.NombreCompleto` entrega `Tercero` con `NombreCompleto` y sin ningún otro escalar (ni `Id` ni campos fuera de la selección), sin aviso en `MetaData`
- [x] 2.2 Garantizar que cuando no hay selección de `fields` para un nivel no se aplica recorte proyectivo (la entidad conserva los campos que entrega el resolver), aplicando el recorte únicamente cuando existe selección para ese nivel; verificar con una prueba en `helper/expand_test.go` que una expansión sin `fields` conserva los campos de la entidad
- [x] 2.3 En `helper/seleccion.go` introducir en `NodoSeleccion` un marcador de "relación completa" que `ParsearFields` activa cuando una ruta termina en esa relación (hoja); en `helper/expand.go` hacer que un nodo marcado completo expanda su subárbol por configuración (hasta la profundidad efectiva) e ignore los escalares/hijos acumulados por otras rutas (precedencia por relación de la selección más amplia), mientras un nodo parcial proyecta a sus escalares seleccionados y expande solo sus hijos nombrados. Verificar con pruebas de tabla en `helper/expand_test.go`, usando fixtures compartidos, la matriz de selección en el mismo nivel: (1) `[escalar]` entrega solo ese escalar; (2) `[relación hoja]` la expande completa; (3) `[escalar, escalar]` entrega solo esos escalares; (4) `[escalar, relación-hoja misma]` la relación completa subsume el escalar; (5) `[relación hoja, relación hoja]` distintas coexisten completas; (6) `[relación hoja, descendiente misma]` la completa subsume el descendiente; (7) `[descendiente, descendiente]` de la misma relación conviven como esqueleto; (8) `[relación hoja, descendiente de hermana]` coexisten con trato distinto (una completa, otra esqueleto). Verificar además que las relaciones no alcanzadas por ninguna ruta no se consultan

## 3. Id implícito en selección y expansión

- [x] 3.1 En `helper/seleccion.go` permitir que la ruta del `Id` de una relación sea válida aunque `Id` no esté en `allowed_fields` (p. ej. `campoPermitido` acepta `Id` para un nivel de relación); verificar con una prueba en `helper/seleccion_test.go` que `fields=Tercero.Id` se acepta cuando `allowed_fields` de `Tercero` no lista `Id`, y que un campo no-`Id` fuera de `allowed_fields` sigue rechazándose. La omisión del `Id` por `fields` (cuando la selección del nivel no lo menciona) ya queda cubierta por la proyección estricta del grupo 2 y no requiere trato especial del `Id`
- [x] 3.2 En `helper/expand.go` garantizar que el `Id` de cada entidad expandida se conserva por defecto (expansión sin `fields`) aunque `allowed_fields` no lo liste, apoyándose en el no-recorte por defecto del grupo 2; verificar con una prueba en `helper/expand_test.go` que una relación con `allowed_fields` sin `Id` se expande con su `Id` presente

## 4. Carga y validación de la lista negra al arranque

- [ ] 4.1 Añadir en `config/` el tipo y el cargador de `config/global_blacklist.yaml` (estructura `global_blacklist: related_api -> related_table -> [field]`, palabra reservada `ANY`), tolerando archivo ausente o vacío como "sin supresión"; verificar con una prueba en `config/` que un archivo de ejemplo parsea a la estructura esperada y que la ausencia del archivo no es error
- [ ] 4.2 Implementar la validación de arranque de la lista negra: un componente exacto (sin `ANY`) debe existir en el modelo conocido (unión de `related_api`/`related_table`/`allowed_fields` + `Id` + `related_field` de las `ServiceConfig` cargadas); `ANY` no se verifica; `ANY/ANY/ANY` se rechaza; verificar con pruebas en `config/` que un componente exacto inexistente aborta, que `ANY/ANY/ANY` aborta y que una entrada con `ANY` en un nivel no falla por inexistencia
- [ ] 4.3 Crear `config/global_blacklist.yaml` con contenido inicial (comentarios explicativos y, si aplica, entradas de ejemplo o archivo sin entradas) coherente con el modelo del proveedor; verificar con `go build ./...` y el arranque/validación de 4.2 que el archivo carga sin abortar

## 5. Recorte por lista negra sobre la salida de todo resolver

- [ ] 5.1 En `helper/expand.go` extender el choke-point de proyección del grupo 2 para recortar también los campos vetados del registro devuelto por el resolver antes de anidarlo, aplicando precedencia lista negra > `allowed_fields` > `Id` implícito, usando `RelatedAPI`/`RelatedTable` de la relación para casar; verificar con una prueba en `helper/expand_test.go` que un campo permitido por `allowed_fields` y vetado se omite, y que un `Id` implícito vetado se omite, sin aviso en `MetaData`
- [ ] 5.2 Verificar que el resolver genérico obedece la lista negra: una relación resuelta por el genérico cuyo registro trae un campo vetado lo entrega recortado; verificar con una prueba en `helper/` que cubre la rama del resolver genérico y afirma el campo ausente (más el aviso `generic_resolver` intacto)
- [ ] 5.3 Hacer que pedir explícitamente en `fields` un campo vetado produzca el mismo resultado que un campo no permitido, reutilizando el rechazo genérico existente sin distinguir el motivo ni emitir aviso; verificar con una prueba en `helper/seleccion_test.go` que `fields` con una ruta vetada se rechaza igual que una ruta no permitida

## 6. Limpieza de configuración (opcional)

- [ ] 6.1 Retirar `Id` de las `allowed_fields` de las relaciones en `config/relaciones_proveedor.yaml` (ya innecesario por el `Id` implícito) y verificar con `go test ./...` que la expansión sigue entregando el `Id` de cada relación y que todo compila

## 7. Verificación integral

- [ ] 7.1 Ejecutar `go build ./...` y `go test ./...` para confirmar que la proyección estricta de `fields`, el `Id` implícito, la lista negra (incluida la ruta del resolver genérico), el rechazo de un campo vetado en `fields` y el invariante de tipos de aviso funcionan en conjunto sin regresiones

## Workflow follow-up

- Coordinar el orden de implementación con el cambio en curso
  `proveedor-read-post-consulta` (preferible construir primero la migración de
  transporte para no rehacer la rutina compartida de validación de `fields`).
- Ejecutar `openspec validate expansion-id-implicito-lista-negra` antes de
  sincronizar y archivar el cambio.
