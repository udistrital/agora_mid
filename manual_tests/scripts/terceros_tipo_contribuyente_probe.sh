#!/usr/bin/env bash
# Terceros remoto: catálogo P_NATURAL con la URL correcta (sin /v1 doble)
# Uso: ./scripts/terceros_tipo_contribuyente_probe.sh
set -u
DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$DIR/.."
OUT_DIR="${OUT_DIR:-$ROOT/responses/current}"
mkdir -p "$OUT_DIR"
NOMBRE="terceros_tipo_contribuyente_probe"
curl -s -m 15 -X GET "http://pruebasapi.intranetoas.udistrital.edu.co:8121/v1/tipo_contribuyente?query=CodigoAbreviacion:P_NATURAL,Activo:true&limit=2" -o "$OUT_DIR/$NOMBRE.json" -w '%{http_code}\n' > "$OUT_DIR/$NOMBRE.status"
echo "$NOMBRE HTTP $(cat "$OUT_DIR/$NOMBRE.status")"
echo "cuerpo: $OUT_DIR/$NOMBRE.json"
