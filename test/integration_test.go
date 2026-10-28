package test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/udistrital/agora_mid/config"
)

// respuestaEnvoltorio es la vista mínima del envoltorio OAS plano que las
// pruebas necesitan para verificar Status, Data y MetaData.Errors.
type respuestaEnvoltorio struct {
	Success  bool           `json:"Success"`
	Status   string         `json:"Status"`
	Message  string         `json:"Message"`
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

// Creación completa vía POST con una FK ascendente desactualizada: el fixture
// reenvía TerceroId=25 en un proveedor NUEVO, mientras el tercero se materializa
// con su propio Id autoritativo. El motor canoniza esa diferencia en la entidad
// nueva y responde 206 con un aviso field_normalization, conservando en Data el
// subárbol materializado con las llaves autoritativas y su tipo numérico.
func TestIntegracionCreacionCompleta(t *testing.T) {
	crud := NuevoCRUDSimulado(t, nil)
	router := routerProveedor(t, crud)

	rec, env := ejecutar(t, router, http.MethodPost, comoJSON(t, cuerpoProveedorNatural()))

	// La FK ascendente reenviada (TerceroId=25) difiere del Id autoritativo del
	// tercero recién materializado: una entidad nueva canoniza y avisa (206).
	if rec.Code != http.StatusPartialContent {
		t.Fatalf("status = %d; esperado 206 por canonización de la FK ascendente. body=%s", rec.Code, rec.Body.String())
	}
	if env.Status != "206" || !env.Success {
		t.Errorf("envoltorio inesperado: %+v", env)
	}
	if env.MetaData == nil || len(env.MetaData.Errors) == 0 {
		t.Fatalf("se esperaba MetaData.Errors con el aviso de normalización; env=%+v", env)
	}
	var vioNormalizacion bool
	for _, e := range env.MetaData.Errors {
		if kind, _ := e["kind"].(string); kind == "field_normalization" {
			vioNormalizacion = true
		}
	}
	if !vioNormalizacion {
		t.Errorf("se esperaba un aviso con Kind field_normalization; errors=%+v", env.MetaData.Errors)
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

// ejecutarConsulta envía un POST a /v1/proveedor/consulta con el cuerpo JSON
// {id, fields} y decodifica el envoltorio de respuesta. Es la ruta de lectura por
// cuerpo (handler.Consultar); se usa para ejercitar la validación de fields sin
// depender del GET, que será retirado.
func ejecutarConsulta(t *testing.T, router http.Handler, id int, fields []string) (*httptest.ResponseRecorder, respuestaEnvoltorio) {
	t.Helper()
	cuerpo := comoJSON(t, map[string]any{"id": id, "fields": fields})
	req := httptest.NewRequest(http.MethodPost, "/v1/proveedor/consulta", strings.NewReader(cuerpo))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var env respuestaEnvoltorio
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("respuesta no es un envoltorio JSON válido: %v; body=%s", err, rec.Body.String())
	}
	return rec, env
}

// Negación tipo allowed_fields por lista negra: pedir en fields un campo que SÍ
// está en la configuración (allowed_fields de la relación en el YAML real) pero
// que la lista negra veta se rechaza con 400 y el mismo mensaje observable que
// un campo no permitido, sin distinguir el caso ni consultar el CRUD dueño
// (política anti-sondeo + fallo temprano). Se ejercita por POST
// /v1/proveedor/consulta (lectura por cuerpo).
func TestIntegracionFieldVetadoSeRechazaComoNoPermitido(t *testing.T) {
	// Tercero.NombreCompleto SÍ está declarado en allowed_fields de la relación
	// Tercero (terceros/tercero) en config/relaciones_proveedor.yaml. La lista
	// negra lo veta de forma transversal.
	blacklist := config.GlobalBlacklist{
		"terceros": {
			"tercero": {"NombreCompleto"},
		},
	}

	crud := NuevoCRUDSimulado(t, nil)
	router := routerProveedorConBlacklist(t, crud, blacklist)

	// Caso A: campo configurado (allowed_fields) pero vetado por la lista negra.
	recVetado, envVetado := ejecutarConsulta(t, router, 101, []string{"Tercero.NombreCompleto"})
	if recVetado.Code != http.StatusBadRequest {
		t.Fatalf("un campo vetado debe rechazarse con 400; status=%d body=%s", recVetado.Code, recVetado.Body.String())
	}

	// Caso B: campo realmente no permitido (fuera de allowed_fields), sin lista
	// negra, sobre el mismo router/configuración.
	routerSinBL := routerProveedor(t, crud)
	recNoPermitido, envNoPermitido := ejecutarConsulta(t, routerSinBL, 101, []string{"Tercero.CampoInexistente"})
	if recNoPermitido.Code != http.StatusBadRequest {
		t.Fatalf("un campo no permitido debe rechazarse con 400; status=%d body=%s", recNoPermitido.Code, recNoPermitido.Body.String())
	}

	// Ambos caminos deben producir el mismo código y el mismo mensaje observable:
	// la negación por lista negra es indistinguible de la negación por
	// allowed_fields, salvo el nombre del campo citado.
	if recVetado.Code != recNoPermitido.Code {
		t.Errorf("el status debe coincidir; vetado=%d no_permitido=%d", recVetado.Code, recNoPermitido.Code)
	}
	if envVetado.Status != "400" || envNoPermitido.Status != "400" {
		t.Errorf("ambos envoltorios deben reportar 400; vetado=%q no_permitido=%q", envVetado.Status, envNoPermitido.Status)
	}
	if envVetado.Success || envNoPermitido.Success {
		t.Errorf("ambas respuestas deben ser no exitosas; vetado=%v no_permitido=%v", envVetado.Success, envNoPermitido.Success)
	}

	// El mensaje del campo vetado debe usar la misma plantilla "no está permitido"
	// que el no permitido (sin revelar que el campo existe o que está vetado).
	if !strings.Contains(envVetado.Message, "no está permitido") {
		t.Errorf("el mensaje del campo vetado debe usar la plantilla de no permitido; mensaje=%q", envVetado.Message)
	}
	if !strings.Contains(envVetado.Message, "Tercero.NombreCompleto") {
		t.Errorf("el mensaje debe citar la ruta solicitada; mensaje=%q", envVetado.Message)
	}

	// Fallo temprano: el rechazo ocurre antes de consultar el CRUD dueño, de modo
	// que una ruta vetada no dispara ninguna petición de lectura del proveedor.
	for _, c := range crud.Capturas() {
		if recursoDePath(c.Path) == "proveedor" {
			t.Errorf("un fields vetado no debe consultar el CRUD; captura=%+v", c)
		}
	}
}

// --- Grupo 3.2: reenvío de Data completo con la FK ascendente ausente, null o
// coincidente -> 200 sin aviso de normalización ---

// Con la FK ascendente (TerceroId) AUSENTE en un proveedor nuevo, el motor la
// completa con el Id autoritativo del tercero y responde 200 sin aviso.
func TestIntegracionFKAscendenteAusente200(t *testing.T) {
	crud := NuevoCRUDSimulado(t, nil)
	router := routerProveedor(t, crud)

	rec, env := ejecutar(t, router, http.MethodPost,
		comoJSON(t, cuerpoProveedorNaturalConTerceroID(nil, false)))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; esperado 200 (FK ausente se completa sin aviso). body=%s", rec.Code, rec.Body.String())
	}
	if env.MetaData != nil && len(env.MetaData.Errors) > 0 {
		t.Errorf("no debe haber avisos al completar una FK ausente; errors=%+v", env.MetaData.Errors)
	}
	// La FK quedó completada con el Id autoritativo del tercero, con tipo numérico.
	tercero, _ := env.Data["Tercero"].(map[string]any)
	if tercero == nil {
		t.Fatalf("falta el Tercero materializado; data=%+v", env.Data)
	}
	refID, ok := env.Data["TerceroId"].(float64)
	if !ok || int(refID) != enteroDe(tercero["Id"]) {
		t.Errorf("TerceroId debe completarse numérico con el Id del tercero; data=%+v", env.Data)
	}
}

// Con la FK ascendente (TerceroId) en NULL, el motor la trata como ausente, la
// completa con el Id autoritativo y responde 200 sin aviso.
func TestIntegracionFKAscendenteNull200(t *testing.T) {
	crud := NuevoCRUDSimulado(t, nil)
	router := routerProveedor(t, crud)

	rec, env := ejecutar(t, router, http.MethodPost,
		comoJSON(t, cuerpoProveedorNaturalConTerceroID(nil, true)))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; esperado 200 (FK null se completa sin aviso). body=%s", rec.Code, rec.Body.String())
	}
	if env.MetaData != nil && len(env.MetaData.Errors) > 0 {
		t.Errorf("no debe haber avisos al tratar null como ausencia; errors=%+v", env.MetaData.Errors)
	}
	tercero, _ := env.Data["Tercero"].(map[string]any)
	if tercero == nil {
		t.Fatalf("falta el Tercero materializado; data=%+v", env.Data)
	}
	if refID, ok := env.Data["TerceroId"].(float64); !ok || int(refID) != enteroDe(tercero["Id"]) {
		t.Errorf("TerceroId null debe completarse numérico con el Id del tercero; data=%+v", env.Data)
	}
}

