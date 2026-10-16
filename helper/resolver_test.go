package helper

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/udistrital/agora_mid/clients"
	"github.com/udistrital/agora_mid/config"
)

func relacionPrueba(base string) (config.RelationConfig, config.ServiceURLs) {
	rel := config.RelationConfig{
		Name:           "Tercero",
		RelatedService: "tercero",
		RelatedAPI:     "terceros",
		RelatedTable:   "tercero",
		Direction:      config.DirectionLocalToRelated,
		LocalField:     "TerceroId",
		RelatedField:   "Id",
		Cardinality:    config.CardinalityOne,
		Depth:          1,
		MappedName:     "Tercero",
		Endpoint:       "/v1/tercero",
		QueryTemplate:  "Id:{ids}",
	}
	urls := config.ServiceURLs{"terceros": base}
	return rel, urls
}

// Respuesta enveloped OAS estándar.
func TestHTTPResolverEnvelope(t *testing.T) {
	var queryVista string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queryVista = r.URL.Query().Get("query")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"Success":true,"Status":"200","Message":"ok","Data":[{"Id":10,"NombreCompleto":"Ana"},{"Id":11,"NombreCompleto":"Luis"}]}`)
	}))
	defer server.Close()

	rel, urls := relacionPrueba(server.URL)
	res := NewHTTPResolver(clients.HTTPClient{Client: server.Client()}, urls)

	registros, err := res.Resolve(context.Background(), rel, []string{"10", "11"})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if len(registros) != 2 {
		t.Fatalf("se esperaban 2 registros; recibidos: %d", len(registros))
	}
	if registros[0]["NombreCompleto"] != "Ana" {
		t.Errorf("payload inesperado: %+v", registros[0])
	}
	if queryVista != "Id:10,11" {
		t.Errorf("query codificada inesperada: %q", queryVista)
	}
}

// Lista JSON directa (compatibilidad).
func TestHTTPResolverListaDirecta(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"Id":10},{"Id":11},{"Id":12}]`)
	}))
	defer server.Close()

	rel, urls := relacionPrueba(server.URL)
	res := NewHTTPResolver(clients.HTTPClient{Client: server.Client()}, urls)

	registros, err := res.Resolve(context.Background(), rel, []string{"10"})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if len(registros) != 3 {
		t.Fatalf("se esperaban 3 registros; recibidos: %d", len(registros))
	}
}

// related_api sin URL base configurada es un error de configuración.
func TestHTTPResolverSinURL(t *testing.T) {
	rel, _ := relacionPrueba("")
	res := NewHTTPResolver(clients.HTTPClient{}, config.ServiceURLs{})

	_, err := res.Resolve(context.Background(), rel, []string{"10"})
	if err == nil {
		t.Fatal("se esperaba error por falta de URL base")
	}
	if !strings.Contains(err.Error(), "related_api") {
		t.Errorf("el error debe identificar related_api; mensaje: %v", err)
	}
}

// Cancelación por contexto.
func TestHTTPResolverContextoCancelado(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	rel, urls := relacionPrueba(server.URL)
	res := NewHTTPResolver(clients.HTTPClient{Client: server.Client()}, urls)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := res.Resolve(ctx, rel, []string{"10"})
	if err == nil {
		t.Fatal("se esperaba error por contexto cancelado")
	}
}

// Timeout por contexto.
func TestHTTPResolverTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	}))
	defer server.Close()

	rel, urls := relacionPrueba(server.URL)
	res := NewHTTPResolver(clients.HTTPClient{Client: server.Client()}, urls)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := res.Resolve(ctx, rel, []string{"10"})
	if err == nil {
		t.Fatal("se esperaba error por timeout")
	}
}

// Respuesta inválida produce error de decodificación.
func TestHTTPResolverRespuestaInvalida(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"Success":true,"Status":"200"}`) // sin Data
	}))
	defer server.Close()

	rel, urls := relacionPrueba(server.URL)
	res := NewHTTPResolver(clients.HTTPClient{Client: server.Client()}, urls)

	_, err := res.Resolve(context.Background(), rel, []string{"10"})
	if err == nil {
		t.Fatal("se esperaba error por respuesta sin lista Data")
	}
}

// El resolver genérico entrega Data sin transformar y se detecta como
// genérico para emitir el aviso generic_resolver.
func TestGenericResolverSinTransformacion(t *testing.T) {
	payload := `{"Success":true,"Status":"200","Message":"ok","Data":[{"Id":99,"CampoArbitrario":"valor","Anidado":{"x":1}}]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, payload)
	}))
	defer server.Close()

	rel, urls := relacionPrueba(server.URL)
	generico := NewGenericResolver(NewHTTPResolver(clients.HTTPClient{Client: server.Client()}, urls))

	if !EsGenerico(generico) {
		t.Fatal("GenericResolver debe detectarse como genérico")
	}

	registros, err := generico.Resolve(context.Background(), rel, []string{"99"})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if len(registros) != 1 {
		t.Fatalf("se esperaba 1 registro; recibidos: %d", len(registros))
	}
	// Payload sin transformación: se conservan campos arbitrarios y anidados.
	r0 := registros[0]
	if r0["CampoArbitrario"] != "valor" {
		t.Errorf("campo arbitrario no conservado: %+v", r0)
	}
	anidado, ok := r0["Anidado"].(map[string]any)
	if !ok || anidado["x"] == nil {
		t.Errorf("estructura anidada no conservada: %+v", r0)
	}

	// Aviso generic_resolver bien formado y sin datos sensibles.
	aviso := AvisoGenericResolver("Tercero", rel)
	if aviso.Kind != KindGenericResolver {
		t.Errorf("Kind inesperado: %q", aviso.Kind)
	}
	if aviso.Path != "Tercero" || aviso.Service != "tercero" {
		t.Errorf("aviso con campos inesperados: %+v", aviso)
	}
	if strings.Contains(aviso.Message, server.URL) {
		t.Errorf("el aviso no debe exponer URLs internas: %q", aviso.Message)
	}
}

// Un resolver no genérico no se detecta como genérico.
func TestHTTPResolverNoEsGenerico(t *testing.T) {
	_, urls := relacionPrueba("http://x.test")
	res := NewHTTPResolver(clients.HTTPClient{}, urls)
	if EsGenerico(res) {
		t.Error("HTTPResolver no debe detectarse como genérico")
	}
}
