# Hallazgos de las pruebas manuales de agora_mid

Cada hallazgo tiene una carpeta con `finding.md`: explicación, causa, estado y propuesta. Los que tienen un cambio aplicado incluyen la lista de archivos modificados y cómo verificarlos (sección `## Cambio aplicado`). Las decisiones pendientes están al final de este documento. El procedimiento de pruebas está en `../manual_tests.md`.

## Resumen de estado

Última comprobación: 17/11/2026, 19:03 (hora local). Se actualiza al cerrar cada sesión (`../manual_tests.md`, 3.8).

- **Rama:** temporal. Los cambios de esta rama no llegarán a la rama principal. Sirven como referencia para planear las correcciones.
- **Hallazgos:** 26. Por estado: resuelto 6, en_decision 1, abierto 3, aceptado 10, cerrado_sin_solucion 7. Por solución: aplicada_sin_commit 16, propuesta 8, ninguna 3. Por gravedad: crítica 2, alta 12, media 7, baja 5. `findings.py verificar`: incumplimientos 0.
- **Pruebas del gestor (`utils/tests`):** 55 pruebas, OK (17/11, 19:03).
- **Pruebas Go (17/11, 19:03):** `go build ./...` pasa. `go test -count=1 ./...` pasa en todos los paquetes. No hay fallos.
- **Lecturas (`scripts/`, 17/11, 19:15):** 10 de 10 respondieron. Nueve con 200 y `mid_proveedor_consulta_1` con 206, igual que el 14/11. Tras la corrección de F14 (19:3x), la consulta viva da 206 con 4 avisos `depth` (antes 17), con rutas de hijas cortadas, y ningún `relation`.
- **Matriz de escritura:** registro en `../escritura/resultados.md` (inicio 14/11, 15:5x; la corrida terminó hacia las 16:47 según la hora de los archivos de `responses/escritura/current/`). De 39 casos, 38 coinciden en código HTTP. P11 no coincide (F22). Discrepancias de efecto: P05 (F20), U11 y U05 (F29). No se comprobó si el código de esa corrida coincide con el código actual de la rama (HEAD `695f005`).
- **Servicios (17/11, 19:3x):** MID reiniciado por el tester con 3.2 tras la corrección de F14 (PID 166563, proceso propio). Responde 405 a la consulta de comprobación. CRUD `gestion_personas` responde 200. La base no se comprobó en esta sesión.
- **Git (solo lectura, 17/11, 19:15):** HEAD `695f005`. Cambios sin commit: `manual_tests/manual_tests.md`, `manual_tests/findings/findings.md` y `manual_tests/responses/current/mid_proveedor_consulta_1.json`.
- **Pendientes de decisión:** ver la sección Decisiones pendientes. Antes de planear las correcciones, el usuario decide F04, F17, F20, F22, F24 y F29.
- **Reproducido el 17/11 (19:20):**
  - F13: en la base de pruebas `agora_pruebas_manual` (copia del esquema, vaciada), el seed carga 2, 1, 1, 1, 1 filas y el cleanup deja 0 en todas las tablas. Sin errores. Falta probar el cleanup con filas ajenas a los ids 1 y 2 que dependan de ellos.
  - F24: `P04` de la matriz responde 400 con efecto ninguno (respaldo `snapshots/proveedores_antes_f24_17nov.sql`, conteos sin cambio). GET a terceros `tercero/999999` responde 404.
  - F06: no reproducido. Requiere un CRUD de `gestion_personas` apuntando a una base vacía, y el CRUD actual usa `agora_db`.
- **Pendientes de prueba:** reproducir F06 (ver arriba). Verificar F11 con `bee`, fuera de este procedimiento. Hay un proceso `bee run` activo (PID 135358) que no lo arrancó el tester.

## Índice

