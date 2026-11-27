package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/udistrital/agora_mid/models"
)

type catalogosJuridicaFake struct {
	*catalogosCaracterizacionFake
	tiposDocumento       map[string]int64
	tiposDocumentoID     map[int64]bool
	tiposDocumentoPorID  map[int64]string
	tercerosPorDocumento map[string]map[string]any
	identificaciones     map[int64]map[string]any
}

func (f *catalogosJuridicaFake) TipoDocumentoExiste(
	_ context.Context,
	id int64,
) (bool, error) {
	return f.tiposDocumentoID[id], nil
}

func (f *catalogosJuridicaFake) TipoDocumentoPorCodigo(
	_ context.Context,
	codigo string,
) (int64, bool, bool, error) {
	id, ok := f.tiposDocumento[codigo]
	return id, ok, false, nil
}

func (f *catalogosJuridicaFake) TipoDocumentoPorID(
	_ context.Context,
	id int64,
) (string, bool, error) {
	codigo, ok := f.tiposDocumentoPorID[id]
	return codigo, ok, nil
}

func (f *catalogosJuridicaFake) TerceroPorDocumento(
	_ context.Context,
	tipoDocumentoID int64,
	numero string,
) (map[string]any, bool, error) {
	registro, ok := f.tercerosPorDocumento[fmt.Sprintf("%d|%s", tipoDocumentoID, numero)]
	return registro, ok, nil
}

func (f *catalogosJuridicaFake) IdentificacionActiva(
	_ context.Context,
	terceroID int64,
) (map[string]any, bool, error) {
	registro, ok := f.identificaciones[terceroID]
	return registro, ok, nil
}

type parametrosJuridicaFake struct {
	*parametrosCaracterizacionFake
	porCodigo map[string]map[string]int64
}

func (f *parametrosJuridicaFake) ResolverParametroPorCodigo(
	_ context.Context,
	codigo string,
	tipo string,
) (int64, bool, bool, error) {
	porTipo, ok := f.porCodigo[tipo]
	if !ok {
		return 0, false, false, nil
	}
	id, ok := porTipo[codigo]
	return id, ok, false, nil
}

func servicioJuridicaPrueba(
	t *testing.T,
	fake *fakeCaracterizacion,
	catalogos *catalogosJuridicaFake,
	parametros *parametrosJuridicaFake,
) CaracterizacionJuridicaService {
	t.Helper()
	crudDoble(t, fake.ServeHTTP, fake.ServeHTTP)
	if catalogos == nil {
		catalogos = catalogosJuridicaBase()
	}
	if parametros == nil {
		parametros = parametrosJuridicaBase()
	}
	return NewCaracterizacionJuridicaService(
		catalogos,
		catalogos,
		parametros,
		2,
		1,
		2,
		1,
		"CIIU",
		"MED_COMUNICACION",
		"REG_CONT",
		"TIPO_CONFORMACION",
		"PROCEDENCIA",
		"Grupo_5",
		"PROVEEDOR_JURIDICO",
	)
}

func catalogosJuridicaBase() *catalogosJuridicaFake {
	base := &catalogosCaracterizacionFake{
		infoInactiva: map[int64]bool{},
		codigos: map[string]int64{
			"NACIONAL":             10,
			"PROCEDENCIA":          10,
			"CARGO_REPRESENTANTE":  20,
			"DIRECCIÓN":            30,
			"SITIO_WEB":            31,
			"ACTIVIDAD_PRINCIPAL":  32,
			"TIPO_CONF_CUENTA":     33,
			"CALIDAD_GRAN_CONTRIB": 34,
			"PROFESIONAL":          24,
		},
		ambiguos: map[string]bool{},
		grupos: map[int64]string{
			10: "PROCEDENCIA",
			20: "PROVEEDOR_JURIDICO",
			30: "Grupo_10",
			31: "PROVEEDOR_JURIDICO",
			32: "PROVEEDOR_JURIDICO",
			33: "PROVEEDOR_JURIDICO",
			34: "PROVEEDOR_JURIDICO",
			24: "Grupo_5",
		},
		detalles:    map[int64]InfoComplementariaResuelta{},
		lugaresMalo: map[int64]bool{},
	}
	return &catalogosJuridicaFake{
		catalogosCaracterizacionFake: base,
		tiposDocumento:               map[string]int64{"CC": 3, "PAS": 9},
		tiposDocumentoID:             map[int64]bool{3: true, 9: true},
		tiposDocumentoPorID:          map[int64]string{3: "CC", 9: "PAS"},
		tercerosPorDocumento: map[string]map[string]any{
			"3|1012345678": {
				"Id":                  55,
				"Activo":              true,
				"TipoContribuyenteId": map[string]any{"Id": 1},
				"NombreCompleto":      "Carlos Rodriguez",
			},
		},
		identificaciones: map[int64]map[string]any{
			55: {"Id": 900, "TerceroId": 55, "TipoDocumentoId": 3, "Numero": "1012345678", "Activo": true},
			66: {"Id": 901, "TerceroId": 66, "TipoDocumentoId": 3, "Numero": "1099887002", "Activo": true},
		},
	}
}

