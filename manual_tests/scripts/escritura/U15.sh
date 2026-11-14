#!/usr/bin/env bash
# U15: PATCH /v1/proveedor. HTTP esperado: 200. Cuerpo: requests/escritura/u15_coleccion_vacia.json
set -u
DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$DIR/../.."
OUT_DIR="${OUT_DIR:-$ROOT/responses/escritura/current}"
mkdir -p "$OUT_DIR"
NOMBRE="U15"
curl -s -m 30 -X PATCH "http://localhost:8080/v1/proveedor" -H 'Content-Type: application/json' --data "@$ROOT/requests/escritura/u15_coleccion_vacia.json" -o "$OUT_DIR/$NOMBRE.json" -w '%{http_code}\n' > "$OUT_DIR/$NOMBRE.status"
echo "$NOMBRE $(cat "$OUT_DIR/$NOMBRE.status")"
