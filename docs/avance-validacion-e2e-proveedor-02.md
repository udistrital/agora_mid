# Segundo avance de validación E2E - Consulta de proveedores

**Proyecto:** `agora_mid_issue49`
**Rama de trabajo:** `feature/49-add-collection-read`
**Issue de seguimiento:** `udistrital/agora_documentacion#49`
**PR relacionado:** `udistrital/agora_mid#4`
**Fecha de corte:** 30 de octubre de 2026
**Endpoint:** `POST /v1/proveedor/consulta/lista`
**Documento anterior:** `docs/avance-validacion-e2e-proveedor.md`

---

## 1. Propósito y alcance de este segundo avance

Este documento amplía el primer informe de validación E2E. Su objetivo es dejar trazabilidad técnica de las pruebas adicionales ejecutadas con los servicios reales `terceros_crud` y `gestion_personas_crud`, así como de la inspección del código fuente y de las pruebas unitarias del MID.

El alcance de este avance comprende:

- Ampliación del conjunto de datos de prueba a tres proveedores.
- Validación de filtros simples y combinados.
- Validación del operador `contains`.
- Validación de paginación mediante `limit` y `offset`.
- Verificación del comportamiento cuando el desplazamiento supera los registros disponibles.
- Inspección de la respuesta del CRUD de proveedores cuando no hay resultados.
- Rastreo del origen de un registro vacío que termina materializado como un proveedor con valores cero en el MID.
- Inspección del mecanismo utilizado para calcular `MetaData.Total`.
- Revisión de pruebas unitarias de servicios, controladores y expansión de relaciones.
- Revisión del manejo de errores de expansión bajo las políticas `notify` y `fail`.

Este documento **no registra correcciones implementadas**. Los cambios sugeridos se presentan como propuestas sujetas a revisión, autorización, implementación y pruebas posteriores.

## 2. Relación con el primer informe

El primer informe, `docs/avance-validacion-e2e-proveedor.md`, documentó la preparación inicial de datos, el arranque del MID, la consulta básica y los problemas de expansión anidada.

Este segundo informe continúa esa validación. No reemplaza ni modifica el documento anterior.

| Aspecto | Primer informe | Segundo informe |
|---|---|---|
| Escenario de datos | Un proveedor principal y relaciones asociadas | Tres proveedores para probar filtros y paginación |
| Consulta básica | Verificada | Se conserva como antecedente |
| Expansión de relaciones | Parcial; relaciones anidadas pendientes | Se revisa adicionalmente el manejo de errores de expansión |
| Filtros | Pendientes | Se prueban filtros simples, combinados y `contains` |
| Paginación | Pendiente | Se prueban varias páginas y un desplazamiento fuera de rango |
| Total global | Consulta inicial devolvía 1 | Se identifica que el total puede confundirse con el tamaño de página |
| Respuesta sin registros | No documentada | Se reproduce y rastrea el registro vacío |
| Pruebas unitarias | Pendientes | Se inspeccionan pruebas existentes y vacíos de cobertura |

## 3. Entorno E2E utilizado

| Componente | Dirección local | Base de datos |
|---|---|---|
| `terceros_crud` | `127.0.0.1:8084` | `terceros_crud_test` |
| `gestion_personas_crud` | `127.0.0.1:8086` | `gestion_personas_crud_test` |
| `agora_mid_issue49` | `127.0.0.1:8090` | No aplica |

La base PostgreSQL de pruebas se ejecutó en Docker. Se identificó el contenedor `bb90892d068e`, con el puerto host `5433` redirigido al puerto interno `5432`.

Bases de datos identificadas en el entorno:

- `terceros_crud_test`
- `gestion_personas_crud_test`
- `cotizaciones_crud_mid_test`

Las variables de entorno y credenciales no se reproducen en este documento.

## 4. Preparación y ampliación de datos de prueba

### 4.1. Tipo de contribuyente

En `terceros_crud_test`, esquema `terceros`, se utilizó el tipo de contribuyente:

| ID | Nombre | Código |
|---:|---|---|
| 1 | Persona natural | `P_NATURAL` |

### 4.2. Terceros

Se prepararon tres terceros para relacionarlos con los proveedores:

| ID | Nombre |
|---:|---|
| 1 | `Proveedor de prueba E2E` |
| 2 | `Proveedor E2E Paginacion 02` |
| 3 | `Proveedor E2E Paginacion 03` |

Los terceros 2 y 3 se crearon mediante `POST http://127.0.0.1:8084/v1/tercero`, con respuesta HTTP `201 Created`.

El tercero 1 ya formaba parte del escenario inicial descrito en el primer informe.

### 4.3. Proveedores

En `gestion_personas_crud_test`, esquema `proveedores`, se crearon los siguientes registros mediante `POST http://127.0.0.1:8086/v1/proveedor`.

| ID del proveedor | TerceroId | TipoRegistroId | Activo | Descripción |
|---:|---:|---:|---|---|
| 2 | 1 | 1 | `true` | `Proveedor de prueba E2E para validar consulta de lista` |
| 3 | 2 | 1 | `true` | `Proveedor E2E Paginacion 02` |
| 4 | 3 | 1 | `true` | `Proveedor E2E Paginacion 03` |

