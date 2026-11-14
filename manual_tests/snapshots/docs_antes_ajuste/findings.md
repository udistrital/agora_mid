# Hallazgos de las pruebas manuales de agora_mid

Estado vigente al 14/11/2026. Cada hallazgo tiene una carpeta con `finding.md`: explicación, causa, estado y propuesta. Los que tienen un cambio aplicado incluyen la lista de archivos modificados y cómo verificarlos (sección `## Cambio aplicado`). Las propuestas para decidir están al final de `../manual_tests.md`.

Los hallazgos resueltos y sin más detalle útil se podaron de la carpeta: F01 (columna `tercero_id`), F02 (404 de contacto y cuenta, consecuencia de F01 y F16), F12 (DBML y base coinciden), F16 (modelo de cuenta bancaria; el CRUD responde 200) y F21 (PK compartida; resuelta por el cambio de F19).

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
| F09 | en_decision | El arranque depende de un catálogo remoto | `F09_arranque_depende_catalogo_remoto/` |
| F10 | abierto | Tres objetos de configuración sin referencia | `F10_huerfanos_config/` |
| F11 | abierto | `bee run` no termina cuando el binario sale | `F11_bee_run_no_termina/` |
| F13 | abierto | Datos dummy con supuestos sin verificar; cleanup sin probar | `F13_seed_supuestos_y_cleanup/` |
| F14 | en_decision | 17 avisos `depth` en una respuesta sin pérdida de datos | `F14_profundidad_notify_ruido/` |
| F15 | abierto | Relaciones a `lugar` fallan: el host de ubicaciones rechaza conexiones | `F15_relacion_lugar_falla/` |
| F17 | en_decision | Guarda de solo lectura para terceros, parámetros y ubicaciones | `F17_guard_solo_lectura/` |
| F18 | abierto | Nombres de `API.md` distintos de la configuración | `F18_nombres_contrato_api/` |
| F19 | resuelto | El MID enviaba PATCH al CRUD, que solo acepta PUT | `F19_crud_no_patch/` |
| F20 | abierto | Proveedor sin `TerceroId` se guarda con `tercero_id = 0` | `F20_tercero_id_cero/` |
| F22 | abierto | No se validan los campos requeridos de un hijo | `F22_sin_validacion_requeridos/` |
| F23 | abierto | PATCH con `TerceroId` distinto responde 200 y no escribe | `F23_tercero_conflicto_200/` |
| F24 | en_decision | `TerceroId` no se verifica contra terceros | `F24_tercero_no_validado/` |
| F25 | en_decision | El CRUD exige `ProveedorId` y `ContactoId` como objeto | `F25_crud_fk_objeto_escritura/` |
| F26 | en_decision | `aCadena` no compara FK como objeto en escritura | `F26_fk_objeto_comparacion_escritura/` |
| F27 | en_decision | PATCH sin `Id` crea un proveedor | `F27_patch_sin_id_crea/` |
| F28 | en_decision | PATCH con `Id` inexistente crea un registro | `F28_patch_id_inexistente_crea/` |

## Estado de las pruebas Go

`go build ./...` pasa. `go test ./...` pasa en todos los paquetes salvo `test`, donde falla solo `TestIntegracionConsultaSinFieldsNoEntregaAuditoria` (F04).
