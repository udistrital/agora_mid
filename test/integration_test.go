package test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// respuestaEnvoltorio es la vista mínima del envoltorio OAS plano que las
// pruebas necesitan para verificar Status, Data y MetaData.Errors.
type respuestaEnvoltorio struct {
	Success  bool           `json:"Success"`
	Status   string         `json:"Status"`
	Data     map[string]any `json:"Data"`
	MetaData *struct {
		Errors []map[string]any `json:"Errors"`
	} `json:"MetaData"`
}

// ejecutar envía una petición al router y decodifica el envoltorio de respuesta.
func ejecutar(t *testing.T, router http.Handler, metodo, cuerpo string) (*httptest.ResponseRecorder, respuestaEnvoltorio) {
	t.Helper()
	req := httptest.NewRequest(metodo, "/v1/proveedor", strings.NewReader(cuerpo))
	req.Header.Set("X-Autor-Id", "9")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var env respuestaEnvoltorio
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("respuesta no es un envoltorio JSON válido: %v; body=%s", err, rec.Body.String())
	}
	return rec, env
}

func comoJSON(t *testing.T, v any) string {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("no se pudo serializar el cuerpo: %v", err)
	}
	return string(data)
}

// peticionesPorMetodo cuenta, por recurso (p. ej. "proveedor"), cuántas
// peticiones de cada método recibió el CRUD simulado, tolerando tanto
// /v1/<recurso> (listas, creación) como /v1/<recurso>/{id} (actualización).
// Sirve para afirmar idempotencia (no duplicar POST).
func peticionesPorMetodo(capturas []PeticionCRUD, recurso string) (post, patch, get int) {
	for _, c := range capturas {
		if recursoDePath(c.Path) != recurso {
			continue
		}
		switch c.Metodo {
		case http.MethodPost:
			post++
		case http.MethodPatch:
			patch++
		case http.MethodGet:
			get++
		}
	}
	return
}

