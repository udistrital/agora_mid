# F24: TerceroId no se verifica contra terceros

Estado: abierto. Reproducido en la corrida 4: P04 (`TerceroId` 999999) responde 200.
Solución: aplicada_sin_commit. Ver `## Cambio aplicado`. Pendiente: verificación en vivo (P04) y decisión del PATCH.
Gravedad: alta. Clasificación propuesta, ver utils/README.md.

## Explicación
El proveedor se crea con `tercero_id = 999999`. Ese tercero no existe: `GET ${AGORA_MID_TERCEROS_CRUD_URL}/v1/tercero/999999` responde 404 (solo lectura). El contrato trata `TerceroId` como FK autoritativa del servicio de terceros.

## Propuesta
Al escribir la raíz, consultar `GET /v1/tercero/{TerceroId}` y devolver 400 si responde 404. Requiere decidir si el MID consulta terceros en cada escritura.

## Cambio aplicado

Corrección rápida a nivel de servicio, sin commit. Limitada al alta raíz (POST) con `TerceroId` suelto, es decir, sin objeto `Tercero` en el cuerpo.

Comportamiento aplicado:
- POST con `TerceroId` mayor que cero y sin objeto `Tercero`: el servicio hace `GET {terceros}/v1/tercero/{TerceroId}` antes de escribir. Si responde 404, devuelve 400 en `TerceroId` ("el TerceroId no existe en el CRUD de terceros") sin escribir. Cualquier otro fallo devuelve error (502 o 504 según el caso) sin escribir.
- `TerceroId` ausente, 0 o null: no consulta terceros.
- Con objeto `Tercero`: no verifica `TerceroId`, porque el motor crea o actualiza el tercero en la misma escritura y propaga su Id.
- PATCH: sin cambio. Ver `## Notas de especificación`.

**Archivos modificados:**

- `services/proveedor_write.go`: verificación en `EscribirProveedor` (solo POST con `TerceroId` suelto), función `verificarTerceroExistente` y constante `motivoTerceroInexistente`.
- `services/proveedor_tercero_inexistente_test.go` (nuevo): pruebas de 400 sin escritura, tercero existente, fallo de terceros sin escritura, `TerceroId` ausente o 0 sin consulta, y objeto `Tercero` sin consulta.

**Verificación:** las pruebas nuevas y el paquete `services` pasan. `go build ./...` pasa. `go test -count=1 ./...` pasa en todos los paquetes salvo `test`, donde falla solo `TestIntegracionConsultaSinFieldsNoEntregaAuditoria` (F04, anterior a este cambio).

**Verificación en vivo (14/11, 14:55):** MID reiniciado con el binario que incluye este cambio (PID 74767, `/tmp/agora_mid_run_f24`, log en `/tmp/agora_mid_run.log`). Respaldo previo: `snapshots/proveedores_antes_f24.sql`. Se ejecutaron solo los casos POST con `TerceroId` suelto:
- P04 (`TerceroId` 999999): 400 con "el TerceroId no existe en el CRUD de terceros". Sin escritura.
- P03 (`TerceroId` 101): 200. Reutiliza el proveedor 1 y no crea otro. Coincide con la matriz.
- P05 (`TerceroId` null): 200 y crea el proveedor 3, pero con `tercero_id` 0, no NULL. La matriz espera NULL. Es el comportamiento de F20, no de este cambio.

Después se restauró el respaldo y los conteos volvieron a la línea base (proveedor 2, natural 1, jurídico 1, contacto 1, cuenta 1). `escritura/resultados.md` no se reemplazó, porque esta no fue una corrida completa de la matriz.

## Notas de especificación

- Conflicto con `API.md:197`: "distinto en una entidad nueva se canoniza (206)". Con `TerceroId` suelto no hay un valor autoritativo distinto al que canonizar, así que el sistema responde 400 si el tercero no existe. Texto propuesto para la especificación: "WHEN el alta trae `TerceroId` mayor que cero sin objeto `Tercero` y terceros no lo conoce, THEN el sistema responde 400 en `TerceroId` y no escribe". Pendiente de aprobación.
- Pendiente: un PATCH que asigna un `TerceroId` suelto a un registro sin tercero no se verifica contra terceros.
- Riesgo verificado: cada alta con `TerceroId` suelto depende de la disponibilidad del host de terceros. Un fallo de ese host devuelve 502 o 504 y no escribe.

## Estado (14/11, 15:25)

Queda abierto. La decisión pendiente depende de la guarda de terceros de F17: la escritura a terceros requiere retirarla en un entorno local. El PATCH sin verificar sigue pendiente.
