package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// escribirComposicion crea un directorio temporal con los archivos indicados y
// carga la composición desde él.
func escribirComposicion(t *testing.T, archivos map[string]string) (Composicion, error) {
	t.Helper()
	dir := t.TempDir()
	for nombre, contenido := range archivos {
		if err := os.WriteFile(filepath.Join(dir, nombre), []byte(contenido), 0o644); err != nil {
			t.Fatalf("no se pudo escribir %s: %v", nombre, err)
		}
	}
	return CargarComposicion(dir, urlsPrueba())
}

const objetoTercero = `name: Tercero
related_api: terceros
related_table: tercero
endpoint: /v1/tercero
campos:
  - name: NombreCompleto
    type: string
    readable: true
    writable: true
  - name: PrimerNombre
    type: string
    writable: true
  - name: TipoContribuyenteId
    type: integer
    readable: true
    writable: true
referencias:
  - objeto: DatosIdentificacion
    direction: related_to_local
    local_field: Id
    related_field: TerceroId
    cardinality: many
    depth: 1
    mapped_name: DatosIdentificacion
    key_propagation: fk
    policy:
      creatable: true
      updatable: true
      deletable: false
  - objeto: TipoContribuyente
    direction: local_to_related
    local_field: TipoContribuyenteId
    related_field: Id
    cardinality: one
    depth: 1
    mapped_name: TipoContribuyente
`

const objetoDatosIdentificacion = `name: DatosIdentificacion
related_api: terceros
related_table: datos_identificacion
endpoint: /v1/datos_identificacion
natural_key:
  - Numero
  - TipoDocumentoId
campos:
  - name: Numero
    type: string
    readable: true
    writable: true
    immutable: true
  - name: TipoDocumentoId
    type: integer
    readable: true
    writable: true
    immutable: true
  - name: CiudadExpedicion
    type: integer
    readable: true
    writable: true
`

const objetoTipoContribuyente = `name: TipoContribuyente
related_api: terceros
related_table: tipo_contribuyente
endpoint: /v1/tipo_contribuyente
campos:
  - name: Nombre
    type: string
    readable: true
`

const servicioProveedorComposicion = `service: proveedor
max_depth: 4
on_relation_error: notify
max_related_ids: 100
referencias:
  - objeto: Tercero
    direction: local_to_related
    local_field: TerceroId
    related_field: Id
    cardinality: one
    depth: 3
    mapped_name: Tercero
    key_propagation: fk_inbound
    policy:
      creatable: true
      updatable: true
      deletable: false
`

