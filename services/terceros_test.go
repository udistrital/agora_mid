package services

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	beego "github.com/beego/beego/v2/server/web"
)

// fijarHost apunta TercerosCrudHost a la URL dada durante la prueba.
func fijarHost(t *testing.T, host string) {
	t.Helper()
	if err := beego.AppConfig.Set(claveHostTerceros, host); err != nil {
		t.Fatalf("no se pudo fijar %s: %v", claveHostTerceros, err)
	}
}

// servidor responde con el status y cuerpo dados; registra la última petición.
func servidor(t *testing.T, status int, cuerpo string, capturar func(*http.Request, []byte)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		datos, _ := io.ReadAll(r.Body)
		if capturar != nil {
			capturar(r, datos)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if cuerpo != "" {
			_, _ = w.Write([]byte(cuerpo))
		}
	}))
}

// 1. GetAll conserva prefijo, consulta previa, valores repetidos y caracteres
// especiales; decodifica la lista en el target declarado.
func TestGetAllDecodificaYConservaQuery(t *testing.T) {
	var vistoPath, vistoQuery string
	srv := servidor(t, http.StatusOK, `[{"id":1},{"id":2}]`, func(r *http.Request, _ []byte) {
		vistoPath = r.URL.Path
		vistoQuery = r.URL.RawQuery
	})
	defer srv.Close()
	fijarHost(t, srv.URL+"/v1")

	q := url.Values{}
	q.Set("query", "nombre:Juan Pérez")
	q.Add("sortby", "nombre")
	q.Add("sortby", "fecha")

	var terceros []map[string]any
	ok, status, err := GetAll(context.Background(), "tercero?activo=true", q, &terceros)
	if !ok || status != http.StatusOK || err != nil {
		t.Fatalf("ok=%v status=%d err=%v, se esperaba true/200/nil", ok, status, err)
	}
	if vistoPath != "/v1/tercero" {
		t.Errorf("path = %q, se esperaba /v1/tercero (prefijo conservado)", vistoPath)
	}
	valores, _ := url.ParseQuery(vistoQuery)
	if valores.Get("activo") != "true" {
		t.Errorf("se perdió la consulta previa: %q", vistoQuery)
	}
	if valores.Get("query") != "nombre:Juan Pérez" {
		t.Errorf("query decodificado = %q", valores.Get("query"))
	}
	if v := valores["sortby"]; len(v) != 2 || v[0] != "nombre" || v[1] != "fecha" {
		t.Errorf("sortby = %v, se esperaban [nombre fecha]", v)
	}
	if !strings.Contains(vistoQuery, "%") {
		t.Errorf("el query crudo %q debería escapar espacios/acentos", vistoQuery)
	}
	if len(terceros) != 2 {
		t.Errorf("target = %v, se esperaban 2 elementos decodificados", terceros)
	}
}

// 2. Post envía el payload una sola vez y decodifica la 201 en un target
// distinto del objeto enviado.
func TestPostDecodificaEnTargetDistinto(t *testing.T) {
	var peticiones int
	var vistoCuerpo map[string]any
	var vistoCT, vistoMethod string
	srv := servidor(t, http.StatusCreated, `{"id":99,"nombre":"ACME"}`, func(r *http.Request, datos []byte) {
		peticiones++
		vistoCT = r.Header.Get("Content-Type")
		vistoMethod = r.Method
		_ = json.Unmarshal(datos, &vistoCuerpo)
	})
	defer srv.Close()
	fijarHost(t, srv.URL)

	enviado := map[string]any{"nombre": "ACME"}
	var creado map[string]any
	ok, status, err := Post(context.Background(), "tercero", enviado, &creado)
	if !ok || status != http.StatusCreated || err != nil {
		t.Fatalf("ok=%v status=%d err=%v, se esperaba true/201/nil", ok, status, err)
	}
	if peticiones != 1 {
		t.Errorf("peticiones = %d, se esperaba 1", peticiones)
	}
	if vistoMethod != http.MethodPost || vistoCT != "application/json" {
		t.Errorf("método/CT = %s/%s", vistoMethod, vistoCT)
	}
	if vistoCuerpo["nombre"] != "ACME" {
		t.Errorf("payload recibido = %v", vistoCuerpo)
	}
	if creado["id"].(float64) != 99 {
		t.Errorf("target creado = %v, se esperaba id 99", creado)
	}
}

// 3. Put con 204 vacía devuelve (true, 204, nil).
func TestPut204SinContenido(t *testing.T) {
	srv := servidor(t, http.StatusNoContent, "", nil)
	defer srv.Close()
	fijarHost(t, srv.URL)

	var target map[string]any
	ok, status, err := Put(context.Background(), "tercero/15", map[string]any{"nombre": "x"}, &target)
	if !ok || status != http.StatusNoContent || err != nil {
		t.Fatalf("ok=%v status=%d err=%v, se esperaba true/204/nil", ok, status, err)
	}
}

