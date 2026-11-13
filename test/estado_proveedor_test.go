package test

import (
	"net/http"
	"testing"
)

// El estado activo previo se da de baja después de crear el nuevo, de modo que un fallo
// intermedio deja el proveedor con el nuevo estado y no sin estado activo.
func TestIntegracionTransicionEstadoDaDeBajaElPrevioDespuesDeCrear(t *testing.T) {
	crud := NuevoCRUDSimulado(t, map[string][]map[string]any{
		"GET /v1/proveedor": {{"Id": float64(101), "TerceroId": float64(25)}},
		"GET /v1/estado_proveedor": {
			{"Id": float64(501), "ProveedorId": float64(101), "EstadoId": float64(1), "Activo": true},
		},
	})
	router := routerProveedor(t, crud)

	cuerpo := cuerpoProveedorNaturalUpdate(101, 201, 301)
	cuerpo["EstadoProveedor"] = []any{map[string]any{"EstadoId": float64(2)}}

	rec, _ := ejecutar(t, router, http.MethodPatch, comoJSON(t, cuerpo))
	if rec.Code != http.StatusOK && rec.Code != http.StatusPartialContent {
		t.Fatalf("status = %d; esperado 200 o 206. body=%s", rec.Code, rec.Body.String())
	}

	idxCrear, idxBaja := -1, -1
	for i, c := range crud.Capturas() {
		if recursoDePath(c.Path) != "estado_proveedor" {
			continue
		}
		if c.Metodo == http.MethodPost && idxCrear < 0 {
			if v, _ := c.Cuerpo["EstadoId"].(float64); int(v) == 2 {
				idxCrear = i
			}
		}
		if c.Metodo == http.MethodDelete && c.Path == "/v1/estado_proveedor/501" && idxBaja < 0 {
			idxBaja = i
		}
	}
	if idxCrear < 0 {
		t.Fatalf("debe crearse el nuevo estado con EstadoId=2; capturas=%+v", crud.Capturas())
	}
	if idxBaja < 0 {
		t.Fatalf("debe darse de baja el estado activo previo 501; capturas=%+v", crud.Capturas())
	}
	if idxCrear > idxBaja {
		t.Errorf("el nuevo estado debe crearse antes de dar de baja el previo; crear=%d baja=%d", idxCrear, idxBaja)
	}
}

// Reenviar la misma intención no crea ni da de baja estados.
func TestIntegracionTransicionEstadoMismoEstadoEsNoOp(t *testing.T) {
	crud := NuevoCRUDSimulado(t, map[string][]map[string]any{
		"GET /v1/proveedor": {{"Id": float64(101), "TerceroId": float64(25)}},
		"GET /v1/estado_proveedor": {
			{"Id": float64(501), "ProveedorId": float64(101), "EstadoId": float64(1), "Activo": true},
		},
	})
	router := routerProveedor(t, crud)

	cuerpo := cuerpoProveedorNaturalUpdate(101, 201, 301)
	cuerpo["EstadoProveedor"] = []any{map[string]any{"EstadoId": float64(1)}}

	rec, _ := ejecutar(t, router, http.MethodPatch, comoJSON(t, cuerpo))
	if rec.Code != http.StatusOK && rec.Code != http.StatusPartialContent {
		t.Fatalf("status = %d; esperado 200 o 206. body=%s", rec.Code, rec.Body.String())
	}

	for _, c := range crud.Capturas() {
		if recursoDePath(c.Path) != "estado_proveedor" {
			continue
		}
		if c.Metodo == http.MethodPost || c.Metodo == http.MethodDelete {
			t.Errorf("un estado igual al activo no debe crear ni dar de baja estados; captura=%+v", c)
		}
	}
}

// Una escritura registra un solo estado: varios estados o un estado sin EstadoId
// responden 400 sin escribir en ningún CRUD.
func TestIntegracionTransicionRechazaEstadoInvalidoSinEscribir(t *testing.T) {
	casos := map[string][]any{
		"varios estados": {
			map[string]any{"EstadoId": float64(2)},
			map[string]any{"EstadoId": float64(3)},
		},
		"estado sin EstadoId": {
			map[string]any{"Observacion": "sin estado"},
		},
	}
	for nombre, estados := range casos {
		t.Run(nombre, func(t *testing.T) {
			crud := NuevoCRUDSimulado(t, map[string][]map[string]any{
				"GET /v1/proveedor": {{"Id": float64(101), "TerceroId": float64(25)}},
				"GET /v1/estado_proveedor": {
					{"Id": float64(501), "ProveedorId": float64(101), "EstadoId": float64(1), "Activo": true},
				},
			})
			router := routerProveedor(t, crud)

			cuerpo := cuerpoProveedorNaturalUpdate(101, 201, 301)
			cuerpo["EstadoProveedor"] = estados

			rec, _ := ejecutar(t, router, http.MethodPatch, comoJSON(t, cuerpo))
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d; esperado 400. body=%s", rec.Code, rec.Body.String())
			}
			for _, c := range crud.Capturas() {
				if c.Metodo != http.MethodGet {
					t.Errorf("un estado inválido no debe escribir; captura=%+v", c)
				}
			}
		})
	}
}

