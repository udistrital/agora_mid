#!/usr/bin/env bash
# CRUD 8081: OpenAPI pedido directamente (stub Beego, ver F05)
# Uso: ./scripts/crud_swagger_json.sh
set -u
DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$DIR/.."
OUT_DIR="${OUT_DIR:-$ROOT/responses/current}"
mkdir -p "$OUT_DIR"
NOMBRE="crud_swagger_json"
curl -s -m 15 -X GET "http://localhost:8081/swagger/swagger.json" -o "$OUT_DIR/$NOMBRE.json" -w '%{http_code}\n' > "$OUT_DIR/$NOMBRE.status"
echo "$NOMBRE HTTP $(cat "$OUT_DIR/$NOMBRE.status")"
echo "cuerpo: $OUT_DIR/$NOMBRE.json"
