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

type escritorCaracterizacionJuridicaFake struct {
	resultado   services.CaracterizacionJuridicaResult
	err         error
	llamado     bool
	terceroID   int64
	proveedorID int64
	verbo       string
}

func (f *escritorCaracterizacionJuridicaFake) Registrar(
	_ context.Context,
	_ models.RegistroCaracterizacionJuridica,
	terceroID int64,
	verbo string,
) (services.CaracterizacionJuridicaResult, error) {
	f.llamado = true
	f.terceroID = terceroID
	f.verbo = verbo
	return f.resultado, f.err
}

func (f *escritorCaracterizacionJuridicaFake) RegistrarPorProveedor(
	_ context.Context,
	_ models.RegistroCaracterizacionJuridica,
	proveedorID int64,
	verbo string,
) (services.CaracterizacionJuridicaResult, error) {
	f.llamado = true
	f.proveedorID = proveedorID
	f.verbo = verbo
	return f.resultado, f.err
}

func resultadoCaracterizacionJuridicaFake(parcial bool) services.CaracterizacionJuridicaResult {
	resultado := services.CaracterizacionJuridicaResult{
		Data: models.CaracterizacionJuridicaData{
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
			Path: "ProveedorJuridico", Service: "gestion_proveedores_crud",
			Message: "fallo simulado", Kind: "dependency_failure",
		}}
		return resultado
	}
	resultado.Data.Resultado.Creados = []models.EntidadId{
		{Entidad: "Proveedor", Id: 77},
	}
	return resultado
}

const cuerpoCaracterizacionJuridica = `{
  "TerceroId": 101,
  "DescripcionPortafolio": "Servicios corporativos"
}`

func TestPostCaracterizacionJuridicaCrea(t *testing.T) {
	fake := &escritorCaracterizacionJuridicaFake{
		resultado: resultadoCaracterizacionJuridicaFake(false),
	}
	controllers.ConfigurarCaracterizacionJuridicaWriteService(fake)

	codigo, resp := peticion(t, http.MethodPost,
		"/v1/proveedores/juridica/caracterizacion", cuerpoCaracterizacionJuridica)
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

func TestPostCaracterizacionJuridicaParcial(t *testing.T) {
	fake := &escritorCaracterizacionJuridicaFake{
		resultado: resultadoCaracterizacionJuridicaFake(true),
	}
	controllers.ConfigurarCaracterizacionJuridicaWriteService(fake)

	codigo, resp := peticion(t, http.MethodPost,
		"/v1/proveedores/juridica/caracterizacion", cuerpoCaracterizacionJuridica)
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

func TestPutCaracterizacionJuridicaUsaTerceroIdDeLaRuta(t *testing.T) {
	fake := &escritorCaracterizacionJuridicaFake{
		resultado: resultadoCaracterizacionJuridicaFake(false),
	}
	controllers.ConfigurarCaracterizacionJuridicaWriteService(fake)

	codigo, _ := peticion(t, http.MethodPut,
		"/v1/proveedores/juridica/caracterizacion/101", cuerpoCaracterizacionJuridica)
	if codigo != http.StatusCreated {
		t.Fatalf("status = %d, se esperaba 201", codigo)
	}
	if !fake.llamado || fake.terceroID != 101 || fake.verbo != helper.WriteVerbUpdate {
		t.Fatalf("llamada inesperada: %+v", fake)
	}
}

func TestPutCaracterizacionJuridicaRechazaTerceroIdDistinto(t *testing.T) {
	fake := &escritorCaracterizacionJuridicaFake{
		resultado: resultadoCaracterizacionJuridicaFake(false),
	}
	controllers.ConfigurarCaracterizacionJuridicaWriteService(fake)

	codigo, _ := peticion(t, http.MethodPut,
		"/v1/proveedores/juridica/caracterizacion/102", cuerpoCaracterizacionJuridica)
	if codigo != http.StatusBadRequest {
		t.Fatalf("status = %d, se esperaba 400", codigo)
	}
}

func TestPutCaracterizacionJuridicaPorProveedor(t *testing.T) {
	fake := &escritorCaracterizacionJuridicaFake{
		resultado: resultadoCaracterizacionJuridicaFake(false),
	}
	controllers.ConfigurarCaracterizacionJuridicaWriteService(fake)
	cuerpo := `{"DescripcionPortafolio":"Servicios corporativos"}`

	codigo, _ := peticion(t, http.MethodPut,
		"/v1/proveedores/juridica/caracterizacion/proveedor/77", cuerpo)
	if codigo != http.StatusCreated {
		t.Fatalf("status = %d, se esperaba 201", codigo)
	}
	if !fake.llamado || fake.proveedorID != 77 || fake.verbo != helper.WriteVerbUpdate {
		t.Fatalf("llamada inesperada: %+v", fake)
	}
}

func TestPostCaracterizacionJuridicaSinTerceroId(t *testing.T) {
	fake := &escritorCaracterizacionJuridicaFake{
		resultado: resultadoCaracterizacionJuridicaFake(false),
	}
	controllers.ConfigurarCaracterizacionJuridicaWriteService(fake)

	codigo, _ := peticion(t, http.MethodPost,
		"/v1/proveedores/juridica/caracterizacion", `{"DescripcionPortafolio":"x"}`)
	if codigo != http.StatusBadRequest {
		t.Fatalf("status = %d, se esperaba 400", codigo)
	}
}
