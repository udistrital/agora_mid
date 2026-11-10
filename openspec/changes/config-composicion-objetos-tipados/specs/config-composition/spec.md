# Spec Delta

## Purpose

Define los objetos de configuración como definiciones direccionables de primera
clase con una tabla de campos tipada unificada, permite que un servicio arme su
árbol referenciando esos objetos con semántica de arista propia, y compone y valida
el árbol efectivo por servicio al arranque para que los motores de lectura y
escritura lo consuman ya materializado.

## ADDED Requirements

### Requirement: Objeto de configuración direccionable
El sistema SHALL definir cada entidad como un objeto direccionable de primera clase,
identificado por un nombre único, independiente del servicio que lo use, de modo que
una misma definición de objeto pueda ser referenciada por varios servicios.

#### Scenario: Un objeto definido se referencia por su nombre
- **WHEN** un servicio referencia un objeto por su nombre direccionable
- **THEN** el cargador resuelve la definición de ese objeto y la incorpora al árbol
  efectivo del servicio

#### Scenario: Una definición de objeto sirve a varios servicios
- **WHEN** dos servicios referencian el mismo objeto direccionable
- **THEN** ambos reutilizan la única definición del objeto sin duplicarla, cada uno
  con la semántica de arista de su propia referencia

#### Scenario: Nombre de objeto duplicado impide el arranque
- **WHEN** dos definiciones de objeto declaran el mismo nombre direccionable
- **THEN** el API no inicia y reporta un mensaje que identifica el nombre de objeto
  duplicado

### Requirement: Tabla de campos tipada unificada por objeto
El sistema SHALL declarar los campos de cada objeto en una única tabla por objeto,
donde cada campo lleva su tipo y sus banderas de rol (`readable`, `writable`,
`immutable`, `required`). Esta tabla es la única fuente de verdad de los campos del
objeto para la proyección de lectura y la matriz de escritura.

#### Scenario: Un campo declara tipo y roles en un solo lugar
- **WHEN** un objeto declara un campo con su tipo y sus banderas de rol
- **THEN** la proyección de lectura usa el rol `readable` y la matriz de escritura
  usa los roles `writable` e `immutable` del mismo campo, sin listas separadas

#### Scenario: Campo inmutable no escribible impide el arranque
- **WHEN** un campo declara `immutable: true` pero `writable: false`
- **THEN** el API no inicia y reporta el objeto y el campo incoherente

#### Scenario: Campo requerido no escribible impide el arranque
- **WHEN** un campo declara `required: true` pero `writable: false`
- **THEN** el API no inicia y reporta el objeto y el campo incoherente

#### Scenario: Campo sin tipo impide el arranque
- **WHEN** un campo de un objeto no declara tipo
- **THEN** el API no inicia y reporta el objeto y el campo sin tipo

### Requirement: Vocabulario cerrado de tipos de campo
El sistema SHALL aceptar únicamente los tipos de campo del vocabulario cerrado de
**almacenamiento** `boolean`, `integer`, `jsonb`, `numeric`, `string`, `timestamp` y
`uuid`, donde `string` cubre las columnas de texto del modelo (`varchar`, `text`).
Este tipo es el de almacenamiento y guía la lógica de negocio; cómo se transmite o se
renderiza en documentación no se fija en este cambio. Un tipo fuera de este conjunto
impide el arranque.

#### Scenario: Tipo del vocabulario cerrado se acepta
- **WHEN** un campo declara un tipo del conjunto `boolean`, `integer`, `jsonb`,
  `numeric`, `string`, `timestamp` o `uuid`
- **THEN** el cargador acepta el campo y conserva su tipo en la tabla de campos del
  objeto

#### Scenario: Tipo fuera del vocabulario impide el arranque
- **WHEN** un campo declara un tipo que no pertenece al vocabulario cerrado
- **THEN** el API no inicia y reporta el objeto, el campo y el tipo no reconocido

