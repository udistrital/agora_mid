package generador

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestEscribirServicios_ArchivoDeServicioConClavesDelCargadorYDeterminista(t *testing.T) {
	write := "proveedores.proveedor:\n  creatable: true\n  updatable: true\n  deletable: false\n"
	_, servicios, err := construirServiciosPrueba(t, dbmlPrueba, map[string]string{
		"expand.yaml":  "- proveedores.contacto.proveedor_id\n",
		"write.yaml":   write,
		"service.yaml": servicioPrueba,
	})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	dir1, dir2 := t.TempDir(), t.TempDir()
	if err := EscribirServicios(servicios, dir1); err != nil {
		t.Fatalf("escribiendo la primera salida: %v", err)
	}
	if err := EscribirServicios(servicios, dir2); err != nil {
		t.Fatalf("escribiendo la segunda salida: %v", err)
	}

	ruta := filepath.Join("gestion_personas_crud", "relaciones_proveedor.yaml")
	a, err := os.ReadFile(filepath.Join(dir1, ruta))
	if err != nil {
		t.Fatalf("se esperaba %s: %v", ruta, err)
	}
	b, err := os.ReadFile(filepath.Join(dir2, ruta))
	if err != nil {
		t.Fatalf("segunda ejecución sin %s: %v", ruta, err)
	}
	if !bytes.Equal(a, b) {
		t.Fatal("el archivo de servicio difiere entre ejecuciones")
	}

	var doc map[string]any
	if err := yaml.Unmarshal(a, &doc); err != nil {
		t.Fatalf("el archivo no es YAML válido: %v", err)
	}
	cadenas := map[string]string{"service": "proveedor", "raiz": "Proveedor", "on_relation_error": "notify"}
	for clave, esperado := range cadenas {
		if doc[clave] != esperado {
			t.Errorf("%s = %v, se esperaba %q", clave, doc[clave], esperado)
		}
	}
	filtros, _ := doc["filterable_fields"].([]any)
	if len(filtros) != 2 || filtros[0] != "Id" || filtros[1] != "TerceroId" {
		t.Errorf("filterable_fields = %v, se esperaba [Id TerceroId]", filtros)
	}
	politica, _ := doc["write"].(map[string]any)
	if politica == nil || politica["policy"] == nil {
		t.Errorf("falta write.policy en el archivo de servicio: %v", doc["write"])
	}
	if _, ok := doc["referencias"].([]any); !ok {
		t.Errorf("referencias debe ser una lista: %v", doc["referencias"])
	}
}
