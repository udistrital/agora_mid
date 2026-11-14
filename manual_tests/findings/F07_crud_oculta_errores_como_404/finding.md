# F07: el CRUD convierte cualquier error de consulta en 404

Estado: abierto.
Solución: propuesta. Ver `## Propuesta`.
Gravedad: media. Clasificación propuesta, ver utils/README.md.

## Explicación
En `controllers/*.go` (por ejemplo `contacto.go`, `GetAll`), cualquier error del modelo responde 404 con "no record exists". Un error SQL se ve igual que una lista vacía o un Id inexistente. Esto oculta el motivo real en el MID.

## Propuesta
Responder 404 solo para `orm.ErrNoRows` y 500 para el resto. Mientras tanto, el error real queda en el log del CRUD (`beeLogger.Log.Error`).

## Reverificación (14/11, 15:5x)

- `GET /v1/proveedor/abc` y `GET /v1/contacto/abc` responden 404 con el mensaje "no record exists".
- `GET /v1/proveedor?query=Foo:bar` responde 500 con una página HTML de error de beego, no con JSON.

La afirmación de 404 para cualquier error es parcial: el parámetro no numérico da 404 y el filtro inválido da 500.
