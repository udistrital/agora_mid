# Design

## Context

Ver `proposal.md` (sección Why) para la motivación. El estado actual relevante:

- `helper/write_engine.go` orquesta la escritura en cascada. El pre-pass
  `materializarInbound` materializa la entidad `fk_inbound` (p. ej. `tercero`) y
  recién después llama a `conciliarCampoDerivado` sobre la FK local del nodo
  (`proveedor.TerceroId`). Esa conciliación puede devolver `*ErrorEscritura`
  (mapeado a `400`), pero el `tercero` ya fue escrito.
- Los modos descendentes (`fk`, `shared_pk`) ya concilian antes de escribir:
  `materializarValorRelacion` llama a `llaveDescendente` antes de
  `materializarEntidad`. `fk_inbound` es la única asimetría.
- `services/write_resolver.go` expone `WriteResolver.Materialize`, que internamente
  hace un GET (`resolverExistente`, devuelve solo el `Id`) seguido del POST/PATCH,
  en una sola llamada opaca para el motor.
- `validarNivel` ya aplica `immutable_fields` y `writable_fields` contra el cuerpo
  (gobierno por campo real en ejecución). `policy.*` está declarado pero el motor
  no lo lee (deuda conocida, fuera de alcance).
- Existe un arnés de pruebas (`CRUDSimulado`, `peticionesPorMetodo`) que captura
  peticiones por método y recurso, apto para afirmar "cero escrituras".

## Goals / Non-Goals

**Goals:**
- Que el pre-pass `fk_inbound` concilie y valide contra el estado recuperado ANTES
  de materializar, eliminando la ventana de escritura parcial.
- Separar en el contrato de escritura la recuperación (lectura) de la
  materialización (escritura), para que el motor pueda decidir con información
  autoritativa sin efectos colaterales.
- Dejar el invariante "validar antes de escribir" expresado y cubierto por prueba,
  de modo que aplique a modos futuros.

**Non-Goals:**
- No se aplica `policy.creatable/updatable/deletable` en ejecución (deuda conocida).
- No cambia ninguna firma HTTP del MID ni el contrato de los CRUD dueños.
- No se aborda la validación de inmutables contra el registro remoto más allá de lo
  necesario para `fk_inbound`; la decisión de dominio es que el MID SÍ actualiza un
  `tercero` existente respetando sus `immutable_fields` (que hoy están vacíos para
  `tercero`).

## Decisions

### Decisión 1: Separar "recuperar" de "materializar" en el contrato WriteResolver
El contrato gana una operación de recuperación que resuelve si la entidad existe
(por `Id` del payload o por clave natural) usando únicamente GET, devolviendo lo
necesario para la conciliación (al menos el `Id` autoritativo). `Materialize`
conserva la escritura.

- **Por qué:** el motor necesita el `Id` autoritativo de la entidad referenciada
  para conciliar la FK local del nodo, sin escribirla todavía. El GET ya existe
  dentro de `resolverExistente`; se eleva al contrato en lugar de duplicarlo.
- **Alternativa descartada:** pre-chequeo barato comparando solo `Id`s presentes en
  el cuerpo, sin tocar el CRUD. Se descarta porque no cubre el caso donde la
  entidad referenciada se resuelve por clave natural, dejando una ventana abierta.
- **Alternativa descartada:** mover tal cual la conciliación existente antes de
  materializar. No es posible: hoy compara contra "la entidad recién materializada"
  y ese `Id` no existe aún; por eso se introduce la recuperación por lectura.

### Decisión 2: Reordenar materializarInbound a recuperar -> conciliar -> materializar
El pre-pass primero recupera el `Id` autoritativo de la entidad referenciada, luego
concilia la FK local del nodo (`conciliarCampoDerivado`) y solo si no hay conflicto
materializa la entidad referenciada y propaga su `Id`.

- **Por qué:** alinea `fk_inbound` con los modos descendentes (conciliar antes de
  escribir) y cierra la ventana de escritura parcial.
- **Trade-off:** un GET adicional explícito en el pre-pass. Es la misma lectura que
  `resolverExistente` ya hacía dentro de `Materialize`; no añade un viaje nuevo al
  CRUD en el caso común, lo adelanta.

### Decisión 3: La recuperación honra on_relation_error
Un fallo del GET de recuperación se trata con la misma política que un fallo de
escritura (`manejarErrorRelacion`): aviso en modo tolerante, aborto en modo
estricto o relación bloqueante.

- **Por qué:** evita introducir una ruta de error distinta y mantiene coherente el
  reporte no sensible de errores.

## Risks / Trade-offs

- [Reordenar el pre-pass rompe escenarios que asumían "materializar-luego-comparar"]
  → Las referencias `fk_inbound` en `helper/write_engine_test.go` (líneas ~759,
  916, 943) deben revisarse y ajustarse a la nueva semántica; la prueba de
  canonización de entidad nueva (206) debe seguir pasando porque sin `Id` del nodo
  no hay conflicto y la ruta de creación es la misma.
- [Elevar el GET al contrato cambia la interfaz WriteResolver] → Es un cambio
  interno de un contrato del propio repo; `EntityWriteResolver` ya contiene la
  lógica y solo se reorganiza. No afecta APIs externas.
- [Un GET adicional en casos donde antes no se consultaba] → En el caso común el
  GET ya ocurría dentro de `Materialize`; el riesgo real es solo de orden, no de
  costo adicional.

## Open Questions

Ninguna que afecte specs, enfoque o desglose de tareas. La auditoría de otros
puntos del motor que pudieran escribir antes de una validación con `400` se recoge
como una tarea explícita de verificación, no como pregunta abierta.
