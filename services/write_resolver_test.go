package services

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/udistrital/agora_mid/clients"
	"github.com/udistrital/agora_mid/config"
)

// peticionCapturada registra una petición recibida por el CRUD simulado para
// poder verificar método, path y cuerpo.
type peticionCapturada struct {
	metodo string
	path   string
	query  string
	cuerpo map[string]any
}

// crudSimulado enruta por método+path y permite responder con un cuerpo fijo por
// path. Captura cada petición para las aserciones.
func crudSimulado(
	t *testing.T,
	respuestas map[string]string,
	capturas *[]peticionCapturada,
) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var cuerpo map[string]any
		if r.Body != nil {
			data, _ := io.ReadAll(r.Body)
			if len(data) > 0 {
				_ = json.Unmarshal(data, &cuerpo)
			}
		}
		*capturas = append(*capturas, peticionCapturada{
			metodo: r.Method,
			path:   r.URL.Path,
			query:  r.URL.Query().Get("query"),
			cuerpo: cuerpo,
		})
		w.Header().Set("Content-Type", "application/json")
		clave := r.Method + " " + r.URL.Path
		if cuerpoResp, ok := respuestas[clave]; ok {
			_, _ = io.WriteString(w, cuerpoResp)
			return
		}
		if cuerpoResp, ok := respuestas[r.URL.Path]; ok {
			_, _ = io.WriteString(w, cuerpoResp)
			return
		}
		_, _ = io.WriteString(w, `{"Data":[]}`)
	}))
}

func urlsDe(server *httptest.Server) config.ServiceURLs {
	return config.ServiceURLs{
		"terceros":              server.URL,
		"parametros":            server.URL,
		"gestion_personas_crud": server.URL,
	}
}

// Sin registro existente, el resolver crea la entidad por POST y devuelve el Id
// materializado; además inyecta el autor de la modificación desde el contexto.
func TestWriteResolverCreaSinInyectarAutor(t *testing.T) {
	var capturas []peticionCapturada
	server := crudSimulado(t, map[string]string{
		"POST /v1/proveedor/": `{"Data":{"Id":77}}`,
	}, &capturas)
	defer server.Close()

	r := NewEntityWriteResolver(
		clients.HTTPClient{Client: server.Client()},
		urlsDe(server),
		"gestion_personas_crud",
		endpointProveedor,
		queryPorTerceroId,
	)

	ctx := context.Background()
	salida, err := r.Materialize(ctx, config.RelationConfig{}, map[string]any{
		"TerceroId": float64(25),
	}, []string{"25"})
	if err != nil {
		t.Fatalf("Materialize no debe fallar: %v", err)
	}
	if idEntero(salida["Id"]) != 77 {
		t.Errorf("se esperaba Id materializado 77; salida=%+v", salida)
	}

	// Debe haber una búsqueda por clave natural (GET) y luego un POST.
	var post *peticionCapturada
	for i := range capturas {
		if capturas[i].metodo == http.MethodPost {
			post = &capturas[i]
		}
	}
	if post == nil {
		t.Fatalf("se esperaba un POST de creación; capturas=%+v", capturas)
	}
	if _, trae := post.cuerpo["AutorModificacionId"]; trae {
		t.Errorf("el POST no debe inyectar AutorModificacionId; cuerpo=%+v", post.cuerpo)
	}
	if _, trae := post.cuerpo["Id"]; trae {
		t.Errorf("una creación no debe enviar Id; cuerpo=%+v", post.cuerpo)
	}
}

// Idempotencia del proveedor por tercero_id: si ya existe un proveedor para el
// tercero, el resolver lo reutiliza (PATCH) en vez de crear otro.
func TestWriteResolverProveedorIdempotentePorTerceroId(t *testing.T) {
	var capturas []peticionCapturada
	server := crudSimulado(t, map[string]string{
		"GET /v1/proveedor":      `{"Data":[{"Id":88,"TerceroId":25}]}`,
		"PATCH /v1/proveedor/88": `{"Data":{"Id":88}}`,
	}, &capturas)
	defer server.Close()

	r := NewEntityWriteResolver(
		clients.HTTPClient{Client: server.Client()},
		urlsDe(server),
		"gestion_personas_crud",
		endpointProveedor,
		queryPorTerceroId,
	)

	reg, err := r.Recuperar(context.Background(), config.RelationConfig{}, map[string]any{
		"TerceroId": float64(25),
	}, []string{"25"})
	if err != nil {
		t.Fatalf("Recuperar no debe fallar: %v", err)
	}
	salida, err := r.Materialize(context.Background(), config.RelationConfig{}, map[string]any{
		"Id":        idEntero(reg["Id"]),
		"TerceroId": float64(25),
	}, []string{"25"})
	if err != nil {
		t.Fatalf("Materialize no debe fallar: %v", err)
	}
	if idEntero(salida["Id"]) != 88 {
		t.Errorf("se esperaba reutilizar el proveedor 88; salida=%+v", salida)
	}

	huboPost := false
	huboPatch := false
	for _, c := range capturas {
		switch c.metodo {
		case http.MethodPost:
			huboPost = true
		case http.MethodPatch:
			huboPatch = true
		}
	}
	if huboPost {
		t.Errorf("no debe crear un segundo proveedor; capturas=%+v", capturas)
	}
	if !huboPatch {
		t.Errorf("debe actualizar el proveedor existente por PATCH; capturas=%+v", capturas)
	}
}

