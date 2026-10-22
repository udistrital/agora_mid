package controllers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/udistrital/agora_mid/helper"
	"github.com/udistrital/agora_mid/responses"
	"github.com/udistrital/agora_mid/services"
)

type proveedorReaderStub struct {
	resultado helper.ExpandResult
	err       error
	id        int
	fields    []string
}

func (s *proveedorReaderStub) LeerProveedor(
	_ context.Context,
	id int,
	fields []string,
) (helper.ExpandResult, error) {
	s.id = id
	s.fields = fields
	return s.resultado, s.err
}

func TestProveedorHandlerLeer(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		resultado  helper.ExpandResult
		err        error
		wantStatus int
		wantBody   string
	}{
		{name: "id obligatorio", query: "", wantStatus: 400, wantBody: `"Status":"400"`},
		{name: "id no numérico", query: "?id=abc", wantStatus: 400, wantBody: `"Status":"400"`},
		{name: "id no positivo", query: "?id=0", wantStatus: 400, wantBody: `"Status":"400"`},
		{name: "fields vacío", query: "?id=12&fields=", wantStatus: 400, wantBody: `"Status":"400"`},
		{name: "proveedor inexistente", query: "?id=12", err: services.ErrProveedorNoEncontrado, wantStatus: 404, wantBody: `"Status":"404"`},
		{name: "error dependencia", query: "?id=12", err: errors.New("error interno"), wantStatus: 502, wantBody: `"Status":"502"`},
		{
			name:  "consulta correcta",
			query: "?id=12&fields=Id,Tercero.Id",
			resultado: helper.ExpandResult{
				Data: map[string]any{"Id": 12, "TerceroId": 30},
			},
			wantStatus: 200,
			wantBody:   `"Status":"200"`,
		},
		{
			name:  "expansión parcial",
			query: "?id=12",
			resultado: helper.ExpandResult{
				Data: map[string]any{"Id": 12},
				Warnings: []responses.RelationError{
					{Path: "Tercero", Service: "tercero", Message: "No se pudo resolver la relación", Kind: "relation"},
				},
			},
			wantStatus: 206,
			wantBody:   `"Status":"206"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := &proveedorReaderStub{resultado: tt.resultado, err: tt.err}
			handler := NewProveedorHandler(stub)
			req := httptest.NewRequest(http.MethodGet, "/v1/proveedor"+tt.query, nil)
			rec := httptest.NewRecorder()

			handler.Leer(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d; esperado %d. Body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
				t.Errorf("Content-Type inesperado: %q", got)
			}
			if !strings.Contains(rec.Body.String(), tt.wantBody) {
				t.Errorf("la respuesta no contiene %q: %s", tt.wantBody, rec.Body.String())
			}
			if tt.wantStatus == 200 {
				if stub.id != 12 {
					t.Errorf("ID enviado = %d; esperado 12", stub.id)
				}
				if len(stub.fields) != 2 || stub.fields[0] != "Id" || stub.fields[1] != "Tercero.Id" {
					t.Errorf("fields enviados: %#v", stub.fields)
				}
			}
			if tt.wantStatus == 206 && !strings.Contains(rec.Body.String(), `"Errors"`) {
				t.Errorf("la respuesta parcial no contiene MetaData.Errors: %s", rec.Body.String())
			}
		})
	}
}

func TestProveedorHandlerMetodoNoPermitido(t *testing.T) {
	handler := NewProveedorHandler(&proveedorReaderStub{})
	req := httptest.NewRequest(http.MethodPost, "/v1/proveedor?id=12", nil)
	rec := httptest.NewRecorder()

	handler.Leer(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d; esperado %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if got := rec.Header().Get("Allow"); got != http.MethodGet {
		t.Errorf("Allow = %q; esperado GET", got)
	}
}