La fecha de creación se envió explícitamente porque el controlador CRUD no la asignó automáticamente en el escenario observado. El valor utilizado fue:

```json
{
  "FechaCreacion": "2026-10-30T00:00:00Z"
}
```

Los registros se verificaron mediante consultas SQL en la base de pruebas. Se confirmó que los tres proveedores estaban activos y que sus identificadores de tercero correspondían a los registros preparados.

**Nota de trazabilidad:** el informe registra el resultado de las verificaciones SQL realizadas. No reproduce una sentencia SQL literal porque el texto exacto de todas esas consultas no quedó conservado en la evidencia disponible para este documento. No se debe presentar una consulta reconstruida como si hubiera sido ejecutada.

## 5. Configuración de campos filtrables del MID

Archivo inspeccionado: `config/relaciones_proveedor.yaml`.

Configuración observada:

```yaml
filterable_fields:
  - Id
  - TerceroId
  - TipoRegistroId
  - DescripcionPortafolio
```

Esta configuración permite que el listado valide y admita filtros sobre esos campos. Los nombres de campo utilizados en las pruebas respetaron la nomenclatura configurada.

## 6. Pruebas E2E de filtros

Las peticiones se realizaron contra:

```text
POST http://127.0.0.1:8090/v1/proveedor/consulta/lista
```

### 6.1. Filtro por `TerceroId`

Cuerpo de la petición:

```json
{
  "query": ["TerceroId:1"],
  "fields": [],
  "limit": 10,
  "offset": 0
}
```

Resultado observado:

- HTTP `200 OK`.
- Se recuperó el proveedor con `Id: 2`.
- `MetaData.Total` fue `1`.

**Conclusión:** el filtro simple por `TerceroId` funcionó para el registro de prueba.

### 6.2. Filtro por identificador del proveedor

Cuerpo:

```json
{
  "query": ["Id:2"],
  "fields": [],
  "limit": 10,
  "offset": 0
}
```

Resultado: HTTP `200 OK`, proveedor ID 2 y total 1.

**Conclusión:** el filtro por `Id` funcionó.

### 6.3. Filtro por tipo de registro

Cuerpo:

```json
{
  "query": ["TipoRegistroId:1"],
  "fields": [],
  "limit": 10,
  "offset": 0
}
```

Resultado observado: HTTP `200 OK`, proveedor ID 2 y total 1.

**Conclusión:** el filtro por `TipoRegistroId` funcionó en el escenario consultado.

### 6.4. Filtro de texto con operador `contains`

Cuerpo:

```json
{
  "query": ["DescripcionPortafolio__contains:prueba E2E"],
  "fields": [],
  "limit": 10,
  "offset": 0
}
```

Resultado observado:

- HTTP `200 OK`.
- Se recuperó el proveedor ID 2.
- `MetaData.Total` fue `1`.

**Conclusión:** el operador `contains` funcionó para la descripción consultada.

### 6.5. Filtros combinados

Cuerpo:

```json
{
  "query": ["TerceroId:1", "TipoRegistroId:1"],
  "fields": [],
  "limit": 10,
  "offset": 0
}
```

Resultado observado: HTTP `200 OK`, proveedor ID 2 y total 1.

La prueba confirmó el comportamiento conjunto de los dos criterios para este escenario. La interpretación de la prueba fue una combinación AND: el registro debe cumplir ambos filtros.

### 6.6. Resumen de filtros

| Prueba | Criterio | HTTP | Resultado observado | Estado |
|---|---|---:|---|---|
| Filtro por tercero | `TerceroId:1` | 200 | Proveedor 2; total 1 | Correcto |
| Filtro por proveedor | `Id:2` | 200 | Proveedor 2; total 1 | Correcto |
| Filtro por tipo | `TipoRegistroId:1` | 200 | Proveedor 2; total 1 | Correcto |
| Filtro textual | `DescripcionPortafolio__contains:prueba E2E` | 200 | Proveedor 2; total 1 | Correcto |
| Filtros combinados | `TerceroId:1` + `TipoRegistroId:1` | 200 | Proveedor 2; total 1 | Correcto |

Los resultados acreditan los casos concretos ejecutados; no equivalen a una prueba exhaustiva de todos los operadores y combinaciones posibles.

## 7. Pruebas E2E de paginación

Se utilizaron los tres proveedores activos, ordenados según el comportamiento observado en el CRUD.

### 7.1. Primera página

Petición:

```json
{
  "query": [],
  "fields": [],
  "limit": 1,
  "offset": 0
}
```

Resultado:

- HTTP `200 OK`.
- Se recibió el proveedor ID 2.
- `MetaData.Total` fue `1`.

### 7.2. Segunda página

Petición:

```json
{
  "query": [],
  "fields": [],
  "limit": 1,
  "offset": 1
}
```

Resultado:

