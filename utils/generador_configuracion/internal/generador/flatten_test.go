package generador

import "testing"

func TestConstruirObjetos_FlattenMarcaAristaDirectaYSuprimeInversa(t *testing.T) {
	m, s, err := prepararPrueba(t, dbmlPrueba, map[string]string{
		"expand.yaml":  "- proveedores.proveedor_natural.id\n",
		"flatten.yaml": "- proveedores.proveedor_natural.id\n",
	})
	if err != nil {
		t.Fatalf("cargando sidecars: %v", err)
	}
	objetos, err := ConstruirObjetos(m, s)
	if err != nil {
		t.Fatalf("construyendo objetos: %v", err)
	}
	padre := buscarObjeto(t, objetos, "Proveedor")
	if arista := buscarArista(t, padre, "ProveedorNatural"); !arista.Flatten {
		t.Fatalf("la arista padre->hijo debe llevar flatten: true: %+v", arista)
	}
	if hijo := buscarObjeto(t, objetos, "ProveedorNatural"); len(hijo.Referencias) != 0 {
		t.Fatalf("la especialización no debe reexpandir el proveedor: %+v", hijo.Referencias)
	}
}

func TestConstruirObjetos_AristaSinEntradaFlattenNoSeMarca(t *testing.T) {
	m, s, err := prepararPrueba(t, dbmlPrueba, map[string]string{
		"expand.yaml": "- proveedores.proveedor_natural.id\n",
	})
	if err != nil {
		t.Fatalf("cargando sidecars: %v", err)
	}
	objetos, err := ConstruirObjetos(m, s)
	if err != nil {
		t.Fatalf("construyendo objetos: %v", err)
	}
	arista := buscarArista(t, buscarObjeto(t, objetos, "Proveedor"), "ProveedorNatural")
	if arista.Flatten {
		t.Fatalf("sin entrada en flatten la arista no debe llevar la bandera: %+v", arista)
	}
}

func TestCargarSidecars_FlattenNoPKCompartidaFalla(t *testing.T) {
	_, _, err := prepararPrueba(t, dbmlPrueba, map[string]string{
		"flatten.yaml": "- proveedores.contacto.proveedor_id\n",
	})
	if err == nil {
		t.Fatal("flatten sobre una FK que no es PK compartida debe fallar")
	}
}

func TestCargarSidecars_FlattenEnFKInboundFalla(t *testing.T) {
	_, _, err := prepararPrueba(t, dbmlPrueba, map[string]string{
		"flatten.yaml":    "- proveedores.proveedor_natural.id\n",
		"fk_inbound.yaml": "- proveedores.proveedor_natural.id\n",
	})
	if err == nil {
		t.Fatal("una columna en flatten y en fk_inbound debe fallar")
	}
}
