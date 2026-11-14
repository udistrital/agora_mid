#!/usr/bin/env bash
# MID 8080: POST /v1/proveedor/consulta con body requests/mid_proveedor_consulta_1.json
# Uso: ./scripts/mid_proveedor_consulta_1.sh   (cuerpo: requests/mid_proveedor_consulta_1.json)
set -u
DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$DIR/.."
OUT_DIR="${OUT_DIR:-$ROOT/responses/current}"
mkdir -p "$OUT_DIR"
NOMBRE="mid_proveedor_consulta_1"
curl -s -m 15 -X POST "http://localhost:8080/v1/proveedor/consulta" -H 'Content-Type: application/json' --data "@$ROOT/requests/mid_proveedor_consulta_1.json" -o "$OUT_DIR/$NOMBRE.json" -w '%{http_code}\n' > "$OUT_DIR/$NOMBRE.status"
echo "$NOMBRE HTTP $(cat "$OUT_DIR/$NOMBRE.status")"
echo "cuerpo: $OUT_DIR/$NOMBRE.json"