- HTTP `200 OK`.
- Se recibió el proveedor ID 3.
- `MetaData.Total` fue `1`.

### 7.3. Tercera página

Petición:

```json
{
  "query": [],
  "fields": [],
  "limit": 1,
  "offset": 2
}
```

Resultado:

- HTTP `200 OK`.
- Se recibió el proveedor ID 4.
- `MetaData.Total` fue `1`.

### 7.4. Consulta sin restricción de página

Petición:

```json
{
  "query": [],
  "fields": [],
  "limit": 10,
  "offset": 0
}
```

Resultado:

- HTTP `200 OK`.
- Se recibieron los proveedores IDs 2, 3 y 4.
- `MetaData.Total` fue `3`.

### 7.5. Desplazamiento posterior al último registro

Petición:

```json
{
  "query": [],
  "fields": [],
  "limit": 1,
  "offset": 3
}
```

Resultado observado:

- HTTP `200 OK`.
- El MID devolvió un elemento con valores cero:

```json
{
  "Activo": false,
  "Id": 0,
  "TerceroId": 0,
  "TipoRegistroId": 0
}
```

- `MetaData.Total` fue `1`.

Este resultado no representa una página vacía correcta. Se esperaba que `Data` fuera una lista vacía y que el total global siguiera reflejando los tres registros coincidentes.

### 7.6. Matriz de paginación

| `limit` | `offset` | Resultado esperado | Resultado observado |
|---:|---:|---|---|
| 1 | 0 | Proveedor 2 | Proveedor 2; total 1 |
| 1 | 1 | Proveedor 3 | Proveedor 3; total 1 |
| 1 | 2 | Proveedor 4 | Proveedor 4; total 1 |
| 10 | 0 | Proveedores 2, 3 y 4 | Proveedores 2, 3 y 4; total 3 |
| 1 | 3 | `Data: []`; total global 3 | Objeto con campos cero; total 1 |

**Conclusión:** el desplazamiento y el límite seleccionaron correctamente los tres registros existentes. Sin embargo, el caso fuera de rango reveló un defecto en la representación de la lista vacía y el conteo observado no corresponde al total global.

## 8. Inspección directa del CRUD cuando no existen resultados

Se consultó directamente el CRUD:

```text
GET http://127.0.0.1:8086/v1/proveedor?limit=1&offset=3
```

Respuesta HTTP `200 OK`:

```json
{
  "Data": [{}],
  "Message": "Request successful",
  "Status": "200",
  "Success": true
}
```

La respuesta demuestra que el CRUD no entrega `Data: []` cuando la consulta no encuentra registros. Entrega una lista que contiene un mapa vacío: `[{}]`.

Este resultado permite localizar el origen del comportamiento antes de la expansión de relaciones del MID.

## 9. Análisis de código del CRUD: origen de `[{}]`

### 9.1. Modelo `GetAllProveedor`

Archivo inspeccionado: `gestion_personas_crud/models/proveedor.go`.

Fragmento relevante:

```go
var l []Proveedor
qs = qs.OrderBy(sortFields...)
if _, err = qs.Limit(limit, offset).All(&l, fields...); err == nil {
    if len(fields) == 0 {
        for _, v := range l {
            ml = append(ml, v)
        }
    } else {
        // trim unused fields
        for _, v := range l {
            m := make(map[string]interface{})
            val := reflect.ValueOf(v)
            for _, fname := range fields {
                m[fname] = val.FieldByName(fname).Interface()
            }
            ml = append(ml, m)
        }
    }
    return ml, nil
}
return nil, err
```

Cuando no existen registros, el bucle no agrega elementos a `ml`; por tanto, `ml` permanece `nil`.

### 9.2. Controlador `GetAll`

Archivo inspeccionado: `gestion_personas_crud/controllers/proveedor.go`.

Fragmento relevante:

```go
l, err := models.GetAllProveedor(query, fields, sortby, order, offset, limit)
if err != nil {
    // manejo del error
} else {
    if l == nil {
        l = append(l, map[string]interface{}{})
    }
    c.Data["json"] = map[string]interface{}{
        "Success": true,
        "Status": "200",
        "Message": "Request successful",
        "Data": l,
    }
}
```

### 9.3. Causa raíz confirmada

La causa inmediata de `[{}]` está en el controlador CRUD: cuando el modelo devuelve `nil` por no encontrar filas, el controlador agrega explícitamente un mapa vacío.

La secuencia observada es:

1. La consulta SQL no encuentra registros.
2. `GetAllProveedor` retorna `ml == nil`.
3. El controlador detecta `l == nil`.
4. El controlador agrega `map[string]interface{}{}`.
5. La respuesta HTTP contiene `Data: [{}]`.
6. El MID interpreta ese elemento como un registro de proveedor.

**Clasificación:** causa raíz confirmada por inspección del código y reproducción de la respuesta HTTP directa del CRUD.

### 9.4. Propuesta de corrección en el CRUD

La respuesta del CRUD debería representar la ausencia de resultados mediante una lista vacía:

```json
{
  "Data": [],
  "Message": "Request successful",
  "Status": "200",
  "Success": true
}
```

