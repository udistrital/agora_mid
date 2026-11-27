package controllers

import (
	"context"
	"errors"
	"net/http"

	beego "github.com/beego/beego/v2/server/web"

	"github.com/udistrital/agora_mid/helper"
	"github.com/udistrital/agora_mid/models"
	"github.com/udistrital/agora_mid/services"
	"github.com/udistrital/agora_mid/utils"
)

// CaracterizacionJuridicaWriter abstrae el servicio del dominio 2 jurídico.
type CaracterizacionJuridicaWriter interface {
	Registrar(
		ctx context.Context,
		in models.RegistroCaracterizacionJuridica,
		terceroID int64,
		verb string,
	) (services.CaracterizacionJuridicaResult, error)
	RegistrarPorProveedor(
		ctx context.Context,
		in models.RegistroCaracterizacionJuridica,
		proveedorID int64,
		verb string,
	) (services.CaracterizacionJuridicaResult, error)
}

var servicioCaracterizacionJuridica CaracterizacionJuridicaWriter

// ConfigurarCaracterizacionJuridicaWriteService inyecta el servicio del dominio
// 2 jurídico. Se llama desde main antes de beego.Run().
func ConfigurarCaracterizacionJuridicaWriteService(servicio CaracterizacionJuridicaWriter) {
	servicioCaracterizacionJuridica = servicio
}

// CaracterizacionJuridicaController expone el registro y la actualización de la
// caracterización de persona jurídica.
type CaracterizacionJuridicaController struct {
	beego.Controller
}

// URLMapping registra los métodos del controlador.
func (c *CaracterizacionJuridicaController) URLMapping() {
	c.Mapping("Post", c.Post)
	c.Mapping("Put", c.Put)
}

// Post registra o continúa la caracterización jurídica.
// @Title Post
// @Description Registra o actualiza la caracterización de persona jurídica (dominio 2)
// @Param body body models.RegistroCaracterizacionJuridica true "cuerpo de la caracterización"
// @Success 201 {object} models.CaracterizacionJuridicaData
// @Success 200 {object} models.CaracterizacionJuridicaData
// @Success 206 {object} models.CaracterizacionJuridicaData
// @Failure 400 petición inválida
// @Failure 404 tercero inexistente
// @Failure 409 conflicto
// @Failure 502 dependencia no disponible
// @router /juridica/caracterizacion [post]
func (c *CaracterizacionJuridicaController) Post() {
	var in models.RegistroCaracterizacionJuridica
	if !decodificarCuerpo(&c.Controller, &in) {
		return
	}
	if in.TerceroId <= 0 {
		utils.AbortWithError(&c.Controller, http.StatusBadRequest,
			"TerceroId debe ser un entero positivo", nil)
		return
	}
	c.ejecutar(in, in.TerceroId, helper.WriteVerbCreate)
}

// Put actualiza la caracterización jurídica por TerceroId.
// @Title Put
// @Description Actualiza la caracterización de persona jurídica (dominio 2)
// @Param terceroId path int true "Id del tercero"
// @Param body body models.RegistroCaracterizacionJuridica true "cuerpo de la caracterización"
// @Success 200 {object} models.CaracterizacionJuridicaData
// @Success 201 {object} models.CaracterizacionJuridicaData
// @Success 206 {object} models.CaracterizacionJuridicaData
// @Failure 400 petición inválida
// @Failure 404 tercero o proveedor inexistente
// @Failure 409 conflicto
// @Failure 502 dependencia no disponible
// @router /juridica/caracterizacion/{terceroId} [put]
func (c *CaracterizacionJuridicaController) Put() {
	terceroID, ok := leerTerceroID(&c.Controller)
	if !ok {
		return
	}
	var in models.RegistroCaracterizacionJuridica
	if !decodificarCuerpo(&c.Controller, &in) {
		return
	}
	if in.TerceroId != 0 && in.TerceroId != terceroID {
		utils.AbortWithError(&c.Controller, http.StatusBadRequest,
			"TerceroId del cuerpo no coincide con el de la ruta", nil)
		return
	}
	c.ejecutar(in, terceroID, helper.WriteVerbUpdate)
}

