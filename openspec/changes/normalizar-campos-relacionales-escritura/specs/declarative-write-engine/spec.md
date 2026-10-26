# Spec Delta

## MODIFIED Requirements

### Requirement: Escritura declarativa dirigida por configuración
El sistema SHALL ejecutar la escritura en cascada únicamente a partir de la configuración versionada por servicio cargada al arranque, y nunca a partir de endpoints, tablas o rutas provistos por el cliente. Los campos derivados de una relación se reconocen según esa configuración, aunque no pertenezcan a `writable_fields`.

#### Scenario: La estructura de escritura proviene de la configuración
- **WHEN** se procesa un cuerpo de escritura anidado
- **THEN** el motor usa el árbol de relaciones y las políticas de escritura declaradas en configuración para decidir qué entidades materializar y en qué CRUD, sin aceptar destinos del cliente

#### Scenario: Campo de escritura no permitido se rechaza
- **WHEN** el cuerpo incluye un campo fuera de los campos escribibles declarados para una relación, o intenta cambiar el valor de un campo inmutable que no es un campo derivado reenviado
- **THEN** el sistema responde `400 Bad Request` identificando el campo inválido y no materializa ninguna escritura

#### Scenario: Campo derivado relacional reenviado se reconoce
- **WHEN** el cuerpo incluye un campo derivado por `fk`, `shared_pk` o `fk_inbound` que corresponde a una relación declarada
- **THEN** el motor no lo rechaza como campo desconocido y lo valida contra la llave autoritativa de la relación

#### Scenario: Campo inmutable derivado reenviado coincidente
- **WHEN** el cuerpo reenvía un campo que es a la vez inmutable y derivado de una relación, con el valor que coincide con la llave autoritativa
- **THEN** el sistema no lo rechaza por inmutable y lo concilia como campo derivado, sin materializar un cambio prohibido

### Requirement: Orden topológico y propagación de llave
El sistema SHALL recorrer el árbol en orden topológico, materializando cada padre antes que sus hijos y propagando la llave del padre al hijo según el modo declarado. La llave autoritativa prevalece sobre un valor derivado reenviado y conserva su tipo numérico.

#### Scenario: Propagación por llave foránea
- **WHEN** una relación declara propagación por FK (por ejemplo `contacto` por `proveedor_id`)
- **THEN** el motor materializa primero el padre y fija la FK del hijo con el `Id` del padre recién materializado

#### Scenario: Propagación por PK compartida
- **WHEN** una relación declara PK compartida (por ejemplo `proveedor_natural` cuyo `Id` es el del proveedor)
- **THEN** el motor crea el detalle usando como `Id` el del padre, sin esperar un identificador autogenerado distinto

#### Scenario: Propagación por FK ascendente (entidad referenciada por el nodo)
- **WHEN** un nodo referencia a otra entidad por una FK propia (por ejemplo `proveedor` referencia a `tercero` por `TerceroId`) y la relación declara propagación `fk_inbound`
- **THEN** el motor materializa la entidad referenciada ANTES que el nodo y, en ese pre-pass, compara la FK local reenviada (`TerceroId`) contra el `Id` de la entidad recién materializada antes de fijarla: si el nodo no tiene `Id` propio la fija con el `Id` autoritativo; si el nodo tiene `Id` propio y la FK local reenviada difiere, no la sobrescribe y devuelve un error de cuerpo inválido

#### Scenario: La llave propagada conserva su tipo numérico
- **WHEN** el `Id` del padre materializado es un valor numérico y el motor lo propaga a la FK del hijo, al `Id` del detalle con PK compartida o a la FK local del nodo en una propagación `fk_inbound`
- **THEN** el valor propagado conserva su tipo numérico tanto en la petición enviada al CRUD dueño como en el subárbol devuelto, y nunca se convierte a una cadena de texto

#### Scenario: Campo relacional ausente o nulo
- **WHEN** el campo derivado no está presente o llega como `null`
- **THEN** el motor lo trata como ausente, lo completa con la llave autoritativa y lo envía al CRUD dueño

#### Scenario: Campo relacional reenviado coincide
- **WHEN** el campo derivado está presente y coincide con la llave autoritativa
- **THEN** el motor acepta el cuerpo, conserva la llave y no agrega un aviso de normalización

#### Scenario: Entidad nueva con campo relacional desactualizado
- **WHEN** una entidad sin `Id` propio contiene un campo derivado diferente de la llave autoritativa
- **THEN** el motor no lo trata como conflicto: materializa la entidad con la llave autoritativa por la vía de creación/reconciliación idempotente existente, devuelve el valor canonizado y agrega un aviso no sensible

#### Scenario: Entidad existente con relación incompatible
- **WHEN** una entidad con `Id` propio contiene un campo derivado incompatible con la relación del padre
- **THEN** el motor no la reasigna silenciosamente a otro padre y, por ser una entidad existente, devuelve un error de cuerpo inválido cuando el conflicto no es compatible con la relación existente

#### Scenario: La llave canonizada conserva su tipo numérico
- **WHEN** la llave autoritativa es numérica
- **THEN** el valor conserva su tipo numérico en la petición enviada al CRUD dueño y en el subárbol devuelto
