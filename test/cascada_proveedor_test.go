package test

import (
	"net/http"
	"testing"
)

// tipoRegistroJuridicoCascada es un TipoRegistroId distinto de tipoRegistroNatural,
// que el servicio de escritura trata como persona jurídica.
const tipoRegistroJuridicoCascada = 2

// Un registro de tipo jurídico materializa solo la rama ProveedorJuridico, con PK
// compartida con el proveedor, y no crea ProveedorNatural.
func TestIntegracionCreacionJuridicoMaterializaSoloRamaJuridica(t *testing.T) {
	crud := NuevoCRUDSimulado(t, nil)
	router := routerProveedor(t, crud)

	cuerpo := map[string]any{
		"TipoRegistroId":        float64(tipoRegistroJuridicoCascada),
		"DescripcionPortafolio": "Servicios de consultoría",
		"Tercero": map[string]any{
			"NombreCompleto": "Empresa Demo SAS",
		},
		"ProveedorJuridico": map[string]any{
			"NombreComercial":    "Empresa Demo",
			"MatriculaMercantil": "MM-778",
			"FechaConstitucion":  "2020-01-15T00:00:00Z",
			"CotizaBolsa":        false,
		},
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
		"TipoRegistroId":        float64(tipoRegistroNatural),
		"DescripcionPortafolio": "Servicios de consultoría",
		"Tercero": map[string]any{
			"NombreCompleto": "Ana Pérez",
		},
		"ProveedorNatural": map[string]any{
			"PerfilDeclarado": float64(3),
		},
		"CuentaBancaria": []any{
			map[string]any{
				"EntidadBancariaId": float64(4),
				"TipoCuentaId":      float64(2),
				"NumeroCuenta":      "001-123456",
				"NombreTitular":     "Ana Pérez",
				"CiudadAperturaId":  float64(11),
				"EsPrincipal":       true,
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
