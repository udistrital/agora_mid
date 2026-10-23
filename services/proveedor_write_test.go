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
					KeyPropagation: config.KeyPropagationSharedPK,
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

// Creación completa: el servicio materializa el proveedor, su tercero (PK
// compartida), los datos de identificación (FK), la rama natural (PK compartida)
// y los contactos (FK), devolviendo los identificadores materializados.
func TestEscribirProveedorCreacionCompleta(t *testing.T) {
	var capturas []peticionCapturada
	server := crudSimulado(t, map[string]string{
		"POST /v1/proveedor/":            `{"Data":{"Id":10}}`,
		"POST /v1/tercero/":              `{"Data":{"Id":10}}`,
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
	// Tercero por PK compartida hereda el Id de la raíz.
	tercero, _ := res.Data["Tercero"].(map[string]any)
	if tercero == nil || idEntero(tercero["Id"]) != 10 {
		t.Errorf("Tercero (shared_pk) debe heredar Id=10; tercero=%+v", tercero)
	}
	// Datos de identificación materializados bajo el tercero con FK TerceroId=10.
	di, _ := tercero["DatosIdentificacion"].([]any)
	if len(di) != 1 {
		t.Fatalf("se esperaba 1 datos_identificacion; di=%+v", di)
	}
	diM := di[0].(map[string]any)
	if idEntero(diM["Id"]) != 50 || comoTexto(diM["TerceroId"]) != "10" {
		t.Errorf("datos_identificacion debe tener Id=50 y TerceroId=10 (fk); di=%+v", diM)
	}
	// Rama natural por PK compartida.
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

// El inmutable declarado se rechaza con 400 en una actualización, y la regla la
// aplica el servicio a través del motor sin materializar nada.
func TestEscribirProveedorInmutableEnUpdate(t *testing.T) {
	var capturas []peticionCapturada
	server := crudSimulado(t, nil, &capturas)
	defer server.Close()

	svc := servicioWritePrueba(urlsDe(server), clients.HTTPClient{Client: server.Client()})

	// TerceroId es inmutable: enviarlo en un PATCH se rechaza. Se usa un
	// TipoRegistroId natural coherente para aislar el rechazo al inmutable.
	body := map[string]any{
		"Id":             float64(10),
		"TerceroId":      float64(25),
		"TipoRegistroId": float64(tipoRegistroNaturalPrueba),
	}

	_, err := svc.EscribirProveedor(context.Background(), body, helper.WriteVerbUpdate)
	var ee *helper.ErrorEscritura
	if !errors.As(err, &ee) {
		t.Fatalf("se esperaba *helper.ErrorEscritura por inmutable; err=%v", err)
	}
	for _, c := range capturas {
		if c.metodo == http.MethodPost || c.metodo == http.MethodPatch {
			t.Errorf("no debe materializar nada ante un inmutable; capturas=%+v", capturas)
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
