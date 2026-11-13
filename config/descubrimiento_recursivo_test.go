package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// escribirArbol crea un directorio raíz temporal con los archivos indicados por su
// ruta relativa, que puede incluir subdirectorios, y devuelve la ruta de la raíz.
func escribirArbol(t *testing.T, archivos map[string]string) string {
	t.Helper()
	raiz := t.TempDir()
	for ruta, contenido := range archivos {
		completa := filepath.Join(raiz, filepath.FromSlash(ruta))
		if err := os.MkdirAll(filepath.Dir(completa), 0o755); err != nil {
			t.Fatalf("no se pudo crear el directorio de %s: %v", ruta, err)
		}
		if err := os.WriteFile(completa, []byte(contenido), 0o644); err != nil {
			t.Fatalf("no se pudo escribir %s: %v", ruta, err)
		}
	}
	return raiz
}

// Prueba que el cargador descubre objetos y servicios repartidos en subdirectorios
// por CRUD a cualquier nivel, los reúne en un único catálogo y compone el árbol
// efectivo del servicio con los objetos de los demás subdirectorios.
func TestCargarComposicion_DescubreArchivosEnSubdirectoriosPorCRUD(t *testing.T) {
	raiz := escribirArbol(t, map[string]string{
		"terceros/objetos_tercero.yaml":              objetoTercero,
		"terceros/objetos_datos_identificacion.yaml": objetoDatosIdentificacion,
		"parametros/objetos_tipo_contribuyente.yaml": objetoTipoContribuyente,
		"proveedores/relaciones_proveedor.yaml":      servicioProveedorComposicion,
	})

	comp, err := CargarComposicion(raiz, urlsPrueba())
	if err != nil {
		t.Fatalf("la disposición por subdirectorios debe componer sin error: %v", err)
	}
	sc, ok := comp.Servicios["proveedor"]
	if !ok {
		t.Fatal("el servicio declarado en proveedores/ debe componerse")
	}
	if len(sc.Relations) != 1 || sc.Relations[0].Name != "Tercero" {
		t.Fatalf("relaciones de primer nivel = %d, se esperaba la relación Tercero", len(sc.Relations))
	}
	anidadas := sc.Relations[0].NestedRelations
	if len(anidadas) != 2 || anidadas[0].Name != "DatosIdentificacion" || anidadas[1].Name != "TipoContribuyente" {
		t.Errorf("relaciones anidadas de Tercero = %+v, se esperaban DatosIdentificacion y TipoContribuyente", anidadas)
	}
	if len(comp.Huerfanos) != 0 {
		t.Errorf("Huerfanos = %v, se esperaba ninguno", comp.Huerfanos)
	}
}

// Prueba que, con dos archivos inválidos en subdirectorios distintos, el error
// nombra siempre el mismo archivo, el primero en orden de ruta, en cada carga
// sobre el mismo árbol. Así el reporte de arranque es reproducible.
func TestCargarComposicion_OrdenDeDescubrimientoDeterminista(t *testing.T) {
	const invalido = "service: x\nmax_depth: 0\non_relation_error: notify\nmax_related_ids: 100\n"
	raiz := escribirArbol(t, map[string]string{
		"z_crud/relaciones_a.yaml": invalido,
		"b_crud/relaciones_b.yaml": invalido,
	})

	primero := ""
	for i := 0; i < 20; i++ {
		_, err := CargarComposicion(raiz, urlsPrueba())
		if err == nil {
			t.Fatal("dos archivos inválidos deben abortar la carga")
		}
		if primero == "" {
			primero = err.Error()
			if !strings.Contains(primero, "b_crud") || strings.Contains(primero, "z_crud") {
				t.Fatalf("el error debe nombrar el primer archivo en orden de ruta (b_crud), obtuvo: %v", err)
			}
			continue
		}
		if err.Error() != primero {
			t.Fatalf("carga %d reporta %q; la primera reportó %q", i, err.Error(), primero)
		}
	}
}

// Prueba que mover un objeto de un subdirectorio a otro, sin cambiar su contenido,
// produce el mismo árbol efectivo y el mismo reporte de huérfanos que la disposición
// de origen y que la disposición plana.
func TestCargarComposicion_MoverObjetoEntreSubdirectoriosNoAlteraArbol(t *testing.T) {
	base := map[string]string{
		"terceros/objetos_tercero.yaml":              objetoTercero,
		"terceros/objetos_datos_identificacion.yaml": objetoDatosIdentificacion,
		"parametros/objetos_tipo_contribuyente.yaml": objetoTipoContribuyente,
		"proveedores/relaciones_proveedor.yaml":      servicioProveedorComposicion,
	}
	movido := map[string]string{
		"proveedores/objetos_tercero.yaml":           objetoTercero,
		"terceros/objetos_datos_identificacion.yaml": objetoDatosIdentificacion,
		"parametros/objetos_tipo_contribuyente.yaml": objetoTipoContribuyente,
		"proveedores/relaciones_proveedor.yaml":      servicioProveedorComposicion,
	}
	plana := map[string]string{
		"objetos_tercero.yaml":              objetoTercero,
		"objetos_datos_identificacion.yaml": objetoDatosIdentificacion,
		"objetos_tipo_contribuyente.yaml":   objetoTipoContribuyente,
		"relaciones_proveedor.yaml":         servicioProveedorComposicion,
	}

	referencia, err := CargarComposicion(escribirArbol(t, base), urlsPrueba())
	if err != nil {
		t.Fatalf("la disposición base debe componer sin error: %v", err)
	}
	if _, ok := referencia.Servicios["proveedor"]; !ok {
		t.Fatal("la disposición base debe producir el servicio proveedor")
	}

	for nombre, archivos := range map[string]map[string]string{"Tercero movido de subdirectorio": movido, "disposición plana": plana} {
		obtenida, err := CargarComposicion(escribirArbol(t, archivos), urlsPrueba())
		if err != nil {
			t.Fatalf("%s debe componer sin error: %v", nombre, err)
		}
		if !reflect.DeepEqual(referencia.Servicios, obtenida.Servicios) {
			t.Errorf("%s: el árbol efectivo cambió al mover el archivo", nombre)
		}
		if !reflect.DeepEqual(referencia.Huerfanos, obtenida.Huerfanos) {
			t.Errorf("%s: huérfanos = %v, se esperaban %v", nombre, obtenida.Huerfanos, referencia.Huerfanos)
		}
	}
}

