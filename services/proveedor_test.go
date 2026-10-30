package services

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/udistrital/agora_mid/clients"
	"github.com/udistrital/agora_mid/config"
	"github.com/udistrital/agora_mid/helper"
)

// tipoRegistroNaturalPrueba es el identificador de tipo de contribuyente que se
// trata como persona natural en las pruebas.
const tipoRegistroNaturalPrueba = 1

// cfgProveedorPrueba arma una configuración de relaciones del proveedor con las
// entidades del núcleo: tercero (con datos_identificacion y tipo_contribuyente),
// las dos ramas de detalle (natural con perfil, jurídico), contacto y
// cuenta_bancaria.
func cfgProveedorPrueba() config.ServiceConfig {
	return config.ServiceConfig{
		Service:         "proveedor",
		MaxDepth:        4,
		OnRelationError: config.OnRelationErrorNotify,
		MaxRelatedIDs:   100,
		Relations: []config.RelationConfig{
			{
				Name: "Tercero", RelatedService: ServicioTercero, RelatedAPI: "terceros",
				RelatedTable: "tercero", Direction: config.DirectionLocalToRelated,
				LocalField: "TerceroId", RelatedField: "Id", Cardinality: config.CardinalityOne,
				Depth: 3, MappedName: "Tercero", Endpoint: "/v1/tercero", QueryTemplate: "Id:{ids}",
				AllowedFields: []string{"Id", "NombreCompleto", "TipoContribuyenteId"},
				NestedRelations: []config.RelationConfig{
					{
						Name: "DatosIdentificacion", RelatedService: ServicioDatosIdentificacion,
						RelatedAPI: "terceros", RelatedTable: "datos_identificacion",
						Direction: config.DirectionRelatedToLocal, LocalField: "Id", RelatedField: "TerceroId",
						Cardinality: config.CardinalityMany, Depth: 1, MappedName: "DatosIdentificacion",
						Endpoint: "/v1/datos_identificacion", QueryTemplate: "TerceroId:{ids}",
						AllowedFields: []string{"Id", "Numero", "CiudadExpedicion"},
					},
				},
			},
			{
				Name: "ProveedorNatural", RelatedService: ServicioProveedorNatural,
				RelatedAPI: "gestion_personas_crud", RelatedTable: "proveedor_natural",
				Direction: config.DirectionLocalToRelated, LocalField: "Id", RelatedField: "Id",
				Cardinality: config.CardinalityOne, Depth: 2, MappedName: "ProveedorNatural",
				Endpoint: "/v1/proveedor_natural", QueryTemplate: "Id:{ids}",
				AllowedFields: []string{"Id", "PerfilDeclarado"},
			},
			{
				Name: "ProveedorJuridico", RelatedService: ServicioProveedorJuridico,
				RelatedAPI: "gestion_personas_crud", RelatedTable: "proveedor_juridico",
				Direction: config.DirectionLocalToRelated, LocalField: "Id", RelatedField: "Id",
				Cardinality: config.CardinalityOne, Depth: 2, MappedName: "ProveedorJuridico",
				Endpoint: "/v1/proveedor_juridico", QueryTemplate: "Id:{ids}",
				AllowedFields: []string{"Id", "NombreComercial"},
			},
			{
				Name: "Contacto", RelatedService: ServicioContacto,
				RelatedAPI: "gestion_personas_crud", RelatedTable: "contacto",
				Direction: config.DirectionRelatedToLocal, LocalField: "Id", RelatedField: "ProveedorId",
				Cardinality: config.CardinalityMany, Depth: 1, MappedName: "Contacto",
				Endpoint: "/v1/contacto", QueryTemplate: "ProveedorId:{ids}",
				AllowedFields: []string{"Id", "ValorEspecifico"},
			},
			{
				Name: "CuentaBancaria", RelatedService: ServicioCuentaBancaria,
				RelatedAPI: "gestion_personas_crud", RelatedTable: "cuenta_bancaria",
				Direction: config.DirectionRelatedToLocal, LocalField: "Id", RelatedField: "ProveedorId",
				Cardinality: config.CardinalityMany, Depth: 1, MappedName: "CuentaBancaria",
				Endpoint: "/v1/cuenta_bancaria", QueryTemplate: "ProveedorId:{ids}",
				AllowedFields: []string{"Id", "NumeroCuenta", "CiudadAperturaId"},
			},
		},
	}
}

