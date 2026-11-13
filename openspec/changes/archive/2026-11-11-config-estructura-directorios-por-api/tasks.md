# Tasks

## 1. Descubrimiento recursivo en el cargador

- [x] 1.1 Escribir la prueba del cargador (con marcador de rojo) que coloca
  `objetos_*.yaml` y `relaciones_*.yaml` en subdirectorios por CRUD bajo un directorio
  raíz temporal y verifica que `CargarComposicion` reúne todas las definiciones y
  compone el árbol efectivo, y confirmar que falla contra el descubrimiento de un
  nivel actual
- [x] 1.2 Escribir la prueba (con marcador de rojo) que verifica que el orden de
  descubrimiento es determinista: dado un árbol con archivos en varios subdirectorios,
  el conjunto de rutas procesadas es estable y reproducible entre ejecuciones, y
  confirmar que falla
- [x] 1.3 Reemplazar el descubrimiento de `filepath.Glob` de un nivel por un recorrido
  recursivo (`filepath.WalkDir`) en `cargarServicios` y `cargarCatalogoObjetos`,
  filtrando por prefijo (`objetos_`, `relaciones_`) y extensión `.yaml` y conservando
  el `sort.Strings`, y verificar que las pruebas 1.1–1.2 pasan y se retiran sus
  marcadores

## 2. Invariancia e integridad sobre el catálogo reunido

- [x] 2.1 Escribir la prueba (con marcador de rojo) que verifica que mover una
  definición de objeto de un subdirectorio a otro, sin cambiar su contenido, produce
  un árbol efectivo idéntico, y confirmar que falla si el descubrimiento aún no es
  recursivo
- [x] 2.2 Escribir la prueba (con marcador de rojo) que verifica que dos definiciones
  de objeto con el mismo nombre en subdirectorios de CRUD distintos abortan el arranque
  con un mensaje que identifica el nombre duplicado, y confirmar que falla
- [x] 2.3 Verificar que la lógica de unicidad, referencia colgante, ciclo y huérfano
  opera sobre el catálogo reunido desde todos los subdirectorios; ajustar solo lo
  necesario para que las pruebas 2.1–2.2 pasen y retirar sus marcadores

## 3. Traslado de los archivos de configuración y convenciones

- [x] 3.1 Mover los `objetos_*.yaml` y `relaciones_*.yaml` existentes del directorio
  plano a subdirectorios por CRUD bajo el directorio raíz de configuración, y verificar
  que la suite del paquete `config` sigue en verde con la nueva disposición
- [x] 3.2 Trasladar el `README.md` de convenciones al directorio raíz de configuración y
  actualizar su descripción a la disposición por subdirectorios, y verificar que las
  pruebas que validan el README (si existen) quedan coherentes con el contenido movido

## 4. Verificación de integración

- [x] 4.1 Ejecutar `go test ./...` y `go build ./...` y verificar que toda la suite pasa
  sin marcadores de rojo remanentes
