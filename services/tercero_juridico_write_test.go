package services

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/udistrital/agora_mid/helper"
	"github.com/udistrital/agora_mid/models"
)

func entradaJuridicaPrueba() models.RegistroTerceroJuridico {
	lugar := int64(149)
	digito := float64(7)
	return models.RegistroTerceroJuridico{
		DatosBasicos: models.DatosBasicosJuridico{
			RazonSocial:         "ACME S.A.S.",
			LugarOrigen:         &lugar,
			TipoContribuyenteId: 2,
		},
		DatosIdentificacion: models.DatosIdentificacionTercero{
			Numero:             "900123456",
			TipoDocumentoId:    7,
			DigitoVerificacion: &digito,
		},
	}
}

func terceroJuridicoFake() map[string]any {
	return map[string]any{
		"Id":                  101,
		"NombreCompleto":      "ACME S.A.S.",
		"LugarOrigen":         149,
		"Activo":              true,
		"TipoContribuyenteId": map[string]any{"Id": 2},
	}
}

func identificacionJuridicaFake() map[string]any {
	return map[string]any{
		"Id":              140,
		"TerceroId":       map[string]any{"Id": 101},
		"TipoDocumentoId": map[string]any{"Id": 7},
		"Numero":          "900123456",
		"Activo":          true,
	}
}

func TestEscribirTerceroJuridicoCreaTerceroEIdentificacion(t *testing.T) {
	fake := nuevoCrudFake()
	servicio := servicioTerceroPrueba(t, fake)

	resultado, err := servicio.EscribirTerceroJuridico(
		context.Background(), entradaJuridicaPrueba(), 0, helper.WriteVerbCreate,
	)
	if err != nil {
		t.Fatalf("EscribirTerceroJuridico no debe fallar: %v", err)
	}
	if resultado.TerceroId != 101 || resultado.DatosIdentificacionId != 140 {
		t.Fatalf("ids inesperados: tercero=%d ident=%d", resultado.TerceroId, resultado.DatosIdentificacionId)
	}
	if len(resultado.Creados) != 2 {
		t.Fatalf("se esperaban dos creaciones, obtuvo: %+v", resultado.Creados)
	}

	postTercero := buscarCaptura(fake.capturas, http.MethodPost, "/v1/tercero/")
	if postTercero == nil {
		t.Fatalf("no se envió POST de tercero; capturas=%+v", fake.capturas)
	}
	if postTercero.cuerpo["Activo"] != true {
		t.Errorf("el tercero debe crearse activo: %+v", postTercero.cuerpo)
	}
	if idDeRelacion(postTercero.cuerpo["TipoContribuyenteId"]) != 2 {
		t.Errorf("TipoContribuyenteId debe viajar como objeto Id=2: %+v", postTercero.cuerpo["TipoContribuyenteId"])
	}
	if postTercero.cuerpo["NombreCompleto"] != "ACME S.A.S." {
		t.Errorf("NombreCompleto inesperado: %v", postTercero.cuerpo["NombreCompleto"])
	}
	if _, trae := postTercero.cuerpo["PrimerNombre"]; trae {
		t.Errorf("el alta jurídica no debe enviar PrimerNombre: %+v", postTercero.cuerpo)
	}
	if postTercero.cuerpo["LugarOrigen"] != float64(149) && postTercero.cuerpo["LugarOrigen"] != int64(149) {
		t.Errorf("LugarOrigen inesperado: %v", postTercero.cuerpo["LugarOrigen"])
	}

	postIdent := buscarCaptura(fake.capturas, http.MethodPost, "/v1/datos_identificacion/")
	if postIdent == nil {
		t.Fatalf("no se envió POST de identificación; capturas=%+v", fake.capturas)
	}
	if postIdent.cuerpo["Activo"] != true {
		t.Errorf("la identificación debe crearse activa: %+v", postIdent.cuerpo)
	}
	if idDeRelacion(postIdent.cuerpo["TipoDocumentoId"]) != 7 {
		t.Errorf("TipoDocumentoId debe viajar como objeto Id=7: %+v", postIdent.cuerpo["TipoDocumentoId"])
	}
	if idDeRelacion(postIdent.cuerpo["TerceroId"]) != 101 {
		t.Errorf("TerceroId debe viajar como objeto Id=101: %+v", postIdent.cuerpo["TerceroId"])
	}
	if postIdent.cuerpo["DigitoVerificacion"] != float64(7) {
		t.Errorf("DigitoVerificacion inesperado: %v", postIdent.cuerpo["DigitoVerificacion"])
	}
}

