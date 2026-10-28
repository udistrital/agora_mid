# Spec Delta

## MODIFIED Requirements

### Requirement: Reconciliación idempotente por PATCH
El sistema SHALL exponer `PATCH /v1/proveedor` que reconcilia el subárbol: crea lo
ausente, actualiza lo presente y nunca borra, de modo que reenviar el mismo cuerpo
o el `Data` materializado produce el mismo estado final sin exigir eliminar llaves
relacionales.

#### Scenario: Continuación tras creación parcial
- **WHEN** un `POST`/`PATCH` previo dejó el subárbol parcialmente creado y se
  reenvía el mismo cuerpo con `PATCH`
- **THEN** las entidades ya existentes se reutilizan por su clave natural o `Id` y
  solo se crean o actualizan las faltantes, sin duplicar registros

#### Scenario: Colección sin clave natural
- **WHEN** un elemento de una colección (por ejemplo `Contacto` o
  `CuentaBancaria`) se envía sin `Id`
- **THEN** el elemento se crea; un elemento con `Id` presente se actualiza

#### Scenario: Reenvío de Data materializado
- **WHEN** se reenvía un `Data` completo con sus llaves relacionales y los valores
  coinciden con las relaciones autoritativas
- **THEN** el MID acepta el cuerpo, reutiliza o actualiza las entidades por `Id` o
  clave natural y no genera avisos por esos campos

#### Scenario: Reenvío de una FK inmutable que coincide
- **WHEN** un `PATCH` reenvía la FK ascendente del proveedor (`TerceroId`),
  declarada inmutable, con el mismo valor que la relación autoritativa
- **THEN** el MID lo acepta sin tratarlo como una modificación prohibida y responde
  sin avisos por ese campo

#### Scenario: Campo relacional nulo reenviado
- **WHEN** una respuesta reenviada contiene `null` en un campo relacional derivado
- **THEN** el MID lo trata como ausente, calcula la llave correcta y continúa la
  reconciliación

#### Scenario: Campo relacional desactualizado en entidad nueva
- **WHEN** una entidad hija sin `Id` trae una llave relacional distinta de la del
  padre
- **THEN** la entidad se materializa con la llave del padre, `Data` devuelve el
  valor corregido y la respuesta usa `206` con un aviso

#### Scenario: Conflicto relacional en entidad existente
- **WHEN** una entidad hija con `Id` trae una llave que implicaría asociarla a otro
  padre
- **THEN** el MID no cambia silenciosamente la asociación y responde `400` cuando
  no puede demostrar compatibilidad

#### Scenario: Conflicto de FK ascendente en proveedor existente no deja escrituras parciales
- **WHEN** un `PATCH` sobre un proveedor existente reenvía una FK ascendente
  (`TerceroId`) incompatible con el `tercero` resuelto para el cuerpo
- **THEN** el MID responde `400` sin haber enviado ninguna creación ni
  actualización al CRUD de `tercero` ni al de `proveedor`
