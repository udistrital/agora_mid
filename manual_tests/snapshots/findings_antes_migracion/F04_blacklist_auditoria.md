# F04: la lista negra global no oculta los campos de auditoría

Estado: abierto. Única prueba roja de la suite: `test/lectura_proveedor_test.go`, `TestIntegracionConsultaSinFieldsNoEntregaAuditoria`.

## Explicación
Sin `fields`, `Tercero` y `ProveedorNatural` deberían omitir `FechaCreacion`, `FechaModificacion`, `Activo` y `UsuarioModificacionId`. Ahora aparecen, porque el bloque de la lista negra se quitó (F00, R1).

## Causa
- El HEAD de `config/global_blacklist.yaml` tiene `ANY/ANY/ANY`. `config/blacklist.go` lo rechaza, y por eso se quitó.
- Una entrada concreta (`gestion_personas_crud/ANY`) también fue rechazada: `FechaCreacion` no está en el modelo conocido por el validador (solo en `filterable_fields`, con `readable: false`).

## Propuesta
A. Restaurar una entrada concreta por recurso (por ejemplo `terceros: tercero: [FechaCreacion, FechaModificacion, Activo, UsuarioModificacionId]`) y ajustar el validador para aceptarla si el campo está en `allowed_fields` de alguna relación. Revisar primero el validador.
B. Cambiar el validador para consultar también los campos de los objetos generados. Requiere cambiar `openspec/specs/nested-object-expansion`.
