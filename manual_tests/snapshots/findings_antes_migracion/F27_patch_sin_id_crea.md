# F27: un PATCH sin Id crea un proveedor nuevo

Estado: abierto. Reproducido en la corrida 4: U16 responde 200.

## Explicación
`PATCH /v1/proveedor` sin `Id` en la raíz crea un proveedor. El contrato no define si debe crear o responder 400. La especificación ("PATCH crea lo ausente") cubre el caso de clave natural, pero no un PATCH sin identificador de recurso.

## Propuesta
Decidir con el dueño del contrato. La propuesta del MID: responder 400 si el PATCH no trae `Id` ni clave natural.
