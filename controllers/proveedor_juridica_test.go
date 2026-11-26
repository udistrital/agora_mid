package controllers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	beego "github.com/beego/beego/v2/server/web"

	"github.com/udistrital/agora_mid/controllers"
	"github.com/udistrital/agora_mid/helper"
	"github.com/udistrital/agora_mid/models"
	"github.com/udistrital/agora_mid/services"
)

type escritorJuridicoFake struct {
	resultado services.TerceroWriteResult
	err       error
	llamado   bool
	terceroID int64
	verbo     string
}

func (f *escritorJuridicoFake) EscribirTerceroJuridico(
	_ context.Context,
	_ models.RegistroTerceroJuridico,
	terceroID int64,
	verbo string,
) (services.TerceroWriteResult, error) {
	f.llamado = true
	f.terceroID = terceroID
	f.verbo = verbo
	return f.resultado, f.err
}

const cuerpoJuridico = `{
  "DatosBasicos": {
    "RazonSocial": "ACME S.A.S.",
    "LugarOrigen": 149,
    "TipoContribuyenteId": 2
  },
  "DatosIdentificacion": {
    "Numero": "900123456",
    "TipoDocumentoId": 7,
    "CiudadExpedicion": 149,
    "FechaExpedicion": "2015-01-01T00:00:00Z",
    "DigitoVerificacion": 7
  }
}`

func resultadoJuridicoFake() services.TerceroWriteResult {
	return services.TerceroWriteResult{
		Data: helper.WriteResult{Data: map[string]any{
			"Id":             101,
			"NombreCompleto": "ACME S.A.S.",
			"PrimerNombre":   "",
			"LugarOrigen":    149,
			"Activo":         true,
			"TipoContribuyenteId": map[string]any{
				"Id": 2,
			},
			"DatosIdentificacion": []any{
				map[string]any{"Id": 140, "Numero": "900123456"},
			},
		}},
		TerceroId:             101,
		DatosIdentificacionId: 140,
		Creados: []models.EntidadId{
			{Entidad: "Tercero", Id: 101},
			{Entidad: "DatosIdentificacion", Id: 140},
		},
	}
}

func TestPostJuridicaCreaYProyectaRazonSocial(t *testing.T) {
	fake := &escritorJuridicoFake{resultado: resultadoJuridicoFake()}
	controllers.ConfigurarTerceroJuridicoWriteService(fake)

	code, resp := peticion(t, http.MethodPost, "/v1/proveedores/juridica", cuerpoJuridico)
	if code != http.StatusCreated {
		t.Fatalf("code = %d, se esperaba 201; resp=%v", code, resp)
	}
	if !fake.llamado || fake.verbo != helper.WriteVerbCreate || fake.terceroID != 0 {
		t.Fatalf("llamada inesperada al servicio: %+v", fake)
	}
	if resp["Success"] != true || resp["Status"] != "201" {
		t.Errorf("envelope inesperado: %+v", resp)
	}

	data, ok := resp["Data"].(map[string]any)
	if !ok {
		t.Fatalf("Data no es objeto: %+v", resp["Data"])
	}
	if int(data["TerceroId"].(float64)) != 101 {
		t.Errorf("TerceroId = %v", data["TerceroId"])
	}
	basicos, ok := data["DatosBasicos"].(map[string]any)
	if !ok {
		t.Fatalf("DatosBasicos no es objeto: %+v", data["DatosBasicos"])
	}
	if basicos["RazonSocial"] != "ACME S.A.S." {
		t.Errorf("RazonSocial inesperada: %+v", basicos)
	}
	if _, trae := basicos["NombreCompleto"]; trae {
		t.Errorf("DatosBasicos no debe exponer NombreCompleto: %+v", basicos)
	}
	if _, trae := basicos["PrimerNombre"]; trae {
		t.Errorf("DatosBasicos no debe exponer PrimerNombre: %+v", basicos)
	}
	if _, trae := basicos["DatosIdentificacion"]; trae {
		t.Errorf("DatosBasicos no debe incluir la colección DatosIdentificacion: %+v", basicos)
	}
}

func TestPutJuridicaUsaTerceroIdDeLaRuta(t *testing.T) {
	fake := &escritorJuridicoFake{resultado: services.TerceroWriteResult{
		Data:                  helper.WriteResult{Data: map[string]any{"Id": 101}},
		TerceroId:             101,
		DatosIdentificacionId: 140,
		Actualizados: []models.EntidadId{
			{Entidad: "Tercero", Id: 101},
		},
	}}
	controllers.ConfigurarTerceroJuridicoWriteService(fake)

	code, resp := peticion(t, http.MethodPut, "/v1/proveedores/juridica/101", cuerpoJuridico)
	if code != http.StatusOK {
		t.Fatalf("code = %d, se esperaba 200; resp=%v", code, resp)
	}
	if fake.terceroID != 101 || fake.verbo != helper.WriteVerbUpdate {
		t.Fatalf("llamada inesperada al servicio: %+v", fake)
	}
	if resp["Status"] != "200" {
		t.Errorf("Status inesperado: %+v", resp)
	}
}