func parametrosJuridicaBase() *parametrosJuridicaFake {
	return &parametrosJuridicaFake{
		parametrosCaracterizacionFake: &parametrosCaracterizacionFake{
			noExiste: map[int64]bool{},
			tipos: map[int64]string{
				500:  "REG_CONT",
				501:  "TIPO_CONFORMACION",
				7778: "CIIU",
				1:    "MED_COMUNICACION",
				2:    "MED_COMUNICACION",
			},
		},
		porCodigo: map[string]map[string]int64{
			"REG_CONT": {
				"COMUN_ORDINARIO": 500,
			},
			"TIPO_CONFORMACION": {
				"SAS": 501,
			},
		},
	}
}

func fakeJuridicaBase() *fakeCaracterizacion {
	fake := nuevoFakeCaracterizacion()
	fake.terceros[101] = map[string]any{
		"Id":                  101,
		"Activo":              true,
		"TipoContribuyenteId": map[string]any{"Id": 2},
		"NombreCompleto":      "ACME S.A.S.",
	}
	fake.terceros[55] = map[string]any{
		"Id":                  55,
		"Activo":              true,
		"TipoContribuyenteId": map[string]any{"Id": 1},
		"NombreCompleto":      "Carlos Rodriguez",
	}
	fake.proveedores[88] = map[string]any{
		"Id":            88,
		"TerceroId":     55,
		"TipoProveedor": 1,
		"Activo":        true,
	}
	fake.naturales[88] = map[string]any{
		"Id":     88,
		"Activo": true,
	}
	return fake
}

func entradaCaracterizacionJuridicaPrueba() models.RegistroCaracterizacionJuridica {
	descripcion := "Servicios de consultoría"
	consentimiento := true
	return models.RegistroCaracterizacionJuridica{
		TerceroId: 101,
		Representante: &models.RepresentanteInput{
			CodigoTipoDocumento: ptrString("CC"),
			Numero:              ptrString("1012345678"),
			CodigoCargo:         ptrString("PROFESIONAL"),
		},
		InfoComplementariaTercero: []models.InfoComplementariaTerceroInput{
			{
				CodigoInfoComplementaria:      ptrString("TIPO_CONF_CUENTA"),
				CodigoGrupoInfoComplementaria: ptrString("PROVEEDOR_JURIDICO"),
				Dato:                          json.RawMessage(`"INDIVIDUAL_PRINCIPAL"`),
			},
			{
				CodigoInfoComplementaria:      ptrString("CALIDAD_GRAN_CONTRIB"),
				CodigoGrupoInfoComplementaria: ptrString("PROVEEDOR_JURIDICO"),
				Dato:                          json.RawMessage(`"NO"`),
			},
			{
				CodigoInfoComplementaria:      ptrString("SITIO_WEB"),
				CodigoGrupoInfoComplementaria: ptrString("PROVEEDOR_JURIDICO"),
				Dato:                          json.RawMessage(`"https://acme.example"`),
			},
			{
				CodigoInfoComplementaria:      ptrString("ACTIVIDAD_PRINCIPAL"),
				CodigoGrupoInfoComplementaria: ptrString("PROVEEDOR_JURIDICO"),
				Dato:                          json.RawMessage(`7778`),
			},
			{
				CodigoInfoComplementaria:      ptrString("DIRECCIÓN"),
				CodigoGrupoInfoComplementaria: ptrString("Grupo_10"),
				Dato: json.RawMessage(`{
					"PaisId":48,
					"DepartamentoId":260,
					"CiudadId":824,
					"ViaId":128058,
					"DetalleVia":"10",
					"NumeroCruce":"20",
					"PlacaPuerta":"30",
					"Direccion":"CALLE 10 # 20 - 30"
				}`),
			},
		},
		ProveedorJuridico: &models.ProveedorJuridicoInput{
			CodigoProcedencia:         ptrString("NACIONAL"),
			CiudadOrigenId:            ptrInt64(824),
			PaisOrigenId:              ptrInt64(48),
			CodigoTipoConformacion:    ptrString("SAS"),
			CodigoRegimenContributivo: ptrString("COMUN_ORDINARIO"),
			ExclusividadProducto:      ptrBool(false),
			Pyme:                      ptrBool(true),
			RegistroMercantil:         ptrBool(true),
			SujetoRetencion:           ptrBool(true),
			AgenteRetenedor:           ptrBool(false),
			ResponsableIca:            ptrBool(true),
			ResponsableIva:            ptrBool(true),
		},
		Contacto: []models.ContactoProveedorInput{
			{
				TipoCanalContactoId: 1,
				Proposito:           "NOTIFICACIONES",
				ValorEspecifico:     "notificaciones@acme.co",
				NombreContacto:      ptrString("ACME S.A.S."),
			},
			{
				TipoCanalContactoId: 2,
				Proposito:           "COMERCIAL",
				ValorEspecifico:     "3186749920",
				Extension:           ptrString("104"),
			},
			{
				TipoCanalContactoId: 2,
				Proposito:           "ASESOR",
				ValorEspecifico:     "3186749921",
				NombreContacto:      ptrString("Carlos Rodriguez"),
			},
		},
		ActividadEconomicaProveedor: []models.ActividadEconomicaProveedorInput{
			{ActividadEconomicaId: 7778},
		},
		DescripcionPortafolio: &descripcion,
		Consentimiento:        &consentimiento,
	}
}

