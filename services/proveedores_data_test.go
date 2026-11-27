package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestGetAllProveedoresMapasAceptaEnvelopeYListaDirecta(t *testing.T) {
	casos := []struct {
		nombre string
		cuerpo string
		status int
	}{
		{
			nombre: "envelope",
			cuerpo: `{"Success":true,"Status":"200","Message":"ok","Data":[{"Id":1}],"Total":1}`,
			status: http.StatusOK,
		},
		{
			nombre: "lista directa",
			cuerpo: `[{"Id":1}]`,
			status: http.StatusOK,
		},
		{
			nombre: "404 vacío",
			cuerpo: `{"Success":false,"Status":"404","Message":"no record exists"}`,
			status: http.StatusNotFound,
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(caso.status)
				_, _ = w.Write([]byte(caso.cuerpo))
			}))
			defer servidor.Close()
			fijarHostProveedores(t, servidor.URL)

			var registros []map[string]any
			ok, status, err := GetAllProveedoresMapas(
				context.Background(), "proveedor", url.Values{}, &registros,
			)
			if err != nil || !ok {
				t.Fatalf("ok=%v status=%d err=%v", ok, status, err)
			}
			if caso.status == http.StatusNotFound {
				if len(registros) != 0 {
					t.Fatalf("se esperaba lista vacía: %+v", registros)
				}
				return
			}
			if len(registros) != 1 || idDeMapa(registros[0]) != 1 {
				t.Fatalf("registros inesperados: %+v", registros)
			}
		})
	}
}
