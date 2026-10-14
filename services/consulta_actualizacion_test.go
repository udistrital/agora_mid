package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/udistrital/agora_mid/clients"
	"github.com/udistrital/agora_mid/dto"
)

func TestConsultarPersonaNaturalIntegraTercerosYAgora(t *testing.T) {
	terceros := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/tercero/25":
			fmt.Fprint(w, `{"Success":true,"Data":{"Id":25,"NombreCompleto":"Ana Pérez","Activo":true}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/v1/datos_identificacion":
			if !strings.Contains(r.URL.Query().Get("query"), "TerceroId.Id:25") {
				t.Errorf("filtro de identificaciones incorrecto: %s", r.URL.Query().Get("query"))
			}
			fmt.Fprint(w, `{"Success":true,"Data":[{"Id":99,"Numero":"123456","TerceroId":{"Id":25},"Activo":true}]}`)
		default:
			t.Errorf("petición inesperada a Terceros: %s %s", r.Method, r.URL.String())
			http.NotFound(w, r)
		}
	}))
	defer terceros.Close()

	crud := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/proveedor/":
			if r.URL.Query().Get("query") != "TerceroId:25" {
				t.Errorf("filtro de proveedor incorrecto: %s", r.URL.Query().Get("query"))
			}
			fmt.Fprint(w, `{"Success":true,"Data":[{"Id":81,"TerceroId":25,"TipoRegistroId":1,"Activo":true}]}`)
		case r.Method == http.MethodGet && r.URL.Path == "/v1/proveedor/81":
			fmt.Fprint(w, `{"Success":true,"Data":{"Id":81,"TerceroId":25,"TipoRegistroId":1,"DescripcionPortafolio":"Servicios TI","Activo":true}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/v1/proveedor_natural/81":
			fmt.Fprint(w, `{"Success":true,"Data":{"Id":81,"PerfilDeclarado":{"Id":2,"Nombre":"Persona natural"},"ExperienciaLaboralMeses":12,"ExperienciaProfesionalMeses":6,"Activo":true}}`)
		default:
			t.Errorf("petición inesperada al CRUD Agora: %s %s", r.Method, r.URL.String())
			http.NotFound(w, r)
		}
	}))
	defer crud.Close()

	service := RegistroPersonaService{
		Terceros:        TercerosService{BaseURL: terceros.URL, Client: clients.HTTPClient{Client: terceros.Client()}},
		GestionPersonas: GestionPersonasService{BaseURL: crud.URL, Client: clients.HTTPClient{Client: crud.Client()}},
	}
	result, err := service.ConsultarPersona(context.Background(), 25, "natural")
	if err != nil {
		t.Fatalf("ConsultarPersona devolvió error: %v", err)
	}
	if result.TerceroID != 25 || string(result.ProveedorNatural) == "" || string(result.Proveedor) == "" {
		t.Fatalf("respuesta integrada incompleta: %+v", result)
	}
	var tercero map[string]any
	if err := json.Unmarshal(result.Terceros, &tercero); err != nil || tercero["NombreCompleto"] != "Ana Pérez" {
		t.Fatalf("tercero no incluido en la respuesta: %s, error=%v", result.Terceros, err)
	}
	var ids []map[string]any
	if err := json.Unmarshal(result.DatosIdentificacion, &ids); err != nil || len(ids) != 1 || ids[0]["Numero"] != "123456" {
		t.Fatalf("identificación no incluida: %s, error=%v", result.DatosIdentificacion, err)
	}
}

