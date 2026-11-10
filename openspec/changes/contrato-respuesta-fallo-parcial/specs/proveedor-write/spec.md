# Spec Delta

## ADDED Requirements

### Requirement: Identidad de reconciliación para el reintento tras 206
El sistema SHALL permitir que un reintento tras una respuesta `206` vuelva a
enlazar cada entidad ya materializada buscándola por su `Id` propio y, en su
defecto, por su clave natural, de modo que el reintento reutilice o actualice la
entidad existente en lugar de crear un duplicado.

#### Scenario: Reintento re-enlaza por Id materializado
- **WHEN** un reintento reenvía el `Data` de un `206` previo, en el que las
  entidades ya materializadas traen su `Id`
- **THEN** el MID re-encuentra cada entidad por su `Id`, la reutiliza o actualiza
  sin recrearla, y continúa materializando solo los nodos pendientes

#### Scenario: Reintento re-enlaza por clave natural
- **WHEN** una entidad materializada no trae `Id` en el cuerpo reenviado pero tiene
  clave natural
- **THEN** el MID la re-encuentra por su clave natural y no crea un duplicado

### Requirement: Idempotencia limitada de colecciones sin clave estable
El sistema SHALL garantizar la idempotencia del reintento sobre los elementos de
una colección que carecen de `Id` y de clave natural únicamente por la vía de
reenvío del `Data` materializado, donde cada elemento ya creado trae su `Id`; el
reenvío del cuerpo original de esa colección puede crear duplicados, y el contrato
lo declara explícitamente sin asumir idempotencia que la colección no soporta.

#### Scenario: Reenvío del Data no duplica elementos sin clave
- **WHEN** un reintento reenvía el `Data` de un `206`, en el que los elementos ya
  creados de una colección sin clave natural traen su `Id`
- **THEN** el MID re-encuentra cada elemento por su `Id` y no crea duplicados

#### Scenario: Reenvío del cuerpo original puede duplicar elementos sin clave
- **WHEN** un reintento reenvía el cuerpo original de una colección cuyos elementos
  no tienen `Id` ni clave natural
- **THEN** el contrato no garantiza idempotencia para esos elementos y el MID puede
  crearlos de nuevo, de acuerdo con la limitación declarada

### Requirement: Identidad de reconciliación de la transición de estado
El sistema SHALL usar `estado_id` como identidad de reconciliación de la transición
de estado del proveedor, de modo que un reintento tras un `206` re-encuentre el
nuevo estado por su `estado_id` y re-intente únicamente la baja pendiente del estado
previo, sin crear un segundo estado.

#### Scenario: Reintento tras transición de estado a medio camino
- **WHEN** un intento previo creó el nuevo `estado_proveedor` pero no alcanzó a dar
  de baja el estado activo previo, y se reintenta la operación
- **THEN** el MID re-encuentra el nuevo estado por su `estado_id` como no-op y
  re-intenta solo la baja pendiente del estado previo, sin crear un segundo estado

### Requirement: Distinción entre ausencia y valor nulo en la escritura
El sistema SHALL distinguir un campo ausente del cuerpo de un campo presente con
valor `null`: un campo ausente conserva el valor existente; un campo escalar
presente con `null` se fija a `null` cuando es escribible y admite nulo, y se
rechaza con `400` cuando no admite nulo; una FK relacional presente con `null`
conserva la regla del motor de tratarla como ausente y derivar la llave
autoritativa.

#### Scenario: Campo ausente conserva su valor
- **WHEN** el cuerpo de una actualización omite un campo escribible de una entidad
- **THEN** el MID conserva el valor existente de ese campo y no lo modifica

#### Scenario: Campo escalar nullable fijado a null
- **WHEN** el cuerpo incluye explícitamente `null` en un campo escalar escribible
  que admite nulo
- **THEN** el MID fija ese campo a `null` en el CRUD dueño

#### Scenario: Campo escalar no nullable con null se rechaza
- **WHEN** el cuerpo incluye explícitamente `null` en un campo escalar escribible
  que no admite nulo
- **THEN** el sistema responde `400 Bad Request` identificando el campo, sin
  materializar escrituras

#### Scenario: FK relacional con null se trata como ausente
- **WHEN** el cuerpo incluye `null` en una FK relacional derivada
- **THEN** el MID la trata como ausente, deriva la llave autoritativa y continúa la
  reconciliación, en coherencia con la regla del motor

### Requirement: La colección vacía no elimina elementos existentes
El sistema SHALL tratar una colección enviada vacía (`[]`) como la ausencia de
elementos afirmados en ese cuerpo y NUNCA como una orden de eliminar los elementos
existentes de esa colección, de modo que vaciar una colección no dispare ninguna
baja que el motor no infiere.

#### Scenario: Colección vacía no dispara eliminaciones
- **WHEN** el cuerpo incluye una colección como `[]`
- **THEN** el MID no elimina ningún elemento existente de esa colección y no
  materializa ninguna baja por ese motivo

#### Scenario: Colección ausente y colección vacía no eliminan
- **WHEN** el cuerpo omite una colección o la envía como `[]`
- **THEN** en ambos casos el MID conserva los elementos existentes y no infiere
  eliminaciones, reservando la baja a los nodos que el árbol marca explícitamente