// Reenvío idempotente del Data completo: tras un POST, el Data materializado se
// reenvía por PATCH con todas sus llaves relacionales; como la FK ascendente
// coincide con el Id autoritativo, la operación responde 200 sin aviso y reutiliza
// las entidades por Id sin duplicarlas.
func TestIntegracionReenvioDataCoincide200(t *testing.T) {
	// El proveedor ya existe por su TerceroId para que el reenvío lo reutilice por
	// PATCH en lugar de crear uno nuevo.
	crud := NuevoCRUDSimulado(t, map[string][]map[string]any{
		"GET /v1/proveedor": {{"Id": float64(101), "TerceroId": float64(201)}},
	})
	router := routerProveedor(t, crud)

	// Cuerpo que reenvía el Data materializado: proveedor y tercero con Id propio y
	// la FK ascendente coincidente (TerceroId == tercero.Id == 201).
	cuerpo := map[string]any{
		"Id":                    float64(101),
		"TerceroId":             float64(201),
		"TipoRegistroId":        float64(tipoRegistroNatural),
		"DescripcionPortafolio": "Servicios de consultoría",
		"Tercero": map[string]any{
			"Id":             float64(201),
			"NombreCompleto": "Ana Pérez",
		},
		"ProveedorNatural": map[string]any{"Id": float64(101), "PerfilDeclarado": float64(3)},
	}

	rec, env := ejecutar(t, router, http.MethodPatch, comoJSON(t, cuerpo))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; esperado 200 (FK coincidente, sin aviso). body=%s", rec.Code, rec.Body.String())
	}
	if env.MetaData != nil && len(env.MetaData.Errors) > 0 {
		t.Errorf("un reenvío coincidente no debe producir avisos; errors=%+v", env.MetaData.Errors)
	}
	if refID, ok := env.Data["TerceroId"].(float64); !ok || int(refID) != 201 {
		t.Errorf("TerceroId coincidente debe conservarse numérico 201; data=%+v", env.Data)
	}
	// Idempotencia: no se crea un segundo proveedor; se reutiliza por PATCH.
	post, patch, _ := peticionesPorMetodo(crud.Capturas(), "proveedor")
	if post != 0 || patch != 1 {
		t.Errorf("proveedor: se esperaba 0 POST / 1 PATCH; POST=%d PATCH=%d", post, patch)
	}
}

