package test

import (
	"net/http"
	"testing"
	"time"

	"github.com/udistrital/agora_mid/config"
	"github.com/udistrital/agora_mid/controllers"
	"github.com/udistrital/agora_mid/services"
)

// routerProveedorConPolitica arma el router de escritura con una política de error
// de relación explícita, sobrescribiendo la de la configuración real.
func routerProveedorConPolitica(t *testing.T, crud *CRUDSimulado, politica string) http.Handler {
	t.Helper()
	urls := crud.URLs()
	relaciones, err := config.LoadRelaciones("../config", urls)
	if err != nil {
		t.Fatalf("cargando configuración de relaciones real: %v", err)
	}
	cfg := relaciones["proveedor"]
	cfg.OnRelationError = politica

	escritura := services.NewProveedorWriteService(crud.Client(), urls, cfg, tipoRegistroNatural)
	lectura := services.NewProveedorService(crud.Client(), urls, cfg, tipoRegistroNatural, nil, nil)
	handler := controllers.NewProveedorHandlerCompleto(lectura, escritura)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/proveedor/consulta/lista":
			handler.ConsultarLista(w, r)
			return
		case "/v1/proveedor/consulta":
			handler.Consultar(w, r)
			return
		}
		handler.Escribir(w, r)
	})
}

// escriturasNoGET cuenta las peticiones que no son lecturas.
func escriturasNoGET(crud *CRUDSimulado) int {
	n := 0
	for _, c := range crud.Capturas() {
		if c.Metodo != http.MethodGet {
			n++
		}
	}
	return n
}

// existentesBase precarga el proveedor y su estado activo para una actualización.
func existentesBase() map[string][]map[string]any {
	return map[string][]map[string]any{
		"GET /v1/proveedor": {{"Id": float64(101), "TerceroId": float64(25)}},
		"GET /v1/estado_proveedor": {
			{"Id": float64(501), "ProveedorId": float64(101), "EstadoId": float64(1), "Activo": true},
		},
	}
}

// Dependencia inalcanzable en la lectura de la raíz durante el pre-pass: 502 sin
// Data y sin ninguna escritura.
func TestPrePassRaizInaccesibleResponde502SinEscribir(t *testing.T) {
	crud := NuevoCRUDSimulado(t, existentesBase())
	crud.FallarMetodoDe(http.MethodGet, "proveedor", http.StatusServiceUnavailable)
	router := routerProveedor(t, crud)

	rec, env := ejecutar(t, router, http.MethodPatch, comoJSON(t, cuerpoProveedorNaturalUpdate(101, 201, 301)))
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d; esperado 502. body=%s", rec.Code, rec.Body.String())
	}
	if env.Data != nil {
		t.Errorf("no debe haber Data parcial; data=%+v", env.Data)
	}
	if n := escriturasNoGET(crud); n != 0 {
		t.Errorf("no debe haber escrituras; peticiones no GET=%d", n)
	}
}

// Tiempo de espera en la lectura de la raíz durante el pre-pass: 504 sin Data y sin
// ninguna escritura.
func TestPrePassRaizTiempoAgotadoResponde504SinEscribir(t *testing.T) {
	crud := NuevoCRUDSimulado(t, existentesBase())
	crud.ConTimeoutCliente(200 * time.Millisecond)
	crud.RetardarMetodoDe(http.MethodGet, "proveedor", 5*time.Second)
	router := routerProveedor(t, crud)

	rec, env := ejecutar(t, router, http.MethodPatch, comoJSON(t, cuerpoProveedorNaturalUpdate(101, 201, 301)))
	if rec.Code != http.StatusGatewayTimeout {
		t.Fatalf("status = %d; esperado 504. body=%s", rec.Code, rec.Body.String())
	}
	if env.Data != nil {
		t.Errorf("no debe haber Data parcial; data=%+v", env.Data)
	}
	if n := escriturasNoGET(crud); n != 0 {
		t.Errorf("no debe haber escrituras; peticiones no GET=%d", n)
	}
}

// Política fail en una relación no raíz: la lectura previa caída aborta con 502 y
// sin ninguna escritura.
func TestPrePassPoliticaFailNoRaizResponde502SinEscribir(t *testing.T) {
	crud := NuevoCRUDSimulado(t, existentesBase())
	crud.FallarMetodoDe(http.MethodGet, "estado_proveedor", http.StatusServiceUnavailable)
	router := routerProveedorConPolitica(t, crud, config.OnRelationErrorFail)

	cuerpo := cuerpoProveedorNaturalUpdate(101, 201, 301)
	cuerpo["EstadoProveedor"] = []any{map[string]any{"EstadoId": float64(2)}}
	rec, _ := ejecutar(t, router, http.MethodPatch, comoJSON(t, cuerpo))
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d; esperado 502 con política fail. body=%s", rec.Code, rec.Body.String())
	}
	if n := escriturasNoGET(crud); n != 0 {
		t.Errorf("no debe haber escrituras; peticiones no GET=%d", n)
	}
}

