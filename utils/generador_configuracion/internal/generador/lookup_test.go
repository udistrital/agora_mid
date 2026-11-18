package generador

import (
	"reflect"
	"strings"
	"testing"
)

// dbmlConsulta tiene una FK hacia un catálogo con sufijo _id, una FK sin sufijo y una
// colección de tercero que referencia de vuelta a tercero.
const dbmlConsulta = `
Table terceros.tercero {
  id serial [pk]
  nombre_completo varchar(255)
}

Table terceros.seguridad_social_tercero {
  id serial [pk]
  tercero_id integer
  tercero_entidad_id integer
}

Table ubicaciones.lugar {
  id serial [pk]
  nombre varchar(100)
}

Table parametros.parametro {
  id serial [pk]
  nombre varchar(100)
}

Table proveedores.proveedor {
  id serial [pk]
  tipo_proveedor integer
  tercero_id integer
}

Table proveedores.proveedor_natural {
  id serial [pk]
  pais_nacimiento_id integer
}

Ref fk_proveedor_natural_lugar {
  ubicaciones.lugar.id < proveedores.proveedor_natural.pais_nacimiento_id
}

Ref fk_proveedor_parametro {
  parametros.parametro.id < proveedores.proveedor.tipo_proveedor
}

Ref fk_proveedor_tercero {
  terceros.tercero.id < proveedores.proveedor.tercero_id
}

Ref rel_seguridad_social_tercero_tercero {
  terceros.tercero.id < terceros.seguridad_social_tercero.tercero_id
}

Ref rel_seguridad_social_tercero_tercero_entidad {
  terceros.tercero.id < terceros.seguridad_social_tercero.tercero_entidad_id
}
`

// construirConsulta carga los archivos dados sobre dbmlConsulta y construye los
// objetos.
func construirConsulta(t *testing.T, archivos map[string]string) ([]ObjetoConfig, error) {
	t.Helper()
	m, s, err := prepararPrueba(t, dbmlConsulta, archivos)
	if err != nil {
		return nil, err
	}
	return ConstruirObjetos(m, s)
}

func TestConstruirObjetos_LookupProduceAristaDeSoloLectura(t *testing.T) {
	objetos, err := construirConsulta(t, map[string]string{
		"lookup.yaml": "- entrada: proveedores.proveedor_natural.pais_nacimiento_id\n  depth: 2\n",
		"write.yaml":  "ubicaciones.lugar:\n  creatable: true\n  updatable: true\n  deletable: true\n",
	})
	if err != nil {
		t.Fatalf("construyendo objetos: %v", err)
	}
	arista := buscarArista(t, buscarObjeto(t, objetos, "ProveedorNatural"), "Lugar")
	esperada := AristaConfig{
		Objeto:       "Lugar",
		Direction:    "local_to_related",
		Cardinality:  "one",
		LocalField:   "PaisNacimientoId",
		RelatedField: "Id",
		Depth:        2,
		MappedName:   "PaisNacimiento",
	}
	if !reflect.DeepEqual(arista, esperada) {
		t.Fatalf("arista = %+v\nse esperaba %+v", arista, esperada)
	}
	if lugar := buscarObjeto(t, objetos, "Lugar"); len(lugar.Referencias) != 0 {
		t.Fatalf("el objeto referenciado no debe recibir arista: %+v", lugar.Referencias)
	}
}

func TestCargarSidecars_LookupEnFlattenFalla(t *testing.T) {
	col := "proveedores.proveedor_natural.id"
	m, s, err := prepararPrueba(t, dbmlPrueba, map[string]string{
		"lookup.yaml":  "- " + col + "\n",
		"flatten.yaml": "- " + col + "\n",
	})
	if err == nil {
		_, err = ConstruirObjetos(m, s)
	}
	if err == nil {
		t.Fatal("una columna en lookup y en flatten debe fallar")
	}
	if !strings.Contains(err.Error(), col) {
		t.Fatalf("el error debe identificar la entrada: %v", err)
	}
}

func TestCargarSidecars_LookupEnConflictoFalla(t *testing.T) {
	col := "proveedores.proveedor_natural.pais_nacimiento_id"
	casos := map[string]map[string]string{
		"sin FK":        {"lookup.yaml": "- proveedores.proveedor_natural.id\n"},
		"en expand":     {"lookup.yaml": "- " + col + "\n", "expand.yaml": "- " + col + "\n"},
		"en fk_inbound": {"lookup.yaml": "- " + col + "\n", "fk_inbound.yaml": "- " + col + "\n"},
		"en blacklist":  {"lookup.yaml": "- " + col + "\n", "blacklist.yaml": "- " + col + "\n"},
	}
	for nombre, archivos := range casos {
		if _, err := construirConsulta(t, archivos); err == nil {
			t.Errorf("%s: se esperaba error", nombre)
		}
	}
}

func TestConstruirObjetos_NombreDeAristaSinSufijoId(t *testing.T) {
	objetos, err := construirConsulta(t, map[string]string{
		"fk_inbound.yaml": "- proveedores.proveedor.tercero_id\n",
	})
	if err != nil {
		t.Fatalf("construyendo objetos: %v", err)
	}
	arista := buscarArista(t, buscarObjeto(t, objetos, "Proveedor"), "Tercero")
	if arista.MappedName != "Tercero" || arista.LocalField != "TerceroId" {
		t.Fatalf("arista fk_inbound = %+v; se esperaba mapped_name Tercero y local_field TerceroId", arista)
	}
}

func TestConstruirObjetos_MappedNameExplicitoPrevalece(t *testing.T) {
	objetos, err := construirConsulta(t, map[string]string{
		"lookup.yaml": "- entrada: proveedores.proveedor.tipo_proveedor\n  mapped_name: TipoProveedorDetalle\n",
	})
	if err != nil {
		t.Fatalf("construyendo objetos: %v", err)
	}
	arista := buscarArista(t, buscarObjeto(t, objetos, "Proveedor"), "Parametro")
	if arista.MappedName != "TipoProveedorDetalle" {
		t.Fatalf("mapped_name = %q; se esperaba TipoProveedorDetalle", arista.MappedName)
	}
}

func TestConstruirObjetos_MappedNameQueCoincideConCampoFalla(t *testing.T) {
	_, err := construirConsulta(t, map[string]string{
		"lookup.yaml": "- proveedores.proveedor.tipo_proveedor\n",
	})
	if err == nil {
		t.Fatal("un nombre de arista igual al campo TipoProveedor debe fallar")
	}
	if !strings.Contains(err.Error(), "proveedores.proveedor.tipo_proveedor") {
		t.Fatalf("el error debe identificar la entrada: %v", err)
	}
}

func TestConstruirObjetos_CicloDeReferenciasFalla(t *testing.T) {
	_, err := construirConsulta(t, map[string]string{
		"expand.yaml": "- terceros.seguridad_social_tercero.tercero_id\n",
		"lookup.yaml": "- terceros.seguridad_social_tercero.tercero_entidad_id\n",
	})
	if err == nil {
		t.Fatal("Tercero -> SeguridadSocialTercero -> Tercero debe fallar como ciclo")
	}
	if !strings.Contains(err.Error(), "ciclo") {
		t.Fatalf("el error debe nombrar el ciclo: %v", err)
	}
}