La implementación debe asegurar que el tipo de colección retornado por el modelo/controlador se serialice como arreglo vacío y no como `null` ni como una lista con un mapa vacío.

Esta corrección es una propuesta. No se ha modificado `gestion_personas_crud` ni se ha ejecutado una prueba posterior que la valide.

## 10. Análisis de código del MID: materialización de `{}` como proveedor con valores cero

### 10.1. Estructura `ProveedorRecord`

Archivo inspeccionado: `services/proveedor.go`.

```go
type ProveedorRecord struct {
    Id             int  `json:"Id"`
    TerceroId      int  `json:"TerceroId"`
    TipoRegistroId int  `json:"TipoRegistroId"`
    Activo         bool `json:"Activo"`
}
```

### 10.2. Decodificación de la lista

Archivo inspeccionado: `services/api_response.go`.

La función `decodeAPIListWithTotal[T]` acepta una respuesta directa como lista o un sobre que contiene `Data`. En el caso del sobre, el comportamiento relevante es:

```go
var envelope apiListEnvelope[T]
if err := json.Unmarshal(raw, &envelope); err != nil {
    return apiListResult[T]{}, fmt.Errorf(...)
}
if envelope.Data == nil {
    return apiListResult[T]{}, errors.New("la respuesta del CRUD no contiene una lista Data")
}
total := len(envelope.Data)
if envelope.Total != nil {
    total = *envelope.Total
}
return apiListResult[T]{Data: envelope.Data, Total: total}, nil
```

La función verifica que `Data` exista como lista, pero no valida que cada elemento represente un registro válido.

En Go, al deserializar `{}` en una estructura cuyos campos son numéricos y booleanos, los campos no presentes conservan sus valores cero. En este caso:

- `Id` queda en `0`.
- `TerceroId` queda en `0`.
- `TipoRegistroId` queda en `0`.
- `Activo` queda en `false`.

Por eso, `Data: [{}]` no produce necesariamente un error de deserialización.

### 10.3. Construcción de mapas en `ListarProveedores`

Archivo inspeccionado: `services/proveedor.go`.

Fragmento relevante:

```go
lista, err := decodeAPIListWithTotal[ProveedorRecord](raw)
...
records := make([]map[string]any, 0, len(lista.Data))
for _, registro := range lista.Data {
    records = append(records, map[string]any{
        "Id":             registro.Id,
        "TerceroId":      registro.TerceroId,
        "TipoRegistroId": registro.TipoRegistroId,
        "Activo":         registro.Activo,
    })
}
return helper.ListCollection(ctx, records, lista.Total, ...)
```

El MID transforma cada elemento deserializado en un mapa, incluso cuando `Id` es cero.

### 10.4. Causa inmediata confirmada en el MID

La cadena de transformación es:

```text
CRUD: Data: [{}]
       |
       v
JSON decode en ProveedorRecord
       |
       v
ProveedorRecord{Id: 0, TerceroId: 0, TipoRegistroId: 0, Activo: false}
       |
       v
ListarProveedores construye map[string]any
       |
       v
helper.ListCollection procesa el registro
       |
       v
Respuesta MID con proveedor de valores cero
```

**Causa raíz primaria:** el CRUD transforma una lista sin resultados en `[{}]`.

**Causa de propagación en el MID:** el servicio acepta el elemento vacío y lo convierte en una estructura de proveedor sin validar su identificador.

Ambas capas contribuyen al resultado observado. La corrección debería proteger tanto el contrato de salida del CRUD como la validación de entrada del MID.

### 10.5. Propuesta de validación en el MID

Se propone validar que cada registro recibido tenga un identificador válido (`Id > 0`) antes de construir el mapa y continuar con `ListCollection`.

La validación no debería descartar silenciosamente registros defectuosos. Debe diferenciar:

- Una lista vacía válida (`Data: []`).
- Un registro malformado (`Data: [{}]`).
- Un registro con identificador inválido (`Id: 0`).

Para los dos últimos escenarios se propone devolver un error de respuesta inválida del CRUD, con contexto suficiente para diagnóstico interno y sin exponer datos sensibles al cliente.

La propuesta no está implementada ni validada.

## 11. Análisis del cálculo de `MetaData.Total`

### 11.1. Comportamiento observado

Con tres proveedores existentes, las pruebas entregaron:

| Petición | Cantidad de elementos en `Data` | `MetaData.Total` observado |
|---|---:|---:|
| `limit=1`, `offset=0` | 1 | 1 |
| `limit=1`, `offset=1` | 1 | 1 |
| `limit=1`, `offset=2` | 1 | 1 |
| `limit=10`, `offset=0` | 3 | 3 |
| `limit=1`, `offset=3` | Registro inválido | 1 |

El valor `1` en las páginas individuales coincide con el tamaño de página, pero no con el total de registros coincidentes, que es tres.

### 11.2. Respuesta del CRUD

La consulta directa al CRUD con `limit=1&offset=0` devolvió el registro, pero no incluyó un campo `Total` en el cuerpo ni una cabecera `Total-Count`.

