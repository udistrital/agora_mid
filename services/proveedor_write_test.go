package services

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"testing"

	"github.com/udistrital/agora_mid/clients"
	"github.com/udistrital/agora_mid/config"
	"github.com/udistrital/agora_mid/helper"
)

// cfgProveedorWritePrueba arma una configuración del proveedor con bloques write
// en la raíz y en el subárbol del núcleo, para ejercitar el servicio de
// escritura de extremo a extremo contra CRUD simulados.
func cfgProveedorWritePrueba() config.ServiceConfig {
	return config.ServiceConfig{
		Service:         "proveedor",
		MaxDepth:        4,
		OnRelationError: config.OnRelationErrorNotify,
		MaxRelatedIDs:   100,
		Write: &config.WriteConfig{
			WritableFields:  []string{"TerceroId", "TipoRegistroId", "DescripcionPortafolio"},
			ImmutableFields: []string{"TerceroId"},
			NaturalKey:      []string{"TerceroId"},
			Policy:          config.WriteVerbPolicy{Creatable: true, Updatable: true},
		},
		Relations: []config.RelationConfig{
			{
				Name: "Tercero", RelatedService: ServicioTercero, RelatedAPI: "terceros",
				RelatedTable: "tercero", Direction: config.DirectionLocalToRelated,
				LocalField: "TerceroId", RelatedField: "Id", Cardinality: config.CardinalityOne,
				Depth: 3, MappedName: "Tercero", Endpoint: "/v1/tercero", QueryTemplate: "Id:{ids}",
				Write: &config.WriteConfig{
					WritableFields: []string{"NombreCompleto", "TipoContribuyenteId"},
					KeyPropagation: config.KeyPropagationFKInbound,
					Policy:         config.WriteVerbPolicy{Creatable: true, Updatable: true},
				},
				NestedRelations: []config.RelationConfig{
					{
						Name: "DatosIdentificacion", RelatedService: ServicioDatosIdentificacion,
						RelatedAPI: "terceros", RelatedTable: "datos_identificacion",
						Direction: config.DirectionRelatedToLocal, LocalField: "Id", RelatedField: "TerceroId",
						Cardinality: config.CardinalityMany, Depth: 1, MappedName: "DatosIdentificacion",
						Endpoint: "/v1/datos_identificacion", QueryTemplate: "TerceroId:{ids}",
						Write: &config.WriteConfig{
							WritableFields:  []string{"Numero", "TipoDocumentoId"},
							ImmutableFields: []string{"Numero", "TipoDocumentoId"},
							NaturalKey:      []string{"Numero", "TipoDocumentoId"},
							KeyPropagation:  config.KeyPropagationFK,
							Policy:          config.WriteVerbPolicy{Creatable: true, Updatable: true},
						},
					},
				},
			},
			{
				Name: "ProveedorNatural", RelatedService: ServicioProveedorNatural,
				RelatedAPI: "gestion_personas_crud", RelatedTable: "proveedor_natural",
				Direction: config.DirectionLocalToRelated, LocalField: "Id", RelatedField: "Id",
				Cardinality: config.CardinalityOne, Depth: 2, MappedName: "ProveedorNatural",
				Endpoint: "/v1/proveedor_natural", QueryTemplate: "Id:{ids}",
				Write: &config.WriteConfig{
					WritableFields: []string{"PerfilDeclarado"},
					KeyPropagation: config.KeyPropagationSharedPK,
					Policy:         config.WriteVerbPolicy{Creatable: true, Updatable: true},
				},
			},
			{
				Name: "ProveedorJuridico", RelatedService: ServicioProveedorJuridico,
				RelatedAPI: "gestion_personas_crud", RelatedTable: "proveedor_juridico",
				Direction: config.DirectionLocalToRelated, LocalField: "Id", RelatedField: "Id",
				Cardinality: config.CardinalityOne, Depth: 2, MappedName: "ProveedorJuridico",
				Endpoint: "/v1/proveedor_juridico", QueryTemplate: "Id:{ids}",
				Write: &config.WriteConfig{
					WritableFields: []string{"NombreComercial"},
					KeyPropagation: config.KeyPropagationSharedPK,
					Policy:         config.WriteVerbPolicy{Creatable: true, Updatable: true},
				},
			},
			{
				Name: "Contacto", RelatedService: ServicioContacto,
				RelatedAPI: "gestion_personas_crud", RelatedTable: "contacto",
				Direction: config.DirectionRelatedToLocal, LocalField: "Id", RelatedField: "ProveedorId",
				Cardinality: config.CardinalityMany, Depth: 1, MappedName: "Contacto",
				Endpoint: "/v1/contacto", QueryTemplate: "ProveedorId:{ids}",
				Write: &config.WriteConfig{
					WritableFields: []string{"ValorEspecifico"},
					KeyPropagation: config.KeyPropagationFK,
					Policy:         config.WriteVerbPolicy{Creatable: true, Updatable: true, Deletable: true},
				},
			},
		},
	}
}

