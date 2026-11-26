package models

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestRegistroCuentaBancariaJSON(t *testing.T) {
	entrada := `{
		"CuentaBancaria": [
			{
				"EntidadBancariaId": 7,
				"TipoCuentaId": 1,
				"NumeroCuenta": "000123456789",
				"MontoCapitalAutorizado": 1000000,
				"ContactoId": null
			},
			{
				"EntidadBancariaId": 8,
				"TipoCuentaId": 2,
				"NumeroCuenta": "987654321",
				"MontoCapitalAutorizado": 250000,
				"ContactoId": 31
			}
		],
		"ContactoPago": {
			"TipoCanalContactoId": 2,
			"ValorEspecifico": "pagos@example.test",
			"NombreContacto": "Contacto de prueba",
			"Extension": null
		}
	}`

	var registro RegistroCuentaBancaria
	if err := json.Unmarshal([]byte(entrada), &registro); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(registro.CuentaBancaria) != 2 {
		t.Fatalf("len(CuentaBancaria) = %d, se esperaban 2 cuentas", len(registro.CuentaBancaria))
	}
	if registro.CuentaBancaria[0].NumeroCuenta != "000123456789" {
		t.Errorf("NumeroCuenta = %q, se esperaban conservar los ceros iniciales", registro.CuentaBancaria[0].NumeroCuenta)
	}
	if registro.CuentaBancaria[0].ContactoId != nil {
		t.Errorf("ContactoId = %v, se esperaba null", registro.CuentaBancaria[0].ContactoId)
	}
	if registro.CuentaBancaria[1].ContactoId == nil || *registro.CuentaBancaria[1].ContactoId != 31 {
		t.Errorf("ContactoId = %v, se esperaba 31", registro.CuentaBancaria[1].ContactoId)
	}
	if registro.ContactoPago.TipoCanalContactoId != 2 ||
		registro.ContactoPago.ValorEspecifico != "pagos@example.test" ||
		registro.ContactoPago.NombreContacto != "Contacto de prueba" ||
		registro.ContactoPago.Extension != nil {
		t.Errorf("ContactoPago deserializado incorrectamente: %+v", registro.ContactoPago)
	}

	salida, err := json.Marshal(registro)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	jsonSalida := string(salida)
	for _, campo := range []string{
		`"CuentaBancaria"`,
		`"ContactoPago"`,
		`"EntidadBancariaId"`,
		`"TipoCuentaId"`,
		`"NumeroCuenta":"000123456789"`,
		`"MontoCapitalAutorizado"`,
		`"ContactoId":null`,
		`"TipoCanalContactoId"`,
		`"ValorEspecifico"`,
		`"NombreContacto"`,
		`"Extension":null`,
	} {
		if !strings.Contains(jsonSalida, campo) {
			t.Errorf("la serialización no contiene %s: %s", campo, jsonSalida)
		}
	}

	var roundTrip RegistroCuentaBancaria
	if err := json.Unmarshal(salida, &roundTrip); err != nil {
		t.Fatalf("unmarshal del JSON serializado: %v", err)
	}
	if !reflect.DeepEqual(registro, roundTrip) {
		t.Errorf("round-trip = %+v, se esperaba %+v", roundTrip, registro)
	}

	extension := "123"
	registro.ContactoPago.Extension = &extension
	salida, err = json.Marshal(registro)
	if err != nil {
		t.Fatalf("marshal con Extension: %v", err)
	}
	if !strings.Contains(string(salida), `"Extension":"123"`) {
		t.Errorf("Extension no se serializó como string: %s", salida)
	}
}

func TestCuentaBancariaDataJSON(t *testing.T) {
	esperado := CuentaBancariaData{
		TerceroId:      101,
		ProveedorId:    202,
		ContactoPagoId: 303,
		CuentaBancaria: []CuentaBancariaRegistrada{{
			Id:                     404,
			EntidadBancariaId:      7,
			TipoCuentaId:           1,
			NumeroCuenta:           "000123456789",
			MontoCapitalAutorizado: 1000000,
			ContactoId:             303,
		}},
		Resultado: Resultado{
			Creados:      []EntidadId{{Entidad: "CuentaBancaria", Id: 404}},
			Actualizados: []EntidadId{},
			Reutilizados: []EntidadId{},
		},
	}

	salida, err := json.Marshal(esperado)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	jsonSalida := string(salida)
	for _, campo := range []string{
		`"TerceroId"`,
		`"ProveedorId"`,
		`"ContactoPagoId"`,
		`"CuentaBancaria"`,
		`"Resultado"`,
		`"Id"`,
		`"ContactoId"`,
		`"Creados"`,
		`"Actualizados"`,
		`"Reutilizados"`,
	} {
		if !strings.Contains(jsonSalida, campo) {
			t.Errorf("la serialización no contiene %s: %s", campo, jsonSalida)
		}
	}

	var obtenido CuentaBancariaData
	if err := json.Unmarshal(salida, &obtenido); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(obtenido, esperado) {
		t.Errorf("round-trip = %+v, se esperaba %+v", obtenido, esperado)
	}
}
