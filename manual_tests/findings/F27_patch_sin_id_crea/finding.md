# F27: un PATCH sin Id crea un proveedor nuevo

Estado: aceptado. Reproducido en la corrida 4: U16 respondía 200. Regla de identidad aplicada sin commit (400 sin Id ni TerceroId; 404 si no hay proveedor). Conflicto con `openspec/specs/proveedor-write` (requisito "Reconciliación idempotente por PATCH", que dice "crea lo ausente"): la especificación debe actualizarse o la regla revisarse. Texto propuesto en `## Notas de especificación`.
Solución: aplicada_sin_commit. Ver `## Cambio aplicado`.
Gravedad: alta. Clasificación propuesta, ver utils/README.md.

## Explicación
`PATCH /v1/proveedor` sin `Id` en la raíz crea un proveedor. El contrato no define si debe crear o responder 400. La especificación ("PATCH crea lo ausente") cubre el caso de clave natural, pero no un PATCH sin identificador de recurso.

## Propuesta
Decidir con el dueño del contrato. La propuesta del MID: responder 400 si el PATCH no trae `Id` ni clave natural.

## Cambio aplicado

Regla de identidad del PATCH de la raíz, definida por el usuario el 14/11/2026 (13:45) y aplicada sin commit en `services/proveedor_write.go`:

- PATCH sin `Id` y sin `TerceroId` mayor que cero: 400 (`ErrorEscritura`, motivo "un PATCH requiere Id o TerceroId para identificar el proveedor"). No escribe.
- PATCH con `Id`: el proveedor debe existir por `GET /v1/proveedor/{Id}`. Si el CRUD responde 404, 404. Ver F28.
- PATCH sin `Id` con `TerceroId`: se busca por la clave natural (`TerceroId:{n},Activo:true`, mismo filtro que el motor). Si no hay coincidencias, 404. No escribe.
- POST no cambia: sigue creando.

**Archivos modificados:**

- `services/proveedor_write.go`: `verificarIdentidadPatch`, `verificarProveedorPorTercero`, `baseProveedor`, `motivoPatchSinIdentidad`.
- `services/proveedor_patch_identidad_test.go` (nuevo): pruebas de 400, 404 por clave natural, clave natural existente y POST sin identificador.
- `controllers/proveedor.go`: sin cambios funcionales; el 400 de `ErrorEscritura` ya existía.

**Verificación:** `go test -count=1 ./...` pasa salvo `TestIntegracionConsultaSinFieldsNoEntregaAuditoria` (F04). Verificación en vivo (14/11, MID PID 32858): U16 responde 400. `PATCH {"TerceroId":999,...}` responde 404. U13 responde 404. Conteos de `proveedores` sin cambios.

**Conflicto con la especificación:** `openspec/specs/proveedor-write/spec.md`, requisito "Reconciliación idempotente por PATCH", dice que PATCH "crea lo ausente". La regla nueva contradice esa frase para la raíz: un PATCH con clave natural ausente ya no crea. La especificación debe actualizarse o la regla debe revisarse. Pendiente de decisión del dueño.

## Notas de especificación

**Conflicto.** `openspec/specs/proveedor-write/spec.md`, requisito "Reconciliación idempotente por PATCH", dice: "El sistema SHALL exponer `PATCH /v1/proveedor` que reconcilia el subárbol: crea lo ausente, actualiza lo presente y nunca borra". La regla de identidad de la raíz contradice "crea lo ausente" en un punto: un PATCH cuyo `Id` o `TerceroId` no existe ya no crea. El requisito de hijos ("crea lo ausente" dentro de un proveedor existente) no cambia.

**Decisión aplicada.** Ver la sección `## Cambio aplicado`. Un PATCH de la raíz necesita un identificador válido y existente:
- Sin `Id` mayor que cero ni `TerceroId` mayor que cero: 400, sin escribir.
- `Id` mayor que cero sin proveedor: 404, sin escribir.
- Sin `Id`, `TerceroId` mayor que cero sin proveedor activo: 404, sin escribir.

**Ids y claves en cero.** `Id` 0 o negativo, y `TerceroId` 0 o null, cuentan como ausentes. El código compara `> 0` en todos los casos. Pruebas: `TestEscribirProveedorPatchIdCeroSeTrataComoAusente`, `TestEscribirProveedorPatchIdCeroUsaClaveNatural` y `TestEscribirProveedorPatchTerceroIdCeroDevuelve400`. Estas pruebas pasaron desde la primera ejecución: fijan un comportamiento que ya existía en la implementación, no un requisito nuevo. Un `Id` negativo no tiene prueba propia.

**Texto propuesto para la especificación** (pendiente de aprobación; no se ha escrito en `openspec/`):

> ### Requirement: Reconciliación idempotente por PATCH
> El sistema SHALL exponer `PATCH /v1/proveedor` que reconcilia el subárbol de un proveedor existente. La raíz se identifica por `Id` mayor que cero o, si no hay `Id`, por `TerceroId` mayor que cero. Si ninguno está presente, el sistema responde 400 sin escribir. Si el `Id` o el `TerceroId` no corresponden a un proveedor, responde 404 sin escribir. Dentro de un proveedor identificado, el sistema crea lo ausente, actualiza lo presente y nunca borra.
>
> #### Scenario: PATCH sin identificador
> - **WHEN** el cuerpo no trae `Id` ni `TerceroId` mayor que cero
> - **THEN** el sistema responde 400 y no escribe nada
>
> #### Scenario: PATCH con Id inexistente
> - **WHEN** el cuerpo trae `Id` mayor que cero y ningún proveedor lo tiene
> - **THEN** el sistema responde 404 y no escribe nada
>
> #### Scenario: PATCH con TerceroId sin proveedor
> - **WHEN** el cuerpo no trae `Id` y ningún proveedor activo tiene ese `TerceroId`
> - **THEN** el sistema responde 404 y no escribe nada
>
> #### Scenario: PATCH de proveedor existente con hijos ausentes
> - **WHEN** el proveedor existe y el cuerpo trae un hijo sin `Id`
> - **THEN** el sistema crea el hijo y actualiza los hijos presentes

**Riesgos verificados.**
- La búsqueda por `TerceroId` usa `GET /v1/proveedor?query=TerceroId:{n},Activo:true&limit=1`. El cliente HTTP trata cualquier estado fuera de 2xx como error (`clients/http_client.go`). Un 404 del CRUD por error interno (F07) llega como error de lectura y responde 502, no 404 falso.
- Un cliente que enviaba PATCH con una clave natural nueva ya no crea el proveedor; recibe 404. Es un cambio de contrato visible para ese cliente.
- No hay pruebas existentes que fallen con esta regla. El caso U16 pasa de "por definir" a 400.

**Pendiente para programar.** (1) Aprobar el texto de la especificación o revisar la regla. (2) Actualizar `openspec/specs/proveedor-write/spec.md` con el texto aprobado. (3) Actualizar `API.md` con los códigos 400 y 404 del PATCH.

## Decisión

Aprobado el comportamiento: PATCH sin `Id` ni `TerceroId` responde 400, y `Id` o clave natural sin proveedor responde 404. El texto de `openspec/specs/proveedor-write` y de `API.md` no se aplica desde estas pruebas; se sincroniza en un paso posterior. Sin verificación en vivo en esta revisión.