// 4. Un rechazo HTTP conserva el estado remoto y devuelve error público.
func TestRechazoHTTP(t *testing.T) {
	leerLogs := capturarLogs(t)
	srv := servidor(t, http.StatusNotFound, `{"Message":"no existe"}`, nil)
	defer srv.Close()
	fijarHost(t, srv.URL)

	var target map[string]any
	ok, status, err := GetOne(context.Background(), "tercero/999", &target)
	if ok || status != http.StatusNotFound || err == nil {
		t.Fatalf("ok=%v status=%d err=%v, se esperaba false/404/err", ok, status, err)
	}
	if strings.Contains(err.Error(), srv.Listener.Addr().String()) {
		t.Errorf("el error público no debe contener el host: %q", err.Error())
	}
	entradas := leerLogs()
	if len(entradas) != 1 || entradas[0].etapa != "rechazo-http" || entradas[0].servicio != servicioTerceros {
		t.Errorf("se esperaba una entrada de log rechazo-http de terceros, got %+v", entradas)
	}
}

// 5. JSON truncado o incompatible conserva el estado, ok=false y loguea la causa.
func TestDecodificacionFallida(t *testing.T) {
	leerLogs := capturarLogs(t)
	srv := servidor(t, http.StatusOK, `{"id": 1`, nil) // JSON truncado
	defer srv.Close()
	fijarHost(t, srv.URL)

	var target map[string]any
	ok, status, err := GetOne(context.Background(), "tercero/1", &target)
	if ok || status != http.StatusOK || err == nil {
		t.Fatalf("ok=%v status=%d err=%v, se esperaba false/200/err", ok, status, err)
	}
	entradas := leerLogs()
	if len(entradas) != 1 || entradas[0].etapa != "decodificacion" {
		t.Errorf("se esperaba una entrada de log de decodificacion, got %+v", entradas)
	}
}

// 5b. Precondición del target: nil o no puntero ante respuesta exitosa.
func TestTargetNilONoPuntero(t *testing.T) {
	leerLogs := capturarLogs(t)
	srv := servidor(t, http.StatusOK, `{"id":1}`, nil)
	defer srv.Close()
	fijarHost(t, srv.URL)

	// target nil.
	ok, status, err := GetOne(context.Background(), "tercero/1", nil)
	if ok || status != http.StatusOK || err == nil {
		t.Fatalf("target nil: ok=%v status=%d err=%v, se esperaba false/200/err", ok, status, err)
	}

	// target no puntero (valor, no dirección).
	var noPuntero map[string]any
	ok, status, err = GetOne(context.Background(), "tercero/1", noPuntero)
	if ok || status != http.StatusOK || err == nil {
		t.Fatalf("target no puntero: ok=%v status=%d err=%v, se esperaba false/200/err", ok, status, err)
	}
	entradas := leerLogs()
	if len(entradas) != 2 {
		t.Errorf("se esperaban 2 entradas de log (nil y no puntero), got %d", len(entradas))
	}
	for _, e := range entradas {
		if e.etapa != "decodificacion" {
			t.Errorf("etapa = %q, se esperaba decodificacion", e.etapa)
		}
	}
}

// 6. Conexión rechazada: (false, 500, errPublico); host en log, fuera del mensaje.
func TestConexionRechazada(t *testing.T) {
	leerLogs := capturarLogs(t)
	srv := servidor(t, http.StatusOK, `{}`, nil)
	host := srv.URL
	addr := srv.Listener.Addr().String()
	srv.Close() // cerrado antes de la petición
	fijarHost(t, host)

	var target map[string]any
	ok, status, err := GetOne(context.Background(), "tercero/1", &target)
	if ok || status != http.StatusInternalServerError || err == nil {
		t.Fatalf("ok=%v status=%d err=%v, se esperaba false/500/err", ok, status, err)
	}
	if strings.Contains(err.Error(), addr) {
		t.Errorf("el error público no debe contener el host %q: %q", addr, err.Error())
	}
	entradas := leerLogs()
	if len(entradas) != 1 {
		t.Fatalf("se esperaba una sola entrada de log, got %d", len(entradas))
	}
	if entradas[0].host == "" {
		t.Errorf("el log debería identificar el host")
	}
	if entradas[0].servicio != servicioTerceros {
		t.Errorf("servicio = %q, se esperaba %q", entradas[0].servicio, servicioTerceros)
	}
}

// 7. Payload no serializable: cero solicitudes. Y propagación de cancelación.
func TestPayloadNoSerializableYCancelacion(t *testing.T) {
	t.Run("payload no serializable", func(t *testing.T) {
		var peticiones int
		srv := servidor(t, http.StatusCreated, `{}`, func(*http.Request, []byte) { peticiones++ })
		defer srv.Close()
		fijarHost(t, srv.URL)

		var target map[string]any
		ok, status, err := Post(context.Background(), "tercero", make(chan int), &target)
		if ok || status != http.StatusInternalServerError || err == nil {
			t.Fatalf("ok=%v status=%d err=%v, se esperaba false/500/err", ok, status, err)
		}
		if peticiones != 0 {
			t.Errorf("peticiones = %d, se esperaban 0 con payload no serializable", peticiones)
		}
	})

	t.Run("contexto cancelado", func(t *testing.T) {
		srv := servidor(t, http.StatusOK, `{}`, nil)
		defer srv.Close()
		fijarHost(t, srv.URL)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		var target map[string]any
		ok, status, err := GetOne(ctx, "tercero/1", &target)
		if ok || status != http.StatusInternalServerError || err == nil {
			t.Fatalf("ok=%v status=%d err=%v, se esperaba false/500/err", ok, status, err)
		}
	})
}
