// Package test contiene las pruebas de integración y de extremo a extremo (e2e)
// del middleware de escritura del proveedor. Ejercita el stack completo
// controlador -> servicio -> motor de escritura -> write-resolvers contra CRUD
// simulados, sin red real ni base de datos.
//
// Este archivo centraliza los fixtures y mocks compartidos por las pruebas de
// integración (integration_test.go) y e2e (e2e_test.go) del paquete, de modo que
// ambos reutilicen el mismo CRUD simulado, el mismo cableado de router y la misma
// configuración declarativa real cargada desde config/relaciones_proveedor.yaml.
package test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/udistrital/agora_mid/clients"
	"github.com/udistrital/agora_mid/config"
	"github.com/udistrital/agora_mid/controllers"
	"github.com/udistrital/agora_mid/services"
)

// tipoRegistroNatural es el identificador de tipo de contribuyente que las
// pruebas tratan como persona natural. Debe coincidir con el TipoRegistroId
// enviado en los cuerpos para que el servicio conserve la rama natural.
const tipoRegistroNatural = 1

// PeticionCRUD registra una petición recibida por el CRUD simulado para que las
// pruebas verifiquen método, path, query y cuerpo. No se exponen URLs ni
// cabeceras sensibles: solo lo necesario para las aserciones.
type PeticionCRUD struct {
	Metodo string
	Path   string
	Query  string
	Cuerpo map[string]any
}

// CRUDSimulado imita a los CRUD dueños (terceros y gestión de personas) sobre un
// único servidor HTTP de prueba. Enruta por método y path:
//
//   - POST /v1/<recurso>/      crea y devuelve {"Data":{"Id":n}} con un Id
//     autoincremental por recurso, imitando el 201 del CRUD real.
//   - PATCH /v1/<recurso>/{id} actualiza y devuelve {"Data":{"Id":{id}}}.
//   - GET  /v1/<recurso>       lista por clave natural y devuelve {"Data":[...]}
//     a partir de los registros precargados en existentes.
//
// Captura cada petición en orden para las aserciones del flujo en cascada.
type CRUDSimulado struct {
	server *httptest.Server

	mu         sync.Mutex
	capturas   []PeticionCRUD
	siguiente  map[string]int              // recurso -> próximo Id a asignar
	existentes map[string][]map[string]any // "GET /v1/<recurso>" -> registros
}

// NuevoCRUDSimulado arranca el CRUD simulado. existentes precarga, por clave
// "GET /v1/<recurso>", los registros que una búsqueda por clave natural o por Id
// debe devolver (para los escenarios de idempotencia y de continuación tras
// creación parcial). Puede ser nil.
func NuevoCRUDSimulado(
	t *testing.T,
	existentes map[string][]map[string]any,
) *CRUDSimulado {
	t.Helper()
	if existentes == nil {
		existentes = map[string][]map[string]any{}
	}
	c := &CRUDSimulado{
		siguiente:  map[string]int{},
		existentes: existentes,
	}
	c.server = httptest.NewServer(http.HandlerFunc(c.manejar))
	t.Cleanup(c.server.Close)
	return c
}

// URLs devuelve el mapa de URLs de servicio apuntando todas al CRUD simulado,
// con las tres llaves que el loader de configuración exige resolver.
func (c *CRUDSimulado) URLs() config.ServiceURLs {
	return config.ServiceURLs{
		"terceros":              c.server.URL,
		"parametros":            c.server.URL,
		"gestion_personas_crud": c.server.URL,
	}
}

// Client devuelve un cliente HTTP ligado al servidor de prueba.
func (c *CRUDSimulado) Client() clients.HTTPClient {
	return clients.HTTPClient{Client: c.server.Client()}
}

// Capturas devuelve una copia de las peticiones recibidas, en orden.
func (c *CRUDSimulado) Capturas() []PeticionCRUD {
	c.mu.Lock()
	defer c.mu.Unlock()
	salida := make([]PeticionCRUD, len(c.capturas))
	copy(salida, c.capturas)
	return salida
}

