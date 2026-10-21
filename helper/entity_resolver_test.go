package helper

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/udistrital/agora_mid/clients"
	"github.com/udistrital/agora_mid/config"
)

// relacionNucleo arma una relación con campos de origen deliberadamente falsos
// (endpoint, related_api, related_field, query_template). Permite comprobar que
// el EntityResolver impone el origen real de la entidad e ignora esos valores.
func relacionNucleo(mappedName, relatedService string, cardinalidad string) config.RelationConfig {
	return config.RelationConfig{
		Name:           mappedName,
		RelatedService: relatedService,
		RelatedAPI:     "NO_DEBE_USARSE",
		RelatedTable:   "t",
		Direction:      config.DirectionLocalToRelated,
		LocalField:     "X",
		RelatedField:   "NO_DEBE_USARSE",
		Cardinality:    cardinalidad,
		Depth:          1,
		MappedName:     mappedName,
		Endpoint:       "/NO_DEBE_USARSE",
		QueryTemplate:  "NO:{ids}",
	}
}

// servidorCaptura devuelve un servidor de prueba que registra el path y el
// parámetro query de la última petición y responde con el payload indicado.
func servidorCaptura(t *testing.T, payload string, capturaPath, capturaQuery *string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if capturaPath != nil {
			*capturaPath = r.URL.Path
		}
		if capturaQuery != nil {
			*capturaQuery = r.URL.Query().Get("query")
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, payload)
	}))
}

// Cada resolver del núcleo fija el endpoint y el campo de consulta reales de su
// entidad, con independencia de lo declarado en la relación.
func TestResolversNucleoCampoDeConsulta(t *testing.T) {
	casos := []struct {
		servicio         string
		mappedName       string
		cardinalidad     string
		keys             []string
		endpointEsperado string
		queryEsperada    string
	}{
		{ServicioTercero, "Tercero", config.CardinalityOne, []string{"5"}, endpointTercero, "Id:5"},
		{ServicioDatosIdentificacion, "DatosIdentificacion", config.CardinalityMany, []string{"5"}, endpointDatosIdentificacion, "TerceroId:5"},
		{ServicioTipoContribuyente, "TipoContribuyente", config.CardinalityOne, []string{"3"}, endpointTipoContribuyente, "Id:3"},
		{ServicioProveedorNatural, "ProveedorNatural", config.CardinalityOne, []string{"7"}, endpointProveedorNatural, "Id:7"},
		{ServicioProveedorJuridico, "ProveedorJuridico", config.CardinalityOne, []string{"7"}, endpointProveedorJuridico, "Id:7"},
		{ServicioPerfil, "Perfil", config.CardinalityOne, []string{"2"}, endpointPerfil, "Id:2"},
		{ServicioContacto, "Contacto", config.CardinalityMany, []string{"7"}, endpointContacto, "ProveedorId:7"},
		{ServicioCuentaBancaria, "CuentaBancaria", config.CardinalityMany, []string{"7"}, endpointCuentaBancaria, "ProveedorId:7"},
	}

	for _, c := range casos {
		t.Run(c.servicio, func(t *testing.T) {
			var path, query string
			server := servidorCaptura(t, `{"Success":true,"Status":"200","Message":"ok","Data":[]}`, &path, &query)
			defer server.Close()

			// Todos los related_api apuntan al mismo servidor para aislar la
			// verificación en el endpoint y el campo de consulta.
			urls := config.ServiceURLs{
				"terceros":              server.URL,
				"parametros":            server.URL,
				"gestion_personas_crud": server.URL,
			}
			inner := NewHTTPResolver(clients.HTTPClient{Client: server.Client()}, urls)
			resolvers := NewResolversNucleo(inner)

			res, ok := resolvers[c.servicio]
			if !ok {
				t.Fatalf("no existe resolver para el servicio %q", c.servicio)
			}

			rel := relacionNucleo(c.mappedName, c.servicio, c.cardinalidad)
			if _, err := res.Resolve(context.Background(), rel, c.keys); err != nil {
				t.Fatalf("error inesperado: %v", err)
			}

			if path != c.endpointEsperado {
				t.Errorf("endpoint inesperado: esperado %q, recibido %q", c.endpointEsperado, path)
			}
			if query != c.queryEsperada {
				t.Errorf("campo de consulta inesperado: esperado %q, recibido %q", c.queryEsperada, query)
			}
		})
	}
}

