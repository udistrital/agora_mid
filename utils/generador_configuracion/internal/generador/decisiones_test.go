package generador

import (
	"os"
	"path/filepath"
	"testing"
)

// dbmlUbicaciones declara un lugar del esquema ubicaciones, cuyo CRUD existe en la
// configuración (ubicaciones_crud_url).
const dbmlUbicaciones = `
Table ubicaciones.lugar {
  id serial [pk]
  nombre varchar(100)
}
`

func TestConstruirObjetos_EsquemaUbicacionesGeneraObjetosConSuCRUD(t *testing.T) {
	m, err := ParsearDBML(dbmlUbicaciones)
	if err != nil {
		t.Fatalf("parseando: %v", err)
	}
	s, err := CargarSidecars(t.TempDir(), m)
	if err != nil {
		t.Fatalf("cargando sidecars: %v", err)
	}
	objetos, err := ConstruirObjetos(m, s)
	if err != nil {
		t.Fatalf("construyendo: %v", err)
	}
	lugar := buscarObjeto(t, objetos, "Lugar")
	if lugar.RelatedAPI != "ubicaciones" || lugar.RelatedTable != "lugar" {
		t.Fatalf("api=%q tabla=%q, se esperaba ubicaciones/lugar", lugar.RelatedAPI, lugar.RelatedTable)
	}
}

func TestGenerar_LimpiaLaSalidaGeneradaAntesDeEscribirYConservaLoAjeno(t *testing.T) {
	entradas := t.TempDir()
	if err := os.WriteFile(filepath.Join(entradas, "modelo.dbml"), []byte(dbmlPrueba), 0o644); err != nil {
		t.Fatal(err)
	}
	salida := t.TempDir()
	obsoleto := filepath.Join(salida, "gestion_personas_crud", "objetos_tabla_eliminada.yaml")
	ajeno := filepath.Join(salida, "notas.txt")
	if err := os.MkdirAll(filepath.Dir(obsoleto), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(obsoleto, []byte(encabezadoGenerado+"name: Vieja\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ajeno, []byte("escrito a mano\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Generar(filepath.Join(entradas, "modelo.dbml"), entradas, salida); err != nil {
		t.Fatalf("generando: %v", err)
	}

	if _, err := os.Stat(obsoleto); !os.IsNotExist(err) {
		t.Errorf("el objeto obsoleto generado debe eliminarse antes de regenerar: %v", err)
	}
	if _, err := os.Stat(ajeno); err != nil {
		t.Errorf("un archivo sin encabezado generado no debe eliminarse: %v", err)
	}
	if _, err := os.Stat(filepath.Join(salida, "gestion_personas_crud", "objetos_proveedor.yaml")); err != nil {
		t.Errorf("la regeneración debe escribir los objetos: %v", err)
	}
}

// Esta prueba fija una decisión ya implementada: fk_inbound prevalece sobre expand
// cuando una FK aparece en ambos sidecars. No exige cambio de código.
func TestConstruirObjetos_FKInboundPrevaleceSobreExpandEnConflicto(t *testing.T) {
	objetos := mustConstruir(t,
		"- entrada: proveedores.contacto.proveedor_id\n  depth: 3\n",
		"- entrada: proveedores.contacto.proveedor_id\n  depth: 2\n",
		"", "")

	contacto := buscarObjeto(t, objetos, "Contacto")
	if len(contacto.Referencias) != 1 {
		t.Fatalf("referencias de Contacto = %+v, se esperaba una arista fk_inbound", contacto.Referencias)
	}
	arista := contacto.Referencias[0]
	if arista.KeyPropagation != "fk_inbound" || arista.Objeto != "Proveedor" || arista.Depth != 2 {
		t.Fatalf("arista = %+v, se esperaba fk_inbound hacia Proveedor con depth 2", arista)
	}
}