func TestEscribirTerceroJuridicoReutilizaDocumentoExistente(t *testing.T) {
	fake := nuevoCrudFake()
	fake.terceros[101] = terceroJuridicoFake()
	fake.identificaciones[140] = identificacionJuridicaFake()

	resultado, err := servicioTerceroPrueba(t, fake).EscribirTerceroJuridico(
		context.Background(), entradaJuridicaPrueba(), 0, helper.WriteVerbCreate,
	)
	if err != nil {
		t.Fatalf("EscribirTerceroJuridico no debe fallar: %v", err)
	}
	if resultado.TerceroId != 101 || resultado.DatosIdentificacionId != 140 {
		t.Fatalf("ids inesperados: tercero=%d ident=%d", resultado.TerceroId, resultado.DatosIdentificacionId)
	}
	if len(resultado.Reutilizados) != 1 || resultado.Reutilizados[0].Entidad != "Tercero" {
		t.Fatalf("se esperaba reutilizar el tercero: %+v", resultado.Reutilizados)
	}

	if post := buscarCaptura(fake.capturas, http.MethodPost, "/v1/tercero/"); post != nil {
		t.Fatalf("no debe crear un tercero cuando el documento ya existe: %+v", post)
	}
	if put := buscarCaptura(fake.capturas, http.MethodPut, "/v1/tercero/101"); put != nil {
		t.Fatalf("no debe reescribir el tercero en un POST de reutilización: %+v", put)
	}
	if resultado.Data.Data["NombreCompleto"] != "ACME S.A.S." {
		t.Errorf("el tercero debe conservar NombreCompleto: %+v", resultado.Data.Data)
	}
}

func TestEscribirTerceroJuridicoActualizaRazonSocialYLugarOrigen(t *testing.T) {
	fake := nuevoCrudFake()
	existente := terceroJuridicoFake()
	existente["PrimerNombre"] = "Viejo"
	fake.terceros[101] = existente
	fake.identificaciones[140] = identificacionJuridicaFake()

	entrada := entradaJuridicaPrueba()
	entrada.DatosBasicos.RazonSocial = "  ACME   TECHNOLOGIES S.A.S.  "
	lugar := int64(150)
	entrada.DatosBasicos.LugarOrigen = &lugar

	resultado, err := servicioTerceroPrueba(t, fake).EscribirTerceroJuridico(
		context.Background(), entrada, 101, helper.WriteVerbUpdate,
	)
	if err != nil {
		t.Fatalf("EscribirTerceroJuridico no debe fallar: %v", err)
	}
	if resultado.DatosIdentificacionId != 140 {
		t.Fatalf("se esperaba reutilizar la identificación 140; obtuvo %d", resultado.DatosIdentificacionId)
	}

	putTercero := buscarCaptura(fake.capturas, http.MethodPut, "/v1/tercero/101")
	if putTercero == nil {
		t.Fatalf("no se envió PUT de tercero; capturas=%+v", fake.capturas)
	}
	if putTercero.cuerpo["NombreCompleto"] != "ACME TECHNOLOGIES S.A.S." {
		t.Errorf("NombreCompleto inesperado: %v", putTercero.cuerpo["NombreCompleto"])
	}
	if putTercero.cuerpo["LugarOrigen"] != float64(150) && putTercero.cuerpo["LugarOrigen"] != int64(150) {
		t.Errorf("LugarOrigen inesperado: %v", putTercero.cuerpo["LugarOrigen"])
	}
	if putTercero.cuerpo["Activo"] != true {
		t.Errorf("el PUT debe conservar Activo: %+v", putTercero.cuerpo)
	}
	if putTercero.cuerpo["PrimerNombre"] != "Viejo" {
		t.Errorf("el PUT debe conservar campos no enviados: %+v", putTercero.cuerpo)
	}
}

