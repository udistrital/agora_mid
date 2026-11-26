package models

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestRegistroTerceroJuridicoConservaContrato verifica que el DTO jurídico
// acepta RazonSocial y conserva Numero como string.
func TestRegistroTerceroJuridicoConservaContrato(t *testing.T) {
	entrada := `{
		"DatosBasicos": {
			"RazonSocial": "ACME S.A.S.",
			"LugarOrigen": 149,
			"TipoContribuyenteId": 2
		},
		"DatosIdentificacion": {
			"Numero": "00900123456",
			"TipoDocumentoId": 1,
			"DigitoVerificacion": 7
		}
	}`

	var in RegistroTerceroJuridico
	if err := json.Unmarshal([]byte(entrada), &in); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if in.DatosBasicos.RazonSocial != "ACME S.A.S." {
		t.Errorf("RazonSocial = %q", in.DatosBasicos.RazonSocial)
	}
	if in.DatosBasicos.LugarOrigen == nil || *in.DatosBasicos.LugarOrigen != 149 {
		t.Errorf("LugarOrigen inesperado: %v", in.DatosBasicos.LugarOrigen)
	}
	if in.DatosIdentificacion.Numero != "00900123456" {
		t.Errorf("Numero = %q, se esperaba conservar ceros iniciales", in.DatosIdentificacion.Numero)
	}

	salida, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(salida), `"RazonSocial":"ACME S.A.S."`) {
		t.Errorf("se esperaba RazonSocial en la salida: %s", salida)
	}
	if !strings.Contains(string(salida), `"Numero":"00900123456"`) {
		t.Errorf("Numero debería serializarse como string con ceros: %s", salida)
	}
}