// Creación completa vía POST: el router materializa el subárbol en cascada a
// través de los CRUD simulados y devuelve 200 con los identificadores
// materializados de la raíz y de cada entidad anidada.
func TestIntegracionCreacionCompleta(t *testing.T) {
	crud := NuevoCRUDSimulado(t, nil)
	router := routerProveedor(t, crud)

	rec, env := ejecutar(t, router, http.MethodPost, comoJSON(t, cuerpoProveedorNatural()))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; esperado 200. body=%s", rec.Code, rec.Body.String())
	}
	if env.Status != "200" || !env.Success {
		t.Errorf("envoltorio inesperado: %+v", env)
	}
	// La raíz materializó su Id (rango proveedor = 100+).
	if id, _ := env.Data["Id"].(float64); int(id) != 101 {
		t.Errorf("el proveedor raíz debe materializar Id=101; data=%+v", env.Data)
	}
	// FK ascendente: el tercero es dueño de su propio Id (rango 200+), distinto del
	// proveedor, y el proveedor lo referencia por su FK TerceroId.
	tercero, _ := env.Data["Tercero"].(map[string]any)
	if tercero == nil {
		t.Fatalf("falta el Tercero materializado; data=%+v", env.Data)
	}
	if id, _ := tercero["Id"].(float64); int(id) != 201 {
		t.Errorf("Tercero (fk_inbound) debe materializar su propio Id=201; tercero=%+v", tercero)
	}
	if autorModificacion, _ := tercero["AutorModificacionId"].(float64); int(autorModificacion) != 9 {
		t.Errorf("Tercero debe heredar AutorModificacionId=9 del proveedor; tercero=%+v", tercero)
	}

	// La FK ascendente se propaga preservando su tipo numérico (no texto).
	terceroIDRef, ok := env.Data["TerceroId"].(float64)
	if !ok {
		t.Errorf("TerceroId debe ser numérico, no %T; data=%+v", env.Data["TerceroId"], env.Data)
	} else if int(terceroIDRef) != 201 {
		t.Errorf("el proveedor debe referenciar al tercero por TerceroId=201; data=%+v", env.Data)
	}

	// FK compartida: el proveedor_natural no tiene un Id propio, sino que hereda el del
	// proveedor (rango 100-). El proveedor_natural.Id = proveedor.Id = 101.
	proveedorNatural, _ := env.Data["ProveedorNatural"].(map[string]any)
	if proveedorNatural == nil {
		t.Fatalf("falta el ProveedorNatural materializado; data=%+v", env.Data)
	}
	if id, _ := proveedorNatural["Id"].(float64); int(id) != 101 {
		t.Errorf("ProveedorNatural debe heredar Id=101 del proveedor; proveedor_natural=%+v", proveedorNatural)
	}
	if autorModificacion, _ := proveedorNatural["AutorModificacionId"].(float64); int(autorModificacion) != 9 {
		t.Errorf("ProveedorNatural debe heredar AutorModificacionId=9 del proveedor; proveedor_natural=%+v", proveedorNatural)
	}

	// FK descendiente: los contactos son dueños de su propio Id (rango 600+), destinto
	// del proveedor y referencian al proveedor por su FK ProveedorId.
	contacto, _ := env.Data["Contacto"].([]any)
	if contacto == nil {
		t.Fatalf("falta el Contacto materializado; data=%+v", env.Data)
	}
	for _, c := range contacto {
		c, _ := c.(map[string]any)
		if c == nil {
			t.Fatalf("Algún contacto no es un objeto; contacto=%+v", contacto)
		}
		if id, _ := c["Id"].(float64); int(id) < 601 || int(id) > 699 {
			t.Errorf("Los contactos deben materializar su propio Id=601; contacto=%+v", contacto)
		}
		// La FK descendente ProveedorId se propaga preservando su tipo numérico.
		provIDRef, ok := c["ProveedorId"].(float64)
		if !ok {
			t.Errorf("Contacto.ProveedorId debe ser numérico, no %T; contacto=%+v", c["ProveedorId"], c)
		} else if int(provIDRef) != 101 {
			t.Errorf("Los contactos deben referenciar al proveedor por ProveedorId=101; contacto=%+v", contacto)
		}
		if autorModificacion, _ := c["AutorModificacionId"].(float64); int(autorModificacion) != 9 {
			t.Errorf("Contacto debe heredar AutorModificacionId=9 del proveedor; contacto=%+v", c)
		}
	}

	// El tercero se materializó ANTES que el proveedor (orden fk_inbound).
	capturas := crud.Capturas()
	idxTercero, idxProveedor := -1, -1
	for i, c := range capturas {
		if c.Metodo == http.MethodPost && recursoDePath(c.Path) == "tercero" && idxTercero < 0 {
			idxTercero = i
		}
		if c.Metodo == http.MethodPost && recursoDePath(c.Path) == "proveedor" && idxProveedor < 0 {
			idxProveedor = i
		}
	}
	if idxTercero < 0 || idxProveedor < 0 || idxTercero > idxProveedor {
		t.Errorf("el tercero (fk_inbound) debe crearse antes que el proveedor; capturas=%+v", capturas)
	}

	// Se creó exactamente un proveedor (un solo POST a /v1/proveedor/).
	post, _, _ := peticionesPorMetodo(capturas, "proveedor")
	if post != 1 {
		t.Errorf("se esperaba 1 POST de proveedor; hubo %d. capturas=%+v", post, capturas)
	}
	// Nunca se materializó la rama jurídica (podada para un natural).
	for _, c := range crud.Capturas() {
		if strings.HasPrefix(c.Path, "/v1/proveedor_juridico") &&
			(c.Metodo == http.MethodPost || c.Metodo == http.MethodPatch) {
			t.Errorf("no debe materializar la rama jurídica para un natural; capturas=%+v", crud.Capturas())
		}
	}
	// El autor de auditoría viaja al CRUD desde la cabecera X-Autor-Id.
	var vioAutor bool
	for _, c := range crud.Capturas() {
		if c.Metodo == http.MethodPost {
			if a, _ := c.Cuerpo["AutorModificacionId"].(float64); int(a) == 9 {
				vioAutor = true
			}
		}
	}
	if !vioAutor {
		t.Errorf("el autor 9 debe inyectarse en los POST; capturas=%+v", crud.Capturas())
	}
}

