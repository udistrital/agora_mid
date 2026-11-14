#!/usr/bin/env bash
# U09: PATCH /v1/proveedor. HTTP esperado: 200 ambas. Cuerpo: requests/escritura/u09_idempotente.json
set -u
DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$DIR/../.."
OUT_DIR="${OUT_DIR:-$ROOT/responses/escritura/current}"
mkdir -p "$OUT_DIR"
NOMBRE="U09"
curl -s -m 30 -X PATCH "http://localhost:8080/v1/proveedor" -H 'Content-Type: application/json' --data "@$ROOT/requests/escritura/u09_idempotente.json" -o "$OUT_DIR/$NOMBRE.json" -w '%{http_code}\n' > "$OUT_DIR/$NOMBRE.status"
curl -s -m 30 -X PATCH "http://localhost:8080/v1/proveedor" -H 'Content-Type: application/json' --data "@$ROOT/requests/escritura/u09_idempotente.json" -o "$OUT_DIR/$NOMBRE.2.json" -w '%{http_code}\n' >> "$OUT_DIR/$NOMBRE.status"
echo "$NOMBRE $(tr '\n' ' ' < "$OUT_DIR/$NOMBRE.status")"