### Requirement: Semántica de arista en la referencia
El sistema SHALL declarar la semántica de arista (`direction`, `cardinality`,
`key_propagation`, `policy` de escritura, `depth`, `mapped_name`) en la referencia
que un servicio hace a un objeto, y nunca en la definición del objeto, de modo que
un mismo objeto pueda referenciarse con aristas distintas según el servicio.

#### Scenario: La referencia aporta la arista, no el objeto
- **WHEN** un servicio referencia un objeto y declara la semántica de arista en esa
  referencia
- **THEN** el cargador compone el nodo del árbol efectivo con la arista de la
  referencia y los campos de la definición del objeto

#### Scenario: El mismo objeto con aristas distintas en dos servicios
- **WHEN** dos servicios referencian el mismo objeto con semántica de arista
  diferente
- **THEN** cada árbol efectivo refleja la arista de su propia referencia sin que una
  afecte a la otra

#### Scenario: Referencia con arista inválida impide el arranque
- **WHEN** una referencia declara una arista inválida (por ejemplo una propagación de
  llave o cardinalidad no reconocida)
- **THEN** el API no inicia y reporta el servicio, la referencia y el objeto
  afectado

### Requirement: Composición del árbol efectivo al arranque
El sistema SHALL componer y aplanar, al arranque, el árbol efectivo de cada servicio
resolviendo todas sus referencias de objetos en la misma forma de árbol de relaciones
que los motores de lectura y escritura consumen, de modo que los motores reciban un
árbol ya materializado y no resuelvan referencias por sí mismos.

#### Scenario: El árbol efectivo se materializa antes de servir peticiones
- **WHEN** el cargador compone la configuración al arranque
- **THEN** produce para cada servicio un árbol efectivo con las referencias ya
  resueltas y aplanadas, listo para los motores, sin resolución diferida en tiempo de
  petición

#### Scenario: Los motores consumen el árbol ya compuesto
- **WHEN** el motor de lectura o el de escritura procesa una petición
- **THEN** opera sobre el árbol efectivo ya compuesto y no resuelve ninguna
  referencia de objeto por sí mismo

### Requirement: Integridad de la composición al arranque
El sistema SHALL validar la integridad del grafo de composición al arranque y abortar
el inicio ante un ciclo de referencias entre objetos, una referencia colgante o una
profundidad de composición que exceda el límite configurado. La identidad del error
SHALL nombrar el fragmento de objeto o la referencia afectada, no el árbol ensamblado.

#### Scenario: Ciclo de referencias entre objetos impide el arranque
- **WHEN** la composición contiene un ciclo de referencias (un objeto se referencia a
  sí mismo directa o indirectamente a través de otros objetos)
- **THEN** el API no inicia y reporta el fragmento de objeto que cierra el ciclo

#### Scenario: Referencia colgante impide el arranque
- **WHEN** un servicio o un objeto referencia un objeto que no existe
- **THEN** el API no inicia y reporta la referencia colgante nombrando el objeto
  ausente y quién lo referencia

#### Scenario: Profundidad de composición excedida impide el arranque
- **WHEN** la cadena de referencias de un árbol efectivo excede el límite de
  profundidad de composición configurado
- **THEN** el API no inicia y reporta la referencia en la que se excede el límite

#### Scenario: El error de composición nombra el fragmento, no el ensamblado
- **WHEN** la composición falla por integridad
- **THEN** el mensaje identifica el fragmento de objeto o la referencia responsable y
  no describe únicamente el árbol ensamblado final

### Requirement: Objeto definido pero no referenciado
El sistema SHALL detectar al arranque los objetos definidos que ningún servicio
referencia (huérfanos) y reportarlos de forma identificable, sin incorporarlos a
ningún árbol efectivo.

#### Scenario: Objeto huérfano se reporta por su nombre
- **WHEN** un objeto está definido pero ningún servicio ni otro objeto lo referencia
- **THEN** el sistema lo reporta como huérfano nombrando el objeto y no lo incorpora a
  ningún árbol efectivo