func servicioWritePrueba(urls config.ServiceURLs, client clients.HTTPClient) ProveedorWriteService {
	return ProveedorWriteService{
		Config:              cfgProveedorWritePrueba(),
		Registry:            registroWriteNucleo(client, urls),
		TipoRegistroNatural: tipoRegistroNaturalPrueba,
	}
}

func registroWriteNucleo(client clients.HTTPClient, urls config.ServiceURLs) helper.WriteResolverRegistry {
	raiz, hijos := NewWriteResolversNucleo(client, urls)
	return helper.NewWriteResolverRegistry(raiz, hijos)
}

// Creación completa: el servicio materializa el tercero (FK ascendente, dueño de
// su propio Id) antes que el proveedor, propaga ese Id a proveedor.TerceroId,
// materializa los datos de identificación (FK bajo el tercero), la rama natural
// (PK compartida con el proveedor) y los contactos (FK bajo el proveedor),
// devolviendo los identificadores materializados.
func TestEscribirProveedorCreacionCompleta(t *testing.T) {
	var capturas []peticionCapturada
	server := crudSimulado(t, map[string]string{
		"POST /v1/proveedor/":            `{"Data":{"Id":10}}`,
		"POST /v1/tercero/":              `{"Data":{"Id":20}}`,
		"POST /v1/datos_identificacion/": `{"Data":{"Id":50}}`,
		"POST /v1/proveedor_natural/":    `{"Data":{"Id":10}}`,
		"POST /v1/contacto/":             `{"Data":{"Id":70}}`,
	}, &capturas)
	defer server.Close()

	svc := servicioWritePrueba(urlsDe(server), clients.HTTPClient{Client: server.Client()})

	body := map[string]any{
		"TerceroId":      float64(25),
		"TipoRegistroId": float64(tipoRegistroNaturalPrueba),
		"Tercero": map[string]any{
			"NombreCompleto":    "Ana",
			"TipoContribuyente": map[string]any{"Id": float64(2)},
			"DatosIdentificacion": []any{
				map[string]any{"Numero": "123", "TipoDocumentoId": float64(1)},
			},
		},
		"ProveedorNatural": map[string]any{"PerfilDeclarado": float64(3)},
		"Contacto": []any{
			map[string]any{"ValorEspecifico": "a@b.co"},
		},
	}

	ctx := ConContextoAutor(context.Background(), 9)
	res, err := svc.EscribirProveedor(ctx, body, helper.WriteVerbCreate)
	if err != nil {
		t.Fatalf("creación completa no debe fallar: %v", err)
	}

	// El proveedor raíz materializó su Id.
	if idEntero(res.Data["Id"]) != 10 {
		t.Errorf("el proveedor raíz debe materializar Id=10; data=%+v", res.Data)
	}
	// FK ascendente: el tercero es dueño de su propio Id (20), distinto del
	// proveedor, y el proveedor lo referencia por su FK TerceroId=20.
	tercero, _ := res.Data["Tercero"].(map[string]any)
	if tercero == nil || idEntero(tercero["Id"]) != 20 {
		t.Errorf("Tercero (fk_inbound) debe materializar su propio Id=20; tercero=%+v", tercero)
	}
	if comoTexto(res.Data["TerceroId"]) != "20" {
		t.Errorf("el proveedor debe referenciar al tercero por TerceroId=20; data=%+v", res.Data)
	}
	// El tercero se materializó ANTES que el proveedor (orden fk_inbound).
	idxTercero, idxProveedor := -1, -1
	for i, c := range capturas {
		if c.metodo == http.MethodPost && c.path == "/v1/tercero/" && idxTercero < 0 {
			idxTercero = i
		}
		if c.metodo == http.MethodPost && c.path == "/v1/proveedor/" && idxProveedor < 0 {
			idxProveedor = i
		}
	}
	if idxTercero < 0 || idxProveedor < 0 || idxTercero > idxProveedor {
		t.Errorf("el tercero (fk_inbound) debe crearse antes que el proveedor; capturas=%+v", capturas)
	}
	// Datos de identificación materializados bajo el tercero con FK TerceroId=20.
	di, _ := tercero["DatosIdentificacion"].([]any)
	if len(di) != 1 {
		t.Fatalf("se esperaba 1 datos_identificacion; di=%+v", di)
	}
	diM := di[0].(map[string]any)
	if idEntero(diM["Id"]) != 50 || comoTexto(diM["TerceroId"]) != "20" {
		t.Errorf("datos_identificacion debe tener Id=50 y TerceroId=20 (fk del tercero); di=%+v", diM)
	}
	// Rama natural por PK compartida con el proveedor.
	pn, _ := res.Data["ProveedorNatural"].(map[string]any)
	if pn == nil || idEntero(pn["Id"]) != 10 {
		t.Errorf("ProveedorNatural (shared_pk) debe heredar Id=10; pn=%+v", pn)
	}
	// Contacto por FK ProveedorId=10.
	contactos, _ := res.Data["Contacto"].([]any)
	if len(contactos) != 1 {
		t.Fatalf("se esperaba 1 contacto; contactos=%+v", contactos)
	}
	cM := contactos[0].(map[string]any)
	if idEntero(cM["Id"]) != 70 || comoTexto(cM["ProveedorId"]) != "10" {
		t.Errorf("contacto debe tener Id=70 y ProveedorId=10 (fk); c=%+v", cM)
	}

	// La rama jurídica, podada, nunca se consultó.
	for _, c := range capturas {
		if c.path == "/v1/proveedor_juridico/" {
			t.Errorf("no debe materializar la rama jurídica para un proveedor natural; capturas=%+v", capturas)
		}
	}
}

