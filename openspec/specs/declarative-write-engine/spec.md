# declarative-write-engine Specification

## Purpose

Define un motor de escritura declarativo y reutilizable, espejo del de expansión,
que materializa un árbol anidado de entidades contra otros CRUD a partir de
configuración versionada por servicio, sin que el MID sea dueño de los datos.

## Requirements

### Requirement: Escritura declarativa dirigida por configuración
El sistema SHALL ejecutar la escritura en cascada únicamente a partir de la
configuración versionada por servicio cargada al arranque, y nunca a partir de
endpoints, tablas o rutas provistos por el cliente.

#### Scenario: La estructura de escritura proviene de la configuración
- **WHEN** se procesa un cuerpo de escritura anidado
- **THEN** el motor usa el árbol de relaciones y las políticas de escritura
  declaradas en configuración para decidir qué entidades materializar y en qué
  CRUD, sin aceptar destinos del cliente

#### Scenario: Campo de escritura no permitido se rechaza
- **WHEN** el cuerpo incluye un campo fuera de los campos escribibles declarados
  para una relación, o intenta modificar un campo inmutable
- **THEN** el sistema responde `400 Bad Request` identificando el campo inválido y
  no materializa ninguna escritura

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

### Requirement: Aplanado de parámetros anidados a llave foránea
El sistema SHALL aceptar relaciones de parámetro expresadas como objeto anidado
con `Id` y aplanarlas a la llave foránea correspondiente al escribir, de forma
simétrica a la expansión de lectura.

#### Scenario: Objeto de parámetro anidado se aplana a FK
- **WHEN** el cuerpo contiene `{ Tercero: { TipoContribuyente: { Id: 2 } } }`
- **THEN** el motor escribe el tercero con la FK plana equivalente
  (`TipoContribuyenteId: 2`) y no intenta crear el parámetro

### Requirement: Idempotencia por clave natural
El sistema SHALL resolver, antes de crear una entidad con clave natural declarada,
si ya existe por esa clave, reutilizándola en lugar de crear un duplicado.

#### Scenario: Entidad existente por clave natural se reutiliza
- **WHEN** una entidad con clave natural declarada ya existe para los valores del
  cuerpo
- **THEN** el motor reutiliza el registro existente y aplica la actualización en
  vez de crear uno nuevo

#### Scenario: Entidad sin clave natural declarada
- **WHEN** una entidad no declara clave natural y el cuerpo no trae `Id`
- **THEN** el motor crea un nuevo registro y devuelve su `Id` materializado

### Requirement: Reporte tolerante de errores de escritura
El sistema SHALL honrar la política de error de relación configurada durante la
escritura: en modo tolerante conserva lo ya materializado y reporta avisos; en
modo estricto aborta la cascada.

#### Scenario: Fallo de relación tolerado durante la escritura
- **WHEN** una relación no bloqueante falla y la política es tolerante
- **THEN** el motor conserva las entidades ya materializadas, continúa con las
  demás y agrega un aviso no sensible con la ruta lógica de la relación

#### Scenario: Fallo de relación bloqueante
- **WHEN** una relación bloqueante falla o la política es estricta
- **THEN** el motor detiene la cascada y reporta el error sin exponer URLs,
  credenciales ni cuerpos remotos

### Requirement: Coherencia de configuración lectura/escritura
El sistema SHALL validar al arranque que el bloque de escritura de cada servicio
es coherente con el árbol de relaciones de lectura, abortando el inicio ante
inconsistencias.

#### Scenario: Relación de escritura sin contraparte de lectura
- **WHEN** el bloque de escritura declara una relación que no existe en el árbol de
  relaciones del servicio
- **THEN** el API no inicia y reporta un mensaje que identifica el archivo y la
  relación incoherente

#### Scenario: Propagación de llave ausente o inválida
- **WHEN** una relación escribible no declara un modo de propagación de llave
  válido
- **THEN** el API no inicia y reporta el archivo y la relación afectada
