package generador

import (
	"fmt"
	"strings"
	"testing"
)

// rolEntrada es una entrada de field_roles para armar el YAML de prueba.
type rolEntrada struct {
	clave     string
	readable  bool
	writable  bool
	immutable bool
}

// yamlRoles convierte entradas en el contenido del sidecar field_roles.
func yamlRoles(entradas []rolEntrada) string {
	var b strings.Builder
	for _, e := range entradas {
		fmt.Fprintf(&b, "%s:\n  readable: %t\n  writable: %t\n  immutable: %t\n", e.clave, e.readable, e.writable, e.immutable)
	}
	return b.String()
}

// construirConRoles carga field_roles y construye los objetos del DBML de prueba.
func construirConRoles(t *testing.T, entradas []rolEntrada) ([]ObjetoConfig, error) {
	t.Helper()
	m, s, err := prepararPrueba(t, dbmlPrueba, map[string]string{
		"field_roles.yaml": yamlRoles(entradas),
	})
	if err != nil {
		return nil, err
	}
	return ConstruirObjetos(m, s)
}

// buscarCampo devuelve el campo del objeto con el nombre dado o falla la prueba.
func buscarCampo(t *testing.T, o ObjetoConfig, nombre string) CampoConfig {
	t.Helper()
	for _, c := range o.Campos {
		if c.Name == nombre {
			return c
		}
	}
	t.Fatalf("el objeto %q no tiene el campo %q", o.Name, nombre)
	return CampoConfig{}
}

func TestConstruirObjetos_FieldRolesComodinFijaRolEnTodosLosObjetos(t *testing.T) {
	objetos, err := construirConRoles(t, []rolEntrada{
		{clave: "ANY.ANY.activo", readable: false, writable: false},
	})
	if err != nil {
		t.Fatalf("construyendo objetos: %v", err)
	}
	for _, nombre := range []string{"Proveedor", "Tercero", "ProveedorNatural"} {
		activo := buscarCampo(t, buscarObjeto(t, objetos, nombre), "Activo")
		if activo.Readable || activo.Writable {
			t.Errorf("%s.Activo debe ser no legible y no escribible: %+v", nombre, activo)
		}
	}
}

func TestConstruirObjetos_FieldRolesPrecedenciaPorNivel(t *testing.T) {
	tier1 := rolEntrada{clave: "proveedores.proveedor_natural.cargo", readable: false, writable: false}
	tier2 := rolEntrada{clave: "proveedores.ANY.cargo", readable: true, writable: false}
	tier3 := rolEntrada{clave: "ANY.proveedor_natural.cargo", readable: false, writable: true}
	tier4 := rolEntrada{clave: "ANY.ANY.cargo", readable: true, writable: true}
	todas := []rolEntrada{tier1, tier2, tier3, tier4}

	casos := []struct {
		nombre             string
		entradas           []rolEntrada
		readable, writable bool
	}{
		{"gana esquema.tabla.columna", todas, false, false},
		{"sin tier 1 gana esquema.ANY.columna", todas[1:], true, false},
		{"sin tiers 1 y 2 gana ANY.tabla.columna", todas[2:], false, true},
		{"solo ANY.ANY.columna", todas[3:], true, true},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			objetos, err := construirConRoles(t, c.entradas)
			if err != nil {
				t.Fatalf("construyendo objetos: %v", err)
			}
			cargo := buscarCampo(t, buscarObjeto(t, objetos, "ProveedorNatural"), "Cargo")
			if cargo.Readable != c.readable || cargo.Writable != c.writable {
				t.Fatalf("Cargo = readable %t writable %t; se esperaba readable %t writable %t",
					cargo.Readable, cargo.Writable, c.readable, c.writable)
			}
		})
	}
}

func TestConstruirObjetos_FieldRolesCampoSinEntradaUsaRolPorOmision(t *testing.T) {
	objetos, err := construirConRoles(t, []rolEntrada{
		{clave: "ANY.ANY.activo", readable: false, writable: false},
	})
	if err != nil {
		t.Fatalf("construyendo objetos: %v", err)
	}
	cargo := buscarCampo(t, buscarObjeto(t, objetos, "ProveedorNatural"), "Cargo")
	if !cargo.Readable || !cargo.Writable {
		t.Fatalf("un campo sin entrada debe tomar el rol por omisión legible y escribible: %+v", cargo)
	}
}

func TestConstruirObjetos_FieldRolesEscribibleNoLegibleSeEmiteTalCual(t *testing.T) {
	objetos, err := construirConRoles(t, []rolEntrada{
		{clave: "ANY.ANY.usuario_modificacion_id", readable: false, writable: true},
	})
	if err != nil {
		t.Fatalf("construyendo objetos: %v", err)
	}
	campo := buscarCampo(t, buscarObjeto(t, objetos, "Proveedor"), "UsuarioModificacionId")
	if campo.Readable || !campo.Writable {
		t.Fatalf("UsuarioModificacionId debe ser escribible y no legible: %+v", campo)
	}
}

func TestCargarSidecars_FieldRolesColumnaInexistenteFalla(t *testing.T) {
	_, err := construirConRoles(t, []rolEntrada{
		{clave: "proveedores.proveedor_natural.noexiste", readable: true, writable: true},
	})
	if err == nil {
		t.Fatal("una entrada sin comodín que nombra una columna inexistente debe fallar")
	}
}

func TestCargarSidecars_FieldRolesComodinEnColumnaFalla(t *testing.T) {
	for _, clave := range []string{
		"proveedores.proveedor_natural.ANY",
		"proveedores.ANY.ANY",
		"ANY.ANY.ANY",
	} {
		_, err := construirConRoles(t, []rolEntrada{
			{clave: clave, readable: true, writable: true},
		})
		if err == nil {
			t.Errorf("%s: ANY en el segmento de columna debe fallar", clave)
			continue
		}
		if !strings.Contains(err.Error(), clave) {
			t.Errorf("%s: el error debe identificar la entrada: %v", clave, err)
		}
	}
}

func TestCargarSidecars_FieldRolesImmutableSinWritableFalla(t *testing.T) {
	_, err := construirConRoles(t, []rolEntrada{
		{clave: "proveedores.proveedor_natural.cargo", readable: true, writable: false, immutable: true},
	})
	if err == nil {
		t.Fatal("immutable: true con writable: false debe fallar, de forma coherente con el cargador")
	}
}
