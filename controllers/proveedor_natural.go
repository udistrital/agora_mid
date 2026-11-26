// Package controllers expone los controladores Beego del middleware.
package controllers

import (
	"context"
	"net/http"
	"strings"

	beego "github.com/beego/beego/v2/server/web"

	"github.com/udistrital/agora_mid/helper"
	"github.com/udistrital/agora_mid/models"
	"github.com/udistrital/agora_mid/services"
	"github.com/udistrital/agora_mid/utils"
)

// TerceroWriter abstrae el servicio del dominio 1 para poder probar el
// controlador sin un CRUD real.
type TerceroWriter interface {
	EscribirTercero(
		ctx context.Context,
		in models.RegistroTerceroNatural,
		terceroID int64,
		verb string,
	) (services.TerceroWriteResult, error)
}

var servicioTercero TerceroWriter

// ConfigurarTerceroWriteService inyecta el servicio del dominio 1. Se llama
// desde main antes de beego.Run().
func ConfigurarTerceroWriteService(servicio TerceroWriter) {
	servicioTercero = servicio
}

// ProveedorNaturalController expone el registro y la actualización del dominio 1.
type ProveedorNaturalController struct {
	beego.Controller
}

// URLMapping registra los métodos del controlador.
func (c *ProveedorNaturalController) URLMapping() {
	c.Mapping("Post", c.Post)
	c.Mapping("Put", c.Put)
}

// Post registra o continúa el registro de un tercero persona natural y su
// identificación.
// @Title Post
// @Description Registra o actualiza tercero y datos_identificacion (dominio 1)
// @Param	body	body	models.RegistroTerceroNatural	true	"cuerpo del registro"
// @Success 201 {object} models.RegistroTerceroData
// @Success 200 {object} models.RegistroTerceroData
// @Success 206 {object} models.RegistroTerceroData
// @Failure 400 petición inválida
// @Failure 409 conflicto de tipo o documento
// @Failure 502 dependencia no disponible
// @router /natural [post]
func (c *ProveedorNaturalController) Post() {
	in, ok := c.decodificarYValidar()
	if !ok {
		return
	}
	c.ejecutar(in, 0, helper.WriteVerbCreate)
}

// Put actualiza el tercero y su identificación por TerceroId.
// @Title Put
// @Description Actualiza tercero y datos_identificacion (dominio 1)
// @Param	terceroId	path	int	true	"Id del tercero"
// @Param	body	body	models.RegistroTerceroNatural	true	"cuerpo de la actualización"
// @Success 200 {object} models.RegistroTerceroData
// @Success 201 {object} models.RegistroTerceroData
// @Success 206 {object} models.RegistroTerceroData
// @Failure 400 petición inválida
// @Failure 404 tercero inexistente
// @Failure 409 conflicto de tipo o documento
// @Failure 502 dependencia no disponible
// @router /natural/{terceroId} [put]
func (c *ProveedorNaturalController) Put() {
	terceroID, ok := leerTerceroID(&c.Controller)
	if !ok {
		return
	}
	in, ok := c.decodificarYValidar()
	if !ok {
		return
	}
	c.ejecutar(in, terceroID, helper.WriteVerbUpdate)
}

// decodificarYValidar lee el cuerpo con campos desconocidos rechazados y valida
// los obligatorios del dominio 1.
func (c *ProveedorNaturalController) decodificarYValidar() (models.RegistroTerceroNatural, bool) {
	var in models.RegistroTerceroNatural
	if !decodificarCuerpo(&c.Controller, &in) {
		return in, false
	}
	if msg, valido := validarTerceroNatural(in); !valido {
		utils.AbortWithError(&c.Controller, http.StatusBadRequest, msg, nil)
		return in, false
	}
	return in, true
}

// ejecutar invoca el servicio y traduce el resultado o el error al envelope.
func (c *ProveedorNaturalController) ejecutar(
	in models.RegistroTerceroNatural,
	terceroID int64,
	verb string,
) {
	if servicioTercero == nil {
		utils.AbortWithError(&c.Controller, http.StatusInternalServerError,
			"El servicio de registro de terceros no está configurado", nil)
		return
	}

	resultado, err := servicioTercero.EscribirTercero(
		c.Ctx.Request.Context(), in, terceroID, verb,
	)
	if err != nil {
		c.responderError(err)
		return
	}

	escribirRespuestaTercero(
		&c.Controller,
		resultado,
		dataDeResultado(resultado),
		"Tercero e identificación creados",
		"Tercero e identificación actualizados",
	)
}

// responderError mapea los errores del servicio a códigos HTTP sin exponer
// detalles internos de los CRUD.
func (c *ProveedorNaturalController) responderError(err error) {
	responderErrorTercero(&c.Controller, err, "P_NATURAL")
}

// validarTerceroNatural comprueba los campos obligatorios del payload.
func validarTerceroNatural(in models.RegistroTerceroNatural) (string, bool) {
	if strings.TrimSpace(in.DatosBasicos.PrimerNombre) == "" ||
		strings.TrimSpace(in.DatosBasicos.PrimerApellido) == "" {
		return "DatosBasicos.PrimerNombre y PrimerApellido son obligatorios", false
	}
	if in.DatosBasicos.TipoContribuyenteId <= 0 {
		return "DatosBasicos.TipoContribuyenteId es obligatorio", false
	}
	if strings.TrimSpace(in.DatosIdentificacion.Numero) == "" {
		return "DatosIdentificacion.Numero es obligatorio", false
	}
	if in.DatosIdentificacion.TipoDocumentoId <= 0 {
		return "DatosIdentificacion.TipoDocumentoId es obligatorio", false
	}
	return "", true
}
