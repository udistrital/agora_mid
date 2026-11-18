# Design

## Context

Ver `proposal.md` (Why). Estado actual relevante: `config/relaciones_loader.go`
descubre la configuración con dos globs de un solo nivel sobre un directorio `dir`:
`cargarServicios` usa `filepath.Glob(dir, "relaciones_*.yaml")` y
`cargarCatalogoObjetos` usa `filepath.Glob(dir, "objetos_*.yaml")`. Ambos ordenan
las rutas con `sort.Strings` antes de procesarlas, de modo que el orden ya es
determinista. `CargarComposicion(dir, urls)` recibe un único `dir` y reúne servicios
y catálogo desde él. Las validaciones de unicidad de nombre, referencia colgante,
ciclo y huérfano operan sobre el conjunto reunido, no por archivo.

## Goals / Non-Goals

**Goals:**
- Reemplazar el descubrimiento de un nivel por un recorrido recursivo del árbol de
  directorios, conservando el orden determinista y el conjunto de validaciones.
- Mantener invariante el árbol efectivo: mover un archivo entre subdirectorios no
  cambia la composición.
- Mantener la firma pública `CargarComposicion(dir, urls)`: `dir` pasa a ser el
  directorio raíz de configuración que se recorre, no el único nivel que se lista.

**Non-Goals:**
- Regenerar o reescribir el contenido de la configuración (pertenece a
  `absorber-modelo-58-proveedor`).
- Cambiar el formato de `objetos_*.yaml` / `relaciones_*.yaml`, la composición del
  árbol, o los motores de lectura y escritura.
- Introducir un concepto de espacio de nombres por subdirectorio: el nombre de
  objeto sigue siendo único en todo el catálogo.
- Fijar la ubicación física exacta del directorio raíz de configuración como
  contrato; el valor concreto es decisión de implementación mientras el recorrido
  recursivo y las convenciones se respeten.

## Decisions

### Decisión 1: Recorrido recursivo con `filepath.WalkDir`, no globs anidados
El descubrimiento pasa de `filepath.Glob` de un nivel a `filepath.WalkDir` desde el
directorio raíz de configuración, filtrando por los prefijos de nombre existentes
(`objetos_` y `relaciones_`) y la extensión `.yaml`. Las rutas encontradas se
acumulan y se ordenan con `sort.Strings` antes de procesarlas, igual que hoy.

Por qué: `WalkDir` recorre cualquier profundidad de subdirectorios con una sola
pasada y sin fijar el número de niveles. Un glob como `*/objetos_*.yaml` fijaría un
solo nivel de subdirectorio y volvería a romperse si un CRUD agrupa sus objetos en
sub-subcarpetas. `WalkDir` no asume la profundidad.

Alternativa considerada: `filepath.Glob` con patrón de dos niveles
(`*/objetos_*.yaml`). Rechazada: codifica la profundidad en el patrón y reintroduce
la fragilidad que este cambio busca eliminar.

Alternativa considerada: leer un índice declarado (un archivo que liste las rutas).
Rechazada: añade una fuente de verdad que puede desincronizarse del árbol real; el
descubrimiento por recorrido no puede quedar desactualizado.

### Decisión 2: El orden determinista se conserva ordenando las rutas recolectadas
`WalkDir` visita las entradas de cada directorio en orden lexicográfico, pero el
orden entre subdirectorios depende de la estructura. Para no depender de ese detalle,
se recolectan todas las rutas y se ordenan con `sort.Strings` antes de decodificar,
tal como hacen hoy `cargarServicios` y `cargarCatalogoObjetos`.

Por qué: el reporte de arranque, el primer error reportado y el listado de huérfanos
deben ser reproducibles entre máquinas y sistemas de archivos. Ordenar el conjunto
completo de rutas hace el resultado independiente del orden de recorrido.

### Decisión 3: La unicidad y las validaciones operan sobre el catálogo reunido
La detección de nombre de objeto duplicado, referencia colgante, ciclo y huérfano se
mantiene sin cambios de lógica: opera sobre el catálogo y los servicios ya reunidos
desde todos los subdirectorios. Un nombre repetido en dos CRUD distintos es un
duplicado y aborta el arranque.

Por qué: el catálogo es global por diseño (un objeto puede ser referenciado por
varios servicios). Hacer la unicidad por subdirectorio rompería esa propiedad y
permitiría dos objetos con el mismo nombre direccionable, ambiguo para las
referencias.

## Risks / Trade-offs

- [`WalkDir` podría recorrer directorios no relacionados bajo el raíz] → El filtro por
  prefijo de nombre (`objetos_`, `relaciones_`) y extensión `.yaml` descarta todo lo
  demás; un `README.md` u otros archivos se ignoran sin abortar.
- [Mover archivos entre subdirectorios podría cambiar el orden y, con él, el primer
  error reportado] → El ordenamiento del conjunto completo de rutas por
  `sort.Strings` hace que el orden dependa solo de las rutas, no del recorrido del
  sistema de archivos.
- [La disposición por CRUD podría sugerir, erróneamente, un espacio de nombres por
  CRUD] → La prueba de colisión de nombres entre subdirectorios fija la unicidad
  global y documenta que la carpeta es organización, no espacio de nombres.
- [Las pruebas existentes del cargador escriben archivos planos en un directorio
  temporal] → Se actualizan a la disposición por subdirectorios; las que verifican el
  descubrimiento ahora crean subdirectorios y comprueban que el recorrido los reúne.

## Migration Plan

- Configuración: los archivos existentes se trasladan del directorio plano a
  subdirectorios por CRUD bajo el directorio raíz de configuración; el `README.md` de
  convenciones se mueve al raíz y describe la disposición.
- Código: `cargarServicios` y `cargarCatalogoObjetos` cambian su descubrimiento a
  `filepath.WalkDir` conservando el ordenamiento; `CargarComposicion` sigue recibiendo
  el directorio raíz.
- Rollback: restituir la disposición plana y el descubrimiento por `Glob` revierte el
  cambio; no hay estado persistido que deshacer.
