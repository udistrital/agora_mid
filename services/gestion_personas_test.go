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

func TestBuscarProveedorPorTercero(t *testing.T) {
	tests := []struct {
		name      string
		response  string
		wantID    int
		wantFound bool
		wantError error
	}{
		{
			name:     "proveedor inexistente",
			response: `{"Data":[{}]}`,
		},
		{
			name:     "proveedor asociado a otro tercero",
			response: `{"Data":[{"Id":81,"TerceroId":77}]}`,
		},
		{
			name:      "proveedor encontrado",
			response:  `{"Data":[{"Id":8,"TerceroId":25,"TipoRegistroId":1,"Activo":true}]}`,
			wantID:    8,
			wantFound: true,
		},
		{
			name:      "varios proveedores",
			response:  `{"Data":[{"Id":8,"TerceroId":25},{"Id":9,"TerceroId":25}]}`,
			wantError: ErrProveedorAmbiguo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != "/v1/proveedor/" {
						t.Errorf("ruta inesperada: %s", r.URL.Path)
					}
					if got := r.URL.Query().Get("query"); got != "TerceroId:25" {
						t.Errorf("filtro inesperado: %s", got)
					}
					fmt.Fprint(w, tt.response)
				},
			))
			defer server.Close()

			service := GestionPersonasService{
				BaseURL: server.URL,
				Client:  clients.HTTPClient{Client: server.Client()},
			}

			proveedor, err := service.BuscarProveedorPorTercero(
				context.Background(), 25,
			)

			if tt.wantError != nil {
				if !errors.Is(err, tt.wantError) {
					t.Fatalf("error recibido %v; se esperaba %v", err, tt.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("error inesperado: %v", err)
			}
			if tt.wantFound {
				if proveedor == nil || proveedor.Id != tt.wantID {
					t.Fatalf("proveedor recibido %v; se esperaba ID %d", proveedor, tt.wantID)
				}
			} else if proveedor != nil {
				t.Fatalf("se esperaba ningún proveedor, se recibió %+v", proveedor)
			}
		})
	}
}
