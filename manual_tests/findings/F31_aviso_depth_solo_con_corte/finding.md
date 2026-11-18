# F31: el aviso depth nombra la relación padre en vez de la hija cortada

Estado: aceptado. Decisión aprobada por el usuario (17/11/2026, 19:40).
Solución: aplicada_sin_commit. Cambio de código en el árbol de trabajo. La especificación no está cambiada. Ver `## Notas de especificación`.
Gravedad: media. Clasificación propuesta, ver utils/README.md.

## Explicación
El aviso `depth` usaba la ruta de la relación que se expandía, no la de la hija que el límite cortó. En la primera versión, la respuesta de `POST /v1/proveedor/consulta {"id":1}` traía los paths `Contacto`, `CuentaBancaria` y `ProveedorNatural.PaisNacimiento`. Esas relaciones sí se expanden. Las hijas cortadas son `Contacto.TipoCanalContacto`, `CuentaBancaria.EntidadBancaria`, `CuentaBancaria.TipoCuenta` y `ProveedorNatural.PaisNacimiento.TipoLugarDetalle`.

## Decisión
Aprobada por el usuario (17/11/2026, 19:40). El `path` del aviso `depth` es la ruta completa de la hija cortada. Se emite un aviso por cada hija cortada.

## Cambio aplicado

**Archivos modificados:**

- `helper/expand.go`: el aviso se emite por cada hija cortada, con `path = ruta de la relación + "." + nombre de la hija`.
- `helper/expand_test.go`: `TestExpandAvisoDepthRutaDeLaHijaCortada`, escrita en rojo (fallaba con `Path: Tercero`) y luego en verde.

**Verificación:** `go build ./...` y `go test -count=1 ./...` pasan. En vivo, `POST /v1/proveedor/consulta {"id":1}` (19:3x) responde 206 con 4 avisos `depth`, en las rutas `Contacto.TipoCanalContacto`, `CuentaBancaria.EntidadBancaria`, `CuentaBancaria.TipoCuenta` y `ProveedorNatural.PaisNacimiento.TipoLugarDetalle`.

## Notas de especificación

Estado de la especificación: sin cambios. `openspec/specs/nested-object-expansion/spec.md` y `swagger/swagger.yml` están como en el repositorio. Por el procedimiento (3.10), el cambio se aprueba y se aplica por separado.

Texto propuesto para `openspec/specs/nested-object-expansion/spec.md`, en el requisito "Límites, ciclos y profundidad efectiva". Va después del escenario "Expansión recursiva hasta max_depth":

```
#### Scenario: Límite que corta hijos pedidos
- **WHEN** el límite de profundidad impide expandir una relación hija pedida por la selección (o configurada, si no hay selección) y la relación padre tiene registros
- **THEN** se agrega un aviso de tipo `depth` por cada hija cortada, con su `path` completo (por ejemplo `Contacto.TipoCanalContacto`)
```

Texto propuesto para `swagger/swagger.yml`, descripción de `kind`. Reemplaza "expansión: `relation`, `depth`, `cycle`, `generic_resolver`. Aviso de" por:

```
expansión: `relation`, `depth` (una relación hija pedida no se expande
por el límite de profundidad; `path` es la ruta de la hija), `cycle`,
`generic_resolver`. Aviso de
```

Si `swagger/swagger.yml` se genera desde código con Beego, el texto debe ir en la fuente generadora. Esto no se revisó.

Pendiente: confirmar en datos que las hijas cortadas tienen registros. Si no, el aviso sobra.
