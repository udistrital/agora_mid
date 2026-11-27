package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/udistrital/agora_mid/clients"
)

func servidorCatalogosJuridica(t *testing.T) *httptest.Server {
	t.Helper()
	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/v1/tipo_documento":
			query := r.URL.Query().Get("query")
			switch {
			case strings.Contains(query, "CodigoAbreviacion:CC"):
				_ = json.NewEncoder(w).Encode([]map[string]any{
					{"Id": 30, "CodigoAbreviacion": "CC", "Activo": false},
					{"Id": 3, "CodigoAbreviacion": "CC", "Activo": true},
				})
			case strings.Contains(query, "CodigoAbreviacion:PAS"):
				_ = json.NewEncoder(w).Encode([]map[string]any{
					{"Id": 9, "CodigoAbreviacion": "PAS", "Activo": true},
				})
			case strings.Contains(query, "CodigoAbreviacion:XX"):
				_ = json.NewEncoder(w).Encode([]map[string]any{
					{"Id": 40, "CodigoAbreviacion": "XX", "Activo": true},
					{"Id": 41, "CodigoAbreviacion": "XX", "Activo": true},
				})
			default:
				_ = json.NewEncoder(w).Encode([]map[string]any{})
			}
		case r.URL.Path == "/v1/tipo_documento/3":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"Id": 3, "CodigoAbreviacion": "CC", "Activo": true,
			})
		case r.URL.Path == "/v1/tipo_documento/30":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"Id": 30, "CodigoAbreviacion": "CC", "Activo": false,
			})
		case r.URL.Path == "/v1/datos_identificacion":
			query := r.URL.Query().Get("query")
			switch {
			case strings.Contains(query, "Numero:123"):
				_ = json.NewEncoder(w).Encode([]map[string]any{
					{"Id": 900, "TerceroId": map[string]any{"Id": 55},
						"TipoDocumentoId": map[string]any{"Id": 3},
						"Numero":          "123", "Activo": true},
				})
			case strings.Contains(query, "Numero:AMB"):
				_ = json.NewEncoder(w).Encode([]map[string]any{
					{"Id": 901, "TerceroId": map[string]any{"Id": 55},
						"TipoDocumentoId": map[string]any{"Id": 3},
						"Numero":          "AMB", "Activo": true},
					{"Id": 902, "TerceroId": map[string]any{"Id": 66},
						"TipoDocumentoId": map[string]any{"Id": 3},
						"Numero":          "AMB", "Activo": true},
				})
			default:
				_ = json.NewEncoder(w).Encode([]map[string]any{})
			}
		case r.URL.Path == "/v1/tercero/55":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"Id": 55, "Activo": true,
				"TipoContribuyenteId": map[string]any{"Id": 1},
				"NombreCompleto":      "Carlos Rodriguez",
			})
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"Message":"no encontrado"}`))
		}
	}))
	t.Cleanup(servidor.Close)
	return servidor
}

func TestCatalogosJuridicaTipoDocumentoPorCodigo(t *testing.T) {
	servidor := servidorCatalogosJuridica(t)
	catalogos := NewCatalogosService(
		clients.HTTPClient{Client: servidor.Client()},
		servidor.URL,
		servidor.URL,
	)
	id, existe, ambiguo, err := catalogos.TipoDocumentoPorCodigo(context.Background(), "CC")
	if err != nil || !existe || ambiguo || id != 3 {
		t.Fatalf("CC: id=%d existe=%v ambiguo=%v err=%v", id, existe, ambiguo, err)
	}
	id, existe, ambiguo, err = catalogos.TipoDocumentoPorCodigo(context.Background(), "XX")
	if err != nil || !existe || !ambiguo || id != 40 {
		t.Fatalf("XX debía resolver al menor y ser ambiguo: id=%d existe=%v ambiguo=%v err=%v", id, existe, ambiguo, err)
	}
	_, existe, _, err = catalogos.TipoDocumentoPorCodigo(context.Background(), "NO_EXISTE")
	if err != nil || existe {
		t.Fatalf("NO_EXISTE debía faltar: existe=%v err=%v", existe, err)
	}
}

func TestCatalogosJuridicaTipoDocumentoPorID(t *testing.T) {
	servidor := servidorCatalogosJuridica(t)
	catalogos := NewCatalogosService(
		clients.HTTPClient{Client: servidor.Client()},
		servidor.URL,
		servidor.URL,
	)
	codigo, existe, err := catalogos.TipoDocumentoPorID(context.Background(), 3)
	if err != nil || !existe || codigo != "CC" {
		t.Fatalf("id 3: codigo=%q existe=%v err=%v", codigo, existe, err)
	}
	_, existe, err = catalogos.TipoDocumentoPorID(context.Background(), 30)
	if err != nil || existe {
		t.Fatalf("id 30 inactivo no debía existir: existe=%v err=%v", existe, err)
	}
}

func TestParametrosResolverPorCodigo(t *testing.T) {
	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/v1/parametro" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		query := r.URL.Query().Get("query")
		switch {
		case strings.Contains(query, "COMUN_ORDINARIO"):
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"Id": 8392, "CodigoAbreviacion": "COMUN_ORDINARIO", "Activo": true,
					"TipoParametroId": map[string]any{"Id": 1, "CodigoAbreviacion": "REG_CONT"}},
			})
		case strings.Contains(query, "DUP"):
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"Id": 900, "CodigoAbreviacion": "DUP", "Activo": true,
					"TipoParametroId": map[string]any{"Id": 1, "CodigoAbreviacion": "REG_CONT"}},
				{"Id": 800, "CodigoAbreviacion": "DUP", "Activo": true,
					"TipoParametroId": map[string]any{"Id": 1, "CodigoAbreviacion": "REG_CONT"}},
			})
		default:
			_ = json.NewEncoder(w).Encode([]map[string]any{})
		}
	}))
	t.Cleanup(servidor.Close)
	parametros := NewParametrosService(
		clients.HTTPClient{Client: servidor.Client()}, servidor.URL,
	)

	id, existe, ambiguo, err := parametros.ResolverParametroPorCodigo(
		context.Background(), "COMUN_ORDINARIO", "REG_CONT",
	)
	if err != nil || !existe || ambiguo || id != 8392 {
		t.Fatalf("COMUN_ORDINARIO: id=%d existe=%v ambiguo=%v err=%v", id, existe, ambiguo, err)
	}
	id, existe, ambiguo, err = parametros.ResolverParametroPorCodigo(
		context.Background(), "DUP", "REG_CONT",
	)
	if err != nil || !existe || !ambiguo || id != 800 {
		t.Fatalf("DUP debía resolver al menor y ser ambiguo: id=%d existe=%v ambiguo=%v err=%v", id, existe, ambiguo, err)
	}
	_, existe, _, err = parametros.ResolverParametroPorCodigo(
		context.Background(), "NO_EXISTE", "REG_CONT",
	)
	if err != nil || existe {
		t.Fatalf("NO_EXISTE debía faltar: existe=%v err=%v", existe, err)
	}
}

func TestCatalogosJuridicaTerceroPorDocumento(t *testing.T) {
	servidor := servidorCatalogosJuridica(t)
	catalogos := NewCatalogosService(
		clients.HTTPClient{Client: servidor.Client()},
		servidor.URL,
		servidor.URL,
	)
	tercero, ok, err := catalogos.TerceroPorDocumento(context.Background(), 3, "123")
	if err != nil || !ok || idDeMapa(tercero) != 55 {
		t.Fatalf("123: ok=%v tercero=%+v err=%v", ok, tercero, err)
	}
	_, ok, err = catalogos.TerceroPorDocumento(context.Background(), 3, "NO_EXISTE")
	if err != nil || ok {
		t.Fatalf("NO_EXISTE debía faltar: ok=%v err=%v", ok, err)
	}
	_, ok, err = catalogos.TerceroPorDocumento(context.Background(), 3, "AMB")
	if err == nil || ok {
		t.Fatalf("AMB debía ser ambiguo: ok=%v err=%v", ok, err)
	}
}
