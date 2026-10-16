package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/udistrital/agora_mid/clients"
	"github.com/udistrital/agora_mid/dto"
)

func TestActualizarPersonaRechazaIDProveedorInconsistente(t *testing.T) {
	llamadas := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		llamadas++

		if r.Method != http.MethodGet || r.URL.Path != "/v1/proveedor/" {
			t.Errorf("petición inesperada: %s %s", r.Method, r.URL.String())
			http.Error(w, "petición inesperada", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w,
			`{"Success":true,"Data":[{"Id":81,"TerceroId":25}]}`,
		)
	}))
	defer server.Close()

	service := RegistroPersonaService{
		GestionPersonas: GestionPersonasService{
			BaseURL: server.URL,
			Client:  clients.HTTPClient{Client: server.Client()},
		},
	}

	req := dto.ActualizarPersonaRequest{
		TerceroID:   25,
		ProveedorID: 99,
		Proveedor: map[string]json.RawMessage{
			"descripcion_portafolio": json.RawMessage(`"Nueva descripción"`),
		},
	}

	_, err := service.ActualizarPersona(context.Background(), req, "natural")
	if !errors.Is(err, ErrActualizacionInvalida) {
		t.Fatalf("se esperaba ErrActualizacionInvalida; se obtuvo: %v", err)
	}

	if llamadas != 1 {
		t.Fatalf("se esperaba una llamada al CRUD; se hicieron %d", llamadas)
	}
}

func TestActualizarPersonaRechazaProveedorDeOtroTercero(t *testing.T) {
	llamadas := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		llamadas++

		if r.Method != http.MethodGet || r.URL.Path != "/v1/proveedor/" {
			t.Errorf("petición inesperada: %s %s", r.Method, r.URL.String())
			http.Error(w, "petición inesperada", http.StatusNotFound)
			return
		}

		if got := r.URL.Query().Get("query"); got != "TerceroId:25" {
			t.Errorf("filtro de tercero inesperado: %s", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w,
			`{"Success":true,"Data":[{"Id":81,"TerceroId":77}]}`,
		)
	}))
	defer server.Close()

	service := RegistroPersonaService{
		GestionPersonas: GestionPersonasService{
			BaseURL: server.URL,
			Client:  clients.HTTPClient{Client: server.Client()},
		},
	}

	req := dto.ActualizarPersonaRequest{
		TerceroID: 25,
		Proveedor: map[string]json.RawMessage{
			"descripcion_portafolio": json.RawMessage(`"Nueva descripción"`),
		},
	}

	_, err := service.ActualizarPersona(context.Background(), req, "natural")
	if !errors.Is(err, ErrPersonaNoEncontrada) {
		t.Fatalf("se esperaba ErrPersonaNoEncontrada al recibir un proveedor de otro tercero; se obtuvo: %v", err)
	}

	if llamadas != 1 {
		t.Fatalf("se esperaba una llamada al CRUD; se hicieron %d", llamadas)
	}
}
