# declarative-write-engine Specification

## Purpose

Define un motor de escritura declarativo y reutilizable, espejo del de expansión,
que materializa un árbol anidado de entidades contra otros CRUD a partir de
configuración versionada por servicio, sin que el MID sea dueño de los datos.

## Requirements
### Requirement: Escritura declarativa dirigida por configuración
El sistema SHALL ejecutar la escritura en cascada únicamente a partir del árbol
efectivo compuesto por servicio (ver `config-composition`), cargado y aplanado al
arranque, y nunca a partir de endpoints, tablas o rutas provistos por el cliente. El
motor consume el árbol ya compuesto y no resuelve referencias de objetos por sí
mismo. Los campos que el cliente puede enviar para una entidad son los que la tabla
de campos tipada unificada del objeto marca con el rol `writable`; los campos
derivados de una relación se reconocen según la configuración aunque no lleven el rol
`writable`.

#### Scenario: La estructura de escritura proviene de la configuración
- **WHEN** se procesa un cuerpo de escritura anidado
- **THEN** el motor usa el árbol de relaciones y las políticas de escritura
  declaradas en configuración para decidir qué entidades materializar y en qué
  CRUD, sin aceptar destinos del cliente

#### Scenario: Campo de escritura no permitido se rechaza
- **WHEN** el cuerpo incluye un campo que no lleva el rol `writable` para una
  relación, o intenta cambiar el valor de un campo con rol `immutable` que no es un
  campo derivado reenviado
- **THEN** el sistema responde `400 Bad Request` identificando el campo inválido y
  no materializa ninguna escritura

#### Scenario: Campo derivado relacional reenviado se reconoce
- **WHEN** el cuerpo incluye un campo derivado por `fk`, `shared_pk` o `fk_inbound`
  que corresponde a una relación declarada
- **THEN** el motor no lo rechaza como campo desconocido y lo valida contra la
  llave autoritativa de la relación

#### Scenario: Campo inmutable derivado reenviado coincidente
- **WHEN** el cuerpo reenvía un campo que lleva a la vez el rol `immutable` y es
  derivado de una relación, con el valor que coincide con la llave autoritativa
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
- **THEN** el motor no la reasigna silenciosamente a otro padre y, por aplicación
  de la regla de no-reparent, devuelve un error de cuerpo inválido antes de
  cualquier escritura cuando el conflicto no es compatible con la relación
  existente

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
modo estricto detiene la cascada, y el código y el cuerpo de la respuesta dependen
de si ya se materializó alguna entidad.

#### Scenario: Fallo de relación tolerado durante la escritura
- **WHEN** una relación no bloqueante falla y la política es tolerante
- **THEN** el motor conserva las entidades ya materializadas, continúa con las
  demás y agrega un aviso no sensible con la ruta lógica de la relación

#### Scenario: Fallo de relación bloqueante
- **WHEN** una relación bloqueante falla o la política es estricta, y aún no se ha
  materializado ninguna entidad
- **THEN** el motor detiene la cascada y reporta el error sin exponer URLs,
  credenciales ni cuerpos remotos, sin devolver ningún subárbol parcial

#### Scenario: Fallo estricto después de materializar (stop-abort)
- **WHEN** una relación bloqueante falla o la política es estricta después de que
  el motor materializó al menos una entidad
- **THEN** el motor detiene la cascada, no intenta los nodos posteriores y devuelve
  el subárbol ya materializado con un aviso tipado que nombra el nodo de parada,
  sin exponer URLs, credenciales ni cuerpos remotos

### Requirement: Validación antes de cualquier escritura
El sistema SHALL evaluar toda validación capaz de producir `400` —un campo
inmutable incompatible, la conciliación de un campo derivado de relación, el gate
de la política de escritura por relación (crear/actualizar/eliminar) o la regla de
no-reparent— sobre el árbol completo ANTES de ejecutar cualquier escritura de la
entidad involucrada, de modo que un `400` nunca deje escrituras parciales en los
CRUD dueños. El motor resuelve y retiene el estado de cada nodo en un pre-pass de
solo lectura antes de decidir.

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

#### Scenario: Violación de política o no-reparent no produce escrituras
- **WHEN** una arista del árbol viola su política de escritura (crear con
  `creatable: false`, actualizar con `updatable: false` o eliminar con
  `deletable: false`) o infringe la regla de no-reparent
- **THEN** el motor responde `400` sobre el árbol completo sin enviar ninguna
  creación, actualización ni eliminación a ningún CRUD dueño, aunque la violación
  esté en un nodo posterior al recorrido

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
El sistema SHALL garantizar, por construcción del árbol efectivo compuesto, que los
roles de escritura (`writable`, `immutable`), de lectura (`readable`) y la nulabilidad
de cada campo provienen de la misma tabla de campos tipada por objeto, de modo que no
exista un bloque de escritura que contradiga la contraparte de lectura ni una fuente
de nulabilidad paralela. El cargador de composición SHALL abortar el inicio cuando una
relación escribible del árbol efectivo no declare una propagación de llave válida.

#### Scenario: Relación de escritura sin contraparte de lectura
- **WHEN** un objeto declara sus campos en la tabla de campos tipada unificada
- **THEN** el rol de lectura y los roles de escritura de cada campo provienen de la
  misma definición, sin un bloque de escritura separado que pueda declarar una
  relación escribible inexistente en el árbol de lectura

#### Scenario: Propagación de llave ausente o inválida
- **WHEN** una relación escribible del árbol efectivo no declara un modo de
  propagación de llave válido