// Prueba que el cargador resuelve la referencia del servicio al objeto Tercero y
// produce un árbol efectivo por servicio, con las vistas de lectura y escritura del
// objeto y la arista de la referencia.
func TestCargarComposicion_ArbolEfectivoConProyecciones(t *testing.T) {
	comp, err := escribirComposicion(t, map[string]string{
		"objetos_tercero.yaml":              objetoTercero,
		"objetos_datos_identificacion.yaml": objetoDatosIdentificacion,
		"objetos_tipo_contribuyente.yaml":   objetoTipoContribuyente,
		"relaciones_proveedor.yaml":         servicioProveedorComposicion,
	})
	if err != nil {
		t.Fatalf("composición válida rechazada: %v", err)
	}

	sc, ok := comp.Servicios["proveedor"]
	if !ok {
		t.Fatal("no se produjo el árbol efectivo del servicio proveedor")
	}
	if len(sc.Relations) != 1 {
		t.Fatalf("se esperaba 1 relación de primer nivel, hay %d", len(sc.Relations))
	}

	rel := sc.Relations[0]
	if rel.Name != "Tercero" || rel.MappedName != "Tercero" || rel.RelatedAPI != "terceros" ||
		rel.RelatedTable != "tercero" || rel.Endpoint != "/v1/tercero" {
		t.Errorf("nodo Tercero = name:%q mapped:%q api:%q tabla:%q endpoint:%q",
			rel.Name, rel.MappedName, rel.RelatedAPI, rel.RelatedTable, rel.Endpoint)
	}
	if rel.Direction != DirectionLocalToRelated || rel.Cardinality != CardinalityOne ||
		rel.LocalField != "TerceroId" || rel.RelatedField != "Id" || rel.Depth != 3 {
		t.Errorf("arista de Tercero incorrecta: %+v", rel)
	}
	if rel.QueryTemplate != "Id:{ids}" {
		t.Errorf("QueryTemplate = %q, se esperaba %q", rel.QueryTemplate, "Id:{ids}")
	}
	if !sonIgualesCadenas(rel.AllowedFields, []string{"NombreCompleto", "TipoContribuyenteId"}) {
		t.Errorf("AllowedFields = %v", rel.AllowedFields)
	}
	if rel.Write == nil {
		t.Fatal("Tercero debe tener bloque write por su propagación fk_inbound")
	}
	if !sonIgualesCadenas(rel.Write.WritableFields, []string{"NombreCompleto", "PrimerNombre", "TipoContribuyenteId"}) {
		t.Errorf("WritableFields = %v", rel.Write.WritableFields)
	}
	if rel.Write.KeyPropagation != KeyPropagationFKInbound || !rel.Write.Policy.Creatable || rel.Write.Policy.Deletable {
		t.Errorf("write de Tercero = prop:%q policy:%+v", rel.Write.KeyPropagation, rel.Write.Policy)
	}

	if len(rel.NestedRelations) != 2 {
		t.Fatalf("se esperaban 2 relaciones anidadas, hay %d", len(rel.NestedRelations))
	}

	datos := rel.NestedRelations[0]
	if datos.Name != "DatosIdentificacion" || datos.QueryTemplate != "TerceroId:{ids}" ||
		datos.Direction != DirectionRelatedToLocal || datos.Cardinality != CardinalityMany {
		t.Errorf("nodo DatosIdentificacion = %+v", datos)
	}
	if !sonIgualesCadenas(datos.AllowedFields, []string{"Numero", "TipoDocumentoId", "CiudadExpedicion"}) {
		t.Errorf("AllowedFields de DatosIdentificacion = %v", datos.AllowedFields)
	}
	if datos.Write == nil || !sonIgualesCadenas(datos.Write.ImmutableFields, []string{"Numero", "TipoDocumentoId"}) ||
		!sonIgualesCadenas(datos.Write.NaturalKey, []string{"Numero", "TipoDocumentoId"}) ||
		datos.Write.KeyPropagation != KeyPropagationFK {
		t.Errorf("write de DatosIdentificacion incorrecto: %+v", datos.Write)
	}

	tipo := rel.NestedRelations[1]
	if tipo.Name != "TipoContribuyente" || tipo.Write != nil {
		t.Errorf("TipoContribuyente debe ser de solo lectura (write nil), obtuvo %+v", tipo.Write)
	}
}

// Prueba que un objeto definido sin referenciar se reporta como huérfano por nombre
// y no entra al árbol efectivo de ningún servicio.
func TestCargarComposicion_HuerfanoSeReportaYNoEntraAlArbol(t *testing.T) {
	comp, err := escribirComposicion(t, map[string]string{
		"objetos_tercero.yaml":              objetoTercero,
		"objetos_datos_identificacion.yaml": objetoDatosIdentificacion,
		"objetos_tipo_contribuyente.yaml":   objetoTipoContribuyente,
		"objetos_solo.yaml":                 "name: Solo\nrelated_api: terceros\nrelated_table: solo\nendpoint: /v1/solo\ncampos: []\n",
		"relaciones_proveedor.yaml":         servicioProveedorComposicion,
	})
	if err != nil {
		t.Fatalf("composición con huérfano no debe abortar: %v", err)
	}
	if len(comp.Huerfanos) != 1 || comp.Huerfanos[0] != "Solo" {
		t.Errorf("Huerfanos = %v, se esperaba [Solo]", comp.Huerfanos)
	}
	for _, rel := range comp.Servicios["proveedor"].Relations {
		if rel.Name == "Solo" {
			t.Error("el huérfano Solo no debe entrar al árbol efectivo")
		}
	}
}

