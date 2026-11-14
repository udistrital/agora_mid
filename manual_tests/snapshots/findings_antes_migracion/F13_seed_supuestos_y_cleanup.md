# F13: supuestos de los datos dummy y cleanup sin probar

Estado: abierto. Solo afecta al entorno de pruebas.

## Explicación
`seed/dummy_data.sql` usa `tipo_proveedor` 1 (natural) y 2 (jurídica), y catálogos sin verificar (`tipo_canal_contacto_id`, `tipo_cuenta_id`, `entidad_bancaria_id`, `pais_nacimiento_id`, `procedencia_empresa_id`, `tipo_conformacion_id`). Los catálogos viven en otros CRUD y no hay FK en la base. `seed/cleanup_dummy_data.sql` no se ha probado.

## Propuesta
Probar `cleanup_dummy_data.sql` en la base de pruebas antes de volver a cargar el seed. Verificar los catálogos contra sus CRUD.
