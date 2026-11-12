package generador

import (
	"reflect"
	"strings"
	"testing"
)

func TestParsearDBML_ExpandeTablePartialsYOmiteIndexes(t *testing.T) {
	m, err := ParsearDBML(dbmlPrueba)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	var proveedor *Tabla
	for i := range m.Tablas {
		if m.Tablas[i].Nombre == "proveedor" {
			proveedor = &m.Tablas[i]
		}
	}
	if proveedor == nil {
		t.Fatal("no se leyó la tabla proveedores.proveedor")
	}
	nombres := make([]string, 0, len(proveedor.Columnas))
	for _, c := range proveedor.Columnas {
		nombres = append(nombres, c.Nombre)
	}
	esperados := []string{"id", "tipo_proveedor", "tercero_id", "activo", "fecha_creacion", "fecha_modificacion", "usuario_modificacion_id"}
	if !reflect.DeepEqual(nombres, esperados) {
		t.Fatalf("columnas = %v, se esperaba %v", nombres, esperados)
	}
	if len(m.Refs) != 3 {
		t.Fatalf("referencias = %d, se esperaban 3", len(m.Refs))
	}
	primera := m.Refs[0]
	if primera.Padre.Clave() != "proveedores.proveedor.id" || primera.Hijo.Clave() != "proveedores.contacto.proveedor_id" || primera.Compartida {
		t.Fatalf("referencia `<` mal leída: %+v", primera)
	}
	if !m.Refs[2].Compartida || m.Refs[2].Hijo.Clave() != "proveedores.proveedor_natural.id" {
		t.Fatalf("referencia `-` debe ser PK compartida con el hijo en la especialización: %+v", m.Refs[2])
	}
}

func TestParsearDBML_OperadoresOpcionalesYMayorInvierten(t *testing.T) {
	dbml := `
Table proveedores.a {
  id serial [pk]
  b_id integer
}
Table proveedores.b {
  id serial [pk]
}
Ref r1 {
  proveedores.b.id ?< proveedores.a.b_id
}
Ref r2 {
  proveedores.a.b_id > proveedores.b.id
}
`
	m, err := ParsearDBML(dbml)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if m.Refs[0].Hijo.Clave() != "proveedores.a.b_id" || m.Refs[0].Padre.Clave() != "proveedores.b.id" {
		t.Fatalf("`?<` mal leído: %+v", m.Refs[0])
	}
	if m.Refs[1].Hijo.Clave() != "proveedores.a.b_id" || m.Refs[1].Padre.Clave() != "proveedores.b.id" {
		t.Fatalf("`>` debe invertir los lados: %+v", m.Refs[1])
	}
}

func TestParsearDBML_ErroresDeSintaxisYReferenciasFallan(t *testing.T) {
	if _, err := ParsearDBML(dbmlPrueba); err != nil {
		t.Fatalf("el DBML válido de prueba debe parsear: %v", err)
	}
	casos := map[string]string{
		"tipo no soportado":              "Table proveedores.x {\n  id serial [pk]\n  geo geometry\n}\n",
		"spread desconocido":             "Table proveedores.x {\n  ~noexiste\n}\n",
		"referencia a tabla inexistente": "Table proveedores.x {\n  id serial [pk]\n  y_id integer\n}\nRef r {\n  proveedores.x.id < proveedores.y.id\n}\n",
		"tabla sin esquema":              "Table x {\n  id serial [pk]\n}\n",
	}
	for nombre, dbml := range casos {
		if _, err := ParsearDBML(dbml); err == nil {
			t.Errorf("%s: se esperaba error", nombre)
		}
	}
}

func TestConstruirObjetos_TablaProduceObjetoConCamposTipadosYOmiteBlacklist(t *testing.T) {
	objetos := mustConstruir(t, "", "", "- proveedores.proveedor_natural.cargo\n", "")
	o := buscarObjeto(t, objetos, "ProveedorNatural")
	if o.RelatedAPI != "gestion_personas_crud" || o.RelatedTable != "proveedor_natural" || o.Endpoint != "/v1/proveedor_natural" {
		t.Fatalf("ubicación del objeto incorrecta: api=%q tabla=%q endpoint=%q", o.RelatedAPI, o.RelatedTable, o.Endpoint)
	}
	esperados := []CampoConfig{
		{Name: "Id", Type: "integer", Readable: true, Writable: false},
		{Name: "PaisNacimientoId", Type: "integer", Readable: true, Writable: true},
		{Name: "Activo", Type: "boolean", Readable: true, Writable: false},
		{Name: "FechaCreacion", Type: "timestamp", Readable: true, Writable: false},
		{Name: "FechaModificacion", Type: "timestamp", Readable: true, Writable: false},
		{Name: "UsuarioModificacionId", Type: "integer", Readable: true, Writable: true},
	}
	if !reflect.DeepEqual(o.Campos, esperados) {
		t.Fatalf("campos = %+v\nse esperaban %+v", o.Campos, esperados)
	}
	for _, c := range o.Campos {
		if strings.EqualFold(c.Name, "Cargo") {
			t.Fatal("la columna en blacklist no debe aparecer en el objeto")
		}
	}
}
