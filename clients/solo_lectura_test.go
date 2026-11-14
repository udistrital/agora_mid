package clients

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Cuenta las peticiones que llegan al servidor de prueba.
func servidorContador(t *testing.T) (*httptest.Server, *int) {
	t.Helper()
	llamadas := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		llamadas++
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"Id":1}`)
	}))
	t.Cleanup(srv.Close)
	return srv, &llamadas
}

func protegerONoFalla(t *testing.T, bases ...string) {
	t.Helper()
	if err := ProtegerSoloLectura(bases...); err != nil {
		t.Fatalf("ProtegerSoloLectura: %v", err)
	}
	t.Cleanup(func() { _ = ProtegerSoloLectura() })
}

func TestEscrituraAProtegidoSeBloqueaSinLlamarAlServicio(t *testing.T) {
	srv, llamadas := servidorContador(t)
	protegerONoFalla(t, srv.URL)
	cliente := HTTPClient{}
	ctx := context.Background()

	var resp map[string]any
	if _, err := cliente.Post(ctx, srv.URL+"/v1/tercero/", map[string]any{"NombreCompleto": "x"}, &resp); err == nil {
		t.Error("POST a un CRUD protegido debe fallar")
	}
	if _, err := cliente.Patch(ctx, srv.URL+"/v1/tercero/1", map[string]any{"NombreCompleto": "x"}, &resp); err == nil {
		t.Error("PATCH a un CRUD protegido debe fallar")
	}
	if _, err := cliente.Put(ctx, srv.URL+"/v1/tercero/1", map[string]any{"NombreCompleto": "x"}, &resp); err == nil {
		t.Error("PUT a un CRUD protegido debe fallar")
	}
	if _, err := cliente.Delete(ctx, srv.URL+"/v1/tercero/1", nil); err == nil {
		t.Error("DELETE a un CRUD protegido debe fallar")
	}
	if *llamadas != 0 {
		t.Errorf("ninguna escritura debe llegar al servicio protegido; llegaron %d", *llamadas)
	}
}

func TestLecturaAProtegidoSigueFuncionando(t *testing.T) {
	srv, llamadas := servidorContador(t)
	protegerONoFalla(t, srv.URL)

	var resp map[string]any
	if _, err := (HTTPClient{}).Get(context.Background(), srv.URL+"/v1/tercero/1", &resp); err != nil {
		t.Fatalf("GET a un CRUD protegido debe funcionar: %v", err)
	}
	if *llamadas != 1 {
		t.Errorf("el GET debe llegar una vez; llegaron %d", *llamadas)
	}
}

func TestEscrituraAOtroServicioNoSeBloquea(t *testing.T) {
	protegido, _ := servidorContador(t)
	permitido, llamadas := servidorContador(t)
	protegerONoFalla(t, protegido.URL)

	var resp map[string]any
	if _, err := (HTTPClient{}).Post(context.Background(), permitido.URL+"/v1/contacto/", map[string]any{"a": 1}, &resp); err != nil {
		t.Fatalf("POST a un CRUD no protegido debe funcionar: %v", err)
	}
	if *llamadas != 1 {
		t.Errorf("el POST debe llegar al servicio permitido; llegaron %d", *llamadas)
	}
}

func TestEsProtegidaCompara_EsquemaHostYRuta(t *testing.T) {
	protegerONoFalla(t, "http://h.example:8121")

	casos := []struct {
		url  string
		want bool
	}{
		{"http://h.example:8121/v1/tercero/", true},
		{"http://h.example:8121", true},
		{"http://h.example:81210/v1/tercero/", false},
		{"https://h.example:8121/v1/tercero/", false},
		{"http://otro.example:8121/v1/tercero/", false},
	}
	for _, c := range casos {
		if got := esProtegida(c.url); got != c.want {
			t.Errorf("esProtegida(%q) = %v; se esperaba %v", c.url, got, c.want)
		}
	}
}

func TestProtegerConBaseVaciaFalla(t *testing.T) {
	if err := ProtegerSoloLectura("  "); err == nil {
		t.Error("una base vacía no debe aceptarse: fallaría abierto")
	}
}

func TestErrorDeBloqueoNoExponeLaURL(t *testing.T) {
	srv, _ := servidorContador(t)
	protegerONoFalla(t, srv.URL)

	var resp map[string]any
	_, err := (HTTPClient{}).Post(context.Background(), srv.URL+"/v1/tercero/", map[string]any{"a": 1}, &resp)
	if err == nil {
		t.Fatal("POST a un CRUD protegido debe fallar")
	}
	if strings.Contains(err.Error(), srv.URL) {
		t.Errorf("el error no debe exponer la URL del servicio: %v", err)
	}
}

// Regresión: escrito después de la implementación, no visto en rojo primero.
func TestRedireccionDeEscrituraAProtegidoSeBloquea(t *testing.T) {
	protegido, llamadasProtegido := servidorContador(t)
	protegerONoFalla(t, protegido.URL)

	redir := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, protegido.URL+"/v1/tercero/", http.StatusTemporaryRedirect)
	}))
	t.Cleanup(redir.Close)

	var resp map[string]any
	if _, err := (HTTPClient{}).Post(context.Background(), redir.URL+"/v1/x/", map[string]any{"a": 1}, &resp); err == nil {
		t.Error("una redirección de escritura hacia un CRUD protegido debe fallar")
	}
	if *llamadasProtegido != 0 {
		t.Errorf("la escritura redirigida no debe llegar al CRUD protegido; llegaron %d", *llamadasProtegido)
	}
}
