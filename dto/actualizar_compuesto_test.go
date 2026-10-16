package dto

import (
	"encoding/json"
	"testing"
)

func TestActualizarPersonaMapeaContratoNatural(t *testing.T) {
	payload := []byte(`{
		"DescripcionPortafolio": "Servicios tecnológicos",
		"PerfilDeclarado": {"Id": 7},
		"ExperienciaLaboralMeses": 24,
		"ExperienciaProfesionalMeses": 36,
		"Tercero": {
			"Id": 25,
			"NombreCompleto": "Ana Perez",
			"DatosIdentificacion": [{
				"Id": 99,
				"Numero": "123456",
				"TipoDocumentoId": {"CodigoAbreviacion": "CC"},
				"DigitoVerificacion": 9,
				"CiudadExpedicion": 11001,
				"FechaExpedicion": "2020-01-02"
			}]
		}
	}`)

	var req ActualizarPersonaRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		t.Fatalf("error al decodificar la solicitud: %v", err)
	}

	if req.TerceroID != 25 {
		t.Errorf("TerceroID = %d; esperado 25", req.TerceroID)
	}
	if string(req.Terceros["nombre_completo"]) != `"Ana Perez"` {
		t.Errorf("nombre_completo inesperado: %s", req.Terceros["nombre_completo"])
	}
	if string(req.DatosIdentificacion["id"]) != "99" {
		t.Errorf("identificación inesperada: %s", req.DatosIdentificacion["id"])
	}
	if string(req.DatosIdentificacion["numero_documento"]) != `"123456"` {
		t.Errorf("número documental inesperado: %s", req.DatosIdentificacion["numero_documento"])
	}
	if string(req.DatosIdentificacion["codigo_tipo_documento"]) != `"CC"` {
		t.Errorf("tipo documental inesperado: %s", req.DatosIdentificacion["codigo_tipo_documento"])
	}
	if string(req.Proveedor["descripcion_portafolio"]) != `"Servicios tecnológicos"` {
		t.Errorf("descripción inesperada: %s", req.Proveedor["descripcion_portafolio"])
	}
	if string(req.ProveedorNatural["perfil_declarado_id"]) != "7" {
		t.Errorf("perfil inesperado: %s", req.ProveedorNatural["perfil_declarado_id"])
	}
	if string(req.ProveedorNatural["experiencia_laboral_meses"]) != "24" {
		t.Errorf("experiencia inesperada: %s", req.ProveedorNatural["experiencia_laboral_meses"])
	}
}

func TestActualizarPersonaMapeaContratoJuridico(t *testing.T) {
	payload := []byte(`{
		"NombreComercial": "Empresa de Prueba",
		"MatriculaMercantil": 12345,
		"FechaConstitucion": "2020-01-15",
		"FechaRenovacion": "2026-01-15",
		"ReportaBeneficiosFinales": true,
		"CotizaBolsa": false,
		"RequiereRevisorFiscal": true,
		"Tercero": {"Id": 42}
	}`)

	var req ActualizarPersonaRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		t.Fatalf("error al decodificar la solicitud: %v", err)
	}

	if req.TerceroID != 42 {
		t.Errorf("TerceroID = %d; esperado 42", req.TerceroID)
	}
	if string(req.ProveedorJuridico["nombre_comercial"]) != `"Empresa de Prueba"` {
		t.Errorf("nombre comercial inesperado: %s", req.ProveedorJuridico["nombre_comercial"])
	}
	if string(req.ProveedorJuridico["matricula_mercantil"]) != "12345" {
		t.Errorf("matrícula inesperada: %s", req.ProveedorJuridico["matricula_mercantil"])
	}
	if string(req.ProveedorJuridico["reporta_beneficios_finales"]) != "true" {
		t.Errorf("valor booleano inesperado: %s", req.ProveedorJuridico["reporta_beneficios_finales"])
	}
}

func TestActualizarPersonaRechazaVariasIdentificaciones(t *testing.T) {
	payload := []byte(`{
		"Tercero": {
			"Id": 25,
			"DatosIdentificacion": [
				{"Id": 99},
				{"Id": 100}
			]
		}
	}`)

	var req ActualizarPersonaRequest
	if err := json.Unmarshal(payload, &req); err == nil {
		t.Fatal("se esperaba un error cuando hay varias identificaciones")
	}
}

func TestActualizarPersonaRechazaCamposNoPermitidos(t *testing.T) {
	casos := []struct {
		nombre  string
		payload string
	}{
		{
			nombre:  "campo desconocido en proveedor",
			payload: `{"Id":81,"Tercero":{"Id":25},"CampoInterno":"x"}`,
		},
		{
			nombre:  "campo desconocido en Tercero",
			payload: `{"Tercero":{"Id":25,"CampoInterno":"x"}}`,
		},
		{
			nombre:  "campo desconocido en identificación",
			payload: `{"Tercero":{"Id":25,"DatosIdentificacion":[{"Id":99,"CampoInterno":"x"}]}}`,
		},
		{
			nombre:  "contrato antiguo con bloques separados",
			payload: `{"tercero_id":25,"terceros":{"nombre_completo":"Ana"}}`,
		},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			var req ActualizarPersonaRequest
			if err := json.Unmarshal([]byte(caso.payload), &req); err == nil {
				t.Fatal("se esperaba rechazar el contrato")
			}
		})
	}
}

func TestActualizarPersonaRechazaCambioNumeroSinTipoDocumento(t *testing.T) {
	payload := []byte(`{
		"Tercero": {
			"Id": 25,
			"DatosIdentificacion": [
				{"Numero": "987654"}
			]
		}
	}`)

	var req ActualizarPersonaRequest
	if err := json.Unmarshal(payload, &req); err == nil {
		t.Fatal("se esperaba un error al cambiar Numero sin TipoDocumentoId")
	}
}
