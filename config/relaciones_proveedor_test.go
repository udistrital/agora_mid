package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// localizarArchivoProveedor devuelve la ruta del archivo de configuración real
// del proveedor dentro del árbol de config, en su subdirectorio por CRUD.
func localizarArchivoProveedor(t *testing.T) string {
	t.Helper()
	rutas, err := descubrirArchivos(".", "relaciones_proveedor")
	if err != nil {
		t.Fatal(err)
	}
	if len(rutas) != 1 {
		t.Fatalf("se esperaba un único relaciones_proveedor.yaml bajo config, hay %d: %v", len(rutas), rutas)
	}
	return rutas[0]
}

// archivosProveedor devuelve los archivos de configuración del proveedor: el
// archivo de relaciones indicado y las definiciones de objeto de todo el árbol real.
// Así las referencias del proveedor pueden resolverse.
func archivosProveedor(t *testing.T, relaciones string) map[string]string {
	t.Helper()
	rutas, err := descubrirArchivos(".", "objetos_")
	if err != nil {
		t.Fatal(err)
	}
	if len(rutas) == 0 {
		t.Fatal("no se encontraron definiciones de objeto bajo config/")
	}
	archivos := map[string]string{"relaciones_proveedor.yaml": relaciones}
	for _, ruta := range rutas {
		contenido, err := os.ReadFile(ruta)
		if err != nil {
			t.Fatal(err)
		}
		archivos[filepath.Base(ruta)] = string(contenido)
	}
	return archivos
}

// El archivo real relaciones_proveedor.yaml se carga y valida sin error, y su
// árbol del núcleo se interpreta correctamente.
func TestRelacionesProveedorCargaYValida(t *testing.T) {
	contenido, err := os.ReadFile(localizarArchivoProveedor(t))
	if err != nil {
		t.Fatal(err)
	}
	comp, err := escribirComposicion(t, archivosProveedor(t, string(contenido)))
	if err != nil {
		t.Fatalf("relaciones_proveedor.yaml debe cargar y validar sin error: %v", err)
	}

	sc, ok := comp.Servicios["proveedor"]
	if !ok {
		t.Fatalf("no se cargó el servicio proveedor; servicios=%+v", comp.Servicios)
	}
	if sc.MaxDepth <= 0 || sc.MaxRelatedIDs <= 0 {
		t.Errorf("max_depth y max_related_ids deben ser positivos: %+v", sc)
	}
	if sc.OnRelationError != OnRelationErrorNotify && sc.OnRelationError != OnRelationErrorFail {
		t.Errorf("on_relation_error inválido: %q", sc.OnRelationError)
	}

	// Deben existir las relaciones del núcleo en el nivel superior.
	esperadas := map[string]bool{
		"Tercero": false, "ProveedorNatural": false, "ProveedorJuridico": false,
		"Contacto": false, "CuentaBancaria": false,
	}
	for _, rel := range sc.Relations {
		if _, ok := esperadas[rel.MappedName]; ok {
			esperadas[rel.MappedName] = true
		}
	}
	for nombre, presente := range esperadas {
		if !presente {
			t.Errorf("falta la relación del núcleo %q en relaciones_proveedor.yaml", nombre)
		}
	}
}

// Verifica que el recurso raíz del proveedor declare las columnas de primer
// nivel que pueden utilizarse como filtros de colección.
func TestRelacionesProveedorDeclaraCamposFiltrables(t *testing.T) {
	contenido, err := os.ReadFile(localizarArchivoProveedor(t))
	if err != nil {
		t.Fatal(err)
	}
	comp, err := escribirComposicion(t, archivosProveedor(t, string(contenido)))
	if err != nil {
		t.Fatalf("relaciones_proveedor.yaml debe cargar y validar sin error: %v", err)
	}

	sc := comp.Servicios["proveedor"]
	esperados := []string{"Id", "TerceroId", "TipoRegistroId", "DescripcionPortafolio"}
	if len(sc.FilterableFields) != len(esperados) {
		t.Fatalf("campos filtrables inesperados: got=%v want=%v", sc.FilterableFields, esperados)
	}
	for i, esperado := range esperados {
		if sc.FilterableFields[i] != esperado {
			t.Errorf("campo filtrable %d inesperado: got=%q want=%q", i, sc.FilterableFields[i], esperado)
		}
	}
}

// La configuración del proveedor no nombra implementaciones de código: solo
// declara related_service (nombre lógico) y campos del contrato declarativo; no
// hay un campo que apunte a un struct o función resolver.
func TestRelacionesProveedorNoNombraImplementaciones(t *testing.T) {
	contenido, err := os.ReadFile(localizarArchivoProveedor(t))
	if err != nil {
		t.Fatal(err)
	}
	texto := string(contenido)

	// Claves que implicarían nombrar una implementación de código.
	prohibidas := []string{"resolver:", "resolver_name", "handler:", "struct:", "impl:", "go_type"}
	for _, clave := range prohibidas {
		if strings.Contains(texto, clave) {
			t.Errorf("la configuración no debe nombrar implementaciones; se encontró %q", clave)
		}
	}
}

// Una arista mal declarada en la referencia del proveedor aborta la carga con un
// mensaje que identifica el archivo, el objeto referenciado y el campo afectado.
func TestRelacionesProveedorMalformadaAborta(t *testing.T) {
	contenido, err := os.ReadFile(localizarArchivoProveedor(t))
	if err != nil {
		t.Fatal(err)
	}

	// Corromper la dirección de la referencia a Contacto.
	corrupto := strings.Replace(
		string(contenido),
		"  - objeto: Contacto\n    direction: related_to_local",
		"  - objeto: Contacto\n    direction: hacia_cualquier_lado",
		1,
	)
	if corrupto == string(contenido) {
		t.Fatal("no se pudo corromper la referencia Contacto para la prueba")
	}

	_, err = escribirComposicion(t, archivosProveedor(t, corrupto))
	if err == nil {
		t.Fatal("una arista mal declarada debe abortar la carga")
	}
	msg := err.Error()
	for _, frag := range []string{"relaciones_proveedor.yaml", "Contacto", "direction"} {
		if !strings.Contains(msg, frag) {
			t.Errorf("el mensaje debe contener %q; mensaje: %s", frag, msg)
		}
	}
}