func TestActualizarIdentificacionPreservaCamposNoEnviados(t *testing.T) {
	putRealizado := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			if r.URL.Path != "/v1/datos_identificacion/99" {
				t.Errorf("ruta GET inesperada: %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			fmt.Fprint(w, `{"Success":true,"Data":{"Id":99,"Numero":"123456","TipoDocumentoId":{"Id":3},"TerceroId":{"Id":25},"DigitoVerificacion":0,"CiudadExpedicion":1,"Activo":true}}`)
		case http.MethodPut:
			if r.URL.Path != "/v1/datos_identificacion/99" {
				t.Errorf("ruta PUT inesperada: %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			var body map[string]json.RawMessage
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("PUT con JSON inválido: %v", err)
			}
			var numero string
			_ = json.Unmarshal(body["Numero"], &numero)
			if numero != "123456" {
				t.Errorf("se sobrescribió el número de documento: %q", numero)
			}
			var dv float64
			_ = json.Unmarshal(body["DigitoVerificacion"], &dv)
			if dv != 9 {
				t.Errorf("DV actualizado incorrectamente: %v", dv)
			}
			putRealizado = true
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{}`)
		default:
			t.Errorf("método inesperado: %s", r.Method)
			http.Error(w, "método no permitido", http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	service := TercerosService{BaseURL: server.URL, Client: clients.HTTPClient{Client: server.Client()}}
	input := map[string]json.RawMessage{"id": json.RawMessage("99"), "digito_verificacion": json.RawMessage("9")}
	if err := service.actualizarIdentificacion(context.Background(), 25, input); err != nil {
		t.Fatalf("actualizarIdentificacion devolvió error: %v", err)
	}
	if !putRealizado {
		t.Fatal("no se ejecutó el PUT")
	}
}

func TestActualizarPersonaRechazaCambiarNumeroDocumentoAntesDeLlamarServicios(t *testing.T) {
	service := RegistroPersonaService{}
	_, err := service.ActualizarPersona(context.Background(), dto.ActualizarPersonaRequest{
		TerceroID: 25,
		Terceros:  map[string]json.RawMessage{"numero_documento": json.RawMessage(`"999"`)},
	}, "natural")
	if err == nil || !strings.Contains(err.Error(), "no actualizable") {
		t.Fatalf("se esperaba rechazar cambio del documento antes de consultar servicios; error: %v", err)
	}
}

func TestActualizarProveedorNaturalPreservaCamposOmitidos(t *testing.T) {
	putRealizado := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			if r.URL.Path != "/v1/proveedor_natural/81" {
				t.Errorf("ruta GET inesperada: %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			fmt.Fprint(w, `{"Success":true,"Data":{"Id":81,"PerfilDeclarado":{"Id":4},"ExperienciaLaboralMeses":12,"ExperienciaProfesionalMeses":6,"Activo":true,"FechaCreacion":"2026-01-01T00:00:00Z"}}`)
		case http.MethodPut:
			var body map[string]json.RawMessage
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("PUT con JSON inválido: %v", err)
			}
			var laboral, profesional int
			_ = json.Unmarshal(body["ExperienciaLaboralMeses"], &laboral)
			_ = json.Unmarshal(body["ExperienciaProfesionalMeses"], &profesional)
			if laboral != 24 || profesional != 6 {
				t.Errorf("meses inesperados tras combinar actualización parcial: laboral=%d profesional=%d", laboral, profesional)
			}
			if string(body["Activo"]) != "true" || len(body["PerfilDeclarado"]) == 0 {
				t.Errorf("se perdieron campos existentes: %s", body)
			}
			putRealizado = true
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{}`)
		default:
			t.Errorf("método inesperado: %s", r.Method)
			http.Error(w, "método no permitido", http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	service := GestionPersonasService{BaseURL: server.URL, Client: clients.HTTPClient{Client: server.Client()}}
	input := map[string]json.RawMessage{"experiencia_laboral_meses": json.RawMessage("24")}
	if err := service.actualizarProveedorNatural(context.Background(), 81, input); err != nil {
		t.Fatalf("actualizarProveedorNatural devolvió error: %v", err)
	}
	if !putRealizado {
		t.Fatal("no se ejecutó el PUT")
	}
}

func TestValidarDocumentoDisponibleRechazaOtraIdentificacionActiva(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/datos_identificacion" {
			t.Errorf("petición inesperada: %s %s", r.Method, r.URL.String())
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"Success":true,"Data":[{"Id":77,"Numero":"999999","Activo":true,"TipoDocumentoId":{"CodigoAbreviacion":"CC"},"TerceroId":{"Id":900}}]}`)
	}))
	defer server.Close()
	service := TercerosService{BaseURL: server.URL, Client: clients.HTTPClient{Client: server.Client()}}
	err := service.validarDocumentoDisponible(context.Background(), "999999", "CC", 66)
	if !errors.Is(err, ErrDocumentoDuplicado) {
		t.Fatalf("se esperaba ErrDocumentoDuplicado, recibido: %v", err)
	}
}
