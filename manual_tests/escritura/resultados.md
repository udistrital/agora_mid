# Resultados de la última corrida de la matriz de escritura

Fecha: 14/11/2026, 15:5x (hora local).
MID: `/tmp/agora_mid_run_f04`, compilado con el código del 15:37 (cambios sin commit). Puerto 8080.
Respaldo previo y restauración: `snapshots/proveedores_antes_matriz_2026-11-14.sql`. Los conteos volvieron a la línea base (proveedor 2, natural 1, jurídico 1, contacto 1, cuenta 1).
Corrida con huella de filas por caso (script temporal en `/tmp/matriz_run.sh`, no versionado).

## Código HTTP por caso

| ID | HTTP | Esperado | Coincide |
|---|---|---|---|
| P01 | 200 | 200 | sí |
| P02 | 200 | 200 | sí |
| P03 | 200 | 200 | sí |
| P04 | 400 | 400 | sí |
| P05 | 200 | 200 | sí |
| P06 | 200 | 200 | sí |
| P07 | 400 | 400 | sí |
| P08 | 400 | 400 | sí |
| P09 | 400 | 400 | sí |
| P10 | 200 | 200 | sí |
| P11 | 200 | 400 o 502 | no |
| P12 | 200 | 200 | sí |
| P13 | 400 | 400 | sí |
| P14 | 400 | 400 | sí |
| P15 | 400 | 400 | sí |
| P16 | 400 | 400 | sí |
| P17 | 400 | 400 | sí |
| P18 | 400 | 400 | sí |
| P19 | 200 | 200 | sí |
| P20 | 200 | 200 | sí |
| U01 | 200 | 200 | sí |
| U02 | 200 | 200 | sí |
| U03 | 400 | 400 | sí |
| U04 | 200 | 200 | sí |
| U05 | 200 | 200 | sí |
| U06 | 400 | 400 | sí |
| U07 | 200 | 200 | sí |
| U08 | 200 | 200 | sí |
| U09 | 200 200 | 200 ambas | sí |
| U10 | 200 | 200 | sí |
| U11 | 200 | 200 | sí |
| U12 | 400 | 400 | sí |
| U13 | 404 | 404 | sí |
| U14 | 400 | 400 | sí |
| U15 | 200 | 200 | sí |
| U16 | 400 | 400 | sí |
| U17 | 200 | 200 | sí |
| U18 | 200 | 200 | sí |
| U19 | 200 | 200 | sí |

38 de 39 casos coinciden en código HTTP. El caso que no coincide es P11.

## Discrepancias de efecto y de código

- **P11**: responde 200 y crea un contacto con `proposito` vacío. Esperado 400 o 502. Hallazgo F22.
- **P05**: responde 200 como se esperaba, pero `tercero_id` queda en 0. Esperado NULL. Hallazgo F20.
- **U11**: responde 200 y `descripcion_portafolio` queda como cadena vacía. Esperado NULL. Hallazgo F29.
- **U05**: responde 200 y `contacto.extension` pasa de cadena vacía a NULL. El caso no pide ese cambio. Hallazgo F29.

Los casos P03, P04, U01, U03, U04, U07, U08, U09, U13, U15 y U17 tienen el efecto en filas esperado.

## Cambios de procedimiento en esta corrida

- `scripts/escritura/U09.sh` ahora envía el cuerpo dos veces, como dice la matriz. Antes enviaba una vez.
- La matriz fija P04 en 400 con efecto ninguno (decisión de F24).
