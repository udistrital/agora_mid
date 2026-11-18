package config

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// bloquesYAMLReadme devuelve el contenido de cada bloque ```yaml del README del
// directorio config, en orden de aparición.
func bloquesYAMLReadme(t *testing.T) []string {
	t.Helper()
	contenido, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("no se pudo leer config/README.md: %v", err)
	}
	coincidencias := regexp.MustCompile("(?s)```yaml\n(.*?)```").FindAllStringSubmatch(string(contenido), -1)
	bloques := make([]string, 0, len(coincidencias))
	for _, c := range coincidencias {
		bloques = append(bloques, c[1])
	}
	if len(bloques) == 0 {
		t.Fatal("config/README.md no contiene bloques yaml")
	}
	return bloques
}

// Cada ejemplo YAML del README se parsea sin error con el decodificador del
// cargador: los objetos con DecodificarObjetoDireccionable y los servicios con la
// decodificación estricta de relaciones.
func TestReadmeCadaEjemploYAMLSeParsea(t *testing.T) {
	for i, bloque := range bloquesYAMLReadme(t) {
		if strings.HasPrefix(strings.TrimSpace(bloque), "name:") {
			if _, err := DecodificarObjetoDireccionable([]byte(bloque)); err != nil {
				t.Errorf("ejemplo %d (objeto) del README no se parsea: %v", i, err)
			}
			continue
		}
		ruta := filepath.Join(t.TempDir(), "relaciones_ejemplo.yaml")
		if err := os.WriteFile(ruta, []byte(bloque), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := cargarArchivoRelaciones(ruta); err != nil {
			t.Errorf("ejemplo %d (servicio) del README no se parsea: %v", i, err)
		}
	}
}

// El primer objeto y el primer servicio del README forman una composición válida
// que el cargador acepta, de modo que el ejemplo de referencias es ejecutable.
func TestReadmeEjemplosObjetoYServicioComponen(t *testing.T) {
	bloques := bloquesYAMLReadme(t)
	if len(bloques) < 2 {
		t.Fatalf("se esperaban al menos 2 bloques yaml en el README, hay %d", len(bloques))
	}
	comp, err := escribirComposicion(t, map[string]string{
		"objetos_contacto.yaml":     bloques[0],
		"relaciones_proveedor.yaml": bloques[1],
	})
	if err != nil {
		t.Fatalf("los ejemplos del README no componen: %v", err)
	}
	if n := len(comp.Servicios["proveedor"].Relations); n != 1 {
		t.Errorf("relaciones compuestas desde el ejemplo = %d; se esperaba 1", n)
	}
}