// Prueba que dos definiciones de objeto con el mismo nombre en subdirectorios de
// CRUD distintos abortan el arranque nombrando el objeto duplicado.
func TestCargarComposicion_NombreDuplicadoEntreSubdirectoriosAborta(t *testing.T) {
	raiz := escribirArbol(t, map[string]string{
		"terceros/objetos_tercero.yaml":              objetoTercero,
		"parametros/objetos_tercero.yaml":            objetoTercero,
		"terceros/objetos_datos_identificacion.yaml": objetoDatosIdentificacion,
		"parametros/objetos_tipo_contribuyente.yaml": objetoTipoContribuyente,
		"proveedores/relaciones_proveedor.yaml":      servicioProveedorComposicion,
	})

	_, err := CargarComposicion(raiz, urlsPrueba())
	if err == nil {
		t.Fatal("un nombre de objeto duplicado entre subdirectorios debe abortar el arranque")
	}
	if !strings.Contains(err.Error(), `"Tercero"`) || !strings.Contains(err.Error(), "duplicado") {
		t.Errorf("el error debe nombrar el objeto duplicado Tercero, obtuvo: %v", err)
	}
}

// Prueba que las validaciones de integridad (ciclo, referencia colgante y huérfano)
// operan sobre el catálogo reunido desde todos los subdirectorios: un ciclo, una
// referencia colgante o un huérfano que cruzan subdirectorios se detectan igual que
// en la disposición plana.
func TestCargarComposicion_IntegridadOperaSobreCatalogoCompleto(t *testing.T) {
	const servicioRefA = "service: proveedor\nmax_depth: 4\non_relation_error: notify\nmax_related_ids: 100\nreferencias:\n  - objeto: A\n    direction: local_to_related\n    local_field: X\n    related_field: Id\n    cardinality: one\n    depth: 1\n    mapped_name: IrA\n"

	t.Run("ciclo que cruza subdirectorios aborta", func(t *testing.T) {
		raiz := escribirArbol(t, map[string]string{
			"terceros/objetos_a.yaml":               "name: A\nrelated_api: terceros\nrelated_table: a\nendpoint: /v1/a\ncampos: []\nreferencias:\n  - objeto: B\n    direction: local_to_related\n    local_field: X\n    related_field: Id\n    cardinality: one\n    depth: 1\n    mapped_name: IrB\n",
			"parametros/objetos_b.yaml":             "name: B\nrelated_api: parametros\nrelated_table: b\nendpoint: /v1/b\ncampos: []\nreferencias:\n  - objeto: A\n    direction: local_to_related\n    local_field: X\n    related_field: Id\n    cardinality: one\n    depth: 1\n    mapped_name: VolverA\n",
			"proveedores/relaciones_proveedor.yaml": servicioRefA,
		})
		_, err := CargarComposicion(raiz, urlsPrueba())
		if err == nil || !strings.Contains(err.Error(), "ciclo de referencias") {
			t.Fatalf("se esperaba aborto por ciclo entre subdirectorios, obtuvo: %v", err)
		}
	})

	t.Run("referencia colgante desde un subdirectorio nombra el objeto ausente", func(t *testing.T) {
		raiz := escribirArbol(t, map[string]string{
			"terceros/objetos_a.yaml":               "name: A\nrelated_api: terceros\nrelated_table: a\nendpoint: /v1/a\ncampos: []\nreferencias:\n  - objeto: NoExiste\n    direction: local_to_related\n    local_field: X\n    related_field: Id\n    cardinality: one\n    depth: 1\n    mapped_name: Roto\n",
			"proveedores/relaciones_proveedor.yaml": servicioRefA,
		})
		_, err := CargarComposicion(raiz, urlsPrueba())
		if err == nil || !strings.Contains(err.Error(), "NoExiste") {
			t.Fatalf("se esperaba aborto nombrando NoExiste, obtuvo: %v", err)
		}
	})

	t.Run("huérfano en un subdirectorio se reporta", func(t *testing.T) {
		raiz := escribirArbol(t, map[string]string{
			"terceros/objetos_tercero.yaml":              objetoTercero,
			"terceros/objetos_datos_identificacion.yaml": objetoDatosIdentificacion,
			"parametros/objetos_tipo_contribuyente.yaml": objetoTipoContribuyente,
			"parametros/objetos_solo.yaml":               "name: Solo\nrelated_api: terceros\nrelated_table: solo\nendpoint: /v1/solo\ncampos: []\n",
			"proveedores/relaciones_proveedor.yaml":      servicioProveedorComposicion,
		})
		comp, err := CargarComposicion(raiz, urlsPrueba())
		if err != nil {
			t.Fatalf("un huérfano no debe abortar el arranque: %v", err)
		}
		if !reflect.DeepEqual(comp.Huerfanos, []string{"Solo"}) {
			t.Errorf("Huerfanos = %v, se esperaba [Solo]", comp.Huerfanos)
		}
	})
}