func TestEscribirTerceroJuridicoPutCreaIdentificacion(t *testing.T) {
	fake := nuevoCrudFake()
	fake.terceros[101] = terceroJuridicoFake()

	resultado, err := servicioTerceroPrueba(t, fake).EscribirTerceroJuridico(
		context.Background(), entradaJuridicaPrueba(), 101, helper.WriteVerbUpdate,
	)
	if err != nil {
		t.Fatalf("EscribirTerceroJuridico no debe fallar: %v", err)
	}
	if resultado.DatosIdentificacionId != 140 {
		t.Fatalf("se esperaba crear la identificación 140; obtuvo %d", resultado.DatosIdentificacionId)
	}
	if len(resultado.Creados) != 1 || resultado.Creados[0].Entidad != "DatosIdentificacion" {
		t.Fatalf("se esperaba crear la identificación: %+v", resultado.Creados)
	}

	postIdent := buscarCaptura(fake.capturas, http.MethodPost, "/v1/datos_identificacion/")
	if postIdent == nil {
		t.Fatalf("no se envió POST de identificación; capturas=%+v", fake.capturas)
	}
	if idDeRelacion(postIdent.cuerpo["TerceroId"]) != 101 {
		t.Errorf("TerceroId inesperado: %+v", postIdent.cuerpo["TerceroId"])
	}
}

func TestEscribirTerceroJuridicoRechazaTipoNatural(t *testing.T) {
	fake := nuevoCrudFake()
	entrada := entradaJuridicaPrueba()
	entrada.DatosBasicos.TipoContribuyenteId = 1

	_, err := servicioTerceroPrueba(t, fake).EscribirTerceroJuridico(
		context.Background(), entrada, 0, helper.WriteVerbCreate,
	)
	if !errors.Is(err, ErrTipoContribuyenteJuridicoInvalido) {
		t.Fatalf("se esperaba ErrTipoContribuyenteJuridicoInvalido; obtuvo %v", err)
	}
	if len(fake.capturas) != 0 {
		t.Fatalf("no debe llamar al CRUD con un tipo inválido; capturas=%+v", fake.capturas)
	}
}

func TestEscribirTerceroJuridicoRechazaCambioDeIdentificacion(t *testing.T) {
	fake := nuevoCrudFake()
	fake.terceros[101] = terceroJuridicoFake()
	fake.identificaciones[140] = identificacionJuridicaFake()

	entrada := entradaJuridicaPrueba()
	entrada.DatosIdentificacion.Numero = "900999999"

	_, err := servicioTerceroPrueba(t, fake).EscribirTerceroJuridico(
		context.Background(), entrada, 101, helper.WriteVerbUpdate,
	)
	if !errors.Is(err, ErrIdentificacionInmutable) {
		t.Fatalf("se esperaba ErrIdentificacionInmutable; obtuvo %v", err)
	}
	for _, captura := range fake.capturas {
		if captura.metodo == http.MethodPut || captura.metodo == http.MethodPost {
			t.Fatalf("no debe escribir si cambia la identidad; capturas=%+v", fake.capturas)
		}
	}
}

func TestEscribirTerceroJuridicoRechazaTipoDocumentoNoNIT(t *testing.T) {
	fake := nuevoCrudFake()
	fake.terceros[101] = terceroJuridicoFake()
	fake.identificaciones[140] = identificacionJuridicaFake()

	entrada := entradaJuridicaPrueba()
	entrada.DatosIdentificacion.TipoDocumentoId = 3

	_, err := servicioTerceroPrueba(t, fake).EscribirTerceroJuridico(
		context.Background(), entrada, 101, helper.WriteVerbUpdate,
	)
	var validacion *ValidationError
	if !errors.As(err, &validacion) {
		t.Fatalf("se esperaba ValidationError; obtuvo %v", err)
	}
	if len(fake.capturas) != 0 {
		t.Fatalf("no debe consultar ni escribir con un tipo no permitido; capturas=%+v", fake.capturas)
	}
}

