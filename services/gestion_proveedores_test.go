package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	beego "github.com/beego/beego/v2/server/web"
)

// fijarHostProveedores apunta GestionProveedoresCrudHost a la URL dada.
func fijarHostProveedores(t *testing.T, host string) {
	t.Helper()
	if err := beego.AppConfig.Set(claveHostProveedores, host); err != nil {
		t.Fatalf("no se pudo fijar %s: %v", claveHostProveedores, err)
	}
}

// 3.2 / ejemplos GetOne vs GetAll: verbo, ruta y target esperados.
func TestProveedoresGetOneVsGetAll(t *testing.T) {
	var rutas []string
	var metodos []string
	// La forma del cuerpo depende de la ruta: el recurso individual responde un
	// objeto; el listado responde un arreglo.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rutas = append(rutas, r.URL.Path)
		metodos = append(metodos, r.Method)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if strings.HasSuffix(r.URL.Path, "/proveedor/15") {
			_, _ = w.Write([]byte(`{"id":15,"razon":"ACME"}`))
		} else {
			_, _ = w.Write([]byte(`[{"id":15,"razon":"ACME"}]`))
		}
	}))
	defer srv.Close()
	fijarHostProveedores(t, srv.URL+"/v1")

	// Ejemplo GetOne: recurso individual por endpoint.
	var proveedor map[string]any
	ok, status, err := GetOneProveedores(context.Background(), "proveedor/15", &proveedor)
	if !ok || status != http.StatusOK || err != nil {
		t.Fatalf("GetOne: ok=%v status=%d err=%v", ok, status, err)
	}

	// Ejemplo GetAll: listado.
	var lista []map[string]any
	ok, status, err = GetAllProveedores(context.Background(), "proveedor", nil, &lista)
	if !ok || status != http.StatusOK || err != nil {
		t.Fatalf("GetAll: ok=%v status=%d err=%v", ok, status, err)
	}

	if len(rutas) != 2 || rutas[0] != "/v1/proveedor/15" || rutas[1] != "/v1/proveedor" {
		t.Errorf("rutas = %v, se esperaba [/v1/proveedor/15 /v1/proveedor]", rutas)
	}
	if metodos[0] != http.MethodGet || metodos[1] != http.MethodGet {
		t.Errorf("métodos = %v, se esperaban GET/GET", metodos)
	}
	if proveedor["id"].(float64) != 15 {
		t.Errorf("GetOne target = %v, se esperaba id 15", proveedor)
	}
}

// 4.1 Aislamiento: cada service llega solo a su servidor.
func TestAislamientoPorService(t *testing.T) {
	var golpeTerceros, golpeProveedores int
	srvT := servidor(t, http.StatusOK, `{"id":1}`, func(*http.Request, []byte) { golpeTerceros++ })
	defer srvT.Close()
	srvP := servidor(t, http.StatusOK, `{"id":2}`, func(*http.Request, []byte) { golpeProveedores++ })
	defer srvP.Close()
	fijarHost(t, srvT.URL)
	fijarHostProveedores(t, srvP.URL)

	var tt, pp map[string]any
	if _, _, err := GetOne(context.Background(), "tercero/1", &tt); err != nil {
		t.Fatalf("terceros err: %v", err)
	}
	if _, _, err := GetOneProveedores(context.Background(), "proveedor/2", &pp); err != nil {
		t.Fatalf("proveedores err: %v", err)
	}
	if golpeTerceros != 1 || golpeProveedores != 1 {
		t.Errorf("golpes terceros=%d proveedores=%d, se esperaba 1 y 1", golpeTerceros, golpeProveedores)
	}
}

