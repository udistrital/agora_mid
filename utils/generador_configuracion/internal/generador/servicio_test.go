package generador

import (
	"reflect"
	"strings"
	"testing"
)

func TestCargarSidecars_ServicioAportaParametrosPorRaiz(t *testing.T) {
	_, s, err := prepararPrueba(t, dbmlPrueba, map[string]string{"service.yaml": servicioPrueba})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	got, ok := s.Service["proveedores.proveedor"]
	if !ok {
		t.Fatal("falta la entrada de la tabla raíz proveedores.proveedor")
	}
	esperado := ServicioSidecar{
		Service:          "proveedor",
		MaxDepth:         4,
		OnRelationError:  "notify",
		MaxRelatedIDs:    100,
		FilterableFields: []string{"id", "tercero_id"},
	}
	if !reflect.DeepEqual(got, esperado) {
		t.Fatalf("servicio = %+v\nse esperaba %+v", got, esperado)
	}
}

func TestCargarSidecars_ServicioFallaIdentificandoLaEntradaInvalida(t *testing.T) {
	casos := map[string]struct {
		servicio, entrada string
	}{
		"clave que no es tabla del DBML": {
			servicio: "proveedores.tabla_fantasma:\n  service: x\n",
			entrada:  "proveedores.tabla_fantasma",
		},
		"filterable_fields que no es columna de la raíz": {
			servicio: "proveedores.proveedor:\n  service: proveedor\n  filterable_fields:\n    - columna_fantasma\n",
			entrada:  "columna_fantasma",
		},
	}
	for nombre, c := range casos {
		_, _, err := prepararPrueba(t, dbmlPrueba, map[string]string{"service.yaml": c.servicio})
		if err == nil {
			t.Errorf("%s: se esperaba error", nombre)
			continue
		}
		if !strings.Contains(err.Error(), c.entrada) {
			t.Errorf("%s: el error %q no identifica la entrada %q", nombre, err.Error(), c.entrada)
		}
	}
}

// Una entrada del sidecar service sin el campo service falla identificando la
// entrada responsable.
func TestCargarSidecars_ServicioSinNombreFallaIdentificandoLaEntrada(t *testing.T) {
	_, _, err := prepararPrueba(t, dbmlPrueba, map[string]string{
		"service.yaml": "proveedores.proveedor:\n  max_depth: 4\n",
	})
	if err == nil {
		t.Fatal("se esperaba error por entrada de service sin service")
	}
	for _, frag := range []string{"proveedores.proveedor", "service"} {
		if !strings.Contains(err.Error(), frag) {
			t.Errorf("el error %q no contiene %q", err.Error(), frag)
		}
	}
}

// dbmlConTablaSinCRUD añade una tabla de un esquema sin CRUD mapeado al modelo de
// prueba, para probar que no puede ser raíz de un servicio.
const dbmlConTablaSinCRUD = dbmlPrueba + `
Table otro.tabla {
  id serial [pk]
}
`

// Una raíz de servicio cuyo esquema no tiene CRUD mapeado falla identificando la
// entrada responsable.
func TestConstruirServicios_RaizDeEsquemaSinCRUDFallaIdentificandolaEntrada(t *testing.T) {
	_, _, err := construirServiciosPrueba(t, dbmlConTablaSinCRUD, map[string]string{
		"service.yaml": "otro.tabla:\n  service: otra\n",
	})
	if err == nil {
		t.Fatal("se esperaba error por raíz fuera de un CRUD mapeado")
	}
	if !strings.Contains(err.Error(), "otro.tabla") {
		t.Errorf("el error %q no identifica la raíz otro.tabla", err.Error())
	}
}
