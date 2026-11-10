# Spec Delta

## ADDED Requirements

### Requirement: Árbol de escritura del modelo extendido del proveedor
El sistema SHALL aceptar y materializar en cascada el árbol del proveedor del
modelo extendido: `proveedor` (con `tipo_proveedor`, `descripcion_portafolio`,
`consentimiento`, `tercero_id`) y sus entidades `estado_proveedor`, `documento`,
`perfil_proveedor`, `titulo_proveedor_natural`, `actividad_economica_proveedor`,
`representacion`, `persona_juridica_tipo_entidad` y las especializaciones
`proveedor_natural` / `proveedor_juridico`, cada una por su CRUD dueño.

#### Scenario: Registro de persona natural
- **WHEN** se envía `POST /v1/proveedor` con `tipo_proveedor` de persona natural y
  un subárbol válido del modelo extendido
- **THEN** el MID crea el proveedor y sus entidades admitidas enviando cada campo a
  su CRUD dueño, y devuelve los identificadores materializados

#### Scenario: Registro de persona jurídica
- **WHEN** se envía `POST /v1/proveedor` con `tipo_proveedor` de persona jurídica y
  un subárbol válido, incluida la rama `persona_juridica_tipo_entidad`
- **THEN** el MID materializa la rama jurídica y sus entidades por su CRUD dueño

#### Scenario: Colección del modelo extendido sin clave natural
- **WHEN** un elemento de una colección del modelo extendido (por ejemplo
  `Documento`, `PerfilProveedor` o `ActividadEconomicaProveedor`) se envía sin `Id`
- **THEN** el elemento se crea; un elemento con `Id` presente se actualiza

### Requirement: Escritura de campos de especialización aplanados
El sistema SHALL aceptar los campos de la especialización (`proveedor_natural` o
`proveedor_juridico`) dentro del cuerpo plano del proveedor y enrutarlos, por PK
compartida, al CRUD dueño de la especialización que corresponde a `tipo_proveedor`,
sin exigir un objeto anidado aparte.

#### Scenario: Campos jurídicos planos se enrutan a su CRUD
- **WHEN** el cuerpo del proveedor es de persona jurídica e incluye campos de
  `proveedor_juridico` (por ejemplo `Pyme`, `RegimenContributivo`) al nivel del
  proveedor
- **THEN** el motor materializa esos campos en el CRUD de `proveedor_juridico`
  usando como `Id` el del proveedor (PK compartida), y los campos propios del
  proveedor en el CRUD de `proveedor`

#### Scenario: Campo de especialización ajeno al tipo se rechaza
- **WHEN** el cuerpo declara `tipo_proveedor` de persona natural pero incluye campos
  de `proveedor_juridico`
- **THEN** el sistema responde `400 Bad Request` identificando el campo ajeno al
  tipo, sin materializar escrituras

### Requirement: Autoría de la modificación por cuerpo
El sistema SHALL aceptar `UsuarioModificacionId` como campo escribible en cada
entidad que tenga campos de auditoría, tomando su valor del cuerpo de la escritura
y propagándolo al CRUD dueño de esa entidad.

#### Scenario: UsuarioModificacionId se toma del cuerpo
- **WHEN** el cuerpo de una entidad con campos de auditoría incluye
  `UsuarioModificacionId`
- **THEN** el MID envía ese valor al CRUD dueño como autor de la modificación de esa
  entidad

### Requirement: Campos de auditoría administrados por el sistema
El sistema SHALL rechazar los campos de auditoría administrados por el sistema
(`FechaCreacion`, `FechaModificacion`, `Activo`) cuando lleguen en el cuerpo, como
campos no escribibles, sin materializar escrituras.

#### Scenario: Campo de auditoría administrado enviado en el cuerpo
- **WHEN** el cuerpo de una entidad incluye `FechaCreacion`, `FechaModificacion` o
  `Activo`
- **THEN** el sistema responde `400 Bad Request` identificando el campo no
  escribible y no materializa ninguna escritura

### Requirement: Transición de estado del proveedor
El sistema SHALL tratar el registro de un `estado_proveedor` como una transición:
si el `estado_id` entrante difiere del estado activo vigente del proveedor, el MID
crea el nuevo estado y marca para baja el estado activo previo; si el `estado_id`
entrante coincide con el activo vigente, la operación es un no-op idempotente. La
baja del estado previo se expresa en el árbol que el servicio entrega al motor.

#### Scenario: Nuevo estado da de baja el activo previo
- **WHEN** se registra un `estado_proveedor` cuyo `estado_id` difiere del estado
  activo vigente del proveedor
- **THEN** el MID crea el nuevo estado y da de baja el estado activo previo por la
  capacidad de eliminación del motor, de modo que a lo sumo un estado queda activo

#### Scenario: Reenvío del estado activo vigente es idempotente
- **WHEN** se registra un `estado_proveedor` cuyo `estado_id` coincide con el estado
  activo vigente del proveedor
- **THEN** el MID no crea un nuevo estado ni da de baja ninguno, conservando el
  estado activo actual

#### Scenario: La nueva transición precede a la baja del estado previo
- **WHEN** una transición de estado crea el nuevo estado y da de baja el previo
- **THEN** el MID crea primero el nuevo estado y da de baja el previo después, de
  modo que un fallo intermedio no deja al proveedor sin estado activo

### Requirement: Referencias de un salto en la escritura
El sistema SHALL aceptar `representacion.representante_id` y
`cuenta_bancaria.contacto_id` como llaves foráneas planas a profundidad 0, sin
materializar ni reconciliar la entidad referenciada.

#### Scenario: Representante como FK plana
- **WHEN** el cuerpo de una `representacion` incluye `RepresentanteId`
- **THEN** el MID lo envía al CRUD dueño como FK, sin crear ni actualizar el tercero
  representante

#### Scenario: Contacto de la cuenta bancaria como FK plana
- **WHEN** el cuerpo de una `cuenta_bancaria` incluye `ContactoId`
- **THEN** el MID lo envía al CRUD dueño como FK, sin crear ni actualizar el
  contacto referenciado
