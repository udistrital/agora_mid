# Tasks

## 1. Modelo de avisos y validación de configuración

- [ ] 1.1 Declarar la constante de aviso `KindFieldNormalization = "field_normalization"` junto a las existentes y verificar con pruebas unitarias que se distingue de `relation` y que respeta el invariante abierto "todo `Kind` emitido tiene una constante declarada", sin exponer datos remotos.
- [ ] 1.2 Extender la validación de configuración para identificar de forma determinista el campo derivado de `fk`, `shared_pk` y `fk_inbound`, y verificar rechazo de relaciones incoherentes con pruebas del cargador.

## 2. Motor de escritura y canonización

- [ ] 2.1 Ajustar la validación previa para reconocer campos derivados declarados por la relación, rechazar campos desconocidos y tratar `null` como ausencia; verificar que los cuerpos inválidos no materializan nada.
- [ ] 2.2 Implementar la comparación de campos derivados contra la llave autoritativa en sus dos puntos: descendente (padre->hijo) para `fk` y `shared_pk`, y ascendente (pre-pass, antes de fijar `local_field`) para `fk_inbound`; verificar con pruebas unitarias en `helper/write_engine_test.go` (patrón `resolverFalso`, en memoria, sin red ni `/test`) un caso por cada fila de la matriz y por cada modo: FK ausente (completa, sin aviso), `null` (completa, sin aviso), coincide (conserva, sin aviso) y diferencia.
- [ ] 2.3 Canonizar diferencias de entidades nuevas, registrar el aviso y preservar la llave numérica en el payload del resolver y en `Data`; verificar `206` y `MetaData.Errors`.
- [ ] 2.4 Rechazar conflictos derivados de entidades existentes (hijo con `Id` propio y llave divergente) antes de su PATCH, sin agregar un GET previo genérico ni permitir reasignación silenciosa; verificar con pruebas unitarias en `helper/write_engine_test.go` que no se envía la operación incompatible, cubriendo los tres modos e incluyendo el conflicto `fk_inbound` (nodo con `Id` cuyo `local_field` reenviado difiere del `Id` de la entidad referenciada -> `*ErrorEscritura`).
- [ ] 2.5 Conservar en el aplanado únicamente los campos derivados internos validados o generados, manteniendo la separación con `writable_fields`; verificar que una FK llega al resolver aunque no sea escribible por el cliente.

## 3. Contrato del endpoint y pruebas de integración

- [ ] 3.1 Actualizar el controlador y el contrato de respuesta para emitir `200` sin aviso cuando el campo coincide y `206` con el nuevo `Kind` cuando se canoniza; verificar el envoltorio y `MetaData.Errors`.
- [ ] 3.2 Agregar pruebas de integración y e2e (menos exigentes que las unitarias: verifican el observable extremo a extremo vía el router y el CRUD simulado) que reenvíen el `Data` completo de POST/PATCH, incluyendo FKs, PK compartidas, el campo derivado **ausente** y el campo derivado en `null` (ambos -> `200`) y el reenvío que **coincide** (-> `200` sin aviso); verificar reutilización idempotente y tipos numéricos.
- [ ] 3.3 Agregar pruebas de integración para diferencias en entidades nuevas (-> `206` + `MetaData.Errors` con `Kind: field_normalization`) y conflictos con entidades existentes (-> `400`); añadir un caso de integración de fallo de relación tolerado (`notify` -> `206` + `MetaData.Errors` con `Kind: relation`); verificar creación parcial, rechazo `400` y ausencia de reasignación. El caso `fail` -> `502/504` queda cubierto por la prueba unitaria del motor y por la prueba del controlador que mapea el error fatal al código HTTP, sin repetirlo aquí.
- [ ] 3.4 Actualizar `test/tests.md` con actores, cuerpos, estado de CRUD simulados, códigos de respuesta, avisos e interacciones, incluyendo una tabla de la matriz de comportamiento (ausente, `null`, coincide, diferencia en nuevo, diferencia en existente, fallo notify, fallo fail) que indique por fila qué nivel la cubre (unitaria en `helper/`, integración en `test/`, o prueba del controlador); verificar que cada prueba nueva esté documentada.

## 4. Documentación publicada y verificación

- [ ] 4.1 Actualizar `swagger/swagger.yml` (única fuente de verdad; `swagger/swagger.json` se regenera en el arranque y no se edita a mano) y `API.md` para documentar campos derivados reenviables, añadir `field_normalization` al tipo de aviso de `MetaData.Errors` y describir las respuestas `200`/`206`/`400` de escritura; verificar que el arranque regenera el JSON sin error y que los ejemplos coinciden con el comportamiento implementado.
- [ ] 4.2 Ejecutar `openspec validate --change normalizar-campos-relacionales-escritura --strict` y corregir cualquier inconsistencia de artefactos.
- [ ] 4.3 Ejecutar las pruebas focalizadas del motor, integración y e2e, después `go build ./...` y `go test ./...`; verificar que no existan regresiones.
