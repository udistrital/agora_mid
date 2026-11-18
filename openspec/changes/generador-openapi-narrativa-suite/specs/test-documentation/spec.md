# Spec Delta

## ADDED Requirements

### Requirement: La narrativa cubre el árbol del modelo extendido del proveedor
El sistema SHALL mantener en `test/tests.md` la narrativa de las pruebas del
subárbol del proveedor según el modelo extendido #58, describiendo el registro de
persona natural y de persona jurídica y las entidades del árbol (estado, documento,
perfil, títulos, actividades, representación y la rama jurídica), con sus entradas y
salidas esperadas.

#### Scenario: Narrativa del registro de persona natural
- **WHEN** se consulta `test/tests.md` para el flujo de registro de persona natural
- **THEN** el documento describe el cuerpo anidado del árbol #58, los CRUD dueños
  simulados y los identificadores materializados esperados en la respuesta

#### Scenario: Narrativa del registro de persona jurídica
- **WHEN** se consulta `test/tests.md` para el flujo de registro de persona
  jurídica
- **THEN** el documento describe la rama jurídica del árbol #58 y el rechazo de un
  cuerpo que combine simultáneamente la rama natural y la jurídica

### Requirement: La narrativa cubre el contrato de respuesta y el fallo parcial
El sistema SHALL describir en `test/tests.md` los flujos del contrato de respuesta
del subárbol del proveedor, incluidos el éxito completo, la materialización detenida
a mitad de camino y los códigos de estado diferenciados, con sus avisos esperados en
`MetaData.Errors`.

#### Scenario: Narrativa del stop-abort a mitad de cascada
- **WHEN** la suite incluye una prueba de materialización que se detiene por un
  fallo o un tiempo de espera de una dependencia remota
- **THEN** `test/tests.md` describe el `206` esperado, el subárbol parcial en `Data`
  y la entrada de `MetaData.Errors` con el `Kind` (`dependency_failure` o `timeout`)
  que nombra el punto de parada

#### Scenario: Narrativa del reintento idempotente tras 206
- **WHEN** la suite verifica un reintento que reenvía el `Data` de un `206` previo
- **THEN** `test/tests.md` describe la reconciliación por `Id`/clave natural, la
  ausencia de duplicados y, cuando aplica, la baja pendiente de la transición de
  estado identificada por `estado_id`

### Requirement: La narrativa se mantiene coherente con la fuente del contrato
El sistema SHALL mantener la narrativa coherente con el contrato derivado de la
configuración, de modo que `test/tests.md` describa las entradas y salidas de las
pruebas en términos del árbol efectivo y del contrato publicado vigentes, sin
describir pruebas inexistentes.

#### Scenario: La narrativa no describe pruebas inexistentes
- **WHEN** se agrega, elimina o cambia el comportamiento de una prueba del subárbol
  del proveedor
- **THEN** `test/tests.md` se actualiza para reflejar la prueba vigente y no
  describe ninguna prueba que no exista en la suite
