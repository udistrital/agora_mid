package generador

import (
	"strings"
	"testing"
)

func TestConstruirObjetos_EntradaCadenaOMapaFijaProfundidadDeArista(t *testing.T) {
	expand := "- entrada: proveedores.contacto.proveedor_id\n  depth: 3\n- proveedores.proveedor_natural.id\n"
	fkInbound := "- entrada: proveedores.proveedor.tercero_id\n  depth: 2\n"
	objetos := mustConstruir(t, expand, fkInbound, "", "")

	padre := buscarObjeto(t, objetos, "Proveedor")
	if d := buscarArista(t, padre, "Contacto").Depth; d != 3 {
		t.Errorf("profundidad de Contacto = %d, se esperaba 3", d)
	}
	if d := buscarArista(t, padre, "ProveedorNatural").Depth; d != 1 {
		t.Errorf("profundidad por omisión de ProveedorNatural = %d, se esperaba 1", d)
	}
	if d := buscarArista(t, padre, "Tercero").Depth; d != 2 {
		t.Errorf("profundidad de Tercero = %d, se esperaba 2", d)
	}
}

func TestConstruirObjetos_EntradaDeMapaMalFormadaFalla(t *testing.T) {
	casos := map[string]struct {
		expand, entrada string
	}{
		"depth cero": {
			expand:  "- entrada: proveedores.contacto.proveedor_id\n  depth: 0\n",
			entrada: "proveedores.contacto.proveedor_id",
		},
		"clave desconocida": {
			expand:  "- entrada: proveedores.contacto.proveedor_id\n  profundidad: 2\n",
			entrada: "profundidad",
		},
		"sin campo entrada": {
			expand:  "- depth: 2\n",
			entrada: "entrada",
		},
	}
	for nombre, c := range casos {
		_, err := construirPrueba(t, c.expand, "", "", "")
		if err == nil {
			t.Errorf("%s: se esperaba error", nombre)
			continue
		}
		if !strings.Contains(err.Error(), c.entrada) {
			t.Errorf("%s: el error %q no identifica %q", nombre, err.Error(), c.entrada)
		}
	}
}
