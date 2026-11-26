package models

type RegistroCuentaBancaria struct {
	CuentaBancaria []CuentaBancariaInput `json:"CuentaBancaria"`
	ContactoPago   ContactoPagoInput     `json:"ContactoPago"`
}

type CuentaBancariaInput struct {
	EntidadBancariaId      int64   `json:"EntidadBancariaId"`
	TipoCuentaId           int64   `json:"TipoCuentaId"`
	NumeroCuenta           string  `json:"NumeroCuenta"`
	MontoCapitalAutorizado float64 `json:"MontoCapitalAutorizado"`
	ContactoId             *int64  `json:"ContactoId"`
}

type ContactoPagoInput struct {
	TipoCanalContactoId int64   `json:"TipoCanalContactoId"`
	ValorEspecifico     string  `json:"ValorEspecifico"`
	NombreContacto      string  `json:"NombreContacto"`
	Extension           *string `json:"Extension"`
}

type CuentaBancariaData struct {
	TerceroId      int64                      `json:"TerceroId"`
	ProveedorId    int64                      `json:"ProveedorId"`
	ContactoPagoId int64                      `json:"ContactoPagoId"`
	CuentaBancaria []CuentaBancariaRegistrada `json:"CuentaBancaria"`
	Resultado      Resultado                  `json:"Resultado"`
}

type CuentaBancariaRegistrada struct {
	Id                     int64   `json:"Id"`
	EntidadBancariaId      int64   `json:"EntidadBancariaId"`
	TipoCuentaId           int64   `json:"TipoCuentaId"`
	NumeroCuenta           string  `json:"NumeroCuenta"`
	MontoCapitalAutorizado float64 `json:"MontoCapitalAutorizado"`
	ContactoId             int64   `json:"ContactoId"`
}
