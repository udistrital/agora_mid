#!/usr/bin/env bash
# U16: PATCH /v1/proveedor. HTTP esperado: Por definir. Cuerpo: requests/escritura/u16_sin_id_root.json
set -u
DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$DIR/../.."
OUT_DIR="${OUT_DIR:-$ROOT/responses/escritura/current}"
mkdir -p "$OUT_DIR"
NOMBRE="U16"
curl -s -m 30 -X PATCH "http://localhost:8080/v1/proveedor" -H 'Content-Type: application/json' --data "@$ROOT/requests/escritura/u16_sin_id_root.json" -o "$OUT_DIR/$NOMBRE.json" -w '%{http_code}\n' > "$OUT_DIR/$NOMBRE.status"
echo "$NOMBRE $(cat "$OUT_DIR/$NOMBRE.status")"
