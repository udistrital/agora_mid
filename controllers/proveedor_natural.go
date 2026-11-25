// Package controllers expone los controladores Beego del middleware.
package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
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
	terceroID, ok := c.leerTerceroID()
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
	cuerpo := c.Ctx.Input.RequestBody
	if len(cuerpo) == 0 && c.Ctx.Request.Body != nil {
		cuerpo, _ = io.ReadAll(c.Ctx.Request.Body)
	}
	dec := json.NewDecoder(bytes.NewReader(cuerpo))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		utils.AbortWithError(&c.Controller, http.StatusBadRequest,
			"La petición no es un JSON válido o contiene campos no permitidos", err)
		return in, false
	}
	if msg, valido := validarTerceroNatural(in); !valido {
		utils.AbortWithError(&c.Controller, http.StatusBadRequest, msg, nil)
		return in, false
	}
	return in, true
}

// leerTerceroID toma el parámetro de ruta y exige un entero positivo.
func (c *ProveedorNaturalController) leerTerceroID() (int64, bool) {
	valor := c.Ctx.Input.Param(":terceroId")
	if strings.TrimSpace(valor) == "" {
		valor = c.Ctx.Input.Param(":id")
	}
	id, err := strconv.ParseInt(strings.TrimSpace(valor), 10, 64)
	if err != nil || id <= 0 {
		utils.AbortWithError(&c.Controller, http.StatusBadRequest,
			"terceroId debe ser un entero positivo", err)
		return 0, false
	}
	return id, true
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

	data := dataDeResultado(resultado)
	switch {
	case resultado.Parcial:
		utils.WritePartial(&c.Controller, "Registro parcial", data, resultado.Avisos)
	case len(resultado.Creados) > 0:
		utils.WriteSuccessWithWarnings(&c.Controller, http.StatusCreated,
			"Tercero e identificación creados", data, resultado.Avisos)
	default:
		utils.WriteSuccessWithWarnings(&c.Controller, http.StatusOK,
			"Tercero e identificación actualizados", data, resultado.Avisos)
	}
}

// responderError mapea los errores del servicio a códigos HTTP sin exponer
// detalles internos de los CRUD.
func (c *ProveedorNaturalController) responderError(err error) {
	switch {
	case errors.Is(err, services.ErrTipoContribuyenteInvalido):
		utils.AbortWithError(&c.Controller, http.StatusBadRequest, err.Error(), nil)

	case errors.Is(err, services.ErrTipoContribuyenteConflicto):
		utils.AbortWithError(&c.Controller, http.StatusConflict, err.Error(), nil)

	case errors.Is(err, services.ErrTerceroNoEncontrado):
		utils.AbortWithError(&c.Controller, http.StatusNotFound, err.Error(), nil)

	case errors.Is(err, services.ErrIdentificacionDeOtroTercero):
		utils.AbortWithError(&c.Controller, http.StatusConflict, err.Error(), nil)

	case errors.Is(err, services.ErrCatalogoNoEncontrado),
		errors.Is(err, services.ErrCatalogoAmbiguo),
		errors.Is(err, services.ErrTipoContribuyenteNoConfigurado):
		utils.AbortWithError(&c.Controller, http.StatusInternalServerError,
			"No fue posible resolver el tipo de contribuyente P_NATURAL", err)

	default:
		var errorEscritura *helper.ErrorEscritura
		if errors.As(err, &errorEscritura) {
			utils.AbortWithError(&c.Controller, http.StatusBadRequest,
				errorEscritura.Error(), nil)
			return
		}

		if helper.EsErrorFatalEscritura(err) {
			if esTimeout(err) {
				utils.WriteError(&c.Controller, http.StatusGatewayTimeout,
					"Tiempo de espera agotado al escribir en terceros_crud", nil)
				return
			}
			utils.AbortWithError(&c.Controller, http.StatusBadGateway,
				"No fue posible escribir en terceros_crud", err)
			return
		}

		if esTimeout(err) {
			utils.WriteError(&c.Controller, http.StatusGatewayTimeout,
				"Tiempo de espera agotado al consultar terceros_crud", nil)
			return
		}
		utils.AbortWithError(&c.Controller, http.StatusBadGateway,
			"No fue posible procesar el registro del tercero", err)
	}
}

func esTimeout(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
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

// dataDeResultado adapta el subárbol del motor al contrato del dominio 1.
func dataDeResultado(resultado services.TerceroWriteResult) models.RegistroTerceroData {
	basicos := copiarMapa(resultado.Data.Data)
	identificacion := map[string]any{}
	if valor, ok := basicos["DatosIdentificacion"]; ok {
		identificacion = primerMapa(valor)
		delete(basicos, "DatosIdentificacion")
	}
	return models.RegistroTerceroData{
		TerceroId:             resultado.TerceroId,
		DatosBasicos:          basicos,
		DatosIdentificacion:   identificacion,
		DatosIdentificacionId: resultado.DatosIdentificacionId,
		Resultado: models.Resultado{
			Creados:      listaEntidades(resultado.Creados),
			Reutilizados: listaEntidades(resultado.Reutilizados),
			Actualizados: listaEntidades(resultado.Actualizados),
		},
	}
}

func listaEntidades(valores []models.EntidadId) []models.EntidadId {
	if valores == nil {
		return []models.EntidadId{}
	}
	return valores
}

func copiarMapa(original map[string]any) map[string]any {
	copia := make(map[string]any, len(original))
	for k, v := range original {
		copia[k] = v
	}
	return copia
}

func primerMapa(valor any) map[string]any {
	switch v := valor.(type) {
	case map[string]any:
		return v
	case []any:
		for _, elem := range v {
			if m, ok := elem.(map[string]any); ok {
				return m
			}
		}
	case []map[string]any:
		if len(v) > 0 {
			return v[0]
		}
	}
	return map[string]any{}
}