func TestCaracterizacionJuridicaPostCompleto(t *testing.T) {
	fake := fakeJuridicaBase()
	servicio := servicioJuridicaPrueba(t, fake, nil, nil)

	resultado, err := servicio.Registrar(
		context.Background(), entradaCaracterizacionJuridicaPrueba(), 101, "create",
	)
	if err != nil {
		t.Fatalf("Registrar devolvió error: %v", err)
	}
	if resultado.Parcial {
		t.Fatalf("no se esperaba parcial: %v", resultado.Avisos)
	}
	if resultado.Data.ProveedorId <= 0 {
		t.Fatalf("ProveedorId inválido: %+v", resultado.Data)
	}
	if resultado.Data.ProveedorJuridico == nil {
		t.Fatalf("se esperaba ProveedorJuridico: %+v", resultado.Data)
	}
	if resultado.Data.Representacion == nil || resultado.Data.Representacion.Id <= 0 {
		t.Fatalf("se esperaba Representacion: %+v", resultado.Data)
	}
	if len(resultado.Data.InfoComplementariaTercero) != 6 {
		t.Fatalf("info esperada = 6, obtenida = %d", len(resultado.Data.InfoComplementariaTercero))
	}
	if len(resultado.Data.Contacto) != 3 {
		t.Fatalf("contactos esperados = 3, obtenidos = %d", len(resultado.Data.Contacto))
	}
	if len(resultado.Data.ActividadEconomicaProveedor) != 1 {
		t.Fatalf("actividades esperadas = 1, obtenidas = %d", len(resultado.Data.ActividadEconomicaProveedor))
	}
	if len(resultado.Data.Resultado.Creados) == 0 {
		t.Fatalf("se esperaban entidades creadas: %+v", resultado.Data.Resultado)
	}
	proveedorID := resultado.Data.ProveedorId
	verificarUsuarioModificacion(t, "proveedor jurídico", fake.proveedores[proveedorID], 101)
	verificarUsuarioModificacion(t, "proveedor_juridico", fake.juridicos[proveedorID], 101)
	verificarUsuarioModificacion(t, "representacion", fake.representaciones[resultado.Data.Representacion.Id], 101)
	for _, contacto := range resultado.Data.Contacto {
		verificarUsuarioModificacion(t, "contacto jurídico", fake.contactos[contacto.Id], 101)
	}
	for _, actividad := range resultado.Data.ActividadEconomicaProveedor {
		verificarUsuarioModificacion(t, "actividad jurídica", fake.actividades[actividad.Id], 101)
	}
}

