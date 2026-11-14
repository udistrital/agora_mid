#!/usr/bin/env bash
# P16: POST /v1/proveedor. HTTP esperado: 400. Cuerpo: requests/escritura/p16_sin_tipo.json
set -u
DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$DIR/../.."
OUT_DIR="${OUT_DIR:-$ROOT/responses/escritura/current}"
mkdir -p "$OUT_DIR"
NOMBRE="P16"
curl -s -m 30 -X POST "http://localhost:8080/v1/proveedor" -H 'Content-Type: application/json' --data "@$ROOT/requests/escritura/p16_sin_tipo.json" -o "$OUT_DIR/$NOMBRE.json" -w '%{http_code}\n' > "$OUT_DIR/$NOMBRE.status"
echo "$NOMBRE $(cat "$OUT_DIR/$NOMBRE.status")"