func TestEscribirTerceroJuridicoRechazaDocumentoDeOtroTercero(t *testing.T) {
	fake := nuevoCrudFake()
	fake.terceros[101] = terceroJuridicoFake()
	ident := identificacionJuridicaFake()
	ident["TerceroId"] = map[string]any{"Id": 999}
	fake.identificaciones[140] = ident

	_, err := servicioTerceroPrueba(t, fake).EscribirTerceroJuridico(
		context.Background(), entradaJuridicaPrueba(), 101, helper.WriteVerbUpdate,
	)
	if !errors.Is(err, ErrIdentificacionDeOtroTercero) {
		t.Fatalf("se esperaba ErrIdentificacionDeOtroTercero; obtuvo %v", err)
	}
}

func TestEscribirTerceroJuridicoRechazaTerceroNatural(t *testing.T) {
	fake := nuevoCrudFake()
	fake.terceros[101] = map[string]any{
		"Id":                  101,
		"NombreCompleto":      "Ana Pérez",
		"Activo":              true,
		"TipoContribuyenteId": map[string]any{"Id": 1},
	}

	_, err := servicioTerceroPrueba(t, fake).EscribirTerceroJuridico(
		context.Background(), entradaJuridicaPrueba(), 101, helper.WriteVerbUpdate,
	)
	if !errors.Is(err, ErrTipoContribuyenteConflicto) {
		t.Fatalf("se esperaba ErrTipoContribuyenteConflicto; obtuvo %v", err)
	}
}

func TestEscribirTerceroJuridicoPutInexistente(t *testing.T) {
	fake := nuevoCrudFake()
	fake.vacioComoObjeto = true

	_, err := servicioTerceroPrueba(t, fake).EscribirTerceroJuridico(
		context.Background(), entradaJuridicaPrueba(), 999999, helper.WriteVerbUpdate,
	)
	if !errors.Is(err, ErrTerceroNoEncontrado) {
		t.Fatalf("se esperaba ErrTerceroNoEncontrado; obtuvo %v", err)
	}
	for _, captura := range fake.capturas {
		if captura.metodo == http.MethodPost || captura.metodo == http.MethodPut {
			t.Fatalf("no debe escribir con un tercero inexistente; capturas=%+v", fake.capturas)
		}
	}
}

func TestEscribirTerceroJuridicoCreaConRespuestaVaciaObjeto(t *testing.T) {
	fake := nuevoCrudFake()
	fake.vacioComoObjeto = true

	resultado, err := servicioTerceroPrueba(t, fake).EscribirTerceroJuridico(
		context.Background(), entradaJuridicaPrueba(), 0, helper.WriteVerbCreate,
	)
	if err != nil {
		t.Fatalf("EscribirTerceroJuridico no debe fallar con [{}]: %v", err)
	}
	if resultado.TerceroId != 101 || resultado.DatosIdentificacionId != 140 {
		t.Fatalf("ids inesperados: tercero=%d ident=%d", resultado.TerceroId, resultado.DatosIdentificacionId)
	}
}

func sinEscrituras(t *testing.T, fake *crudTercerosFake) {
	t.Helper()
	for _, captura := range fake.capturas {
		if captura.metodo == http.MethodPost || captura.metodo == http.MethodPut {
			t.Fatalf("no debe escribir con datos inválidos; capturas=%+v", fake.capturas)
		}
	}
}

func exigirValidacion(t *testing.T, err error) {
	t.Helper()
	var validacion *ValidationError
	if !errors.As(err, &validacion) {
		t.Fatalf("se esperaba ValidationError; obtuvo %v", err)
	}
}