func TestCaracterizacionJuridicaIdempotente(t *testing.T) {
	fake := fakeJuridicaBase()
	servicio := servicioJuridicaPrueba(t, fake, nil, nil)
	in := entradaCaracterizacionJuridicaPrueba()

	primero, err := servicio.Registrar(context.Background(), in, 101, "create")
	if err != nil {
		t.Fatalf("primer Registrar devolvió error: %v", err)
	}
	segundo, err := servicio.Registrar(context.Background(), in, 101, "create")
	if err != nil {
		t.Fatalf("segundo Registrar devolvió error: %v", err)
	}
	if segundo.Parcial {
		t.Fatalf("no se esperaba parcial: %v", segundo.Avisos)
	}
	if segundo.Data.ProveedorId != primero.Data.ProveedorId {
		t.Fatalf("ProveedorId cambió: %d -> %d", primero.Data.ProveedorId, segundo.Data.ProveedorId)
	}
	if len(segundo.Data.Resultado.Creados) != 0 {
		t.Fatalf("la segunda llamada no debe crear entidades: %+v", segundo.Data.Resultado)
	}
}

func TestCaracterizacionJuridicaActualizaAuditoriaExistente(t *testing.T) {
	fake := fakeJuridicaBase()
	servicio := servicioJuridicaPrueba(t, fake, nil, nil)
	in := entradaCaracterizacionJuridicaPrueba()

	primero, err := servicio.Registrar(context.Background(), in, 101, "create")
	if err != nil {
		t.Fatalf("primer Registrar devolvió error: %v", err)
	}
	proveedorID := primero.Data.ProveedorId
	representacionID := primero.Data.Representacion.Id

	fake.proveedores[proveedorID]["UsuarioModificacionId"] = 0
	fake.juridicos[proveedorID]["UsuarioModificacionId"] = 0
	fake.representaciones[representacionID]["UsuarioModificacionId"] = 0
	for _, contacto := range primero.Data.Contacto {
		fake.contactos[contacto.Id]["UsuarioModificacionId"] = 0
	}
	for _, actividad := range primero.Data.ActividadEconomicaProveedor {
		fake.actividades[actividad.Id]["UsuarioModificacionId"] = 0
	}

	segundo, err := servicio.Registrar(context.Background(), in, 101, "create")
	if err != nil {
		t.Fatalf("segundo Registrar devolvió error: %v", err)
	}
	if segundo.Parcial {
		t.Fatalf("no se esperaba parcial: %+v", segundo.Avisos)
	}
	verificarUsuarioModificacion(t, "proveedor auditoría", fake.proveedores[proveedorID], 101)
	verificarUsuarioModificacion(t, "proveedor_juridico auditoría", fake.juridicos[proveedorID], 101)
	verificarUsuarioModificacion(t, "representacion auditoría", fake.representaciones[representacionID], 101)
	for _, contacto := range primero.Data.Contacto {
		verificarUsuarioModificacion(t, "contacto auditoría", fake.contactos[contacto.Id], 101)
	}
	for _, actividad := range primero.Data.ActividadEconomicaProveedor {
		verificarUsuarioModificacion(t, "actividad auditoría", fake.actividades[actividad.Id], 101)
	}
}

func TestCaracterizacionJuridicaRepresentanteSinProveedorNatural(t *testing.T) {
	fake := fakeJuridicaBase()
	delete(fake.proveedores, 88)
	servicio := servicioJuridicaPrueba(t, fake, nil, nil)

	_, err := servicio.Registrar(
		context.Background(), entradaCaracterizacionJuridicaPrueba(), 101, "create",
	)
	if err == nil {
		t.Fatal("se esperaba error por representante sin proveedor natural")
	}
	caracterizacionErr, ok := err.(*CaracterizacionNaturalError)
	if !ok || caracterizacionErr.StatusCode != 409 {
		t.Fatalf("error esperado 409, obtenido %+v", err)
	}
}

