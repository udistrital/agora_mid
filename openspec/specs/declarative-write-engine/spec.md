# declarative-write-engine Specification

## Purpose

Define un motor de escritura declarativo y reutilizable, espejo del de expansión,
que materializa un árbol anidado de entidades contra otros CRUD a partir de
configuración versionada por servicio, sin que el MID sea dueño de los datos.

## Requirements

### Requirement: Escritura declarativa dirigida por configuración
El sistema SHALL ejecutar la escritura en cascada únicamente a partir de la
configuración versionada por servicio cargada al arranque, y nunca a partir de
endpoints, tablas o rutas provistos por el cliente. Los campos derivados de una
relación se reconocen según esa configuración, aunque no pertenezcan a
`writable_fields`.

#### Scenario: La estructura de escritura proviene de la configuración
- **WHEN** se procesa un cuerpo de escritura anidado
- **THEN** el motor usa el árbol de relaciones y las políticas de escritura
  declaradas en configuración para decidir qué entidades materializar y en qué
  CRUD, sin aceptar destinos del cliente

#### Scenario: Campo de escritura no permitido se rechaza
- **WHEN** el cuerpo incluye un campo fuera de los campos escribibles declarados
  para una relación, o intenta cambiar el valor de un campo inmutable que no es un
  campo derivado reenviado
- **THEN** el sistema responde `400 Bad Request` identificando el campo inválido y
  no materializa ninguna escritura

#### Scenario: Campo derivado relacional reenviado se reconoce
- **WHEN** el cuerpo incluye un campo derivado por `fk`, `shared_pk` o `fk_inbound`
  que corresponde a una relación declarada
- **THEN** el motor no lo rechaza como campo desconocido y lo valida contra la
  llave autoritativa de la relación

#### Scenario: Campo inmutable derivado reenviado coincidente
- **WHEN** el cuerpo reenvía un campo que es a la vez inmutable y derivado de una
  relación, con el valor que coincide con la llave autoritativa
- **THEN** el sistema no lo rechaza por inmutable y lo concilia como campo
  derivado, sin materializar un cambio prohibido

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
