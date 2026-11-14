# Hallazgos de las pruebas manuales de agora_mid

Estado vigente al 14/11/2026. Cada hallazgo tiene una carpeta con `finding.md`: explicación, causa, estado y propuesta. Los que tienen un cambio aplicado incluyen la lista de archivos modificados y cómo verificarlos (sección `## Cambio aplicado`). Las propuestas para decidir están al final de `../manual_tests.md`.

Los hallazgos resueltos y sin más detalle útil se podaron de la carpeta: F01 (columna `tercero_id`), F02 (404 de contacto y cuenta, consecuencia de F01 y F16), F12 (DBML y base coinciden), F16 (modelo de cuenta bancaria; el CRUD responde 200) y F21 (PK compartida; resuelta por el cambio de F19).

## Índice

| ID | Estado | Resumen | Carpeta |
|---|---|---|---|
| F00 | Resuelto (R1 en el árbol de trabajo, con diff) | Lista negra `ANY/ANY/ANY` que aborta el arranque; URLs de `.env` con `/v1/` | `F00_resueltos/` |
| F03 | Corregido en el árbol de trabajo (con diff) | FK como objeto anidado en la expansión | `F03_fk_anidada_crud/` |
| F04 | Abierto | La lista negra no oculta la auditoría; falla una prueba de integración | `F04_blacklist_auditoria/` |
| F05 | Abierto | Swagger del CRUD es un stub de Beego | `F05_swagger_crud_stub/` |
| F06 | Abierto, sin evidencia vigente | Lista vacía: 200 con `[]` en unos recursos, 404 en otros | `F06_listas_vacias_inconsistentes/` |
| F07 | Abierto | El CRUD convierte cualquier error en 404 | `F07_crud_oculta_errores_como_404/` |
| F08 | Abierto | `API.md`: `ExperienciaLaboralMeses` vs `ExperienciaLaboral` | `F08_docs_nombre_experiencia/` |
| F09 | Abierto | El arranque depende de un catálogo remoto | `F09_arranque_depende_catalogo_remoto/` |
| F10 | Abierto, informativo | Tres objetos de configuración sin referencia | `F10_huerfanos_config/` |
| F11 | Abierto, operativo | `bee run` no termina cuando el binario sale | `F11_bee_run_no_termina/` |
| F13 | Abierto (entorno) | Datos dummy con supuestos sin verificar; cleanup sin probar | `F13_seed_supuestos_y_cleanup/` |
| F14 | Abierto | 17 avisos `depth` en una respuesta sin pérdida de datos | `F14_profundidad_notify_ruido/` |
| F15 | Abierto (dependencia externa) | Relaciones a `lugar` fallan: el host de ubicaciones rechaza conexiones | `F15_relacion_lugar_falla/` |
| F17 | Implementado en el árbol de trabajo (con diff); conflicto abierto | Guarda de solo lectura para terceros, parámetros y ubicaciones | `F17_guard_solo_lectura/` |
| F18 | Abierto | Nombres de `API.md` distintos de la configuración | `F18_nombres_contrato_api/` |
| F19 | Corregido en el árbol de trabajo (con diff); verificado en vivo | El MID enviaba PATCH al CRUD, que solo acepta PUT | `F19_crud_no_patch/` |
| F20 | Abierto | Proveedor sin `TerceroId` se guarda con `tercero_id = 0` | `F20_tercero_id_cero/` |
| F22 | Abierto | No se validan los campos requeridos de un hijo | `F22_sin_validacion_requeridos/` |
| F23 | Abierto | PATCH con `TerceroId` distinto responde 200 y no escribe | `F23_tercero_conflicto_200/` |
| F24 | Abierto | `TerceroId` no se verifica contra terceros | `F24_tercero_no_validado/` |
| F25 | Corregido (rápido) en el árbol de trabajo (con diff) | El CRUD exige `ProveedorId` y `ContactoId` como objeto | `F25_crud_fk_objeto_escritura/` |
| F26 | Corregido (rápido) en el árbol de trabajo (con diff) | `aCadena` no compara FK como objeto en escritura | `F26_fk_objeto_comparacion_escritura/` |
| F27 | Abierto (por definir) | PATCH sin `Id` crea un proveedor | `F27_patch_sin_id_crea/` |
| F28 | Abierto (regresión de F19) | PATCH con `Id` inexistente crea un registro | `F28_patch_id_inexistente_crea/` |

## Estado de las pruebas Go

`go build ./...` pasa. `go test ./...` pasa en todos los paquetes salvo `test`, donde falla solo `TestIntegracionConsultaSinFieldsNoEntregaAuditoria` (F04).