// Regla irreductible de coherencia natural/jurídico: un proveedor natural que
// trae la rama jurídica (o ambas ramas) se rechaza con 400 sin materializar.
func TestEscribirProveedorRamaIncoherente(t *testing.T) {
	var capturas []peticionCapturada
	server := crudSimulado(t, nil, &capturas)
	defer server.Close()

	svc := servicioWritePrueba(urlsDe(server), clients.HTTPClient{Client: server.Client()})

	casos := []struct {
		nombre string
		body   map[string]any
	}{
		{
			nombre: "natural trae rama jurídica",
			body: map[string]any{
				"TerceroId":         float64(25),
				"TipoRegistroId":    float64(tipoRegistroNaturalPrueba),
				"ProveedorJuridico": map[string]any{"NombreComercial": "ACME"},
			},
		},
		{
			nombre: "ambas ramas a la vez",
			body: map[string]any{
				"TerceroId":         float64(25),
				"TipoRegistroId":    float64(tipoRegistroNaturalPrueba),
				"ProveedorNatural":  map[string]any{"PerfilDeclarado": float64(3)},
				"ProveedorJuridico": map[string]any{"NombreComercial": "ACME"},
			},
		},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			capturas = nil
			_, err := svc.EscribirProveedor(context.Background(), c.body, helper.WriteVerbCreate)
			if !errors.Is(err, ErrRamaDetalleIncoherente) {
				t.Fatalf("se esperaba ErrRamaDetalleIncoherente; err=%v", err)
			}
			for _, cap := range capturas {
				if cap.metodo == http.MethodPost || cap.metodo == http.MethodPatch {
					t.Errorf("no debe materializar nada ante rama incoherente; capturas=%+v", capturas)
				}
			}
		})
	}
}

