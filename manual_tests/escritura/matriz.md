# Matriz de pruebas de escritura (POST y PATCH) del proveedor

Versión vigente: 14/11/2026. Casos y resultados esperados. Se revisa antes de ejecutar cualquier escritura.

## Restricciones

- Solo se escriben objetos del esquema `proveedores`. Ningún cuerpo incluye el objeto `Tercero`, porque el middleware intentaría escribir en terceros y la guarda de F17 lo bloquearía. Es una escritura prohibida, así que no se prueba.
- `TerceroId` sí se envía como escalar, porque es una FK del proveedor. El middleware solo lee el tercero para validarla (GET).
- Parámetros y ubicaciones: no se escriben ni se referencian con objetos anidados.
- Antes de escribir se hace un respaldo del esquema `proveedores` (`../snapshots/`) para poder volver al estado inicial.

## Estado inicial esperado

- `proveedor` 1: natural, `TipoProveedor` 1, `TerceroId` 101, con `contacto` 1 y `cuenta_bancaria` 1.
- `proveedor` 2: jurídico, `TipoProveedor` 2, `TerceroId` 102, con `proveedor_juridico` 2.
- Secuencias en 2 (proveedor), 1 (contacto y cuenta), y las demás según el seed.

## Convenciones de la matriz

- Columna "HTTP esperado": código del envoltorio (`Status`). Para cada caso se confirma también el código de transporte.
- Columna "Efecto en BD": filas que deben cambiar. "ninguno" significa que no debe cambiar nada.
- "Por definir": el contrato no dice el resultado. El resultado observado se registra como hallazgo.
- Si un caso falla por algo ajeno al contrato, se registra en `findings/` y se intenta un arreglo rápido (ver la última columna y la sección "Arreglos rápidos").

## Regla de la raíz en PATCH

La tabla de la raíz de `proveedor-write` marca `TipoRegistroId` (en la configuración, `TipoProveedor`) como obligatorio. Un PATCH sin ese campo responde 400, aunque lleve `Id`. Por eso cada caso de PATCH de reconciliación incluye `TipoProveedor`, salvo U12 (null), U14 (valor incoherente) y U16 (sin Id), que prueban la validación de la raíz.

## Naming del contrato

El contrato de `API.md` difiere de la configuración. La configuración gana porque es lo que el motor lee. Se prueba con la configuración y se anota la diferencia como hallazgo (F18).

| Concepto | API.md | Configuración / CRUD |
|---|---|---|
| Tipo de registro | `TipoRegistroId` | `TipoProveedor` |
| Contacto canal | `TipoCanal` | `TipoCanalContactoId` |
| Contacto finalidad | `Finalidad` | `Proposito` |
| Contacto principal | `EsPrincipal` | no existe |
| Experiencia | `ExperienciaLaboralMeses` | `ExperienciaLaboral` |

## Matriz

### Creación (POST /v1/proveedor)

| ID | Cuerpo (archivo en `requests/escritura/`) | HTTP esperado | Efecto en BD esperado | Por qué |
|---|---|---|---|---|
| P01 | `p01_natural_minimo.json`: `TipoProveedor` 1 | 200 | +1 `proveedor` | Caso base: el root mínimo. |
| P02 | `p02_natural_descripcion_consentimiento.json` | 200 | +1 `proveedor` con esos valores | Campos escalares del root. |
| P03 | `p03_tercero_id_existente.json`: `TerceroId` 101 | 200 | Ningún `proveedor` nuevo. Reutiliza el proveedor 1 (`tercero_id` 101) y lo actualiza | `proveedor` declara `natural_key: [TerceroId]` (`config/gestion_personas_crud/objetos_proveedor.yaml`). Spec `declarative-write-engine`, requisito "Idempotencia por clave natural". |
| P04 | `p04_tercero_id_inexistente.json`: `TerceroId` 999999 | 400 | ninguno | Tercero que no existe en el CRUD de terceros: 400 en `TerceroId` con el mensaje "el TerceroId no existe en el CRUD de terceros". No escribe. |
| P05 | `p05_tercero_id_null.json`: `TerceroId` null | 200 | +1 `proveedor` con `tercero_id` NULL | Null en FK relacional se trata como ausente. |
| P06 | `p06_natural_con_proveedor_natural.json` | 200 | +1 `proveedor`, +1 `proveedor_natural` con el mismo Id | Rama de detalle válida para `TipoProveedor` 1. |
| P07 | `p07_juridico_con_proveedor_natural.json`: `TipoProveedor` 2 y rama natural | 400 | ninguno | Rama de detalle incoherente. |
| P08 | `p08_natural_con_proveedor_juridico.json`: `TipoProveedor` 1 y rama jurídica | 400 | ninguno | Rama incoherente (inversa). |
| P09 | `p09_natural_con_ambas_ramas.json` | 400 | ninguno | Mutuamente excluyentes. |
| P10 | `p10_contacto_valido.json`: una colección `Contacto` con campos not-null | 200 | +1 `proveedor`, +1 `contacto` | Colección con clave de FK propagada. |
| P11 | `p11_contacto_sin_proposito.json`: falta `Proposito` (not-null) | 400 o 502 | ninguno | Campo requerido no enviado. |
| P12 | `p12_cuenta_valida.json`: `CuentaBancaria` con `TipoCuentaId`, `NumeroCuenta`, `EntidadBancariaId` | 200 | +1 `proveedor`, +1 `cuenta_bancaria` | Colección de cuentas. |
| P13 | `p13_campo_no_escribible.json`: `Foo: 1` en el root | 400 | ninguno | Campo fuera de la tabla escribible. |
| P14 | `p14_auditoria_en_cuerpo.json`: `FechaCreacion` | 400 | ninguno | Campo administrado por el sistema. |
| P15 | `p15_cuerpo_array.json`: un arreglo JSON | 400 | ninguno | Cuerpo que no es objeto. |
| P16 | `p16_sin_tipo.json`: sin `TipoProveedor` | 400 | ninguno | Campo obligatorio ausente. |
| P17 | `p17_tipo_cero.json`: `TipoProveedor` 0 | 400 | ninguno | Valor no positivo. |
| P18 | `p18_tipo_null.json`: `TipoProveedor` null | 400 | ninguno | Null en campo no nulable. |
| P19 | `p19_grande_natural.json`: root, `ProveedorNatural`, 3 contactos, 2 cuentas | 200 | +1 `proveedor`, +1 `proveedor_natural`, +3 `contacto`, +2 `cuenta_bancaria` | Carga mayor en una sola escritura. |
| P20 | `p20_usuario_modificacion.json`: `UsuarioModificacionId` 9 en root y en un contacto | 200 | `usuario_modificacion_id` 9 en root y contacto | Propagación de autoría. |