La consulta directa con `limit=1&offset=3` devolvió `Data: [{}]`, también sin un total global.

### 11.3. Fallback del MID

Archivo: `services/api_response.go`.

Fragmento inspeccionado:

```go
total := len(envelope.Data)
if envelope.Total != nil {
    total = *envelope.Total
}
return apiListResult[T]{Data: envelope.Data, Total: total}, nil
```

Cuando el CRUD no proporciona `Total`, el MID utiliza `len(envelope.Data)`.

Esto explica los resultados:

- Una página de un elemento produce `Total = 1`.
- Una consulta que devuelve tres elementos produce `Total = 3`.
- El MID no puede inferir el total global a partir de una página limitada.

### 11.4. Causa raíz confirmada

El CRUD no expone el total global de registros coincidentes y el MID usa el tamaño de `Data` como fallback.

Por tanto, cuando hay paginación, `MetaData.Total` puede representar la cantidad de elementos de la página y no el total de resultados que cumplen los filtros.

### 11.5. Propuesta de solución

Se propone que el CRUD calcule el total global aplicando los mismos filtros de la consulta, pero antes de aplicar `limit` y `offset`, y que lo incluya en la respuesta mediante un campo `Total` o un mecanismo contractual equivalente.

El MID ya contempla `envelope.Total`; por ello, si el CRUD entrega el total correcto, el MID puede utilizarlo.

También debe definirse qué hacer cuando un CRUD no entrega el total:

- No presentar `len(Data)` como si fuera el total global.
- Definir explícitamente si la ausencia de `Total` es un error contractual o si se permite un valor de total desconocido.
- Mantener coherencia entre el contrato documentado y la respuesta real.

No se ha implementado esta propuesta.

## 12. Revisión del flujo de expansión de relaciones

Aunque los hallazgos principales de este segundo avance están relacionados con listas vacías y conteos, también se revisó el tratamiento de errores de expansión para determinar si era necesario modificar el controlador.

### 12.1. Flujo general

En `services/proveedor.go`, `ListarProveedores` decodifica la respuesta del CRUD, transforma los registros y delega el procesamiento de la colección a `helper.ListCollection`.

En `helper/expand.go`, `Expand` utiliza `expandirValor` y `comoRecursos` para procesar mapas y listas.

La función `comoRecursos` reconoce, entre otros, `map[string]any`, `[]map[string]any` y `[]any`. En el caso de un mapa vacío, sigue siendo un mapa válido y no lo descarta por carecer de identificador.

Esto significa que la capa de expansión no constituye la validación del contrato del registro raíz. La validación de identidad del proveedor debe realizarse antes de entregar el registro a la expansión.

### 12.2. Políticas `notify` y `fail`

La función `manejarErrorRelacionNivel` aplica la política configurada:

```go
if ex.onRelError == config.OnRelationErrorFail {
    return &errorFatalExpansion{path: rutaStr, servicio: rel.RelatedService}
}
for _, recurso := range recursos {
    ex.insertarVacio(recurso, rel)
}
ex.agregarAviso(responses.RelationError{
    Path:    rutaStr,
    Service: rel.RelatedService,
    Message: "no se pudo resolver la relación; se conserva el recurso principal",
    Kind:    KindRelation,
})
```

Comportamiento:

| Política | Comportamiento |
|---|---|
| `fail` | Aborta la expansión con `*errorFatalExpansion` |
| `notify` | Conserva el recurso principal, inserta valor vacío en la relación y agrega aviso |

### 12.3. Clasificación del error fatal

Implementación inspeccionada:

```go
func (e *errorFatalExpansion) Unwrap() error { return e.err }

// EsErrorFatalExpansion informa si err corresponde a un aborto por modo fail.
func EsErrorFatalExpansion(err error) bool {
    _, ok := err.(*errorFatalExpansion)
    return ok
}
```

La función utiliza una aserción directa de tipo. No utiliza `errors.As`, por lo que no reconoce una instancia de `errorFatalExpansion` si llega envuelta por otro error.

La búsqueda de usos encontró referencias en:

- `controllers/proveedor.go`, manejo del listado.
- `controllers/proveedor.go`, manejo de la consulta individual.
- `helper/expand_test.go`, pruebas del helper.
- `services/proveedor.go`, documentación del servicio.

No se encontró evidencia en el flujo inspeccionado de que actualmente se envuelva este error antes de llegar al controlador.

**Clasificación:** observación de robustez, no causa raíz reproducida del defecto de lista vacía o del total.

### 12.4. Pruebas existentes del helper

`TestExpandNotify` simula un fallo del resolver con `503 service unavailable` y comprueba:

- No se devuelve error en modo `notify`.
- Se conserva el recurso principal.
- La relación de cardinalidad `one` queda en `null`.
- Se genera un aviso `relation`.
- El aviso no expone el mensaje remoto.

`TestExpandFail` simula un error de resolver con `504 timeout` y comprueba:

- Se devuelve un error.
- El error es reconocido por `EsErrorFatalExpansion`.