// Reconciliación vía PATCH: cuando el proveedor ya existe por su tercero_id, la
// operación lo reutiliza (PATCH) en vez de crear un duplicado. PATCH es
// idempotente.
func TestIntegracionReconciliacionPatchIdempotente(t *testing.T) {
	// El proveedor ya existe por TerceroId=25.
	crud := NuevoCRUDSimulado(t, map[string][]map[string]any{
		"GET /v1/proveedor": {{"Id": float64(101), "TerceroId": float64(25)}},
	})
	router := routerProveedor(t, crud)

	rec, env := ejecutar(t, router, http.MethodPatch, comoJSON(t, cuerpoProveedorNaturalUpdate(101, 201, 301)))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; esperado 200. body=%s", rec.Code, rec.Body.String())
	}
	if id, _ := env.Data["Id"].(float64); int(id) != 101 {
		t.Errorf("debe reutilizar el proveedor existente 101; data=%+v", env.Data)
	}

	post, patch, _ := peticionesPorMetodo(crud.Capturas(), "proveedor")
	if post != 0 {
		t.Errorf("no debe crear un segundo proveedor; POST=%d. capturas=%+v", post, crud.Capturas())
	}
	if patch != 1 {
		t.Errorf("debe actualizar el proveedor existente por PATCH; PATCH=%d. capturas=%+v", patch, crud.Capturas())
	}
}

// Continuación tras creación parcial: un POST previo dejó el proveedor y su
// tercero creados pero no el resto. Reenviar el mismo cuerpo con PATCH reutiliza
// lo existente por clave natural / Id y solo crea lo faltante, sin duplicar.
func TestIntegracionContinuacionTrasCreacionParcial(t *testing.T) {
	// Estado parcial: el proveedor (por tercero_id) y su datos_identificacion
	// (por clave natural numero+tipo_documento_id) ya existen.
	crud := NuevoCRUDSimulado(t, map[string][]map[string]any{
		"GET /v1/proveedor":            {{"Id": float64(101), "TerceroId": float64(25)}},
		"GET /v1/datos_identificacion": {{"Id": float64(301), "TerceroId": float64(101)}},
	})
	router := routerProveedor(t, crud)

	rec, env := ejecutar(t, router, http.MethodPatch, comoJSON(t, cuerpoProveedorNaturalUpdate(101, 201, 301)))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; esperado 200. body=%s", rec.Code, rec.Body.String())
	}
	if id, _ := env.Data["Id"].(float64); int(id) != 101 {
		t.Errorf("debe continuar sobre el proveedor 101; data=%+v", env.Data)
	}

	// No se duplica ni el proveedor ni los datos de identificación ya existentes.
	postProv, patchProv, _ := peticionesPorMetodo(crud.Capturas(), "proveedor")
	if postProv != 0 || patchProv != 1 {
		t.Errorf("proveedor: se esperaba 0 POST / 1 PATCH; POST=%d PATCH=%d", postProv, patchProv)
	}
	postDI, patchDI, _ := peticionesPorMetodo(crud.Capturas(), "datos_identificacion")
	if postDI != 0 {
		t.Errorf("datos_identificacion existente no debe duplicarse; POST=%d. capturas=%+v", postDI, crud.Capturas())
	}
	if patchDI != 1 {
		t.Errorf("datos_identificacion existente debe actualizarse por PATCH; PATCH=%d", patchDI)
	}
	// El contacto, que no existía, sí se crea.
	postContacto, _, _ := peticionesPorMetodo(crud.Capturas(), "contacto")
	if postContacto != 1 {
		t.Errorf("el contacto faltante debe crearse; POST=%d. capturas=%+v", postContacto, crud.Capturas())
	}
}