func TestEscribirTerceroJuridicoRechazaNITSinDV(t *testing.T) {
	fake := nuevoCrudFake()
	entrada := entradaJuridicaPrueba()
	entrada.DatosIdentificacion.DigitoVerificacion = nil

	_, err := servicioTerceroPrueba(t, fake).EscribirTerceroJuridico(
		context.Background(), entrada, 0, helper.WriteVerbCreate,
	)
	exigirValidacion(t, err)
	sinEscrituras(t, fake)
}

func TestEscribirTerceroJuridicoRechazaNITConLetras(t *testing.T) {
	fake := nuevoCrudFake()
	entrada := entradaJuridicaPrueba()
	entrada.DatosIdentificacion.Numero = "90012A456"

	_, err := servicioTerceroPrueba(t, fake).EscribirTerceroJuridico(
		context.Background(), entrada, 0, helper.WriteVerbCreate,
	)
	exigirValidacion(t, err)
	sinEscrituras(t, fake)
}

func TestEscribirTerceroJuridicoRechazaDVFueraDeRango(t *testing.T) {
	fake := nuevoCrudFake()
	entrada := entradaJuridicaPrueba()
	digito := float64(10)
	entrada.DatosIdentificacion.DigitoVerificacion = &digito

	_, err := servicioTerceroPrueba(t, fake).EscribirTerceroJuridico(
		context.Background(), entrada, 0, helper.WriteVerbCreate,
	)
	exigirValidacion(t, err)
	sinEscrituras(t, fake)
}

func TestEscribirTerceroJuridicoRechazaRazonSocialLarga(t *testing.T) {
	fake := nuevoCrudFake()
	entrada := entradaJuridicaPrueba()
	entrada.DatosBasicos.RazonSocial = strings.Repeat("A", 256)

	_, err := servicioTerceroPrueba(t, fake).EscribirTerceroJuridico(
		context.Background(), entrada, 0, helper.WriteVerbCreate,
	)
	exigirValidacion(t, err)
	sinEscrituras(t, fake)
}

func TestEscribirTerceroJuridicoRechazaNumeroConComa(t *testing.T) {
	fake := nuevoCrudFake()
	entrada := entradaJuridicaPrueba()
	entrada.DatosIdentificacion.Numero = "900,123456"

	_, err := servicioTerceroPrueba(t, fake).EscribirTerceroJuridico(
		context.Background(), entrada, 0, helper.WriteVerbCreate,
	)
	exigirValidacion(t, err)
	sinEscrituras(t, fake)
}

func TestEscribirTerceroJuridicoRechazaLugarInexistente(t *testing.T) {
	fake := nuevoCrudFake()
	entrada := entradaJuridicaPrueba()
	lugar := int64(9999)
	entrada.DatosBasicos.LugarOrigen = &lugar

	_, err := servicioTerceroPrueba(t, fake).EscribirTerceroJuridico(
		context.Background(), entrada, 0, helper.WriteVerbCreate,
	)
	exigirValidacion(t, err)
	sinEscrituras(t, fake)
}

func TestEscribirTerceroJuridicoRechazaFechaFutura(t *testing.T) {
	fake := nuevoCrudFake()
	entrada := entradaJuridicaPrueba()
	futuro := time.Now().Add(24 * time.Hour)
	entrada.DatosIdentificacion.FechaExpedicion = &futuro

	_, err := servicioTerceroPrueba(t, fake).EscribirTerceroJuridico(
		context.Background(), entrada, 0, helper.WriteVerbCreate,
	)
	exigirValidacion(t, err)
	sinEscrituras(t, fake)
}

func TestEscribirTerceroJuridicoRechazaTerceroInactivo(t *testing.T) {
	fake := nuevoCrudFake()
	inactivo := terceroJuridicoFake()
	inactivo["Activo"] = false
	fake.terceros[101] = inactivo
	fake.identificaciones[140] = identificacionJuridicaFake()

	_, err := servicioTerceroPrueba(t, fake).EscribirTerceroJuridico(
		context.Background(), entradaJuridicaPrueba(), 101, helper.WriteVerbUpdate,
	)
	if !errors.Is(err, ErrTerceroInactivo) {
		t.Fatalf("se esperaba ErrTerceroInactivo; obtuvo %v", err)
	}
	sinEscrituras(t, fake)
}