### Reconciliación (PATCH /v1/proveedor)

| ID | Cuerpo | HTTP esperado | Efecto en BD esperado | Por qué |
|---|---|---|---|---|
| U01 | `u01_descripcion_proveedor1.json`: `Id` 1 y `DescripcionPortafolio` nuevo | 200 | `descripcion_portafolio` de 1 cambia | Cambio de un escalar. |
| U02 | `u02_mismo_tercero.json`: `Id` 1 y `TerceroId` 101 | 200 | sin cambio | Reenvío coincidente (la guía dice sin avisos). |
| U03 | `u03_tercero_distinto.json`: `Id` 1 y `TerceroId` 999 | 400 | ninguno | Reasignar el tercero de un proveedor existente. |
| U04 | `u04_tercero_null.json`: `Id` 1 y `TerceroId` null | 200 | `tercero_id` de 1 conservado | Null trata la FK como ausente. |
| U05 | `u05_contacto_actualiza.json`: `Id` 1 con `Contacto` `[{Id:1, ValorEspecifico:...}]` | 200 | `valor_especifico` de contacto 1 cambia | Actualización por Id. |
| U06 | `u06_contacto_otro_padre.json`: contacto con `Id` 1 bajo proveedor 2 | 400 | ninguno | No-reparent. |
| U07 | `u07_contacto_nuevo.json`: `Id` 1 con contacto sin Id | 200 | +1 `contacto` para proveedor 1 | Creación en colección. |
| U08 | `u08_reenvio_data.json`: `Id` 1 con el subárbol devuelto por un GET | 200 | sin cambio | Reenvío del Data materializado. |
| U09 | `u09_idempotente.json`: mismo cuerpo que U07, enviado dos veces | 200 ambas | +2 contactos (duplica) | La especificación (`proveedor-write`, "Idempotencia limitada de colecciones") declara que el reenvío de un cuerpo sin `Id` puede duplicar. Se prueba la limitación, no la idempotencia. |
| U10 | `u10_campo_omitido.json`: `Id` 1 sin `DescripcionPortafolio` | 200 | `descripcion_portafolio` conservado | Ausente conserva valor. |
| U11 | `u11_nullable_null.json`: `DescripcionPortafolio` null | 200 | `descripcion_portafolio` NULL | Null fija en campo nullable. |
| U12 | `u12_no_nulable_null.json`: `TipoProveedor` null | 400 | ninguno | Null en no nulable. |
| U13 | `u13_id_inexistente.json`: `Id` 999999 | 404 | ninguno | Proveedor que no existe. Regla de identidad PATCH (F28). |
| U14 | `u14_cambio_tipo_con_rama.json`: `Id` 1, `TipoProveedor` 2 con rama natural | 400 | ninguno | Rama incoherente en PATCH. |
| U15 | `u15_coleccion_vacia.json`: `Id` 1 con `Contacto` `[]` | 200 | ningún contacto eliminado | Colección vacía no borra. |
| U16 | `u16_sin_id_root.json`: sin `Id` en el root (PATCH) | 400 | ninguno | PATCH sin `Id` ni `TerceroId` no identifica el proveedor (F27). |
| U17 | `u17_grande.json`: `Id` 1 con `ProveedorNatural`, un contacto update, uno nuevo, y `CuentaBancaria` update | 200 | +1 contacto, +0 proveedor, actualizaciones aplicadas | Carga mayor en PATCH. |
| U18 | `u18_autoria.json`: `Id` 1 con `UsuarioModificacionId` 9 | 200 | `usuario_modificacion_id` 9 en root y en contactos | Propagación en PATCH. |
| U19 | `u19_proveedor2_juridico.json`: `Id` 2 con `ProveedorJuridico` y un campo | 200 | `proveedor_juridico` actualizado | PATCH de rama jurídica. |

## Arreglos rápidos

Un arreglo rápido solo se aplica si el caso bloquea la prueba. Se documenta en `findings/` con su causa, su estado y la lista de archivos modificados con su verificación.

## Criterio de éxito

- Cada caso con HTTP esperado conocido debe coincidir. Si no coincide, se registra como hallazgo con la evidencia.
- Ningún caso debe modificar filas fuera de `proveedores`. Se verifica con un conteo de tablas antes y después de la corrida.
- Se restaura el respaldo al final si la corrida deja datos no deseados.
