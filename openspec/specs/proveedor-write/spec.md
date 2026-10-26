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