// Idempotencia de datos_identificacion por numero + tipo_documento_id: la
// búsqueda usa ambos valores de la clave natural en el orden declarado.
func TestWriteResolverDatosIdentificacionClaveNatural(t *testing.T) {
	var capturas []peticionCapturada
	server := crudSimulado(t, map[string]string{
		"POST /v1/datos_identificacion/": `{"Data":{"Id":5}}`,
	}, &capturas)
	defer server.Close()

	r := NewEntityWriteResolver(
		clients.HTTPClient{Client: server.Client()},
		urlsDe(server),
		"terceros",
		endpointDatosIdentificacion,
		queryPorNumeroYTipoDocumento,
	)

	_, err := r.Recuperar(context.Background(), config.RelationConfig{}, map[string]any{
		"Numero":          "123",
		"TipoDocumentoId": float64(1),
		"TerceroId":       float64(25),
	}, []string{"123", "1"})
	if err != nil {
		t.Fatalf("Recuperar no debe fallar: %v", err)
	}

	var get *peticionCapturada
	for i := range capturas {
		if capturas[i].metodo == http.MethodGet {
			get = &capturas[i]
		}
	}
	if get == nil {
		t.Fatalf("se esperaba una búsqueda por clave natural; capturas=%+v", capturas)
	}
	if !strings.Contains(get.query, "Numero:123") || !strings.Contains(get.query, "TipoDocumentoId:1") {
		t.Errorf("la búsqueda debe usar numero y tipo_documento_id; query=%q", get.query)
	}
}

// Cuando el payload trae Id (PK compartida o colección con Id), el resolver
// actualiza ese registro por PATCH sin buscar por clave natural.
func TestWriteResolverUpsertPorIdDelCuerpo(t *testing.T) {
	var capturas []peticionCapturada
	server := crudSimulado(t, map[string]string{
		"PATCH /v1/proveedor_natural/5": `{"Data":{"Id":5}}`,
	}, &capturas)
	defer server.Close()

	r := NewEntityWriteResolver(
		clients.HTTPClient{Client: server.Client()},
		urlsDe(server),
		"gestion_personas_crud",
		endpointProveedorNatural,
		nil,
	)

	salida, err := r.Materialize(context.Background(), config.RelationConfig{}, map[string]any{
		"Id":              float64(5),
		"PerfilDeclarado": float64(2),
	}, nil)
	if err != nil {
		t.Fatalf("Materialize no debe fallar: %v", err)
	}
	if idEntero(salida["Id"]) != 5 {
		t.Errorf("debe conservar el Id propagado 5; salida=%+v", salida)
	}
	for _, c := range capturas {
		if c.metodo == http.MethodGet {
			t.Errorf("no debe buscar por clave natural cuando hay Id; capturas=%+v", capturas)
		}
	}
}

// La operación de recuperación (Recuperar) resuelve el Id de un registro
// existente usando SOLO lecturas (GET), sin POST ni PATCH: es la pieza que el
// motor usa para validar antes de escribir. Aquí el proveedor ya existe por su
// clave natural (TerceroId=25); Recuperar debe devolver su Id (88) consultando
// el CRUD por GET y no debe emitir ninguna escritura.
func TestWriteResolverRecuperaExistentePorClaveNaturalSoloGET(t *testing.T) {
	var capturas []peticionCapturada
	server := crudSimulado(t, map[string]string{
		"GET /v1/proveedor": `{"Data":[{"Id":88,"TerceroId":25,"Estado":"A"}]}`,
	}, &capturas)
	defer server.Close()

	r := NewEntityWriteResolver(
		clients.HTTPClient{Client: server.Client()},
		urlsDe(server),
		"gestion_personas_crud",
		endpointProveedor,
		queryPorTerceroId,
	)

	reg, err := r.Recuperar(context.Background(), config.RelationConfig{}, map[string]any{
		"TerceroId": float64(25),
	}, []string{"25"})
	if err != nil {
		t.Fatalf("Recuperar no debe fallar: %v", err)
	}
	if idEntero(reg["Id"]) != 88 {
		t.Errorf("Recuperar debe devolver el registro existente con Id 88; registro=%+v", reg)
	}
	// El registro devuelto trae los campos almacenados, no solo el Id.
	if reg["Estado"] != "A" {
		t.Errorf("Recuperar debe devolver los campos del registro existente; registro=%+v", reg)
	}

	// Solo se permiten lecturas: ninguna escritura (POST/PATCH).
	for _, c := range capturas {
		if c.metodo == http.MethodPost || c.metodo == http.MethodPatch {
			t.Errorf("Recuperar no debe emitir escrituras; captura=%+v", c)
		}
	}
	var huboGet bool
	for _, c := range capturas {
		if c.metodo == http.MethodGet {
			huboGet = true
		}
	}
	if !huboGet {
		t.Errorf("Recuperar debe resolver por GET; capturas=%+v", capturas)
	}
}

