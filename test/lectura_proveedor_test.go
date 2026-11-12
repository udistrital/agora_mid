package test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/udistrital/agora_mid/config"
)

// Una lectura sin fields expande por configuración real: cada entidad anidada
// conserva su Id implícito y la rama jurídica se poda para un proveedor natural.
// Las advertencias de expansión que devuelva el motor deben ser solo de
// profundidad, sin errores de otra clase.
func TestIntegracionConsultaSinFieldsExpandeConIdImplicitoYPodaRamaJuridica(t *testing.T) {
	crud := NuevoCRUDSimulado(t, map[string][]map[string]any{
		"GET /v1/proveedor": {{
			"Id": float64(101), "TerceroId": float64(201),
			"TipoProveedor": float64(tipoRegistroNatural), "DescripcionPortafolio": "Servicios",
		}},
		"GET /v1/tercero": {{
			"Id": float64(201), "NombreCompleto": "Ana Pérez",
			"TipoContribuyenteId": float64(2),
		}},
		"GET /v1/proveedor_natural": {{
			"Id":                 float64(101),
			"ExperienciaLaboral": float64(12),
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

	// La especialización natural se aplana en el proveedor: sus campos aparecen al nivel
	// de la raíz, sin objeto anidado bajo ProveedorNatural.
	if meses, _ := env.Data["ExperienciaLaboral"].(float64); int(meses) != 12 {
		t.Errorf("ExperienciaLaboral de la especialización natural debe aplanarse al proveedor; data=%+v", env.Data)
	}
	if anidado, presente := env.Data["ProveedorNatural"]; presente {
		t.Errorf("la especialización natural no debe anidarse bajo ProveedorNatural; valor=%+v", anidado)
	}

	if juridico, ok := env.Data["ProveedorJuridico"]; ok && juridico != nil {
		t.Errorf("la rama jurídica debe podarse para un proveedor natural; data=%+v", env.Data)
	}
}

// TODO: TDD RED - Pendiente de implementación: la raíz entrega Activo sin fields; la lista negra global solo recorta objetos anidados (1g.1).
// Los campos de auditoría administrados por el sistema no se entregan en una consulta
// de proveedor sin fields, ni en la raíz ni en las entidades anidadas. La lista negra
// se carga desde la configuración real.
func TestIntegracionConsultaSinFieldsNoEntregaAuditoria(t *testing.T) {
	crud := NuevoCRUDSimulado(t, map[string][]map[string]any{
		"GET /v1/proveedor": {{
			"Id": float64(101), "TerceroId": float64(201),
			"TipoProveedor": float64(tipoRegistroNatural), "DescripcionPortafolio": "Servicios",
			"FechaCreacion": "2026-01-01T00:00:00Z", "FechaModificacion": "2026-01-02T00:00:00Z",
			"Activo": true, "UsuarioModificacionId": float64(9),
		}},
		"GET /v1/tercero": {{
			"Id": float64(201), "NombreCompleto": "Ana Pérez",
			"TipoContribuyenteId": float64(2),
			"FechaCreacion":       "2026-01-01T00:00:00Z", "FechaModificacion": "2026-01-02T00:00:00Z",
			"Activo": true, "UsuarioModificacionId": float64(9),
		}},
		"GET /v1/proveedor_natural": {{
			"Id":            float64(101),
			"FechaCreacion": "2026-01-01T00:00:00Z", "Activo": true,
		}},
	})
	blacklist, err := config.LoadBlacklist("../config/global_blacklist.yaml")
	if err != nil {
		t.Fatalf("cargando la lista negra global real: %v", err)
	}
	router := routerProveedorConBlacklist(t, crud, blacklist)

	env := consultaProveedorCuerpo(t, router, `{"id":101}`)

	auditoria := []string{"FechaCreacion", "FechaModificacion", "Activo", "UsuarioModificacionId"}
	for _, nivel := range []struct {
		nombre string
		mapa   map[string]any
	}{
		{"proveedor", env.Data},
		{"Tercero", mapaAnidado(env.Data, "Tercero")},
		{"ProveedorNatural", mapaAnidado(env.Data, "ProveedorNatural")},
	} {
		for _, campo := range auditoria {
			if _, presente := nivel.mapa[campo]; presente {
				t.Errorf("%s no debe entregar el campo de auditoría %s sin fields; mapa=%+v", nivel.nombre, campo, nivel.mapa)
			}
		}
	}
}

// mapaAnidado devuelve el objeto anidado bajo la clave dada, o nil si no existe.
func mapaAnidado(data map[string]any, clave string) map[string]any {
	m, _ := data[clave].(map[string]any)
	return m
}

// Un lugar de ubicaciones referenciado por el núcleo se expande bajo su mapped_name con su
// identificador, su nombre y su tipo de lugar anidado (de un salto).
func TestIntegracionConsultaCiudadExpandidaConTipo(t *testing.T) {
	crud := NuevoCRUDSimulado(t, map[string][]map[string]any{
		"GET /v1/proveedor": {{
			"Id": float64(101), "TerceroId": float64(201),
			"TipoProveedor": float64(tipoRegistroNatural), "DescripcionPortafolio": "Servicios",
		}},
		"GET /v1/tercero": {{
			"Id": float64(201), "NombreCompleto": "Ana Pérez", "LugarOrigen": float64(11),
		}},
		"GET /v1/lugar": {{
			"Id": float64(11), "Nombre": "Bogotá", "TipoLugar": float64(2),
		}},
		"GET /v1/tipo_lugar": {{
			"Id": float64(2), "Nombre": "Ciudad",
		}},
	})
	router := routerProveedor(t, crud)

	env := consultaProveedorCuerpo(t, router, `{"id":101,"fields":["Tercero.LugarOrigenDetalle.Id","Tercero.LugarOrigenDetalle.Nombre","Tercero.LugarOrigenDetalle.TipoLugarDetalle.Nombre"]}`)

	tercero := mapaAnidado(env.Data, "Tercero")
	lugar := mapaAnidado(tercero, "LugarOrigenDetalle")
	if lugar == nil {
		t.Fatalf("el lugar de origen debe expandirse bajo LugarOrigenDetalle; tercero=%+v", tercero)
	}
	if id, _ := lugar["Id"].(float64); int(id) != 11 {
		t.Errorf("el lugar debe conservar su identificador 11; lugar=%+v", lugar)
	}
	if nombre, _ := lugar["Nombre"].(string); nombre != "Bogotá" {
		t.Errorf("el lugar debe entregar su nombre; lugar=%+v", lugar)
	}
	tipo := mapaAnidado(lugar, "TipoLugarDetalle")
	if tipo == nil {
		t.Fatalf("el tipo de lugar debe expandirse anidado bajo TipoLugarDetalle; lugar=%+v", lugar)
	}
	if nombre, _ := tipo["Nombre"].(string); nombre != "Ciudad" {
		t.Errorf("el tipo de lugar debe entregar su nombre; tipo=%+v", tipo)
	}
}

// La proyección aplanada de la especialización es de un solo salto: consultar el proveedor
// con fields de su especialización consulta el proveedor una sola vez (la raíz), y la
// especialización aplanada no reexpande el proveedor ni anida una referencia de regreso.
func TestIntegracionAplanadoEsDeUnSaltoSinReexpandirProveedor(t *testing.T) {
	crud := NuevoCRUDSimulado(t, map[string][]map[string]any{
		"GET /v1/proveedor": {{
			"Id": float64(101), "TerceroId": float64(201),
			"TipoProveedor": float64(tipoRegistroNatural), "DescripcionPortafolio": "Servicios",
		}},
		"GET /v1/tercero": {{
			"Id": float64(201), "NombreCompleto": "Ana Pérez",
		}},
		"GET /v1/proveedor_natural": {{
			"Id": float64(101), "ExperienciaLaboral": float64(12),
		}},
	})
	router := routerProveedor(t, crud)

	env := consultaProveedorCuerpo(t, router, `{"id":101,"fields":["ProveedorNatural.ExperienciaLaboral"]}`)

	if meses, _ := env.Data["ExperienciaLaboral"].(float64); int(meses) != 12 {
		t.Fatalf("la especialización debe aplanarse en la raíz; data=%+v", env.Data)
	}
	if _, anidado := env.Data["ProveedorNatural"]; anidado {
		t.Errorf("la especialización aplanada no debe anidarse bajo ProveedorNatural; data=%+v", env.Data)
	}
	if _, regreso := env.Data["Proveedor"]; regreso {
		t.Errorf("la especialización no debe anidar una referencia de regreso al proveedor; data=%+v", env.Data)
	}

	lecturasProveedor := 0
	for _, c := range crud.Capturas() {
		if c.Metodo == http.MethodGet && recursoDePath(c.Path) == "proveedor" {
			lecturasProveedor++
		}
	}
	if lecturasProveedor != 1 {
		t.Errorf("el proveedor debe consultarse una sola vez (la raíz); consultas=%d capturas=%+v", lecturasProveedor, crud.Capturas())
	}
}