func TestPostJuridicaRechazaFaltaRazonSocial(t *testing.T) {
	fake := &escritorJuridicoFake{resultado: resultadoJuridicoFake()}
	controllers.ConfigurarTerceroJuridicoWriteService(fake)

	cuerpo := strings.Replace(cuerpoJuridico, `"RazonSocial": "ACME S.A.S.",`, "", 1)
	code, resp := peticion(t, http.MethodPost, "/v1/proveedores/juridica", cuerpo)
	if code != http.StatusBadRequest {
		t.Fatalf("code = %d, se esperaba 400; resp=%v", code, resp)
	}
	if fake.llamado {
		t.Error("no debe invocar el servicio sin RazonSocial")
	}
	if resp["Success"] != false || resp["Status"] != "400" {
		t.Errorf("envelope de error inesperado: %+v", resp)
	}
}

func TestPostJuridicaRechazaFaltaTipoContribuyente(t *testing.T) {
	fake := &escritorJuridicoFake{resultado: resultadoJuridicoFake()}
	controllers.ConfigurarTerceroJuridicoWriteService(fake)

	cuerpo := strings.Replace(cuerpoJuridico, `"TipoContribuyenteId": 2`, `"TipoContribuyenteId": 0`, 1)
	code, _ := peticion(t, http.MethodPost, "/v1/proveedores/juridica", cuerpo)
	if code != http.StatusBadRequest {
		t.Fatalf("code = %d, se esperaba 400", code)
	}
	if fake.llamado {
		t.Error("no debe invocar el servicio sin TipoContribuyenteId")
	}
}

func TestPostJuridicaRechazaNombresNaturales(t *testing.T) {
	fake := &escritorJuridicoFake{resultado: resultadoJuridicoFake()}
	controllers.ConfigurarTerceroJuridicoWriteService(fake)

	cuerpo := strings.Replace(cuerpoJuridico,
		`"DatosBasicos": {`, `"DatosBasicos": {"PrimerNombre": "Ana",`, 1)
	code, _ := peticion(t, http.MethodPost, "/v1/proveedores/juridica", cuerpo)
	if code != http.StatusBadRequest {
		t.Fatalf("code = %d, se esperaba 400", code)
	}
	if fake.llamado {
		t.Error("no debe invocar el servicio con nombres de persona natural")
	}
}

func TestPostJuridicaRechazaTipoProveedor(t *testing.T) {
	fake := &escritorJuridicoFake{resultado: resultadoJuridicoFake()}
	controllers.ConfigurarTerceroJuridicoWriteService(fake)

	cuerpo := strings.Replace(cuerpoJuridico, `"DatosBasicos": {`, `"TipoProveedor": 1, "DatosBasicos": {`, 1)
	code, _ := peticion(t, http.MethodPost, "/v1/proveedores/juridica", cuerpo)
	if code != http.StatusBadRequest {
		t.Fatalf("code = %d, se esperaba 400", code)
	}
	if fake.llamado {
		t.Error("no debe invocar el servicio con TipoProveedor")
	}
}

func TestPutJuridicaMapeaTerceroNoEncontrado(t *testing.T) {
	fake := &escritorJuridicoFake{err: services.ErrTerceroNoEncontrado}
	controllers.ConfigurarTerceroJuridicoWriteService(fake)

	code, resp := peticion(t, http.MethodPut, "/v1/proveedores/juridica/101", cuerpoJuridico)
	if code != http.StatusNotFound {
		t.Fatalf("code = %d, se esperaba 404; resp=%v", code, resp)
	}
	if resp["Success"] != false || resp["Status"] != "404" {
		t.Errorf("envelope de error inesperado: %+v", resp)
	}
}

func TestPutJuridicaMapeaIdentificacionInmutable(t *testing.T) {
	fake := &escritorJuridicoFake{err: services.ErrIdentificacionInmutable}
	controllers.ConfigurarTerceroJuridicoWriteService(fake)

	code, resp := peticion(t, http.MethodPut, "/v1/proveedores/juridica/101", cuerpoJuridico)
	if code != http.StatusConflict {
		t.Fatalf("code = %d, se esperaba 409; resp=%v", code, resp)
	}
	if resp["Success"] != false || resp["Status"] != "409" {
		t.Errorf("envelope de error inesperado: %+v", resp)
	}
}

func TestPutJuridicaRechazaTerceroIdInvalido(t *testing.T) {
	fake := &escritorJuridicoFake{resultado: resultadoJuridicoFake()}
	controllers.ConfigurarTerceroJuridicoWriteService(fake)

	code, _ := peticion(t, http.MethodPut, "/v1/proveedores/juridica/abc", cuerpoJuridico)
	if code != http.StatusBadRequest {
		t.Fatalf("code = %d, se esperaba 400", code)
	}
	if fake.llamado {
		t.Error("no debe invocar el servicio con un id inválido")
	}
}

