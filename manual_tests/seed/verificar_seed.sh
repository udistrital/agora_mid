#!/usr/bin/env bash
# Verifica dummy_data.sql y cleanup_dummy_data.sql en una base de pruebas.
# Uso: bash manual_tests/seed/verificar_seed.sh [base]. Por omisión: agora_pruebas_manual.
# Solo acepta bases cuyo nombre empiece por agora_pruebas. Vacía las tablas de proveedores de esa base.
set -u
DB="${1:-agora_pruebas_manual}"
case "$DB" in
  agora_pruebas*) ;;
  *) echo "Rechazado: $DB no es una base de pruebas (debe empezar por agora_pruebas)."; exit 2 ;;
esac
DIR="$(cd "$(dirname "$0")" && pwd)"
psqlq() { docker exec -i agora_db psql -U postgres -d "$DB" -v ON_ERROR_STOP=1 -q "$@"; }
consulta() { docker exec agora_db psql -U postgres -d "$DB" -At -c "$1"; }
conteos() {
  local out=""
  for t in proveedor proveedor_natural proveedor_juridico contacto cuenta_bancaria; do
    out="$out $t=$(consulta "select count(*) from proveedores.$t;")"
  done
  echo "$out"
}
fallo() { echo "FALLO: $1"; exit 1; }

psqlq -c "TRUNCATE proveedores.cuenta_bancaria, proveedores.contacto, proveedores.proveedor_juridico, proveedores.proveedor_natural, proveedores.proveedor RESTART IDENTITY CASCADE;" 2>/dev/null || fallo "vaciado previo"
echo "vacía:$(conteos)"

psqlq -f - < "$DIR/dummy_data.sql" >/dev/null || fallo "carga de dummy_data.sql"
[ "$(conteos)" = " proveedor=2 proveedor_natural=1 proveedor_juridico=1 contacto=1 cuenta_bancaria=1" ] || fallo "conteos despues de la carga:$(conteos)"
echo "seed cargado:$(conteos)"

# Filas ajenas que dependen de los proveedores dummy. El cleanup debe borrarlas junto con su padre.
psqlq -c "INSERT INTO proveedores.contacto (id, nombre_contacto, valor_especifico, proveedor_id, tipo_canal_contacto_id, proposito, activo, fecha_creacion, fecha_modificacion) VALUES (3, 'Ajena', 'ajena@example.com', 1, 1, 'Notificaciones', true, now(), now());" || fallo "insercion de contacto ajeno"
psqlq -c "INSERT INTO proveedores.cuenta_bancaria (id, tipo_cuenta_id, numero_cuenta, entidad_bancaria_id, proveedor_id, activo, fecha_creacion, fecha_modificacion) VALUES (2, 1, '999', 7, 2, true, now(), now());" || fallo "insercion de cuenta ajena"
echo "con filas ajenas:$(conteos)"

psqlq -f - < "$DIR/cleanup_dummy_data.sql" >/dev/null || fallo "cleanup_dummy_data.sql (exit distinto de 0)"
[ "$(conteos)" = " proveedor=0 proveedor_natural=0 proveedor_juridico=0 contacto=0 cuenta_bancaria=0" ] || fallo "tras el cleanup quedan filas:$(conteos)"
echo "cleanup OK:$(conteos)"
echo "OK: seed y cleanup verificados en $DB"