`TestExpandOnRelationErrorInvalido` comprueba que una política inválida produce un error normal y no un error fatal de expansión.

### 12.5. Cobertura de expansión

| Escenario | Cobertura observada |
|---|---|
| Error de relación en modo `notify` | Cubierto |
| Error de relación en modo `fail` | Cubierto |
| Política `on_relation_error` inválida | Cubierto |
| Reconocimiento de error fatal envuelto | No cubierto |
| Traducción HTTP del error fatal en listado | No cubierta en la prueba del listado inspeccionada |
| Timeout HTTP 504 en listado | No cubierto en la prueba del listado inspeccionada |

La limitación de `EsErrorFatalExpansion` debe tratarse como posible mejora independiente. No se debe atribuir a ella el defecto E2E observado.

## 13. Revisión del controlador del endpoint de listado

### 13.1. Prueba existente `TestProveedorHandlerConsultarLista`

Archivo: `controllers/proveedor_test.go`.

La prueba configura un `proveedorCollectionStub` con una respuesta exitosa:

```go
stub := &proveedorCollectionStub{
    resultado: helper.CollectionResult{
        Data:  []any{map[string]any{"Id": 5}},
        Total: 9,
    },
}
```

Envía:

```json
{
  "query": ["Id:5"],
  "fields": ["Tercero.Id"],
  "limit": 2,
  "offset": 1
}
```

Comprueba:

- HTTP `200`.
- Presencia de `"Total":9` en el cuerpo.
- Traslado de `query`, `fields`, `limit` y `offset` al servicio.
- Rechazo de GET, PUT y DELETE con HTTP `405`.
- Cabecera `Allow: POST`.

### 13.2. Stub disponible

El stub contiene un campo `err`:

```go
type proveedorCollectionStub struct {
    resultado helper.CollectionResult
    err       error
    query     []string
    fields    []string
    limit     int
    offset    int
}
```

Y su método retorna el resultado y el error configurados:

```go
func (s *proveedorCollectionStub) ListarProveedores(
    _ context.Context,
    query, fields []string,
    limit, offset int,
) (helper.CollectionResult, error) {
    s.query, s.fields, s.limit, s.offset = query, fields, limit, offset
    return s.resultado, s.err
}
```

Por tanto, la infraestructura ya permite simular un error de dependencia sin modificar el stub.

### 13.3. Ausencia de cobertura de errores del listado

`TestProveedorHandlerConsultarLista` no incluye escenarios en los que `ListarProveedores` devuelve un error.

En el mismo archivo existe un caso llamado `error dependencia`, pero pertenece a `TestProveedorHandlerConsultar`, que prueba la consulta individual. Ese caso espera HTTP `502`; no cubre el endpoint de listado.

**Conclusión:** falta cobertura específica de errores del endpoint de listado.

## 14. Manejo HTTP de errores del listado

Archivo: `controllers/proveedor.go`.

La función `responderErrorLista` aplica las siguientes reglas:

```go
if errors.Is(err, services.ErrProveedorConsultaInvalida) {
    h.responder(w, http.StatusBadRequest, false, err.Error(), nil, nil)
    return
}
if helper.EsErrorFatalExpansion(err) {
    status := http.StatusBadGateway
    var netErr net.Error
    if errors.Is(err, context.DeadlineExceeded) ||
        errors.As(err, &netErr) && netErr.Timeout() {
        status = http.StatusGatewayTimeout
    }
    h.responder(w, status, false, http.StatusText(status), nil, nil)
    return
}
var errorFields *helper.ErrorFields
if errors.As(err, &errorFields) {
    h.responder(w, http.StatusBadRequest, false, errorFields.Error(), nil, nil)
    return
}
h.responder(w, http.StatusBadGateway, false, "Error al consultar los proveedores", nil, nil)
```

Matriz observada:

| Tipo de error | HTTP |
|---|---:|
| Consulta inválida | 400 |
| Error fatal de expansión | 502 |
| Timeout o `context.DeadlineExceeded` | 504 |
| Campos inválidos (`helper.ErrorFields`) | 400 |
| Otro error | 502 |

El controlador ya dispone de un fallback genérico HTTP `502`. No hay evidencia que justifique modificarlo para el error genérico de respuesta inválida del CRUD. Si se incorpora un error específico de respuesta inválida, el fallback existente lo clasificaría como `502`, salvo que se decida expresamente una clasificación más específica.

## 15. Vacíos de cobertura identificados

### 15.1. Decodificación de listas

Archivo: `services/api_response_test.go`.

La prueba identificada `TestDecodeAPIListCapturaTotal` comprueba un sobre con un elemento y total global explícito:

```go
raw := json.RawMessage(`{"Data":[{"Id":1}],"Total":7}`)
resultado, err := decodeAPIListWithTotal[map[string]any](raw)
```

Comprueba el caso de un `Data` con un elemento y `Total` igual a 7.

No se identificaron pruebas específicas para:

