package test

import (
	"net/http"
	"testing"
)

// cuentaPOST cuenta las creaciones dirigidas a un recurso.
func cuentaPOST(crud *CRUDSimulado, recurso string) int {
	n := 0
	for _, c := range crud.Capturas() {
		if c.Metodo == http.MethodPost && recursoDePath(c.Path) == recurso {
			n++
		}
	}
	return n
}

// Reenviar el Data materializado: cada entidad con Id se reencuentra por Id y se
// actualiza sin crearse de nuevo.
func TestReintentoReencuentraEntidadPorIdSinDuplicar(t *testing.T) {
	existentes := existentesBase()
	existentes["GET /v1/contacto/601"] = []map[string]any{{"Id": float64(601), "ProveedorId": float64(101)}}
	crud := NuevoCRUDSimulado(t, existentes)
	router := routerProveedor(t, crud)

	cuerpo := cuerpoProveedorNaturalUpdate(101, 201, 301)
	cuerpo["Contacto"] = []any{map[string]any{
		"Id":                  float64(601),
		"Proposito":           "notificaciones",
		"TipoCanalContactoId": float64(tipoCanalCorreo),
		"ValorEspecifico":     "ana@correo.co",
	}}
	rec, _ := ejecutar(t, router, http.MethodPatch, comoJSON(t, cuerpo))
	if rec.Code != http.StatusOK && rec.Code != http.StatusPartialContent {
		t.Fatalf("status = %d; esperado 200 o 206. body=%s", rec.Code, rec.Body.String())
	}
	if n := cuentaPOST(crud, "contacto"); n != 0 {
		t.Errorf("la entidad con Id no debe recrearse; POST contacto=%d", n)
	}
}

// Una entidad sin Id pero con clave natural se reencuentra por su clave natural: el
// proveedor declara TerceroId como clave natural, así que un cuerpo sin Id con ese
// TerceroId actualiza el proveedor existente y no crea otro.
func TestReintentoReencuentraEntidadPorClaveNaturalSinDuplicar(t *testing.T) {
	crud := NuevoCRUDSimulado(t, existentesBase())
	router := routerProveedor(t, crud)

	rec, _ := ejecutar(t, router, http.MethodPatch, comoJSON(t, cuerpoProveedorNatural()))
	if rec.Code != http.StatusOK && rec.Code != http.StatusPartialContent {
		t.Fatalf("status = %d; esperado 200 o 206. body=%s", rec.Code, rec.Body.String())
	}
	if n := cuentaPOST(crud, "proveedor"); n != 0 {
		t.Errorf("la clave natural TerceroId debe reencontrar el proveedor sin crear duplicado; POST proveedor=%d", n)
	}
}

// Creación en frío sin Id ni FK ascendente: se crea exactamente una vez.
func TestCreacionEnFriaCreaUnaSolaVez(t *testing.T) {
	crud := NuevoCRUDSimulado(t, nil)
	router := routerProveedor(t, crud)

	rec, _ := ejecutar(t, router, http.MethodPost, comoJSON(t, cuerpoProveedorNaturalConTerceroID(nil, false)))
	if rec.Code != http.StatusOK && rec.Code != http.StatusPartialContent {
		t.Fatalf("status = %d; esperado 200 o 206. body=%s", rec.Code, rec.Body.String())
	}
	if n := cuentaPOST(crud, "proveedor"); n != 1 {
		t.Errorf("la creación en frío debe crear el proveedor una sola vez; POST proveedor=%d", n)
	}
}

// Reenviar el cuerpo original de una colección sin clave puede volver a crearla: es
// la limitación declarada del contrato, no una garantía de idempotencia.
func TestReenvioCuerpoOriginalColeccionSinClavePuedeDuplicar(t *testing.T) {
	crud := NuevoCRUDSimulado(t, existentesBase())
	router := routerProveedor(t, crud)

	cuerpo := cuerpoProveedorNaturalUpdate(101, 201, 301)
	cuerpo["Contacto"] = []any{map[string]any{
		"Proposito":           "notificaciones",
		"TipoCanalContactoId": float64(tipoCanalCorreo),
		"ValorEspecifico":     "ana@correo.co",
	}}
	ejecutar(t, router, http.MethodPatch, comoJSON(t, cuerpo))
	ejecutar(t, router, http.MethodPatch, comoJSON(t, cuerpo))
	if n := cuentaPOST(crud, "contacto"); n < 2 {
		t.Errorf("reenviar el cuerpo original debe poder volver a crear el contacto sin clave; POST contacto=%d", n)
	}
}

// Reintento tras transición de estado a medio camino: el nuevo estado ya existe y el
// previo sigue activo; el reintento no crea un segundo estado y solo da de baja el previo.
func TestReintentoTransicionEstadoMedioCaminoSoloBajaPendiente(t *testing.T) {
	existentes := existentesBase()
	existentes["GET /v1/estado_proveedor"] = []map[string]any{
		{"Id": float64(501), "ProveedorId": float64(101), "EstadoId": float64(1), "Activo": true},
		{"Id": float64(502), "ProveedorId": float64(101), "EstadoId": float64(2), "Activo": true},
	}
	crud := NuevoCRUDSimulado(t, existentes)
	router := routerProveedor(t, crud)

	cuerpo := cuerpoProveedorNaturalUpdate(101, 201, 301)
	cuerpo["EstadoProveedor"] = []any{map[string]any{"EstadoId": float64(2)}}
	rec, _ := ejecutar(t, router, http.MethodPatch, comoJSON(t, cuerpo))
	if rec.Code != http.StatusOK && rec.Code != http.StatusPartialContent {
		t.Fatalf("status = %d; esperado 200 o 206. body=%s", rec.Code, rec.Body.String())
	}
	if n := cuentaPOST(crud, "estado_proveedor"); n != 0 {
		t.Errorf("el reintento no debe crear un segundo estado; POST estado=%d", n)
	}
	var bajaPrevio bool
	for _, c := range crud.Capturas() {
		if c.Metodo == http.MethodDelete && c.Path == "/v1/estado_proveedor/501" {
			bajaPrevio = true
		}
	}
	if !bajaPrevio {
		t.Errorf("el reintento debe completar la baja pendiente del estado previo 501")
	}
}
