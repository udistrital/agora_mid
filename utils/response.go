// Package utils reúne componentes compartidos por la capa de servicios:
// el formato de respuesta estándar de la OAS y el cliente HTTP reutilizable
// para consumir los CRUD externos.
package utils

import (
	"strconv"

	"github.com/beego/beego/v2/core/logs"
	beego "github.com/beego/beego/v2/server/web"

	"github.com/udistrital/agora_mid/responses"
)

// Response es el formato de respuesta estándar de la OAS que devuelven los
// CRUD externos (el mismo de gestion_contractual_crud). La capa de servicios
// entrega este tipo a sus consumidores.
type Response struct {
	Success  bool        `json:"Success"`
	Status   int         `json:"Status"`
	Message  string      `json:"Message"`
	Data     interface{} `json:"Data"`
	Metadata *Metadata   `json:"Metadata,omitempty"`
}

// Metadata es la paginación que el CRUD devuelve en el listado (GetAll). En
// las respuestas de GetOne, Post y Put queda nil y se omite por omitempty.
type Metadata struct {
	Total           int64 `json:"total"`
	Limit           int64 `json:"limit"`
	Offset          int64 `json:"offset"`
	CurrentPage     int64 `json:"currentPage"`
	TotalPages      int64 `json:"totalPages"`
	HasNextPage     bool  `json:"hasNextPage"`
	HasPreviousPage bool  `json:"hasPreviousPage"`
}

// APIResponse es el envelope estándar del MID: Success, Status string,
// Message, Data y un bloque MetaData opcional para avisos.
type APIResponse struct {
	Success  bool      `json:"Success"`
	Status   string    `json:"Status"`
	Message  string    `json:"Message"`
	Data     any       `json:"Data"`
	MetaData *MetaData `json:"MetaData,omitempty"`
}

// MetaData agrupa los avisos del MID. Se omite cuando no hay avisos.
type MetaData struct {
	Errors []responses.RelationError `json:"Errors,omitempty"`
}

// WriteSuccess escribe una respuesta exitosa con el envelope del MID.
func WriteSuccess(c *beego.Controller, status int, message string, data any) {
	c.Ctx.Output.SetStatus(status)
	c.Data["json"] = APIResponse{
		Success: true,
		Status:  strconv.Itoa(status),
		Message: message,
		Data:    data,
	}
	_ = c.ServeJSON()
}

// WriteSuccessWithWarnings escribe una respuesta exitosa e incluye los avisos
// informativos (por ejemplo ambiguous_match) en MetaData.Errors.
func WriteSuccessWithWarnings(c *beego.Controller, status int, message string, data any, avisos []responses.RelationError) {
	c.Ctx.Output.SetStatus(status)
	resp := APIResponse{
		Success: true,
		Status:  strconv.Itoa(status),
		Message: message,
		Data:    data,
	}
	if len(avisos) > 0 {
		resp.MetaData = &MetaData{Errors: avisos}
	}
	c.Data["json"] = resp
	_ = c.ServeJSON()
}

// WritePartial escribe una respuesta 206 conservando los IDs materializados y
// publicando los avisos en MetaData.Errors.
func WritePartial(c *beego.Controller, message string, data any, avisos []responses.RelationError) {
	c.Ctx.Output.SetStatus(206)
	c.Data["json"] = APIResponse{
		Success:  false,
		Status:   "206",
		Message:  message,
		Data:     data,
		MetaData: &MetaData{Errors: avisos},
	}
	_ = c.ServeJSON()
}

// WriteError escribe un error con el mismo envelope. Se usa para códigos que
// customerror no define (por ejemplo 504).
func WriteError(c *beego.Controller, status int, message string, data any) {
	c.Ctx.Output.SetStatus(status)
	c.Data["json"] = APIResponse{
		Success: false,
		Status:  strconv.Itoa(status),
		Message: message,
		Data:    data,
	}
	_ = c.ServeJSON()
}

// AbortWithError delega el error en customerror.CustomErrorController, que
// construye el envelope estándar. Solo usar con códigos que customerror define:
// 400, 401, 404, 409, 500, 501, 502 y 509.
func AbortWithError(c *beego.Controller, status int, message string, err error) {
	if err != nil {
		logs.Error("status=%d mensaje=%s causa=%v", status, message, err)
	} else {
		logs.Error("status=%d mensaje=%s", status, message)
	}
	c.Data["message"] = message
	c.Data["data"] = nil
	c.Abort(strconv.Itoa(status))
}
