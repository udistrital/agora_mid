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

func TestResolverTipoContribuyente(t *testing.T) {
	tests := []struct {
		name     string
		response string
		wantID   int
		wantErr  error
	}{
		{"codigo unico", `[{"Id":1,"CodigoAbreviacion":"P_NATURAL","Activo":true}]`, 1, nil},
		{"sin coincidencias", `[{}]`, 0, ErrCatalogoNoEncontrado},
		{"codigo duplicado", `[{"Id":1,"CodigoAbreviacion":"P_NATURAL","Activo":true},{"Id":2,"CodigoAbreviacion":"P_NATURAL","Activo":true}]`, 0, ErrCatalogoAmbiguo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/tipo_contribuyente" {
					t.Errorf("ruta inesperada: %s", r.URL.Path)
				}
				fmt.Fprint(w, tt.response)
			}))
			defer server.Close()

			svc := TercerosService{
				BaseURL: server.URL,
				Client:  clients.HTTPClient{Client: server.Client()},
			}
			id, err := svc.ResolverTipoContribuyente(context.Background(), "P_NATURAL")

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("error recibido %v; se esperaba %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("error inesperado: %v", err)
			}
			if id != tt.wantID {
				t.Errorf("ID recibido %d; se esperaba %d", id, tt.wantID)
			}
		})
	}
}