// PutPorProveedor actualiza la caracterización jurídica usando el Id del
// proveedor.
// @Title PutPorProveedor
// @Description Actualiza la caracterización de persona jurídica por ProveedorId (dominio 2)
// @Param proveedorId path int true "Id del proveedor"
// @Param body body models.RegistroCaracterizacionJuridica true "cuerpo de la caracterización"
// @Success 200 {object} models.CaracterizacionJuridicaData
// @Success 201 {object} models.CaracterizacionJuridicaData
// @Success 206 {object} models.CaracterizacionJuridicaData
// @Failure 400 petición inválida
// @Failure 404 proveedor inexistente
// @Failure 409 conflicto
// @Failure 502 dependencia no disponible
// @router /juridica/caracterizacion/proveedor/{proveedorId} [put]
func (c *CaracterizacionJuridicaController) PutPorProveedor() {
	proveedorID, ok := leerProveedorID(&c.Controller)
	if !ok {
		return
	}
	var in models.RegistroCaracterizacionJuridica
	if !decodificarCuerpo(&c.Controller, &in) {
		return
	}
	c.ejecutarPorProveedor(in, proveedorID)
}

func (c *CaracterizacionJuridicaController) ejecutarPorProveedor(
	in models.RegistroCaracterizacionJuridica,
	proveedorID int64,
) {
	if servicioCaracterizacionJuridica == nil {
		utils.AbortWithError(&c.Controller, http.StatusInternalServerError,
			"El servicio de caracterización jurídica no está configurado", nil)
		return
	}
	resultado, err := servicioCaracterizacionJuridica.RegistrarPorProveedor(
		c.Ctx.Request.Context(), in, proveedorID, helper.WriteVerbUpdate,
	)
	if err != nil {
		c.responderError(err)
		return
	}
	c.responderResultado(resultado)
}

func (c *CaracterizacionJuridicaController) ejecutar(
	in models.RegistroCaracterizacionJuridica,
	terceroID int64,
	verb string,
) {
	if servicioCaracterizacionJuridica == nil {
		utils.AbortWithError(&c.Controller, http.StatusInternalServerError,
			"El servicio de caracterización jurídica no está configurado", nil)
		return
	}
	resultado, err := servicioCaracterizacionJuridica.Registrar(
		c.Ctx.Request.Context(), in, terceroID, verb,
	)
	if err != nil {
		c.responderError(err)
		return
	}
	c.responderResultado(resultado)
}

func (c *CaracterizacionJuridicaController) responderResultado(
	resultado services.CaracterizacionJuridicaResult,
) {
	if resultado.Parcial {
		utils.WritePartial(&c.Controller, "Registro parcial", resultado.Data, resultado.Avisos)
		return
	}
	if len(resultado.Data.Resultado.Creados) > 0 {
		utils.WriteSuccessWithWarnings(&c.Controller, http.StatusCreated,
			"Caracterización registrada", resultado.Data, resultado.Avisos)
		return
	}
	utils.WriteSuccessWithWarnings(&c.Controller, http.StatusOK,
		"Caracterización actualizada", resultado.Data, resultado.Avisos)
}

func (c *CaracterizacionJuridicaController) responderError(err error) {
	var caracterizacionErr *services.CaracterizacionNaturalError
	if !errors.As(err, &caracterizacionErr) {
		utils.AbortWithError(&c.Controller, http.StatusInternalServerError,
			"Error interno al registrar la caracterización jurídica", err)
		return
	}
	switch caracterizacionErr.StatusCode {
	case http.StatusBadRequest, http.StatusNotFound, http.StatusConflict,
		http.StatusInternalServerError, http.StatusBadGateway:
		utils.AbortWithError(&c.Controller, caracterizacionErr.StatusCode,
			caracterizacionErr.Message, caracterizacionErr.Cause)
	case http.StatusGatewayTimeout:
		utils.WriteError(&c.Controller, http.StatusGatewayTimeout,
			caracterizacionErr.Message, nil)
	default:
		utils.AbortWithError(&c.Controller, http.StatusInternalServerError,
			"Error interno al registrar la caracterización jurídica", err)
	}
}
