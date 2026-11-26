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

	"github.com/udistrital/agora_mid/clients"
	"github.com/udistrital/agora_mid/helper"
	"github.com/udistrital/agora_mid/models"
	"github.com/udistrital/agora_mid/services"
	"github.com/udistrital/agora_mid/utils"
)

// maxCuerpoTercero limita el tamaño del JSON de entrada del dominio 1.
const maxCuerpoTercero = 64 * 1024

// decodificarCuerpo lee el cuerpo de la petición con campos desconocidos
// rechazados, sin datos de cola y con Content-Type y tamaño controlados.
func decodificarCuerpo(c *beego.Controller, destino any) bool {
	tipoContenido := strings.TrimSpace(c.Ctx.Input.Header("Content-Type"))
	if tipoContenido != "" && !strings.HasPrefix(strings.ToLower(tipoContenido), "application/json") {
		utils.AbortWithError(c, http.StatusBadRequest,
			"El Content-Type debe ser application/json", nil)
		return false
	}

	cuerpo := c.Ctx.Input.RequestBody
	if len(cuerpo) == 0 && c.Ctx.Request.Body != nil {
		cuerpo, _ = io.ReadAll(c.Ctx.Request.Body)
	}
	if len(cuerpo) > maxCuerpoTercero {
		utils.AbortWithError(c, http.StatusBadRequest,
			"El cuerpo de la petición supera el tamaño máximo permitido", nil)
		return false
	}

	dec := json.NewDecoder(bytes.NewReader(cuerpo))
	dec.DisallowUnknownFields()
	if err := dec.Decode(destino); err != nil {
		utils.AbortWithError(c, http.StatusBadRequest,
			"La petición no es un JSON válido o contiene campos no permitidos", err)
		return false
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		utils.AbortWithError(c, http.StatusBadRequest,
			"La petición debe contener un único objeto JSON", nil)
		return false
	}
	return true
}

// leerTerceroID toma el parámetro de ruta y exige un entero positivo.
func leerTerceroID(c *beego.Controller) (int64, bool) {
	valor := c.Ctx.Input.Param(":terceroId")
	if strings.TrimSpace(valor) == "" {
		valor = c.Ctx.Input.Param(":id")
	}
	id, err := strconv.ParseInt(strings.TrimSpace(valor), 10, 64)
	if err != nil || id <= 0 {
		utils.AbortWithError(c, http.StatusBadRequest,
			"terceroId debe ser un entero positivo", err)
		return 0, false
	}
	return id, true
}

// escribirRespuestaTercero publica el resultado con el envelope del MID:
// 206 si hubo avisos de materialización parcial, 201 si se creó algo y 200 si
// solo hubo actualización o reutilización.
func escribirRespuestaTercero(
	c *beego.Controller,
	resultado services.TerceroWriteResult,
	data any,
	mensajeCreado string,
	mensajeActualizado string,
) {
	switch {
	case resultado.Parcial:
		utils.WritePartial(c, "Registro parcial", data, resultado.Avisos)
	case len(resultado.Creados) > 0:
		utils.WriteSuccessWithWarnings(c, http.StatusCreated,
			mensajeCreado, data, resultado.Avisos)
	default:
		utils.WriteSuccessWithWarnings(c, http.StatusOK,
			mensajeActualizado, data, resultado.Avisos)
	}
}

