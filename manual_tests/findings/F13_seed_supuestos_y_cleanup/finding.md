# F13: supuestos de los datos dummy y cleanup sin probar

Estado: cerrado_sin_solucion. Solo afecta al entorno de pruebas.
Solución: aplicada_sin_commit. Ver `## Propuesta`.
Gravedad: baja. Clasificación propuesta, ver utils/README.md.

## Explicación
`seed/dummy_data.sql` usa `tipo_proveedor` 1 (natural) y 2 (jurídica), y catálogos sin verificar (`tipo_canal_contacto_id`, `tipo_cuenta_id`, `entidad_bancaria_id`, `pais_nacimiento_id`, `procedencia_empresa_id`, `tipo_conformacion_id`). Los catálogos viven en otros CRUD y no hay FK en la base. `seed/cleanup_dummy_data.sql` no se ha probado.

## Propuesta
Probar `cleanup_dummy_data.sql` en la base de pruebas antes de volver a cargar el seed. Verificar los catálogos contra sus CRUD.

## Verificación del seed y del cleanup (17/11/2026, 20:3x)

Base de pruebas `agora_pruebas_manual`, copia del esquema y tablas vaciadas.

- Seed sobre tablas vacías: carga correcta (proveedor 2, natural 1, jurídico 1, contacto 1, cuenta 1).
- Cleanup sin filas ajenas: borra todo, sin error (verificado el 17/11 a las 19:1x).
- Cleanup con una fila ajena `contacto` (id 3) que depende del proveedor 1: falla con `violates foreign key constraint "fk_contacto_proveedor"`, exit 3. La transacción no deja borrados parciales: los conteos quedan igual que antes del cleanup (proveedor 2, contacto 2).

Catálogos del seed, con GET de solo lectura a los servicios de parámetros y ubicaciones:
- `parametro/1` existe, pero es "Salario minimo mensual legal vigente" (TipoParametro 1). El seed lo usa como `tipo_canal_contacto_id`, que debería ser un canal de contacto. El id existe, pero el significado no coincide.
- `parametro/7` existe, pero es "Inscripción posgrado" (TipoParametro 2). El seed lo usa como `entidad_bancaria_id`. El id existe, pero el significado no coincide.
- `lugar/1` responde 200 (es el país de nacimiento usado en el seed).

Conclusión: el cleanup falla si hay filas ajenas que dependen de los ids 1 y 2. Los catálogos del seed existen, pero algunos ids no corresponden al tipo de catálogo que el seed supone. El seed sigue con supuestos incorrectos.

## Cambio aplicado

Corrección rápida sin commit (17/11/2026, 20:5x), a pedido del usuario.

**Archivos modificados:**

- `seed/cleanup_dummy_data.sql`: borra en orden de dependencia las filas hijas de los proveedores 1 y 2 (cuenta bancaria, contacto, actividad económica, documento, estado, perfil, representación, título, tipo de entidad jurídica y las ramas natural y jurídica), y luego los proveedores.
- `seed/dummy_data.sql`: `tipo_cuenta_id` 7939 ("Cuenta de Ahorros") y `entidad_bancaria_id` 7401 ("Banco Agrario de Colombia"), verificados en el servicio de parámetros. El encabezado distingue qué catálogos están verificados.
- `seed/verificar_seed.sh` (nuevo): vacía una base de pruebas, carga el seed, agrega filas ajenas que dependen de los proveedores dummy, ejecuta el cleanup y comprueba que no queda ninguna fila. Solo acepta bases cuyo nombre empiece por `agora_pruebas`.

**Verificación:** `bash manual_tests/seed/verificar_seed.sh` en `agora_pruebas_manual` termina con `OK`. Antes del cambio, la misma prueba fallaba con `FALLO: cleanup_dummy_data.sql` (violación de `fk_contacto_proveedor`). Se ejecutó dos veces con el mismo resultado.

**Pendiente, no cerrado por este cambio:** `tipo_canal_contacto_id` 1, `procedencia_empresa_id` 1 y `tipo_conformacion_id` 1 siguen sin verificar. El id 1 de parámetros es "Salario minimo mensual legal vigente", y no se encontró el catálogo de canal, procedencia ni conformación en el servicio de parámetros. Hace falta que el dueño de los catálogos indique los ids correctos.

## Búsqueda de catálogos sin verificar (17/11/2026, 20:4x)

- `utils/generador_configuracion/entradas/agora_modelo_extendido.dbml` declara `contacto.tipo_canal_contacto_id`, `proveedor_juridico.procedencia_empresa_id` y `proveedor_juridico.tipo_conformacion_id` como FK a `parametros.parametro.id`. Las tres columnas apuntan al mismo servicio, pero el DBML no indica a qué tipo de parámetro pertenece cada una.
- En el servicio de parámetros (202 tipos de parámetro, 6639 parámetros) no hay un tipo de parámetro con nombre de canal de contacto, procedencia de empresa ni conformación. Se revisaron nombres con "canal", "contacto", "notific", "procedencia", "conformación", "origen", "medio", "correo" y "teléfono".
- El id 1 de parámetros existe (`GET /v1/parametro/1` responde 200), así que el FK del seed se cumple. Su nombre es "Salario minimo mensual legal vigente", que no corresponde a ninguno de los tres catálogos.

Conclusión: no hay forma de verificar el significado de estos tres ids con lo que hay en el repositorio ni en el servicio de parámetros. Hace falta que el dueño de estos catálogos indique el id correcto de cada uno. Hasta entonces, el seed sigue con los ids 1.

## Decisión
Cerrado sin solución por el usuario (17/11/2026, 20:4x). Lo que queda abierto es un asunto de dominio, no de código ni de scripts: los ids de `tipo_canal_contacto_id`, `procedencia_empresa_id` y `tipo_conformacion_id` en `seed/dummy_data.sql` no se pueden verificar con el repositorio ni con el servicio de parámetros. Durante la integración real no es un problema, porque los datos los dan los catálogos reales. El seed sigue usando los ids 1 para esos tres campos, y se mantiene solo como datos dummy de prueba. Lo que sí está verificado: la carga del seed y el cleanup (`seed/verificar_seed.sh` termina en `OK` en una base de pruebas).
