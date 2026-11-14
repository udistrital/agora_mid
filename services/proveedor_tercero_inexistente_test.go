package services

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/udistrital/agora_mid/helper"
)

// servidorConTercero simula en un solo servidor el CRUD gestion_personas_crud y el de
// terceros. GET /v1/tercero/<id> responde estadoTercero; el resto de peticiones
// responden 200. Cada petición se registra en capturas.
func servidorConTercero(t *testing.T, estadoTercero int, capturas *[]peticionCapturada) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var cuerpo map[string]any
		if r.Body != nil {
			if data, _ := io.ReadAll(r.Body); len(data) > 0 {
				_ = json.Unmarshal(data, &cuerpo)
			}
		}
		*capturas = append(*capturas, peticionCapturada{
			metodo: r.Method,
			path:   r.URL.Path,
			query:  r.URL.Query().Get("query"),
			cuerpo: cuerpo,
		})
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v1/tercero/") {
			w.WriteHeader(estadoTercero)
			_, _ = io.WriteString(w, `{"Data":null,"Status":"`+strconv.Itoa(estadoTercero)+`","Success":false}`)
			return
		}
		if r.Method == http.MethodPost {
			_, _ = io.WriteString(w, `{"Data":{"Id":10}}`)
			return
		}
		_, _ = io.WriteString(w, `{"Data":[]}`)
	}))
}

// cuerpoAltaConTercero devuelve un alta de proveedor natural sin Id y con el TerceroId dado.
func cuerpoAltaConTercero(terceroID float64) map[string]any {
	cuerpo := cuerpoProveedorConId(0)
	delete(cuerpo, "Id")
	cuerpo["TerceroId"] = terceroID
	return cuerpo
}

// capturasPOSTProveedor cuenta las creaciones de proveedor registradas.
func capturasPOSTProveedor(capturas []peticionCapturada) int {
	n := 0
	for _, c := range capturas {
		if c.metodo == http.MethodPost && c.path == "/v1/proveedor/" {
			n++
		}
	}
	return n
}

// capturasGETTercero cuenta las consultas a terceros registradas.
func capturasGETTercero(capturas []peticionCapturada) int {
	n := 0
	for _, c := range capturas {
		if c.metodo == http.MethodGet && strings.HasPrefix(c.path, "/v1/tercero/") {
			n++
		}
	}
	return n
}

// Un alta con TerceroId que terceros no conoce debe responder 400 en TerceroId y no
// crear el proveedor.
func TestEscribirProveedorPostConTerceroInexistenteDevuelve400SinEscribir(t *testing.T) {
	var capturas []peticionCapturada
	server := servidorConTercero(t, http.StatusNotFound, &capturas)
	defer server.Close()
	svc := servicioConCRUDSimulado(server)

	_, err := svc.EscribirProveedor(context.Background(), cuerpoAltaConTercero(999), helper.WriteVerbCreate)

	var errEsc *helper.ErrorEscritura
	if !errors.As(err, &errEsc) {
		t.Fatalf("se esperaba ErrorEscritura (400) por tercero inexistente, se obtuvo %v", err)
	}
	if errEsc.Path != "TerceroId" {
		t.Errorf("el error debe apuntar a TerceroId, apunta a %q", errEsc.Path)
	}
	if capturasPOSTProveedor(capturas) != 0 {
		t.Errorf("no debe crear el proveedor con un tercero inexistente; capturas=%+v", capturas)
	}
}

// Un alta con un TerceroId que terceros conoce sigue su curso y crea el proveedor.
func TestEscribirProveedorPostConTerceroExistenteCreaProveedor(t *testing.T) {
	var capturas []peticionCapturada
	server := servidorConTercero(t, http.StatusOK, &capturas)
	defer server.Close()
	svc := servicioConCRUDSimulado(server)

	_, err := svc.EscribirProveedor(context.Background(), cuerpoAltaConTercero(101), helper.WriteVerbCreate)
	if err != nil {
		t.Fatalf("un tercero existente no debe impedir el alta: %v", err)
	}
	if capturasPOSTProveedor(capturas) == 0 {
		t.Errorf("el alta debe crear el proveedor; capturas=%+v", capturas)
	}
}

// Si terceros falla por otra causa (5xx), el alta no escribe y el error no es un 400
// de tercero inexistente.
func TestEscribirProveedorPostConErrorDeTercerosNoEscribeNiDevuelve400(t *testing.T) {
	var capturas []peticionCapturada
	server := servidorConTercero(t, http.StatusInternalServerError, &capturas)
	defer server.Close()
	svc := servicioConCRUDSimulado(server)

	_, err := svc.EscribirProveedor(context.Background(), cuerpoAltaConTercero(101), helper.WriteVerbCreate)
	if err == nil {
		t.Fatal("un fallo de terceros debe devolver error")
	}
	var errEsc *helper.ErrorEscritura
	if errors.As(err, &errEsc) {
		t.Errorf("un fallo de terceros no es un 400 de validación: %v", err)
	}
	if capturasPOSTProveedor(capturas) != 0 {
		t.Errorf("un fallo de terceros no debe escribir; capturas=%+v", capturas)
	}
}

// Sin TerceroId, o con 0, el alta no consulta terceros: el dato es opcional.
func TestEscribirProveedorPostSinTerceroIdONuloNoConsultaTerceros(t *testing.T) {
	casos := map[string]func() map[string]any{
		"ausente": func() map[string]any {
			c := cuerpoAltaConTercero(0)
			delete(c, "TerceroId")
			return c
		},
		"cero": func() map[string]any { return cuerpoAltaConTercero(0) },
	}
	for nombre, cuerpo := range casos {
		t.Run(nombre, func(t *testing.T) {
			var capturas []peticionCapturada
			server := servidorConTercero(t, http.StatusNotFound, &capturas)
			defer server.Close()
			svc := servicioConCRUDSimulado(server)

			_, _ = svc.EscribirProveedor(context.Background(), cuerpo(), helper.WriteVerbCreate)
			if n := capturasGETTercero(capturas); n != 0 {
				t.Errorf("sin TerceroId válido no debe consultar terceros; consultas=%d capturas=%+v", n, capturas)
			}
		})
	}
}

// Un alta que trae el objeto Tercero crea o actualiza el tercero en el mismo flujo, así
// que su TerceroId no se consulta en terceros antes de escribir.
func TestEscribirProveedorPostConObjetoTerceroNoConsultaTerceroIdEnTerceros(t *testing.T) {
	var capturas []peticionCapturada
	server := servidorConTercero(t, http.StatusNotFound, &capturas)
	defer server.Close()
	svc := servicioConCRUDSimulado(server)

	cuerpo := cuerpoAltaConTercero(999)
	cuerpo["Tercero"] = map[string]any{"NombreCompleto": "Ana"}

	_, err := svc.EscribirProveedor(context.Background(), cuerpo, helper.WriteVerbCreate)
	var errEsc *helper.ErrorEscritura
	if errors.As(err, &errEsc) && errEsc.Path == "TerceroId" {
		t.Fatalf("con objeto Tercero no debe verificarse TerceroId en terceros: %v", err)
	}
	for _, c := range capturas {
		if c.metodo == http.MethodGet && c.path == "/v1/tercero/999" {
			t.Errorf("con objeto Tercero no debe consultarse /v1/tercero/999; capturas=%+v", capturas)
		}
	}
}
