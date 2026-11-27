package controllers_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/udistrital/agora_mid/controllers"
	"github.com/udistrital/agora_mid/helper"
	"github.com/udistrital/agora_mid/models"
	"github.com/udistrital/agora_mid/responses"
	"github.com/udistrital/agora_mid/services"
)

type escritorCaracterizacionFake struct {
	resultado   services.CaracterizacionNaturalResult
	err         error
	llamado     bool
	terceroID   int64
	proveedorID int64
	verbo       string
}

func (f *escritorCaracterizacionFake) Registrar(
	_ context.Context,
	_ models.RegistroCaracterizacionNatural,
	terceroID int64,
	verbo string,
) (services.CaracterizacionNaturalResult, error) {
	f.llamado = true
	f.terceroID = terceroID
	f.verbo = verbo
	return f.resultado, f.err
}

func (f *escritorCaracterizacionFake) RegistrarPorProveedor(
	_ context.Context,
	_ models.RegistroCaracterizacionNatural,
	proveedorID int64,
	verbo string,
) (services.CaracterizacionNaturalResult, error) {
	f.llamado = true
	f.proveedorID = proveedorID
	f.verbo = verbo
	return f.resultado, f.err
}

func resultadoCaracterizacionFake(parcial bool) services.CaracterizacionNaturalResult {
	resultado := services.CaracterizacionNaturalResult{
		Data: models.CaracterizacionNaturalData{
			TerceroId:                   101,
			ProveedorId:                 77,
			InfoComplementariaTercero:   []models.InfoComplementariaTerceroRegistrada{},
			PerfilProveedor:             []models.EntidadId{},
			Contacto:                    []models.EntidadId{},
			ActividadEconomicaProveedor: []models.EntidadId{},
			Resultado: models.Resultado{
				Creados:      []models.EntidadId{},
				Actualizados: []models.EntidadId{},
				Reutilizados: []models.EntidadId{},
			},
		},
	}
	if parcial {
		resultado.Parcial = true
		resultado.Avisos = []responses.RelationError{{
			Path: "ProveedorNatural", Service: "gestion_proveedores_crud",
			Message: "fallo simulado", Kind: "dependency_failure",
		}}
		return resultado
	}
	resultado.Data.Resultado.Creados = []models.EntidadId{
		{Entidad: "Proveedor", Id: 77},
	}
	return resultado
}

const cuerpoCaracterizacion = `{
  "TerceroId": 101,
  "DescripcionPortafolio": "Servicios de consultoría"
}`

func TestPostCaracterizacionNaturalCrea(t *testing.T) {
	fake := &escritorCaracterizacionFake{resultado: resultadoCaracterizacionFake(false)}
	controllers.ConfigurarCaracterizacionNaturalWriteService(fake)

	codigo, resp := peticion(t, http.MethodPost,
		"/v1/proveedores/natural/caracterizacion", cuerpoCaracterizacion)
	if codigo != http.StatusCreated {
		t.Fatalf("status = %d, se esperaba 201; respuesta=%v", codigo, resp)
	}
	if resp["Status"] != "201" || resp["Success"] != true {
		t.Fatalf("envelope inesperado: %v", resp)
	}
	if !fake.llamado || fake.terceroID != 101 || fake.verbo != helper.WriteVerbCreate {
		t.Fatalf("llamada inesperada: %+v", fake)
	}
}

func TestPostCaracterizacionNaturalParcial(t *testing.T) {
	fake := &escritorCaracterizacionFake{resultado: resultadoCaracterizacionFake(true)}
	controllers.ConfigurarCaracterizacionNaturalWriteService(fake)

	codigo, resp := peticion(t, http.MethodPost,
		"/v1/proveedores/natural/caracterizacion", cuerpoCaracterizacion)
	if codigo != http.StatusPartialContent {
		t.Fatalf("status = %d, se esperaba 206; respuesta=%v", codigo, resp)
	}
	if resp["Status"] != "206" || resp["Success"] != false {
		t.Fatalf("envelope inesperado: %v", resp)
	}
	meta, ok := resp["MetaData"].(map[string]any)
	if !ok || meta["Errors"] == nil {
		t.Fatalf("se esperaba MetaData.Errors: %v", resp)
	}
}

func TestPutCaracterizacionNaturalUsaTerceroIdDeLaRuta(t *testing.T) {
	fake := &escritorCaracterizacionFake{resultado: resultadoCaracterizacionFake(false)}
	controllers.ConfigurarCaracterizacionNaturalWriteService(fake)

	codigo, _ := peticion(t, http.MethodPut,
		"/v1/proveedores/natural/caracterizacion/101", cuerpoCaracterizacion)
	if codigo != http.StatusCreated {
		t.Fatalf("status = %d, se esperaba 201", codigo)
	}
	if !fake.llamado || fake.terceroID != 101 || fake.verbo != helper.WriteVerbUpdate {
		t.Fatalf("llamada inesperada: %+v", fake)
	}
}