// Prueba la integridad de la composición al arranque: cada caso aborta nombrando el
// fragmento responsable.
func TestCargarComposicion_IntegridadAbortaNombrandoElFragmento(t *testing.T) {
	cadena := func() map[string]string {
		archivos := map[string]string{
			"relaciones_proveedor.yaml": "service: proveedor\nmax_depth: 4\non_relation_error: notify\nmax_related_ids: 100\nreferencias:\n  - objeto: O1\n    direction: local_to_related\n    local_field: X\n    related_field: Id\n    cardinality: one\n    depth: 1\n    mapped_name: M1\n",
		}
		for i := 1; i <= 9; i++ {
			siguiente := ""
			if i < 9 {
				siguiente = fmt.Sprintf("referencias:\n  - objeto: O%d\n    direction: local_to_related\n    local_field: X\n    related_field: Id\n    cardinality: one\n    depth: 1\n    mapped_name: M%d\n", i+1, i+1)
			}
			archivos[fmt.Sprintf("objetos_o%d.yaml", i)] = fmt.Sprintf("name: O%d\nrelated_api: terceros\nrelated_table: t\nendpoint: /v1/t\ncampos: []\n%s", i, siguiente)
		}
		return archivos
	}

	ciclo := map[string]string{
		"relaciones_proveedor.yaml": "service: proveedor\nmax_depth: 4\non_relation_error: notify\nmax_related_ids: 100\nreferencias:\n  - objeto: A\n    direction: local_to_related\n    local_field: X\n    related_field: Id\n    cardinality: one\n    depth: 1\n    mapped_name: IrA\n",
		"objetos_a.yaml":            "name: A\nrelated_api: terceros\nrelated_table: a\nendpoint: /v1/a\ncampos: []\nreferencias:\n  - objeto: B\n    direction: local_to_related\n    local_field: X\n    related_field: Id\n    cardinality: one\n    depth: 1\n    mapped_name: IrB\n",
		"objetos_b.yaml":            "name: B\nrelated_api: terceros\nrelated_table: b\nendpoint: /v1/b\ncampos: []\nreferencias:\n  - objeto: A\n    direction: local_to_related\n    local_field: X\n    related_field: Id\n    cardinality: one\n    depth: 1\n    mapped_name: VolverA\n",
	}

	colgantePorServicio := map[string]string{
		"relaciones_proveedor.yaml": "service: proveedor\nmax_depth: 4\non_relation_error: notify\nmax_related_ids: 100\nreferencias:\n  - objeto: Fantasma\n    direction: local_to_related\n    local_field: X\n    related_field: Id\n    cardinality: one\n    depth: 1\n    mapped_name: F\n",
	}

	colgantePorObjeto := map[string]string{
		"relaciones_proveedor.yaml": "service: proveedor\nmax_depth: 4\non_relation_error: notify\nmax_related_ids: 100\nreferencias:\n  - objeto: A\n    direction: local_to_related\n    local_field: X\n    related_field: Id\n    cardinality: one\n    depth: 1\n    mapped_name: IrA\n",
		"objetos_a.yaml":            "name: A\nrelated_api: terceros\nrelated_table: a\nendpoint: /v1/a\ncampos: []\nreferencias:\n  - objeto: NoExiste\n    direction: local_to_related\n    local_field: X\n    related_field: Id\n    cardinality: one\n    depth: 1\n    mapped_name: Roto\n",
	}

	casos := []struct {
		nombre    string
		archivos  map[string]string
		nombrados []string
	}{
		{"ciclo de referencias nombra el fragmento que cierra el ciclo", ciclo, []string{"ciclo", "VolverA", "B"}},
		{"referencia colgante desde servicio nombra objeto ausente y servicio", colgantePorServicio, []string{"Fantasma", "proveedor"}},
		{"referencia colgante desde objeto nombra objeto ausente y objeto", colgantePorObjeto, []string{"NoExiste", "A"}},
		{"profundidad excedida nombra la referencia que la excede", cadena(), []string{"M9"}},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			_, err := escribirComposicion(t, c.archivos)
			if err == nil {
				t.Fatal("se esperaba aborto del arranque")
			}
			for _, esperado := range c.nombrados {
				if !strings.Contains(err.Error(), esperado) {
					t.Errorf("el error debe nombrar %q, obtuvo: %v", esperado, err)
				}
			}
		})
	}
}

