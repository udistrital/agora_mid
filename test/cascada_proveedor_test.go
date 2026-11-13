package test

import (
	"net/http"
	"strings"
	"testing"
)

// tipoRegistroJuridicoCascada es un TipoProveedor distinto de tipoRegistroNatural,
// que el servicio de escritura trata como persona jurídica.
const tipoRegistroJuridicoCascada = 2

// Un registro de tipo jurídico materializa solo la rama ProveedorJuridico, con PK
// compartida con el proveedor, y no crea ProveedorNatural.
func TestIntegracionCreacionJuridicoMaterializaSoloRamaJuridica(t *testing.T) {
	crud := NuevoCRUDSimulado(t, nil)
	router := routerProveedor(t, crud)

	cuerpo := map[string]any{
		"TipoProveedor":         float64(tipoRegistroJuridicoCascada),
		"DescripcionPortafolio": "Servicios de consultoría",
		"Tercero": map[string]any{
			"NombreCompleto": "Empresa Demo SAS",
		},
		"ProveedorJuridico": map[string]any{},
	}

	rec, env := ejecutar(t, router, http.MethodPost, comoJSON(t, cuerpo))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; esperado 200. body=%s", rec.Code, rec.Body.String())
	}

	juridico, _ := env.Data["ProveedorJuridico"].(map[string]any)
	if juridico == nil {
		t.Fatalf("falta el ProveedorJuridico materializado; data=%+v", env.Data)
	}
	if id, _ := juridico["Id"].(float64); int(id) != 101 {
		t.Errorf("ProveedorJuridico debe heredar Id=101 del proveedor (PK compartida); juridico=%+v", juridico)
	}
	if natural, ok := env.Data["ProveedorNatural"]; ok && natural != nil {
		t.Errorf("un registro jurídico no debe devolver ProveedorNatural; data=%+v", env.Data)
	}

	capturas := crud.Capturas()
	if post, _, _ := peticionesPorMetodo(capturas, "proveedor_natural"); post != 0 {
		t.Errorf("no debe crear proveedor_natural para un registro jurídico; POST=%d", post)
	}
	post, patch, _ := peticionesPorMetodo(capturas, "proveedor_juridico")
	if post+patch != 1 {
		t.Errorf("se esperaba 1 escritura a proveedor_juridico; POST=%d PATCH=%d. capturas=%+v", post, patch, capturas)
	}
}

// Una cuenta bancaria anidada se materializa con su propio Id y referencia al
// proveedor por ProveedorId, preservando el tipo numérico de la FK.
func TestIntegracionCreacionCuentaBancariaReferenciaProveedorPorFK(t *testing.T) {
	crud := NuevoCRUDSimulado(t, nil)
	router := routerProveedor(t, crud)

	cuerpo := map[string]any{
		"TipoProveedor":         float64(tipoRegistroNatural),
		"DescripcionPortafolio": "Servicios de consultoría",
		"Tercero": map[string]any{
			"NombreCompleto": "Ana Pérez",
		},
		"PerfilProveedor": []any{map[string]any{"PerfilId": float64(3)}},
		"CuentaBancaria": []any{
			map[string]any{
				"EntidadBancariaId": float64(4),
				"TipoCuentaId":      float64(2),
				"NumeroCuenta":      "001-123456",
			},
		},
	}

	rec, env := ejecutar(t, router, http.MethodPost, comoJSON(t, cuerpo))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; esperado 200. body=%s", rec.Code, rec.Body.String())
	}

	cuentas, _ := env.Data["CuentaBancaria"].([]any)
	if len(cuentas) != 1 {
		t.Fatalf("se esperaba 1 CuentaBancaria materializada; data=%+v", env.Data)
	}
	cuenta, _ := cuentas[0].(map[string]any)
	if id, _ := cuenta["Id"].(float64); int(id) != 701 {
		t.Errorf("CuentaBancaria debe materializar su propio Id=701; cuenta=%+v", cuenta)
	}
	if prov, _ := cuenta["ProveedorId"].(float64); int(prov) != 101 {
		t.Errorf("CuentaBancaria debe referenciar al proveedor por ProveedorId=101 numérico; cuenta=%+v", cuenta)
	}

	var enviada map[string]any
	for _, c := range crud.Capturas() {
		if c.Metodo == http.MethodPost && recursoDePath(c.Path) == "cuenta_bancaria" {
			enviada = c.Cuerpo
		}
	}
	if enviada == nil {
		t.Fatalf("no se recibió el POST de cuenta_bancaria; capturas=%+v", crud.Capturas())
	}
	if prov, _ := enviada["ProveedorId"].(float64); int(prov) != 101 {
		t.Errorf("el POST de cuenta_bancaria debe enviar ProveedorId=101; cuerpo=%+v", enviada)
	}
	if numero, _ := enviada["NumeroCuenta"].(string); numero != "001-123456" {
		t.Errorf("el POST de cuenta_bancaria debe enviar NumeroCuenta; cuerpo=%+v", enviada)
	}
}