func TestCaracterizacionJuridicaProveedorDeOtroTipo(t *testing.T) {
	fake := fakeJuridicaBase()
	fake.proveedores[77] = map[string]any{
		"Id":            77,
		"TerceroId":     101,
		"TipoProveedor": 1,
		"Activo":        true,
	}
	servicio := servicioJuridicaPrueba(t, fake, nil, nil)

	_, err := servicio.Registrar(
		context.Background(), entradaCaracterizacionJuridicaPrueba(), 101, "create",
	)
	if err == nil {
		t.Fatal("se esperaba error por proveedor de otro tipo")
	}
	caracterizacionErr, ok := err.(*CaracterizacionNaturalError)
	if !ok || caracterizacionErr.StatusCode != 409 {
		t.Fatalf("error esperado 409, obtenido %+v", err)
	}
}

func TestCaracterizacionJuridicaPutSinProveedor(t *testing.T) {
	fake := fakeJuridicaBase()
	servicio := servicioJuridicaPrueba(t, fake, nil, nil)

	_, err := servicio.RegistrarPorProveedor(
		context.Background(), entradaCaracterizacionJuridicaPrueba(), 999, "update",
	)
	if err == nil {
		t.Fatal("se esperaba error por proveedor inexistente")
	}
	caracterizacionErr, ok := err.(*CaracterizacionNaturalError)
	if !ok || caracterizacionErr.StatusCode != 404 {
		t.Fatalf("error esperado 404, obtenido %+v", err)
	}
}

func TestCaracterizacionJuridicaDetalleIncompleto(t *testing.T) {
	fake := fakeJuridicaBase()
	servicio := servicioJuridicaPrueba(t, fake, nil, nil)
	in := entradaCaracterizacionJuridicaPrueba()
	in.ProveedorJuridico.Pyme = nil

	_, err := servicio.Registrar(context.Background(), in, 101, "create")
	if err == nil {
		t.Fatal("se esperaba error por ProveedorJuridico incompleto")
	}
	caracterizacionErr, ok := err.(*CaracterizacionNaturalError)
	if !ok || caracterizacionErr.StatusCode != 400 {
		t.Fatalf("error esperado 400, obtenido %+v", err)
	}
}

func TestCaracterizacionJuridicaTipoConformacionInvalido(t *testing.T) {
	fake := fakeJuridicaBase()
	servicio := servicioJuridicaPrueba(t, fake, nil, nil)
	in := entradaCaracterizacionJuridicaPrueba()
	in.ProveedorJuridico.CodigoTipoConformacion = ptrString("NO_EXISTE")

	_, err := servicio.Registrar(context.Background(), in, 101, "create")
	if err == nil {
		t.Fatal("se esperaba error por tipo de conformación inválido")
	}
	caracterizacionErr, ok := err.(*CaracterizacionNaturalError)
	if !ok || caracterizacionErr.StatusCode != 400 {
		t.Fatalf("error esperado 400, obtenido %+v", err)
	}
}

func TestCaracterizacionJuridicaInfoDesconocida(t *testing.T) {
	fake := fakeJuridicaBase()
	servicio := servicioJuridicaPrueba(t, fake, nil, nil)
	in := entradaCaracterizacionJuridicaPrueba()
	in.InfoComplementariaTercero = append(in.InfoComplementariaTercero,
		models.InfoComplementariaTerceroInput{
			CodigoInfoComplementaria:      ptrString("NO_EXISTE"),
			CodigoGrupoInfoComplementaria: ptrString("PROVEEDOR_JURIDICO"),
			Dato:                          json.RawMessage(`"x"`),
		})

	_, err := servicio.Registrar(context.Background(), in, 101, "create")
	if err == nil {
		t.Fatal("se esperaba error por código desconocido")
	}
	caracterizacionErr, ok := err.(*CaracterizacionNaturalError)
	if !ok || caracterizacionErr.StatusCode != 400 {
		t.Fatalf("error esperado 400, obtenido %+v", err)
	}
}

func TestCaracterizacionJuridicaDireccionIncoherente(t *testing.T) {
	fake := fakeJuridicaBase()
	servicio := servicioJuridicaPrueba(t, fake, nil, nil)
	in := entradaCaracterizacionJuridicaPrueba()
	in.ProveedorJuridico.CiudadOrigenId = ptrInt64(999)

	_, err := servicio.Registrar(context.Background(), in, 101, "create")
	if err == nil {
		t.Fatal("se esperaba error por ciudad incoherente")
	}
	caracterizacionErr, ok := err.(*CaracterizacionNaturalError)
	if !ok || caracterizacionErr.StatusCode != 400 {
		t.Fatalf("error esperado 400, obtenido %+v", err)
	}
}