- **THEN** el API no inicia y reporta el objeto o la referencia afectada

#### Scenario: La nulabilidad proviene de la tabla de campos tipada
- **WHEN** un objeto declara sus campos en la tabla de campos tipada unificada
- **THEN** la nulabilidad de cada campo proviene de esa misma definición y se proyecta
  al árbol efectivo compuesto, sin una fuente de nulabilidad separada del bloque de
  escritura

### Requirement: Política de escritura por relación
El sistema SHALL honrar la política de verbos declarada en la arista de cada
relación del árbol efectivo, gateando la acción de escritura que materializaría:
`creatable: false` impide crear una entidad nueva por esa arista, `updatable:
false` impide actualizar una entidad existente por esa arista y `deletable: false`
impide eliminar una entidad por esa arista. La acción se determina por la
resolución previa de la entidad y por lo que el árbol compuesto marca, no por el
verbo HTTP del endpoint.

#### Scenario: Creación no permitida por la política de la arista
- **WHEN** una arista con `creatable: false` recibe una entidad que no resuelve a
  ningún registro existente (crearía uno nuevo)
- **THEN** el sistema responde `400 Bad Request` nombrando el objeto o la arista y
  no materializa ninguna escritura

#### Scenario: Actualización no permitida por la política de la arista
- **WHEN** una arista con `updatable: false` recibe una entidad que resuelve a un
  registro existente con cambios en sus campos escribibles
- **THEN** el sistema responde `400 Bad Request` nombrando el objeto o la arista y
  no materializa ninguna escritura

#### Scenario: Acción permitida por la política de la arista
- **WHEN** una arista con `creatable: true` crea una entidad nueva, o una arista
  con `updatable: true` actualiza una existente
- **THEN** el motor materializa la acción con normalidad según el orden topológico

#### Scenario: Eliminación no permitida por la política de la arista
- **WHEN** una arista con `deletable: false` recibe un nodo que el árbol efectivo
  compuesto marca para eliminación
- **THEN** el sistema responde `400 Bad Request` nombrando el objeto o la arista y
  no materializa ninguna escritura

#### Scenario: Eliminación permitida materializa la baja
- **WHEN** una arista con `deletable: true` recibe un nodo que el árbol efectivo
  compuesto marca para eliminación
- **THEN** el motor materializa la baja contra el CRUD dueño del nodo por la ruta
  de escritura existente

### Requirement: Eliminación dirigida por el árbol compuesto
El sistema SHALL materializar la eliminación de una entidad únicamente cuando el
árbol efectivo compuesto marca ese nodo para eliminación, gateada por `deletable`,
y nunca por inferencia propia del motor ni por un verbo HTTP del cliente.

#### Scenario: El motor no infiere eliminaciones
- **WHEN** el árbol efectivo compuesto no marca ningún nodo para eliminación
- **THEN** el motor no elimina ninguna entidad, aunque falten nodos respecto al
  estado actual del CRUD dueño

#### Scenario: Eliminación sin endpoint DELETE de controlador
- **WHEN** el árbol efectivo compuesto marca un nodo para eliminación durante un
  `POST`/`PATCH`
- **THEN** el motor materializa esa baja por la ruta de escritura existente, sin
  requerir un endpoint DELETE propio

### Requirement: No-reparent de hijos existentes
El sistema SHALL rechazar con `400`, antes de cualquier escritura, una entidad con
`Id` propio cuya llave propagada por la arista implicaría reasociarla a un padre
distinto del que la posee, sin reasignarla silenciosamente.

#### Scenario: Entidad existente apuntada a otro padre
- **WHEN** un nodo con `Id` propio llega bajo un padre cuya llave propagada
  difiere de la asociación actual del registro existente (lo reparentaría)
- **THEN** el sistema responde `400 Bad Request` nombrando el objeto o la arista y
  no materializa ninguna escritura

#### Scenario: Entidad existente bajo su padre legítimo
- **WHEN** un nodo con `Id` propio llega bajo el padre que ya lo posee y la llave
  propagada coincide con su asociación actual
- **THEN** el motor no lo trata como reparent y continúa la validación y la
  materialización con normalidad

### Requirement: Rechazo de null en campo escalar no nulable antes de escribir
El sistema SHALL rechazar con `400`, antes de cualquier escritura, un campo escalar
escribible presente con valor `null` cuya nulabilidad en el árbol efectivo compuesto lo
declara no nulable, identificando el campo y sin materializar ninguna escritura. La
nulabilidad usada en la decisión SHALL provenir del árbol efectivo compuesto, no de una
fuente paralela derivada por convención de nombres.

#### Scenario: Null en un escalar no nulable se rechaza sin escribir
- **WHEN** el cuerpo presenta `null` en un campo escalar escribible que el árbol
  efectivo declara no nulable
- **THEN** el sistema responde `400` identificando el campo y no envía ninguna
  escritura a los CRUD dueños

#### Scenario: Null en un escalar nulable no se rechaza por esta validación
- **WHEN** el cuerpo presenta `null` en un campo escalar escribible que el árbol
  efectivo declara nulable
- **THEN** esta validación no rechaza el campo y la escritura continúa

#### Scenario: La decisión usa la nulabilidad del árbol efectivo compuesto
- **WHEN** el motor evalúa un `null` escalar de una entidad anidada del árbol
- **THEN** resuelve la nulabilidad del campo por la tabla de campos del objeto que el
  árbol efectivo asigna a esa entidad, no por el nombre de la clave del cuerpo
