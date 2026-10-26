# Tasks

## 1. Constantes de tipos de aviso e invariante

- [x] 1.1 En `helper/expand.go` añadir la constante `KindGenericResolver = "generic_resolver"` junto a `KindRelation/KindDepth/KindCycle` y hacer que `AvisoGenericResolver` use la constante; verificar con `go test ./helper/...` que el aviso emitido conserva `Kind: "generic_resolver"`
- [x] 1.2 Agregar una prueba en `helper/` que afirme el invariante "todo `Kind` emitido tiene una constante declarada": recorre el conjunto de constantes de aviso ({relation, depth, cycle, generic_resolver}) y verifica que cada ruta de emisión usa una de ellas y que no hay ningún `Kind` emitido fuera del conjunto; verificar con `go test ./helper/...`

## 2. Id implícito en selección y expansión

- [ ] 2.1 En `helper/seleccion.go` permitir que la ruta del `Id` de una relación sea válida aunque `Id` no esté en `allowed_fields` (p. ej. `campoPermitido` acepta `Id` para un nivel de relación); verificar con una prueba en `helper/seleccion_test.go` que `fields=Tercero.Id` se acepta cuando `allowed_fields` de `Tercero` no lista `Id`, y que un campo no-`Id` fuera de `allowed_fields` sigue rechazándose
- [ ] 2.2 En `helper/expand.go` garantizar que el `Id` de cada entidad expandida se conserva por defecto (expansión sin `fields`) aunque `allowed_fields` no lo liste; verificar con una prueba en `helper/expand_test.go` que una relación con `allowed_fields` sin `Id` se expande con su `Id` presente
- [ ] 2.3 Verificar la omisión por `fields`: cuando `fields` de una relación no menciona su `Id`, el `Id` no se incluye para esa solicitud y no se emite aviso; verificar con una prueba en `helper/expand_test.go` (o `seleccion_test.go`) que `fields=Tercero.NombreCompleto` entrega `Tercero` sin `Id` y sin entrada en `MetaData.Errors`

## 3. Carga y validación de la lista negra al arranque

- [ ] 3.1 Añadir en `config/` el tipo y el cargador de `config/global_blacklist.yaml` (estructura `global_blacklist: related_api -> related_table -> [field]`, palabra reservada `ANY`), tolerando archivo ausente o vacío como "sin supresión"; verificar con una prueba en `config/` que un archivo de ejemplo parsea a la estructura esperada y que la ausencia del archivo no es error
- [ ] 3.2 Implementar la validación de arranque de la lista negra: un componente exacto (sin `ANY`) debe existir en el modelo conocido (unión de `related_api`/`related_table`/`allowed_fields` + `Id` + `related_field` de las `ServiceConfig` cargadas); `ANY` no se verifica; `ANY/ANY/ANY` se rechaza; verificar con pruebas en `config/` que un componente exacto inexistente aborta, que `ANY/ANY/ANY` aborta y que una entrada con `ANY` en un nivel no falla por inexistencia
- [ ] 3.3 Crear `config/global_blacklist.yaml` con contenido inicial (comentarios explicativos y, si aplica, entradas de ejemplo o archivo sin entradas) coherente con el modelo del proveedor; verificar con `go build ./...` y el arranque/validación de 3.2 que el archivo carga sin abortar

## 4. Recorte por lista negra sobre la salida de todo resolver

- [ ] 4.1 En `helper/expand.go` introducir el choke-point de proyección que recorta los campos vetados del registro devuelto por el resolver antes de anidarlo, aplicando precedencia lista negra > `allowed_fields` > `Id` implícito, usando `RelatedAPI`/`RelatedTable` de la relación para casar; verificar con una prueba en `helper/expand_test.go` que un campo permitido por `allowed_fields` y vetado se omite, y que un `Id` implícito vetado se omite, sin aviso en `MetaData`
- [ ] 4.2 Verificar que el resolver genérico obedece la lista negra: una relación resuelta por el genérico cuyo registro trae un campo vetado lo entrega recortado; verificar con una prueba en `helper/` que cubre la rama del resolver genérico y afirma el campo ausente (más el aviso `generic_resolver` intacto)
- [ ] 4.3 Hacer que pedir explícitamente en `fields` un campo vetado produzca el mismo resultado que un campo no permitido, reutilizando el rechazo genérico existente sin distinguir el motivo ni emitir aviso; verificar con una prueba en `helper/seleccion_test.go` que `fields` con una ruta vetada se rechaza igual que una ruta no permitida

## 5. Limpieza de configuración (opcional)

- [ ] 5.1 Retirar `Id` de las `allowed_fields` de las relaciones en `config/relaciones_proveedor.yaml` (ya innecesario por el `Id` implícito) y verificar con `go test ./...` que la expansión sigue entregando el `Id` de cada relación y que todo compila

## 6. Verificación integral

- [ ] 6.1 Ejecutar `go build ./...` y `go test ./...` para confirmar que el `Id` implícito, la lista negra (incluida la ruta del resolver genérico), el rechazo de un campo vetado en `fields` y el invariante de tipos de aviso funcionan en conjunto sin regresiones

## Workflow follow-up

- Coordinar el orden de implementación con el cambio en curso
  `proveedor-read-post-consulta` (preferible construir primero la migración de
  transporte para no rehacer la rutina compartida de validación de `fields`).
- Ejecutar `openspec validate expansion-id-implicito-lista-negra` antes de
  sincronizar y archivar el cambio.