func TestPutCaracterizacionNaturalRechazaTerceroIdDistinto(t *testing.T) {
	fake := &escritorCaracterizacionFake{resultado: resultadoCaracterizacionFake(false)}
	controllers.ConfigurarCaracterizacionNaturalWriteService(fake)

	codigo, _ := peticion(t, http.MethodPut,
		"/v1/proveedores/natural/caracterizacion/102", cuerpoCaracterizacion)
	if codigo != http.StatusBadRequest {
		t.Fatalf("status = %d, se esperaba 400", codigo)
	}
	if fake.llamado {
		t.Fatal("no debía invocar el servicio")
	}
}

func TestPostCaracterizacionNaturalRechazaTerceroIdAusente(t *testing.T) {
	fake := &escritorCaracterizacionFake{resultado: resultadoCaracterizacionFake(false)}
	controllers.ConfigurarCaracterizacionNaturalWriteService(fake)

	codigo, _ := peticion(t, http.MethodPost,
		"/v1/proveedores/natural/caracterizacion", `{"DescripcionPortafolio":"x"}`)
	if codigo != http.StatusBadRequest {
		t.Fatalf("status = %d, se esperaba 400", codigo)
	}
	if fake.llamado {
		t.Fatal("no debía invocar el servicio")
	}
}

func TestPostCaracterizacionNaturalRechazaJSONConCola(t *testing.T) {
	fake := &escritorCaracterizacionFake{resultado: resultadoCaracterizacionFake(false)}
	controllers.ConfigurarCaracterizacionNaturalWriteService(fake)

	codigo, _ := peticion(t, http.MethodPost,
		"/v1/proveedores/natural/caracterizacion", cuerpoCaracterizacion+`{}`)
	if codigo != http.StatusBadRequest {
		t.Fatalf("status = %d, se esperaba 400", codigo)
	}
}

func TestPostCaracterizacionNaturalMapeaConflicto(t *testing.T) {
	fake := &escritorCaracterizacionFake{err: &services.CaracterizacionNaturalError{
		StatusCode: http.StatusConflict, Message: "conflicto simulado",
	}}
	controllers.ConfigurarCaracterizacionNaturalWriteService(fake)

	codigo, resp := peticion(t, http.MethodPost,
		"/v1/proveedores/natural/caracterizacion", cuerpoCaracterizacion)
	if codigo != http.StatusConflict {
		t.Fatalf("status = %d, se esperaba 409; respuesta=%v", codigo, resp)
	}
	if resp["Message"] != "conflicto simulado" {
		t.Fatalf("mensaje inesperado: %v", resp)
	}
}

func TestPostCaracterizacionNaturalMapeaTimeout(t *testing.T) {
	fake := &escritorCaracterizacionFake{err: &services.CaracterizacionNaturalError{
		StatusCode: http.StatusGatewayTimeout, Message: "timeout simulado",
	}}
	controllers.ConfigurarCaracterizacionNaturalWriteService(fake)

	codigo, resp := peticion(t, http.MethodPost,
		"/v1/proveedores/natural/caracterizacion", cuerpoCaracterizacion)
	if codigo != http.StatusGatewayTimeout {
		t.Fatalf("status = %d, se esperaba 504; respuesta=%v", codigo, resp)
	}
	if resp["Status"] != "504" {
		t.Fatalf("Status inesperado: %v", resp)
	}
}

func TestPutCaracterizacionNaturalPorProveedorId(t *testing.T) {
	fake := &escritorCaracterizacionFake{resultado: resultadoCaracterizacionFake(false)}
	controllers.ConfigurarCaracterizacionNaturalWriteService(fake)

	codigo, _ := peticion(t, http.MethodPut,
		"/v1/proveedores/natural/caracterizacion/proveedor/77", cuerpoCaracterizacion)
	if codigo != http.StatusCreated {
		t.Fatalf("status = %d, se esperaba 201", codigo)
	}
	if !fake.llamado || fake.proveedorID != 77 || fake.verbo != helper.WriteVerbUpdate {
		t.Fatalf("llamada inesperada: %+v", fake)
	}
}

func TestPutCaracterizacionNaturalRechazaProveedorIdInvalido(t *testing.T) {
	fake := &escritorCaracterizacionFake{resultado: resultadoCaracterizacionFake(false)}
	controllers.ConfigurarCaracterizacionNaturalWriteService(fake)

	codigo, _ := peticion(t, http.MethodPut,
		"/v1/proveedores/natural/caracterizacion/proveedor/abc", cuerpoCaracterizacion)
	if codigo != http.StatusBadRequest {
		t.Fatalf("status = %d, se esperaba 400", codigo)
	}
	if fake.llamado {
		t.Fatal("no debía invocar el servicio")
	}
}