// responderErrorTercero mapea los errores del dominio 1 a códigos HTTP sin
// exponer detalles internos de los CRUD. codigo es P_NATURAL o P_JURIDICA y solo
// se usa en el mensaje de configuración/catálogo.
func responderErrorTercero(c *beego.Controller, err error, codigo string) {
	var validacion *services.ValidationError
	if errors.As(err, &validacion) {
		utils.AbortWithError(c, http.StatusBadRequest, validacion.Message, nil)
		return
	}

	switch {
	case errors.Is(err, services.ErrTipoContribuyenteInvalido),
		errors.Is(err, services.ErrTipoContribuyenteJuridicoInvalido):
		utils.AbortWithError(c, http.StatusBadRequest, err.Error(), nil)

	case errors.Is(err, services.ErrTipoContribuyenteConflicto):
		utils.AbortWithError(c, http.StatusConflict, err.Error(), nil)

	case errors.Is(err, services.ErrTerceroNoEncontrado):
		utils.AbortWithError(c, http.StatusNotFound, err.Error(), nil)

	case errors.Is(err, services.ErrTerceroInactivo),
		errors.Is(err, services.ErrIdentificacionInactiva):
		utils.AbortWithError(c, http.StatusConflict, err.Error(), nil)

	case errors.Is(err, services.ErrIdentificacionDeOtroTercero),
		errors.Is(err, services.ErrIdentificacionInmutable):
		utils.AbortWithError(c, http.StatusConflict, err.Error(), nil)

	case errors.Is(err, services.ErrCatalogoNoEncontrado),
		errors.Is(err, services.ErrCatalogoAmbiguo),
		errors.Is(err, services.ErrTipoContribuyenteNoConfigurado),
		errors.Is(err, services.ErrTipoContribuyenteJuridicoNoConfigurado),
		errors.Is(err, services.ErrTipoDocumentoNITNoConfigurado):
		utils.AbortWithError(c, http.StatusInternalServerError,
			"No fue posible resolver los catálogos de "+codigo, err)

	default:
		var errorEscritura *helper.ErrorEscritura
		if errors.As(err, &errorEscritura) {
			utils.AbortWithError(c, http.StatusBadRequest,
				errorEscritura.Error(), nil)
			return
		}

		var httpErr *clients.HTTPStatusError
		if errors.As(err, &httpErr) {
			switch httpErr.StatusCode {
			case http.StatusBadRequest:
				utils.AbortWithError(c, http.StatusBadRequest,
					"Los datos fueron rechazados por terceros_crud", nil)
			case http.StatusConflict:
				utils.AbortWithError(c, http.StatusConflict,
					"Conflicto al escribir en terceros_crud", nil)
			default:
				utils.AbortWithError(c, http.StatusBadGateway,
					"No fue posible procesar el registro del tercero", err)
			}
			return
		}

		if helper.EsErrorFatalEscritura(err) {
			if esTimeout(err) {
				utils.WriteError(c, http.StatusGatewayTimeout,
					"Tiempo de espera agotado al escribir en terceros_crud", nil)
				return
			}
			utils.AbortWithError(c, http.StatusBadGateway,
				"No fue posible escribir en terceros_crud", err)
			return
		}

		if esTimeout(err) {
			utils.WriteError(c, http.StatusGatewayTimeout,
				"Tiempo de espera agotado al consultar terceros_crud", nil)
			return
		}
		utils.AbortWithError(c, http.StatusBadGateway,
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

// dataDeResultado adapta el subárbol del motor al contrato del dominio 1 y
// normaliza las FK a la forma { "Id": n }, incluso cuando el motor reutiliza un
// registro sin cambios y lo devuelve aplanado.
func dataDeResultado(resultado services.TerceroWriteResult) models.RegistroTerceroData {
	basicos := copiarMapa(resultado.Data.Data)
	identificacion := map[string]any{}
	if valor, ok := basicos["DatosIdentificacion"]; ok {
		identificacion = primerMapa(valor)
		delete(basicos, "DatosIdentificacion")
	}
	normalizarFK(basicos, "TipoContribuyenteId")
	normalizarFK(identificacion, "TipoDocumentoId")
	normalizarFK(identificacion, "TerceroId")
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

// normalizarFK convierte un escalar o un objeto con Id en { "Id": n }.
func normalizarFK(mapa map[string]any, campo string) {
	valor, ok := mapa[campo]
	if !ok || valor == nil {
		return
	}
	if objeto, ok := valor.(map[string]any); ok {
		if id := idDeValor(objeto["Id"]); id > 0 {
			mapa[campo] = map[string]any{"Id": id}
		}
		return
	}
	if id := idDeValor(valor); id > 0 {
		mapa[campo] = map[string]any{"Id": id}
	}
}

func idDeValor(valor any) int64 {
	switch v := valor.(type) {
	case int:
		return int64(v)
	case int64:
		return v
	case float64:
		return int64(v)
	case json.Number:
		id, _ := v.Int64()
		return id
	case string:
		id, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		return id
	default:
		return 0
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
