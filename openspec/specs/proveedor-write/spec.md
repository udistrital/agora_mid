# proveedor-write Specification

## Purpose

Define la escritura del proveedor como recurso raíz mediante cuerpo anidado,
creando y reconciliando en cascada su subárbol del núcleo a través de los CRUD
dueños, sin que el MID sea dueño de los datos.

## Requirements
### Requirement: Creación del subárbol del proveedor
El sistema SHALL exponer `POST /v1/proveedor` que recibe un cuerpo anidado y crea
el subárbol del proveedor en cascada a través de los CRUD dueños, devolviendo los
identificadores materializados en el envoltorio estándar.

#### Scenario: Creación completa exitosa
- **WHEN** se envía un `POST /v1/proveedor` con un subárbol válido y todas las
  entidades se crean correctamente
- **THEN** la respuesta usa el envoltorio plano con `Status: "200"` y `Data`
  incluye los identificadores materializados del proveedor y de sus entidades
  anidadas

#### Scenario: Catálogos y parámetros solo por Id
- **WHEN** el cuerpo referencia una entidad de parámetros o catálogo
  (por ejemplo `TipoContribuyente` o `Perfil`)
- **THEN** el sistema toma únicamente su `Id` y nunca crea ni modifica el catálogo

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

### Requirement: Cascada entre CRUD dueños con tercero no eliminable
El sistema SHALL ejecutar la escritura del subárbol a través de los CRUD dueños
(terceros y gestión de personas), creando y modificando `tercero` y
`datos_identificacion` pero sin eliminarlos.

#### Scenario: Tercero se crea y modifica pero no se elimina
- **WHEN** una operación de escritura requiere crear o actualizar el `tercero`
- **THEN** el sistema lo crea o actualiza en el CRUD de terceros, y ninguna
  operación de escritura elimina un `tercero`

#### Scenario: Reporte de creación parcial
- **WHEN** la cascada falla en una relación tolerada después de materializar otras
- **THEN** el recurso se conserva, la respuesta usa `Status: "206"` con
  `MetaData.Errors` y `Data` incluye los identificadores ya materializados para
  permitir el reintento

### Requirement: Llaves numéricas en el subárbol materializado
El sistema SHALL devolver, en el `Data` materializado de `POST`/`PATCH
/v1/proveedor`, las llaves foráneas y las PK compartidas del subárbol con su tipo
numérico, coherente con el contrato de los CRUD dueños.

#### Scenario: FK devueltas como número
- **WHEN** una escritura materializa el subárbol y devuelve `Data` con la FK
  ascendente del proveedor (`TerceroId`) y las FK descendentes de las colecciones
  (`Contacto.ProveedorId`, `CuentaBancaria.ProveedorId`) o de
  `DatosIdentificacion.TerceroId`
- **THEN** cada una de esas llaves se serializa como número en el `Data`
  devuelto, no como cadena de texto

#### Scenario: PK compartida devuelta como número
- **WHEN** una escritura materializa una rama de detalle con PK compartida
  (`ProveedorNatural` o `ProveedorJuridico`) cuyo `Id` hereda del proveedor
- **THEN** el `Id` heredado se serializa como número en el `Data` devuelto

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
El sistema SHALL aceptar `UsuarioModificacionId` como campo escribible, tomado del
cuerpo, en las entidades cuya tabla de auditoría del DBML declara la columna
`usuario_modificacion_id` (las del CRUD `gestion_personas_crud`, que usan el partial
`campos_auditoria`), propagándolo al CRUD dueño de esa entidad. Las entidades de
`terceros` usan `terceros_campos_auditoria`, que no incluye esa columna, de modo que
`UsuarioModificacionId` no es escribible en ellas. El valor enviado en la raíz se
propaga a cada entidad anidada que declara el campo y no trae un valor propio.

#### Scenario: UsuarioModificacionId se toma del cuerpo
- **WHEN** el cuerpo de una entidad cuyo partial de auditoría declara
  `usuario_modificacion_id` incluye `UsuarioModificacionId`
- **THEN** el MID envía ese valor al CRUD dueño como autor de la modificación de esa
  entidad

#### Scenario: UsuarioModificacionId de la raíz se propaga a entidades anidadas
- **WHEN** el cuerpo envía `UsuarioModificacionId` en la raíz y un objeto anidado
  cuyo partial de auditoría declara la columna no trae un valor propio
- **THEN** el MID envía ese valor como autor de la modificación tanto de la raíz como
  de ese objeto anidado

#### Scenario: Un valor propio de la entidad anidada prevalece
- **WHEN** un objeto anidado trae su propio `UsuarioModificacionId` distinto del de
  la raíz
- **THEN** el MID envía el valor propio del objeto anidado, no el de la raíz

#### Scenario: Una entidad de terceros no acepta UsuarioModificacionId
- **WHEN** el cuerpo de una entidad de `terceros` (por ejemplo `Tercero`) incluye
  `UsuarioModificacionId`
- **THEN** el sistema responde `400 Bad Request` identificando el campo no escribible
  y no materializa ninguna escritura

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
El sistema SHALL resolver el registro de un `estado_proveedor` comparándolo con el estado activo vigente del proveedor, con esta precedencia: si el registro trae el `Id` de un estado existente, el MID lo actualiza en sitio con los campos enviados, sin crear ni dar de baja ningún estado; si no trae `Id` y todos sus campos coinciden con los del estado activo vigente, la operación es un no-op idempotente; si no trae `Id` y algún campo difiere del estado activo vigente, el MID crea el nuevo estado y da de baja el estado activo previo. La baja del estado previo se expresa en el árbol que el servicio entrega al motor.

#### Scenario: Id existente se actualiza en sitio
- **WHEN** se registra un `estado_proveedor` que trae el `Id` del estado activo vigente y una `Observacion` distinta
- **THEN** el MID actualiza ese estado en sitio, sin crear un nuevo estado ni dar de baja ninguno

#### Scenario: Id existente con EstadoId distinto se actualiza en sitio
- **WHEN** se registra un `estado_proveedor` que trae el `Id` del estado activo vigente y un `EstadoId` distinto del actual
- **THEN** el MID actualiza ese mismo registro con el `EstadoId` enviado, sin crear un nuevo estado ni dar de baja ninguno, porque la coincidencia por `Id` tiene precedencia sobre `estado_id`

#### Scenario: Reenvío del estado activo vigente es idempotente
- **WHEN** se registra un `estado_proveedor` sin `Id` cuyos campos coinciden todos con los del estado activo vigente
- **THEN** el MID no crea ni da de baja ningún estado, conservando el estado activo actual

#### Scenario: Sin Id y mismo EstadoId con Observacion distinta crea y da de baja
- **WHEN** se registra un `estado_proveedor` sin `Id` cuyo `EstadoId` coincide con el estado activo vigente pero cuya `Observacion` difiere
- **THEN** el MID crea el nuevo estado y da de baja el estado activo previo, en ese orden

#### Scenario: Nuevo estado da de baja el activo previo
- **WHEN** se registra un `estado_proveedor` sin `Id` cuyo `estado_id` difiere del estado activo vigente del proveedor
- **THEN** el MID crea el nuevo estado y da de baja el estado activo previo por la capacidad de eliminación del motor, de modo que a lo sumo un estado queda activo

#### Scenario: La nueva transición precede a la baja del estado previo
- **WHEN** una transición de estado crea el nuevo estado y da de baja el previo
- **THEN** el MID crea primero el nuevo estado y da de baja el previo después, de modo que un fallo intermedio no deja al proveedor sin estado activo

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
