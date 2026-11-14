# F23: PATCH con un TerceroId distinto responde 200 y no cambia nada

Estado: abierto. Reproducido en la corrida 4: U03 responde 200.

## Explicación
`PATCH /v1/proveedor` con `Id: 1` y `TerceroId: 999` responde 200 "registrado correctamente", con `Data.Id = 1`. La especificación (`proveedor-write`, conflicto de FK ascendente en proveedor existente) pide 400 y no escribir nada. El 200 oculta que no se escribió nada.

## Causa
La validación de no-reasignación (`helper/write_engine.go`, `hayConflictoLlave`/`reasociaExistente`) se aplica a hijos (`!esRaiz`). La FK ascendente de la raíz sigue la regla `fk_inbound`, y la comparación no dispara el 400.

## Propuesta
En la raíz, comparar el `TerceroId` del cuerpo con el del registro existente y responder 400 si difieren.