func TestCaracterizacionJuridicaPrincipalFueraDeActividades(t *testing.T) {
	fake := fakeJuridicaBase()
	servicio := servicioJuridicaPrueba(t, fake, nil, nil)
	in := entradaCaracterizacionJuridicaPrueba()
	in.ActividadEconomicaProveedor = []models.ActividadEconomicaProveedorInput{
		{ActividadEconomicaId: 7777},
	}

	_, err := servicio.Registrar(context.Background(), in, 101, "create")
	if err == nil {
		t.Fatal("se esperaba error por actividad principal fuera de la lista")
	}
	caracterizacionErr, ok := err.(*CaracterizacionNaturalError)
	if !ok || caracterizacionErr.StatusCode != 400 {
		t.Fatalf("error esperado 400, obtenido %+v", err)
	}
}

func TestCaracterizacionJuridicaContactoRepetido(t *testing.T) {
	fake := fakeJuridicaBase()
	servicio := servicioJuridicaPrueba(t, fake, nil, nil)
	in := entradaCaracterizacionJuridicaPrueba()
	in.Contacto = append(in.Contacto, models.ContactoProveedorInput{
		TipoCanalContactoId: 1,
		Proposito:           "NOTIFICACIONES",
		ValorEspecifico:     "otro@acme.co",
	})

	_, err := servicio.Registrar(context.Background(), in, 101, "create")
	if err == nil {
		t.Fatal("se esperaba error por contacto repetido")
	}
	caracterizacionErr, ok := err.(*CaracterizacionNaturalError)
	if !ok || caracterizacionErr.StatusCode != 400 {
		t.Fatalf("error esperado 400, obtenido %+v", err)
	}
}

func TestCaracterizacionJuridicaInfoValoresInvalidos(t *testing.T) {
	casos := []struct {
		nombre string
		codigo string
		dato   json.RawMessage
	}{
		{"sitio web inválido", "SITIO_WEB", json.RawMessage(`"no-es-url"`)},
		{"modalidad de cuenta inválida", "TIPO_CONF_CUENTA", json.RawMessage(`"OTRO"`)},
		{"calidad DIAN inválida", "CALIDAD_GRAN_CONTRIB", json.RawMessage(`"OTRO"`)},
		{"dirección incompleta", "DIRECCIÓN", json.RawMessage(`{"CiudadId":824}`)},
		{"principal no CIIU", "ACTIVIDAD_PRINCIPAL", json.RawMessage(`8419`)},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			fake := fakeJuridicaBase()
			servicio := servicioJuridicaPrueba(t, fake, nil, nil)
			in := entradaCaracterizacionJuridicaPrueba()
			for i := range in.InfoComplementariaTercero {
				if in.InfoComplementariaTercero[i].CodigoInfoComplementaria != nil &&
					*in.InfoComplementariaTercero[i].CodigoInfoComplementaria == caso.codigo {
					in.InfoComplementariaTercero[i].Dato = caso.dato
				}
			}
			_, err := servicio.Registrar(context.Background(), in, 101, "create")
			if err == nil {
				t.Fatal("se esperaba error de validación")
			}
			caracterizacionErr, ok := err.(*CaracterizacionNaturalError)
			if !ok || caracterizacionErr.StatusCode != 400 {
				t.Fatalf("error esperado 400, obtenido %+v", err)
			}
		})
	}
}

func TestCaracterizacionJuridicaCargoInvalido(t *testing.T) {
	fake := fakeJuridicaBase()
	servicio := servicioJuridicaPrueba(t, fake, nil, nil)
	in := entradaCaracterizacionJuridicaPrueba()
	in.Representante.CodigoCargo = ptrString("NO_EXISTE")

	_, err := servicio.Registrar(context.Background(), in, 101, "create")
	if err == nil {
		t.Fatal("se esperaba error por cargo inválido")
	}
	caracterizacionErr, ok := err.(*CaracterizacionNaturalError)
	if !ok || caracterizacionErr.StatusCode != 400 {
		t.Fatalf("error esperado 400, obtenido %+v", err)
	}
}