// 4.2 GetAll conserva prefijo, parámetros previos, repetidos y especiales.
func TestProveedoresGetAllFiltros(t *testing.T) {
	var vistoPath, vistoQuery string
	srv := servidor(t, http.StatusOK, `[]`, func(r *http.Request, _ []byte) {
		vistoPath = r.URL.Path
		vistoQuery = r.URL.RawQuery
	})
	defer srv.Close()
	fijarHostProveedores(t, srv.URL+"/v1")

	q := url.Values{}
	q.Set("query", "razon:ACME S.A.")
	q.Add("sortby", "nombre")
	q.Add("sortby", "fecha")

	var lista []map[string]any
	ok, _, err := GetAllProveedores(context.Background(), "proveedor?activo=true", q, &lista)
	if !ok || err != nil {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if vistoPath != "/v1/proveedor" {
		t.Errorf("path = %q, se esperaba /v1/proveedor", vistoPath)
	}
	valores, _ := url.ParseQuery(vistoQuery)
	if valores.Get("activo") != "true" {
		t.Errorf("se perdió la consulta previa: %q", vistoQuery)
	}
	if valores.Get("query") != "razon:ACME S.A." {
		t.Errorf("query = %q", valores.Get("query"))
	}
	if v := valores["sortby"]; len(v) != 2 || v[0] != "nombre" || v[1] != "fecha" {
		t.Errorf("sortby = %v", v)
	}
	if !strings.Contains(vistoQuery, "%") {
		t.Errorf("el query crudo %q debería escapar caracteres especiales", vistoQuery)
	}
}

// 4.3 Post/Put envían el payload una sola vez; payload no serializable → 0 req.
func TestProveedoresPostPut(t *testing.T) {
	t.Run("post una sola vez y target distinto", func(t *testing.T) {
		var peticiones int
		var recibido map[string]any
		srv := servidor(t, http.StatusCreated, `{"id":7}`, func(r *http.Request, datos []byte) {
			peticiones++
			_ = json.Unmarshal(datos, &recibido)
		})
		defer srv.Close()
		fijarHostProveedores(t, srv.URL)

		enviado := map[string]any{"razon": "ACME"}
		var creado map[string]any
		ok, status, err := PostProveedores(context.Background(), "proveedor", enviado, &creado)
		if !ok || status != http.StatusCreated || err != nil {
			t.Fatalf("ok=%v status=%d err=%v", ok, status, err)
		}
		if peticiones != 1 {
			t.Errorf("peticiones = %d, se esperaba 1", peticiones)
		}
		if recibido["razon"] != "ACME" || creado["id"].(float64) != 7 {
			t.Errorf("payload=%v target=%v", recibido, creado)
		}
	})

	t.Run("put payload no serializable → 0 peticiones", func(t *testing.T) {
		var peticiones int
		srv := servidor(t, http.StatusOK, `{}`, func(*http.Request, []byte) { peticiones++ })
		defer srv.Close()
		fijarHostProveedores(t, srv.URL)

		var target map[string]any
		ok, status, err := PutProveedores(context.Background(), "proveedor/1", make(chan int), &target)
		if ok || status != http.StatusInternalServerError || err == nil {
			t.Fatalf("ok=%v status=%d err=%v, se esperaba false/500/err", ok, status, err)
		}
		if peticiones != 0 {
			t.Errorf("peticiones = %d, se esperaban 0", peticiones)
		}
	})
}

// 5.1 Mismo tratamiento de resultados que terceros.
func TestProveedoresTratamientoResultados(t *testing.T) {
	t.Run("204 sin contenido", func(t *testing.T) {
		srv := servidor(t, http.StatusNoContent, "", nil)
		defer srv.Close()
		fijarHostProveedores(t, srv.URL)
		var target map[string]any
		ok, status, err := PutProveedores(context.Background(), "proveedor/1", map[string]any{"x": 1}, &target)
		if !ok || status != http.StatusNoContent || err != nil {
			t.Fatalf("ok=%v status=%d err=%v, se esperaba true/204/nil", ok, status, err)
		}
	})

	t.Run("rechazo HTTP conserva status", func(t *testing.T) {
		srv := servidor(t, http.StatusBadRequest, `{"Message":"malo"}`, nil)
		defer srv.Close()
		fijarHostProveedores(t, srv.URL)
		var target map[string]any
		ok, status, err := GetOneProveedores(context.Background(), "proveedor/1", &target)
		if ok || status != http.StatusBadRequest || err == nil {
			t.Fatalf("ok=%v status=%d err=%v, se esperaba false/400/err", ok, status, err)
		}
	})

	t.Run("JSON incompatible", func(t *testing.T) {
		srv := servidor(t, http.StatusOK, `{"id": `, nil) // truncado
		defer srv.Close()
		fijarHostProveedores(t, srv.URL)
		var target map[string]any
		ok, status, err := GetOneProveedores(context.Background(), "proveedor/1", &target)
		if ok || status != http.StatusOK || err == nil {
			t.Fatalf("ok=%v status=%d err=%v, se esperaba false/200/err", ok, status, err)
		}
	})

	t.Run("fallo de conexión", func(t *testing.T) {
		srv := servidor(t, http.StatusOK, `{}`, nil)
		host := srv.URL
		srv.Close()
		fijarHostProveedores(t, host)
		var target map[string]any
		ok, status, err := GetOneProveedores(context.Background(), "proveedor/1", &target)
		if ok || status != http.StatusInternalServerError || err == nil {
			t.Fatalf("ok=%v status=%d err=%v, se esperaba false/500/err", ok, status, err)
		}
	})
}

// 5.2 El log identifica el service de proveedores y omite el host del error.
func TestProveedoresLogYErrorPublico(t *testing.T) {
	leerLogs := capturarLogs(t)
	srv := servidor(t, http.StatusOK, `{}`, nil)
	host := srv.URL
	addr := srv.Listener.Addr().String()
	srv.Close()
	fijarHostProveedores(t, host)

	var target map[string]any
	_, _, err := GetOneProveedores(context.Background(), "proveedor/1", &target)
	if err == nil {
		t.Fatal("se esperaba error")
	}
	if strings.Contains(err.Error(), addr) {
		t.Errorf("el error público no debe contener el host %q: %q", addr, err.Error())
	}
	entradas := leerLogs()
	if len(entradas) != 1 {
		t.Fatalf("se esperaba una sola entrada de log, got %d", len(entradas))
	}
	e := entradas[0]
	if e.servicio != servicioProveedores {
		t.Errorf("servicio = %q, se esperaba %q", e.servicio, servicioProveedores)
	}
	if e.host == "" {
		t.Errorf("el log debería identificar el host")
	}
	// El log no debe contener payloads ni valores de filtros.
	if strings.Contains(e.linea, "razon") || strings.Contains(e.linea, "ACME") {
		t.Errorf("el log no debe contener payloads ni filtros: %q", e.linea)
	}
}

// 5.3 Propagación de cancelación en proveedores.
func TestProveedoresContextoCancelado(t *testing.T) {
	srv := servidor(t, http.StatusOK, `{}`, nil)
	defer srv.Close()
	fijarHostProveedores(t, srv.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var target map[string]any
	ok, status, err := GetOneProveedores(ctx, "proveedor/1", &target)
	if ok || status != http.StatusInternalServerError || err == nil {
		t.Fatalf("ok=%v status=%d err=%v, se esperaba false/500/err", ok, status, err)
	}
}
