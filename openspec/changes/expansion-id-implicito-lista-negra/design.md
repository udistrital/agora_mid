# Design

## Context

Ver proposal.md (Why). El motor de expansión (`helper/expand.go`) resuelve cada
relación declarada y anida el registro bajo su `mapped_name`. Hoy la selección de
campos (`helper/seleccion.go`, `ParsearFields`/`NodoSeleccion`) cumple dos
papeles: valida las rutas de `fields` contra `allowed_fields`/`nested_relations`
(rechazando una ruta no permitida) y, cuando hay `fields`, decide qué relaciones
expandir. El `Id` solo "existe" porque cada `allowed_fields` de
`config/relaciones_proveedor.yaml` lo lista por convención; nada lo garantiza.

Hechos del código relevantes para el diseño:
- `config.RelationConfig` ya expone `RelatedAPI`, `RelatedTable` y `RelatedField`
  (`config/relaciones.go`). La lista negra puede casar por estos campos sin tocar
  los resolvers.
- `helper/expand.go` inserta el registro del resolver tal cual (específico o
  genérico): no hay un punto de recorte de campos escalares sobre la salida.
- Los tipos de aviso declarados son `KindRelation/KindDepth/KindCycle`, pero
  `generic_resolver` se emite vía `AvisoGenericResolver` sin constante hermana;
  `api-response-contract` ya enumera los cuatro.
- `campoPermitido(allowed, campo)` trata `allowed == nil` como "sin restricción".
  La base de nivel superior pasa `allowed = nil`; las relaciones pasan su
  `AllowedFields`.

## Goals / Non-Goals

**Goals:**
- Garantizar el `Id` implícito en expansión sin depender de `allowed_fields`.
- Introducir un único punto de recorte (lista negra) sobre la salida de todo
  resolver, de modo que el genérico también la obedezca.
- Cerrar la deriva de tipos de aviso con un invariante verificable.

**Non-Goals:**
- No se añade sintaxis de exclusión a `fields` (la omisión del `Id` es por no
  mencionarlo).
- No se toca el motor de escritura (`helper/write_engine.go`): la ausencia de `Id`
  en escritura sigue significando "crear".
- No se añade un tipo de aviso para la lista negra: su supresión es silenciosa.
- No se migra el transporte de lectura (eso es el cambio
  `proveedor-read-post-consulta`).

## Decisions

### D1. El `Id` es implícito por normalización de la selección, no por el YAML
La selección acepta la ruta del `Id` de cualquier relación aunque no esté en
`allowed_fields`, y la expansión conserva el `Id` cuando no se recorta por
`fields`. Rationale: cada tabla tiene `Id`; forzar su declaración en cada
`allowed_fields` es la fuente exacta de objetos anónimos. Alternativa descartada:
exigir `Id` en `allowed_fields` y validarlo al arranque — sigue siendo sensible a
olvidos y no resuelve el caso de `fields`.

### D2. El recorte real de escalares se hace en el motor, sobre la salida de cada resolver
El punto de recorte (proyección) vive en `helper/expand.go`, aplicado al registro
devuelto por el resolver antes de anidarlo, sea específico o genérico. Rationale:
el resolver genérico inserta el registro sin transformar; si la lista negra
viviera dentro de los resolvers, el genérico la saltaría — justo la fuga que se
quiere cerrar. Un único choke-point también cubre el `Id` implícito: el `Id`
puede estar presente y ser recortado después por la lista negra.

### D3. La lista negra es un archivo global propio, estructura jerárquica de mapas
`config/global_blacklist.yaml`, cargado al arranque, con la forma:

```yaml
global_blacklist:
  ANY:
    ANY:
      - AuditingField
  terceros:
    tercero:
      - NumeroDocumentoSensible
  gestion_personas_crud:
    contacto:
      - ANY
```

Ejes `related_api -> related_table -> [field]`; `ANY` es la palabra reservada
comodín en cualquier nivel; `- ANY` como único/elemento de la lista de campos
significa "todos los campos de esa tabla". Rationale: mapa plano (no lista de
mapas de una clave) para que YAML imponga unicidad de clave por api/tabla y un
bloque duplicado sea un error del propio YAML. `ANY` como palabra (no `*`) evita
la ambigüedad del sigilo de anclas YAML; los campos son PascalCase y las
tablas/apis minúsculas, así que `ANY` en mayúsculas no colisiona.

