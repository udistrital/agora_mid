package config

import (
	"os"
	"testing"
)

// cargarObjetoReal decodifica una definición de objeto del árbol real de configuración.
func cargarObjetoReal(t *testing.T, ruta string) ObjetoDireccionable {
	t.Helper()
	contenido, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatalf("no se pudo leer %s: %v", ruta, err)
	}
	obj, err := DecodificarObjetoDireccionable(contenido)
	if err != nil {
		t.Fatalf("%s debe decodificar sin error: %v", ruta, err)
	}
	return obj
}

// nombresCampos devuelve el conjunto de nombres de la tabla de campos del objeto.
func nombresCampos(t TablaCampos) map[string]bool {
	nombres := map[string]bool{}
	for _, c := range t {
		nombres[c.Name] = true
	}
	return nombres
}

// verificarCamposPresentes falla por cada campo esperado que falte en la tabla.
func verificarCamposPresentes(t *testing.T, objeto string, tabla TablaCampos, esperados []string) {
	t.Helper()
	nombres := nombresCampos(tabla)
	for _, campo := range esperados {
		if !nombres[campo] {
			t.Errorf("%s: falta el campo %q de la tabla de campos tipada", objeto, campo)
		}
	}
}

// La tabla de campos tipada de ProveedorNatural contiene las columnas de persona natural
// del modelo #58 (sin FK de la PK compartida, que el aplanado resuelve).
func TestModeloExtendidoProveedorNaturalTablaDeCampos(t *testing.T) {
	obj := cargarObjetoReal(t, "gestion_personas_crud/objetos_proveedor_natural.yaml")
	verificarCamposPresentes(t, "ProveedorNatural", obj.Campos, []string{
		"PaisNacimientoId", "Cargo", "MedicinaPrepagada", "ValorUvtPrepagada",
		"InteresViviendaAfc", "DependienteHijoMejorEdad", "DependienteConyuge",
		"DependientePadreOHermano", "Pensionado", "ValorAfc", "ResponsableIva",
		"ResideColombia", "PensionVoluntaria", "ValorPensionVoluntaria",
		"ExperienciaLaboral", "ExperienciaProfesional",
	})
}

// La tabla de campos tipada de ProveedorJuridico contiene las columnas de persona jurídica.
func TestModeloExtendidoProveedorJuridicoTablaDeCampos(t *testing.T) {
	obj := cargarObjetoReal(t, "gestion_personas_crud/objetos_proveedor_juridico.yaml")
	verificarCamposPresentes(t, "ProveedorJuridico", obj.Campos, []string{
		"ProcedenciaEmpresaId", "CiudadOrigenId", "PaisOrigenId", "CodigoPostal",
		"TipoConformacionId", "ExclusividadProducto", "RegimenContributivo", "Pyme",
		"RegistroMercantil", "SujetoRetencion", "AgenteRetenedor", "ResponsableIca",
		"ResponsableIva",
	})
}

// El proveedor declara el campo Consentimiento del modelo #58.
func TestModeloExtendidoProveedorDeclaraConsentimiento(t *testing.T) {
	obj := cargarObjetoReal(t, "gestion_personas_crud/objetos_proveedor.yaml")
	verificarCamposPresentes(t, "Proveedor", obj.Campos, []string{"Consentimiento"})
}

// CuentaBancaria referencia su tipo con tipo_cuenta_id y no declara el campo fantasma
// tipo_cuenta_bancaria_id.
func TestModeloExtendidoCuentaBancariaUsaTipoCuentaId(t *testing.T) {
	obj := cargarObjetoReal(t, "gestion_personas_crud/objetos_cuenta_bancaria.yaml")
	nombres := nombresCampos(obj.Campos)
	if !nombres["TipoCuentaId"] {
		t.Error("CuentaBancaria debe declarar TipoCuentaId")
	}
	if nombres["TipoCuentaBancariaId"] {
		t.Error("CuentaBancaria no debe declarar el campo fantasma TipoCuentaBancariaId")
	}
}
