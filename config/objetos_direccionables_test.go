package config

import (
	"strings"
	"testing"
)

// Prueba que una definición de objeto YAML se parsea con nombre, tabla de campos
// tipada y referencias a otros objetos con su semántica de arista.
func TestDecodificarObjetoDireccionable_ParseaNombreCamposYReferencias(t *testing.T) {
	contenido := []byte(`
name: Contacto
campos:
  - name: NombreContacto
    type: string
    readable: true
    writable: true
referencias:
  - objeto: Tercero
    direction: related_to_local
    local_field: Id
    related_field: ProveedorId
    cardinality: many
    depth: 1
    mapped_name: Contacto
    key_propagation: fk
    policy:
      creatable: true
      updatable: true
      deletable: true
`)

	obj, err := DecodificarObjetoDireccionable(contenido)
	if err != nil {
		t.Fatalf("objeto válido rechazado: %v", err)
	}
	if obj.Name != "Contacto" {
		t.Errorf("Name = %q, se esperaba Contacto", obj.Name)
	}
	if len(obj.Campos) != 1 || obj.Campos[0].Name != "NombreContacto" || obj.Campos[0].Type != TipoString {
		t.Fatalf("Campos = %+v, se esperaba NombreContacto de tipo string", obj.Campos)
	}
	if len(obj.Referencias) != 1 {
		t.Fatalf("se esperaba 1 referencia, hay %d", len(obj.Referencias))
	}

	ref := obj.Referencias[0]
	if ref.Objeto != "Tercero" || ref.Direction != DirectionRelatedToLocal || ref.Cardinality != CardinalityMany {
		t.Errorf("referencia = {objeto:%q direction:%q cardinality:%q}", ref.Objeto, ref.Direction, ref.Cardinality)
	}
	if ref.Depth != 1 || ref.MappedName != "Contacto" || ref.LocalField != "Id" || ref.RelatedField != "ProveedorId" {
		t.Errorf("arista = depth:%d mapped_name:%q local:%q related:%q", ref.Depth, ref.MappedName, ref.LocalField, ref.RelatedField)
	}
	if ref.KeyPropagation != KeyPropagationFK {
		t.Errorf("KeyPropagation = %q, se esperaba %q", ref.KeyPropagation, KeyPropagationFK)
	}
	if !ref.Policy.Creatable || !ref.Policy.Updatable || !ref.Policy.Deletable {
		t.Errorf("Policy = %+v, se esperaban los tres verbos admitidos", ref.Policy)
	}
}

// Prueba que la referencia de un servicio a un objeto porta la arista completa.
func TestDecodificarReferenciasServicio_PortaLaArista(t *testing.T) {
	contenido := []byte(`
- objeto: Contacto
  direction: related_to_local
  local_field: Id
  related_field: ProveedorId
  cardinality: many
  depth: 1
  mapped_name: Contacto
  key_propagation: fk
  policy:
    creatable: true
    updatable: true
    deletable: true
`)

	refs, err := DecodificarReferenciasServicio("proveedor", contenido)
	if err != nil {
		t.Fatalf("referencia válida rechazada: %v", err)
	}
	if len(refs) != 1 {
		t.Fatalf("se esperaba 1 referencia, hay %d", len(refs))
	}
	ref := refs[0]
	if ref.Objeto != "Contacto" || ref.MappedName != "Contacto" || ref.Depth != 1 {
		t.Errorf("referencia = %+v", ref)
	}
	if ref.Direction != DirectionRelatedToLocal || ref.Cardinality != CardinalityMany || ref.KeyPropagation != KeyPropagationFK {
		t.Errorf("arista = direction:%q cardinality:%q key_propagation:%q", ref.Direction, ref.Cardinality, ref.KeyPropagation)
	}
}