### D4. Precedencia: lista negra > allowed_fields > Id implícito
La lista negra gana sobre todo: recorta un campo aunque `allowed_fields` lo
permita y aunque sea el `Id` implícito. Rationale: su doble propósito es (a)
suprimir `Id`/campos sensibles y (b) ser red de seguridad sobre una
`allowed_fields` que incluyó un campo por error.

### D5. Pedir un campo vetado en `fields` = mismo resultado que un campo no permitido
Cuando `fields` nombra explícitamente un campo vetado, la selección lo rechaza con
el mismo resultado observable que una ruta no permitida, reutilizando el rechazo
genérico existente. Rationale: no revelar a un cliente sondeador que el campo
existe o que está vetado. No se emite aviso; no es un camino de error distinto.
Interacción feliz con `proveedor-read-post-consulta`: ese cambio exige nombres
canónicos (una variante no canónica ya se rechaza antes de casar la lista negra),
de modo que la lista negra solo necesita casar cadenas exactas.

### D6. Invariante de tipos de aviso y constante faltante
Se añade la constante `generic_resolver` junto a las existentes y se establece el
invariante "todo `Kind` emitido tiene una constante declarada". Rationale: cierra
la deriva actual y hace el conjunto de tipos verificable por prueba. El conjunto
queda en exactamente cuatro (`relation`, `depth`, `cycle`, `generic_resolver`); la
lista negra no añade ninguno.

### D7. Validación de la lista negra al arranque
El modelo conocido al arranque es la unión de `related_api`, `related_table` y
`allowed_fields` (más `Id` implícito y `related_field`) de las configuraciones de
servicio cargadas. Una entrada con componentes exactos debe existir en ese modelo;
`ANY` no se verifica; `ANY/ANY/ANY` aborta. Rationale: coherente con que una
configuración de relaciones inválida ya impide el arranque. Un comodín es
intencionalmente amplio y no debe fallar por inexistencia.

## Risks / Trade-offs

- [El recorte por lista negra oculta un campo que la configuración declaró como
  permitido, sin aviso] -> Es deliberado (D5, política anti-sondeo). Mitigación de
  higiene de configuración: la validación de arranque y las pruebas documentan qué
  suprime cada entrada; el operador revisa el archivo como frontera de seguridad.
- [`ANY/ANY` o `ANY` de campo demasiado amplios suprimen más de lo esperado] ->
  Mitigación: `ANY/ANY/ANY` prohibido; el archivo lleva comentarios y las entradas
  son explícitas; la revisión del archivo es parte del cambio.
- [Introducir un choke-point de proyección cambia la forma de la salida y puede
  romper pruebas existentes que esperan el registro completo] -> Mitigación: el
  `Id` implícito mantiene identificables los objetos; las pruebas se actualizan
  para el nuevo invariante (Id presente por defecto, campos vetados ausentes).
- [Colisión blanda con `proveedor-read-post-consulta` en la rutina de validación
  de `fields` y en las pruebas del controlador] -> Mitigación: construir primero
  la migración de transporte; nuestro cambio añade su rama de rechazo sobre la ruta
  ya reubicada.

## Migration Plan

1. Añadir la constante `generic_resolver` y el invariante de tipos de aviso
   (cambio aislado, sin efecto observable).
2. Hacer el `Id` seleccionable/implícito en la selección y la expansión.
3. Añadir el cargador y la validación de `config/global_blacklist.yaml` al
   arranque (archivo vacío/ausente = sin supresión).
4. Añadir el choke-point de recorte sobre la salida de cada resolver y conectar la
   lista negra; conectar el rechazo de un campo vetado en `fields`.
5. Opcional: retirar `Id` de las `allowed_fields` de
   `config/relaciones_proveedor.yaml` (ya no es necesario; la retirada es inocua).

Rollback: la lista negra vacía o ausente desactiva la supresión; el `Id` implícito
y el invariante de avisos son cambios de comportamiento acotados reversibles por
revert del motor.
