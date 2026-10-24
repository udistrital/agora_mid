package test

import (
	"net/http"
	"testing"
)

// TestE2EProveedorCreacionYActualizacion ejercita un flujo e2e completo del
// subárbol del proveedor sobre un único router y CRUD simulado: primero crea el
// subárbol (`POST`) y luego lo actualiza (`PATCH`) reutilizando los
// identificadores materializados que devolvió la creación, tal como lo haría un
// cliente real que continúa sobre lo ya creado.
//
// Actores: el cliente del API (que envía el cuerpo anidado y el autor por
// cabecera), el middleware (router + servicio + motor de escritura) y los CRUD
// dueños simulados (terceros y gestión de personas).
func TestE2EProveedorCreacionYActualizacion(t *testing.T) {
	crud := NuevoCRUDSimulado(t, nil)
	router := routerProveedor(t, crud)

	// --- Fase 1: creación (POST) del subárbol completo ---
	recCrear, envCrear := ejecutar(t, router, http.MethodPost, comoJSON(t, cuerpoProveedorNatural()))

	if recCrear.Code != http.StatusOK {
		t.Fatalf("creación: status = %d; esperado 200. body=%s", recCrear.Code, recCrear.Body.String())
	}
	if envCrear.Status != "200" || !envCrear.Success {
		t.Fatalf("creación: envoltorio inesperado: %+v", envCrear)
	}

	// Extraer los identificadores materializados de la creación para continuar.
	provID := enteroDe(envCrear.Data["Id"])
	if provID <= 0 {
		t.Fatalf("creación: el proveedor debe materializar un Id positivo; data=%+v", envCrear.Data)
	}
	tercero, _ := envCrear.Data["Tercero"].(map[string]any)
	if tercero == nil {
		t.Fatalf("creación: falta el Tercero materializado; data=%+v", envCrear.Data)
	}
	terceroID := enteroDe(tercero["Id"])
	if terceroID <= 0 || terceroID == provID {
		t.Fatalf("creación: el tercero (fk_inbound) debe tener su propio Id distinto del proveedor; tercero=%+v prov=%d", tercero, provID)
	}
	// El proveedor referencia al tercero por su FK ascendente, propagada con su
	// tipo numérico (no texto) e igual al Id del tercero materializado.
	terceroIDRef, ok := envCrear.Data["TerceroId"].(float64)
	if !ok {
		t.Errorf("creación: proveedor.TerceroId debe ser numérico, no %T; data=%+v",
			envCrear.Data["TerceroId"], envCrear.Data)
	} else if int(terceroIDRef) != enteroDe(tercero["Id"]) {
		t.Errorf("creación: proveedor.TerceroId debe igualar tercero.Id; data=%+v", envCrear.Data)
	}
	di := primerElemento(tercero["DatosIdentificacion"])
	diID := enteroDe(di["Id"])
	if diID <= 0 {
		t.Fatalf("creación: falta el Id de datos_identificacion; tercero=%+v", tercero)
	}

	// --- Fase 2: actualización (PATCH) reutilizando los Ids materializados ---
	// El cuerpo de actualización identifica cada entidad por su Id y omite los
	// inmutables; es idempotente y no debe duplicar registros.
	recAct, envAct := ejecutar(t, router, http.MethodPatch,
		comoJSON(t, cuerpoProveedorNaturalUpdate(provID, terceroID, diID)))

	if recAct.Code != http.StatusOK {
		t.Fatalf("actualización: status = %d; esperado 200. body=%s", recAct.Code, recAct.Body.String())
	}
	if enteroDe(envAct.Data["Id"]) != provID {
		t.Errorf("actualización: debe operar sobre el mismo proveedor %d; data=%+v", provID, envAct.Data)
	}

	// La actualización reutiliza las entidades por Id (PATCH), sin crear
	// duplicados del proveedor, el tercero ni los datos de identificación.
	capturas := crud.Capturas()

	// Conteos globales a lo largo de las dos fases: una creación y una
	// actualización por cada entidad del núcleo con Id estable.
	postProv, patchProv, _ := peticionesPorMetodo(capturas, "proveedor")
	if postProv != 1 || patchProv != 1 {
		t.Errorf("proveedor: se esperaba 1 POST (crear) y 1 PATCH (actualizar); POST=%d PATCH=%d", postProv, patchProv)
	}
	postTercero, patchTercero, _ := peticionesPorMetodo(capturas, "tercero")
	if postTercero != 1 || patchTercero != 1 {
		t.Errorf("tercero: se esperaba 1 POST y 1 PATCH; POST=%d PATCH=%d", postTercero, patchTercero)
	}
	postDI, patchDI, _ := peticionesPorMetodo(capturas, "datos_identificacion")
	if postDI != 1 || patchDI != 1 {
		t.Errorf("datos_identificacion: se esperaba 1 POST y 1 PATCH; POST=%d PATCH=%d", postDI, patchDI)
	}
}

// enteroDe normaliza a int un valor numérico decodificado desde JSON.
func enteroDe(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	default:
		return 0
	}
}

// primerElemento devuelve el primer elemento de una colección JSON como mapa, o
// un mapa vacío si no hay ninguno.
func primerElemento(v any) map[string]any {
	lista, ok := v.([]any)
	if !ok || len(lista) == 0 {
		return map[string]any{}
	}
	m, _ := lista[0].(map[string]any)
	if m == nil {
		return map[string]any{}
	}
	return m
}
