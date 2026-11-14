#!/usr/bin/env bash
# CRUD 8081: listado de proveedor (limit 5)
# Uso: ./scripts/crud_proveedor_list.sh
set -u
DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$DIR/.."
OUT_DIR="${OUT_DIR:-$ROOT/responses/current}"
mkdir -p "$OUT_DIR"
NOMBRE="crud_proveedor_list"
curl -s -m 15 -X GET "http://localhost:8081/v1/proveedor/?limit=5" -o "$OUT_DIR/$NOMBRE.json" -w '%{http_code}\n' > "$OUT_DIR/$NOMBRE.status"
echo "$NOMBRE HTTP $(cat "$OUT_DIR/$NOMBRE.status")"
echo "cuerpo: $OUT_DIR/$NOMBRE.json"
