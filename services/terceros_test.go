package services

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/udistrital/agora_mid/clients"
)

func TestBuscarPorDocumento(t *testing.T) {
	tests := []struct {
		name          string
		response      string
		wantID        int
		wantFound     bool
		wantAmbiguous bool
	}{
		{
			name:      "sin coincidencias",
			response:  `[]`,
			wantFound: false,
		},
		{
			name: "un tercero",
			response: `[{
				"Id": 10,
				"Numero": "123456",
				"Activo": true,
				"TerceroId": {"Id": 25},
				"TipoDocumentoId": {"CodigoAbreviacion": "CC"}
			}]`,
			wantID:    25,
			wantFound: true,
		},
		{
			name: "varios terceros",
			response: `[
				{
					"Id": 10,
					"Numero": "123456",
					"Activo": true,
					"TerceroId": {"Id": 25},
					"TipoDocumentoId": {"CodigoAbreviacion": "CC"}
				},
				{
					"Id": 11,
					"Numero": "123456",
					"Activo": true,
					"TerceroId": {"Id": 26},
					"TipoDocumentoId": {"CodigoAbreviacion": "CC"}
				}
			]`,
			wantAmbiguous: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != "/v1/datos_identificacion" {
						t.Errorf("ruta inesperada: %s", r.URL.Path)
					}

					expectedQuery := "Numero:123456,TipoDocumentoId.CodigoAbreviacion:CC,Activo:true"
					if got := r.URL.Query().Get("query"); got != expectedQuery {
						t.Errorf("filtro inesperado: %q", got)
					}

					w.Header().Set("Content-Type", "application/json")
					fmt.Fprint(w, tt.response)
				},
			))
			defer server.Close()

			service := TercerosService{
				BaseURL: server.URL,
				Client: clients.HTTPClient{
					Client: server.Client(),
				},
			}

			id, found, err := service.BuscarPorDocumento(
				context.Background(),
				"123456",
				"CC",
			)

			if tt.wantAmbiguous {
				if !errors.Is(err, ErrTerceroAmbiguo) {
					t.Fatalf("se esperaba ErrTerceroAmbiguo; error recibido: %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("error inesperado: %v", err)
			}
			if id != tt.wantID {
				t.Errorf("ID recibido: %d; esperado: %d", id, tt.wantID)
			}
			if found != tt.wantFound {
				t.Errorf("found recibido: %t; esperado: %t", found, tt.wantFound)
			}
		})
	}
}