// --- Grupo 3.3: conflicto de entidad existente -> 400; fallo de relación
// tolerada (notify) -> 206 con Kind relation ---

// Un proveedor EXISTENTE (con Id propio) cuya FK ascendente reenviada difiere del
// Id del tercero referenciado es un conflicto fk_inbound: se rechaza con 400 sin
// reasignar ni materializar la operación incompatible.
func TestIntegracionConflictoFKAscendenteExistente400(t *testing.T) {
	crud := NuevoCRUDSimulado(t, map[string][]map[string]any{
		"GET /v1/proveedor": {{"Id": float64(101), "TerceroId": float64(201)}},
	})
	router := routerProveedor(t, crud)

	// El tercero referenciado se materializa con su propio Id autoritativo; la FK
	// local reenviada (TerceroId=999) diverge en un nodo con Id propio -> conflicto.
	cuerpo := map[string]any{
		"Id":                    float64(101),
		"TerceroId":             float64(999),
		"TipoRegistroId":        float64(tipoRegistroNatural),
		"DescripcionPortafolio": "Servicios de consultoría",
		"Tercero": map[string]any{
			"Id":             float64(201),
			"NombreCompleto": "Ana Pérez",
		},
		"ProveedorNatural": map[string]any{"Id": float64(101), "PerfilDeclarado": float64(3)},
	}

	rec, env := ejecutar(t, router, http.MethodPatch, comoJSON(t, cuerpo))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d; esperado 400 por conflicto fk_inbound existente. body=%s", rec.Code, rec.Body.String())
	}
	if env.Status != "400" || env.Success {
		t.Errorf("envoltorio inesperado para un conflicto: %+v", env)
	}
	// No debe materializarse el proveedor en conflicto (ni POST ni PATCH).
	post, patch, _ := peticionesPorMetodo(crud.Capturas(), "proveedor")
	if post != 0 || patch != 0 {
		t.Errorf("no debe materializarse el proveedor en conflicto; POST=%d PATCH=%d capturas=%+v", post, patch, crud.Capturas())
	}
}

