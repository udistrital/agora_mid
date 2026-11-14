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

// servidorProveedorExistenteConTercero simula el CRUD gestion_personas_crud con el
// proveedor 1 asociado al tercero 101. GET /v1/proveedor/1 devuelve ese registro. Cada
// petición se registra en capturas.
func servidorProveedorExistenteConTercero(t *testing.T, capturas *[]peticionCapturada) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var cuerpo map[string]any
		if r.Body != nil {
			if data, _ := io.ReadAll(r.Body); len(data) > 0 {
				_ = json.Unmarshal(data, &cuerpo)
			}
		}
		*capturas = append(*capturas, peticionCapturada{metodo: r.Method, path: r.URL.Path, cuerpo: cuerpo})
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && r.URL.Path == "/v1/proveedor/1" {
			_, _ = io.WriteString(w, `{"Data":{"Id":1,"TerceroId":101,"TipoProveedor":1,"Activo":true}}`)
			return
		}
		_, _ = io.WriteString(w, `{"Data":{"Id":1}}`)
	}))
}

// Un PATCH con Id existente y TerceroId distinto del registro responde 400 sin escribir.
func TestEscribirProveedorPatchTerceroDistintoEnExistenteDevuelve400(t *testing.T) {
	var capturas []peticionCapturada
	server := servidorProveedorExistenteConTercero(t, &capturas)
	defer server.Close()
	svc := servicioConCRUDSimulado(server)

	cuerpo := cuerpoProveedorConId(1)
	cuerpo["TerceroId"] = float64(999)

	_, err := svc.EscribirProveedor(context.Background(), cuerpo, helper.WriteVerbUpdate)
	var ee *helper.ErrorEscritura
	if !errors.As(err, &ee) {
		t.Fatalf("err = %v; se esperaba *helper.ErrorEscritura (400) por TerceroId distinto", err)
	}
	for _, c := range capturas {
		if c.metodo == http.MethodPost || c.metodo == http.MethodPut {
			t.Errorf("no debe escribirse con TerceroId en conflicto; petición=%s %s", c.metodo, c.path)
		}
	}
}

// Un PATCH con TerceroId igual al del registro no es conflicto.
func TestEscribirProveedorPatchTerceroIgualEnExistenteNoEsConflicto(t *testing.T) {
	var capturas []peticionCapturada
	server := servidorProveedorExistenteConTercero(t, &capturas)
	defer server.Close()
	svc := servicioConCRUDSimulado(server)

	cuerpo := cuerpoProveedorConId(1)
	cuerpo["TerceroId"] = float64(101)

	_, err := svc.EscribirProveedor(context.Background(), cuerpo, helper.WriteVerbUpdate)
	var ee *helper.ErrorEscritura
	if errors.As(err, &ee) {
		t.Fatalf("TerceroId igual al del registro no debe responder 400: %v", err)
	}
}

// Sin TerceroId en el cuerpo, o con 0 o null, no hay FK que comparar: no es conflicto.
func TestEscribirProveedorPatchTerceroAusenteOCeroEnExistenteNoEsConflicto(t *testing.T) {
	casos := map[string]func(map[string]any){
		"omitido": func(c map[string]any) { delete(c, "TerceroId") },
		"cero":    func(c map[string]any) { c["TerceroId"] = float64(0) },
		"null":    func(c map[string]any) { c["TerceroId"] = nil },
	}
	for nombre, ajustar := range casos {
		t.Run(nombre, func(t *testing.T) {
			var capturas []peticionCapturada
			server := servidorProveedorExistenteConTercero(t, &capturas)
			defer server.Close()
			svc := servicioConCRUDSimulado(server)

			cuerpo := cuerpoProveedorConId(1)
			ajustar(cuerpo)

			_, err := svc.EscribirProveedor(context.Background(), cuerpo, helper.WriteVerbUpdate)
			var ee *helper.ErrorEscritura
			if errors.As(err, &ee) {
				t.Fatalf("TerceroId %s no debe responder 400: %v", nombre, err)
			}
		})
	}
}
