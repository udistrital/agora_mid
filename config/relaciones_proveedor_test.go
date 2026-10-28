package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// localizarArchivoProveedor devuelve la ruta del archivo de configuración real
// del proveedor dentro del paquete config.
func localizarArchivoProveedor(t *testing.T) string {
	t.Helper()
	ruta := "relaciones_proveedor.yaml"
	if _, err := os.Stat(ruta); err != nil {
		t.Fatalf("no se encontró relaciones_proveedor.yaml: %v", err)
	}
	return ruta
}

// El archivo real relaciones_proveedor.yaml se carga y valida sin error, y su
// árbol del núcleo se interpreta correctamente.
func TestRelacionesProveedorCargaYValida(t *testing.T) {
	contenido, err := os.ReadFile(localizarArchivoProveedor(t))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	ruta := filepath.Join(dir, "relaciones_proveedor.yaml")
	if err := os.WriteFile(ruta, contenido, 0o600); err != nil {
		t.Fatal(err)
	}

	configs, err := LoadRelaciones(dir, urlsPrueba())
	if err != nil {
		t.Fatalf("relaciones_proveedor.yaml debe cargar y validar sin error: %v", err)
	}

	sc, ok := configs["proveedor"]
	if !ok {
		t.Fatalf("no se cargó el servicio proveedor; configs=%+v", configs)
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
	dir := t.TempDir()
	ruta := filepath.Join(dir, "relaciones_proveedor.yaml")
	if err := os.WriteFile(ruta, contenido, 0o600); err != nil {
		t.Fatal(err)
	}

	configs, err := LoadRelaciones(dir, urlsPrueba())
	if err != nil {
		t.Fatalf("relaciones_proveedor.yaml debe cargar y validar sin error: %v", err)
	}

	sc := configs["proveedor"]
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

// Una relación mal declarada en el archivo del proveedor aborta la carga con un
// mensaje que identifica el archivo y la relación afectada.
func TestRelacionesProveedorMalformadaAborta(t *testing.T) {
	contenido, err := os.ReadFile(localizarArchivoProveedor(t))
	if err != nil {
		t.Fatal(err)
	}

	// Corromper la dirección de la relación Contacto.
	corrupto := strings.Replace(
		string(contenido),
		"direction: related_to_local\n    local_field: Id\n    related_field: ProveedorId\n    cardinality: many\n    depth: 1\n    mapped_name: Contacto",
		"direction: hacia_cualquier_lado\n    local_field: Id\n    related_field: ProveedorId\n    cardinality: many\n    depth: 1\n    mapped_name: Contacto",
		1,
	)
	if corrupto == string(contenido) {
		t.Fatal("no se pudo corromper la relación Contacto para la prueba")
	}

	dir := t.TempDir()
	ruta := filepath.Join(dir, "relaciones_proveedor.yaml")
	if err := os.WriteFile(ruta, []byte(corrupto), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err = LoadRelaciones(dir, urlsPrueba())
	if err == nil {
		t.Fatal("una relación mal declarada debe abortar la carga")
	}
	msg := err.Error()
	for _, frag := range []string{"relaciones_proveedor.yaml", "Contacto", "direction"} {
		if !strings.Contains(msg, frag) {
			t.Errorf("el mensaje debe contener %q; mensaje: %s", frag, msg)
		}
	}
}
