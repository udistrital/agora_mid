# Design

## Context

Ver `proposal.md` (Why). El estado actual: `config/relaciones.go` modela
`ServiceConfig -> []RelationConfig`, donde cada `RelationConfig` es inline, lleva
`NestedRelations` anidadas, y declara por separado `AllowedFields` (lectura) y un
bloque `Write` con `WritableFields` / `ImmutableFields` (escritura). El cargador
`config/relaciones_loader.go` hace glob de `config/relaciones_*.yaml`, un archivo por
servicio, valida cada árbol y rechaza el arranque ante incoherencias. Un servicio es
exactamente un árbol raíz; un subárbol solo se expresa anidando en el mismo archivo
(`NestedRelations`), sin referencia entre servicios. El mismo árbol alimenta el motor
de lectura (`helper/expand.go`) y el de escritura (`helper/write_engine.go`), ambos
genéricos sobre esa estructura.

Las restricciones que dan forma al enfoque: la lectura y la escritura del proveedor no
pueden cambiar de comportamiento; los motores no deben saber de la composición; y la
validación de integridad debe ocurrir una sola vez, al arranque, no por petición.

## Goals / Non-Goals

**Goals:**
- Objetos de configuración direccionables de primera clase, definidos una vez y
  referenciables por N servicios, con la semántica de arista en la referencia.
- Una tabla de campos tipada unificada por objeto (tipo + roles `readable`,
  `writable`, `immutable`, `required`) como única fuente de verdad por objeto.
- Un cargador que compone y **aplana** el árbol efectivo por servicio al arranque,
  en la misma forma que los motores ya consumen, de modo que los motores no cambien.
- Validación de integridad de composición (ciclos, huérfanos, referencias colgantes,
  profundidad de composición) centralizada en el cargador, con identidad de error por
  fragmento.

**Non-Goals:**
- No cambia el comportamiento observable de lectura ni de escritura del proveedor.
- No introduce lógica en `helper/expand.go` ni `helper/write_engine.go` más allá de
  consumir la nueva forma del árbol efectivo (que es equivalente a la actual).
- No define servicios adicionales ni consumidores nuevos de la composición en este
  cambio; la direccionabilidad se ejercita con un segundo referenciador sintético solo
  en pruebas.
- No genera documentación derivada (OpenAPI) a partir de la tabla tipada; la tabla se
  introduce aquí, su consumo documental es un cambio posterior.
- No fija el mapeo del tipo de almacenamiento a su forma de transporte o de
  documentación (por ejemplo OpenAPI `type`+`format`); ese mapeo queda para los
  cambios posteriores que lo consuman.

## Decisions

### Decisión 1: La composición se resuelve y aplana al arranque; los motores reciben un árbol equivalente al actual
El cargador produce, para cada servicio, un árbol efectivo con las referencias ya
resueltas y los campos ya proyectados a las vistas de lectura y escritura que los
motores consumen hoy. La reforma queda **por debajo de la línea de flotación** de
ambos motores: `helper/expand.go` y `helper/write_engine.go` siguen operando sobre la
misma forma de árbol de relaciones.

Por qué: es la vía honesta para preservar comportamiento. Si el árbol efectivo es
equivalente al árbol inline actual, todos los escenarios vigentes de
`nested-object-expansion` y `declarative-write-engine` siguen cumpliéndose sin tocar
los motores. Además, resolver una sola vez al arranque evita recomponer en cada
petición: la composición es trabajo de carga, no de ruta caliente.

Alternativa considerada: resolución perezosa de referencias en tiempo de petición
(los motores siguen referencias al expandir/escribir). Rechazada: acopla los motores
a la composición, dispersa la validación de integridad y repite trabajo por petición.

### Decisión 2: La semántica de arista vive en la referencia, no en el objeto
`direction`, `cardinality`, `key_propagation`, `policy`, `depth` y `mapped_name` se
declaran donde un servicio referencia un objeto. La definición del objeto aporta solo
su identidad, su tabla de campos tipada y sus referencias a otros objetos.

Por qué: un mismo objeto (por ejemplo `contacto`) debe poder participar en árboles
distintos con aristas distintas. Si la arista viviera en el objeto, la definición no
sería reutilizable sin duplicarse, que es justo lo que este cambio elimina.

Alternativa considerada: arista en el objeto con sobreescritura por servicio.
Rechazada: dos fuentes para el mismo dato (objeto + override) reintroducen la
divergencia que la unificación busca eliminar.