func TestCaracterizacionJuridicaTipoDocumentoInvalido(t *testing.T) {
	fake := fakeJuridicaBase()
	servicio := servicioJuridicaPrueba(t, fake, nil, nil)
	in := entradaCaracterizacionJuridicaPrueba()
	in.Representante.CodigoTipoDocumento = ptrString("XX")

	_, err := servicio.Registrar(context.Background(), in, 101, "create")
	if err == nil {
		t.Fatal("se esperaba error por tipo de documento inválido")
	}
	caracterizacionErr, ok := err.(*CaracterizacionNaturalError)
	if !ok || caracterizacionErr.StatusCode != 400 {
		t.Fatalf("error esperado 400, obtenido %+v", err)
	}
}

func TestCaracterizacionJuridicaRepresentanteNoNatural(t *testing.T) {
	fake := fakeJuridicaBase()
	servicio := servicioJuridicaPrueba(t, fake, nil, nil)
	in := entradaCaracterizacionJuridicaPrueba()
	in.Representante = &models.RepresentanteInput{
		RepresentanteId: ptrInt64(101),
		CargoId:         ptrInt64(24),
	}

	_, err := servicio.Registrar(context.Background(), in, 101, "create")
	if err == nil {
		t.Fatal("se esperaba error por representante no natural")
	}
	caracterizacionErr, ok := err.(*CaracterizacionNaturalError)
	if !ok || caracterizacionErr.StatusCode != 409 {
		t.Fatalf("error esperado 409, obtenido %+v", err)
	}
}

func TestCaracterizacionJuridicaDescripcionLarga(t *testing.T) {
	fake := fakeJuridicaBase()
	servicio := servicioJuridicaPrueba(t, fake, nil, nil)
	in := entradaCaracterizacionJuridicaPrueba()
	texto := strings.Repeat("a", 2001)
	in.DescripcionPortafolio = &texto

	_, err := servicio.Registrar(context.Background(), in, 101, "create")
	if err == nil {
		t.Fatal("se esperaba error por descripción larga")
	}
	caracterizacionErr, ok := err.(*CaracterizacionNaturalError)
	if !ok || caracterizacionErr.StatusCode != 400 {
		t.Fatalf("error esperado 400, obtenido %+v", err)
	}
}

func TestCaracterizacionJuridicaPutTerceroIdIncoherente(t *testing.T) {
	fake := fakeJuridicaBase()
	servicio := servicioJuridicaPrueba(t, fake, nil, nil)
	primero, err := servicio.Registrar(
		context.Background(), entradaCaracterizacionJuridicaPrueba(), 101, "create",
	)
	if err != nil {
		t.Fatalf("Registrar devolvió error: %v", err)
	}
	in := entradaCaracterizacionJuridicaPrueba()
	in.TerceroId = 102

	_, err = servicio.RegistrarPorProveedor(
		context.Background(), in, primero.Data.ProveedorId, "update",
	)
	if err == nil {
		t.Fatal("se esperaba error por TerceroId incoherente")
	}
	caracterizacionErr, ok := err.(*CaracterizacionNaturalError)
	if !ok || caracterizacionErr.StatusCode != 400 {
		t.Fatalf("error esperado 400, obtenido %+v", err)
	}
}

func TestCaracterizacionJuridicaDetalleInactivo(t *testing.T) {
	fake := fakeJuridicaBase()
	servicio := servicioJuridicaPrueba(t, fake, nil, nil)
	primero, err := servicio.Registrar(
		context.Background(), entradaCaracterizacionJuridicaPrueba(), 101, "create",
	)
	if err != nil {
		t.Fatalf("Registrar devolvió error: %v", err)
	}
	proveedorID := primero.Data.ProveedorId
	registro := fake.juridicos[proveedorID]
	registro["Activo"] = false
	fake.juridicos[proveedorID] = registro

	resultado, err := servicio.Registrar(
		context.Background(), entradaCaracterizacionJuridicaPrueba(), 101, "create",
	)
	if err != nil {
		t.Fatalf("Registrar no debe abortar: %v", err)
	}
	if !resultado.Parcial {
		t.Fatalf("se esperaba 206 por detalle inactivo: %+v", resultado)
	}
	if resultado.Data.ProveedorJuridico != nil {
		t.Fatalf("no debía materializarse proveedor_juridico: %+v", resultado.Data)
	}
}

