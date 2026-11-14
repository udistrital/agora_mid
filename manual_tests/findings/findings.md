# Hallazgos de las pruebas manuales de agora_mid

Cada hallazgo tiene una carpeta con `finding.md`: explicación, causa, estado y propuesta. Los que tienen un cambio aplicado incluyen la lista de archivos modificados y cómo verificarlos (sección `## Cambio aplicado`). Las decisiones pendientes están al final de este documento. El procedimiento de pruebas está en `../manual_tests.md`.

## Resumen de estado

Última comprobación: 14/11/2026, 15:5x (hora local). Se actualiza al cerrar cada sesión (`../manual_tests.md`, 3.8).

- **Hallazgos:** 29. Por estado: resuelto 4, en_decision 3, abierto 13, aceptado 5. Por solución: aplicada_sin_commit 11, propuesta 11, ninguna 3. Por gravedad: crítica 2, alta 12, media 6, baja 5. `findings.py verificar`: incumplimientos 0.
- **Pruebas del gestor (`utils/tests`):** 55 pruebas, OK.
- **Pruebas Go:** `go build ./...` pasa. `go test -count=1 ./...` pasa en todos los paquetes, incluido `test`. No hay fallos conocidos.
- **Lecturas (`scripts/`):** ejecutadas el 14/11, 15:5x. Nueve con 200, `mid_proveedor_consulta_1` con 206 (17 avisos `depth` y 3 `relation`, F14 y F15).
- **Matriz de escritura:** corrida el 14/11, 15:5x, contra el MID de `/tmp/agora_mid_run_f04`. 39 casos. 38 coinciden en código HTTP. Discrepancias: P11 (F22), P05 (F20, efecto `tercero_id` 0), U11 y U05 (F29). Registro en `../escritura/resultados.md`. Se restauró el respaldo y los conteos volvieron a la línea base.
- **Verificación de comportamiento de los hallazgos (14/11, 15:5x):** ver la columna de cada `finding.md`. Verificados en vivo: F00, F03, F04, F05, F10, F14, F15, F19, F20, F22, F23, F24, F25 (indirecto), F26, F27 (solo PATCH sin Id), F28, F29. Solo con pruebas unitarias: F17. No verificados: F11, F13. Afirmaciones corregidas: F06 (no se reproduce con filtros), F07 (parcial).
- **Servicios (14/11, 15:5x):** MID en `8080` responde 405 (`/tmp/agora_mid_run_f04`, instancia propia). CRUD `gestion_personas` en `8081` responde 200.
- **Base `proveedores`:** proveedor 2, natural 1, jurídico 1, contacto 1, cuenta 1 (línea base).
- **Cambios sin commit:** `clients/`, `config/`, `controllers/`, `helper/`, `services/`, `test/`, `main.go` y `manual_tests/`. Git solo en lectura.
- **Pendientes de decisión:**
  - F03: aceptar o mantener en decisión.
  - F04: aprobar el texto de `nested-object-expansion` y retirar la solución provisional al aprobarlo.
  - F29: definir el contrato de null y de campo ausente en PATCH.
  - F20 y F22: decisiones de diseño (opciones en su finding).
  - F24: abierto hasta retirar la guarda de terceros (F17) en un entorno local.
  - F14: decisión del dueño de la especificación.
- **Sincronización de especificación (pendiente, un solo paso):** `nested-object-expansion` (F04, F03), `proveedor-write` (F23, F27, F28), `api-response-contract` (tipo de aviso `fk_no_aplanable`), y `API.md` (F08, F18, F23). Ninguno se aplicó desde estas pruebas.
- **Pendientes de prueba:** reproducir F06 con tabla vacía en una base de pruebas. Probar `seed/cleanup_dummy_data.sql` en una base de pruebas (F13). Verificar F11 con `bee` fuera de este procedimiento.

## Índice