// La lista negra de lectura no recorta la escritura: UsuarioModificacionId enviado en
// el cuerpo llega al CRUD dueño de la entidad.
func TestIntegracionEscrituraConservaUsuarioModificacionIdDelCuerpo(t *testing.T) {
	crud := NuevoCRUDSimulado(t, nil)
	router := routerProveedor(t, crud)

	cuerpo := cuerpoProveedorNatural()
	contactos, ok := cuerpo["Contacto"].([]any)
	if !ok || len(contactos) == 0 {
		t.Fatalf("el cuerpo de prueba debe tener Contacto como lista no vacía; cuerpo=%+v", cuerpo)
	}
	contacto, ok := contactos[0].(map[string]any)
	if !ok {
		t.Fatalf("el primer Contacto debe ser un objeto; contacto=%+v", contactos[0])
	}
	contacto["UsuarioModificacionId"] = float64(9)

	rec, _ := ejecutar(t, router, http.MethodPost, comoJSON(t, cuerpo))
	if rec.Code != http.StatusOK && rec.Code != http.StatusPartialContent {
		t.Fatalf("status = %d; esperado 200 o 206. body=%s", rec.Code, rec.Body.String())
	}

	enviado := false
	for _, c := range crud.Capturas() {
		if recursoDePath(c.Path) != "contacto" || c.Metodo != http.MethodPost {
			continue
		}
		if v, _ := c.Cuerpo["UsuarioModificacionId"].(float64); int(v) == 9 {
			enviado = true
		}
	}
	if !enviado {
		t.Errorf("el POST a contacto debe llevar UsuarioModificacionId=9; capturas=%+v", crud.Capturas())
	}
}

// Un campo de la especialización natural enviado plano en el cuerpo del proveedor se
// enruta por PK compartida al CRUD de proveedor_natural, y no al de proveedor.
func TestIntegracionEscrituraEnrutaCampoPlanoDeEspecializacionNatural(t *testing.T) {
	crud := NuevoCRUDSimulado(t, nil)
	router := routerProveedor(t, crud)

	cuerpo := cuerpoProveedorNatural()
	delete(cuerpo, "ProveedorNatural")
	cuerpo["ExperienciaLaboral"] = float64(12)

	rec, _ := ejecutar(t, router, http.MethodPost, comoJSON(t, cuerpo))
	if rec.Code != http.StatusOK && rec.Code != http.StatusPartialContent {
		t.Fatalf("status = %d; esperado 200 o 206. body=%s", rec.Code, rec.Body.String())
	}

	var enNatural, enProveedor bool
	for _, c := range crud.Capturas() {
		if c.Metodo != http.MethodPost && c.Metodo != http.MethodPatch {
			continue
		}
		switch recursoDePath(c.Path) {
		case "proveedor_natural":
			if v, _ := c.Cuerpo["ExperienciaLaboral"].(float64); int(v) == 12 {
				enNatural = true
			}
		case "proveedor":
			if c.Metodo != http.MethodPost {
				continue
			}
			if _, presente := c.Cuerpo["ExperienciaLaboral"]; presente {
				enProveedor = true
			}
		}
	}
	if !enNatural {
		t.Errorf("ExperienciaLaboral=12 debe llegar al CRUD proveedor_natural; capturas=%+v", crud.Capturas())
	}
	if enProveedor {
		t.Errorf("ExperienciaLaboral no debe enviarse al CRUD proveedor; capturas=%+v", crud.Capturas())
	}
}

// Un campo de la especialización jurídica enviado en un registro de tipo natural responde
// 400 identificando el campo, sin materializar ninguna escritura.
func TestIntegracionEscrituraRechazaCampoDeOtroTipoSinEscribir(t *testing.T) {
	crud := NuevoCRUDSimulado(t, nil)
	router := routerProveedor(t, crud)

	cuerpo := cuerpoProveedorNatural()
	delete(cuerpo, "ProveedorNatural")
	cuerpo["RegistroMercantil"] = true

	rec, _ := ejecutar(t, router, http.MethodPost, comoJSON(t, cuerpo))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d; esperado 400. body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "RegistroMercantil") {
		t.Errorf("el error debe identificar el campo ajeno al tipo; body=%s", rec.Body.String())
	}
	if len(crud.Capturas()) != 0 {
		t.Errorf("no debe haber escrituras; capturas=%+v", crud.Capturas())
	}
}