| ID | Estado | Resumen | Carpeta |
|---|---|---|---|
| F00 | resuelto | Lista negra `ANY/ANY/ANY` que aborta el arranque; URLs de `.env` con `/v1/` | `F00_resueltos/` |
| F03 | resuelto | FK como objeto anidado en la expansión | `F03_fk_anidada_crud/` |
| F04 | aceptado | La lista negra no oculta la auditoría; falla una prueba de integración | `F04_blacklist_auditoria/` |
| F05 | cerrado_sin_solucion | Swagger del CRUD es un stub de Beego | `F05_swagger_crud_stub/` |
| F06 | cerrado_sin_solucion | Lista vacía: 200 con `[]` en unos recursos, 404 en otros | `F06_listas_vacias_inconsistentes/` |
| F07 | cerrado_sin_solucion | El CRUD convierte cualquier error en 404 | `F07_crud_oculta_errores_como_404/` |
| F08 | aceptado | `API.md`: `ExperienciaLaboralMeses` vs `ExperienciaLaboral` | `F08_docs_nombre_experiencia/` |
| F09 | resuelto | El arranque depende de un catálogo remoto | `F09_arranque_depende_catalogo_remoto/` |
| F10 | aceptado | Tres objetos de configuración sin referencia | `F10_huerfanos_config/` |
| F11 | cerrado_sin_solucion | `bee run` no termina cuando el binario sale | `F11_bee_run_no_termina/` |
| F13 | cerrado_sin_solucion | Datos dummy con supuestos sin verificar; cleanup sin probar | `F13_seed_supuestos_y_cleanup/` |
| F14 | aceptado | 17 avisos `depth` en una respuesta sin pérdida de datos | `F14_profundidad_notify_ruido/` |
| F15 | resuelto | Relaciones a `lugar` fallan: el host de ubicaciones rechaza conexiones | `F15_relacion_lugar_falla/` |
| F17 | aceptado | Guarda de solo lectura para terceros, parámetros y ubicaciones | `F17_guard_solo_lectura/` |
| F18 | aceptado | Nombres de `API.md` distintos de la configuración | `F18_nombres_contrato_api/` |
| F19 | resuelto | El MID enviaba PATCH al CRUD, que solo acepta PUT | `F19_crud_no_patch/` |
| F20 | cerrado_sin_solucion | Proveedor sin `TerceroId` se guarda con `tercero_id = 0` | `F20_tercero_id_cero/` |
| F22 | aceptado | No se validan los campos requeridos de un hijo | `F22_sin_validacion_requeridos/` |
| F23 | aceptado | PATCH con `TerceroId` distinto responde 200 y no escribe | `F23_tercero_conflicto_200/` |
| F24 | aceptado | `TerceroId` no se verifica contra terceros | `F24_tercero_no_validado/` |
| F25 | aceptado | El CRUD exige `ProveedorId` y `ContactoId` como objeto | `F25_crud_fk_objeto_escritura/` |
| F26 | aceptado | `aCadena` no compara FK como objeto en escritura | `F26_fk_objeto_comparacion_escritura/` |
| F27 | aceptado | PATCH sin `Id` crea un proveedor | `F27_patch_sin_id_crea/` |
| F28 | resuelto | PATCH con `Id` inexistente crea un registro | `F28_patch_id_inexistente_crea/` |
| F29 | cerrado_sin_solucion | Un null o un campo ausente en PATCH cambia el valor guardado | `F29_null_ausente_patch/` |
| F31 | aceptado | El aviso depth nombra la relación padre en vez de la hija cortada | `F31_aviso_depth_solo_con_corte/` |
| F32 | cerrado_sin_solucion | El modelo del CRUD convierte un NULL en valor por defecto al actualizar | `F32_patch_escribe_valores_por_omision/` |

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
9. **F14 y F31 (17/11, 19:27 a 19:44).** F14: sin aviso por reducción que no corta hijos. F31 (aceptado): el aviso `depth` nombra la ruta de la hija cortada. Pendiente: aplicar los dos escenarios de `nested-object-expansion` y la descripción de `swagger/swagger.yml`, con los textos en `F14_profundidad_notify_ruido/finding.md` y `F31_aviso_depth_solo_con_corte/finding.md`, `## Notas de especificación`.
10. **F09 (media).** Comprobar el host de terceros antes de arrancar, o cachear el catálogo `P_NATURAL`.
11. **F07 y F06 (CRUD).** Responder 404 solo para `ErrNoRows`, 500 para el resto. Lista vacía: 200 con `Data: []`. Revalidar F06 primero.
12. **F05, F08, F18 (documentación).** Regenerar el swagger del CRUD. Corregir los nombres de `API.md` según la tabla de naming de `../escritura/matriz.md`.
13. **F13 (entorno).** Probar `seed/cleanup_dummy_data.sql` en una base de pruebas y verificar los catálogos del seed.
14. **F10, F11 (operación).** Borrar los YAML sin referencia o ignorar la advertencia. Arrancar siempre el binario, no `bee run`.
15. **F15 (cerrado 17/11, 19:21).** Resuelto. Ya no hay avisos `relation` y `PaisNacimiento` se expande. Sin cambio en el repositorio.
16. **Cambios de atajo.** F03, F25 y F26 están como arreglos rápidos sin commit. Decidir si se reemplazan por un diseño definitivo o se aceptan. F25 y F26 son parches de forma; F03 aplana cualquier mapa con `Id`.
17. **Etiquetas de commit.** Los hallazgos con solución `aplicada_sin_commit` (F03, F23, F24, F25, F26, F27, F28 y otros) ya están en `815c76d`. Decidir si pasan a otra etiqueta. Cambiarlas requiere `findings.py cambiar` con `--aplicar`, una por una, tras revisar cada propuesta.
