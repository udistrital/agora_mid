# Proposal

## Why

El control de escritura por relación está declarado pero inerte: la política de
verbos por relación (`creatable` / `updatable` / `deletable`) existe en la
configuración y se fija en pruebas, pero el motor de escritura no la consulta en
ninguna parte, de modo que hoy nada impide que una arista cree o actualice una
entidad que su configuración no debería permitir, ni impide reasociar un hijo
existente a otro padre. Este cambio le da "dientes" al control por relación
sobre el árbol efectivo compuesto que introduce `config-composicion-objetos-tipados`.

## What Changes

- Se hace efectiva la **política de escritura por relación** sobre la acción
  materializada de cada arista del árbol efectivo: `creatable: false` impide que
  la arista **cree** una entidad nueva, `updatable: false` impide que
  **actualice** una existente y `deletable: false` impide que **elimine** una
  entidad. La acción (crear / actualizar / eliminar) se decide por resolución de
  la entidad y por lo que el árbol compuesto marca, no por el verbo HTTP del
  endpoint. El indicador `deletable` deja de estar declarado-no-consultado y pasa
  a ser un control vivo: el motor gana una **ruta de materialización de borrado
  dirigida por el árbol compuesto** (el servicio llamante marca un nodo para
  eliminación y el motor la materializa contra el CRUD dueño), sin introducir
  ningún endpoint DELETE en controladores.
- Se introduce la regla de **no-reparent / pertenencia**: una entidad con `Id`
  propio cuya llave propagada implicaría reasociarla a un padre distinto del que
  la posee se rechaza con `400` antes de cualquier escritura. Esto generaliza el
  escenario vigente "Entidad existente con relación incompatible" a una regla de
  pertenencia explícita.
- Los tres controles se evalúan en una **fase de validación previa sobre el árbol
  completo**: el motor resuelve (solo lectura) y retiene el estado de cada nodo,
  clasifica la acción prevista y gatea toda la cascada antes de materializar
  cualquier escritura. Cualquier violación en cualquier punto del árbol produce
  `400` sin que se haya enviado ninguna creación, actualización ni eliminación a
  los CRUD dueños. Esto extiende la cobertura del requisito vigente "Validación
  antes de cualquier escritura" a los controles nuevos.
- La **identidad del error** es por fragmento: nombra el objeto o la arista
  responsable, coherente con la identidad por fragmento que establece
  `config-composicion-objetos-tipados`.

Alcance explícito: este cambio aporta exactamente estos controles (política por
relación para crear/actualizar/eliminar y no-reparent) y nada más. El motor nunca
infiere una eliminación: solo materializa la baja de un nodo que el árbol compuesto
marca para ello. No modifica el contrato de respuesta ni el manejo de fallo parcial
*durante* la materialización (eso es un cambio posterior): la promesa aquí es que la
**validación** es total y precede a **toda** escritura, no que la materialización
sea transaccional.

## Capabilities

### New Capabilities
<!-- Ninguna. -->

### Modified Capabilities
- `declarative-write-engine`: Se añaden los controles de escritura por relación
  (enforcement de la política de verbos para crear/actualizar/eliminar; regla de
  no-reparent / pertenencia de hijos existentes; materialización de borrado
  dirigida por el árbol compuesto) y se extiende el alcance de la validación previa
  a toda escritura para cubrirlos, con identidad de error por fragmento. Opera
  sobre el árbol efectivo compuesto por `config-composicion-objetos-tipados`.

## Impact

- **Dependencia de planeación**: se apila sobre
  `config-composicion-objetos-tipados` (Cambio 0). El delta de este cambio se
  escribe sobre el texto ya reformado por 0 (árbol efectivo compuesto, roles
  tipados `writable`/`immutable`, `policy` declarada en la arista/referencia), no
  sobre el texto del spec previo a 0.
- **Código** (fase apply, no aquí): `helper/write_engine.go` introduce un pre-pass
  de datos por petición que resuelve y retiene el estado de cada nodo de todo el
  árbol en un único recorrido de solo lectura antes de la primera escritura
  —unificando en él el camino `fk_inbound` hoy separado— y sobre ese estado
  retenido consulta la política de la arista (incluido `deletable`), aplica la regla
  de no-reparent y materializa la baja de un nodo marcado por el árbol compuesto; la
  baja usa la ruta de escritura existente, sin un endpoint DELETE de controlador. La
  recuperación previa del `WriteResolver` se amplía para devolver los campos del
  registro existente (no solo su `Id`), con cambio acompañante en
  `services/write_resolver.go`. `config/relaciones.go` conserva `WriteVerbPolicy`
  tal cual (incluido `Deletable`, ahora consultado).
- **Comportamiento observable**: nuevas respuestas `400` deterministas (antes de
  cualquier escritura) cuando una arista intenta una acción no permitida por su
  política (crear/actualizar/eliminar) o cuando se intenta reparentar un hijo
  existente. Ningún otro contrato HTTP cambia; no se añade ningún endpoint DELETE.
- **Configuración**: ninguna forma nueva; se consume la `policy` por arista que ya
  define `config-composicion-objetos-tipados`.