// Prueba que un mismo objeto usado en ramas distintas, incluso por dos servicios, no
// es un ciclo de composición: carga sin error y cada árbol conserva su arista. El
// ciclo de datos de la expansión en tiempo de petición es otro mecanismo, con aviso
// cycle, y no se evalúa al arranque.
func TestCargarComposicion_ReusoEnRamasNoEsCicloDeComposicion(t *testing.T) {
	contacto := "name: Contacto\nrelated_api: gestion_personas_crud\nrelated_table: contacto\nendpoint: /v1/contacto\ncampos: []\n"
	tercero := "name: Tercero\nrelated_api: terceros\nrelated_table: tercero\nendpoint: /v1/tercero\ncampos: []\nreferencias:\n  - objeto: Contacto\n    direction: related_to_local\n    local_field: Id\n    related_field: TerceroId\n    cardinality: many\n    depth: 1\n    mapped_name: Contactos\n"
	proveedor := "service: proveedor\nmax_depth: 4\non_relation_error: notify\nmax_related_ids: 100\nreferencias:\n  - objeto: Tercero\n    direction: local_to_related\n    local_field: TerceroId\n    related_field: Id\n    cardinality: one\n    depth: 2\n    mapped_name: Tercero\n  - objeto: Contacto\n    direction: related_to_local\n    local_field: Id\n    related_field: ProveedorId\n    cardinality: many\n    depth: 1\n    mapped_name: Contacto\n"
	otro := "service: otro\nmax_depth: 4\non_relation_error: notify\nmax_related_ids: 100\nreferencias:\n  - objeto: Tercero\n    direction: local_to_related\n    local_field: TerceroId\n    related_field: Id\n    cardinality: one\n    depth: 1\n    mapped_name: TerceroOtro\n"

	comp, err := escribirComposicion(t, map[string]string{
		"objetos_contacto.yaml":     contacto,
		"objetos_tercero.yaml":      tercero,
		"relaciones_proveedor.yaml": proveedor,
		"relaciones_otro.yaml":      otro,
	})
	if err != nil {
		t.Fatalf("reutilización de un objeto en ramas distintas no debe abortar el arranque: %v", err)
	}

	prov := comp.Servicios["proveedor"]
	if len(prov.Relations) != 2 || prov.Relations[0].Name != "Tercero" || prov.Relations[1].Name != "Contacto" {
		t.Fatalf("relaciones de proveedor = %d nodos, se esperaban Tercero y Contacto", len(prov.Relations))
	}
	if len(prov.Relations[0].NestedRelations) != 1 || prov.Relations[0].NestedRelations[0].Name != "Contacto" {
		t.Error("Contacto debe componerse también bajo Tercero")
	}

	otroTercero := comp.Servicios["otro"].Relations[0]
	if otroTercero.Depth != 1 || otroTercero.MappedName != "TerceroOtro" || prov.Relations[0].Depth != 2 {
		t.Errorf("las aristas de Tercero deben ser independientes entre servicios: proveedor depth=%d otro depth=%d",
			prov.Relations[0].Depth, otroTercero.Depth)
	}
}

// Prueba que un ciclo de referencias entre objetos aborta el arranque aunque los
// objetos estén en un servicio consumidor válido.
func TestCargarComposicion_CicloDeComposicionAbortaElArranque(t *testing.T) {
	auto := "name: Auto\nrelated_api: terceros\nrelated_table: auto\nendpoint: /v1/auto\ncampos: []\nreferencias:\n  - objeto: Auto\n    direction: local_to_related\n    local_field: X\n    related_field: Id\n    cardinality: one\n    depth: 1\n    mapped_name: Si\n"
	_, err := escribirComposicion(t, map[string]string{
		"objetos_auto.yaml":         auto,
		"relaciones_proveedor.yaml": "service: proveedor\nmax_depth: 4\non_relation_error: notify\nmax_related_ids: 100\nreferencias:\n  - objeto: Auto\n    direction: local_to_related\n    local_field: X\n    related_field: Id\n    cardinality: one\n    depth: 1\n    mapped_name: Auto\n",
	})
	if err == nil || !strings.Contains(err.Error(), "ciclo de referencias") {
		t.Fatalf("se esperaba aborto por ciclo de composición, obtuvo: %v", err)
	}
}

// Prueba que un objeto cuyo CRUD no existe en las URLs, o cuyo natural_key no está
// entre los campos writable, aborta el arranque.
func TestCargarComposicion_UbicacionDeObjetoInvalidaAborta(t *testing.T) {
	servicio := "service: proveedor\nmax_depth: 4\non_relation_error: notify\nmax_related_ids: 100\n"

	_, err := escribirComposicion(t, map[string]string{
		"objetos_x.yaml":            "name: X\nrelated_api: fantasma\nrelated_table: x\nendpoint: /v1/x\ncampos: []\n",
		"relaciones_proveedor.yaml": servicio,
	})
	if err == nil || !strings.Contains(err.Error(), "fantasma") {
		t.Errorf("CRUD inexistente debe abortar nombrando la API, obtuvo: %v", err)
	}

	_, err = escribirComposicion(t, map[string]string{
		"objetos_x.yaml":            "name: X\nrelated_api: terceros\nrelated_table: x\nendpoint: /v1/x\nnatural_key:\n  - Clave\ncampos:\n  - name: Clave\n    type: string\n    readable: true\n",
		"relaciones_proveedor.yaml": servicio,
	})
	if err == nil || !strings.Contains(err.Error(), "Clave") {
		t.Errorf("natural_key fuera de writable debe abortar nombrando el campo, obtuvo: %v", err)
	}
}
