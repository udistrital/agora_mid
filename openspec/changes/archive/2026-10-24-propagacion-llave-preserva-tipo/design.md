# Design

## Context

Ver proposal.md — Why. Estado actual relevante:

- `helper/write_engine.go` propaga la llave del padre en tres puntos:
  `materializarHijos` (`padreID := aCadena(v)`, variable `string`),
  `materializarValorRelacion` (pasa `padreID string` a `propagarLlave`),
  `propagarLlave` (asigna `hijo[rel.RelatedField]`/`hijo["Id"]`) y
  `materializarInbound` (`nodo[rel.LocalField] = aCadena(id)`).
- `aCadena` (`helper/expand.go`) se creó para el camino de **lectura**: normaliza
  una llave a `string` solo para **comparar** padres e hijos al reensamblar el
  árbol expandido (`filtrarPorRelatedField`). En lectura nunca muta el dato
  devuelto; la spec `nested-object-expansion` exige "conservar la llave local
  original".
- `services/write_resolver.go` es la frontera con el API: `upsertRespuesta.Id` se
  decodifica como `int` y el resolver fija `salida["Id"]` con ese entero. Es la
  fuente del tipo canónico del `Id`. `idEntero` ya tolera `int`, `float64` y
  `string`.
- El único consumidor aguas abajo de la FK/PK propagada es el resolver, vía
  `idEntero` (para resolver el upsert). Las pruebas actuales
  (`integration_test.go`, `e2e_test.go`, `write_engine_test.go`) comparan esas
  llaves con `comoTexto()`/`aCadena()`, que aceptan string y número por igual y
  por tanto enmascaran la coacción a string.

## Goals / Non-Goals

**Goals:**
- Propagar la llave del padre preservando su tipo numérico, en los tres puntos de
  propagación del motor de escritura.
- Que `aCadena` quede restringido a sus usos legítimos: comparación en lectura y
  construcción de la query de clave natural (texto).
- Que las pruebas afirmen el tipo numérico de las llaves propagadas, dejando de
  enmascararlo.
- Que `swagger/swagger.yml` (y su JSON derivado) describa fielmente los tipos de
  las FK y la forma real del `Data` materializado.

**Non-Goals:**
- No se cambia el contrato de envoltorio (`Success/Status/Message/Data`) ni los
  códigos de respuesta.
- No se cambia el camino de lectura ni el uso de `aCadena` en
  `filtrarPorRelatedField`.
- No se elimina `AutorModificacionId` del `Data` devuelto: se documenta como
  comportamiento real (es el autor que el propio cliente aporta por cabecera).

## Decisions

- **Propagar `any` en lugar de `string`**: `padreID` pasa de `string` a `any` y
  `propagarLlave` recibe `any`. Se asigna el valor de `padre["Id"]` tal cual (ya
  normalizado a `int` por el resolver). Alternativa descartada: normalizar los
  tipos Go a un canónico explícito en una estructura intermedia — innecesario,
  porque nos comunicamos por JSON y `int`/`float64` se serializan al mismo número;
  la frontera API (resolver) ya fija el tipo. La ruta simple evita complejidad sin
  perder corrección.
- **Guarda centinela `nil` en vez de `""`**: hoy `propagarLlave` usa `padreID == ""`
  como señal de "no hay padre que propagar". Con `any`, la señal equivalente es
  `padreID == nil`. `materializarHijos` inicializa `var padreID any` (nil) y solo
  lo asigna cuando `padre["Id"]` existe y no es nil.
- **`idEntero` se conserva tolerante a string**: tras el arreglo, el tipo
  propagado es siempre numérico, así que `idEntero` no recibirá strings en la
  resolución de upsert. Se mantiene la tolerancia como defensa en profundidad
  (barata y correcta), pero la línea de defensa principal pasa a ser el tipo
  preservado más las aserciones de tipo en las pruebas.
- **Pruebas afirman tipo numérico**: las aserciones de FK/PK dejan de usar
  `comoTexto()`/`aCadena()` y verifican que el valor decodificado es un número
  (`float64` tras `json.Unmarshal`), fallando ruidosamente si reaparece el string.
- **Swagger fiel al comportamiento real**: `ProveedorMaterializado` se redefine
  como el subárbol real (entidades anidadas con sus `Id`, FK numéricas y
  `AutorModificacionId`), en lugar de los tres identificadores actuales. Los
  ejemplos y tipos se derivan de los payloads capturados en las pruebas.

## Risks / Trade-offs

- [Cambio observable de tipo: FK de string a número en el `Data` devuelto] →
  documentar en swagger/`API.md` y en el proposal (Impact/Compatibilidad); el
  valor JSON es equivalente (`"201"` -> `201`). Un cliente que dependa del tipo
  string debe ajustarse.
- [Regresión silenciosa si vuelve la coacción a string] → mitigado al endurecer
  las pruebas para afirmar tipo numérico en lugar de tolerar ambos.
- [`materializarInbound` propaga antes de que el nodo exista] → se propaga el
  `Id` de la entidad referenciada ya materializada (`referida["Id"]`, entero del
  resolver); el tipo correcto está disponible en ese punto.
- [JSON derivado desincronizado del YAML] → la capacidad `openapi-publication`
  regenera el JSON desde el YAML al arranque; basta actualizar el YAML fuente.

## Migration Plan

1. Ajustar la propagación en `helper/write_engine.go` (quitar `aCadena`, `any`,
   guarda `nil`).
2. Endurecer las pruebas de motor, integración y e2e para afirmar tipo numérico.
3. Actualizar `swagger/swagger.yml` (tipos de FK y forma real de `Data`) y
   `test/tests.md`.
4. Verificar `go build ./...` y `go test ./...` en verde.
5. Rollback: revertir los cambios de `write_engine.go`; `idEntero` tolerante
   mantiene el comportamiento previo sin romper la resolución de upsert.
