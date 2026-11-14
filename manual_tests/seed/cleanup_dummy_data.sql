-- Limpia los datos dummy cargados por dummy_data.sql. Solo borra los ids fijos
-- 1 y 2 de proveedores (y sus hijos), no toca otras filas.
--
-- Uso:
--   docker exec -i agora_db psql -U postgres -v ON_ERROR_STOP=1 -f - < manual_tests/seed/cleanup_dummy_data.sql
--
-- Advertencia: si ya existen registros reales con esos ids, este script los borra.
-- Ejecutar solo sobre la base de pruebas acordada.

BEGIN;

DELETE FROM proveedores.cuenta_bancaria WHERE id IN (1) AND proveedor_id IN (1, 2);
DELETE FROM proveedores.contacto        WHERE id IN (1) AND proveedor_id IN (1, 2);
DELETE FROM proveedores.proveedor_juridico WHERE id = 2;
DELETE FROM proveedores.proveedor_natural  WHERE id = 1;
DELETE FROM proveedores.proveedor          WHERE id IN (1, 2);

COMMIT;
