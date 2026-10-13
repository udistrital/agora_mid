package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/udistrital/agora_mid/clients"
)

func TestCrearOReutilizarProveedor(t *testing.T) {
	tests := []struct {
		name             string
		existenteInicial bool
		respuestaPerdida bool
		wantID           int
		wantPOST         int
	}{
		{
			name:             "reutiliza proveedor existente",
			existenteInicial: true,
			wantID:           70,
			wantPOST:         0,
		},
		{
			name:     "crea proveedor nuevo",
			wantID:   81,
			wantPOST: 1,
		},
		{
			name:             "recupera proveedor si se pierde respuesta POST",
			respuestaPerdida: true,
			wantID:           81,
			wantPOST:         1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			guardado := tt.existenteInicial
			posts := 0

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")

				switch {
				case r.Method == http.MethodGet && r.URL.Path == "/v1/proveedor/":
					if r.URL.Query().Get("query") != "TerceroId:25" {
						t.Errorf("filtro inesperado: %s", r.URL.Query().Get("query"))
					}

					if guardado {
						id := 81
						if tt.existenteInicial {
							id = 70
						}
						fmt.Fprintf(w,
							`{"Data":[{"Id":%d,"TerceroId":25,"TipoRegistroId":3,"Activo":true}]}`,
							id,
						)
					} else {
						fmt.Fprint(w, `{"Data":[{}]}`)
					}

				case r.Method == http.MethodPost && r.URL.Path == "/v1/proveedor/":
					posts++

					var payload struct {
						TerceroId      int  `json:"TerceroId"`
						TipoRegistroId int  `json:"TipoRegistroId"`
						Activo         bool `json:"Activo"`
					}
					if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
						t.Errorf("payload JSON inválido: %v", err)
					}
					if payload.TerceroId != 25 || payload.TipoRegistroId != 3 || !payload.Activo {
						t.Errorf("payload inesperado: %+v", payload)
					}

					guardado = true
					w.WriteHeader(http.StatusCreated)

					if tt.respuestaPerdida {
						fmt.Fprint(w, `respuesta-no-json`)
						return
					}

					fmt.Fprint(w,
						`{"Success":true,"Status":"201","Data":{"Id":81,"TerceroId":25,"TipoRegistroId":3,"Activo":true}}`,
					)

				default:
					t.Errorf("petición inesperada: %s %s", r.Method, r.URL.String())
					http.NotFound(w, r)
				}
			}))
			defer server.Close()

			service := GestionPersonasService{
				BaseURL: server.URL,
				Client:  clients.HTTPClient{Client: server.Client()},
			}

			got, err := service.CrearOReutilizarProveedor(
				context.Background(), 25, 3, "Portafolio de prueba",
			)
			if err != nil {
				t.Fatalf("error inesperado: %v", err)
			}
			if got.Id != tt.wantID {
				t.Errorf("ID recibido %d; esperado %d", got.Id, tt.wantID)
			}
			if posts != tt.wantPOST {
				t.Errorf("POST enviados %d; esperado %d", posts, tt.wantPOST)
			}
		})
	}
}