// El registro de persona jurídica materializa la rama persona_juridica_tipo_entidad bajo
// proveedor_juridico, con PersonaJuridicaId igual al Id compartido del proveedor.
func TestIntegracionCreacionJuridicoMaterializaPersonaJuridicaTipoEntidad(t *testing.T) {
	crud := NuevoCRUDSimulado(t, nil)
	router := routerProveedor(t, crud)

	cuerpo := map[string]any{
		"TipoProveedor":         float64(tipoRegistroJuridicoCascada),
		"DescripcionPortafolio": "Servicios de consultoría",
		"Tercero":               map[string]any{"NombreCompleto": "Empresa Demo SAS"},
		"ProveedorJuridico": map[string]any{
			"PersonaJuridicaTipoEntidad": []any{
				map[string]any{"NombrePerfilEntidad": "Entidad", "TipoEntidadId": float64(4)},
			},
		},
	}

	rec, _ := ejecutar(t, router, http.MethodPost, comoJSON(t, cuerpo))
	if rec.Code != http.StatusOK && rec.Code != http.StatusPartialContent {
		t.Fatalf("status = %d; esperado 200 o 206. body=%s", rec.Code, rec.Body.String())
	}

	var materializada bool
	for _, c := range crud.Capturas() {
		if c.Metodo != http.MethodPost || recursoDePath(c.Path) != "persona_juridica_tipo_entidad" {
			continue
		}
		pjid, _ := c.Cuerpo["PersonaJuridicaId"].(float64)
		tipo, _ := c.Cuerpo["TipoEntidadId"].(float64)
		if int(pjid) == 101 && int(tipo) == 4 {
			materializada = true
		}
	}
	if !materializada {
		t.Errorf("persona_juridica_tipo_entidad debe crearse con PersonaJuridicaId=101 y TipoEntidadId=4; capturas=%+v", crud.Capturas())
	}
}

// RepresentanteId y ContactoId se envían como FK planas a profundidad 0: no crean ni
// actualizan la entidad referenciada.
func TestIntegracionFKPlanasNoCreanNiActualizanReferenciadas(t *testing.T) {
	crud := NuevoCRUDSimulado(t, nil)
	router := routerProveedor(t, crud)

	cuerpo := cuerpoProveedorNatural()
	cuerpo["Representacion"] = []any{
		map[string]any{"RepresentanteId": float64(301)},
	}
	cuerpo["CuentaBancaria"] = []any{
		map[string]any{
			"EntidadBancariaId": float64(4), "TipoCuentaId": float64(2),
			"NumeroCuenta": "001-123456", "ContactoId": float64(77),
		},
	}

	rec, _ := ejecutar(t, router, http.MethodPost, comoJSON(t, cuerpo))
	if rec.Code != http.StatusOK && rec.Code != http.StatusPartialContent {
		t.Fatalf("status = %d; esperado 200 o 206. body=%s", rec.Code, rec.Body.String())
	}

	var representante, cuentaConContacto bool
	for _, c := range crud.Capturas() {
		switch recursoDePath(c.Path) {
		case "representacion":
			if v, _ := c.Cuerpo["RepresentanteId"].(float64); int(v) == 301 {
				representante = true
			}
		case "cuenta_bancaria":
			if v, _ := c.Cuerpo["ContactoId"].(float64); int(v) == 77 {
				cuentaConContacto = true
			}
		case "contacto", "tercero":
			if id, presente := c.Cuerpo["Id"].(float64); presente && (int(id) == 77 || int(id) == 301) {
				t.Errorf("la FK plana no debe crear ni actualizar la entidad referenciada: %+v", c)
			}
		}
	}
	if !representante {
		t.Errorf("representacion debe enviar RepresentanteId=301 como FK plana; capturas=%+v", crud.Capturas())
	}
	if !cuentaConContacto {
		t.Errorf("cuenta_bancaria debe enviar ContactoId=77 como FK plana; capturas=%+v", crud.Capturas())
	}
}

