# Resultados vigentes de la matriz de escritura

Matriz: `matriz.md`. Scripts: `../scripts/escritura/`. Cuerpos: `../requests/escritura/`. Respuestas vigentes: `../responses/escritura/current/` (cuerpo `.json` y código `.status`).

Última corrida: 14/11/2026, MID con F19 corregido y CRUD de 8081 corregido (modelo de cuenta bancaria). Estado de la base antes y después: proveedor 2, natural 1, jurídico 1, contacto 1, cuenta 1.

Resultado: de los 36 casos con resultado esperado definido, 34 coinciden en código HTTP. No coinciden: P11 y U03. Sin resultado esperado definido (la matriz dice "por definir"): P04, U13 y U16.

| ID | HTTP | Esperado (matriz) | Nota |
|---|---|---|---|
| P01 | 200 | 200 | F20: tercero_id queda en 0. |
| P02 | 200 | 200 | F20. |
| P03 | 200 | 200 | Reutiliza el proveedor 1 por clave natural (`TerceroId` 101). Es el comportamiento especificado. La matriz esperaba +1 y se corrigió. |
| P04 | 200 | por definir | F24: acepta TerceroId inexistente. |
| P05 | 200 | 200 | F20. |
| P06 | 200 | 200 |  |
| P07 | 400 | 400 |  |
| P08 | 400 | 400 |  |
| P09 | 400 | 400 |  |
| P10 | 200 | 200 |  |
| P11 | 200 | 400 o 502 | F22: contacto sin Proposito. Esperado 400/502. |
| P12 | 200 | 200 |  |
| P13 | 400 | 400 |  |
| P14 | 400 | 400 |  |
| P15 | 400 | 400 |  |
| P16 | 400 | 400 |  |
| P17 | 400 | 400 |  |
| P18 | 400 | 400 |  |
| P19 | 200 | 200 |  |
| P20 | 200 | 200 |  |
| U01 | 200 | 200 |  |
| U02 | 200 | 200 |  |
| U03 | 200 | 400 | F23: TerceroId distinto responde 200. |
| U04 | 200 | 200 |  |
| U05 | 200 | 200 |  |
| U06 | 400 | 400 |  |
| U07 | 200 | 200 |  |
| U08 | 200 | 200 |  |
| U09 | 200 | 200 (duplica) | Efecto (+2 contactos) no verificado. |
| U10 | 200 | 200 |  |
| U11 | 200 | 200 |  |
| U12 | 400 | 400 |  |
| U13 | 200 | por definir | F28: crea un proveedor nuevo (regresión de F19). Sin resultado esperado definido; el efecto observado está en F28. |
| U14 | 400 | 400 |  |
| U15 | 200 | 200 |  |
| U16 | 200 | por definir | F27: crea un proveedor sin Id. |
| U17 | 200 | 200 |  |
| U18 | 200 | 200 |  |
| U19 | 200 | 200 |  |

Coinciden en HTTP, pero con un hallazgo: P01, P02 y P05 (F20); U09 (efecto no verificado).