// Cuando el payload trae un Id propio, Recuperar lee el registro por su Id (GET)
// y devuelve sus campos, sin emitir escrituras.
func TestWriteResolverRecuperaPorIdDelCuerpoLeeRegistro(t *testing.T) {
	var capturas []peticionCapturada
	server := crudSimulado(t, map[string]string{
		"GET /v1/proveedor/5": `{"Data":{"Id":5,"TerceroId":25,"Estado":"A"}}`,
	}, &capturas)
	defer server.Close()

	r := NewEntityWriteResolver(
		clients.HTTPClient{Client: server.Client()},
		urlsDe(server),
		"gestion_personas_crud",
		endpointProveedor,
		queryPorTerceroId,
	)

	reg, err := r.Recuperar(context.Background(), config.RelationConfig{}, map[string]any{
		"Id":        float64(5),
		"TerceroId": float64(25),
	}, nil)
	if err != nil {
		t.Fatalf("Recuperar no debe fallar: %v", err)
	}
	if idEntero(reg["Id"]) != 5 || reg["Estado"] != "A" {
		t.Errorf("Recuperar debe devolver el registro 5 con sus campos; registro=%+v", reg)
	}
	for _, c := range capturas {
		if c.metodo != http.MethodGet {
			t.Errorf("Recuperar no debe emitir escrituras; captura=%+v", c)
		}
	}
	if len(capturas) != 1 || capturas[0].path != "/v1/proveedor/5" {
		t.Errorf("Recuperar debe leer exactamente el registro 5 por GET; capturas=%+v", capturas)
	}
}

// Sin registro existente, Recuperar devuelve nil sin error y sin escrituras.
func TestWriteResolverRecuperaSinRegistroDevuelveNil(t *testing.T) {
	var capturas []peticionCapturada
	server := crudSimulado(t, map[string]string{
		"GET /v1/proveedor": `{"Data":[]}`,
	}, &capturas)
	defer server.Close()

	r := NewEntityWriteResolver(
		clients.HTTPClient{Client: server.Client()},
		urlsDe(server),
		"gestion_personas_crud",
		endpointProveedor,
		queryPorTerceroId,
	)

	reg, err := r.Recuperar(context.Background(), config.RelationConfig{}, map[string]any{
		"TerceroId": float64(77),
	}, []string{"77"})
	if err != nil {
		t.Fatalf("Recuperar no debe fallar si no hay registro: %v", err)
	}
	if reg != nil {
		t.Errorf("sin registro existente Recuperar debe devolver nil; registro=%+v", reg)
	}
}

// Materialize no vuelve a buscar por clave natural: la decisión de existencia la toma
// el motor con el estado retenido en el pre-pass. Sin Id en el payload crea (POST)
// sin ninguna lectura previa.
func TestWriteResolverMaterializeNoRelee(t *testing.T) {
	var capturas []peticionCapturada
	server := crudSimulado(t, map[string]string{
		"GET /v1/proveedor":   `{"Data":[{"Id":88,"TerceroId":25}]}`,
		"POST /v1/proveedor/": `{"Data":{"Id":77}}`,
	}, &capturas)
	defer server.Close()

	r := NewEntityWriteResolver(
		clients.HTTPClient{Client: server.Client()},
		urlsDe(server),
		"gestion_personas_crud",
		endpointProveedor,
		queryPorTerceroId,
	)

	salida, err := r.Materialize(context.Background(), config.RelationConfig{}, map[string]any{
		"TerceroId": float64(25),
	}, []string{"25"})
	if err != nil {
		t.Fatalf("Materialize no debe fallar: %v", err)
	}
	if idEntero(salida["Id"]) != 77 {
		t.Errorf("sin Id en el payload debe crear (Id 77); salida=%+v", salida)
	}
	for _, c := range capturas {
		if c.metodo == http.MethodGet {
			t.Errorf("Materialize no debe releer el registro retenido; capturas=%+v", capturas)
		}
	}
}

// Eliminar da de baja el registro con el Id dado mediante DELETE sobre el recurso del
// CRUD dueño, sin emitir POST ni PATCH.
func TestWriteResolverEliminaPorDELETE(t *testing.T) {
	var capturas []peticionCapturada
	server := crudSimulado(t, map[string]string{
		"DELETE /v1/proveedor_natural/5": `{}`,
	}, &capturas)
	defer server.Close()

	r := NewEntityWriteResolver(
		clients.HTTPClient{Client: server.Client()},
		urlsDe(server),
		"gestion_personas_crud",
		endpointProveedorNatural,
		nil,
	)

	if err := r.Eliminar(context.Background(), config.RelationConfig{}, float64(5)); err != nil {
		t.Fatalf("Eliminar no debe fallar: %v", err)
	}
	if len(capturas) != 1 || capturas[0].metodo != http.MethodDelete {
		t.Errorf("Eliminar debe emitir exactamente un DELETE; capturas=%+v", capturas)
	}
}
