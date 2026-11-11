# Spec Delta

## ADDED Requirements

### Requirement: Disposición de archivos de configuración por CRUD
El sistema SHALL organizar los archivos de configuración de relaciones en un
directorio raíz de configuración con un subdirectorio por CRUD, donde cada
subdirectorio contiene las definiciones de objeto (`objetos_*.yaml`) y, cuando el
CRUD expone un servicio, su archivo de servicio (`relaciones_<servicio>.yaml`). La
ubicación de un archivo en un subdirectorio u otro NUNCA SHALL alterar el árbol
efectivo compuesto.

#### Scenario: Objetos agrupados por CRUD se cargan al catálogo único
- **WHEN** las definiciones de objeto están repartidas en subdirectorios por CRUD
  bajo el directorio raíz de configuración
- **THEN** el cargador reúne todas las definiciones en un único catálogo y compone
  el árbol efectivo igual que si estuvieran en un solo directorio

#### Scenario: La ubicación del archivo no altera el árbol compuesto
- **WHEN** un objeto se mueve de un subdirectorio a otro sin cambiar su contenido
- **THEN** el árbol efectivo compuesto es idéntico, porque la pertenencia a un CRUD
  es organización de archivos y no semántica de composición

### Requirement: Descubrimiento recursivo de archivos de configuración
El sistema SHALL descubrir los archivos de objetos y de servicios recorriendo
recursivamente el árbol del directorio raíz de configuración, en un orden
determinista, de modo que el catálogo y el reporte de arranque sean reproducibles
sin importar la profundidad de los subdirectorios.

#### Scenario: Archivos en subdirectorios se descubren por recorrido recursivo
- **WHEN** los archivos de objetos y de servicios residen en subdirectorios del
  directorio raíz de configuración
- **THEN** el cargador los descubre recorriendo el árbol de directorios y los
  incorpora todos, sin depender de un glob de un solo nivel

#### Scenario: El orden de descubrimiento es determinista
- **WHEN** el cargador recorre el árbol de directorios de configuración
- **THEN** procesa los archivos en un orden estable y reproducible entre arranques,
  de modo que los errores y el reporte de huérfanos no varían por el orden del
  sistema de archivos

### Requirement: Unicidad del nombre de objeto sobre el catálogo completo
El sistema SHALL exigir que el nombre direccionable de un objeto sea único en todo
el catálogo reunido desde todos los subdirectorios, y SHALL abortar el arranque ante
un nombre duplicado aunque las definiciones vivan en subdirectorios de CRUD
distintos. El mensaje SHALL identificar el nombre de objeto duplicado.

#### Scenario: Nombre duplicado entre subdirectorios impide el arranque
- **WHEN** dos definiciones de objeto en subdirectorios de CRUD distintos declaran
  el mismo nombre direccionable
- **THEN** el API no inicia y reporta el nombre de objeto duplicado, porque la
  unicidad se evalúa sobre el catálogo completo y no por subdirectorio
