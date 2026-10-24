# Spec Delta

## MODIFIED Requirements

### Requirement: Orden topológico y propagación de llave
El sistema SHALL recorrer el árbol en orden topológico, materializando cada padre
antes que sus hijos y propagando la llave del padre al hijo según el modo de
propagación declarado, preservando el tipo del identificador propagado sin
coaccionarlo a texto.

#### Scenario: Propagación por llave foránea
- **WHEN** una relación declara propagación por FK (por ejemplo `contacto` por
  `proveedor_id`)
- **THEN** el motor materializa primero el padre y fija la FK del hijo con el `Id`
  del padre recién materializado

#### Scenario: Propagación por PK compartida
- **WHEN** una relación declara PK compartida (por ejemplo `proveedor_natural`
  cuyo `Id` es el del `proveedor`)
- **THEN** el motor crea el detalle usando como `Id` el del padre, sin esperar un
  identificador autogenerado distinto

#### Scenario: Propagación por FK ascendente (entidad referenciada por el nodo)
- **WHEN** un nodo referencia a otra entidad por una FK propia (por ejemplo
  `proveedor` referencia a `tercero` por `TerceroId`) y la relación declara
  propagación `fk_inbound`
- **THEN** el motor materializa la entidad referenciada ANTES que el nodo y fija la
  FK del nodo (`TerceroId`) con el `Id` de la entidad recién materializada

#### Scenario: La llave propagada conserva su tipo numérico
- **WHEN** el `Id` del padre materializado es un valor numérico y el motor lo
  propaga a la FK del hijo, al `Id` del detalle con PK compartida o a la FK local
  del nodo en una propagación `fk_inbound`
- **THEN** el valor propagado conserva su tipo numérico tanto en la petición
  enviada al CRUD dueño como en el subárbol devuelto, y nunca se convierte a una
  cadena de texto