func TestPutJuridicaMapeaTerceroInactivo(t *testing.T) {
	fake := &escritorJuridicoFake{err: services.ErrTerceroInactivo}
	controllers.ConfigurarTerceroJuridicoWriteService(fake)

	code, resp := peticion(t, http.MethodPut, "/v1/proveedores/juridica/101", cuerpoJuridico)
	if code != http.StatusConflict {
		t.Fatalf("code = %d, se esperaba 409; resp=%v", code, resp)
	}
	if resp["Success"] != false || resp["Status"] != "409" {
		t.Errorf("envelope de error inesperado: %+v", resp)
	}
}

func TestPutJuridicaMapeaIdentificacionInactiva(t *testing.T) {
	fake := &escritorJuridicoFake{err: services.ErrIdentificacionInactiva}
	controllers.ConfigurarTerceroJuridicoWriteService(fake)

	code, resp := peticion(t, http.MethodPut, "/v1/proveedores/juridica/101", cuerpoJuridico)
	if code != http.StatusConflict {
		t.Fatalf("code = %d, se esperaba 409; resp=%v", code, resp)
	}
	if resp["Success"] != false || resp["Status"] != "409" {
		t.Errorf("envelope de error inesperado: %+v", resp)
	}
}

func TestPostJuridicaMapeaValidationError(t *testing.T) {
	fake := &escritorJuridicoFake{
		err: &services.ValidationError{Message: "DatosBasicos.RazonSocial no puede superar 255 caracteres"},
	}
	controllers.ConfigurarTerceroJuridicoWriteService(fake)

	code, resp := peticion(t, http.MethodPost, "/v1/proveedores/juridica", cuerpoJuridico)
	if code != http.StatusBadRequest {
		t.Fatalf("code = %d, se esperaba 400; resp=%v", code, resp)
	}
	if resp["Message"] != "DatosBasicos.RazonSocial no puede superar 255 caracteres" {
		t.Errorf("mensaje inesperado: %+v", resp)
	}
}

func TestPostJuridicaRechazaJSONConCola(t *testing.T) {
	fake := &escritorJuridicoFake{resultado: resultadoJuridicoFake()}
	controllers.ConfigurarTerceroJuridicoWriteService(fake)

	code, resp := peticion(t, http.MethodPost, "/v1/proveedores/juridica", cuerpoJuridico+"{}")
	if code != http.StatusBadRequest {
		t.Fatalf("code = %d, se esperaba 400; resp=%v", code, resp)
	}
	if fake.llamado {
		t.Error("no debe invocar el servicio con datos de cola")
	}
}

func TestPostJuridicaRechazaContentType(t *testing.T) {
	fake := &escritorJuridicoFake{resultado: resultadoJuridicoFake()}
	controllers.ConfigurarTerceroJuridicoWriteService(fake)

	r, err := http.NewRequest(http.MethodPost, "/v1/proveedores/juridica", strings.NewReader(cuerpoJuridico))
	if err != nil {
		t.Fatalf("creando petición: %v", err)
	}
	r.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()
	beego.BeeApp.Handlers.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("code = %d, se esperaba 400; body=%s", w.Code, w.Body.String())
	}
	if fake.llamado {
		t.Error("no debe invocar el servicio con Content-Type inválido")
	}
}

func TestPostJuridicaNormalizaFKs(t *testing.T) {
	fake := &escritorJuridicoFake{resultado: services.TerceroWriteResult{
		Data: helper.WriteResult{Data: map[string]any{
			"Id":                  101,
			"NombreCompleto":      "ACME S.A.S.",
			"Activo":              true,
			"TipoContribuyenteId": 2,
			"DatosIdentificacion": []any{
				map[string]any{
					"Id": 140, "Numero": "900123456",
					"TipoDocumentoId": 7, "TerceroId": 101,
				},
			},
		}},
		TerceroId:             101,
		DatosIdentificacionId: 140,
		Creados: []models.EntidadId{
			{Entidad: "Tercero", Id: 101},
			{Entidad: "DatosIdentificacion", Id: 140},
		},
	}}
	controllers.ConfigurarTerceroJuridicoWriteService(fake)

	code, resp := peticion(t, http.MethodPost, "/v1/proveedores/juridica", cuerpoJuridico)
	if code != http.StatusCreated {
		t.Fatalf("code = %d, se esperaba 201; resp=%v", code, resp)
	}
	data := resp["Data"].(map[string]any)
	basicos := data["DatosBasicos"].(map[string]any)
	if _, ok := basicos["TipoContribuyenteId"].(map[string]any); !ok {
		t.Errorf("TipoContribuyenteId debe ser objeto: %+v", basicos["TipoContribuyenteId"])
	}
	ident := data["DatosIdentificacion"].(map[string]any)
	if _, ok := ident["TipoDocumentoId"].(map[string]any); !ok {
		t.Errorf("TipoDocumentoId debe ser objeto: %+v", ident["TipoDocumentoId"])
	}
	if _, ok := ident["TerceroId"].(map[string]any); !ok {
		t.Errorf("TerceroId debe ser objeto: %+v", ident["TerceroId"])
	}
}
