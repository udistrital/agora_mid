-- Limpia los datos dummy cargados por dummy_data.sql.
--
-- Borra los proveedores con id 1 y 2 y todas las filas hijas que dependen de ellos,
-- en orden de dependencia. No toca proveedores con otros ids.
--
-- Uso:
--   docker exec -i agora_db psql -U postgres -v ON_ERROR_STOP=1 -f - < manual_tests/seed/cleanup_dummy_data.sql
--
-- Advertencia: borra cualquier fila con proveedor_id 1 o 2. Ejecutar solo sobre una base de
-- pruebas, o sobre la base de pruebas acordada. Si la base tiene datos reales con esos ids, se borran.
-- Con ON_ERROR_STOP, un error deshace la transacción completa.

BEGIN;

DELETE FROM proveedores.cuenta_bancaria          WHERE proveedor_id IN (1, 2);
DELETE FROM proveedores.contacto                 WHERE proveedor_id IN (1, 2);
DELETE FROM proveedores.actividad_economica_proveedor WHERE proveedor_id IN (1, 2);
DELETE FROM proveedores.documento                WHERE proveedor_id IN (1, 2);
DELETE FROM proveedores.estado_proveedor         WHERE proveedor_id IN (1, 2);
DELETE FROM proveedores.perfil_proveedor         WHERE proveedor_id IN (1, 2);
DELETE FROM proveedores.representacion           WHERE proveedor_id IN (1, 2);
DELETE FROM proveedores.titulo_proveedor_natural WHERE proveedor_id IN (1, 2);
DELETE FROM proveedores.persona_juridica_tipo_entidad WHERE persona_juridica_id IN (2);
DELETE FROM proveedores.proveedor_juridico       WHERE id = 2;
DELETE FROM proveedores.proveedor_natural        WHERE id = 1;
DELETE FROM proveedores.proveedor                WHERE id IN (1, 2);

COMMIT;
