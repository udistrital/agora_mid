package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/udistrital/agora_mid/controllers"
)

// La ruta /v1/proveedor/consulta se sirve con el handler Consultar. Se verifica
// que POST llega a Consultar (valida el cuerpo antes de tocar el servicio) y que
// GET/PUT/DELETE responden 405 con Allow: POST, igual que en el arranque donde
// beego.Any enlaza http.HandlerFunc(proveedorHandler.Consultar).
func TestEnrutarConsultaProveedorDespachaPorMetodo(t *testing.T) {
	h := controllers.NewProveedorHandler(nil)
	router := http.HandlerFunc(h.Consultar)

	casos := []struct {
		nombre     string
		metodo     string
		cuerpo     string
		wantStatus int
		wantAllow  string
	}{
		{
			// POST con id ausente -> Consultar responde 400 antes del servicio.
			nombre: "POST va a Consultar", metodo: http.MethodPost,
			cuerpo: `{}`, wantStatus: http.StatusBadRequest,
		},
		{
			nombre: "GET responde 405", metodo: http.MethodGet,
			wantStatus: http.StatusMethodNotAllowed, wantAllow: http.MethodPost,
		},
		{
			nombre: "PUT responde 405", metodo: http.MethodPut,
			wantStatus: http.StatusMethodNotAllowed, wantAllow: http.MethodPost,
		},
		{
			nombre: "DELETE responde 405", metodo: http.MethodDelete,
			wantStatus: http.StatusMethodNotAllowed, wantAllow: http.MethodPost,
		},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			req := httptest.NewRequest(c.metodo, "/v1/proveedor/consulta", strings.NewReader(c.cuerpo))
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != c.wantStatus {
				t.Fatalf("status = %d; esperado %d. Body: %s", rec.Code, c.wantStatus, rec.Body.String())
			}
			if c.wantAllow != "" && rec.Header().Get("Allow") != c.wantAllow {
				t.Errorf("Allow = %q; esperado %q", rec.Header().Get("Allow"), c.wantAllow)
			}
		})
	}
}

// enrutarProveedor debe delegar por método: GET a la lectura, POST/PATCH a la
// escritura. Se verifica sin red: cada rama falla temprano con 400 por entrada
// inválida antes de tocar un servicio, lo que evidencia a qué handler llegó.
func TestEnrutarProveedorDespachaPorMetodo(t *testing.T) {
	// Handler sin servicios: las rutas de validación temprana no los invocan.
	h := controllers.NewProveedorHandlerCompleto(nil, nil)
	router := enrutarProveedor(h)

	casos := []struct {
		nombre     string
		metodo     string
		url        string
		cuerpo     string
		wantStatus int
		wantAllow  string
	}{
		{
			// GET sin id -> Leer responde 400 (id obligatorio).
			nombre: "GET va a Leer", metodo: http.MethodGet, url: "/v1/proveedor",
			wantStatus: http.StatusBadRequest,
		},
		{
			// POST con JSON inválido -> Escribir responde 400 antes del servicio.
			nombre: "POST va a Escribir", metodo: http.MethodPost, url: "/v1/proveedor",
			cuerpo: "no-json", wantStatus: http.StatusBadRequest,
		},
		{
			// PATCH con JSON inválido -> Escribir responde 400 antes del servicio.
			nombre: "PATCH va a Escribir", metodo: http.MethodPatch, url: "/v1/proveedor",
			cuerpo: "no-json", wantStatus: http.StatusBadRequest,
		},
		{
			// DELETE cae en la lectura (default) y obtiene 405 con Allow: GET.
			nombre: "DELETE va a Leer (405)", metodo: http.MethodDelete, url: "/v1/proveedor",
			wantStatus: http.StatusMethodNotAllowed, wantAllow: http.MethodGet,
		},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			var cuerpo *strings.Reader
			if c.cuerpo != "" {
				cuerpo = strings.NewReader(c.cuerpo)
			} else {
				cuerpo = strings.NewReader("")
			}
			req := httptest.NewRequest(c.metodo, c.url, cuerpo)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != c.wantStatus {
				t.Fatalf("status = %d; esperado %d. Body: %s", rec.Code, c.wantStatus, rec.Body.String())
			}
			if c.wantAllow != "" && rec.Header().Get("Allow") != c.wantAllow {
				t.Errorf("Allow = %q; esperado %q", rec.Header().Get("Allow"), c.wantAllow)
			}
		})
	}
}