func TestCaracterizacionJuridicaFalloParcial(t *testing.T) {
	fake := fakeJuridicaBase()
	fake.fallar = map[string]int{"POST /v1/proveedor_juridico": 500}
	servicio := servicioJuridicaPrueba(t, fake, nil, nil)

	resultado, err := servicio.Registrar(
		context.Background(), entradaCaracterizacionJuridicaPrueba(), 101, "create",
	)
	if err != nil {
		t.Fatalf("Registrar no debe abortar por una sección: %v", err)
	}
	if !resultado.Parcial {
		t.Fatalf("se esperaba respuesta parcial: %+v", resultado)
	}
	if resultado.Data.ProveedorId <= 0 {
		t.Fatalf("se esperaba proveedor materializado: %+v", resultado.Data)
	}
	if resultado.Data.ProveedorJuridico != nil {
		t.Fatalf("proveedor_juridico no debía materializarse: %+v", resultado.Data.ProveedorJuridico)
	}
	if resultado.Data.Representacion == nil {
		t.Fatalf("las secciones posteriores debían continuar: %+v", resultado.Data)
	}
	if len(resultado.Avisos) == 0 {
		t.Fatalf("se esperaba MetaData.Errors")
	}
}

func TestCaracterizacionJuridicaRepresentantePorIdEnriqueceRespuesta(t *testing.T) {
	fake := fakeJuridicaBase()
	fake.terceros[66] = map[string]any{
		"Id":                  66,
		"Activo":              true,
		"TipoContribuyenteId": map[string]any{"Id": 1},
		"NombreCompleto":      "Ana Perez",
	}
	fake.proveedores[99] = map[string]any{
		"Id":            99,
		"TerceroId":     66,
		"TipoProveedor": 1,
		"Activo":        true,
	}
	fake.naturales[99] = map[string]any{"Id": 99, "Activo": true}
	servicio := servicioJuridicaPrueba(t, fake, nil, nil)
	in := entradaCaracterizacionJuridicaPrueba()
	in.Representante = &models.RepresentanteInput{
		RepresentanteId: ptrInt64(66),
		CargoId:         ptrInt64(24),
	}

	resultado, err := servicio.Registrar(context.Background(), in, 101, "create")
	if err != nil {
		t.Fatalf("Registrar devolvió error: %v", err)
	}
	representante := resultado.Data.Representante
	if representante["TipoDocumentoId"] != int64(3) {
		t.Fatalf("TipoDocumentoId no enriquecido: %+v", representante)
	}
	if representante["Numero"] != "1099887002" {
		t.Fatalf("Numero no enriquecido: %+v", representante)
	}
	if representante["CodigoTipoDocumento"] != "CC" {
		t.Fatalf("CodigoTipoDocumento no enriquecido: %+v", representante)
	}
}

func TestCaracterizacionJuridicaCambioRepresentante(t *testing.T) {
	fake := fakeJuridicaBase()
	servicio := servicioJuridicaPrueba(t, fake, nil, nil)
	in := entradaCaracterizacionJuridicaPrueba()

	primero, err := servicio.Registrar(context.Background(), in, 101, "create")
	if err != nil {
		t.Fatalf("primer Registrar devolvió error: %v", err)
	}

	fake.terceros[66] = map[string]any{
		"Id":                  66,
		"Activo":              true,
		"TipoContribuyenteId": map[string]any{"Id": 1},
		"NombreCompleto":      "Ana Perez",
	}
	fake.proveedores[99] = map[string]any{
		"Id":            99,
		"TerceroId":     66,
		"TipoProveedor": 1,
		"Activo":        true,
	}
	fake.naturales[99] = map[string]any{"Id": 99, "Activo": true}

	in.Representante = &models.RepresentanteInput{
		RepresentanteId: ptrInt64(66),
		CargoId:         ptrInt64(24),
	}
	segundo, err := servicio.Registrar(context.Background(), in, 101, "create")
	if err != nil {
		t.Fatalf("segundo Registrar devolvió error: %v", err)
	}
	if segundo.Data.Representacion == nil {
		t.Fatalf("se esperaba nueva representación: %+v", segundo.Data)
	}
	if segundo.Data.Representacion.Id == primero.Data.Representacion.Id {
		t.Fatalf("la representación debió cambiar: %d", segundo.Data.Representacion.Id)
	}
	antigua := fake.representaciones[primero.Data.Representacion.Id]
	if activo, _ := antigua["Activo"].(bool); activo {
		t.Fatalf("la representación anterior debía quedar inactiva: %+v", antigua)
	}
}
