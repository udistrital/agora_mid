package controllers

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	beego "github.com/beego/beego/v2/server/web"

	"github.com/udistrital/agora_mid/helper"
	"github.com/udistrital/agora_mid/models"
	"github.com/udistrital/agora_mid/services"
	"github.com/udistrital/agora_mid/utils"
)

// TerceroJuridicoWriter abstrae el servicio jurídico del dominio 1 para poder
// probar el controlador sin un CRUD real.
type TerceroJuridicoWriter interface {
	EscribirTerceroJuridico(
		ctx context.Context,
		in models.RegistroTerceroJuridico,
		terceroID int64,
		verb string,
	) (services.TerceroWriteResult, error)
}

var servicioTerceroJuridico TerceroJuridicoWriter

// ConfigurarTerceroJuridicoWriteService inyecta el servicio jurídico del
// dominio 1. Se llama desde main antes de beego.Run().
func ConfigurarTerceroJuridicoWriteService(servicio TerceroJuridicoWriter) {
	servicioTerceroJuridico = servicio
}

// ProveedorJuridicoController expone el registro y la actualización de la
// persona jurídica en el dominio 1.
type ProveedorJuridicoController struct {
	beego.Controller
}

// URLMapping registra los métodos del controlador.
func (c *ProveedorJuridicoController) URLMapping() {
	c.Mapping("Post", c.Post)
	c.Mapping("Put", c.Put)
}

// Post registra o continúa el registro de un tercero persona jurídica y su
// identificación.
// @Title Post
// @Description Registra o actualiza tercero y datos_identificacion (dominio 1 jurídica)
// @Param	body	body	models.RegistroTerceroJuridico	true	"cuerpo del registro"
// @Success 201 {object} models.RegistroTerceroData
// @Success 200 {object} models.RegistroTerceroData
// @Success 206 {object} models.RegistroTerceroData
// @Failure 400 petición inválida
// @Failure 409 conflicto de tipo o documento
// @Failure 502 dependencia no disponible
// @router /juridica [post]
func (c *ProveedorJuridicoController) Post() {
	in, ok := c.decodificarYValidar()
	if !ok {
		return
	}
	c.ejecutar(in, 0, helper.WriteVerbCreate)
}

// Put actualiza el tercero jurídico y su identificación por TerceroId.
// @Title Put
// @Description Actualiza tercero y datos_identificacion (dominio 1 jurídica)
// @Param	terceroId	path	int	true	"Id del tercero"
// @Param	body	body	models.RegistroTerceroJuridico	true	"cuerpo de la actualización"
// @Success 200 {object} models.RegistroTerceroData
// @Success 201 {object} models.RegistroTerceroData
// @Success 206 {object} models.RegistroTerceroData
// @Failure 400 petición inválida
// @Failure 404 tercero inexistente
// @Failure 409 conflicto de tipo o documento
// @Failure 502 dependencia no disponible
// @router /juridica/{terceroId} [put]
func (c *ProveedorJuridicoController) Put() {
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
// los obligatorios del dominio 1 jurídico.
func (c *ProveedorJuridicoController) decodificarYValidar() (models.RegistroTerceroJuridico, bool) {
	var in models.RegistroTerceroJuridico
	if !decodificarCuerpo(&c.Controller, &in) {
		return in, false
	}
	if msg, valido := validarTerceroJuridico(in); !valido {
		utils.AbortWithError(&c.Controller, http.StatusBadRequest, msg, nil)
		return in, false
	}
	return in, true
}

// ejecutar invoca el servicio y traduce el resultado o el error al envelope.
func (c *ProveedorJuridicoController) ejecutar(
	in models.RegistroTerceroJuridico,
	terceroID int64,
	verb string,
) {
	if servicioTerceroJuridico == nil {
		utils.AbortWithError(&c.Controller, http.StatusInternalServerError,
			"El servicio de registro de terceros no está configurado", nil)
		return
	}

	resultado, err := servicioTerceroJuridico.EscribirTerceroJuridico(
		c.Ctx.Request.Context(), in, terceroID, verb,
	)
	if err != nil {
		c.responderError(err)
		return
	}

	escribirRespuestaTercero(
		&c.Controller,
		resultado,
		dataDeResultadoJuridico(resultado),
		"Tercero e identificación creados",
		"Tercero e identificación actualizados",
	)
}

// responderError mapea los errores del servicio jurídico a códigos HTTP.
func (c *ProveedorJuridicoController) responderError(err error) {
	responderErrorTercero(&c.Controller, err, "P_JURIDICA")
}

// validarTerceroJuridico comprueba los campos obligatorios del payload.
func validarTerceroJuridico(in models.RegistroTerceroJuridico) (string, bool) {
	if strings.TrimSpace(in.DatosBasicos.RazonSocial) == "" {
		return "DatosBasicos.RazonSocial es obligatorio", false
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

// dataDeResultadoJuridico proyecta el tercero al contrato jurídico: expone
// RazonSocial (derivada de NombreCompleto) y omite los campos de persona natural
// que estén vacíos.
func dataDeResultadoJuridico(resultado services.TerceroWriteResult) models.RegistroTerceroData {
	data := dataDeResultado(resultado)
	if data.DatosBasicos == nil {
		data.DatosBasicos = map[string]any{}
	}
	if razon := textoDeValor(data.DatosBasicos["NombreCompleto"]); razon != "" {
		data.DatosBasicos["RazonSocial"] = razon
	}
	delete(data.DatosBasicos, "NombreCompleto")
	for _, campo := range []string{
		"PrimerNombre", "SegundoNombre", "PrimerApellido",
		"SegundoApellido", "FechaNacimiento",
	} {
		if textoDeValor(data.DatosBasicos[campo]) == "" {
			delete(data.DatosBasicos, campo)
		}
	}
	return data
}

func textoDeValor(valor any) string {
	switch v := valor.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}
