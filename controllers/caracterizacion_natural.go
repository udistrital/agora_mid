package controllers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	beego "github.com/beego/beego/v2/server/web"

	"github.com/udistrital/agora_mid/helper"
	"github.com/udistrital/agora_mid/models"
	"github.com/udistrital/agora_mid/services"
	"github.com/udistrital/agora_mid/utils"
)

// CaracterizacionNaturalWriter abstrae el servicio del dominio 2 natural.
type CaracterizacionNaturalWriter interface {
	Registrar(
		ctx context.Context,
		in models.RegistroCaracterizacionNatural,
		terceroID int64,
		verb string,
	) (services.CaracterizacionNaturalResult, error)
	RegistrarPorProveedor(
		ctx context.Context,
		in models.RegistroCaracterizacionNatural,
		proveedorID int64,
		verb string,
	) (services.CaracterizacionNaturalResult, error)
}

var servicioCaracterizacionNatural CaracterizacionNaturalWriter

// ConfigurarCaracterizacionNaturalWriteService inyecta el servicio del dominio 2
// natural. Se llama desde main antes de beego.Run().
func ConfigurarCaracterizacionNaturalWriteService(servicio CaracterizacionNaturalWriter) {
	servicioCaracterizacionNatural = servicio
}

// CaracterizacionNaturalController expone el registro y la actualización de la
// caracterización de persona natural.
type CaracterizacionNaturalController struct {
	beego.Controller
}

// URLMapping registra los métodos del controlador.
func (c *CaracterizacionNaturalController) URLMapping() {
	c.Mapping("Post", c.Post)
	c.Mapping("Put", c.Put)
}

// Post registra o continúa la caracterización natural.
// @Title Post
// @Description Registra o actualiza la caracterización de persona natural (dominio 2)
// @Param body body models.RegistroCaracterizacionNatural true "cuerpo de la caracterización"
// @Success 201 {object} models.CaracterizacionNaturalData
// @Success 200 {object} models.CaracterizacionNaturalData
// @Success 206 {object} models.CaracterizacionNaturalData
// @Failure 400 petición inválida
// @Failure 404 tercero inexistente
// @Failure 409 conflicto
// @Failure 502 dependencia no disponible
// @router /natural/caracterizacion [post]
func (c *CaracterizacionNaturalController) Post() {
	var in models.RegistroCaracterizacionNatural
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

// Put actualiza la caracterización natural por TerceroId.
// @Title Put
// @Description Actualiza la caracterización de persona natural (dominio 2)
// @Param terceroId path int true "Id del tercero"
// @Param body body models.RegistroCaracterizacionNatural true "cuerpo de la caracterización"
// @Success 200 {object} models.CaracterizacionNaturalData
// @Success 201 {object} models.CaracterizacionNaturalData
// @Success 206 {object} models.CaracterizacionNaturalData
// @Failure 400 petición inválida
// @Failure 404 tercero o proveedor inexistente
// @Failure 409 conflicto
// @Failure 502 dependencia no disponible
// @router /natural/caracterizacion/{terceroId} [put]
func (c *CaracterizacionNaturalController) Put() {
	terceroID, ok := leerTerceroID(&c.Controller)
	if !ok {
		return
	}
	var in models.RegistroCaracterizacionNatural
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

// PutPorProveedor actualiza la caracterización usando el Id del proveedor.
// @Title PutPorProveedor
// @Description Actualiza la caracterización de persona natural por ProveedorId (dominio 2)
// @Param proveedorId path int true "Id del proveedor"
// @Param body body models.RegistroCaracterizacionNatural true "cuerpo de la caracterización"
// @Success 200 {object} models.CaracterizacionNaturalData
// @Success 201 {object} models.CaracterizacionNaturalData
// @Success 206 {object} models.CaracterizacionNaturalData
// @Failure 400 petición inválida
// @Failure 404 proveedor inexistente
// @Failure 409 conflicto
// @Failure 502 dependencia no disponible
// @router /natural/caracterizacion/proveedor/{proveedorId} [put]
func (c *CaracterizacionNaturalController) PutPorProveedor() {
	proveedorID, ok := leerProveedorID(&c.Controller)
	if !ok {
		return
	}
	var in models.RegistroCaracterizacionNatural
	if !decodificarCuerpo(&c.Controller, &in) {
		return
	}
	c.ejecutarPorProveedor(in, proveedorID)
}

func (c *CaracterizacionNaturalController) ejecutarPorProveedor(
	in models.RegistroCaracterizacionNatural,
	proveedorID int64,
) {
	if servicioCaracterizacionNatural == nil {
		utils.AbortWithError(&c.Controller, http.StatusInternalServerError,
			"El servicio de caracterización no está configurado", nil)
		return
	}
	resultado, err := servicioCaracterizacionNatural.RegistrarPorProveedor(
		c.Ctx.Request.Context(), in, proveedorID, helper.WriteVerbUpdate,
	)
	if err != nil {
		c.responderError(err)
		return
	}
	c.responderResultado(resultado)
}

func (c *CaracterizacionNaturalController) ejecutar(
	in models.RegistroCaracterizacionNatural,
	terceroID int64,
	verb string,
) {
	if servicioCaracterizacionNatural == nil {
		utils.AbortWithError(&c.Controller, http.StatusInternalServerError,
			"El servicio de caracterización no está configurado", nil)
		return
	}

	resultado, err := servicioCaracterizacionNatural.Registrar(
		c.Ctx.Request.Context(), in, terceroID, verb,
	)
	if err != nil {
		c.responderError(err)
		return
	}

	c.responderResultado(resultado)
}

func (c *CaracterizacionNaturalController) responderResultado(
	resultado services.CaracterizacionNaturalResult,
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

func (c *CaracterizacionNaturalController) responderError(err error) {
	var caracterizacionErr *services.CaracterizacionNaturalError
	if !errors.As(err, &caracterizacionErr) {
		utils.AbortWithError(&c.Controller, http.StatusInternalServerError,
			"Error interno al registrar la caracterización", err)
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
			"Error interno al registrar la caracterización", err)
	}
}

func leerProveedorID(c *beego.Controller) (int64, bool) {
	valor := c.Ctx.Input.Param(":proveedorId")
	id, err := strconv.ParseInt(strings.TrimSpace(valor), 10, 64)
	if err != nil || id <= 0 {
		utils.AbortWithError(c, http.StatusBadRequest,
			"proveedorId debe ser un entero positivo", err)
		return 0, false
	}
	return id, true
}