// Fallo de una relación tolerada (notify): con la FK ascendente ausente (para
// aislar el aviso), el CRUD de contacto responde 503; el motor conserva lo ya
// materializado y responde 206 con un aviso Kind relation, sin exponer el error
// interno del CRUD.
func TestIntegracionFalloRelacionToleradaNotify206(t *testing.T) {
	crud := NuevoCRUDSimulado(t, nil)
	crud.FallarPostDe("contacto", http.StatusServiceUnavailable)
	router := routerProveedor(t, crud)

	rec, env := ejecutar(t, router, http.MethodPost,
		comoJSON(t, cuerpoProveedorNaturalConTerceroID(nil, false)))

	if rec.Code != http.StatusPartialContent {
		t.Fatalf("status = %d; esperado 206 por fallo tolerado de contacto. body=%s", rec.Code, rec.Body.String())
	}
	if env.MetaData == nil || len(env.MetaData.Errors) == 0 {
		t.Fatalf("se esperaba MetaData.Errors con el aviso de relación; env=%+v", env)
	}
	var vioRelation bool
	for _, e := range env.MetaData.Errors {
		if kind, _ := e["kind"].(string); kind == "relation" {
			vioRelation = true
		}
		// El aviso no debe exponer el cuerpo interno del CRUD.
		if msg, _ := e["message"].(string); strings.Contains(msg, "servicio no disponible") {
			t.Errorf("el aviso no debe exponer el error interno del CRUD; aviso=%+v", e)
		}
	}
	if !vioRelation {
		t.Errorf("se esperaba un aviso con Kind relation; errors=%+v", env.MetaData.Errors)
	}
	// El proveedor sí se materializó pese al fallo del contacto (creación parcial).
	post, _, _ := peticionesPorMetodo(crud.Capturas(), "proveedor")
	if post != 1 {
		t.Errorf("el proveedor debe materializarse pese al fallo tolerado; POST=%d", post)
	}
}

// Un conflicto de FK ascendente (TerceroId) en un proveedor EXISTENTE responde
// 400 sin dejar escrituras parciales: ni el tercero (entidad referenciada por
// fk_inbound) ni el proveedor deben recibir POST ni PATCH. Solo se permiten GET
// (recuperación por lectura).
func TestIntegracionConflictoFKAscendenteNoDejaEscriturasParciales(t *testing.T) {
	// El proveedor ya existe por su TerceroId=201 (entidad existente). El tercero
	// referenciado se recupera/materializa con su propio Id autoritativo (201); la
	// FK local reenviada (TerceroId=999) diverge en un nodo con Id propio ->
	// conflicto que debe responder 400 sin escrituras.
	crud := NuevoCRUDSimulado(t, map[string][]map[string]any{
		"GET /v1/proveedor": {{"Id": float64(101), "TerceroId": float64(201)}},
		"GET /v1/tercero":   {{"Id": float64(201), "NombreCompleto": "Ana Pérez"}},
	})
	router := routerProveedor(t, crud)

	cuerpo := map[string]any{
		"Id":                    float64(101),
		"TerceroId":             float64(999),
		"TipoRegistroId":        float64(tipoRegistroNatural),
		"DescripcionPortafolio": "Servicios de consultoría",
		"Tercero": map[string]any{
			"Id":             float64(201),
			"NombreCompleto": "Ana Pérez",
		},
		"ProveedorNatural": map[string]any{"Id": float64(101), "PerfilDeclarado": float64(3)},
	}

	rec, env := ejecutar(t, router, http.MethodPatch, comoJSON(t, cuerpo))

	// El conflicto responde 400.
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d; esperado 400 por conflicto fk_inbound existente. body=%s", rec.Code, rec.Body.String())
	}
	if env.Status != "400" || env.Success {
		t.Errorf("envoltorio inesperado para un conflicto: %+v", env)
	}

	// Cero escrituras en tercero y en proveedor (GET permitidos).
	postTercero, patchTercero, _ := peticionesPorMetodo(crud.Capturas(), "tercero")
	if postTercero != 0 || patchTercero != 0 {
		t.Errorf("un 400 no debe escribir en tercero; POST=%d PATCH=%d. capturas=%+v",
			postTercero, patchTercero, crud.Capturas())
	}
	postProveedor, patchProveedor, _ := peticionesPorMetodo(crud.Capturas(), "proveedor")
	if postProveedor != 0 || patchProveedor != 0 {
		t.Errorf("un 400 no debe escribir en proveedor; POST=%d PATCH=%d. capturas=%+v",
			postProveedor, patchProveedor, crud.Capturas())
	}
}