### Decisión 3: Tabla de campos tipada unificada como única fuente de verdad por objeto
Se colapsan `allowed_fields` y `writable_fields`/`immutable_fields` en una tabla por
objeto donde cada campo lleva `type` y banderas `readable`/`writable`/`immutable`/
`required`. La vista de lectura es la proyección de los campos `readable`; la matriz
de escritura es la proyección de `writable`/`immutable`. Coherencias imposibles
(`immutable` sin `writable`, `required` sin `writable`, campo sin `type`) abortan el
arranque.

Por qué: dos listas separadas sobre el mismo campo derivan con el tiempo. Una sola
tabla hace imposible por construcción que el modelo de lectura y el de escritura se
contradigan, y añade el tipo que un cambio posterior necesitará para derivar
documentación. Por eso el cambio es foundational: reescribe el texto de los
requisitos de campo de ambos specs en lugar de dejar dos modelos conviviendo.

Alternativa considerada: mantener listas separadas y validar que no diverjan.
Rechazada: la validación cruzada es más frágil que una única fuente y no aporta el
tipo.

El vocabulario de tipos es un conjunto cerrado derivado de los tipos presentes en el
modelo extendido: `boolean`, `integer`, `jsonb`, `numeric`, `string` (unifica
`varchar` y `text`), `timestamp` y `uuid`. Es el tipo de **almacenamiento** y guía la
lógica de negocio; se conserva la distinción `integer`/`numeric` y `uuid`/`timestamp`
aunque en el transporte JSON del CRUD colapsen (números y cadenas respectivamente, y
`jsonb` como objeto). Se cierra aquí para que un tipo desconocido falle al arranque.
El mapeo de cada tipo de almacenamiento a su forma de transporte/documentación
(por ejemplo OpenAPI `type`+`format`) **no se fija en este cambio** y queda para los
cambios posteriores que lo consuman. Asignar el tipo concreto a cada campo del modelo
#58 también es trabajo de esos cambios, no de este.

### Decisión 4: Integridad de composición al arranque, por fragmento; distinta del ciclo de expansión en tiempo de petición
El cargador valida el **grafo de composición** (objeto A referencia B referencia A,
referencia colgante, profundidad de la cadena de referencias) y aborta el arranque
nombrando el fragmento de objeto o la referencia responsable, no el árbol ensamblado.
Esta es la nueva clase de fallo de arranque y el punto único donde se acota la
recursión de referencias.

Esto es **distinto** del ciclo que `nested-object-expansion` ya maneja en tiempo de
petición: aquel es un ciclo del **grafo de datos** durante la expansión (un registro
que apunta de vuelta a través de relaciones entre CRUDs), se detiene por el límite de
profundidad de expansión, no es un error HTTP y registra un aviso de tipo `cycle`. Ese
comportamiento en tiempo de petición no cambia y permanece en
`nested-object-expansion`. No se mueve la detección de ciclos de datos al arranque.

Por qué: separar las dos nociones mantiene la promesa de behavior-preserving (el
límite de profundidad de expansión en ruta caliente sigue intacto) y a la vez da el
"punto único de fallo si la recursión de referencias va demasiado profundo" que el
modelo de objetos direccionables introduce.

## Risks / Trade-offs

- [El árbol efectivo compuesto no es byte-idéntico al inline actual] → La aceptación
  de este cambio es behavior-preserving para el proveedor; las pruebas de equivalencia
  (lectura y escritura) se escriben primero y deben fallar antes de la reforma y pasar
  después, incluyendo el aplanado de la tabla tipada a las vistas de lectura/escritura.
- [La unificación tipada reescribe texto normativo en dos specs existentes] → Las
  modificaciones conservan cada requisito y escenario vigente, cambiando solo la fuente
  de los campos (de `allowed_fields`/`writable_fields` al rol correspondiente de la
  tabla); ningún escenario de comportamiento se elimina.
- [La nueva integridad de composición podría confundirse con el ciclo de expansión] →
  La Decisión 4 fija la distinción explícitamente; las pruebas cubren un ciclo de
  composición (falla el arranque) y un ciclo de datos en expansión (aviso en ruta), por
  separado.
- [La direccionabilidad no tiene un segundo consumidor real en este cambio] → Se
  ejercita con un segundo referenciador sintético solo de prueba que referencia un
  objeto que ya posee el proveedor, verificando aristas independientes, sin introducir
  un servicio de producción.

## Migration Plan

- La configuración del proveedor se reescribe de árbol inline a definiciones de objeto
  + referencias de servicio preservando el árbol efectivo. El despliegue es un cambio de
  formato de configuración más el cargador que lo compone; no hay migración de datos ni
  de contrato HTTP.
- Rollback: revertir el formato de configuración y el cargador restituye el árbol inline
  actual; al no cambiar el comportamiento observable, no hay estado persistido que
  deshacer.
