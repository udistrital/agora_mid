# F28: un PATCH con un Id que no existe crea un registro nuevo

Estado: resuelto. Regresión introducida por F19 (14/11/2026). Reproducido en la corrida 3 y la corrida 4: U13 respondía 200 y creaba el proveedor 13. Regla de identidad aplicada sin commit (404 por Id o clave natural, ver F27). Conflicto con `openspec/specs/proveedor-write` (requisito "crea lo ausente"): la decisión de contrato está cerrada (404 sin escribir, regla del 14/11 a las 13:45, ver F27). Falta actualizar la especificación, ver `## Notas de especificación`.
Solución: aplicada_sin_commit. Ver `## Cambio aplicado`.
Gravedad: critica. Clasificación propuesta, ver utils/README.md.

## Explicación

Un PATCH con un `Id` que no existe crea un registro nuevo. Es regresión de F19. Ver `## Causa`.

## Causa
Con F19, `Recuperar` devuelve `nil` cuando el CRUD responde 404 (`buscarRegistroExistente`). El motor clasifica el payload como creación. En la raíz, la creación descarta el `Id` del cliente y el CRUD asigna el siguiente de la secuencia. Antes de F19 el caso respondía 502.

Los hijos con PK compartida sí necesitan crear con el Id del padre (P06 y P19 dan 200 con `proveedor_natural.id = proveedor.id`), así que no se puede eliminar la creación con `Id`.

## Propuesta
Distinguir en el motor la raíz con `Id` del cliente de un hijo con PK compartida propagada por el padre. Para la raíz con `Id` ausente, responder 404 (o 400) sin escribir. Para el hijo, mantener la creación con el Id del padre. Decidido: la raíz con `Id` inexistente responde 404 sin escribir.

## Cambio aplicado

Corrección rápida a nivel de servicio, sin commit. Limitada a la raíz `/v1/proveedor` y solo para PATCH con `Id`. La decisión de contrato está cerrada: responde 404 sin escribir, según la regla de identidad acordada el 14/11 (13:45, ver F27).

Comportamiento aplicado:
- PATCH con `Id > 0`: el servicio consulta `GET {gestion_personas_crud}/v1/proveedor/{Id}` antes de cualquier escritura. Si el CRUD responde 404, devuelve `ErrProveedorNoExiste`, que el controlador mapea a 404. Cualquier otro fallo de lectura devuelve error (502 o 504 según el caso) y tampoco escribe.
- POST con `Id` y PATCH sin `Id` no cambian. El POST sigue creando; el PATCH sin `Id` sigue cubierto por F27.

**Archivos modificados:**

- `services/proveedor_write.go`: `ErrProveedorNoExiste`, verificación en `EscribirProveedor`, función `verificarProveedorExistente`.
- `controllers/proveedor.go`: mapeo de `ErrProveedorNoExiste` a 404 en `responderErrorEscritura`; comentario de `Escribir`.
- `services/proveedor_inexistente_test.go` (nuevo): pruebas de PATCH inexistente, PATCH existente, fallo de lectura, POST con `Id` y PATCH sin `Id`.
- `services/proveedor_write_test.go`: `servicioWritePrueba` ahora recibe `Client` y `URLs`, necesarios para la verificación.
- `controllers/proveedor_test.go`: caso "proveedor inexistente en PATCH 404" en la tabla de `TestProveedorHandlerEscribir`.

**Verificación:** `go test -count=1 ./...` pasa en todos los paquetes salvo `test`, donde falla solo `TestIntegracionConsultaSinFieldsNoEntregaAuditoria` (F04, anterior a este cambio). Las pruebas de `services/proveedor_inexistente_test.go` y el caso 404 de `controllers` pasan.

**Verificación en vivo (14/11, 13:43):** MID reiniciado (PID 29453). U13 (`PATCH /v1/proveedor` con `Id` 999999) responde 404 con `"Message":"el proveedor indicado por Id no existe"`. Los conteos de `proveedores` no cambiaron (proveedor 2, natural 1, jurídico 1, contacto 1, cuenta 1). Respaldo previo: `snapshots/proveedores_antes_u13.sql`. Solo se ejecutó U13; el resto de la matriz no se volvió a correr.

**Actualización (14/11, 13:47):** la regla de identidad del PATCH de la raíz (400 sin `Id` ni `TerceroId`; 404 si el `Id` o la clave natural no existen) también se aplica en el mismo servicio. Ver `../F27_patch_sin_id_crea/finding.md`, sección `## Cambio aplicado`. Pendiente la decisión sobre el conflicto con `openspec/specs/proveedor-write`.

## Notas de especificación

Propuesta de texto (no aprobada, no bloquea la implementación; ver F27, `## Notas de especificación`, para el requisito completo de PATCH de la raíz). Escenario propuesto para la especificación:

> #### Scenario: PATCH con Id inexistente
> - **WHEN** el cuerpo trae `Id` mayor que cero y ningún proveedor lo tiene
> - **THEN** el sistema responde 404 y no escribe nada

Hoy la especificación ("crea lo ausente") no prohíbe crear con un `Id` explícito que no existe, así que la regla contradice esa frase para la raíz. Los hijos con clave primaria compartida no cambian: siguen creándose con el `Id` del padre (P06, P19).

Decisión tomada: 404. Queda pendiente aprobar este escenario y aplicarlo en `openspec/specs/proveedor-write`, por separado.
