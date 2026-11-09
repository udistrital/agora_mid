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

type proveedorCollectionStub struct {
	resultado helper.CollectionResult
	err       error
	query     []string
	fields    []string
	limit     int
	offset    int
}

func (s *proveedorCollectionStub) ListarProveedores(
	_ context.Context,
	query, fields []string,
	limit, offset int,
) (helper.CollectionResult, error) {
	s.query, s.fields, s.limit, s.offset = query, fields, limit, offset
	return s.resultado, s.err
}

func TestProveedorHandlerConsultarLista(t *testing.T) {
	stub := &proveedorCollectionStub{
		resultado: helper.CollectionResult{
			Data:  []any{map[string]any{"Id": 5}},
			Total: 9,
		},
	}
	handler := NewProveedorHandlerLista(stub)
	req := httptest.NewRequest(http.MethodPost, "/v1/proveedor/consulta/lista",
		strings.NewReader(`{"query":["Id:5"],"fields":["Tercero.Id"],"limit":2,"offset":1}`))
	rec := httptest.NewRecorder()

	handler.ConsultarLista(rec, req)

	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"Total":9`) {
		t.Fatalf("respuesta inesperada: status=%d body=%s", rec.Code, rec.Body.String())
	}
	if len(stub.query) != 1 || stub.query[0] != "Id:5" ||
		len(stub.fields) != 1 || stub.fields[0] != "Tercero.Id" ||
		stub.limit != 2 || stub.offset != 1 {
		t.Fatalf("parámetros no trasladados: %+v", stub)
	}

	for _, metodo := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
		req = httptest.NewRequest(metodo, "/v1/proveedor/consulta/lista", nil)
		rec = httptest.NewRecorder()
		handler.ConsultarLista(rec, req)
		if rec.Code != http.StatusMethodNotAllowed || rec.Header().Get("Allow") != http.MethodPost {
			t.Fatalf("método %s: status=%d allow=%q", metodo, rec.Code, rec.Header().Get("Allow"))
		}
	}
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

func TestProveedorHandlerConsultar(t *testing.T) {
	tests := []struct {
		name       string
		cuerpo     string
		resultado  helper.ExpandResult
		err        error
		wantStatus int
		wantBody   string
		wantID     int
		wantFields []string
	}{
		{name: "cuerpo malformado", cuerpo: `no-es-json`, wantStatus: 400, wantBody: `"Status":"400"`},
		{name: "cuerpo vacío", cuerpo: ``, wantStatus: 400, wantBody: `"Status":"400"`},
		{name: "id ausente", cuerpo: `{"fields":["Id"]}`, wantStatus: 400, wantBody: `"Status":"400"`},
		{name: "id no positivo", cuerpo: `{"id":0}`, wantStatus: 400, wantBody: `"Status":"400"`},
		{name: "id negativo", cuerpo: `{"id":-5}`, wantStatus: 400, wantBody: `"Status":"400"`},
		{name: "fields con ruta vacía", cuerpo: `{"id":12,"fields":["Id",""]}`, wantStatus: 400, wantBody: `"Status":"400"`},
		{
			name:       "fields no permitido (servicio)",
			cuerpo:     `{"id":12,"fields":["Tercero.NoExiste"]}`,
			err:        &helper.ErrorFields{Path: "Tercero.NoExiste", Motivo: "el campo \"NoExiste\" no está permitido"},
			wantStatus: 400,
			wantBody:   `"Status":"400"`,
		},
		{
			name:       "fields no canónico (servicio)",
			cuerpo:     `{"id":12,"fields":["tercero.id"]}`,
			err:        &helper.ErrorFields{Path: "tercero.id", Motivo: "el campo \"tercero\" no está permitido"},
			wantStatus: 400,
			wantBody:   `"Status":"400"`,
		},
		{
			name:       "fields duplicado (servicio)",
			cuerpo:     `{"id":12,"fields":["Id","Id"]}`,
			err:        &helper.ErrorFields{Path: "Id", Motivo: "ruta duplicada"},
			wantStatus: 400,
			wantBody:   `"Status":"400"`,
		},
		{name: "proveedor inexistente", cuerpo: `{"id":12}`, err: services.ErrProveedorNoEncontrado, wantStatus: 404, wantBody: `"Status":"404"`},
		{name: "error dependencia", cuerpo: `{"id":12}`, err: errors.New("error interno"), wantStatus: 502, wantBody: `"Status":"502"`},
		{
			name:   "consulta correcta con campo anidado",
			cuerpo: `{"id":12,"fields":["Id","Tercero.Id"]}`,
			resultado: helper.ExpandResult{
				Data: map[string]any{"Id": 12, "TerceroId": 30},
			},
			wantStatus: 200,
			wantBody:   `"Status":"200"`,
			wantID:     12,
			wantFields: []string{"Id", "Tercero.Id"},
		},
		{
			name:   "consulta sin fields",
			cuerpo: `{"id":7}`,
			resultado: helper.ExpandResult{
				Data: map[string]any{"Id": 7},
			},
			wantStatus: 200,
			wantBody:   `"Status":"200"`,
			wantID:     7,
			wantFields: nil,
		},
		{
			name:   "expansión parcial con advertencias",
			cuerpo: `{"id":12}`,
			resultado: helper.ExpandResult{
				Data: map[string]any{"Id": 12},
				Warnings: []responses.RelationError{
					{Path: "Tercero", Service: "tercero", Message: "No se pudo resolver la relación", Kind: "relation"},
				},
			},
			wantStatus: 206,
			wantBody:   `"Status":"206"`,
			wantID:     12,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := &proveedorReaderStub{resultado: tt.resultado, err: tt.err}
			handler := NewProveedorHandler(stub)
			req := httptest.NewRequest(http.MethodPost, "/v1/proveedor/consulta", strings.NewReader(tt.cuerpo))
			rec := httptest.NewRecorder()

			handler.Consultar(rec, req)

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
				if stub.id != tt.wantID {
					t.Errorf("ID enviado = %d; esperado %d", stub.id, tt.wantID)
				}
				if len(stub.fields) != len(tt.wantFields) {
					t.Errorf("fields enviados: %#v; esperado %#v", stub.fields, tt.wantFields)
				} else {
					for i := range tt.wantFields {
						if stub.fields[i] != tt.wantFields[i] {
							t.Errorf("fields[%d] = %q; esperado %q", i, stub.fields[i], tt.wantFields[i])
						}
					}
				}
			}
			if tt.wantStatus == 206 && !strings.Contains(rec.Body.String(), `"Errors"`) {
				t.Errorf("la respuesta parcial no contiene MetaData.Errors: %s", rec.Body.String())
			}
			// Un fields inválido no debe producir ninguna llamada al CRUD.
			if tt.wantStatus == 400 && strings.Contains(tt.name, "fields con ruta vacía") && stub.id != 0 {
				t.Errorf("no debió invocarse el servicio con fields inválido; id=%d", stub.id)
			}
		})
	}
}

// TestProveedorHandlerConsultarMetodoNoPermitido verifica que métodos distintos
// de POST sobre /v1/proveedor/consulta respondan 405 anunciando POST.
func TestProveedorHandlerConsultarMetodoNoPermitido(t *testing.T) {
	for _, metodo := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
		t.Run(metodo, func(t *testing.T) {
			handler := NewProveedorHandler(&proveedorReaderStub{})
			req := httptest.NewRequest(metodo, "/v1/proveedor/consulta", nil)
			rec := httptest.NewRecorder()

			handler.Consultar(rec, req)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Fatalf("status = %d; esperado %d", rec.Code, http.StatusMethodNotAllowed)
			}
			if got := rec.Header().Get("Allow"); got != http.MethodPost {
				t.Errorf("Allow = %q; esperado POST", got)
			}
		})
	}
}

// proveedorWriterStub simula el servicio de escritura y captura el verbo, el
// cuerpo y el autor recibidos.
type proveedorWriterStub struct {
	resultado helper.WriteResult
	err       error
	verb      string
	body      map[string]any
	autor     int
	autorOK   bool
}

func (s *proveedorWriterStub) EscribirProveedor(
	ctx context.Context,
	body map[string]any,
	verb string,
) (helper.WriteResult, error) {
	s.verb = verb
	s.body = body
	s.autor, s.autorOK = services.AutorDeContexto(ctx)
	return s.resultado, s.err
}

func TestProveedorHandlerEscribir(t *testing.T) {
	timeoutErr := helper.NewErrorFatalEscritura(context.DeadlineExceeded)
	fatalErr := helper.NewErrorFatalEscritura(errors.New("crud caído"))

	tests := []struct {
		name       string
		metodo     string
		cuerpo     string
		resultado  helper.WriteResult
		err        error
		wantStatus int
		wantBody   string
	}{
		{
			name:   "creación completa 200",
			metodo: http.MethodPost,
			cuerpo: `{"TerceroId":25,"TipoRegistroId":1}`,
			resultado: helper.WriteResult{
				Data: map[string]any{"Id": 10, "TerceroId": 25},
			},
			wantStatus: 200,
			wantBody:   `"Status":"200"`,
		},
		{
			name:   "materialización parcial 206",
			metodo: http.MethodPatch,
			cuerpo: `{"TerceroId":25,"TipoRegistroId":1}`,
			resultado: helper.WriteResult{
				Data: map[string]any{"Id": 10},
				Warnings: []responses.RelationError{
					{Path: "Contacto", Service: "contacto", Message: "no se pudo materializar", Kind: "relation"},
				},
			},
			wantStatus: 206,
			wantBody:   `"Errors"`,
		},
		{
			name:   "canonización de campo derivado 206 field_normalization",
			metodo: http.MethodPost,
			cuerpo: `{"TerceroId":25,"TipoRegistroId":1}`,
			resultado: helper.WriteResult{
				Data: map[string]any{"Id": 10, "TerceroId": 201},
				Warnings: []responses.RelationError{
					{Path: "Tercero", Service: "tercero", Message: "se normalizó un campo relacional derivado con la llave autoritativa", Kind: helper.KindFieldNormalization},
				},
			},
			wantStatus: 206,
			wantBody:   `"field_normalization"`,
		},
		{
			name:   "campo derivado coincidente 200 sin aviso",
			metodo: http.MethodPost,
			cuerpo: `{"TerceroId":201,"TipoRegistroId":1}`,
			resultado: helper.WriteResult{
				Data: map[string]any{"Id": 10, "TerceroId": 201},
			},
			wantStatus: 200,
			wantBody:   `"Status":"200"`,
		},
		{
			name:       "cuerpo JSON inválido 400",
			metodo:     http.MethodPost,
			cuerpo:     `no-es-json`,
			wantStatus: 400,
			wantBody:   `"Status":"400"`,
		},
		{
			name:       "campo inválido 400",
			metodo:     http.MethodPost,
			cuerpo:     `{"TerceroId":25,"TipoRegistroId":1}`,
			err:        &helper.ErrorEscritura{Path: "Foo", Motivo: "campo no escribible"},
			wantStatus: 400,
			wantBody:   `"Status":"400"`,
		},
		{
			name:       "rama incoherente 400",
			metodo:     http.MethodPost,
			cuerpo:     `{"TerceroId":25,"TipoRegistroId":1}`,
			err:        services.ErrRamaDetalleIncoherente,
			wantStatus: 400,
			wantBody:   `"Status":"400"`,
		},
		{
			name:       "fallo bloqueante 502",
			metodo:     http.MethodPost,
			cuerpo:     `{"TerceroId":25,"TipoRegistroId":1}`,
			err:        fatalErr,
			wantStatus: 502,
			wantBody:   `"Status":"502"`,
		},
		{
			name:       "fallo bloqueante por timeout 504",
			metodo:     http.MethodPost,
			cuerpo:     `{"TerceroId":25,"TipoRegistroId":1}`,
			err:        timeoutErr,
			wantStatus: 504,
			wantBody:   `"Status":"504"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := &proveedorWriterStub{resultado: tt.resultado, err: tt.err}
			handler := NewProveedorHandlerCompleto(&proveedorReaderStub{}, stub)
			req := httptest.NewRequest(tt.metodo, "/v1/proveedor", strings.NewReader(tt.cuerpo))
			req.Header.Set("X-Autor-Id", "9")
			rec := httptest.NewRecorder()

			handler.Escribir(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d; esperado %d. Body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), tt.wantBody) {
				t.Errorf("la respuesta no contiene %q: %s", tt.wantBody, rec.Body.String())
			}
			// Una respuesta 200 no debe incluir avisos en MetaData.Errors.
			if tt.wantStatus == 200 && strings.Contains(rec.Body.String(), `"Errors"`) {
				t.Errorf("una respuesta 200 no debe incluir MetaData.Errors: %s", rec.Body.String())
			}
			// En los casos que llegan al servicio, el autor viaja por el contexto.
			if tt.err == nil && tt.wantStatus < 400 {
				if !stub.autorOK || stub.autor != 9 {
					t.Errorf("el autor 9 debe viajar por el contexto; autor=%d ok=%v", stub.autor, stub.autorOK)
				}
				wantVerb := helper.WriteVerbCreate
				if tt.metodo == http.MethodPatch {
					wantVerb = helper.WriteVerbUpdate
				}
				if stub.verb != wantVerb {
					t.Errorf("verbo = %q; esperado %q", stub.verb, wantVerb)
				}
			}
		})
	}
}

func TestProveedorHandlerEscribirMetodoNoPermitido(t *testing.T) {
	handler := NewProveedorHandlerCompleto(&proveedorReaderStub{}, &proveedorWriterStub{})
	req := httptest.NewRequest(http.MethodDelete, "/v1/proveedor", nil)
	rec := httptest.NewRecorder()

	handler.Escribir(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d; esperado %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if got := rec.Header().Get("Allow"); !strings.Contains(got, http.MethodPost) || !strings.Contains(got, http.MethodPatch) {
		t.Errorf("Allow = %q; esperado POST y PATCH", got)
	}
}