// recurso extrae el nombre del recurso de un path "/v1/<recurso>" o
// "/v1/<recurso>/{id}".
func recursoDePath(path string) string {
	resto := strings.TrimPrefix(strings.TrimPrefix(path, "/v1/"), "/")
	if i := strings.IndexByte(resto, '/'); i >= 0 {
		resto = resto[:i]
	}
	return resto
}

func (c *CRUDSimulado) manejar(w http.ResponseWriter, r *http.Request) {
	var cuerpo map[string]any
	if r.Body != nil {
		if data, _ := io.ReadAll(r.Body); len(data) > 0 {
			_ = json.Unmarshal(data, &cuerpo)
		}
	}

	c.mu.Lock()
	c.capturas = append(c.capturas, PeticionCRUD{
		Metodo: r.Method,
		Path:   r.URL.Path,
		Query:  r.URL.Query().Get("query"),
		Cuerpo: cuerpo,
	})
	claveGet := "GET " + strings.TrimRight(r.URL.Path, "/")
	respuesta := c.responder(r.Method, r.URL.Path, claveGet, cuerpo)
	c.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(w, respuesta)
}

// responder construye la respuesta JSON según el verbo, imitando el contrato de
// los CRUD dueños (ver gestion_personas_crud_swagger.yml): lista en GET, objeto
// con Id materializado en POST/PATCH. Debe llamarse con c.mu tomado.
func (c *CRUDSimulado) responder(
	metodo, path, claveGet string,
	cuerpo map[string]any,
) string {
	switch metodo {
	case http.MethodGet:
		if registros, ok := c.existentes[claveGet]; ok && len(registros) > 0 {
			data, _ := json.Marshal(map[string]any{"Data": registros})
			return string(data)
		}
		return `{"Data":[]}`
	case http.MethodPatch:
		// La actualización conserva el Id del cuerpo.
		if id, ok := cuerpo["Id"]; ok {
			data, _ := json.Marshal(map[string]any{"Data": map[string]any{"Id": id}})
			return string(data)
		}
		return `{"Data":{}}`
	default: // POST
		res := recursoDePath(path)
		c.siguiente[res]++
		id := idBase(res) + c.siguiente[res]
		data, _ := json.Marshal(map[string]any{"Data": map[string]any{"Id": id}})
		return string(data)
	}
}

// idBase asigna un rango de Id distinto por recurso para que las aserciones
// distingan de qué CRUD provino cada identificador materializado.
func idBase(res string) int {
	switch res {
	case "proveedor":
		return 100
	case "tercero":
		return 200
	case "datos_identificacion":
		return 300
	case "proveedor_natural":
		return 400
	case "proveedor_juridico":
		return 500
	case "contacto":
		return 600
	case "cuenta_bancaria":
		return 700
	default:
		return 900
	}
}

// routerProveedor arma el router de /v1/proveedor sobre el servicio de escritura
// real cableado contra el CRUD simulado, replicando el despacho por método del
// main (GET -> lectura, POST/PATCH -> escritura). Carga la configuración
// declarativa real desde ../config para que la prueba ejercite el YAML de
// producción, no una copia.
func routerProveedor(t *testing.T, crud *CRUDSimulado) http.Handler {
	t.Helper()
	return routerProveedorConBlacklist(t, crud, nil)
}

