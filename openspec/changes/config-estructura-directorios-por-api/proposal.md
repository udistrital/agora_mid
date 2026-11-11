# Proposal

## Why

La configuración de relaciones vive hoy en un único directorio plano (`config/`),
donde el cargador descubre los archivos con `objetos_*.yaml` y `relaciones_*.yaml`
por un glob de un solo nivel. Al absorber el modelo extendido, el número de objetos
crece de un puñado a decenas repartidos entre varios CRUD (`proveedores`,
`terceros`, `parametros`, `ubicaciones`), y un único directorio plano con decenas de
`objetos_*.yaml` deja de ser navegable y de reflejar a qué CRUD pertenece cada
objeto. Este cambio reorganiza los archivos en un subdirectorio por CRUD y hace que
el cargador los descubra recorriendo el árbol de directorios, como paso previo e
independiente a la regeneración del contenido desde el modelo.

## What Changes

- **BREAKING (estructura de archivos)** Los archivos de configuración de relaciones
  pasan de un directorio plano a un directorio raíz de configuración con un
  subdirectorio por CRUD (por ejemplo `proveedores/`, `terceros/`), donde cada
  subdirectorio contiene sus `objetos_*.yaml` y, cuando aplique, su
  `relaciones_<servicio>.yaml`. El `README.md` de convenciones vive en el directorio
  raíz de configuración.
- El cargador descubre los archivos de objetos y de servicios **recorriendo
  recursivamente** el árbol del directorio raíz de configuración, en lugar de un
  glob de un solo nivel. El orden de descubrimiento permanece determinista
  (ordenado) para que el reporte de arranque y los errores sean reproducibles.
- Las reglas de validación existentes (nombre de objeto duplicado, referencia
  colgante, ciclo, huérfano) se conservan y ahora consideran todos los archivos del
  árbol, sin importar en qué subdirectorio estén. El nombre de objeto sigue siendo
  único **en todo el catálogo**, no por subdirectorio.
- Sin cambios en el formato de los archivos (`objetos_*.yaml`,
  `relaciones_*.yaml`), en la composición del árbol efectivo, ni en los motores de
  lectura y escritura. La ubicación del archivo no altera el árbol compuesto.

## Capabilities

### New Capabilities
<!-- Ninguna. -->

### Modified Capabilities
- `config-composition`: El descubrimiento de archivos de configuración pasa de un
  directorio plano con glob de un nivel a un directorio raíz con un subdirectorio
  por CRUD, descubierto por recorrido recursivo del árbol de directorios. La
  unicidad del nombre de objeto y las demás validaciones de integridad se mantienen
  sobre el catálogo completo reunido desde todos los subdirectorios.

## Impact

- **Configuración**: los archivos `config/objetos_*.yaml` y
  `config/relaciones_*.yaml` se mueven a subdirectorios por CRUD bajo el directorio
  raíz de configuración; el `README.md` de convenciones se traslada al raíz y
  documenta la nueva disposición.
- **Código** (cargador): `cargarCatalogoObjetos` y `cargarServicios` en
  `config/relaciones_loader.go` cambian su descubrimiento de `filepath.Glob` de un
  nivel a un recorrido recursivo (`filepath.WalkDir`) que reúne los archivos de
  todos los subdirectorios, conservando el orden determinista.
- **Pruebas**: las pruebas del cargador que asumen un directorio plano se actualizan
  a la disposición por subdirectorios; las pruebas de descubrimiento recursivo y de
  colisión de nombres entre subdirectorios se agregan.
- **Comportamiento observable**: ninguno en el contrato del API; el árbol efectivo
  compuesto es idéntico. El cambio es de organización de archivos y de la mecánica
  de descubrimiento, no de la semántica de composición.
- **Alcance**: este cambio no regenera ni reescribe el contenido de la
  configuración desde el modelo; esa regeneración pertenece al cambio
  `absorber-modelo-58-proveedor`.
