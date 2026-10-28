# Proposal

## Why

En el motor de escritura, el pre-pass de FK ascendente (`fk_inbound`) materializa
la entidad referenciada (por ejemplo `tercero`) ANTES de validar que la FK local
reenviada del nodo (`proveedor.TerceroId`) sea compatible. Cuando esa validación
falla en una entidad existente, el MID responde `400`, pero el `tercero` ya pudo
crearse o actualizarse en su CRUD: una escritura parcial que contradice la
semántica de un `400` (cuerpo inválido no debe producir efectos). Los modos
descendentes (`fk`, `shared_pk`) ya concilian antes de escribir; `fk_inbound` es
el único que concilia después. Queremos cerrar esa asimetría y sentar un
invariante general para que el patrón no reaparezca en modos futuros.

## What Changes

- Separar en el contrato `WriteResolver` la **recuperación** (GET: resolver si el
  registro existe y traer su estado) de la **materialización** (POST/PATCH), de
  modo que el motor pueda validar antes de ejecutar cualquier escritura.
- Reordenar el pre-pass `fk_inbound` para: recuperar la entidad referenciada,
  conciliar la FK local del nodo contra el `Id` resuelto y validar los campos
  inmutables contra el estado recuperado, y solo entonces materializar. Un `400`
  por conflicto de campo derivado o por campo inmutable incompatible no produce
  ninguna escritura en ningún CRUD.
- Establecer como invariante del motor que toda validación capaz de producir un
  `400` (campo inmutable o conciliación de campo derivado) se evalúa ANTES de
  cualquier escritura de la entidad involucrada.
- El GET de recuperación honra la política `on_relation_error` igual que una
  escritura fallida (no introduce una ruta de error distinta).

Fuera de alcance (deuda conocida, no se aborda aquí): `policy.creatable/updatable/
deletable` está declarado en configuración pero el motor no lo consulta en
ejecución. El gobierno por verbo queda pendiente para un cambio futuro. El
gobierno por campo (`immutable_fields`, `writable_fields`) sí se aplica hoy y es
la base declarativa que este cambio aprovecha. Decisión del dominio: el MID SÍ
actualiza los campos propios de un `tercero` existente, respetando sus
`immutable_fields`.

## Capabilities

### New Capabilities
<!-- Ninguna. El comportamiento vive en capacidades existentes. -->

### Modified Capabilities
- `declarative-write-engine`: reescribir el escenario de propagación `fk_inbound`
  para conciliar y validar contra la entidad recuperada ANTES de materializarla, y
  añadir el invariante "validar antes de escribir" que prohíbe toda escritura
  cuando una validación produce `400`.
- `proveedor-write`: precisar que un conflicto de FK ascendente (`TerceroId`) o un
  campo inmutable incompatible en una entidad existente responde `400` sin dejar
  escrituras parciales en los CRUD dueños.

## Impact

- Código: `helper/write_engine.go` (`materializarInbound`, orden de conciliación);
  `services/write_resolver.go` (`EntityWriteResolver`, separar recuperar de
  materializar); el contrato `WriteResolver` en `helper/write_engine.go`.
- Pruebas: nueva prueba de integración en `test/` que, ante un `TerceroId`
  incompatible en un proveedor existente, afirma `400` y cero `POST`/`PATCH` tanto
  en `tercero` como en `proveedor` (GET permitidos).
- APIs externas: ninguna firma HTTP del MID cambia; el contrato observable es el
  mismo salvo que un `400` deja de producir efectos colaterales.
- Sin impacto en dependencias ni en la configuración declarativa existente.
