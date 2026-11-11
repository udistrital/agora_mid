package test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Una lectura sin fields expande por configuración real: cada entidad anidada
// conserva su Id implícito y la rama jurídica se poda para un proveedor natural.
// Las advertencias de expansión que devuelva el motor deben ser solo de
// profundidad, sin errores de otra clase.
func TestIntegracionConsultaSinFieldsExpandeConIdImplicitoYPodaRamaJuridica(t *testing.T) {
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
			"ExperienciaLaboralMeses": float64(12),
		}},
	})
	router := routerProveedor(t, crud)

	req := httptest.NewRequest(http.MethodPost, "/v1/proveedor/consulta",
		strings.NewReader(`{"id":101}`))
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

	tercero, _ := env.Data["Tercero"].(map[string]any)
	if tercero == nil {
		t.Fatalf("falta Tercero expandido; data=%+v", env.Data)
	}
	if id, _ := tercero["Id"].(float64); int(id) != 201 {
		t.Errorf("Tercero debe conservar su Id implícito 201; tercero=%+v", tercero)
	}
	if nombre, _ := tercero["NombreCompleto"].(string); nombre != "Ana Pérez" {
		t.Errorf("Tercero.NombreCompleto (readable) debe entregarse; tercero=%+v", tercero)
	}

	natural, _ := env.Data["ProveedorNatural"].(map[string]any)
	if natural == nil {
		t.Fatalf("falta ProveedorNatural expandido; data=%+v", env.Data)
	}
	if id, _ := natural["Id"].(float64); int(id) != 101 {
		t.Errorf("ProveedorNatural debe conservar su Id implícito 101; natural=%+v", natural)
	}
	if meses, _ := natural["ExperienciaLaboralMeses"].(float64); int(meses) != 12 {
		t.Errorf("ProveedorNatural.ExperienciaLaboralMeses (readable) debe entregarse; natural=%+v", natural)
	}

	if juridico, ok := env.Data["ProveedorJuridico"]; ok && juridico != nil {
		t.Errorf("la rama jurídica debe podarse para un proveedor natural; data=%+v", env.Data)
	}
}
