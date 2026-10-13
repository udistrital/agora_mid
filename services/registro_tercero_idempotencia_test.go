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

func TestReintentoNoDuplicaIdentificacionSiLaRespuestaFalla(t *testing.T) {
	identificacionGuardada := false
	terceroPosts := 0
	identificacionPosts := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/tipo_documento":
			fmt.Fprint(w, `[{"Id":3,"CodigoAbreviacion":"CC","Activo":true}]`)

		case r.Method == http.MethodGet && r.URL.Path == "/v1/datos_identificacion":
			if identificacionGuardada {
				fmt.Fprint(w, `[{
					"Id":100,
					"Numero":"123456",
					"Activo":true,
					"TerceroId":{"Id":51},
					"TipoDocumentoId":{"CodigoAbreviacion":"CC"}
				}]`)
			} else {
				fmt.Fprint(w, `[]`)
			}

		case r.Method == http.MethodGet && r.URL.Path == "/v1/tipo_contribuyente":
			fmt.Fprint(w, `[{"Id":1,"CodigoAbreviacion":"P_NATURAL","Activo":true}]`)

		case r.Method == http.MethodPost && r.URL.Path == "/v1/tercero/":
			terceroPosts++
			fmt.Fprint(w, `{"Id":51}`)

		case r.Method == http.MethodPost && r.URL.Path == "/v1/datos_identificacion/":
			identificacionPosts++
			identificacionGuardada = true

			// Simula que el servidor guardó el registro, pero la respuesta
			// no puede ser interpretada por el cliente.
			fmt.Fprint(w, `respuesta-no-json`)

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
		NombreCompleto:      "Ana Pérez",
		PrimerNombre:        "Ana",
		PrimerApellido:      "Pérez",
		CodigoTipoDocumento: "CC",
		NumeroDocumento:     "123456",
	}

	id, err := service.RegistrarOReutilizar(context.Background(), datos, "P_NATURAL")
	var parcial *ErrorRegistroParcial
	if !errors.As(err, &parcial) {
		t.Fatalf("se esperaba ErrorRegistroParcial; ID=%d, error=%v", id, err)
	}
	if id != 51 || parcial.TerceroID != 51 {
		t.Fatalf("se debía conservar tercero_id 51; ID=%d, parcial=%+v", id, parcial)
	}

	// El segundo intento reutiliza el tercero y encuentra la identificación
	// que ya fue guardada en el primer intento.
	datos.TerceroID = id
	reintentoID, err := service.RegistrarOReutilizar(
		context.Background(), datos, "P_NATURAL",
	)
	if err != nil {
		t.Fatalf("falló el reintento: %v", err)
	}
	if reintentoID != 51 {
		t.Fatalf("ID del reintento=%d; se esperaba 51", reintentoID)
	}
	if terceroPosts != 1 {
		t.Errorf("se crearon %d terceros; se esperaba 1", terceroPosts)
	}
	if identificacionPosts != 1 {
		t.Errorf("se enviaron %d POST de identificación; se esperaba 1", identificacionPosts)
	}
}