// Prueba que una arista inválida aborta nombrando el objeto referenciado. Cubre
// dirección, cardinalidad, profundidad, mapped_name, campos de unión, propagación
// de llave, política de escritura sin propagación, y claves desconocidas.
func TestReferenciaConArista_AristaInvalidaAborta(t *testing.T) {
	base := func(extra string) []byte {
		return []byte("- objeto: Tercero\n  direction: local_to_related\n  local_field: TerceroId\n  related_field: Id\n  cardinality: one\n  depth: 1\n  mapped_name: Tercero\n" + extra)
	}

	casos := []struct {
		nombre    string
		contenido []byte
	}{
		{"dirección no reconocida", []byte("- objeto: Tercero\n  direction: lateral\n  local_field: TerceroId\n  related_field: Id\n  cardinality: one\n  depth: 1\n  mapped_name: Tercero\n")},
		{"cardinalidad no reconocida", []byte("- objeto: Tercero\n  direction: local_to_related\n  local_field: TerceroId\n  related_field: Id\n  cardinality: several\n  depth: 1\n  mapped_name: Tercero\n")},
		{"profundidad cero", []byte("- objeto: Tercero\n  direction: local_to_related\n  local_field: TerceroId\n  related_field: Id\n  cardinality: one\n  depth: 0\n  mapped_name: Tercero\n")},
		{"mapped_name vacío", []byte("- objeto: Tercero\n  direction: local_to_related\n  local_field: TerceroId\n  related_field: Id\n  cardinality: one\n  depth: 1\n  mapped_name: \"\"\n")},
		{"local_field vacío", []byte("- objeto: Tercero\n  direction: local_to_related\n  local_field: \"\"\n  related_field: Id\n  cardinality: one\n  depth: 1\n  mapped_name: Tercero\n")},
		{"related_field vacío", []byte("- objeto: Tercero\n  direction: local_to_related\n  local_field: TerceroId\n  related_field: \"\"\n  cardinality: one\n  depth: 1\n  mapped_name: Tercero\n")},
		{"key_propagation no reconocida", base("  key_propagation: join\n")},
		{"política escribible sin key_propagation", base("  policy:\n    creatable: true\n")},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			_, err := DecodificarReferenciasServicio("proveedor", c.contenido)
			if err == nil {
				t.Fatal("se esperaba error por arista inválida")
			}
			if !strings.Contains(err.Error(), "Tercero") {
				t.Errorf("el error debe nombrar el objeto referenciado, obtuvo: %v", err)
			}
		})
	}
}

// Prueba que un objeto con nombre vacío o con dos referencias hermanas con el mismo
// mapped_name aborta al validarse.
func TestDecodificarObjetoDireccionable_NombreVacioYMappedNameDuplicado(t *testing.T) {
	if _, err := DecodificarObjetoDireccionable([]byte("name: \"\"\ncampos: []\n")); err == nil {
		t.Error("se esperaba error por nombre de objeto vacío")
	}

	duplicado := []byte(`
name: Proveedor
referencias:
  - objeto: Tercero
    direction: local_to_related
    local_field: TerceroId
    related_field: Id
    cardinality: one
    depth: 1
    mapped_name: Dato
  - objeto: Contacto
    direction: related_to_local
    local_field: Id
    related_field: ProveedorId
    cardinality: many
    depth: 1
    mapped_name: Dato
`)
	_, err := DecodificarObjetoDireccionable(duplicado)
	if err == nil || !strings.Contains(err.Error(), "Dato") {
		t.Errorf("se esperaba error que nombre el mapped_name duplicado Dato, obtuvo: %v", err)
	}
}

