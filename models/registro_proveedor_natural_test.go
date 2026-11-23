package models

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestNumeroConservaCerosYGrafia verifica que Numero (string) conserva ceros
// iniciales y que la clave DatosIdentificacion conserva su grafía.
func TestNumeroConservaCerosYGrafia(t *testing.T) {
	entrada := `{"TipoProveedor":1,"DatosBasicos":{"PrimerNombre":"Carlos","PrimerApellido":"González","FechaNacimiento":"1990-05-15T00:00:00Z"},"DatosIdentificacion":{"Numero":"0023456789","TipoDocumentoId":1,"CiudadExpedicion":123,"FechaExpedicion":"2008-06-20T00:00:00Z"}}`

	var in RegistroProveedorNatural
	if err := json.Unmarshal([]byte(entrada), &in); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if in.DatosIdentificacion.Numero != "0023456789" {
		t.Errorf("Numero = %q, se esperaba conservar ceros iniciales", in.DatosIdentificacion.Numero)
	}

	salida, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(salida), `"DatosIdentificacion"`) {
		t.Errorf("se esperaba conservar la grafía DatosIdentificacion: %s", salida)
	}
	if !strings.Contains(string(salida), `"Numero":"0023456789"`) {
		t.Errorf("Numero debería serializarse como string con ceros: %s", salida)
	}
}
