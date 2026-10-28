# Spec Delta

## ADDED Requirements

### Requirement: Validación antes de cualquier escritura
El sistema SHALL evaluar toda validación capaz de producir `400` —un campo
inmutable incompatible o la conciliación de un campo derivado de relación— ANTES
de ejecutar cualquier escritura de la entidad involucrada, de modo que un `400`
nunca deje escrituras parciales en los CRUD dueños.

#### Scenario: Conflicto de campo derivado no produce escrituras
- **WHEN** una conciliación de campo derivado de relación determina un conflicto de
  entidad existente que debe responder `400`
- **THEN** el motor no envía ninguna petición de creación ni de actualización a los
  CRUD de las entidades involucradas y responde `400`

#### Scenario: Campo inmutable incompatible no produce escrituras
- **WHEN** una entidad existente recuperada trae un campo inmutable con un valor
  distinto del registro autoritativo
- **THEN** el motor responde `400` sin enviar ninguna creación ni actualización a
  ningún CRUD dueño

#### Scenario: Recuperación previa sin efectos de escritura
- **WHEN** el motor recupera el estado de una entidad para validarla antes de
  escribir
- **THEN** la recuperación usa únicamente lecturas (GET) y no modifica ningún
  registro en los CRUD dueños

### Requirement: La recuperación de validación honra la política de error
El sistema SHALL tratar un fallo en la recuperación previa de una entidad bajo la
misma política de error de relación (`on_relation_error`) que un fallo de
escritura, sin introducir una ruta de error distinta.

#### Scenario: Fallo de recuperación en modo tolerante
- **WHEN** la lectura previa de una relación no bloqueante falla y la política es
  tolerante
- **THEN** el motor conserva lo ya materializado, continúa con las demás relaciones
  y agrega un aviso no sensible con la ruta lógica de la relación

#### Scenario: Fallo de recuperación en modo estricto
- **WHEN** la lectura previa de una relación falla y la política es estricta o la
  relación es bloqueante
- **THEN** el motor detiene la cascada y reporta el error sin exponer URLs,
  credenciales ni cuerpos remotos

## MODIFIED Requirements

### Requirement: Orden topológico y propagación de llave
El sistema SHALL recorrer el árbol en orden topológico, materializando cada padre
antes que sus hijos y propagando la llave del padre al hijo según el modo
declarado. La llave autoritativa prevalece sobre un valor derivado reenviado y
conserva su tipo numérico.

#### Scenario: Propagación por llave foránea
- **WHEN** una relación declara propagación por FK (por ejemplo `contacto` por
  `proveedor_id`)
- **THEN** el motor materializa primero el padre y fija la FK del hijo con el `Id`
  del padre recién materializado

#### Scenario: Propagación por PK compartida
- **WHEN** una relación declara PK compartida (por ejemplo `proveedor_natural`
  cuyo `Id` es el del proveedor)
- **THEN** el motor crea el detalle usando como `Id` el del padre, sin esperar un
  identificador autogenerado distinto

#### Scenario: Propagación por FK ascendente (entidad referenciada por el nodo)
- **WHEN** un nodo referencia a otra entidad por una FK propia (por ejemplo
  `proveedor` referencia a `tercero` por `TerceroId`) y la relación declara
  propagación `fk_inbound`
- **THEN** el motor recupera (por lectura) el `Id` autoritativo de la entidad
  referenciada ANTES de materializarla y, en ese pre-pass, compara la FK local
  reenviada (`TerceroId`) contra ese `Id` autoritativo antes de fijarla: si el nodo
  no tiene `Id` propio la fija con el `Id` autoritativo; si el nodo tiene `Id`
  propio y la FK local reenviada difiere, no la sobrescribe, devuelve un error de
  cuerpo inválido y no materializa la entidad referenciada ni el nodo

#### Scenario: La llave propagada conserva su tipo numérico
- **WHEN** el `Id` del padre materializado es un valor numérico y el motor lo
  propaga a la FK del hijo, al `Id` del detalle con PK compartida o a la FK local
  del nodo en una propagación `fk_inbound`
- **THEN** el valor propagado conserva su tipo numérico tanto en la petición
  enviada al CRUD dueño como en el subárbol devuelto, y nunca se convierte a una
  cadena de texto

#### Scenario: Campo relacional ausente o nulo
- **WHEN** el campo derivado no está presente o llega como `null`
- **THEN** el motor lo trata como ausente, lo completa con la llave autoritativa y
  lo envía al CRUD dueño

#### Scenario: Campo relacional reenviado coincide
- **WHEN** el campo derivado está presente y coincide con la llave autoritativa
- **THEN** el motor acepta el cuerpo, conserva la llave y no agrega un aviso de
  normalización

#### Scenario: Entidad nueva con campo relacional desactualizado
- **WHEN** una entidad sin `Id` propio contiene un campo derivado diferente de la
  llave autoritativa
- **THEN** el motor no lo trata como conflicto: materializa la entidad con la llave
  autoritativa por la vía de creación/reconciliación idempotente existente,
  devuelve el valor canonizado y agrega un aviso no sensible

#### Scenario: Entidad existente con relación incompatible
- **WHEN** una entidad con `Id` propio contiene un campo derivado incompatible con
  la relación del padre
- **THEN** el motor no la reasigna silenciosamente a otro padre y, por ser una
  entidad existente, devuelve un error de cuerpo inválido cuando el conflicto no es
  compatible con la relación existente

#### Scenario: La llave canonizada conserva su tipo numérico
- **WHEN** la llave autoritativa es numérica
- **THEN** el valor conserva su tipo numérico en la petición enviada al CRUD dueño
  y en el subárbol devuelto
