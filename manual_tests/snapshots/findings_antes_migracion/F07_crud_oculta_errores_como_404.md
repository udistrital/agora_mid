# F07: el CRUD convierte cualquier error de consulta en 404

Estado: abierto.

## Explicación
En `controllers/*.go` (por ejemplo `contacto.go`, `GetAll`), cualquier error del modelo responde 404 con "no record exists". Un error SQL se ve igual que una lista vacía o un Id inexistente. Esto oculta el motivo real en el MID.

## Propuesta
Responder 404 solo para `orm.ErrNoRows` y 500 para el resto. Mientras tanto, el error real queda en el log del CRUD (`beeLogger.Log.Error`).
