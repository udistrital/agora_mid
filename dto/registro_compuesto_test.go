package dto

import (
	"encoding/json"
	"testing"
)

func TestRegistroPersonaNaturalContratoCompuesto(t *testing.T) {
	payload := []byte(`{
		"DescripcionPortafolio": "Desarrollo de software",
		"PerfilDeclarado": {"Id": 7},
		"ExperienciaLaboralMeses": 48,
		"ExperienciaProfesionalMeses": 36,
		"Tercero": {
			"Id": 81,
			"NombreCompleto": "Ana Maria Perez",
			"CodigoTipoDocumento": "CC",
			"NumeroDocumento": "123456"
		}
	}`)

	var req RegistroPersonaNaturalRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		t.Fatalf("no se pudo procesar el registro natural: %v", err)
	}

	if req.DatosTerceros.TerceroID != 81 {
		t.Errorf("TerceroID = %d; se esperaba 81", req.DatosTerceros.TerceroID)
	}
	if req.DatosTerceros.NumeroDocumento != "123456" {
		t.Errorf("documento inesperado: %q", req.DatosTerceros.NumeroDocumento)
	}
	if req.ProveedorNatural.PerfilDeclaradoID != 7 {
		t.Errorf("PerfilDeclaradoID = %d; se esperaba 7", req.ProveedorNatural.PerfilDeclaradoID)
	}
	if req.ProveedorNatural.ExperienciaLaboralMeses != 48 {
		t.Errorf("ExperienciaLaboralMeses = %d; se esperaba 48", req.ProveedorNatural.ExperienciaLaboralMeses)
	}
}

func TestRegistroPersonaJuridicaContratoCompuesto(t *testing.T) {
	payload := []byte(`{
		"DescripcionPortafolio": "Servicios empresariales",
		"NombreComercial": "Empresa de Prueba",
		"MatriculaMercantil": 12345,
		"FechaConstitucion": "2020-01-15",
		"FechaRenovacion": "2026-01-15",
		"ReportaBeneficiosFinales": true,
		"CotizaBolsa": false,
		"RequiereRevisorFiscal": true,
		"Tercero": {
			"Id": 91,
			"NombreCompleto": "Empresa de Prueba",
			"CodigoTipoDocumento": "NIT",
			"NumeroDocumento": "900123456"
		}
	}`)

	var req RegistroPersonaJuridicaRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		t.Fatalf("no se pudo procesar el registro jurídico: %v", err)
	}

	if req.DatosTerceros.TerceroID != 91 {
		t.Errorf("TerceroID = %d; se esperaba 91", req.DatosTerceros.TerceroID)
	}
	if req.ProveedorJuridico.NombreComercial != "Empresa de Prueba" {
		t.Errorf("NombreComercial inesperado: %q", req.ProveedorJuridico.NombreComercial)
	}
	if !req.ProveedorJuridico.ReportaBeneficiosFinales {
		t.Error("ReportaBeneficiosFinales debería ser true")
	}
}

func TestRegistroCompuestoRechazaContratoSinTercero(t *testing.T) {
	payload := []byte(`{
		"terceros": {"tercero_id": 81},
		"proveedor": {},
		"proveedor_natural": {}
	}`)

	var req RegistroPersonaNaturalRequest
	if err := json.Unmarshal(payload, &req); err == nil {
		t.Fatal("se esperaba un error porque falta el objeto Tercero")
	}
}

func TestRegistroCompuestoRechazaCamposNoPermitidos(t *testing.T) {
	casos := []struct {
		nombre  string
		payload string
	}{
		{
			nombre:  "campo desconocido en la raiz",
			payload: `{"CampoInterno":"x","Tercero":{"Id":81}}`,
		},
		{
			nombre:  "campo desconocido en Tercero",
			payload: `{"Tercero":{"Id":81,"CampoInterno":"x"}}`,
		},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			var req RegistroPersonaNaturalRequest
			if err := json.Unmarshal([]byte(caso.payload), &req); err == nil {
				t.Fatal("se esperaba rechazar el campo no permitido")
			}
		})
	}
}
