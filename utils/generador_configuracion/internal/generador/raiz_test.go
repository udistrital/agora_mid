package generador

import (
	"reflect"
	"testing"
)

// dbmlAnidado tiene una tabla que no es raíz (contacto) con su propio hijo
// (cuenta_bancaria), para probar que sus aristas permanecen en su objeto.
const dbmlAnidado = `
Table proveedores.proveedor {
  id serial [pk]
}

Table proveedores.contacto {
  id serial [pk]
  proveedor_id integer
}

Table proveedores.cuenta_bancaria {
  id serial [pk]
  contacto_id integer
}

Ref r1 {
  proveedores.proveedor.id < proveedores.contacto.proveedor_id
}

Ref r2 {
  proveedores.contacto.id < proveedores.cuenta_bancaria.contacto_id
}
`

// construirServiciosPrueba construye objetos y archivos de servicio desde dbml y los
// archivos de sidecar dados.
func construirServiciosPrueba(t *testing.T, dbml string, archivos map[string]string) ([]ObjetoConfig, []ServicioConfig, error) {
	t.Helper()
	m, s, err := prepararPrueba(t, dbml, archivos)
	if err != nil {
		return nil, nil, err
	}
	objetos, err := ConstruirObjetos(m, s)
	if err != nil {
		return nil, nil, err
	}
	servicios, err := ConstruirServicios(m, s)
	return objetos, servicios, err
}

func TestConstruirServicios_AristaDeRaizVaAlArchivoDeServicioYNoAlObjetoRaiz(t *testing.T) {
	objetos, servicios, err := construirServiciosPrueba(t, dbmlPrueba, map[string]string{
		"expand.yaml":  "- proveedores.contacto.proveedor_id\n",
		"service.yaml": servicioPrueba,
	})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if raiz := buscarObjeto(t, objetos, "Proveedor"); len(raiz.Referencias) != 0 {
		t.Fatalf("el objeto raíz debe salir con referencias vacías: %+v", raiz.Referencias)
	}
	if len(servicios) != 1 {
		t.Fatalf("servicios = %d, se esperaba 1", len(servicios))
	}
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
	if !reflect.DeepEqual(servicios[0].Referencias, []AristaConfig{esperada}) {
		t.Fatalf("referencias del servicio = %+v\nse esperaba [%+v]", servicios[0].Referencias, esperada)
	}
}

func TestConstruirServicios_PoliticaDeRaizVaAWritePolicyYNaturalKeyAlObjeto(t *testing.T) {
	write := "proveedores.proveedor:\n  creatable: true\n  updatable: true\n  deletable: false\n  natural_key:\n    - tercero_id\n"
	objetos, servicios, err := construirServiciosPrueba(t, dbmlPrueba, map[string]string{
		"write.yaml":   write,
		"service.yaml": servicioPrueba,
	})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if len(servicios) != 1 || servicios[0].Write == nil {
		t.Fatalf("el servicio debe llevar write.policy: %+v", servicios)
	}
	esperada := PoliticaVerbos{Creatable: true, Updatable: true, Deletable: false}
	if servicios[0].Write.Policy != esperada {
		t.Fatalf("write.policy = %+v, se esperaba %+v", servicios[0].Write.Policy, esperada)
	}
	raiz := buscarObjeto(t, objetos, "Proveedor")
	if !reflect.DeepEqual(raiz.NaturalKey, []string{"TerceroId"}) {
		t.Fatalf("natural_key de la raíz = %v, se esperaba [TerceroId]", raiz.NaturalKey)
	}
	for _, o := range objetos {
		for _, a := range o.Referencias {
			if a.Objeto == "Proveedor" {
				t.Fatalf("ninguna arista debe apuntar a la raíz para llevar su política: %+v", a)
			}
		}
	}
}

func TestConstruirServicios_AristaDeObjetoNoRaizPermaneceEnSuObjeto(t *testing.T) {
	objetos, servicios, err := construirServiciosPrueba(t, dbmlAnidado, map[string]string{
		"expand.yaml":  "- proveedores.contacto.proveedor_id\n- proveedores.cuenta_bancaria.contacto_id\n",
		"service.yaml": "proveedores.proveedor:\n  service: proveedor\n",
	})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	contacto := buscarObjeto(t, objetos, "Contacto")
	buscarArista(t, contacto, "CuentaBancaria")
	if len(servicios) != 1 {
		t.Fatalf("servicios = %d, se esperaba 1", len(servicios))
	}
	for _, a := range servicios[0].Referencias {
		if a.Objeto == "CuentaBancaria" {
			t.Fatal("la arista de un objeto no raíz no debe trasladarse al servicio")
		}
	}
}
