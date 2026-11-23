package controllers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	beego "github.com/beego/beego/v2/server/web"

	_ "github.com/udistrital/agora_mid/routers"
)

// llamada registra una petición recibida por un servidor de CRUD de prueba.
type llamada struct {
	metodo string
	path   string
	query  string
	cuerpo map[string]any
}

// crudFake es un servidor httptest que responde según el método y la ruta,
// registrando cada llamada. Permite montar un servidor por CRUD.
type crudFake struct {
	srv       *httptest.Server
	llamadas  []llamada
	responder func(r *http.Request) (int, string)
}

func nuevoCRUD(t *testing.T, responder func(r *http.Request) (int, string)) *crudFake {
	t.Helper()
	f := &crudFake{responder: responder}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var cuerpo map[string]any
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&cuerpo)
		}
		f.llamadas = append(f.llamadas, llamada{r.Method, r.URL.Path, r.URL.RawQuery, cuerpo})
		status, body := f.responder(r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if body != "" {
			_, _ = w.Write([]byte(body))
		}
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *crudFake) escrituras() int {
	n := 0
	for _, l := range f.llamadas {
		if l.metodo == http.MethodPost || l.metodo == http.MethodPut {
			n++
		}
	}
	return n
}

// peticion ejecuta un POST /v1/proveedores/natural a través del router de beego.
func peticion(t *testing.T, cuerpo string) (int, map[string]any) {
	t.Helper()
	r, _ := http.NewRequest(http.MethodPost, "/v1/proveedores/natural", strings.NewReader(cuerpo))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	beego.BeeApp.Handlers.ServeHTTP(w, r)
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	return w.Code, resp
}

const cuerpoValido = `{
  "TipoProveedor": 1,
  "DatosBasicos": {"PrimerNombre":"Carlos","SegundoNombre":"Andrés","PrimerApellido":"González","SegundoApellido":"Pérez","FechaNacimiento":"1990-05-15T00:00:00Z"},
  "DatosIdentificacion": {"Numero":"1023456789","TipoDocumentoId":1,"CiudadExpedicion":123,"FechaExpedicion":"2008-06-20T00:00:00Z"}
}`

func fijarHosts(t *testing.T, terceros, proveedores string) {
	t.Helper()
	if err := beego.AppConfig.Set("TercerosCrudHost", terceros); err != nil {
		t.Fatal(err)
	}
	if err := beego.AppConfig.Set("GestionProveedoresCrudHost", proveedores); err != nil {
		t.Fatal(err)
	}
}

// Caso 1: todos existen -> 200, se reutiliza proveedor (0 escrituras en proveedores).
func TestCaso1TodosExisten(t *testing.T) {
	 terceros := nuevoCRUD(t, func(r *http.Request) (int, string) {
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/datos_identificacion"):
			return 200, `[{"Id":88,"Numero":"1023456789","TerceroId":{"Id":55}}]`
		case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/datos_identificacion/"):
			return 200, `{"Id":88,"Numero":"1023456789","TerceroId":{"Id":55}}`
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/tercero/"):
			return 200, `{"Id":55,"PrimerNombre":"Carlos","PrimerApellido":"González"}`
		case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/tercero/"):
			return 200, `{"Id":55,"PrimerNombre":"Carlos","PrimerApellido":"González","FechaNacimiento":"1990-05-15T00:00:00Z"}`
		}
		return 404, `{}`
	})
	proveedores := nuevoCRUD(t, func(r *http.Request) (int, string) {
		return 200, `[{"Id":42,"TerceroId":55,"TipoProveedor":1}]`
	})
	fijarHosts(t, terceros.srv.URL, proveedores.srv.URL)

	code, resp := peticion(t, cuerpoValido)
	if code != http.StatusOK {
		t.Fatalf("code = %d, se esperaba 200", code)
	}
	if proveedores.escrituras() != 0 {
		t.Errorf("proveedores escrituras = %d, se esperaba 0", proveedores.escrituras())
	}
	data := resp["Data"].(map[string]any)
	if int(data["ProveedorId"].(float64)) != 42 {
		t.Errorf("ProveedorId = %v, se esperaba 42", data["ProveedorId"])
	}
	// Nombres conservados del tercero recuperado.
	db := data["DatosBasicos"].(map[string]any)
	if db["PrimerNombre"] != "Carlos" {
		t.Errorf("DatosBasicos = %v", db)
	}
}

// Caso 3: solo falta el proveedor -> se crea una vez, 201.
func TestCaso3SoloFaltaProveedor(t *testing.T) {
	terceros := nuevoCRUD(t, func(r *http.Request) (int, string) {
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/datos_identificacion"):
			return 200, `[{"Id":88,"TerceroId":{"Id":55}}]`
		case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/datos_identificacion/"):
			return 200, `{"Id":88}`
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/tercero/"):
			return 200, `{"Id":55,"PrimerNombre":"Carlos"}`
		case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/tercero/"):
			return 200, `{"Id":55}`
		}
		return 404, `{}`
	})
	proveedores := nuevoCRUD(t, func(r *http.Request) (int, string) {
		if r.Method == http.MethodGet {
			return 200, `[]`
		}
		return 201, `{"Id":77,"TerceroId":55,"TipoProveedor":1}`
	})
	fijarHosts(t, terceros.srv.URL, proveedores.srv.URL)

	code, resp := peticion(t, cuerpoValido)
	if code != http.StatusCreated {
		t.Fatalf("code = %d, se esperaba 201", code)
	}
	if proveedores.escrituras() != 1 {
		t.Errorf("proveedores escrituras = %d, se esperaba 1", proveedores.escrituras())
	}
	data := resp["Data"].(map[string]any)
	if int(data["ProveedorId"].(float64)) != 77 {
		t.Errorf("ProveedorId = %v, se esperaba 77", data["ProveedorId"])
	}
}