// routerProveedorConBlacklist es como routerProveedor pero inyecta una lista
// negra global en el servicio de lectura, de modo que las pruebas puedan
// ejercitar el recorte y el rechazo por lista negra sobre la configuración real.
// Una blacklist nil equivale a "sin supresión".
func routerProveedorConBlacklist(
	t *testing.T,
	crud *CRUDSimulado,
	blacklist config.GlobalBlacklist,
) http.Handler {
	t.Helper()

	urls := crud.URLs()
	relaciones, err := config.LoadRelaciones("../config", urls)
	if err != nil {
		t.Fatalf("cargando configuración de relaciones real: %v", err)
	}
	cfg, ok := relaciones["proveedor"]
	if !ok {
		t.Fatal("no existe la configuración de relaciones del servicio proveedor")
	}

	escritura := services.NewProveedorWriteService(crud.Client(), urls, cfg, tipoRegistroNatural)
	lectura := services.NewProveedorService(crud.Client(), urls, cfg, tipoRegistroNatural, nil, blacklist)
	handler := controllers.NewProveedorHandlerCompleto(lectura, escritura)

	// Replica el enrutado del main: /v1/proveedor/consulta expone la lectura por
	// cuerpo JSON (handler.Consultar, solo POST), y /v1/proveedor despacha por
	// método (GET -> lectura, POST/PATCH -> escritura).
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/proveedor/consulta" {
			handler.Consultar(w, r)
			return
		}
		switch r.Method {
		case http.MethodPost, http.MethodPatch:
			handler.Escribir(w, r)
		default:
			handler.Leer(w, r)
		}
	})
}

// cuerpoProveedorNatural devuelve un cuerpo anidado válido para CREAR el subárbol
// de un proveedor natural: proveedor raíz + tercero (PK compartida) con sus datos
// de identificación (FK) y la FK del tipo de contribuyente (catálogo por Id,
// aplanada), la rama natural (PK compartida) y un contacto (FK). Solo usa campos
// escribibles declarados en el YAML real.
//
// TipoContribuyente es una relación de solo lectura (sin bloque write): en
// escritura se envía su FK plana TipoContribuyenteId, nunca el objeto anidado.
func cuerpoProveedorNatural() map[string]any {
	return map[string]any{
		"TerceroId":             float64(25),
		"TipoRegistroId":        float64(tipoRegistroNatural),
		"DescripcionPortafolio": "Servicios de consultoría",
		"Tercero": map[string]any{
			"NombreCompleto":      "Ana Pérez",
			"TipoContribuyenteId": float64(2),
			"DatosIdentificacion": []any{
				map[string]any{"Numero": "123456", "TipoDocumentoId": float64(1)},
			},
		},
		"ProveedorNatural": map[string]any{"PerfilDeclarado": float64(3)},
		"Contacto": []any{
			map[string]any{
				"Finalidad":       "notificaciones",
				"TipoCanal":       "correo",
				"ValorEspecifico": "ana@correo.co",
			},
		},
	}
}

// cuerpoProveedorNaturalUpdate devuelve un cuerpo de RECONCILIACIÓN (PATCH) del
// subárbol. A diferencia de la creación, identifica cada entidad existente por su
// Id y omite los campos inmutables (Numero y TipoDocumentoId en
// datos_identificacion), que el motor rechaza en una actualización. provID es el
// Id del proveedor existente; terceroID el Id propio del tercero (FK ascendente,
// distinto del proveedor); diID el de sus datos de identificación (0 si aún no
// existe y debe crearse). El contacto se deja sin Id para forzar su creación
// cuando falta.
func cuerpoProveedorNaturalUpdate(provID, terceroID, diID int) map[string]any {
	datos := map[string]any{
		// Campo escribible no inmutable: válido en una actualización.
		"CiudadExpedicion": float64(11001),
	}
	if diID > 0 {
		datos["Id"] = float64(diID)
	} else {
		// Si aún no existe, la creación sí requiere su clave natural.
		datos["Numero"] = "123456"
		datos["TipoDocumentoId"] = float64(1)
	}

	return map[string]any{
		"Id":                    float64(provID),
		"TipoRegistroId":        float64(tipoRegistroNatural),
		"DescripcionPortafolio": "Servicios de consultoría (actualizado)",
		"Tercero": map[string]any{
			"Id":             float64(terceroID),
			"NombreCompleto": "Ana Pérez",
			"DatosIdentificacion": []any{
				datos,
			},
		},
		"ProveedorNatural": map[string]any{"Id": float64(provID), "PerfilDeclarado": float64(4)},
		"Contacto": []any{
			map[string]any{
				"Finalidad":       "notificaciones",
				"TipoCanal":       "correo",
				"ValorEspecifico": "ana@correo.co",
			},
		},
	}
}

