-- Datos dummy para pruebas manuales de agora_mid (esquema proveedores).
--
-- Uso:
--   docker exec -i agora_db psql -U postgres -v ON_ERROR_STOP=1 -f - < manual_tests/seed/dummy_data.sql
--
-- Seguridad: el script aborta si cualquier tabla de proveedores ya contiene filas,
-- para no duplicar datos ni mezclar con registros reales.
--
-- Supuestos (no verificados contra catálogos reales, porque los catálogos viven en
-- otros CRUD y no hay FK en esta base):
--   proveedor.tipo_proveedor = 1  -> persona natural
--   proveedor.tipo_proveedor = 2  -> persona jurídica
--   tipo_canal_contacto_id = 1, tipo_cuenta_id = 1, entidad_bancaria_id = 7,
--   pais_nacimiento_id = 1, procedencia_empresa_id = 1, tipo_conformacion_id = 1.
-- proveedores.proveedor.tercero_id es un entero sin FK en esta base; los terceros
-- dummy 101 y 102 no existen en el CRUD de terceros (no se consultan).
-- cuenta_bancaria.tipo_cuenta_bancaria_id ya no existe en el esquema actual.

BEGIN;

DO $$
DECLARE
    filas bigint;
BEGIN
    SELECT
        (SELECT count(*) FROM proveedores.proveedor)
      + (SELECT count(*) FROM proveedores.proveedor_natural)
      + (SELECT count(*) FROM proveedores.proveedor_juridico)
      + (SELECT count(*) FROM proveedores.contacto)
      + (SELECT count(*) FROM proveedores.cuenta_bancaria)
    INTO filas;
    IF filas > 0 THEN
        RAISE EXCEPTION 'Abortado: ya existen % filas en proveedores; no se inserta', filas;
    END IF;
END
$$;

-- Proveedor 1: persona natural con contacto y cuenta bancaria.
INSERT INTO proveedores.proveedor
    (id, tipo_proveedor, descripcion_portafolio, consentimiento, tercero_id, activo, usuario_modificacion_id)
VALUES
    (1, 1, 'Servicios de consultoria', true, 101, true, NULL);

INSERT INTO proveedores.proveedor_natural
    (id, pais_nacimiento_id, cargo, experiencia_laboral, experiencia_profesional, activo)
VALUES
    (1, 1, 'Consultora', 24, 60, true);

INSERT INTO proveedores.contacto
    (id, nombre_contacto, valor_especifico, proveedor_id, tipo_canal_contacto_id, proposito, extension, activo)
VALUES
    (1, 'Ana Perez', 'ana.perez@example.com', 1, 1, 'Notificaciones', NULL, true);

INSERT INTO proveedores.cuenta_bancaria
    (id, tipo_cuenta_id, numero_cuenta, entidad_bancaria_id, contacto_id,
     monto_capital_autorizado, proveedor_id, activo)
VALUES
    (1, 1, '123456789', 7, 1, NULL, 1, true);

-- Proveedor 2: persona jurídica (PK compartida con proveedor_juridico).
INSERT INTO proveedores.proveedor
    (id, tipo_proveedor, descripcion_portafolio, consentimiento, tercero_id, activo, usuario_modificacion_id)
VALUES
    (2, 2, 'Suministro de papeleria', false, 102, true, NULL);

INSERT INTO proveedores.proveedor_juridico
    (id, procedencia_empresa_id, tipo_conformacion_id, codigo_postal,
     exclusividad_producto, regimen_contributivo, pyme, registro_mercantil,
     sujeto_retencion, agente_retenedor, responsable_ica, responsable_iva, activo)
VALUES
    (2, 1, 1, '110111', false, 1, true, true, true, false, true, true, true);

-- Ajusta las secuencias para que los próximos INSERT sin id no choquen con los ids fijos.
SELECT setval('proveedores.proveedor_id_seq', (SELECT max(id) FROM proveedores.proveedor));
SELECT setval('proveedores.proveedor_natural_id_seq', (SELECT max(id) FROM proveedores.proveedor_natural));
SELECT setval('proveedores.proveedor_juridico_id_seq', (SELECT max(id) FROM proveedores.proveedor_juridico));
SELECT setval('proveedores.contacto_id_seq', (SELECT max(id) FROM proveedores.contacto));
SELECT setval('proveedores.cuenta_bancaria_id_seq', (SELECT max(id) FROM proveedores.cuenta_bancaria));

COMMIT;
