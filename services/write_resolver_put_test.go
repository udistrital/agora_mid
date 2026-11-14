package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/udistrital/agora_mid/clients"
	"github.com/udistrital/agora_mid/config"
)

// Una actualización se envía como PUT con el registro completo: se lee el registro
// existente (GET), se mezclan los valores recibidos y se envía el resultado. El PUT
// del CRUD sobrescribe las columnas que recibe, así que un PUT parcial anularía los
// campos omitidos.
func TestActualizarEnviaPUTConRegistroCompletoLeidoDelCRUD(t *testing.T) {
	var capturas []peticionCapturada
	server := crudSimulado(t, map[string]string{
		"GET /v1/contacto/7": `{"Data":{"Id":7,"NombreContacto":"Ana","ValorEspecifico":"ana@example.com","Proposito":"Notificaciones","ProveedorId":{"Id":1,"TerceroId":101}}}`,
		"PUT /v1/contacto/7": `{"Data":{"Id":7}}`,
	}, &capturas)
	defer server.Close()

	r := NewEntityWriteResolver(
		clients.HTTPClient{Client: server.Client()},
		urlsDe(server),
		"gestion_personas_crud",
		endpointContacto,
		nil,
	)
	salida, err := r.Materialize(context.Background(), config.RelationConfig{}, map[string]any{
		"Id":              float64(7),
		"ValorEspecifico": "nuevo@example.com",
	}, nil)
	if err != nil {
		t.Fatalf("Materialize no debe fallar: %v", err)
	}
	if salida["Id"] != 7 {
		t.Errorf("el Id materializado debe ser 7; salida=%+v", salida)
	}

	for _, c := range capturas {
		if c.metodo == http.MethodPost || c.metodo == http.MethodPatch {
			t.Fatalf("la actualización no debe enviar %s al CRUD; capturas=%+v", c.metodo, capturas)
		}
	}
	if len(capturas) == 0 {
		t.Fatalf("no se recibió ninguna petición")
	}
	put := capturas[len(capturas)-1]
	if put.metodo != http.MethodPut || put.path != "/v1/contacto/7" {
		t.Fatalf("la última petición debe ser PUT /v1/contacto/7; recibida=%s %s", put.metodo, put.path)
	}
	if put.cuerpo["ValorEspecifico"] != "nuevo@example.com" {
		t.Errorf("el valor recibido debe prevalecer: ValorEspecifico=%v", put.cuerpo["ValorEspecifico"])
	}
	if put.cuerpo["NombreContacto"] != "Ana" || put.cuerpo["Proposito"] != "Notificaciones" {
		t.Errorf("los campos omitidos deben conservar el valor leído: cuerpo=%+v", put.cuerpo)
	}
	if put.cuerpo["Id"] != float64(7) {
		t.Errorf("el cuerpo del PUT debe llevar Id=7: cuerpo=%+v", put.cuerpo)
	}
	fk, ok := put.cuerpo["ProveedorId"].(map[string]any)
	if !ok || len(fk) != 1 || fk["Id"] != float64(1) {
		t.Errorf("la FK ProveedorId debe enviarse como {Id:1} sin el objeto completo: %+v", put.cuerpo["ProveedorId"])
	}
}

// Si el CRUD devuelve un registro sin campos (solo Id), no se envía ningún PUT: un
// PUT con esos datos anularía las columnas que no se conocen.
func TestActualizarNoEscribeSiElRegistroExistenteNoTraeCampos(t *testing.T) {
	var capturas []peticionCapturada
	server := crudSimulado(t, map[string]string{
		"GET /v1/contacto/7": `{"Data":{"Id":7}}`,
	}, &capturas)
	defer server.Close()

	r := NewEntityWriteResolver(
		clients.HTTPClient{Client: server.Client()},
		urlsDe(server),
		"gestion_personas_crud",
		endpointContacto,
		nil,
	)
	_, err := r.Materialize(context.Background(), config.RelationConfig{}, map[string]any{
		"Id":              float64(7),
		"ValorEspecifico": "nuevo@example.com",
	}, nil)
	if err == nil {
		t.Fatalf("debe fallar cuando el registro existente no trae campos")
	}
	for _, c := range capturas {
		if c.metodo != http.MethodGet {
			t.Fatalf("no debe escribirse nada si no se pudo leer el registro; recibido %s %s", c.metodo, c.path)
		}
	}
}

