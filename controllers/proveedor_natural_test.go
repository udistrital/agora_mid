package controllers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	beego "github.com/beego/beego/v2/server/web"
	"github.com/udistrital/utils_oas/v2/customerror"

	"github.com/udistrital/agora_mid/controllers"
	"github.com/udistrital/agora_mid/helper"
	"github.com/udistrital/agora_mid/models"
	_ "github.com/udistrital/agora_mid/routers"
	"github.com/udistrital/agora_mid/services"
)

type escritorFake struct {
	resultado services.TerceroWriteResult
	err       error
	llamado   bool
	terceroID int64
	verbo     string
}

func (f *escritorFake) EscribirTercero(
	_ context.Context,
	_ models.RegistroTerceroNatural,
	terceroID int64,
	verbo string,
) (services.TerceroWriteResult, error) {
	f.llamado = true
	f.terceroID = terceroID
	f.verbo = verbo
	return f.resultado, f.err
}

func TestMain(m *testing.M) {
	beego.BConfig.WebConfig.AutoRender = false
	beego.ErrorController(&customerror.CustomErrorController{})
	os.Exit(m.Run())
}

func peticion(t *testing.T, metodo, ruta, cuerpo string) (int, map[string]any) {
	t.Helper()
	r, err := http.NewRequest(metodo, ruta, strings.NewReader(cuerpo))
	if err != nil {
		t.Fatalf("creando petición: %v", err)
	}
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	beego.BeeApp.Handlers.ServeHTTP(w, r)
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	return w.Code, resp
}

const cuerpoNatural = `{
  "DatosBasicos": {
    "PrimerNombre": "Ana",
    "SegundoNombre": "",
    "PrimerApellido": "Pérez",
    "SegundoApellido": "",
    "FechaNacimiento": "1990-05-10T00:00:00Z",
    "LugarOrigen": 149,
    "TipoContribuyenteId": 1
  },
  "DatosIdentificacion": {
    "Numero": "1012345678",
    "TipoDocumentoId": 3,
    "CiudadExpedicion": 149,
    "FechaExpedicion": "2010-01-01T00:00:00Z",
    "DigitoVerificacion": 0
  }
}`

func resultadoFake() services.TerceroWriteResult {
	return services.TerceroWriteResult{
		Data: helper.WriteResult{Data: map[string]any{
			"Id":             101,
			"NombreCompleto": "Ana Pérez",
			"DatosIdentificacion": []any{
				map[string]any{"Id": 140, "Numero": "1012345678"},
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

func TestPostNaturalCreaYComponeRespuesta(t *testing.T) {
	fake := &escritorFake{resultado: resultadoFake()}
	controllers.ConfigurarTerceroWriteService(fake)

	code, resp := peticion(t, http.MethodPost, "/v1/proveedores/natural", cuerpoNatural)
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
	if int(data["DatosIdentificacionId"].(float64)) != 140 {
		t.Errorf("DatosIdentificacionId = %v", data["DatosIdentificacionId"])
	}
	if _, trae := data["DatosBasicos"].(map[string]any)["DatosIdentificacion"]; trae {
		t.Error("DatosBasicos no debe incluir la colección DatosIdentificacion")
	}
}

func TestPutNaturalUsaTerceroIdDeLaRuta(t *testing.T) {
	fake := &escritorFake{resultado: services.TerceroWriteResult{
		Data:                  helper.WriteResult{Data: map[string]any{"Id": 101}},
		TerceroId:             101,
		DatosIdentificacionId: 140,
		Actualizados: []models.EntidadId{
			{Entidad: "Tercero", Id: 101},
		},
	}}
	controllers.ConfigurarTerceroWriteService(fake)

	code, resp := peticion(t, http.MethodPut, "/v1/proveedores/natural/101", cuerpoNatural)
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

func TestPostNaturalRechazaTipoProveedor(t *testing.T) {
	fake := &escritorFake{resultado: resultadoFake()}
	controllers.ConfigurarTerceroWriteService(fake)

	cuerpo := strings.Replace(cuerpoNatural, `"DatosBasicos": {`, `"TipoProveedor": 1, "DatosBasicos": {`, 1)
	code, resp := peticion(t, http.MethodPost, "/v1/proveedores/natural", cuerpo)
	if code != http.StatusBadRequest {
		t.Fatalf("code = %d, se esperaba 400; resp=%v", code, resp)
	}
	if fake.llamado {
		t.Error("no debe invocar el servicio con campos no permitidos")
	}
	if resp["Success"] != false || resp["Status"] != "400" {
		t.Errorf("envelope de error inesperado: %+v", resp)
	}
}

func TestPostNaturalRechazaFaltaTipoContribuyente(t *testing.T) {
	fake := &escritorFake{resultado: resultadoFake()}
	controllers.ConfigurarTerceroWriteService(fake)

	cuerpo := strings.Replace(cuerpoNatural, `"TipoContribuyenteId": 1`, `"TipoContribuyenteId": 0`, 1)
	code, _ := peticion(t, http.MethodPost, "/v1/proveedores/natural", cuerpo)
	if code != http.StatusBadRequest {
		t.Fatalf("code = %d, se esperaba 400", code)
	}
	if fake.llamado {
		t.Error("no debe invocar el servicio sin TipoContribuyenteId")
	}
}

func TestPutNaturalMapeaTerceroNoEncontrado(t *testing.T) {
	fake := &escritorFake{err: services.ErrTerceroNoEncontrado}
	controllers.ConfigurarTerceroWriteService(fake)

	code, resp := peticion(t, http.MethodPut, "/v1/proveedores/natural/101", cuerpoNatural)
	if code != http.StatusNotFound {
		t.Fatalf("code = %d, se esperaba 404; resp=%v", code, resp)
	}
	if resp["Success"] != false || resp["Status"] != "404" {
		t.Errorf("envelope de error inesperado: %+v", resp)
	}
}

func TestPutNaturalRechazaTerceroIdInvalido(t *testing.T) {
	fake := &escritorFake{resultado: resultadoFake()}
	controllers.ConfigurarTerceroWriteService(fake)

	code, _ := peticion(t, http.MethodPut, "/v1/proveedores/natural/abc", cuerpoNatural)
	if code != http.StatusBadRequest {
		t.Fatalf("code = %d, se esperaba 400", code)
	}
	if fake.llamado {
		t.Error("no debe invocar el servicio con un id inválido")
	}
}
