# F04: la lista negra global no oculta los campos de auditoría

Estado: en_decision. Pendiente la aprobación del cambio de definición del modelo conocido en `openspec/specs/nested-object-expansion`.
Solución: aplicada_sin_commit. Ver `## Cambio aplicado`.
Gravedad: media. Clasificación propuesta, ver utils/README.md.

## Explicación
Sin `fields`, la expansión conserva los campos que entrega el resolver (`nested-object-expansion`, escenario "Sin fields no se recorta"). Por eso `Tercero` y `ProveedorNatural` entregan `FechaCreacion`, `FechaModificacion`, `Activo` y `UsuarioModificacionId`. La lista negra global debe ocultarlos.

## Causa
- `config/global_blacklist.yaml` declara los cuatro campos bajo `ANY/ANY`.
- `config/blacklist.go` (`ValidarBlacklist`) valida cada campo exacto contra el modelo conocido, que es la unión de `allowed_fields`, el Id y los `related_field` de las relaciones.
- Los cuatro campos están en los objetos del CRUD (`config/terceros/objetos_tercero.yaml` y `config/gestion_personas_crud/objetos_proveedor_natural.yaml`) con `readable: false`. `CamposLectura()` no los incluye, así que no entran en `allowed_fields`. `UsuarioModificacionId` no aparece en `objetos_tercero.yaml`.
- Por eso el modelo conocido no contiene los campos de auditoría. La lista negra no se puede declarar de forma válida sin cambiar esa definición.

## Propuesta
Que el modelo conocido incluya también los campos declarados en el catálogo de objetos de cada `related_table`, sean o no legibles. La validación de la lista negra se hace contra ese modelo. Con esa definición, la entrada `ANY/ANY` con los cuatro campos es válida y la supresión de auditoría queda en la lista negra.

## Cambio aplicado

Solución provisional, sin commit. Mientras no se apruebe la propuesta, `ValidarBlacklist` registra como advertencia un campo exacto que no esté en el modelo conocido, en lugar de abortar el arranque. Los componentes `related_api`, `related_table` y el comodín en los tres niveles siguen fallando.

**Archivos modificados:**

- `config/blacklist.go`: `ValidarBlacklist` registra `advertencia lista negra: el campo ...` con `log.Printf` cuando el campo no está en el modelo.
- `config/blacklist_test.go`: prueba `TestValidarBlacklistCampoFueraDelModeloAdvierte`. El caso "campo inexistente" queda fuera de `TestValidarBlacklistComponenteInexistente`.
- `config/global_blacklist.yaml`: bloque `global_blacklist.ANY.ANY` con `FechaCreacion`, `FechaModificacion`, `Activo` y `UsuarioModificacionId`.

**Verificación:** `go test -count=1 ./...` pasa en todos los paquetes, incluido `test` (`TestIntegracionConsultaSinFieldsNoEntregaAuditoria`). En vivo (14/11, 15:40, MID PID 956, `/tmp/agora_mid_run_f04`): el arranque registra cuatro advertencias, una por campo. `POST /v1/proveedor/consulta {"id":1}` no devuelve esos campos en `Tercero` ni en `ProveedorNatural`. `Contacto[0].ProveedorId` sigue en 1. Los avisos de la respuesta son 17 `depth` y 3 `relation`.

## Notas de especificación

Requisito a cambiar en `openspec/specs/nested-object-expansion`, "Validación al arranque de la lista negra global". Texto propuesto:

> El sistema SHALL validar al arranque la lista negra global. Un componente exacto (sin comodín) SHALL existir en el modelo conocido. El modelo conocido SHALL incluir `allowed_fields`, el Id, los `related_field` de las relaciones y los campos declarados en el catálogo de objetos de cada `related_table`, sean o no legibles.
>
> #### Scenario: Componente exacto inexistente aborta el arranque
> - **WHEN** una entrada nombra un `related_api`, `related_table` o campo exacto que no existe en el modelo conocido
> - **THEN** el API no inicia y reporta un mensaje que identifica el archivo y la entrada inválida

El código actual contradice este escenario en el caso de campo, por la solución provisional descrita arriba. La solución provisional se retira cuando se apruebe el texto.

Riesgo pendiente: el escenario "Campo vetado se suprime pese a allowed_fields" solo cubre campos legibles. Texto propuesto para cubrir los no legibles:

> #### Scenario: Campo no legible vetado se suprime
> - **WHEN** un campo está declarado con `readable: false` en el catálogo de objetos de una `related_table` y figura en la lista negra para su `related_api` y `related_table`
> - **THEN** el campo se omite de la entidad expandida