// Si el registro con ese Id no existe (404), el payload se crea por POST conservando
// el Id propagado. Es el caso de la PK compartida (proveedor_natural) del nodo que el
// POST del padre acaba de crear.
func TestActualizarConRegistroAusenteCreaPorPOST(t *testing.T) {
	var capturas []peticionCapturada
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturas = append(capturas, peticionCapturada{metodo: r.Method, path: r.URL.Path})
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"Success":false,"Status":"404"}`))
			return
		}
		_, _ = w.Write([]byte(`{"Data":{"Id":7}}`))
	}))
	defer server.Close()

	r := NewEntityWriteResolver(
		clients.HTTPClient{Client: server.Client()},
		urlsDe(server),
		"gestion_personas_crud",
		endpointContacto,
		nil,
	)
	salida, err := r.Materialize(context.Background(), config.RelationConfig{}, map[string]any{
		"Id":              float64(7),
		"ValorEspecifico": "nuevo@example.com",
	}, nil)
	if err != nil {
		t.Fatalf("Materialize no debe fallar si el registro no existe: %v", err)
	}
	if salida["Id"] != 7 {
		t.Errorf("debe conservar el Id propagado 7; salida=%+v", salida)
	}
	if len(capturas) < 2 || capturas[len(capturas)-1].metodo != http.MethodPost || capturas[len(capturas)-1].path != "/v1/contacto/" {
		t.Fatalf("debe crearse por POST /v1/contacto/; capturas=%+v", capturas)
	}
	for _, c := range capturas {
		if c.metodo == http.MethodPut || c.metodo == http.MethodPatch {
			t.Fatalf("no debe actualizarse un registro ausente; capturas=%+v", capturas)
		}
	}
}

// Un error del CRUD al leer el registro (no un 404) aborta la actualización sin
// escribir: no se sabe si el registro existe.
func TestActualizarNoEscribeSiLaLecturaPreviaFallaConError(t *testing.T) {
	var escrituras atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"Success":false,"Status":"500"}`))
			return
		}
		escrituras.Add(1)
		_, _ = w.Write([]byte(`{"Data":{"Id":7}}`))
	}))
	defer server.Close()

	r := NewEntityWriteResolver(
		clients.HTTPClient{Client: server.Client()},
		urlsDe(server),
		"gestion_personas_crud",
		endpointContacto,
		nil,
	)
	_, err := r.Materialize(context.Background(), config.RelationConfig{}, map[string]any{
		"Id":              float64(7),
		"ValorEspecifico": "nuevo@example.com",
	}, nil)
	if err == nil {
		t.Fatalf("debe fallar cuando la lectura previa responde 500")
	}
	if n := escrituras.Load(); n != 0 {
		t.Fatalf("no debe escribirse nada tras una lectura fallida; escrituras=%d", n)
	}
}

// Recuperar con un Id que el CRUD no tiene (404) no es un error: devuelve nil para que
// el motor cree el registro, como indica la documentación de Recuperar.
func TestRecuperarConIdAusenteDevuelveNilSinError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"Success":false,"Status":"404"}`))
	}))
	defer server.Close()

	r := NewEntityWriteResolver(
		clients.HTTPClient{Client: server.Client()},
		urlsDe(server),
		"gestion_personas_crud",
		endpointProveedorNatural,
		nil,
	)
	reg, err := r.Recuperar(context.Background(), config.RelationConfig{}, map[string]any{
		"Id": float64(101),
	}, nil)
	if err != nil {
		t.Fatalf("Recuperar no debe fallar con un Id ausente: %v", err)
	}
	if reg != nil {
		t.Fatalf("Recuperar debe devolver nil para un Id ausente; registro=%+v", reg)
	}
}
