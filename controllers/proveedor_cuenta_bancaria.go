package controllers

import (
	"context"
	"errors"
	"net/http"

	"github.com/beego/beego/v2/core/logs"
	beego "github.com/beego/beego/v2/server/web"

	"github.com/udistrital/agora_mid/models"
	"github.com/udistrital/agora_mid/responses"
	"github.com/udistrital/agora_mid/services"
	"github.com/udistrital/agora_mid/utils"
)

type CuentaBancariaWriter interface {
	Registrar(ctx context.Context, terceroID int64, entrada models.RegistroCuentaBancaria) (models.CuentaBancariaData, error)
}

var cuentaBancariaWriter CuentaBancariaWriter

func ConfigurarCuentaBancariaWriter(writer CuentaBancariaWriter) {
	cuentaBancariaWriter = writer
}

type ProveedorCuentaBancariaController struct {
	beego.Controller
}

func (c *ProveedorCuentaBancariaController) Post() {
	terceroID, ok := leerTerceroID(&c.Controller)
	if !ok {
		return
	}
	var entrada models.RegistroCuentaBancaria
	if !decodificarCuerpo(&c.Controller, &entrada) {
		return
	}
	if cuentaBancariaWriter == nil {
		utils.AbortWithError(&c.Controller, http.StatusInternalServerError,
			"El servicio de cuentas bancarias no está configurado", nil)
		return
	}

	data, err := cuentaBancariaWriter.Registrar(c.Ctx.Request.Context(), terceroID, entrada)
	if err != nil {
		responderErrorCuentaBancaria(&c.Controller, data, err)
		return
	}
	if len(data.Resultado.Creados) > 0 {
		utils.WriteSuccess(&c.Controller, http.StatusCreated,
			"Cuenta bancaria y contacto de pago registrados", data)
		return
	}
	utils.WriteSuccess(&c.Controller, http.StatusOK,
		"Cuenta bancaria y contacto de pago actualizados o reutilizados", data)
}

func responderErrorCuentaBancaria(c *beego.Controller, data models.CuentaBancariaData, err error) {
	var cuentaErr *services.CuentaBancariaError
	if !errors.As(err, &cuentaErr) {
		utils.AbortWithError(c, http.StatusInternalServerError,
			"Error interno al registrar la cuenta bancaria", err)
		return
	}
	if cuentaErr.Partial {
		logs.Error("status=206 mensaje=%s causa=%v", cuentaErr.Message, err)
		path := cuentaErr.Path
		if path == "" {
			path = "CuentaBancaria"
		}
		utils.WritePartial(c, "Registro parcial", data, []responses.RelationError{{
			Path:    path,
			Service: "gestion_proveedores",
			Message: cuentaErr.Message,
			Kind:    "dependency_failure",
		}})
		return
	}

	switch cuentaErr.StatusCode {
	case http.StatusBadRequest, http.StatusNotFound, http.StatusConflict,
		http.StatusInternalServerError, http.StatusBadGateway:
		utils.AbortWithError(c, cuentaErr.StatusCode, cuentaErr.Message, cuentaErr.Cause)
	case http.StatusGatewayTimeout:
		logs.Error("status=%d mensaje=%s causa=%v", cuentaErr.StatusCode, cuentaErr.Message, cuentaErr.Cause)
		utils.WriteError(c, cuentaErr.StatusCode, cuentaErr.Message, nil)
	default:
		utils.AbortWithError(c, http.StatusInternalServerError,
			"Error interno al registrar la cuenta bancaria", err)
	}
}