// El estado nuevo se crea y el activo previo del proveedor resuelto se da de baja.
func TestIntegracionTransicionResuelveProveedorPorTerceroIdSinId(t *testing.T) {
	crud := NuevoCRUDSimulado(t, map[string][]map[string]any{
		"GET /v1/proveedor": {{"Id": float64(101), "TerceroId": float64(25), "Activo": true}},
		"GET /v1/estado_proveedor": {
			{"Id": float64(501), "ProveedorId": float64(101), "EstadoId": float64(1), "Activo": true},
		},
	})
	router := routerProveedor(t, crud)

	cuerpo := cuerpoProveedorNatural()
	cuerpo["EstadoProveedor"] = []any{map[string]any{"EstadoId": float64(2)}}

	rec, _ := ejecutar(t, router, http.MethodPost, comoJSON(t, cuerpo))
	if rec.Code != http.StatusOK && rec.Code != http.StatusPartialContent {
		t.Fatalf("status = %d; esperado 200 o 206. body=%s", rec.Code, rec.Body.String())
	}

	var creado, dadoDeBaja bool
	for _, c := range crud.Capturas() {
		if recursoDePath(c.Path) != "estado_proveedor" {
			continue
		}
		if c.Metodo == http.MethodPost {
			if v, _ := c.Cuerpo["EstadoId"].(float64); int(v) == 2 {
				creado = true
			}
		}
		if c.Metodo == http.MethodDelete && c.Path == "/v1/estado_proveedor/501" {
			dadoDeBaja = true
		}
	}
	if !creado || !dadoDeBaja {
		t.Errorf("el proveedor resuelto por TerceroId debe crear EstadoId=2 y dar de baja 501; creado=%v baja=%v capturas=%+v", creado, dadoDeBaja, crud.Capturas())
	}
}

func TestIntegracionTransicionSinEstadoPreviosCreaSinBaja(t *testing.T) {
	crud := NuevoCRUDSimulado(t, map[string][]map[string]any{
		"GET /v1/proveedor": {{"Id": float64(101), "TerceroId": float64(25)}},
	})
	router := routerProveedor(t, crud)

	cuerpo := cuerpoProveedorNaturalUpdate(101, 201, 301)
	cuerpo["EstadoProveedor"] = []any{map[string]any{"EstadoId": float64(2)}}

	rec, _ := ejecutar(t, router, http.MethodPatch, comoJSON(t, cuerpo))
	if rec.Code != http.StatusOK && rec.Code != http.StatusPartialContent {
		t.Fatalf("status = %d; esperado 200 o 206. body=%s", rec.Code, rec.Body.String())
	}

	var creado bool
	for _, c := range crud.Capturas() {
		if recursoDePath(c.Path) != "estado_proveedor" {
			continue
		}
		if c.Metodo == http.MethodPost {
			if v, _ := c.Cuerpo["EstadoId"].(float64); int(v) == 2 {
				creado = true
			}
		}
		if c.Metodo == http.MethodDelete {
			t.Errorf("sin estado previo no debe darse de baja ningún estado; captura=%+v", c)
		}
	}
	if !creado {
		t.Errorf("debe crearse el estado EstadoId=2; capturas=%+v", crud.Capturas())
	}
}

func TestIntegracionTransicionRechazaMasDeUnEstadoActivoSinEscribir(t *testing.T) {
	crud := NuevoCRUDSimulado(t, map[string][]map[string]any{
		"GET /v1/proveedor": {{"Id": float64(101), "TerceroId": float64(25)}},
		"GET /v1/estado_proveedor": {
			{"Id": float64(501), "ProveedorId": float64(101), "EstadoId": float64(1), "Activo": true},
			{"Id": float64(502), "ProveedorId": float64(101), "EstadoId": float64(3), "Activo": true},
		},
	})
	router := routerProveedor(t, crud)

	cuerpo := cuerpoProveedorNaturalUpdate(101, 201, 301)
	cuerpo["EstadoProveedor"] = []any{map[string]any{"EstadoId": float64(2)}}

	rec, _ := ejecutar(t, router, http.MethodPatch, comoJSON(t, cuerpo))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d; esperado 400 con más de un estado activo. body=%s", rec.Code, rec.Body.String())
	}
	for _, c := range crud.Capturas() {
		if c.Metodo != http.MethodGet {
			t.Errorf("con más de un estado activo no debe escribir; captura=%+v", c)
		}
	}
}

func TestIntegracionTransicionEstadoNuevoRecibeAutorDeRaiz(t *testing.T) {
	crud := NuevoCRUDSimulado(t, map[string][]map[string]any{
		"GET /v1/proveedor": {{"Id": float64(101), "TerceroId": float64(25)}},
		"GET /v1/estado_proveedor": {
			{"Id": float64(501), "ProveedorId": float64(101), "EstadoId": float64(1), "Activo": true},
		},
	})
	router := routerProveedor(t, crud)

	cuerpo := cuerpoProveedorNaturalUpdate(101, 201, 301)
	cuerpo["UsuarioModificacionId"] = float64(9)
	cuerpo["EstadoProveedor"] = []any{map[string]any{"EstadoId": float64(2)}}

	rec, _ := ejecutar(t, router, http.MethodPatch, comoJSON(t, cuerpo))
	if rec.Code != http.StatusOK && rec.Code != http.StatusPartialContent {
		t.Fatalf("status = %d; esperado 200 o 206. body=%s", rec.Code, rec.Body.String())
	}

	var autorOK bool
	for _, c := range crud.Capturas() {
		if c.Metodo == http.MethodPost && recursoDePath(c.Path) == "estado_proveedor" {
			if v, _ := c.Cuerpo["UsuarioModificacionId"].(float64); int(v) == 9 {
				autorOK = true
			}
		}
	}
	if !autorOK {
		t.Errorf("el estado nuevo debe llevar UsuarioModificacionId=9 de la raíz; capturas=%+v", crud.Capturas())
	}
}
