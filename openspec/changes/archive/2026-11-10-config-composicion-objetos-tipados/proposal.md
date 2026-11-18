# Proposal

## Why

Hoy un servicio es un árbol de relaciones: cada entidad existe como una posición
inline dentro de un único árbol raíz, su campo de lectura (`allowed_fields`) y sus
campos de escritura (`writable_fields` / `immutable_fields`) se declaran por
separado en el mismo nodo, y no hay forma de que una entidad definida por un
servicio sea reutilizada por otro. Esa forma impide dos cosas que el middleware ya
necesita: una definición de objeto reutilizable por varios servicios y una única
fuente de verdad tipada por campo que alimente lectura, escritura y la
documentación derivada. Este cambio reforma el modelo de configuración —sin alterar
el comportamiento observable del proveedor— para habilitar todo lo que sigue.

## What Changes

- **BREAKING (configuración)** Los objetos pasan a ser definiciones de primera clase
  direccionables: cada entidad (`contacto`, `cuenta_bancaria`, `proveedor_natural`,
  `tercero`, ...) se define una sola vez y un servicio arma su árbol **referenciando**
  objetos. La semántica de arista (`direction`, `cardinality`, `key_propagation`,
  `policy` de escritura, `depth`, `mapped_name`) vive **en la referencia**, no en el
  objeto, de modo que un mismo objeto puede ser referenciado por N servicios con
  aristas distintas.
- **BREAKING (configuración)** `allowed_fields` (lectura) y `writable_fields` /
  `immutable_fields` (escritura) se unifican en **una sola tabla de campos tipada por
  objeto**: cada campo lleva su `type` y banderas de rol (`readable`, `writable`,
  `immutable`, `required`). Es la única fuente de verdad por objeto que alimenta la
  proyección de lectura, la matriz de escritura y (en un cambio posterior) la
  documentación derivada. El `type` es el tipo de **almacenamiento** y proviene de un
  vocabulario cerrado (`boolean`, `integer`, `jsonb`, `numeric`, `string`,
  `timestamp`, `uuid`); un tipo fuera de ese conjunto impide el arranque. El mapeo de
  ese tipo a su forma de transporte o de documentación no se fija en este cambio.
- Se introduce un **cargador de composición** que resuelve y aplana todas las
  referencias al arranque, produciendo el árbol efectivo por servicio en la misma
  forma que los motores de lectura y escritura ya consumen. Los motores reciben un
  árbol ya materializado y validado; no resuelven referencias por sí mismos.
- La **integridad de la composición** (ciclos de referencia, objetos huérfanos,
  referencias colgantes y la profundidad del grafo de composición) se valida en el
  cargador al arranque, con identidad de error **por fragmento** (nombra el objeto,
  no el árbol ensamblado). El API no inicia ante una composición inválida.
- **Behavior-preserving**: la lectura y la escritura del proveedor son idénticas
  tras la reforma. El árbol efectivo que produce el cargador para el servicio
  `proveedor` equivale al árbol inline actual; los escenarios vigentes de ambos
  motores siguen cumpliéndose sin cambios de comportamiento.

## Capabilities

### New Capabilities
- `config-composition`: Define los objetos de configuración como definiciones
  direccionables de primera clase con una tabla de campos tipada unificada, la
  referencia de objetos por parte de un servicio con semántica de arista propia, el
  cargador que compone y aplana el árbol efectivo por servicio al arranque, y la
  validación de integridad de la composición (ciclos, huérfanos, referencias
  colgantes, profundidad, identidad de error por fragmento).

### Modified Capabilities
- `nested-object-expansion`: Los requisitos que hoy describen `allowed_fields` pasan
  a describir el **rol `readable`** de la tabla de campos tipada unificada definida
  por `config-composition`. La expansión opera sobre el árbol efectivo ya compuesto.
- `declarative-write-engine`: Los requisitos que hoy describen `writable_fields` /
  `immutable_fields` pasan a describir los **roles `writable` / `immutable`** de la
  tabla de campos tipada unificada; la coherencia lectura/escritura se satisface por
  construcción del árbol efectivo compuesto. El motor opera sobre el árbol ya
  aplanado por el cargador.

## Impact

- **Configuración**: el formato de `config/relaciones_*.yaml` cambia de árboles
  inline a definiciones de objeto direccionables + referencias de servicio. La
  configuración del proveedor se reescribe a esta forma preservando su árbol
  efectivo.
- **Código** (se implementa en la fase apply, no aquí): `config/relaciones.go`
  (modelo), `config/relaciones_loader.go` (cargador de composición + validación de
  integridad), y las estructuras que los motores de `helper/expand.go` y
  `helper/write_engine.go` consumen (sin cambio de comportamiento: reciben el árbol
  efectivo ya aplanado).
- **Arranque**: una nueva clase de fallos de arranque (composición inválida) con
  identidad de error por fragmento.
- **Comportamiento observable**: ninguno para el proveedor (lectura y escritura
  idénticas). No se modifica ninguna API HTTP ni contrato de respuesta.
