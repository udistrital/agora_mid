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

	"github.com/udistrital/agora_mid/clients"
	"github.com/udistrital/agora_mid/helper"
)

// servidorProveedorConEstado simula el CRUD gestion_personas_crud. GET
// /v1/proveedor/999 responde estadoGet y las demás peticiones responden 200.
// Cada petición se registra en capturas.
func servidorProveedorConEstado(t *testing.T, estadoGet int, capturas *[]peticionCapturada) *httptest.Server {
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
		if r.Method == http.MethodGet && r.URL.Path == "/v1/proveedor/999" {
			w.WriteHeader(estadoGet)
			_, _ = io.WriteString(w, `{"Data":null,"Status":"`+strconv.Itoa(estadoGet)+`","Success":false}`)
			return
		}
		_, _ = io.WriteString(w, `{"Data":{"Id":10}}`)
	}))
}

// servicioConCRUDSimulado arma el servicio de escritura del proveedor contra un CRUD
// simulado, con el mismo cliente y URLs que usa el servicio en producción.
func servicioConCRUDSimulado(server *httptest.Server) ProveedorWriteService {
	urls := urlsDe(server)
	client := clients.HTTPClient{Client: server.Client()}
	return ProveedorWriteService{
		Config:              cfgProveedorWritePrueba(),
		Registry:            registroWriteNucleo(client, urls),
		TipoRegistroNatural: tipoRegistroNaturalPrueba,
		Client:              client,
		URLs:                urls,
	}
}

// cuerpoProveedorConId devuelve un cuerpo mínimo de proveedor natural con el Id dado.
func cuerpoProveedorConId(id float64) map[string]any {
	return map[string]any{
		"Id":                    id,
		"TipoProveedor":         float64(tipoRegistroNaturalPrueba),
		"TerceroId":             float64(101),
		"DescripcionPortafolio": "Consultoría",
	}
}

// Un PATCH con Id inexistente no debe escribir y debe devolver ErrProveedorNoExiste.
func TestEscribirProveedorPatchIdInexistenteDevuelveNoExiste(t *testing.T) {
	var capturas []peticionCapturada
	server := servidorProveedorConEstado(t, http.StatusNotFound, &capturas)
	defer server.Close()
	svc := servicioConCRUDSimulado(server)

	_, err := svc.EscribirProveedor(context.Background(), cuerpoProveedorConId(999), helper.WriteVerbUpdate)
	if !errors.Is(err, ErrProveedorNoExiste) {
		t.Fatalf("err = %v; se esperaba ErrProveedorNoExiste", err)
	}
	for _, c := range capturas {
		if c.metodo == http.MethodPost || c.metodo == http.MethodPut || c.metodo == http.MethodDelete {
			t.Errorf("no debe escribirse si el proveedor no existe; petición=%s %s", c.metodo, c.path)
		}
	}
}

// Un PATCH con Id existente no debe confundirse con un proveedor inexistente.
func TestEscribirProveedorPatchIdExistenteNoDevuelveNoExiste(t *testing.T) {
	var capturas []peticionCapturada
	server := servidorProveedorConEstado(t, http.StatusOK, &capturas)
	defer server.Close()
	svc := servicioConCRUDSimulado(server)

	_, err := svc.EscribirProveedor(context.Background(), cuerpoProveedorConId(999), helper.WriteVerbUpdate)
	if errors.Is(err, ErrProveedorNoExiste) {
		t.Fatalf("un proveedor existente no debe responder ErrProveedorNoExiste: %v", err)
	}
}

// Un fallo de lectura distinto de 404 no debe confundirse con proveedor inexistente
// y no debe escribir.
func TestEscribirProveedorPatchErrorCRUDNoEsNoExiste(t *testing.T) {
	var capturas []peticionCapturada
	server := servidorProveedorConEstado(t, http.StatusInternalServerError, &capturas)
	defer server.Close()
	svc := servicioConCRUDSimulado(server)

	_, err := svc.EscribirProveedor(context.Background(), cuerpoProveedorConId(999), helper.WriteVerbUpdate)
	if err == nil {
		t.Fatal("se esperaba error ante un fallo de lectura del CRUD")
	}
	if errors.Is(err, ErrProveedorNoExiste) {
		t.Fatalf("un fallo de lectura no es proveedor inexistente: %v", err)
	}
	for _, c := range capturas {
		if c.metodo == http.MethodPost || c.metodo == http.MethodPut {
			t.Errorf("no debe escribirse ante un fallo de lectura; petición=%s %s", c.metodo, c.path)
		}
	}
}

// Un POST con Id conserva su comportamiento: la verificación de existencia es solo
// para PATCH, así que el proveedor se crea.
func TestEscribirProveedorPostConIdNoVerificaExistencia(t *testing.T) {
	var capturas []peticionCapturada
	server := servidorProveedorConEstado(t, http.StatusNotFound, &capturas)
	defer server.Close()
	svc := servicioConCRUDSimulado(server)

	_, err := svc.EscribirProveedor(context.Background(), cuerpoProveedorConId(999), helper.WriteVerbCreate)
	if errors.Is(err, ErrProveedorNoExiste) {
		t.Fatalf("un POST no debe responder ErrProveedorNoExiste: %v", err)
	}
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

// Un PATCH sin Id no usa la verificación por Id; su manejo corresponde a F27.
func TestEscribirProveedorPatchSinIdNoVerificaExistenciaPorId(t *testing.T) {
	var capturas []peticionCapturada
	server := servidorProveedorConEstado(t, http.StatusNotFound, &capturas)
	defer server.Close()
	svc := servicioConCRUDSimulado(server)

	cuerpo := cuerpoProveedorConId(999)
	delete(cuerpo, "Id")

	_, err := svc.EscribirProveedor(context.Background(), cuerpo, helper.WriteVerbUpdate)
	if errors.Is(err, ErrProveedorNoExiste) {
		t.Fatalf("un PATCH sin Id no debe responder ErrProveedorNoExiste: %v", err)
	}
	for _, c := range capturas {
		if c.metodo == http.MethodGet && strings.HasPrefix(c.path, "/v1/proveedor/") {
			t.Errorf("un PATCH sin Id no debe consultar un proveedor por Id; petición=%s", c.path)
		}
	}
}
