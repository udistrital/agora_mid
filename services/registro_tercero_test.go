package services

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/udistrital/agora_mid/clients"
	"github.com/udistrital/agora_mid/dto"
)

func TestRegistrarOReutilizar(t *testing.T) {
	tests := []struct {
		name        string
		terceroID   int
		mode        string
		wantID      int
		wantPartial bool
	}{
		{name: "crea tercero e identificación", mode: "crear", wantID: 51},
		{name: "reutiliza tercero existente", mode: "existente", wantID: 25},
		{name: "conserva ID si falla identificación", mode: "fallo", wantID: 51, wantPartial: true},
		{name: "reanuda registro parcial", mode: "reintento", terceroID: 51, wantID: 51},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")

				switch {
				case r.Method == http.MethodGet && r.URL.Path == "/v1/tipo_documento":
					fmt.Fprint(w, `[{"Id":3,"CodigoAbreviacion":"CC","Activo":true}]`)

				case r.Method == http.MethodGet && r.URL.Path == "/v1/datos_identificacion":
					if tt.mode == "existente" {
						fmt.Fprint(w, `[{"Id":10,"Numero":"123456","Activo":true,"TerceroId":{"Id":25},"TipoDocumentoId":{"CodigoAbreviacion":"CC"}}]`)
					} else {
						fmt.Fprint(w, `[]`)
					}

				case r.Method == http.MethodGet && r.URL.Path == "/v1/tipo_contribuyente":
					fmt.Fprint(w, `[{"Id":1,"CodigoAbreviacion":"P_NATURAL","Activo":true}]`)

				case r.Method == http.MethodGet && r.URL.Path == "/v1/tercero/51":
					fmt.Fprint(w, `{"Id":51}`)

				case r.Method == http.MethodPost && r.URL.Path == "/v1/tercero/":
					if tt.mode == "reintento" || tt.mode == "existente" {
						t.Errorf("no debía crear otro tercero")
					}
					fmt.Fprint(w, `{"Id":51}`)

				case r.Method == http.MethodPost && r.URL.Path == "/v1/datos_identificacion/":
					if tt.mode == "fallo" {
						w.WriteHeader(http.StatusBadRequest)
						fmt.Fprint(w, `{}`)
						return
					}
					fmt.Fprint(w, `{"Id":100}`)

				default:
					t.Errorf("petición inesperada: %s %s", r.Method, r.URL.String())
					http.NotFound(w, r)
				}
			}))
			defer server.Close()

			service := TercerosService{
				BaseURL: server.URL,
				Client:  clients.HTTPClient{Client: server.Client()},
			}

			datos := dto.DatosTercerosDTO{
				TerceroID:           tt.terceroID,
				NombreCompleto:      "Ana Pérez",
				PrimerNombre:        "Ana",
				PrimerApellido:      "Pérez",
				CodigoTipoDocumento: "CC",
				NumeroDocumento:     "123456",
			}

			id, err := service.RegistrarOReutilizar(
				context.Background(), datos, "P_NATURAL",
			)

			if id != tt.wantID {
				t.Fatalf("ID recibido %d; esperado %d; error: %v", id, tt.wantID, err)
			}

			if tt.wantPartial {
				var parcial *ErrorRegistroParcial
				if !errors.As(err, &parcial) {
					t.Fatalf("se esperaba ErrorRegistroParcial; recibido: %v", err)
				}
				if parcial.TerceroID != tt.wantID {
					t.Errorf("ID parcial %d; esperado %d", parcial.TerceroID, tt.wantID)
				}
				return
			}

			if err != nil {
				t.Fatalf("error inesperado: %v", err)
			}
		})
	}
}