// Prueba que un mismo objeto referenciado por dos referenciadores con aristas
// distintas produce nodos de árbol independientes: cada nodo conserva su arista y
// una copia propia de los campos, sin afectar a la definición compartida.
func TestCatalogoObjetos_AristasIndependientesEntreNodos(t *testing.T) {
	tercero := ObjetoDireccionable{
		Name: "Tercero",
		Campos: TablaCampos{
			{Name: "NombreCompleto", Type: TipoString, Readable: true, Writable: true},
		},
	}
	catalogo, err := NuevoCatalogoObjetos([]ObjetoDireccionable{tercero})
	if err != nil {
		t.Fatalf("catálogo válido rechazado: %v", err)
	}

	refA := ReferenciaObjeto{Objeto: "Tercero", Arista: Arista{
		Direction:      DirectionLocalToRelated,
		Cardinality:    CardinalityOne,
		LocalField:     "TerceroId",
		RelatedField:   "Id",
		KeyPropagation: KeyPropagationFKInbound,
		Policy:         WriteVerbPolicy{Creatable: true, Updatable: true},
		Depth:          1,
		MappedName:     "Tercero",
	}}
	refB := ReferenciaObjeto{Objeto: "Tercero", Arista: Arista{
		Direction:    DirectionRelatedToLocal,
		Cardinality:  CardinalityMany,
		LocalField:   "Id",
		RelatedField: "TerceroId",
		Depth:        2,
		MappedName:   "Terceros",
	}}

	nodoA, err := catalogo.Nodo(refA)
	if err != nil {
		t.Fatalf("nodo A rechazado: %v", err)
	}
	nodoB, err := catalogo.Nodo(refB)
	if err != nil {
		t.Fatalf("nodo B rechazado: %v", err)
	}

	if nodoA.Arista.Direction == nodoB.Arista.Direction || nodoA.Arista.MappedName == nodoB.Arista.MappedName {
		t.Fatal("los nodos deben conservar las aristas de su propia referencia")
	}

	// Mutar un nodo no debe alterar el otro, la referencia de origen ni la definición.
	nodoA.Arista.Depth = 9
	nodoA.Campos[0].Name = "Mutado"
	if nodoB.Arista.Depth != 2 || nodoB.Campos[0].Name != "NombreCompleto" {
		t.Errorf("mutar el nodo A alteró el nodo B: depth=%d campo=%q", nodoB.Arista.Depth, nodoB.Campos[0].Name)
	}
	if refA.Depth != 1 {
		t.Errorf("mutar el nodo A alteró la referencia de origen: depth=%d", refA.Depth)
	}
	definicion, _ := catalogo.Objeto("Tercero")
	if definicion.Campos[0].Name != "NombreCompleto" {
		t.Errorf("mutar el nodo A alteró la definición compartida: campo=%q", definicion.Campos[0].Name)
	}
}

// Prueba que un nombre de objeto duplicado aborta el arranque del catálogo nombrando
// el nombre duplicado, y que referenciar un objeto inexistente también aborta.
func TestCatalogoObjetos_NombreDuplicadoYReferenciaInexistente(t *testing.T) {
	a := ObjetoDireccionable{Name: "Tercero", Campos: TablaCampos{}}
	b := ObjetoDireccionable{Name: "Tercero", Campos: TablaCampos{}}

	_, err := NuevoCatalogoObjetos([]ObjetoDireccionable{a, b})
	if err == nil || !strings.Contains(err.Error(), "Tercero") {
		t.Fatalf("se esperaba error que nombre el objeto duplicado Tercero, obtuvo: %v", err)
	}

	catalogo, err := NuevoCatalogoObjetos([]ObjetoDireccionable{a})
	if err != nil {
		t.Fatalf("catálogo válido rechazado: %v", err)
	}
	ref := ReferenciaObjeto{Objeto: "NoExiste", Arista: Arista{
		Direction: DirectionLocalToRelated, Cardinality: CardinalityOne,
		LocalField: "X", RelatedField: "Id", Depth: 1, MappedName: "X",
	}}
	if _, err := catalogo.Nodo(ref); err == nil || !strings.Contains(err.Error(), "NoExiste") {
		t.Errorf("se esperaba error que nombre el objeto inexistente NoExiste, obtuvo: %v", err)
	}
}
