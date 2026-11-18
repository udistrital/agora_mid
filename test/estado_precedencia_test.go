package test

import (
	"net/http"
	"testing"
)

// Con Id presente que coincide con el estado activo, la escritura lo actualiza en
// sitio: no crea ni da de baja ningún estado.
func TestEstadoPrecedenciaIdPresenteActualizaEnSitio(t *testing.T) {
	crud := NuevoCRUDSimulado(t, existentesBase())
	router := routerProveedor(t, crud)

	cuerpo := cuerpoProveedorNaturalUpdate(101, 201, 301)
	cuerpo["EstadoProveedor"] = []any{map[string]any{"Id": float64(501), "EstadoId": float64(1), "Observacion": "nueva"}}
	rec, _ := ejecutar(t, router, http.MethodPatch, comoJSON(t, cuerpo))
	if rec.Code != http.StatusOK && rec.Code != http.StatusPartialContent {
		t.Fatalf("status = %d; esperado 200 o 206. body=%s", rec.Code, rec.Body.String())
	}
	if n := cuentaPOST(crud, "estado_proveedor"); n != 0 {
		t.Errorf("con Id coincidente no debe crearse un estado; POST estado=%d", n)
	}
	for _, c := range crud.Capturas() {
		if c.Metodo == http.MethodDelete && recursoDePath(c.Path) == "estado_proveedor" {
			t.Errorf("con Id coincidente no debe darse de baja ningún estado; captura=%+v", c)
		}
	}
}

// Sin Id y con el mismo EstadoId pero una Observacion distinta, el intento es una
// transición normal: crea el nuevo estado y después da de baja el activo previo.
func TestEstadoPrecedenciaSinIdCamposDistintosCreaYDaBaja(t *testing.T) {
	crud := NuevoCRUDSimulado(t, existentesBase())
	router := routerProveedor(t, crud)

	cuerpo := cuerpoProveedorNaturalUpdate(101, 201, 301)
	cuerpo["EstadoProveedor"] = []any{map[string]any{"EstadoId": float64(1), "Observacion": "nueva"}}
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
			idxCrear = i
		}
		if c.Metodo == http.MethodDelete && c.Path == "/v1/estado_proveedor/501" {
			idxBaja = i
		}
	}
	if idxCrear < 0 {
		t.Fatal("debe crearse el nuevo estado")
	}
	if idxBaja < 0 {
		t.Fatal("debe darse de baja el activo previo 501")
	}
	if idxCrear > idxBaja {
		t.Errorf("el nuevo estado debe crearse antes de la baja; crear=%d baja=%d", idxCrear, idxBaja)
	}
}

// Con Id presente que coincide con el estado activo, la actualización en sitio gana
// aunque el EstadoId del intento sea distinto: no se crea ni se da de baja ningún estado.
func TestEstadoPrecedenciaIdPresenteEstadoIdDistintoActualizaEnSitio(t *testing.T) {
	crud := NuevoCRUDSimulado(t, existentesBase())
	router := routerProveedor(t, crud)

	cuerpo := cuerpoProveedorNaturalUpdate(101, 201, 301)
	cuerpo["EstadoProveedor"] = []any{map[string]any{"Id": float64(501), "EstadoId": float64(3)}}
	rec, _ := ejecutar(t, router, http.MethodPatch, comoJSON(t, cuerpo))
	if rec.Code != http.StatusOK && rec.Code != http.StatusPartialContent {
		t.Fatalf("status = %d; esperado 200 o 206. body=%s", rec.Code, rec.Body.String())
	}
	if n := cuentaPOST(crud, "estado_proveedor"); n != 0 {
		t.Errorf("con Id coincidente no debe crearse un estado; POST estado=%d", n)
	}
	for _, c := range crud.Capturas() {
		if c.Metodo == http.MethodDelete && recursoDePath(c.Path) == "estado_proveedor" {
			t.Errorf("con Id coincidente no debe darse de baja ningún estado; captura=%+v", c)
		}
	}
}
