package generador

import (
	"reflect"
	"testing"
)

func TestConstruirObjetos_FKEnExpandProduceAristaExpandibleEnPadre(t *testing.T) {
	objetos := mustConstruir(t, "- proveedores.contacto.proveedor_id\n", "", "", "")
	padre := buscarObjeto(t, objetos, "Proveedor")
	arista := buscarArista(t, padre, "Contacto")
	esperada := AristaConfig{
		Objeto:         "Contacto",
		Direction:      "related_to_local",
		Cardinality:    "many",
		LocalField:     "Id",
		RelatedField:   "ProveedorId",
		KeyPropagation: "fk",
		Depth:          1,
		MappedName:     "Contacto",
	}
	if !reflect.DeepEqual(arista, esperada) {
		t.Fatalf("arista = %+v\nse esperaba %+v", arista, esperada)
	}
	if hijo := buscarObjeto(t, objetos, "Contacto"); len(hijo.Referencias) != 0 {
		t.Fatalf("el hijo no debe tener arista inversa al padre: %+v", hijo.Referencias)
	}
}

func TestConstruirObjetos_FKFueraDeExpandEsIdentificadorPlano(t *testing.T) {
	objetos := mustConstruir(t, "", "", "", "")
	padre := buscarObjeto(t, objetos, "Proveedor")
	if len(padre.Referencias) != 0 {
		t.Fatalf("sin expand no debe haber aristas: %+v", padre.Referencias)
	}
	encontrado := false
	for _, c := range padre.Campos {
		if c.Name == "TerceroId" {
			encontrado = true
			if !c.Writable {
				t.Fatal("la FK plana debe ser escribible como identificador")
			}
		}
	}
	if !encontrado {
		t.Fatal("la FK plana debe permanecer como campo TerceroId")
	}
}

func TestConstruirObjetos_FKInboundMarcaAristaDeLlaveEntrante(t *testing.T) {
	objetos := mustConstruir(t, "", "- proveedores.proveedor.tercero_id\n", "", "")
	padre := buscarObjeto(t, objetos, "Proveedor")
	arista := buscarArista(t, padre, "Tercero")
	esperada := AristaConfig{
		Objeto:         "Tercero",
		Direction:      "local_to_related",
		Cardinality:    "one",
		LocalField:     "TerceroId",
		RelatedField:   "Id",
		KeyPropagation: "fk_inbound",
		Depth:          1,
		MappedName:     "Tercero",
	}
	if !reflect.DeepEqual(arista, esperada) {
		t.Fatalf("arista = %+v\nse esperaba %+v", arista, esperada)
	}
}

func TestConstruirObjetos_PKCompartidaProduceAristaDeEspecializacion(t *testing.T) {
	objetos := mustConstruir(t, "- proveedores.proveedor_natural.id\n", "", "", "")
	padre := buscarObjeto(t, objetos, "Proveedor")
	arista := buscarArista(t, padre, "ProveedorNatural")
	esperada := AristaConfig{
		Objeto:         "ProveedorNatural",
		Direction:      "related_to_local",
		Cardinality:    "one",
		LocalField:     "Id",
		RelatedField:   "Id",
		KeyPropagation: "shared_pk",
		Depth:          1,
		MappedName:     "ProveedorNatural",
	}
	if !reflect.DeepEqual(arista, esperada) {
		t.Fatalf("arista = %+v\nse esperaba %+v", arista, esperada)
	}
}

func TestConstruirObjetos_WriteFijaPoliticaYNaturalKeyPorObjeto(t *testing.T) {
	write := "proveedores.contacto:\n  creatable: true\n  updatable: true\n  deletable: false\n  natural_key:\n    - valor_especifico\n"
	objetos := mustConstruir(t, "- proveedores.contacto.proveedor_id\n", "", "", write)
	contacto := buscarObjeto(t, objetos, "Contacto")
	if !reflect.DeepEqual(contacto.NaturalKey, []string{"ValorEspecifico"}) {
		t.Fatalf("natural_key = %v, se esperaba [ValorEspecifico]", contacto.NaturalKey)
	}
	arista := buscarArista(t, buscarObjeto(t, objetos, "Proveedor"), "Contacto")
	esperada := PoliticaVerbos{Creatable: true, Updatable: true, Deletable: false}
	if arista.Policy != esperada {
		t.Fatalf("política = %+v, se esperaba %+v", arista.Policy, esperada)
	}
}

func TestConstruirObjetos_ErroresDeSemanticaFallan(t *testing.T) {
	mustConstruir(t, "- proveedores.contacto.proveedor_id\n", "", "", "")
	casos := map[string]struct {
		expand, fkInbound, blacklist, write string
	}{
		"expand sobre columna que no es FK": {expand: "- proveedores.contacto.valor_especifico\n"},
		"natural_key no escribible":         {write: "proveedores.contacto:\n  creatable: true\n  natural_key:\n    - id\n"},
		"expand en blacklist":               {expand: "- proveedores.contacto.proveedor_id\n", blacklist: "- proveedores.contacto.proveedor_id\n"},
		"fk_inbound en blacklist":           {fkInbound: "- proveedores.proveedor.tercero_id\n", blacklist: "- proveedores.proveedor.tercero_id\n"},
	}
	for nombre, c := range casos {
		if _, err := construirPrueba(t, c.expand, c.fkInbound, c.blacklist, c.write); err == nil {
			t.Errorf("%s: se esperaba error", nombre)
		}
	}
}
