# F23: PATCH con un TerceroId distinto responde 200 y no cambia nada

Estado: aceptado. Corrección rápida aplicada sin commit. Ver `## Cambio aplicado`.
Solución: aplicada_sin_commit. Ver `## Cambio aplicado`.
Gravedad: alta. Clasificación propuesta, ver utils/README.md.

## Explicación
`PATCH /v1/proveedor` con `Id: 1` y `TerceroId: 999` responde 200 "registrado correctamente", con `Data.Id = 1`. La especificación (`proveedor-write`, conflicto de FK ascendente en proveedor existente) pide 400 y no escribir nada. El 200 oculta que no se escribió nada.

## Causa
La validación de no-reasignación (`helper/write_engine.go`, `hayConflictoLlave`/`reasociaExistente`) se aplica a hijos (`!esRaiz`). La FK ascendente de la raíz sigue la regla `fk_inbound`, y la comparación no dispara el 400.

## Propuesta
En la raíz, comparar el `TerceroId` del cuerpo con el del registro existente y responder 400 si difieren.

## Reverificación (14/11, 14:03)

U03 (`PATCH` con `Id` 1 y `TerceroId` 999) sigue respondiendo 200. No pasa.

- `tercero_id` del proveedor 1 en la base: 101, sin cambio (respaldo previo en `snapshots/proveedores_antes_u03.sql`).
- La respuesta incluye `"TerceroId": 999` en `Data`, aunque la base no lo guardó. El cliente recibe un dato que no existe.
- Conteos de `proveedores` sin cambio.

La especificación (`proveedor-write`) y `API.md` (línea 197) piden 400 para este caso. No hay conflicto de especificación, así que la corrección no depende de la decisión pendiente de F27.

## Cambio aplicado

Corrección rápida a nivel de servicio, sin commit. Verifica el `TerceroId` del cuerpo contra el del registro existente en un PATCH con `Id`.

Comportamiento aplicado:
- PATCH con `Id` existente, `TerceroId` mayor que cero y distinto del `TerceroId` del registro (también mayor que cero): 400 `ErrorEscritura` en `TerceroId`, sin escritura.
- `TerceroId` omitido, 0 o null: no hay conflicto, como indica el registro de ids en cero (`escritura/matriz.md`).
- Registro sin `TerceroId` (mayor que cero): no bloquea.
- La lectura por `Id` es la misma de F27 y F28. No se hace una lectura extra.

Interpretación de la especificación: `proveedor-write`, escenario "Conflicto de FK ascendente en proveedor existente", dice "incompatible con el `tercero` resuelto para el cuerpo" y exige 400 sin ninguna escritura. El cuerpo solo trae el escalar `TerceroId`, así que el tercero resuelto es ese identificador, y la comparación contra el registro es la forma de demostrar compatibilidad. `API.md` línea 197 lo dice explícitamente: "distinto en una entidad existente responde `400` (sin reasignar)". La spec no nombra la comparación contra el registro; esta es la lectura usada.

**Archivos modificados:**

- `services/proveedor_write.go`: `verificarIdentidadPatch` (comparación de `TerceroId`), `verificarProveedorExistente` (ahora devuelve el registro), `motivoTerceroEnConflicto`.
- `services/proveedor_tercero_conflicto_test.go` (nuevo): conflicto (400), igual (sin 400), omitido, 0 y null (sin 400).

**Verificación:** `go test -count=1 ./...` pasa salvo `TestIntegracionConsultaSinFieldsNoEntregaAuditoria` (F04). En vivo (MID PID 44698, binario de 14:05:33, 14/11): U03 responde 400 con el mensaje de conflicto; U02 (`TerceroId` igual) responde 200 sin cambios. `tercero_id` del proveedor 1 sigue en 101 y los conteos no cambian. Respaldo previo: `snapshots/proveedores_antes_u03_u02.sql`.

## Notas de especificación

Propuesta de texto (no aprobada, no bloquea la implementación). Escenario propuesto para la especificación `proveedor-write`, que hoy solo dice que el conflicto de FK ascendente responde 400 sin escritura parcial:

> #### Scenario: PATCH con TerceroId distinto del registro existente
> - **WHEN** un `PATCH` trae `Id` de un proveedor existente y un `TerceroId` mayor que cero distinto del `TerceroId` del registro
> - **THEN** el sistema responde 400 sin crear ni actualizar nada en los CRUD de `tercero` ni de `proveedor`
>
> #### Scenario: TerceroId ausente, cero o nulo en PATCH de proveedor existente
> - **WHEN** un `PATCH` con `Id` no trae `TerceroId`, o lo trae como 0 o null
> - **THEN** el sistema no lo considera conflicto y continúa la reconciliación

La comparación contra el registro existente se toma de `API.md` línea 197. La especificación dice "el `tercero` resuelto para el cuerpo", que se leyó como el `TerceroId` del cuerpo cuando solo viene el escalar.

## Verificación en vivo (14/11, 15:20)

U03 (`PATCH` con `Id` 1 y `TerceroId` 999) responde 400 con "el TerceroId no coincide con el del proveedor existente; no se reasigna". Respaldo previo: `snapshots/proveedores_antes_f23.sql`. Los conteos y `tercero_id` del proveedor 1 (101) no cambian. La línea base se restauró tras la prueba.

## Decisión

Aceptado. El comportamiento de la corrección aplicada es el esperado por la especificación y por `API.md` (línea 197).

El ajuste de escenarios de `openspec/specs/proveedor-write` (ver Notas de especificación) queda documentado y no se aplica al spec desde estas pruebas. `API.md` se sincronizará con la especificación más adelante, en un solo paso.
