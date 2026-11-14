package test

import (
	"net/http"
	"testing"
)

// peticionPUT devuelve el cuerpo de la última petición PUT enviada al CRUD del recurso.
func peticionPUT(crud *CRUDSimulado, recurso string) (map[string]any, bool) {
	var cuerpo map[string]any
	var hallada bool
	for _, c := range crud.Capturas() {
		if c.Metodo == http.MethodPut && recursoDePath(c.Path) == recurso {
			cuerpo = c.Cuerpo
			hallada = true
		}
	}
	return cuerpo, hallada
}

// Un campo ausente del cuerpo conserva su valor: no se envía al CRUD dueño.
func TestNulosAusenteConservaValor(t *testing.T) {
	crud := NuevoCRUDSimulado(t, existentesBase())
	router := routerProveedor(t, crud)

	cuerpo := cuerpoProveedorNaturalUpdate(101, 201, 301)
	delete(cuerpo, "DescripcionPortafolio")
	rec, _ := ejecutar(t, router, http.MethodPatch, comoJSON(t, cuerpo))
	if rec.Code != http.StatusOK && rec.Code != http.StatusPartialContent {
		t.Fatalf("status = %d; esperado 200 o 206. body=%s", rec.Code, rec.Body.String())
	}
	enviado, ok := peticionPUT(crud, "proveedor")
	if !ok {
		t.Fatal("debe enviarse el PUT del proveedor")
	}
	if _, presente := enviado["DescripcionPortafolio"]; presente {
		t.Errorf("un campo ausente no debe enviarse; cuerpo=%+v", enviado)
	}
}

// null en un escalar nullable se envía al CRUD como null.
func TestNulosEnEscalarNullableSeFija(t *testing.T) {
	crud := NuevoCRUDSimulado(t, existentesBase())
	router := routerProveedor(t, crud)

	cuerpo := cuerpoProveedorNaturalUpdate(101, 201, 301)
	cuerpo["DescripcionPortafolio"] = nil
	rec, _ := ejecutar(t, router, http.MethodPatch, comoJSON(t, cuerpo))
	if rec.Code != http.StatusOK && rec.Code != http.StatusPartialContent {
		t.Fatalf("status = %d; esperado 200 o 206. body=%s", rec.Code, rec.Body.String())
	}
	enviado, ok := peticionPUT(crud, "proveedor")
	if !ok {
		t.Fatal("debe enviarse el PUT del proveedor")
	}
	valor, presente := enviado["DescripcionPortafolio"]
	if !presente || valor != nil {
		t.Errorf("null debe fijar el campo a null en el CRUD; presente=%v valor=%v", presente, valor)
	}
}

// null en un escalar no nullable responde 400 sin ninguna escritura.
func TestNulosEnEscalarNoNullableRechaza400SinEscribir(t *testing.T) {
	crud := NuevoCRUDSimulado(t, existentesBase())
	router := routerProveedor(t, crud)

	cuerpo := cuerpoProveedorNaturalUpdate(101, 201, 301)
	cuerpo["Tercero"].(map[string]any)["NombreCompleto"] = nil
	rec, _ := ejecutar(t, router, http.MethodPatch, comoJSON(t, cuerpo))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d; esperado 400 por null en campo no nullable. body=%s", rec.Code, rec.Body.String())
	}
	if n := escriturasNoGET(crud); n != 0 {
		t.Errorf("un null inválido no debe escribir; peticiones no GET=%d", n)
	}
}

// null en una FK relacional se trata como ausente: no rechaza la escritura.
func TestNulosEnFKRelacionalSeTrataComoAusente(t *testing.T) {
	crud := NuevoCRUDSimulado(t, existentesBase())
	router := routerProveedor(t, crud)

	cuerpo := cuerpoProveedorNaturalUpdate(101, 201, 301)
	cuerpo["Tercero"].(map[string]any)["TipoContribuyenteId"] = nil
	rec, _ := ejecutar(t, router, http.MethodPatch, comoJSON(t, cuerpo))
	if rec.Code == http.StatusBadRequest {
		t.Errorf("null en FK relacional no debe rechazar la escritura. body=%s", rec.Body.String())
	}
}

// Una colección vacía no elimina elementos existentes de su CRUD.
func TestColeccionVaciaNoEliminaElementos(t *testing.T) {
	crud := NuevoCRUDSimulado(t, existentesBase())
	router := routerProveedor(t, crud)

	cuerpo := cuerpoProveedorNaturalUpdate(101, 201, 301)
	cuerpo["Contacto"] = []any{}
	rec, _ := ejecutar(t, router, http.MethodPatch, comoJSON(t, cuerpo))
	if rec.Code != http.StatusOK && rec.Code != http.StatusPartialContent {
		t.Fatalf("status = %d; esperado 200 o 206. body=%s", rec.Code, rec.Body.String())
	}
	for _, c := range crud.Capturas() {
		if c.Metodo == http.MethodDelete && recursoDePath(c.Path) == "contacto" {
			t.Errorf("una colección vacía no debe dar de baja elementos; captura=%+v", c)
		}
	}
}

// EstadoProveedor vacío es ausente: la escritura conserva el estado activo y responde
// 200 o 206, sin crear ni dar de baja ningún estado.
func TestEstadoProveedorVacioEsAusente(t *testing.T) {
	crud := NuevoCRUDSimulado(t, existentesBase())
	router := routerProveedor(t, crud)

	cuerpo := cuerpoProveedorNaturalUpdate(101, 201, 301)
	cuerpo["EstadoProveedor"] = []any{}
	rec, _ := ejecutar(t, router, http.MethodPatch, comoJSON(t, cuerpo))
	if rec.Code != http.StatusOK && rec.Code != http.StatusPartialContent {
		t.Fatalf("status = %d; esperado 200 o 206 con EstadoProveedor vacío. body=%s", rec.Code, rec.Body.String())
	}
	if n := cuentaPOST(crud, "estado_proveedor"); n != 0 {
		t.Errorf("EstadoProveedor vacío no debe crear estado; POST=%d", n)
	}
	for _, c := range crud.Capturas() {
		if c.Metodo == http.MethodDelete && recursoDePath(c.Path) == "estado_proveedor" {
			t.Errorf("EstadoProveedor vacío no debe dar de baja estado; captura=%+v", c)
		}
	}
}