// Las entidades de terceros usan terceros_campos_auditoria, que no declara la columna
// usuario_modificacion_id, así que el campo no es escribible y la escritura se rechaza
// antes de llegar a algún CRUD.
func TestIntegracionEscrituraRechazaUsuarioModificacionIdEnTerceros(t *testing.T) {
	crud := NuevoCRUDSimulado(t, nil)
	router := routerProveedor(t, crud)

	cuerpo := cuerpoProveedorNatural()
	tercero, ok := cuerpo["Tercero"].(map[string]any)
	if !ok {
		t.Fatalf("el cuerpo de prueba debe tener Tercero como objeto; cuerpo=%+v", cuerpo)
	}
	tercero["UsuarioModificacionId"] = float64(9)

	rec, _ := ejecutar(t, router, http.MethodPost, comoJSON(t, cuerpo))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d; esperado 400. body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Tercero.UsuarioModificacionId") {
		t.Errorf("el 400 debe identificar el campo no escribible Tercero.UsuarioModificacionId; body=%s", rec.Body.String())
	}

	for _, c := range crud.Capturas() {
		if c.Metodo != http.MethodGet {
			t.Errorf("una escritura rechazada no debe materializar ninguna mutación; captura=%+v", c)
		}
	}
}

// UsuarioModificacionId del cuerpo llega al CRUD dueño del proveedor, y la escritura
// no lleva AutorModificacionId.
func TestIntegracionEscrituraAutorDeCuerpo(t *testing.T) {
	crud := NuevoCRUDSimulado(t, nil)
	router := routerProveedor(t, crud)

	cuerpo := cuerpoProveedorNatural()
	cuerpo["UsuarioModificacionId"] = float64(9)
	rec, _ := ejecutar(t, router, http.MethodPost, comoJSON(t, cuerpo))
	if rec.Code != http.StatusOK && rec.Code != http.StatusPartialContent {
		t.Fatalf("status = %d; esperado 200 o 206. body=%s", rec.Code, rec.Body.String())
	}

	var enviado map[string]any
	for _, c := range crud.Capturas() {
		if c.Metodo == http.MethodPost && recursoDePath(c.Path) == "proveedor" {
			enviado = c.Cuerpo
		}
	}
	if enviado == nil {
		t.Fatalf("no se recibió el POST de proveedor; capturas=%+v", crud.Capturas())
	}
	if v, _ := enviado["UsuarioModificacionId"].(float64); int(v) != 9 {
		t.Errorf("el POST de proveedor debe llevar UsuarioModificacionId=9 del cuerpo; cuerpo=%+v", enviado)
	}
	if _, presente := enviado["AutorModificacionId"]; presente {
		t.Errorf("la escritura no debe llevar AutorModificacionId; cuerpo=%+v", enviado)
	}
}

// Un campo administrado en el cuerpo responde 400 identificando el campo, y ninguna
// petición de escritura llega a los CRUD dueños.
func TestIntegracionEscrituraRechazaCamposDeAuditoriaAdministrados(t *testing.T) {
	valores := map[string]any{
		"FechaCreacion":     "2026-01-01T00:00:00Z",
		"FechaModificacion": "2026-01-02T00:00:00Z",
		"Activo":            false,
	}
	for campo, valor := range valores {
		t.Run(campo, func(t *testing.T) {
			crud := NuevoCRUDSimulado(t, nil)
			router := routerProveedor(t, crud)

			cuerpo := cuerpoProveedorNatural()
			cuerpo[campo] = valor

			rec, _ := ejecutar(t, router, http.MethodPost, comoJSON(t, cuerpo))
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d; esperado 400. body=%s", rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), campo) {
				t.Errorf("el 400 debe identificar el campo no escribible %s; body=%s", campo, rec.Body.String())
			}
			for _, c := range crud.Capturas() {
				if c.Metodo != http.MethodGet {
					t.Errorf("un campo de auditoría rechazado no debe materializar escrituras; captura=%+v", c)
				}
			}
		})
	}
}

