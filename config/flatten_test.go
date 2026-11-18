package config

import "testing"

// Una arista declarada con flatten: true se carga con la bandera activa; una sin la
// clave queda en falso.
func TestDecodificarReferenciasServicio_FlattenMarcaAristaDirecta(t *testing.T) {
	contenido := []byte(`- objeto: ProveedorJuridico
  direction: related_to_local
  cardinality: one
  local_field: Id
  related_field: Id
  key_propagation: shared_pk
  policy:
    creatable: true
    updatable: true
    deletable: false
  depth: 2
  mapped_name: ProveedorJuridico
  flatten: true
- objeto: Representacion
  direction: related_to_local
  cardinality: many
  local_field: Id
  related_field: ProveedorId
  key_propagation: fk
  policy:
    creatable: true
    updatable: true
    deletable: false
  depth: 2
  mapped_name: Representacion
`)
	refs, err := DecodificarReferenciasServicio("proveedor", contenido)
	if err != nil {
		t.Fatalf("las referencias deben cargar sin error: %v", err)
	}
	if len(refs) != 2 {
		t.Fatalf("se esperaban 2 referencias; hay %d", len(refs))
	}
	if !refs[0].Flatten {
		t.Errorf("la arista con flatten: true debe quedar marcada; arista=%+v", refs[0].Arista)
	}
	if refs[1].Flatten {
		t.Errorf("la arista sin flatten no debe quedar marcada; arista=%+v", refs[1].Arista)
	}
}
