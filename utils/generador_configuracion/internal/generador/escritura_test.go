package generador

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestEscribirObjetos_SalidaDeterministaByteAByte(t *testing.T) {
	expand := "- proveedores.contacto.proveedor_id\n- proveedores.proveedor_natural.id\n"
	fkInbound := "- proveedores.proveedor.tercero_id\n"
	write := "proveedores.contacto:\n  creatable: true\n  updatable: true\n  deletable: true\n  natural_key:\n    - valor_especifico\n"

	primera := mustConstruir(t, expand, fkInbound, "", write)
	segunda := mustConstruir(t, expand, fkInbound, "", write)
	if !reflect.DeepEqual(primera, segunda) {
		t.Fatal("dos construcciones con las mismas entradas difieren")
	}

	dir1, dir2 := t.TempDir(), t.TempDir()
	if err := EscribirObjetos(primera, dir1); err != nil {
		t.Fatalf("escribiendo la primera salida: %v", err)
	}
	if err := EscribirObjetos(segunda, dir2); err != nil {
		t.Fatalf("escribiendo la segunda salida: %v", err)
	}

	archivos1 := archivosRelativos(t, dir1)
	archivos2 := archivosRelativos(t, dir2)
	if len(archivos1) == 0 || !reflect.DeepEqual(archivos1, archivos2) {
		t.Fatalf("conjuntos de archivos distintos o vacíos: %v vs %v", archivos1, archivos2)
	}
	for _, rel := range archivos1 {
		a, err1 := os.ReadFile(filepath.Join(dir1, rel))
		b, err2 := os.ReadFile(filepath.Join(dir2, rel))
		if err1 != nil || err2 != nil {
			t.Fatalf("leyendo %s: %v %v", rel, err1, err2)
		}
		if !bytes.Equal(a, b) {
			t.Fatalf("%s difiere entre ejecuciones", rel)
		}
	}
}

func TestEscribirObjetos_RutaPorCRUDYTabla(t *testing.T) {
	objetos := mustConstruir(t, "- proveedores.contacto.proveedor_id\n", "", "", "")
	dir := t.TempDir()
	if err := EscribirObjetos(objetos, dir); err != nil {
		t.Fatalf("escribiendo objetos: %v", err)
	}
	ruta := filepath.Join(dir, "gestion_personas_crud", "objetos_contacto.yaml")
	if _, err := os.Stat(ruta); err != nil {
		t.Fatalf("se esperaba el archivo %s: %v", ruta, err)
	}
}

// archivosRelativos lista las rutas relativas de los archivos bajo dir, ordenadas.
func archivosRelativos(t *testing.T, dir string) []string {
	t.Helper()
	var rutas []string
	err := filepath.WalkDir(dir, func(ruta string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(dir, ruta)
		if err != nil {
			return err
		}
		rutas = append(rutas, rel)
		return nil
	})
	if err != nil {
		t.Fatalf("recorriendo %s: %v", dir, err)
	}
	return rutas
}

func TestCargarSidecars_EntradaSinCorrespondenciaFallaIdentificandola(t *testing.T) {
	casos := map[string]struct {
		expand, fkInbound, blacklist, write, entrada string
	}{
		"expand sin FK declarada":           {expand: "- proveedores.contacto.columna_inexistente\n", entrada: "proveedores.contacto.columna_inexistente"},
		"fk_inbound sin FK declarada":       {fkInbound: "- proveedores.contacto.valor_especifico\n", entrada: "proveedores.contacto.valor_especifico"},
		"blacklist con columna inexistente": {blacklist: "- proveedores.contacto.inexistente\n", entrada: "proveedores.contacto.inexistente"},
		"write con tabla inexistente":       {write: "proveedores.tabla_fantasma:\n  creatable: true\n", entrada: "proveedores.tabla_fantasma"},
		"write con natural_key inexistente": {write: "proveedores.contacto:\n  natural_key:\n    - columna_fantasma\n", entrada: "columna_fantasma"},
		"write con clave desconocida":       {write: "proveedores.contacto:\n  creatible: true\n", entrada: "creatible"},
	}
	for nombre, c := range casos {
		_, err := construirPrueba(t, c.expand, c.fkInbound, c.blacklist, c.write)
		if err == nil {
			t.Errorf("%s: se esperaba error", nombre)
			continue
		}
		if !strings.Contains(err.Error(), c.entrada) {
			t.Errorf("%s: el error %q no identifica la entrada %q", nombre, err.Error(), c.entrada)
		}
	}
}