// En creación y en reconciliación, el contacto (que declara el campo) recibe el autor
// de la raíz.
func TestIntegracionAutoriaRaizSePropagaAAnidados(t *testing.T) {
	t.Run("POST", func(t *testing.T) {
		crud := NuevoCRUDSimulado(t, nil)
		router := routerProveedor(t, crud)

		cuerpo := cuerpoProveedorNatural()
		cuerpo["UsuarioModificacionId"] = float64(9)
		rec, _ := ejecutar(t, router, http.MethodPost, comoJSON(t, cuerpo))
		if rec.Code != http.StatusOK && rec.Code != http.StatusPartialContent {
			t.Fatalf("status = %d; esperado 200 o 206. body=%s", rec.Code, rec.Body.String())
		}

		var propagado bool
		for _, c := range crud.Capturas() {
			if c.Metodo == http.MethodPost && recursoDePath(c.Path) == "contacto" {
				if v, _ := c.Cuerpo["UsuarioModificacionId"].(float64); int(v) == 9 {
					propagado = true
				}
			}
		}
		if !propagado {
			t.Errorf("el POST de contacto debe llevar UsuarioModificacionId=9 de la raíz; capturas=%+v", crud.Capturas())
		}
	})

	t.Run("PATCH", func(t *testing.T) {
		crud := NuevoCRUDSimulado(t, map[string][]map[string]any{
			"GET /v1/proveedor": {{"Id": float64(101), "TerceroId": float64(25)}},
		})
		router := routerProveedor(t, crud)

		cuerpo := cuerpoProveedorNaturalUpdate(101, 201, 301)
		cuerpo["UsuarioModificacionId"] = float64(9)
		rec, _ := ejecutar(t, router, http.MethodPatch, comoJSON(t, cuerpo))
		if rec.Code != http.StatusOK && rec.Code != http.StatusPartialContent {
			t.Fatalf("status = %d; esperado 200 o 206. body=%s", rec.Code, rec.Body.String())
		}

		var propagado bool
		for _, c := range crud.Capturas() {
			if c.Metodo == http.MethodPost && recursoDePath(c.Path) == "contacto" {
				if v, _ := c.Cuerpo["UsuarioModificacionId"].(float64); int(v) == 9 {
					propagado = true
				}
			}
		}
		if !propagado {
			t.Errorf("el POST de contacto en la reconciliación debe llevar UsuarioModificacionId=9; capturas=%+v", crud.Capturas())
		}
	})
}

// El valor propio de un anidado prevalece, terceros no recibe el campo, y sin valor en
// la raíz no se propaga nada.
func TestIntegracionAutoriaPropiaDeAnidadoPrevaleceYTercerosNoRecibe(t *testing.T) {
	t.Run("valor propio prevalece y terceros no recibe", func(t *testing.T) {
		crud := NuevoCRUDSimulado(t, nil)
		router := routerProveedor(t, crud)

		cuerpo := cuerpoProveedorNatural()
		cuerpo["UsuarioModificacionId"] = float64(9)
		contactos, ok := cuerpo["Contacto"].([]any)
		if !ok || len(contactos) == 0 {
			t.Fatalf("el cuerpo debe tener Contacto como lista no vacía; cuerpo=%+v", cuerpo)
		}
		contactos[0].(map[string]any)["UsuarioModificacionId"] = float64(7)

		rec, _ := ejecutar(t, router, http.MethodPost, comoJSON(t, cuerpo))
		if rec.Code != http.StatusOK && rec.Code != http.StatusPartialContent {
			t.Fatalf("status = %d; esperado 200 o 206. body=%s", rec.Code, rec.Body.String())
		}

		var contactoOK bool
		for _, c := range crud.Capturas() {
			switch {
			case c.Metodo == http.MethodPost && recursoDePath(c.Path) == "contacto":
				if v, _ := c.Cuerpo["UsuarioModificacionId"].(float64); int(v) == 7 {
					contactoOK = true
				} else {
					t.Errorf("el contacto debe conservar su valor propio 7; cuerpo=%+v", c.Cuerpo)
				}
			case c.Metodo == http.MethodPost && recursoDePath(c.Path) == "tercero":
				if _, presente := c.Cuerpo["UsuarioModificacionId"]; presente {
					t.Errorf("tercero no declara el campo y no debe recibirlo; cuerpo=%+v", c.Cuerpo)
				}
			}
		}
		if !contactoOK {
			t.Errorf("no se recibió el POST de contacto con su valor propio; capturas=%+v", crud.Capturas())
		}
	})

	t.Run("sin valor en la raíz no se propaga", func(t *testing.T) {
		crud := NuevoCRUDSimulado(t, nil)
		router := routerProveedor(t, crud)

		rec, _ := ejecutar(t, router, http.MethodPost, comoJSON(t, cuerpoProveedorNatural()))
		if rec.Code != http.StatusOK && rec.Code != http.StatusPartialContent {
			t.Fatalf("status = %d; esperado 200 o 206. body=%s", rec.Code, rec.Body.String())
		}
		for _, c := range crud.Capturas() {
			if c.Metodo != http.MethodPost {
				continue
			}
			if _, presente := c.Cuerpo["UsuarioModificacionId"]; presente {
				t.Errorf("sin valor en la raíz ninguna escritura debe llevar UsuarioModificacionId; captura=%+v", c)
			}
		}
	})
}
