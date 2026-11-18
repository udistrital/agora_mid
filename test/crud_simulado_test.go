package test

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
	"time"
)

// Un fallo inducido solo se observa en el método y recurso indicados; el mismo
// recurso con otro método responde con normalidad.
func TestCRUDSimuladoFallaSoloEnMetodoIndicado(t *testing.T) {
	crud := NuevoCRUDSimulado(t, map[string][]map[string]any{
		"GET /v1/estado_proveedor": {{"Id": float64(501)}},
	})
	crud.FallarMetodoDe(http.MethodGet, "estado_proveedor", http.StatusServiceUnavailable)
	cliente := crud.Client()
	base := crud.URLs()["terceros"]

	var lectura map[string]any
	status, err := cliente.Get(context.Background(), base+"/v1/estado_proveedor", &lectura)
	if err == nil || status != http.StatusServiceUnavailable {
		t.Fatalf("GET debe fallar con 503; status=%d err=%v", status, err)
	}

	var creada map[string]any
	if _, err := cliente.Post(context.Background(), base+"/v1/estado_proveedor", map[string]any{"EstadoId": float64(2)}, &creada); err != nil {
		t.Errorf("POST no debe fallar cuando solo se indujo fallo en GET: %v", err)
	}

	if _, err := cliente.Delete(context.Background(), base+"/v1/estado_proveedor/501", nil); err != nil {
		t.Errorf("DELETE no debe fallar cuando solo se indujo fallo en GET: %v", err)
	}
}

// Un retardo mayor que el tiempo de espera del cliente produce un error de red
// con Timeout verdadero, distinto de una respuesta HTTP de error.
func TestCRUDSimuladoRetardoProvocaTimeoutDelCliente(t *testing.T) {
	crud := NuevoCRUDSimulado(t, nil)
	crud.ConTimeoutCliente(200 * time.Millisecond)
	crud.RetardarMetodoDe(http.MethodGet, "proveedor", 5*time.Second)
	cliente := crud.Client()

	inicio := time.Now()
	var lectura map[string]any
	_, err := cliente.Get(context.Background(), crud.URLs()["terceros"]+"/v1/proveedor", &lectura)
	if err == nil {
		t.Fatal("GET retardado debe agotar el tiempo de espera del cliente")
	}
	var netErr net.Error
	if !errors.As(err, &netErr) || !netErr.Timeout() {
		t.Errorf("el error debe ser un timeout de red; err=%v", err)
	}
	if time.Since(inicio) > 3*time.Second {
		t.Errorf("el cliente debió cortar antes del retardo completo; transcurrido=%v", time.Since(inicio))
	}
}