- `Data: []`.
- `Data: [{}]`.
- `Data` ausente o `null`.
- Elementos con `Id: 0`.
- Ausencia del campo `Total` bajo paginación.

### 15.2. Servicio `ListarProveedores`

En `services/proveedor_test.go` se identificaron pruebas de:

- Validación de filtros antes de consultar el CRUD.
- Traslado de filtros y paginación.
- Total explícito distinto del tamaño de la página.
- Lectura individual.
- Expansión de tercero y detalle.
- Variantes natural/jurídico.
- Selección de campos.

No se identificaron pruebas de listado para:

- Respuesta vacía válida.
- Registro sin `Id`.
- `Id` igual a cero.
- Ausencia de `Total` con página limitada.
- Respuesta malformada `[{}]`.

### 15.3. Controlador del listado

`TestProveedorHandlerConsultarLista` cubre respuesta exitosa, parámetros y métodos HTTP no permitidos. No cubre errores devueltos por `ListarProveedores`.

El stub ya dispone del campo `err`, por lo que agregar escenarios de error no exige cambiar su estructura.

## 16. Causas raíz, impactos y propuestas

| ID | Hallazgo | Causa raíz / explicación | Impacto | Propuesta |
|---|---|---|---|---|
| E2E-01 | Página fuera de rango devuelve un proveedor con campos cero | CRUD devuelve `[{}]` cuando no hay filas; MID deserializa `{}` en una estructura con valores cero | El cliente recibe un registro falso en vez de una lista vacía | CRUD debe devolver `Data: []`; MID debe validar `Id > 0` |
| E2E-02 | `MetaData.Total` es igual al tamaño de página | CRUD no entrega total global; MID utiliza `len(Data)` como fallback | La paginación informa un total incorrecto | CRUD debe calcular y entregar el total global aplicando filtros antes de paginar |
| E2E-03 | Falta prueba de error en listado | `TestProveedorHandlerConsultarLista` solo prueba éxito y métodos no permitidos | No hay regresión automatizada para el mapeo HTTP de errores del listado | Agregar casos con `stub.err` y verificar HTTP 502/504 según el tipo |
| ROB-01 | `EsErrorFatalExpansion` no reconoce error fatal envuelto | Usa aserción directa de tipo en lugar de `errors.As` | Posible clasificación incorrecta si el error se envuelve en el futuro | Evaluar uso de `errors.As` y agregar prueba de error envuelto |
| ROB-02 | Decodificador acepta elementos vacíos | No valida la identidad de cada elemento de `Data` | Respuestas malformadas pueden convertirse en objetos con valores cero | Validar registros y devolver error de respuesta inválida |

## 17. Propuesta de pruebas de regresión

Las siguientes pruebas se proponen; no se han implementado.

### 17.1. CRUD: lista vacía

Preparar una consulta cuyo `offset` supere los registros disponibles.

Respuesta esperada:

```json
{
  "Data": [],
  "Message": "Request successful",
  "Status": "200",
  "Success": true
}
```

Verificar que el JSON contiene un arreglo vacío, no `null` ni `[{}]`.

### 17.2. CRUD: total global con paginación

Con tres registros que cumplen los filtros:

- `limit=1`, `offset=0` → un registro y `Total=3`.
- `limit=1`, `offset=1` → un registro y `Total=3`.
- `limit=1`, `offset=2` → un registro y `Total=3`.
- `limit=1`, `offset=3` → lista vacía y `Total=3`.

El total debe conservarse al cambiar `limit` y `offset`, siempre que los filtros sean iguales.

### 17.3. MID: rechazo de registro malformado

Simular:

```json
{
  "Data": [{}]
}
```

Resultado esperado: error de respuesta inválida del CRUD. No se debe producir un proveedor con `Id: 0`.

### 17.4. MID: lista vacía válida

Simular:

```json
{
  "Data": [],
  "Total": 3
}
```

Resultado esperado: lista sin elementos y total global 3. Esta combinación puede ocurrir en una página fuera de rango cuando existen tres resultados globales.

### 17.5. MID: total explícito

Simular una página de un elemento con total global tres:

```json
{
  "Data": [{"Id": 2}],
  "Total": 3
}
```

Resultado esperado: un elemento en `Data` y `MetaData.Total=3`.

### 17.6. Controlador: error de dependencia del listado

Configurar `proveedorCollectionStub.err` con un error genérico. Verificar HTTP `502` y que el cuerpo no exponga el mensaje interno.

### 17.7. Controlador: timeout del listado

Simular `context.DeadlineExceeded` o un error de red con `Timeout() == true`. Verificar HTTP `504`.

### 17.8. Expansión: error fatal envuelto

Si se decide endurecer `EsErrorFatalExpansion`, crear una prueba que envuelva el error fatal con `%w` y compruebe que `errors.As` permite reconocerlo.

## 18. Estado de implementación

No se realizaron modificaciones de código de producción, CRUD, configuración ni pruebas durante esta etapa de análisis.

