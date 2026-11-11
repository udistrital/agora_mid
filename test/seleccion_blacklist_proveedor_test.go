package test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/udistrital/agora_mid/config"
)

// consultaProveedorCuerpo envía POST /v1/proveedor/consulta con el cuerpo indicado y
// decodifica el envoltorio. Solo acepta 200 o 206, y en 206 exige que los avisos sean
// de profundidad.
func consultaProveedorCuerpo(t *testing.T, router http.Handler, cuerpo string) respuestaEnvoltorio {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/v1/proveedor/consulta", strings.NewReader(cuerpo))
	req.Header.Set("X-Autor-Id", "9")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var env respuestaEnvoltorio
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("respuesta no es un envoltorio JSON válido: %v; body=%s", err, rec.Body.String())
	}
	if rec.Code != http.StatusOK && rec.Code != http.StatusPartialContent {
		t.Fatalf("status = %d; esperado 200 o 206. body=%s", rec.Code, rec.Body.String())
	}
	if env.MetaData != nil {
		for _, e := range env.MetaData.Errors {
			if kind, _ := e["kind"].(string); kind != "depth" {
				t.Errorf("solo se esperan avisos de profundidad; aviso=%+v", e)
			}
		}
	}
	return env
}

// Con fields que selecciona solo un escalar de una relación, la entidad anidada se
// entrega sin su Id porque la selección no lo menciona, y las relaciones hermanas no
// solicitadas no aparecen en la respuesta.
func TestIntegracionConsultaFieldsEscalarOmiteIdYHermanas(t *testing.T) {
	crud := NuevoCRUDSimulado(t, map[string][]map[string]any{
		"GET /v1/proveedor": {{
			"Id": float64(101), "TerceroId": float64(201),
			"TipoRegistroId": float64(tipoRegistroNatural), "DescripcionPortafolio": "Servicios",
		}},
		"GET /v1/tercero": {{
			"Id": float64(201), "NombreCompleto": "Ana Pérez",
			"TipoContribuyenteId": float64(2),
		}},
		"GET /v1/proveedor_natural": {{
			"Id": float64(101), "PerfilDeclarado": float64(3),
		}},
	})
	router := routerProveedor(t, crud)

	env := consultaProveedorCuerpo(t, router, `{"id":101,"fields":["Tercero.NombreCompleto"]}`)

	tercero, _ := env.Data["Tercero"].(map[string]any)
	if tercero == nil {
		t.Fatalf("falta Tercero seleccionado; data=%+v", env.Data)
	}
	if nombre, _ := tercero["NombreCompleto"].(string); nombre != "Ana Pérez" {
		t.Errorf("Tercero.NombreCompleto debe entregarse; tercero=%+v", tercero)
	}
	if _, ok := tercero["Id"]; ok {
		t.Errorf("fields sin Tercero.Id no debe entregar el Id de Tercero; tercero=%+v", tercero)
	}
	if _, ok := env.Data["ProveedorNatural"]; ok {
		t.Errorf("una relación hermana no solicitada no debe aparecer; data=%+v", env.Data)
	}
}

// La lista negra global de la configuración real suprime el campo vetado de la entidad
// expandida y conserva su Id implícito, sin aviso en MetaData.
func TestIntegracionConsultaListaNegraSuprimeCampoVetado(t *testing.T) {
	crud := NuevoCRUDSimulado(t, map[string][]map[string]any{
		"GET /v1/proveedor": {{
			"Id": float64(101), "TerceroId": float64(201),
			"TipoRegistroId": float64(tipoRegistroNatural),
		}},
		"GET /v1/tercero": {{
			"Id": float64(201), "NombreCompleto": "Ana Pérez",
			"TipoContribuyenteId": float64(2),
		}},
	})
	blacklist := config.GlobalBlacklist{
		"terceros": {"tercero": {"NombreCompleto"}},
	}
	router := routerProveedorConBlacklist(t, crud, blacklist)

	env := consultaProveedorCuerpo(t, router, `{"id":101}`)

	tercero, _ := env.Data["Tercero"].(map[string]any)
	if tercero == nil {
		t.Fatalf("falta Tercero expandido; data=%+v", env.Data)
	}
	if _, ok := tercero["NombreCompleto"]; ok {
		t.Errorf("el campo vetado NombreCompleto no debe entregarse; tercero=%+v", tercero)
	}
	if id, _ := tercero["Id"].(float64); int(id) != 201 {
		t.Errorf("el Id implícito de Tercero debe conservarse; tercero=%+v", tercero)
	}
	if env.MetaData != nil {
		for _, e := range env.MetaData.Errors {
			msg, _ := e["message"].(string)
			if kind, _ := e["kind"].(string); kind == "field_blacklisted" || strings.Contains(strings.ToLower(msg), "lista negra") {
				t.Errorf("la supresión por lista negra debe ser silenciosa; aviso=%+v", e)
			}
		}
	}
}
