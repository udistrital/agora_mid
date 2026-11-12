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
