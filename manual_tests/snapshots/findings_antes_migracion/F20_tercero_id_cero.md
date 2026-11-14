# F20: un proveedor sin TerceroId se guarda con tercero_id = 0

Estado: abierto. Confirmado en la corrida 3 (14/11): los proveedores creados sin `TerceroId` quedan con `tercero_id = 0` (P01, P02, P05 y los demás casos sin `TerceroId`).

## Causa
`models/proveedor.go` del CRUD declara `TerceroId int`. Go envía 0 cuando el campo falta, y el ORM lo escribe. La columna es nullable, y 0 no existe en terceros: queda como FK huérfana.

## Propuesta
En el CRUD, declarar `TerceroId *int` o convertir 0 a NULL antes del insert. Alternativa en el MID: no enviar `TerceroId` cuando sea 0.