func TestEscribirTerceroJuridicoRechazaIdentificacionInactiva(t *testing.T) {
	fake := nuevoCrudFake()
	fake.terceros[101] = terceroJuridicoFake()
	inactiva := identificacionJuridicaFake()
	inactiva["Activo"] = false
	fake.identificaciones[140] = inactiva

	_, err := servicioTerceroPrueba(t, fake).EscribirTerceroJuridico(
		context.Background(), entradaJuridicaPrueba(), 101, helper.WriteVerbUpdate,
	)
	if !errors.Is(err, ErrIdentificacionInactiva) {
		t.Fatalf("se esperaba ErrIdentificacionInactiva; obtuvo %v", err)
	}
	sinEscrituras(t, fake)
}

func TestEscribirTerceroJuridicoPostRechazaIdentificacionInactiva(t *testing.T) {
	fake := nuevoCrudFake()
	fake.terceros[101] = terceroJuridicoFake()
	inactiva := identificacionJuridicaFake()
	inactiva["Activo"] = false
	fake.identificaciones[140] = inactiva

	_, err := servicioTerceroPrueba(t, fake).EscribirTerceroJuridico(
		context.Background(), entradaJuridicaPrueba(), 0, helper.WriteVerbCreate,
	)
	if !errors.Is(err, ErrIdentificacionInactiva) {
		t.Fatalf("se esperaba ErrIdentificacionInactiva; obtuvo %v", err)
	}
	sinEscrituras(t, fake)
}

func TestEscribirTerceroJuridicoParcialSinIdentificacion(t *testing.T) {
	fake := nuevoCrudFake()
	fake.fallarIdentificacion = true

	resultado, err := servicioTerceroPrueba(t, fake).EscribirTerceroJuridico(
		context.Background(), entradaJuridicaPrueba(), 0, helper.WriteVerbCreate,
	)
	if err != nil {
		t.Fatalf("el fallo de la identificación debe reportarse como parcial: %v", err)
	}
	if !resultado.Parcial {
		t.Fatalf("se esperaba resultado parcial: %+v", resultado)
	}
	if resultado.TerceroId != 101 {
		t.Fatalf("el tercero debe quedar materializado: %+v", resultado)
	}
	if resultado.DatosIdentificacionId != 0 {
		t.Fatalf("la identificación no debe tener Id: %+v", resultado)
	}
	for _, entidad := range resultado.Creados {
		if entidad.Entidad == "DatosIdentificacion" {
			t.Fatalf("no debe informar una identificación creada sin Id: %+v", resultado.Creados)
		}
	}
}

func TestEscribirTerceroJuridicoReutilizaConservaRazonSocial(t *testing.T) {
	fake := nuevoCrudFake()
	fake.terceros[101] = terceroJuridicoFake()
	fake.identificaciones[140] = identificacionJuridicaFake()

	entrada := entradaJuridicaPrueba()
	entrada.DatosBasicos.RazonSocial = "OTRA RAZON SOCIAL S.A.S."

	resultado, err := servicioTerceroPrueba(t, fake).EscribirTerceroJuridico(
		context.Background(), entrada, 0, helper.WriteVerbCreate,
	)
	if err != nil {
		t.Fatalf("EscribirTerceroJuridico no debe fallar: %v", err)
	}
	if resultado.Data.Data["NombreCompleto"] != "ACME S.A.S." {
		t.Fatalf("el POST de reutilización debe conservar NombreCompleto: %+v", resultado.Data.Data)
	}
	if put := buscarCaptura(fake.capturas, http.MethodPut, "/v1/tercero/101"); put != nil {
		t.Fatalf("no debe reescribir el tercero en un POST de reutilización: %+v", put)
	}
}