| Elemento | Estado |
|---|---|
| Ampliación de datos a tres proveedores | Realizada |
| Filtros simples | Probados correctamente en los casos documentados |
| Filtro `contains` | Probado correctamente en el caso documentado |
| Filtros combinados | Probados correctamente en el caso documentado |
| Paginación de registros existentes | Probada correctamente |
| Página fuera de rango | Defecto reproducido |
| Respuesta directa del CRUD sin filas | Defecto reproducido |
| Causa de `[{}]` en controlador CRUD | Confirmada por inspección |
| Materialización de `{}` como proveedor cero en MID | Confirmada por inspección |
| Total global ausente en CRUD | Confirmado en respuesta inspeccionada |
| Fallback `len(Data)` del MID | Confirmado por inspección |
| Pruebas HTTP de error del listado | Pendientes |
| Corrección del CRUD | Pendiente |
| Validación de `Id` en MID | Pendiente |
| Total global en CRUD | Pendiente |
| Pruebas de regresión | Pendientes |
| Repetición E2E después de correcciones | Pendiente |

## 19. Conclusión general

La ampliación de la validación permitió comprobar filtros por identificador, tercero, tipo de registro, búsqueda textual con `contains` y combinación de criterios. También permitió verificar la paginación de los tres proveedores de prueba.

La prueba con `limit=1` y `offset=3` reveló un defecto reproducible: el endpoint responde HTTP `200`, pero entrega un objeto de proveedor con todos sus campos en valores cero. La consulta directa al CRUD demostró que la respuesta de origen contiene `Data: [{}]`. La inspección del controlador CRUD confirmó que agrega un mapa vacío cuando el modelo devuelve una colección `nil`.

Se identificó además un problema independiente en el contrato de conteo: el CRUD no entrega el total global y el MID utiliza `len(Data)` como alternativa. En consecuencia, con páginas de un elemento, `MetaData.Total` informa 1 aunque existan tres registros coincidentes.

La inspección del servicio MID confirmó que el elemento vacío se deserializa en `ProveedorRecord` con valores cero y después se convierte en un mapa sin validar `Id`. Por ello, la solución requiere protección en ambos extremos: el CRUD debe representar correctamente una colección vacía y el MID debe rechazar registros malformados.

La revisión de la expansión de relaciones permitió confirmar que `notify` y `fail` cuentan con pruebas unitarias a nivel del helper. El controlador del listado ya dispone de un manejo genérico HTTP `502`; el vacío principal identificado es la falta de pruebas de errores en `TestProveedorHandlerConsultarLista`.

En consecuencia, el endpoint tiene cobertura funcional satisfactoria para los filtros y páginas con registros existentes, pero la validación E2E integral permanece pendiente hasta corregir la respuesta vacía y el contrato de total global, agregar pruebas de regresión y repetir las pruebas contra los servicios reales.

---

## Anexo A. Inventario de comandos y consultas de inspección registrados

Los siguientes comandos se ejecutaron durante la investigación de código. Se incluyen para facilitar la reproducción de la inspección:

```bash
grep -nA100 'func TestProveedorHandlerConsultarLista' controllers/proveedor_test.go
grep -nA35 -B5 'type proveedorCollectionStub' controllers/proveedor_test.go
grep -nA55 -B5 'func (h .*responderErrorLista' controllers/proveedor.go
grep -RniA35 -B5 'func EsErrorFatalExpansion' helper
grep -RniE 'errorFatalExpansion|EsErrorFatalExpansion' --include='*.go' .
sed -n '550,590p' helper/expand.go
sed -n '315,360p' helper/expand_test.go
```

Comando para verificar el estado de la rama antes de iniciar la elaboración del segundo informe:

```bash
git status --short --branch
```

Resultado observado:

```text
## feature/49-add-collection-read...origin/feature/49-add-collection-read
```

El árbol de trabajo estaba limpio y la rama sincronizada con su remoto antes de comenzar la elaboración del segundo informe.

## Anexo B. Referencias de archivos inspeccionados

**MID (`agora_mid_issue49`):**

- `config/relaciones_proveedor.yaml`
- `services/proveedor.go`
- `services/api_response.go`
- `services/api_response_test.go`
- `services/proveedor_test.go`
- `controllers/proveedor.go`
- `controllers/proveedor_test.go`
- `helper/expand.go`
- `helper/expand_test.go`

**CRUD de gestión de personas:**

- `models/proveedor.go`
- `controllers/proveedor.go`

**Documento previo:**

- `docs/avance-validacion-e2e-proveedor.md`

## Anexo C. Clasificación de certeza

Para evitar confundir resultados observados con recomendaciones, se utiliza esta clasificación:

| Clasificación | Significado |
|---|---|
| **Reproducido** | Resultado observado directamente en una petición o prueba |
| **Confirmado por código** | El código inspeccionado explica el comportamiento observado |
| **Propuesta** | Alternativa técnica aún no implementada ni validada |
| **Pendiente** | Actividad que requiere ejecución o evidencia adicional |
| **Observación de robustez** | Riesgo potencial no reproducido como causa del defecto E2E |

Esta distinción debe conservarse al actualizar la Issue #49 y al preparar posteriores avances.