// Reenvío de la FK inmutable derivada (TerceroId) que coincide con la llave
// autoritativa en una actualización: no se rechaza por inmutable, porque
// TerceroId es el campo derivado fk_inbound del proveedor y la inmutabilidad
// protege contra el cambio de valor, no contra el reenvío. El servicio materializa
// el subárbol en lugar de devolver un error de cuerpo inválido.
func TestEscribirProveedorReenvioFKInmutableCoincide(t *testing.T) {
	var capturas []peticionCapturada
	server := crudSimulado(t, map[string]string{
		// El tercero existente conserva su Id 20 al actualizarse por PATCH.
		"PATCH /v1/tercero/20":           `{"Data":{"Id":20}}`,
		"PATCH /v1/proveedor/10":         `{"Data":{"Id":10}}`,
		"PATCH /v1/proveedor_natural/10": `{"Data":{"Id":10}}`,
	}, &capturas)
	defer server.Close()

	svc := servicioWritePrueba(urlsDe(server), clients.HTTPClient{Client: server.Client()})

	// El proveedor y el tercero traen su Id; la FK ascendente reenviada (TerceroId)
	// coincide con el Id autoritativo del tercero (20). Reenvía el Data materializado.
	body := map[string]any{
		"Id":             float64(10),
		"TerceroId":      float64(20),
		"TipoRegistroId": float64(tipoRegistroNaturalPrueba),
		"Tercero": map[string]any{
			"Id":             float64(20),
			"NombreCompleto": "Ana",
		},
		"ProveedorNatural": map[string]any{"Id": float64(10), "PerfilDeclarado": float64(3)},
	}

	res, err := svc.EscribirProveedor(context.Background(), body, helper.WriteVerbUpdate)
	if err != nil {
		t.Fatalf("el reenvío coincidente de una FK inmutable derivada no debe fallar: %v", err)
	}
	if idEntero(res.Data["Id"]) != 10 {
		t.Errorf("debe operar sobre el proveedor 10; data=%+v", res.Data)
	}
	// La FK ascendente coincidente se conserva, sin aviso de normalización.
	for _, w := range res.Warnings {
		if w.Kind == helper.KindFieldNormalization {
			t.Errorf("un reenvío coincidente no debe producir aviso de normalización; warnings=%+v", res.Warnings)
		}
	}
}

// Comportamiento contrario del reenvío coincidente: una FK inmutable derivada
// (TerceroId) que DIFIERE del Id autoritativo del tercero, en un proveedor con Id
// propio (entidad existente), es un conflicto fk_inbound. Se rechaza con
// *helper.ErrorEscritura sin materializar la operación incompatible, de modo que
// aceptar el reenvío coincidente no habilita reasignar silenciosamente la
// entidad a otro tercero.
func TestEscribirProveedorReenvioFKInmutableDifiereExistente(t *testing.T) {
	var capturas []peticionCapturada
	server := crudSimulado(t, map[string]string{
		"PATCH /v1/tercero/20": `{"Data":{"Id":20}}`,
	}, &capturas)
	defer server.Close()

	svc := servicioWritePrueba(urlsDe(server), clients.HTTPClient{Client: server.Client()})

	// El tercero referenciado es Id=20, pero la FK local reenviada difiere (99) en
	// un proveedor existente (Id=10): conflicto, no reasignación.
	body := map[string]any{
		"Id":             float64(10),
		"TerceroId":      float64(99),
		"TipoRegistroId": float64(tipoRegistroNaturalPrueba),
		"Tercero": map[string]any{
			"Id":             float64(20),
			"NombreCompleto": "Ana",
		},
		"ProveedorNatural": map[string]any{"Id": float64(10), "PerfilDeclarado": float64(3)},
	}

	_, err := svc.EscribirProveedor(context.Background(), body, helper.WriteVerbUpdate)
	var ee *helper.ErrorEscritura
	if !errors.As(err, &ee) {
		t.Fatalf("se esperaba *helper.ErrorEscritura por conflicto de FK ascendente en entidad existente; err=%v", err)
	}
	// No debe materializarse el proveedor en conflicto (ni POST ni PATCH).
	for _, c := range capturas {
		if (c.metodo == http.MethodPost || c.metodo == http.MethodPatch) && c.path == "/v1/proveedor/10" {
			t.Errorf("no debe materializarse el proveedor en conflicto; capturas=%+v", capturas)
		}
		if c.metodo == http.MethodPost && c.path == "/v1/proveedor/" {
			t.Errorf("no debe crearse un proveedor ante el conflicto; capturas=%+v", capturas)
		}
	}
}

// comoTexto normaliza a string un valor de llave que puede venir como string o
// como número, para comparar las llaves foráneas propagadas por el motor.
func comoTexto(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case int:
		return strconv.Itoa(t)
	case float64:
		return strconv.Itoa(int(t))
	default:
		return ""
	}
}