// servidorCRUD enruta las peticiones por path y devuelve payloads fijos. Captura
// los paths vistos para verificar qué entidades se consultaron.
func servidorCRUD(t *testing.T, porPath map[string]string, vistos *[]string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if vistos != nil {
			*vistos = append(*vistos, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if cuerpo, ok := porPath[r.URL.Path]; ok {
			fmt.Fprint(w, cuerpo)
			return
		}
		fmt.Fprint(w, `{"Data":[]}`)
	}))
}

func servicioPrueba(server *httptest.Server) ProveedorService {
	urls := config.ServiceURLs{
		"terceros":              server.URL,
		"parametros":            server.URL,
		"gestion_personas_crud": server.URL,
	}

	return NewProveedorService(
		clients.HTTPClient{Client: server.Client()},
		urls,
		cfgProveedorPrueba(),
		tipoRegistroNaturalPrueba,
		nil,
		nil,
	)
}

// TestListarProveedoresValidaFiltrosAntesDeConsultarCRUD verifica la gramática
// de filtros, la paginación delegada y el total recibido del CRUD.
func TestListarProveedoresValidaFiltrosAntesDeConsultarCRUD(t *testing.T) {
	var consultas []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		consultas = append(consultas, r.URL.Query().Get("query"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"Data":[{"Id":5,"TerceroId":25,"TipoRegistroId":1,"Activo":true}],"Total":9}`)
	}))
	defer server.Close()

	svc := servicioPrueba(server)
	svc.Config.FilterableFields = []string{"Id", "TerceroId", "TipoRegistroId"}

	resultado, err := svc.ListarProveedores(context.Background(),
		[]string{"TerceroId:25", "Id__in:5|6"},
		nil, 2, 1)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if resultado.Total != 9 || len(resultado.Data) != 1 {
		t.Fatalf("resultado inesperado: %+v", resultado)
	}
	if len(consultas) != 1 || consultas[0] != "TerceroId:25,Id__in:5|6" {
		t.Fatalf("filtros no trasladados con AND: %v", consultas)
	}

	for _, filtro := range []string{
		"NombreCompleto:Ana",
		"Tercero.NombreCompleto:Ana",
		"Id__unsupported:5",
		"Id__in:",
		"Id__in:5",
		"Id__gt:5|6",
		"Id__isnull:maybe",
	} {
		antes := len(consultas)
		if _, err := svc.ListarProveedores(context.Background(), []string{filtro}, nil, 2, 1); err == nil {
			t.Errorf("filtro %q debe rechazarse", filtro)
		}
		if len(consultas) != antes {
			t.Errorf("filtro inválido %q consultó el CRUD", filtro)
		}
	}
}

// Un id no positivo devuelve ErrProveedorIDInvalido sin consultar el CRUD.
func TestLeerProveedorIDInvalido(t *testing.T) {
	var vistos []string
	server := servidorCRUD(t, nil, &vistos)
	defer server.Close()

	svc := servicioPrueba(server)
	for _, id := range []int{0, -5} {
		_, err := svc.LeerProveedor(context.Background(), id, nil)
		if !errors.Is(err, ErrProveedorIDInvalido) {
			t.Errorf("id %d: se esperaba ErrProveedorIDInvalido; recibido: %v", id, err)
		}
	}
	if len(vistos) != 0 {
		t.Errorf("no se debe consultar el CRUD con id inválido; paths: %v", vistos)
	}
}

// Un id sin proveedor correspondiente devuelve ErrProveedorNoEncontrado.
func TestLeerProveedorNoEncontrado(t *testing.T) {
	server := servidorCRUD(t, map[string]string{
		"/v1/proveedor/": `{"Data":[{}]}`,
	}, nil)
	defer server.Close()

	svc := servicioPrueba(server)
	_, err := svc.LeerProveedor(context.Background(), 999, nil)
	if !errors.Is(err, ErrProveedorNoEncontrado) {
		t.Errorf("se esperaba ErrProveedorNoEncontrado; recibido: %v", err)
	}
}

// Un id válido expande el tercero y su detalle conservando las llaves locales
// del proveedor base.
func TestLeerProveedorExpandeTerceroYDetalle(t *testing.T) {
	server := servidorCRUD(t, map[string]string{
		"/v1/proveedor/":        `{"Data":[{"Id":5,"TerceroId":25,"TipoRegistroId":1,"Activo":true}]}`,
		"/v1/tercero":           `{"Data":[{"Id":25,"NombreCompleto":"Ana"}]}`,
		"/v1/proveedor_natural": `{"Data":[{"Id":5,"PerfilDeclarado":2}]}`,
	}, nil)
	defer server.Close()

	svc := servicioPrueba(server)
	res, err := svc.LeerProveedor(context.Background(), 5, []string{
		"Id", "Tercero.Id", "Tercero.NombreCompleto", "ProveedorNatural.Id",
	})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	m := res.Data.(map[string]any)
	// Llaves locales conservadas.
	if m["Id"] != 5 || m["TerceroId"] != 25 || m["TipoRegistroId"] != 1 {
		t.Errorf("las llaves locales del proveedor base deben conservarse: %+v", m)
	}
	// Tercero anidado.
	tercero, ok := m["Tercero"].(map[string]any)
	if !ok || tercero["NombreCompleto"] != "Ana" {
		t.Errorf("Tercero no anidado correctamente: %+v", m["Tercero"])
	}
	// Rama natural presente.
	if _, ok := m["ProveedorNatural"].(map[string]any); !ok {
		t.Errorf("ProveedorNatural debe anidarse para un proveedor natural: %+v", m["ProveedorNatural"])
	}
}

// La decisión de rama natural/jurídico ocurre en el servicio: un proveedor
// natural no consulta la rama jurídica y viceversa.
func TestLeerProveedorRamaNaturalJuridico(t *testing.T) {
	casos := []struct {
		nombre         string
		tipoRegistro   int
		pathConsultado string
		pathExcluido   string
		relIncluida    string
		relExcluida    string
	}{
		{"natural", 1, "/v1/proveedor_natural", "/v1/proveedor_juridico", "ProveedorNatural", "ProveedorJuridico"},
		{"juridico", 2, "/v1/proveedor_juridico", "/v1/proveedor_natural", "ProveedorJuridico", "ProveedorNatural"},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			var vistos []string
			server := servidorCRUD(t, map[string]string{
				"/v1/proveedor/":         fmt.Sprintf(`{"Data":[{"Id":5,"TerceroId":25,"TipoRegistroId":%d,"Activo":true}]}`, c.tipoRegistro),
				"/v1/proveedor_natural":  `{"Data":[{"Id":5,"PerfilDeclarado":2}]}`,
				"/v1/proveedor_juridico": `{"Data":[{"Id":5,"NombreComercial":"ACME"}]}`,
			}, &vistos)
			defer server.Close()

			svc := servicioPrueba(server)
			res, err := svc.LeerProveedor(context.Background(), 5, []string{
				"Id", "ProveedorNatural.Id", "ProveedorJuridico.Id",
			})
			if err != nil {
				t.Fatalf("error inesperado: %v", err)
			}

			m := res.Data.(map[string]any)
			if _, ok := m[c.relIncluida]; !ok {
				t.Errorf("se esperaba la rama %q; data: %+v", c.relIncluida, m)
			}
			if _, ok := m[c.relExcluida]; ok {
				t.Errorf("la rama %q no debe expandirse para un proveedor %s", c.relExcluida, c.nombre)
			}
			if contienePath(vistos, c.pathExcluido) {
				t.Errorf("no se debe consultar %q para un proveedor %s; paths: %v", c.pathExcluido, c.nombre, vistos)
			}
		})
	}
}

// Las ciudades (ciudad de apertura de la cuenta bancaria, ciudad de expedición
// de los datos de identificación) se entregan como ID plano: no existe relación
// declarada para expandirlas, por lo que permanecen como valor escalar.
func TestLeerProveedorCiudadesSinExpandir(t *testing.T) {
	server := servidorCRUD(t, map[string]string{
		"/v1/proveedor/":      `{"Data":[{"Id":5,"TerceroId":25,"TipoRegistroId":1,"Activo":true}]}`,
		"/v1/cuenta_bancaria": `{"Data":[{"Id":1,"ProveedorId":5,"NumeroCuenta":"000","CiudadAperturaId":80}]}`,
	}, nil)
	defer server.Close()

	svc := servicioPrueba(server)
	res, err := svc.LeerProveedor(context.Background(), 5, []string{
		"Id", "CuentaBancaria.Id", "CuentaBancaria.NumeroCuenta", "CuentaBancaria.CiudadAperturaId",
	})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	m := res.Data.(map[string]any)
	cuentas, ok := m["CuentaBancaria"].([]any)
	if !ok || len(cuentas) != 1 {
		t.Fatalf("se esperaba 1 cuenta bancaria; recibido: %+v", m["CuentaBancaria"])
	}
	cuenta := cuentas[0].(map[string]any)
	// La ciudad permanece como ID escalar, no como objeto expandido.
	ciudad := cuenta["CiudadAperturaId"]
	if _, esObjeto := ciudad.(map[string]any); esObjeto {
		t.Errorf("la ciudad no debe expandirse como objeto; recibido: %+v", ciudad)
	}
	if ciudad == nil {
		t.Errorf("la ciudad debe conservarse como ID plano; recibido: %+v", cuenta)
	}
}

// Un fields inválido devuelve *helper.ErrorFields sin consultar el CRUD dueño:
// la validación ocurre antes de traer el proveedor base (fallar temprano). Cubre
// rutas vacías, segmentos vacíos, duplicados y rutas no declaradas.
func TestLeerProveedorFieldsInvalido(t *testing.T) {
	casos := []struct {
		nombre string
		fields []string
	}{
		{"ruta vacía", []string{"Id", "  "}},
		{"segmento vacío", []string{"Tercero."}},
		{"segmento intermedio vacío", []string{"Tercero..Id"}},
		{"ruta duplicada", []string{"Id", "Id"}},
		{"ruta no declarada", []string{"Tercero.CampoInexistente"}},
		{"relación inexistente", []string{"NoExiste.Id"}},
		{"campo base intermedio no es relación", []string{"Activo.Algo"}},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			var vistos []string
			server := servidorCRUD(t, map[string]string{
				"/v1/proveedor/": `{"Data":[{"Id":5,"TerceroId":25,"TipoRegistroId":1,"Activo":true}]}`,
			}, &vistos)
			defer server.Close()

			svc := servicioPrueba(server)
			_, err := svc.LeerProveedor(context.Background(), 5, c.fields)

			var errFields *helper.ErrorFields
			if !errors.As(err, &errFields) {
				t.Fatalf("se esperaba *helper.ErrorFields; recibido: %T (%v)", err, err)
			}
			// Fallar temprano: ningún campo inválido debe producir una llamada al
			// CRUD dueño.
			if len(vistos) != 0 {
				t.Errorf("un fields inválido no debe consultar el CRUD; paths: %v", vistos)
			}
		})
	}
}

func contienePath(paths []string, objetivo string) bool {
	for _, p := range paths {
		if strings.EqualFold(p, objetivo) {
			return true
		}
	}
	return false
}