// Política fail en una relación no raíz con tiempo de espera: 504, no 502, porque el
// tiempo de espera debe conservar su causa hasta el controlador.
func TestPrePassPoliticaFailNoRaizTiempoAgotadoResponde504(t *testing.T) {
	crud := NuevoCRUDSimulado(t, existentesBase())
	crud.ConTimeoutCliente(200 * time.Millisecond)
	crud.RetardarMetodoDe(http.MethodGet, "estado_proveedor", 5*time.Second)
	router := routerProveedorConPolitica(t, crud, config.OnRelationErrorFail)

	cuerpo := cuerpoProveedorNaturalUpdate(101, 201, 301)
	cuerpo["EstadoProveedor"] = []any{map[string]any{"EstadoId": float64(2)}}
	rec, _ := ejecutar(t, router, http.MethodPatch, comoJSON(t, cuerpo))
	if rec.Code != http.StatusGatewayTimeout {
		t.Fatalf("status = %d; esperado 504 con política fail. body=%s", rec.Code, rec.Body.String())
	}
	if n := escriturasNoGET(crud); n != 0 {
		t.Errorf("no debe haber escrituras; peticiones no GET=%d", n)
	}
}

// La lectura del estado activo que decide la transición es bloqueante: si falla,
// la escritura responde 502 sin materializar nada, aunque la política sea notify.
func TestPrePassEstadoLecturaFallaResponde502SinEscribir(t *testing.T) {
	crud := NuevoCRUDSimulado(t, existentesBase())
	crud.FallarMetodoDe(http.MethodGet, "estado_proveedor", http.StatusServiceUnavailable)
	router := routerProveedor(t, crud)

	cuerpo := cuerpoProveedorNaturalUpdate(101, 201, 301)
	cuerpo["EstadoProveedor"] = []any{map[string]any{"EstadoId": float64(2)}}
	rec, env := ejecutar(t, router, http.MethodPatch, comoJSON(t, cuerpo))
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d; esperado 502. body=%s", rec.Code, rec.Body.String())
	}
	if env.Data != nil {
		t.Errorf("no debe haber Data parcial; data=%+v", env.Data)
	}
	if n := escriturasNoGET(crud); n != 0 {
		t.Errorf("no debe haber escrituras; peticiones no GET=%d", n)
	}
}

// El tiempo de espera de la lectura del estado activo responde 504 sin materializar
// nada, con la misma regla bloqueante que el fallo de la lectura.
func TestPrePassEstadoLecturaTiempoAgotadoResponde504SinEscribir(t *testing.T) {
	crud := NuevoCRUDSimulado(t, existentesBase())
	crud.ConTimeoutCliente(200 * time.Millisecond)
	crud.RetardarMetodoDe(http.MethodGet, "estado_proveedor", 5*time.Second)
	router := routerProveedor(t, crud)

	cuerpo := cuerpoProveedorNaturalUpdate(101, 201, 301)
	cuerpo["EstadoProveedor"] = []any{map[string]any{"EstadoId": float64(2)}}
	rec, env := ejecutar(t, router, http.MethodPatch, comoJSON(t, cuerpo))
	if rec.Code != http.StatusGatewayTimeout {
		t.Fatalf("status = %d; esperado 504. body=%s", rec.Code, rec.Body.String())
	}
	if env.Data != nil {
		t.Errorf("no debe haber Data parcial; data=%+v", env.Data)
	}
	if n := escriturasNoGET(crud); n != 0 {
		t.Errorf("no debe haber escrituras; peticiones no GET=%d", n)
	}
}

// Una validación previa produce 400 y nunca 206, aunque la lectura tolerada haya
// generado un aviso antes de detectar el error.
func TestValidacionPreviaNuncaDa206AunqueHayaAvisoTolerado(t *testing.T) {
	casos := map[string]func(map[string]any){
		"varios estados entrantes": func(c map[string]any) {
			c["EstadoProveedor"] = []any{
				map[string]any{"EstadoId": float64(2)},
				map[string]any{"EstadoId": float64(3)},
			}
		},
		"campo de auditoría administrado": func(c map[string]any) {
			c["FechaCreacion"] = "2026-01-01"
		},
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			crud := NuevoCRUDSimulado(t, existentesBase())
			crud.FallarMetodoDe(http.MethodGet, "estado_proveedor", http.StatusServiceUnavailable)
			router := routerProveedor(t, crud)

			cuerpo := cuerpoProveedorNaturalUpdate(101, 201, 301)
			mutar(cuerpo)
			rec, _ := ejecutar(t, router, http.MethodPatch, comoJSON(t, cuerpo))
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d; esperado 400 y nunca 206. body=%s", rec.Code, rec.Body.String())
			}
			if n := escriturasNoGET(crud); n != 0 {
				t.Errorf("una validación previa no debe escribir; peticiones no GET=%d", n)
			}
		})
	}
}
