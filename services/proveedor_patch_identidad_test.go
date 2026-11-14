package services

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/udistrital/agora_mid/helper"
)

// servidorProveedorPorTercero simula el CRUD gestion_personas_crud para la
// identificación de un PATCH sin Id. GET /v1/proveedor/999 responde 404. GET
// /v1/proveedor con query TerceroId:101 devuelve el proveedor 1; cualquier otra
// búsqueda por TerceroId devuelve una lista vacía. Cada petición se registra.
func servidorProveedorPorTercero(t *testing.T, capturas *[]peticionCapturada) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var cuerpo map[string]any
		if r.Body != nil {
			if data, _ := io.ReadAll(r.Body); len(data) > 0 {
				_ = json.Unmarshal(data, &cuerpo)
			}
		}
		consulta := r.URL.Query().Get("query")
		*capturas = append(*capturas, peticionCapturada{
			metodo: r.Method, path: r.URL.Path, query: consulta, cuerpo: cuerpo,
		})
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/proveedor/999":
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"Data":null,"Status":"404","Success":false}`)
		case r.Method == http.MethodGet && r.URL.Path == "/v1/proveedor" && consulta == "TerceroId:101,Activo:true":
			_, _ = io.WriteString(w, `{"Data":[{"Id":1,"TerceroId":101,"Activo":true}],"Total":1}`)
		case r.Method == http.MethodGet && r.URL.Path == "/v1/proveedor":
			_, _ = io.WriteString(w, `{"Data":[],"Total":0}`)
		default:
			_, _ = io.WriteString(w, `{"Data":{"Id":10}}`)
		}
	}))
}

// Un PATCH sin Id ni TerceroId no identifica el proveedor y responde 400, sin escribir.
func TestEscribirProveedorPatchSinIdNiClaveNaturalDevuelve400(t *testing.T) {
	var capturas []peticionCapturada
	server := servidorProveedorPorTercero(t, &capturas)
	defer server.Close()
	svc := servicioConCRUDSimulado(server)

	cuerpo := cuerpoProveedorConId(0)
	delete(cuerpo, "Id")
	delete(cuerpo, "TerceroId")

	_, err := svc.EscribirProveedor(context.Background(), cuerpo, helper.WriteVerbUpdate)
	var ee *helper.ErrorEscritura
	if !errors.As(err, &ee) {
		t.Fatalf("err = %v; se esperaba *helper.ErrorEscritura (400)", err)
	}
	for _, c := range capturas {
		if c.metodo == http.MethodPost || c.metodo == http.MethodPut {
			t.Errorf("no debe escribirse; petición=%s %s", c.metodo, c.path)
		}
	}
}

// TerceroId 0 o null cuenta como ausente: sin Id ni clave natural válida, 400.
func TestEscribirProveedorPatchTerceroIdCeroDevuelve400(t *testing.T) {
	var capturas []peticionCapturada
	server := servidorProveedorPorTercero(t, &capturas)
	defer server.Close()
	svc := servicioConCRUDSimulado(server)

	cuerpo := cuerpoProveedorConId(0)
	delete(cuerpo, "Id")
	cuerpo["TerceroId"] = float64(0)

	_, err := svc.EscribirProveedor(context.Background(), cuerpo, helper.WriteVerbUpdate)
	var ee *helper.ErrorEscritura
	if !errors.As(err, &ee) {
		t.Fatalf("err = %v; se esperaba *helper.ErrorEscritura (400) con TerceroId 0", err)
	}
}

// Un PATCH sin Id cuya clave natural TerceroId no coincide con ningún proveedor
// responde 404 sin escribir.
func TestEscribirProveedorPatchTerceroIdInexistenteDevuelveNoExiste(t *testing.T) {
	var capturas []peticionCapturada
	server := servidorProveedorPorTercero(t, &capturas)
	defer server.Close()
	svc := servicioConCRUDSimulado(server)

	cuerpo := cuerpoProveedorConId(0)
	delete(cuerpo, "Id")
	cuerpo["TerceroId"] = float64(999)

	_, err := svc.EscribirProveedor(context.Background(), cuerpo, helper.WriteVerbUpdate)
	if !errors.Is(err, ErrProveedorNoExiste) {
		t.Fatalf("err = %v; se esperaba ErrProveedorNoExiste", err)
	}
	for _, c := range capturas {
		if c.metodo == http.MethodPost || c.metodo == http.MethodPut {
			t.Errorf("no debe escribirse; petición=%s %s", c.metodo, c.path)
		}
	}
}

// Un PATCH sin Id cuya clave natural sí coincide no se rechaza por existencia.
func TestEscribirProveedorPatchTerceroIdExistenteNoDevuelveNoExiste(t *testing.T) {
	var capturas []peticionCapturada
	server := servidorProveedorPorTercero(t, &capturas)
	defer server.Close()
	svc := servicioConCRUDSimulado(server)

	cuerpo := cuerpoProveedorConId(0)
	delete(cuerpo, "Id")

	_, err := svc.EscribirProveedor(context.Background(), cuerpo, helper.WriteVerbUpdate)
	if errors.Is(err, ErrProveedorNoExiste) {
		t.Fatalf("un proveedor encontrado por TerceroId no debe responder ErrProveedorNoExiste: %v", err)
	}
	var ee *helper.ErrorEscritura
	if errors.As(err, &ee) {
		t.Fatalf("un PATCH con TerceroId válido no debe responder 400: %v", err)
	}
}

// El POST sin Id ni TerceroId sigue creando: la regla de 400 aplica solo a PATCH.
func TestEscribirProveedorPostSinIdNiClaveNaturalCrea(t *testing.T) {
	var capturas []peticionCapturada
	server := servidorProveedorPorTercero(t, &capturas)
	defer server.Close()
	svc := servicioConCRUDSimulado(server)

	cuerpo := cuerpoProveedorConId(0)
	delete(cuerpo, "Id")
	delete(cuerpo, "TerceroId")

	_, _ = svc.EscribirProveedor(context.Background(), cuerpo, helper.WriteVerbCreate)
	creado := false
	for _, c := range capturas {
		if c.metodo == http.MethodPost && c.path == "/v1/proveedor/" {
			creado = true
		}
	}
	if !creado {
		t.Errorf("el POST debe seguir creando el proveedor; capturas=%+v", capturas)
	}
}

// Un Id 0 cuenta como ausente: sin TerceroId válido, el PATCH responde 400 sin escribir.
func TestEscribirProveedorPatchIdCeroSeTrataComoAusente(t *testing.T) {
	var capturas []peticionCapturada
	server := servidorProveedorPorTercero(t, &capturas)
	defer server.Close()
	svc := servicioConCRUDSimulado(server)

	cuerpo := cuerpoProveedorConId(0)
	delete(cuerpo, "TerceroId")

	_, err := svc.EscribirProveedor(context.Background(), cuerpo, helper.WriteVerbUpdate)
	var ee *helper.ErrorEscritura
	if !errors.As(err, &ee) {
		t.Fatalf("err = %v; se esperaba *helper.ErrorEscritura (400) con Id 0 sin TerceroId", err)
	}
	for _, c := range capturas {
		if c.metodo == http.MethodPost || c.metodo == http.MethodPut {
			t.Errorf("no debe escribirse; petición=%s %s", c.metodo, c.path)
		}
	}
}

// Un Id 0 con TerceroId se identifica por la clave natural, no por Id.
func TestEscribirProveedorPatchIdCeroUsaClaveNatural(t *testing.T) {
	var capturas []peticionCapturada
	server := servidorProveedorPorTercero(t, &capturas)
	defer server.Close()
	svc := servicioConCRUDSimulado(server)

	cuerpo := cuerpoProveedorConId(0)
	cuerpo["TerceroId"] = float64(999)

	_, err := svc.EscribirProveedor(context.Background(), cuerpo, helper.WriteVerbUpdate)
	if !errors.Is(err, ErrProveedorNoExiste) {
		t.Fatalf("err = %v; con Id 0 y TerceroId sin coincidencia se esperaba ErrProveedorNoExiste", err)
	}
	for _, c := range capturas {
		if c.metodo == http.MethodGet && c.path == "/v1/proveedor/0" {
			t.Errorf("Id 0 no debe consultarse por Id; petición=%s", c.path)
		}
	}
}