// El resolver específico anida el payload sin transformar: conserva campos
// arbitrarios, estructuras anidadas y tipos, y no se detecta como genérico.
func TestResolverEspecificoSinTransformacion(t *testing.T) {
	payload := `{"Success":true,"Status":"200","Message":"ok","Data":[{"Id":7,"CampoArbitrario":"valor","Anidado":{"x":1},"Activo":true}]}`
	server := servidorCaptura(t, payload, nil, nil)
	defer server.Close()

	urls := config.ServiceURLs{"terceros": server.URL}
	inner := NewHTTPResolver(clients.HTTPClient{Client: server.Client()}, urls)
	res := NewResolversNucleo(inner)[ServicioTercero]

	rel := relacionNucleo("Tercero", ServicioTercero, config.CardinalityOne)
	registros, err := res.Resolve(context.Background(), rel, []string{"7"})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if len(registros) != 1 {
		t.Fatalf("se esperaba 1 registro; recibidos: %d", len(registros))
	}
	r0 := registros[0]
	if r0["CampoArbitrario"] != "valor" {
		t.Errorf("campo arbitrario no conservado: %+v", r0)
	}
	anidado, ok := r0["Anidado"].(map[string]any)
	if !ok || anidado["x"] == nil {
		t.Errorf("estructura anidada no conservada: %+v", r0)
	}
	if r0["Activo"] != true {
		t.Errorf("campo booleano no conservado: %+v", r0)
	}
	if EsGenerico(res) {
		t.Error("el resolver específico no debe detectarse como genérico")
	}
}

// El resolver específico respeta la cancelación por contexto.
func TestResolverEspecificoContextoCancelado(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	urls := config.ServiceURLs{"gestion_personas_crud": server.URL}
	inner := NewHTTPResolver(clients.HTTPClient{Client: server.Client()}, urls)
	res := NewResolversNucleo(inner)[ServicioContacto]

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	rel := relacionNucleo("Contacto", ServicioContacto, config.CardinalityMany)
	if _, err := res.Resolve(ctx, rel, []string{"7"}); err == nil {
		t.Fatal("se esperaba error por contexto cancelado")
	}
}

// El resolver específico decodifica tanto el envoltorio OAS como una lista JSON
// directa, reutilizando la decodificación del HTTPResolver subyacente.
func TestResolverEspecificoListaDirecta(t *testing.T) {
	server := servidorCaptura(t, `[{"Id":1},{"Id":2}]`, nil, nil)
	defer server.Close()

	urls := config.ServiceURLs{"gestion_personas_crud": server.URL}
	inner := NewHTTPResolver(clients.HTTPClient{Client: server.Client()}, urls)
	res := NewResolversNucleo(inner)[ServicioCuentaBancaria]

	rel := relacionNucleo("CuentaBancaria", ServicioCuentaBancaria, config.CardinalityMany)
	registros, err := res.Resolve(context.Background(), rel, []string{"7"})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if len(registros) != 2 {
		t.Fatalf("se esperaban 2 registros; recibidos: %d", len(registros))
	}
}

// El resolver específico no muta la relación del llamador: los campos de
// anidamiento (MappedName, Cardinality) permanecen intactos tras resolver.
func TestResolverEspecificoNoMutaRelacion(t *testing.T) {
	server := servidorCaptura(t, `{"Data":[]}`, nil, nil)
	defer server.Close()

	urls := config.ServiceURLs{"terceros": server.URL}
	inner := NewHTTPResolver(clients.HTTPClient{Client: server.Client()}, urls)
	res := NewResolversNucleo(inner)[ServicioTercero]

	rel := relacionNucleo("Tercero", ServicioTercero, config.CardinalityOne)
	if _, err := res.Resolve(context.Background(), rel, []string{"1"}); err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if rel.MappedName != "Tercero" || rel.Cardinality != config.CardinalityOne {
		t.Errorf("el resolver no debe mutar la relación del llamador: %+v", rel)
	}
}