| ID | Estado | Resumen | Carpeta |
|---|---|---|---|
| F00 | resuelto | Lista negra `ANY/ANY/ANY` que aborta el arranque; URLs de `.env` con `/v1/` | `F00_resueltos/` |
| F03 | en_decision | FK como objeto anidado en la expansión | `F03_fk_anidada_crud/` |
| F04 | en_decision | La lista negra no oculta la auditoría; falla una prueba de integración | `F04_blacklist_auditoria/` |
| F05 | abierto | Swagger del CRUD es un stub de Beego | `F05_swagger_crud_stub/` |
| F06 | abierto | Lista vacía: 200 con `[]` en unos recursos, 404 en otros | `F06_listas_vacias_inconsistentes/` |
| F07 | abierto | El CRUD convierte cualquier error en 404 | `F07_crud_oculta_errores_como_404/` |
| F08 | abierto | `API.md`: `ExperienciaLaboralMeses` vs `ExperienciaLaboral` | `F08_docs_nombre_experiencia/` |
| F09 | resuelto | El arranque depende de un catálogo remoto | `F09_arranque_depende_catalogo_remoto/` |
| F10 | abierto | Tres objetos de configuración sin referencia | `F10_huerfanos_config/` |
| F11 | abierto | `bee run` no termina cuando el binario sale | `F11_bee_run_no_termina/` |
| F13 | abierto | Datos dummy con supuestos sin verificar; cleanup sin probar | `F13_seed_supuestos_y_cleanup/` |
| F14 | en_decision | 17 avisos `depth` en una respuesta sin pérdida de datos | `F14_profundidad_notify_ruido/` |
| F15 | abierto | Relaciones a `lugar` fallan: el host de ubicaciones rechaza conexiones | `F15_relacion_lugar_falla/` |
| F17 | aceptado | Guarda de solo lectura para terceros, parámetros y ubicaciones | `F17_guard_solo_lectura/` |
| F18 | abierto | Nombres de `API.md` distintos de la configuración | `F18_nombres_contrato_api/` |
| F19 | resuelto | El MID enviaba PATCH al CRUD, que solo acepta PUT | `F19_crud_no_patch/` |
| F20 | abierto | Proveedor sin `TerceroId` se guarda con `tercero_id = 0` | `F20_tercero_id_cero/` |
| F22 | abierto | No se validan los campos requeridos de un hijo | `F22_sin_validacion_requeridos/` |
| F23 | aceptado | PATCH con `TerceroId` distinto responde 200 y no escribe | `F23_tercero_conflicto_200/` |
| F24 | abierto | `TerceroId` no se verifica contra terceros | `F24_tercero_no_validado/` |
| F25 | aceptado | El CRUD exige `ProveedorId` y `ContactoId` como objeto | `F25_crud_fk_objeto_escritura/` |
| F26 | aceptado | `aCadena` no compara FK como objeto en escritura | `F26_fk_objeto_comparacion_escritura/` |
| F27 | aceptado | PATCH sin `Id` crea un proveedor | `F27_patch_sin_id_crea/` |
| F28 | resuelto | PATCH con `Id` inexistente crea un registro | `F28_patch_id_inexistente_crea/` |
| F29 | abierto | Un null o un campo ausente en PATCH cambia el valor guardado | `F29_null_ausente_patch/` |

## Decisiones pendientes

Ordenadas por prioridad. Ninguna está aplicada salvo que se indique.

1. **F28 (cerrado 14/11, 14:39).** PATCH con `Id` inexistente responde 404 sin escribir, según la regla de identidad de F27. Pendiente solo aprobar el escenario en `openspec/specs/proveedor-write`, ver `F28_patch_id_inexistente_crea/finding.md`, sección `## Notas de especificación`.
2. **F22 (alta).** Validar en el MID los campos requeridos de cada objeto antes de escribir. Devolver 400 con el nombre del campo. Afecta P11.
3. **F23 (alta).** Corrección rápida aplicada sin commit (14/11, 14:08): PATCH con `Id` y `TerceroId` distinto del registro responde 400 sin escribir. Lectura de la especificación y verificación en vivo en `F23_tercero_conflicto_200/finding.md`, sección `## Cambio aplicado`.
4. **F24 (alta, aplicado sin commit, 14/11).** El alta con `TerceroId` suelto (sin objeto `Tercero`) consulta terceros y responde 400 si el tercero no existe. Pendiente: verificación en vivo con P04 (requiere reinicio del MID y respaldo), aprobar el texto de `API.md:197` y decidir la verificación en PATCH. Ver `F24_tercero_no_validado/finding.md`.
5. **F17 (decisión).** Elegir entre guarda total de `terceros` (y dejar pendiente la escritura de `tercero`) o una lista explícita de destinos. La segunda requiere cambiar `openspec/specs/proveedor-write`.
6. **F04 (decisión).** Opción A: entrada concreta por recurso en `global_blacklist.yaml`, ajustando el validador. Opción B: validador que consulte campos de objetos generados (cambia la especificación `nested-object-expansion`). Corrige la única prueba roja.
7. **F27 (media).** Regla del usuario (14/11, 13:45): PATCH sin `Id` ni `TerceroId` responde 400; `Id` o clave natural sin proveedor responde 404. Ids y `TerceroId` en 0 cuentan como ausentes. Aplicada sin commit en el servicio. La especificación no bloquea: el texto propuesto está en `F27_patch_sin_id_crea/finding.md`, sección `## Notas de especificación`. Se actualizará `openspec/specs/proveedor-write` y `API.md` cuando se decida.
8. **F20 (media).** Convertir `TerceroId` 0 a NULL en el CRUD (`TerceroId *int`), o no enviarlo desde el MID cuando es 0.
9. **F14 (decisión).** Dejar de emitir `depth` cuando la relación se resuelve completa, o documentar que el 206 no implica pérdida de datos.
10. **F09 (media).** Comprobar el host de terceros antes de arrancar, o cachear el catálogo `P_NATURAL`.
11. **F07 y F06 (CRUD).** Responder 404 solo para `ErrNoRows`, 500 para el resto. Lista vacía: 200 con `Data: []`. Revalidar F06 primero.
12. **F05, F08, F18 (documentación).** Regenerar el swagger del CRUD. Corregir los nombres de `API.md` según la tabla de naming de `../escritura/matriz.md`.
13. **F13 (entorno).** Probar `seed/cleanup_dummy_data.sql` en una base de pruebas y verificar los catálogos del seed.
14. **F10, F11 (operación).** Borrar los YAML sin referencia o ignorar la advertencia. Arrancar siempre el binario, no `bee run`.
15. **F15 (dependencia).** No se corrige desde el MID. Repetir la comprobación del host de ubicaciones antes de cada corrida.
16. **Cambios de atajo.** F03, F25 y F26 están como arreglos rápidos sin commit. Decidir si se reemplazan por un diseño definitivo o se aceptan. F25 y F26 son parches de forma; F03 aplana cualquier mapa con `Id`.
