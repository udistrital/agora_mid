#!/usr/bin/env bash
# CRUD 8081: listado de contacto
# Uso: ./scripts/crud_contacto_list.sh
set -u
DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$DIR/.."
OUT_DIR="${OUT_DIR:-$ROOT/responses/current}"
mkdir -p "$OUT_DIR"
NOMBRE="crud_contacto_list"
curl -s -m 15 -X GET "http://localhost:8081/v1/contacto/?limit=5" -o "$OUT_DIR/$NOMBRE.json" -w '%{http_code}\n' > "$OUT_DIR/$NOMBRE.status"
echo "$NOMBRE HTTP $(cat "$OUT_DIR/$NOMBRE.status")"
echo "cuerpo: $OUT_DIR/$NOMBRE.json"
