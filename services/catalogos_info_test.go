package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/udistrital/agora_mid/clients"
)

func servidorCatalogosInfo(t *testing.T) *httptest.Server {
	t.Helper()
	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/v1/info_complementaria":
			query := r.URL.Query().Get("query")
			switch {
			case strings.Contains(query, "MIGRANTE"):
				_ = json.NewEncoder(w).Encode([]map[string]any{
					{"Id": 444, "CodigoAbreviacion": "MIGRANTE", "Activo": true},
					{"Id": 452, "CodigoAbreviacion": "MIGRANTE", "Activo": true},
				})
			case strings.Contains(query, "HIJOS"):
				_ = json.NewEncoder(w).Encode([]map[string]any{
					{"Id": 21, "CodigoAbreviacion": "HIJOS", "Activo": true},
				})
			case strings.Contains(query, "INACTIVO"):
				_ = json.NewEncoder(w).Encode([]map[string]any{
					{"Id": 22, "CodigoAbreviacion": "INACTIVO", "Activo": false},
				})
			default:
				_ = json.NewEncoder(w).Encode([]map[string]any{})
			}
		case r.URL.Path == "/v1/info_complementaria/21":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"Id": 21, "Activo": true,
				"GrupoInfoComplementariaId": map[string]any{"Id": 9},
			})
		case r.URL.Path == "/v1/info_complementaria/22":
			_ = json.NewEncoder(w).Encode(map[string]any{"Id": 22, "Activo": false})
		case r.URL.Path == "/v1/grupo_info_complementaria/9":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"Id": 9, "CodigoAbreviacion": "Grupo_9", "Activo": true,
			})
		case r.URL.Path == "/v1/parametro/10":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"Id": 10, "Activo": true,
				"TipoParametroId": map[string]any{
					"Id": 195, "Activo": false, "CodigoAbreviacion": "",
				},
			})
		case r.URL.Path == "/v1/tipo_parametro/195":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"Id": 195, "CodigoAbreviacion": "CIIU", "Activo": true,
			})
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"Message":"no encontrado"}`))
		}
	}))
	t.Cleanup(servidor.Close)
	return servidor
}

func TestCatalogosInfoComplementariaPorCodigo(t *testing.T) {
	servidor := servidorCatalogosInfo(t)
	catalogos := NewCatalogosService(
		clients.HTTPClient{Client: servidor.Client()},
		servidor.URL,
		servidor.URL,
	)

	id, existe, ambiguo, err := catalogos.InfoComplementariaPorCodigo(context.Background(), "HIJOS")
	if err != nil || !existe || ambiguo || id != 21 {
		t.Fatalf("HIJOS: id=%d existe=%v ambiguo=%v err=%v", id, existe, ambiguo, err)
	}
	id, existe, ambiguo, err = catalogos.InfoComplementariaPorCodigo(context.Background(), "MIGRANTE")
	if err != nil || !existe || !ambiguo || id != 444 {
		t.Fatalf("MIGRANTE debía resolver al primero y ser ambiguo: id=%d existe=%v ambiguo=%v err=%v", id, existe, ambiguo, err)
	}
	id, existe, ambiguo, err = catalogos.InfoComplementariaPorCodigo(context.Background(), "NO_EXISTE")
	if err != nil || existe || ambiguo || id != 0 {
		t.Fatalf("NO_EXISTE debía faltar: id=%d existe=%v ambiguo=%v err=%v", id, existe, ambiguo, err)
	}
	id, existe, ambiguo, err = catalogos.InfoComplementariaPorCodigo(context.Background(), "INACTIVO")
	if err != nil || !existe || ambiguo || id != 22 {
		t.Fatalf("INACTIVO debía resolver a 22: id=%d existe=%v ambiguo=%v err=%v", id, existe, ambiguo, err)
	}
	activo, err := catalogos.InfoComplementariaExiste(context.Background(), 22)
	if err != nil || activo {
		t.Fatalf("INACTIVO no debía estar activo: activo=%v err=%v", activo, err)
	}
}

func TestCatalogosInfoComplementariaDeGrupo(t *testing.T) {
	servidor := servidorCatalogosInfo(t)
	catalogos := NewCatalogosService(
		clients.HTTPClient{Client: servidor.Client()},
		servidor.URL,
		servidor.URL,
	)

	pertenece, err := catalogos.InfoComplementariaDeGrupo(context.Background(), 21, "Grupo_9")
	if err != nil || !pertenece {
		t.Fatalf("debía pertenecer a Grupo_9: %v %v", pertenece, err)
	}
	pertenece, err = catalogos.InfoComplementariaDeGrupo(context.Background(), 21, "Grupo_5")
	if err != nil || pertenece {
		t.Fatalf("no debía pertenecer a Grupo_5: %v %v", pertenece, err)
	}
}

func TestParametrosServiceParametroDeTipo(t *testing.T) {
	servidor := servidorCatalogosInfo(t)
	parametros := NewParametrosService(
		clients.HTTPClient{Client: servidor.Client()},
		servidor.URL,
	)

	existe, err := parametros.ParametroDeTipo(context.Background(), 10, "CIIU")
	if err != nil || !existe {
		t.Fatalf("el parámetro 10 debía ser CIIU: %v %v", existe, err)
	}
	existe, err = parametros.ParametroDeTipo(context.Background(), 10, "TCC")
	if err != nil || existe {
		t.Fatalf("el parámetro 10 no debía ser TCC: %v %v", existe, err)
	}
}

func TestCatalogosInfoComplementariaOverridePorEntorno(t *testing.T) {
	servidor := servidorCatalogosInfo(t)
	catalogos := NewCatalogosService(
		clients.HTTPClient{Client: servidor.Client()},
		servidor.URL,
		servidor.URL,
	)
	catalogos.InfoComplementariaOverrides = map[string]int64{"MIGRANTE": 452}

	id, existe, ambiguo, err := catalogos.InfoComplementariaPorCodigo(context.Background(), "MIGRANTE")
	if err != nil || !existe || ambiguo || id != 452 {
		t.Fatalf("override: id=%d existe=%v ambiguo=%v err=%v", id, existe, ambiguo, err)
	}
}

func TestPrecargarInfoComplementariaEvitaConsultasPorCampo(t *testing.T) {
	var llamadas atomic.Int32
	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		llamadas.Add(1)
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/v1/info_complementaria" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"Message":"no encontrado"}`))
			return
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"Id": 21, "CodigoAbreviacion": "HIJOS", "Activo": true},
			{"Id": 22, "CodigoAbreviacion": "INACTIVO", "Activo": false},
		})
	}))
	t.Cleanup(servidor.Close)

	catalogos := NewCatalogosService(
		clients.HTTPClient{Client: servidor.Client()},
		servidor.URL,
		servidor.URL,
	)
	if err := catalogos.PrecargarInfoComplementaria(context.Background()); err != nil {
		t.Fatalf("PrecargarInfoComplementaria no debe fallar: %v", err)
	}
	antes := llamadas.Load()

	id, existe, ambiguo, err := catalogos.InfoComplementariaPorCodigo(context.Background(), "HIJOS")
	if err != nil || !existe || ambiguo || id != 21 {
		t.Fatalf("HIJOS precargado: id=%d existe=%v ambiguo=%v err=%v", id, existe, ambiguo, err)
	}
	activo, err := catalogos.InfoComplementariaExiste(context.Background(), 21)
	if err != nil || !activo {
		t.Fatalf("HIJOS debía estar activo: activo=%v err=%v", activo, err)
	}
	if despues := llamadas.Load(); despues != antes {
		t.Fatalf("no debía consultar el CRUD después de precargar: antes=%d después=%d", antes, despues)
	}
}
