# F32: el modelo del CRUD convierte un NULL en valor por defecto al actualizar

Estado: cerrado_sin_solucion.
Solución: propuesta. Ver `## Propuesta`.
Gravedad: critica. Clasificación propuesta, ver utils/README.md.

## Explicación
La causa principal está en el modelo de `gestion_personas_crud`. Sus campos usan tipos no nulables (`bool`, `float64`, `int`, `string`) con la etiqueta `null`. Al leer un NULL de la base, el modelo entrega el valor cero. Al actualizar, el CRUD escribe el valor cero de cada columna de su lista de actualización. Por eso un NULL no enviado se convierte en `false`, `0` o cadena vacía.

El MID, que procesa en una sola solicitud varias tablas (la raíz, `ProveedorNatural`, `Contacto`, `CuentaBancaria`), reescribe cada tabla incluida en el cuerpo con lo que devuelve el GET de esa tabla. Así que los NULL de esas tablas quedan en valor cero. El MID no distingue un NULL de un cero, porque el CRUD ya le entregó el cero.

Efecto observado en el caso U20 (PATCH que mezcla proveedor, proveedor_natural, contacto y cuenta_bancaria, `escritura/matriz.md`). La comprobación se hizo con `psql` (17/11/2026, 19:5x), comparando `proveedores` antes y después con `pg_dump --data-only`:

- `proveedor_natural` 1: 14 campos que estaban en NULL pasan a `false` o `0` (`medicina_prepagada`, `valor_uvt_prepagada`, `interes_vivienda_afc`, `dependiente_*`, `pensionado`, `valor_afc`, `responsable_iva`, `reside_colombia`, `pension_voluntaria`, `valor_pension_voluntaria`).
- `cuenta_bancaria` 1: `monto_capital_autorizado` pasa de NULL a `0`.
- `contacto` 1: `extension` pasa de NULL a cadena vacía (mismo patrón que U05 de F29).
- `usuario_modificacion_id` pasa a `0` en `proveedor`, `proveedor_natural`, `contacto` y `cuenta_bancaria`, aunque el cuerpo no lo envió.

Los campos enviados sí cambiaron, como se esperaba: `proveedor.descripcion_portafolio`, `proveedor_natural.cargo`, `contacto` 1 `valor_especifico` y `cuenta_bancaria` 1 `numero_cuenta`. Se creó un `contacto` nuevo. La matriz espera "sin cambio" en los demás campos.

El defecto aparece solo cuando el cuerpo incluye la subtabla. Afecta también a objetos que ya existían antes de la petición (`proveedor_natural` 1, `contacto` 1, `cuenta_bancaria` 1), no solo a altas nuevas.

Relación con F29: F29 registra U11 y U05. Su causa es la misma que aquí: un NULL que el CRUD devuelve como valor por defecto y el MID reenvía. Este hallazgo añade la escritura de `usuario_modificacion_id` en 0 y los casos de subtabla.

## Causa
Causa principal, en el CRUD: `gestion_personas_crud/models/*.go` (por ejemplo `proveedor_natural.go` línea 17: `MedicinaPrepagada bool`, con `orm:"column(medicina_prepagada);null"`). El modelo no puede representar NULL, así que el GET devuelve cero, y la actualización escribe ese cero.

Causa en el MID, que amplifica el efecto: `services/write_resolver.go`:

1. `Materialize` (líneas 95 a 110): si el payload trae `Id`, lee el registro completo con GET (`leerRegistroCompleto`), fusiona con `fusionarRegistro` y envía el resultado por PUT.
2. `fusionarRegistro` (líneas 348 a 358): el registro leído es la base, y el cuerpo recibido sobrescribe las claves que trae.

El MID no tiene forma de saber que un cero vino de un NULL, porque el CRUD ya lo convirtió.

## Propuesta
Corregir primero el modelo del CRUD, porque es la causa:

- Usar tipos anulables (`*bool`, `*float64`, `*int`, `*string`) en los modelos de `proveedor_natural`, `cuenta_bancaria`, `contacto` y `proveedor`, para que el GET devuelva `null` donde la base tiene NULL. Esto cambia el contrato de lectura del CRUD, así que hay que revisarlo con F06 y F07.

Mitigación en el MID, independiente de la anterior:

- No reenviar en el PUT los campos que el cuerpo no envió. Para eso, `fusionarRegistro` debe conocer qué claves vienen del cuerpo.

Además, un PATCH debe actualizar solo los campos presentes en el cuerpo. No debe escribirse `usuario_modificacion_id` con 0 cuando no hay actor identificado: debe quedar NULL o tomar el actor de la solicitud. F29 define qué significa "presente" en el contrato de null y ausente, así que conviene cerrarlo antes de implementar.

## Notas de especificación

Pendiente: definir en `proveedor-write` la regla de campo ausente en PATCH para las subtablas (`ProveedorNatural`, `CuentaBancaria`, `Contacto`). Hoy la especificación no dice qué pasa con los campos de subtabla no enviados. No se modifica la especificación desde estas pruebas.

## Verificación de preservación (17/11/2026, 19:5x)

Prueba con respaldo y restauración: `PATCH /v1/proveedor` con solo `Id`, `TipoProveedor` y `DescripcionPortafolio`, sin `ProveedorNatural` ni `Contacto`.

- `proveedor` 1: `descripcion_portafolio` cambió, como se pidió.
- `proveedor_natural` 1: `cargo` = "Consultora", `pais_nacimiento_id` = 1, `experiencia_laboral` = 24 y `experiencia_profesional` = 60 se conservaron. `medicina_prepagada` seguía NULL.
- `contacto` 1: `valor_especifico`, `extension` (NULL) y `proposito` se conservaron.

Conclusión: un PATCH que no incluye una subtabla no toca sus campos, ni los que son NULL. El cambio de NULL a cero ocurre cuando el cuerpo incluye la subtabla, como en U20.

Limitación: el MID hace GET y después PUT, sin bloqueo. Si otra escritura cambia el registro entre ambos pasos, el PUT sobrescribe ese cambio. Esto no se probó.

Respaldo de esta prueba: `snapshots/proveedores_antes_prueba_preservacion_17nov.sql`. Restaurado y verificado con conteos (proveedor 2, natural 1, jurídico 1, contacto 1, cuenta 1).

## Decisión
Cerrado sin solución por el usuario (17/11/2026, 20:4x). La causa principal está en el modelo del CRUD (`gestion_personas_crud`), fuera de este repositorio. El ajuste no debe hacerse en este repositorio. La mitigación propuesta en el MID tampoco se implementa aquí.
