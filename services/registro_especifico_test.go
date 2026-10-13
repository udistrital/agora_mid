package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/udistrital/agora_mid/clients"
	"github.com/udistrital/agora_mid/dto"
)

func TestCrearProveedorNaturalComparteID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/proveedor_natural/25":
			http.NotFound(w, r)

		case r.Method == http.MethodPost && r.URL.Path == "/v1/proveedor_natural/":
			var body struct {
				Id              int `json:"Id"`
				PerfilDeclarado struct {
					Id int `json:"Id"`
				} `json:"PerfilDeclarado"`
				ExperienciaLaboralMeses     int `json:"ExperienciaLaboralMeses"`
				ExperienciaProfesionalMeses int `json:"ExperienciaProfesionalMeses"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("JSON inválido: %v", err)
			}
			if body.Id != 25 {
				t.Errorf("ID enviado %d; esperado 25", body.Id)
			}
			if body.PerfilDeclarado.Id != 7 {
				t.Errorf("perfil enviado %d; esperado 7", body.PerfilDeclarado.Id)
			}
			if body.ExperienciaLaboralMeses != 12 || body.ExperienciaProfesionalMeses != 6 {
				t.Errorf("experiencia inesperada: %+v", body)
			}
			fmt.Fprint(w, `{"Data":{"Id":25,"PerfilDeclarado":{"Id":7},"ExperienciaLaboralMeses":12,"ExperienciaProfesionalMeses":6,"Activo":true}}`)

		default:
			t.Errorf("petición inesperada: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	service := GestionPersonasService{
		BaseURL: server.URL,
		Client:  clients.HTTPClient{Client: server.Client()},
	}

	got, err := service.CrearOReutilizarProveedorNatural(
		context.Background(),
		25,
		dto.ProveedorNaturalDTO{
			PerfilDeclaradoID:           7,
			ExperienciaLaboralMeses:     12,
			ExperienciaProfesionalMeses: 6,
		},
	)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if got.Id != 25 {
		t.Fatalf("ID recibido %d; esperado 25", got.Id)
	}
}

func TestCrearProveedorJuridicoComparteID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/proveedor_juridico/25":
			http.NotFound(w, r)

		case r.Method == http.MethodPost && r.URL.Path == "/v1/proveedor_juridico/":
			var body struct {
				Id                 int    `json:"Id"`
				NombreComercial    string `json:"NombreComercial"`
				MatriculaMercantil int    `json:"MatriculaMercantil"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("JSON inválido: %v", err)
			}
			if body.Id != 25 {
				t.Errorf("ID enviado %d; esperado 25", body.Id)
			}
			if body.NombreComercial != "Empresa de prueba" || body.MatriculaMercantil != 12345 {
				t.Errorf("datos jurídicos inesperados: %+v", body)
			}
			fmt.Fprint(w, `{"Data":{"Id":25,"NombreComercial":"Empresa de prueba","MatriculaMercantil":12345,"FechaConstitucion":"2020-01-01T00:00:00Z","FechaRenovacion":"2026-01-01T00:00:00Z","ReportaBeneficiosFinales":false,"CotizaBolsa":false,"RequiereRevisorFiscal":false,"Activo":true}}`)

		default:
			t.Errorf("petición inesperada: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	service := GestionPersonasService{
		BaseURL: server.URL,
		Client:  clients.HTTPClient{Client: server.Client()},
	}

	got, err := service.CrearOReutilizarProveedorJuridico(
		context.Background(),
		25,
		dto.ProveedorJuridicoDTO{
			NombreComercial:    "Empresa de prueba",
			MatriculaMercantil: 12345,
			FechaConstitucion:  "2020-01-01",
			FechaRenovacion:    "2026-01-01",
		},
	)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if got.Id != 25 {
		t.Fatalf("ID recibido %d; esperado 25", got.Id)
	}
}