// Casos 4/5: búsqueda [] o [{}] -> crear tercero e identificación, 201.
func TestCaso4y5CrearTercero(t *testing.T) {
	for _, respIdent := range []string{`[]`, `[{}]`} {
		t.Run("ident="+respIdent, func(t *testing.T) {
			var postsTercero int
			terceros := nuevoCRUD(t, func(r *http.Request) (int, string) {
				switch {
				case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/datos_identificacion"):
					return 200, respIdent
				case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/tercero"):
					postsTercero++
					return 201, `{"Id":101,"PrimerNombre":"Carlos"}`
				case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/datos_identificacion"):
					return 201, `{"Id":140,"Numero":"1023456789","TerceroId":{"Id":101}}`
				}
				return 404, `{}`
			})
			proveedores := nuevoCRUD(t, func(r *http.Request) (int, string) {
				if r.Method == http.MethodGet {
					return 200, `[]`
				}
				return 201, `{"Id":77}`
			})
			fijarHosts(t, terceros.srv.URL, proveedores.srv.URL)

			code, resp := peticion(t, cuerpoValido)
			if code != http.StatusCreated {
				t.Fatalf("code = %d, se esperaba 201", code)
			}
			if postsTercero != 1 {
				t.Errorf("altas de tercero = %d, se esperaba 1", postsTercero)
			}
			data := resp["Data"].(map[string]any)
			di := data["DatosIdentificacion"].(map[string]any)
			// La identificación quedó asociada al Id del alta del tercero (101).
			rel := di["TerceroId"].(map[string]any)
			if int(rel["Id"].(float64)) != 101 {
				t.Errorf("identificación asociada a tercero %v, se esperaba 101", rel["Id"])
			}
		})
	}
}

// Caso 7a: falla la consulta inicial -> sin escrituras.
func TestCaso7aFallaConsultaInicial(t *testing.T) {
	var escriturasTerceros int
	terceros := nuevoCRUD(t, func(r *http.Request) (int, string) {
		if r.Method == http.MethodPost || r.Method == http.MethodPut {
			escriturasTerceros++
		}
		return 500, `{}` // la consulta inicial falla
	})
	proveedores := nuevoCRUD(t, func(r *http.Request) (int, string) { return 200, `[]` })
	fijarHosts(t, terceros.srv.URL, proveedores.srv.URL)

	code, _ := peticion(t, cuerpoValido)
	if code == http.StatusOK || code == http.StatusCreated {
		t.Fatalf("code = %d, se esperaba error", code)
	}
	if escriturasTerceros != 0 {
		t.Errorf("escrituras en terceros = %d, se esperaban 0", escriturasTerceros)
	}
	if len(proveedores.llamadas) != 0 {
		t.Errorf("no debió llamarse a proveedores, got %d", len(proveedores.llamadas))
	}
}

// Caso 7b: falla una operación tras un alta -> error, sin repetir la escritura.
func TestCaso7bFallaTrasAlta(t *testing.T) {
	var postsTercero int
	terceros := nuevoCRUD(t, func(r *http.Request) (int, string) {
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/datos_identificacion"):
			return 200, `[]`
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/tercero"):
			postsTercero++
			return 201, `{"Id":101}`
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/datos_identificacion"):
			return 500, `{}` // falla el alta de la identificación
		}
		return 404, `{}`
	})
	proveedores := nuevoCRUD(t, func(r *http.Request) (int, string) { return 200, `[]` })
	fijarHosts(t, terceros.srv.URL, proveedores.srv.URL)

	code, resp := peticion(t, cuerpoValido)
	if code == http.StatusCreated || code == http.StatusOK {
		t.Fatalf("code = %d, se esperaba error (no 201)", code)
	}
	if postsTercero != 1 {
		t.Errorf("altas de tercero = %d, se esperaba 1 (sin repetir)", postsTercero)
	}
	// El error público no debe contener el host del CRUD.
	msg, _ := resp["Message"].(string)
	if strings.Contains(msg, terceros.srv.URL) || strings.Contains(msg, "127.0.0.1") {
		t.Errorf("el mensaje público no debe contener el host: %q", msg)
	}
}

// Petición inválida -> 400 sin tocar los CRUD.
func TestPeticionInvalida(t *testing.T) {
	terceros := nuevoCRUD(t, func(r *http.Request) (int, string) { return 200, `[]` })
	proveedores := nuevoCRUD(t, func(r *http.Request) (int, string) { return 200, `[]` })
	fijarHosts(t, terceros.srv.URL, proveedores.srv.URL)

	code, _ := peticion(t, `{"TipoProveedor":0}`)
	if code != http.StatusBadRequest {
		t.Fatalf("code = %d, se esperaba 400", code)
	}
	if len(terceros.llamadas)+len(proveedores.llamadas) != 0 {
		t.Errorf("no debió llamarse a ningún CRUD")
	}
}
