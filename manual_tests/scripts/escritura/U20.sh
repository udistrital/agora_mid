#!/usr/bin/env bash
# U20: PATCH /v1/proveedor. HTTP esperado: 200. Cuerpo: requests/escritura/u20_mezcla_tablas.json. Mezcla proveedor, proveedor_natural, contacto y cuenta_bancaria en un PATCH
set -u
DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$DIR/../.."
OUT_DIR="${OUT_DIR:-$ROOT/responses/escritura/current}"
mkdir -p "$OUT_DIR"
NOMBRE="U20"
curl -s -m 30 -X PATCH "http://localhost:8080/v1/proveedor" -H 'Content-Type: application/json' --data "@$ROOT/requests/escritura/u20_mezcla_tablas.json" -o "$OUT_DIR/$NOMBRE.json" -w '%{http_code}\n' > "$OUT_DIR/$NOMBRE.status"
echo "$NOMBRE $(cat "$OUT_DIR/$NOMBRE.status")"
