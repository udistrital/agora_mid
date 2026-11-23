// Package controllers expone los controladores Beego del middleware.
package controllers

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	beego "github.com/beego/beego/v2/server/web"

	"github.com/udistrital/agora_mid/models"
	"github.com/udistrital/agora_mid/services"
)

// ProveedorNaturalController expone el registro de proveedores persona natural.
type ProveedorNaturalController struct {
	beego.Controller
}

// URLMapping registra los métodos del controlador.
func (c *ProveedorNaturalController) URLMapping() {
	c.Mapping("Post", c.Post)
}

// Post registra un proveedor persona natural.
// @Title Post
// @Description Registra un proveedor persona natural (tercero + identificación + proveedor)
// @Param	body	body	models.RegistroProveedorNatural	true	"cuerpo del registro"
// @Success 201 {object} models.RegistroProveedorData
// @Success 200 {object} models.RegistroProveedorData
// @Failure 400 petición inválida
// @router /natural [post]
func (c *ProveedorNaturalController) Post() {
	cuerpo := c.Ctx.Input.RequestBody
	if len(cuerpo) == 0 && c.Ctx.Request.Body != nil {
		cuerpo, _ = io.ReadAll(c.Ctx.Request.Body)
	}
	var in models.RegistroProveedorNatural
	if err := json.Unmarshal(cuerpo, &in); err != nil {
		c.responder(http.StatusBadRequest, "La petición no es un JSON válido", nil)
		return
	}
	if msg, ok := validar(in); !ok {
		c.responder(http.StatusBadRequest, msg, nil)
		return
	}

	data, creado, err := services.RegistrarProveedorNatural(c.Ctx.Request.Context(), in)
	if err != nil {
		// Error público neutro; el host de los CRUD no aparece en el mensaje.
		c.responder(http.StatusInternalServerError, err.Error(), nil)
		return
	}

	status := http.StatusOK
	mensaje := "Registro procesado"
	if creado {
		status = http.StatusCreated
		mensaje = "Registro creado"
	}
	c.responder(status, mensaje, data)
}

// validar comprueba los campos obligatorios de la petición.
func validar(in models.RegistroProveedorNatural) (string, bool) {
	if in.TipoProveedor == 0 {
		return "TipoProveedor es obligatorio", false
	}
	if strings.TrimSpace(in.DatosIdentificacion.Numero) == "" {
		return "DatosIdentificacion.Numero es obligatorio", false
	}
	if in.DatosIdentificacion.TipoDocumentoId == 0 {
		return "DatosIdentificacion.TipoDocumentoId es obligatorio", false
	}
	if strings.TrimSpace(in.DatosBasicos.PrimerNombre) == "" || strings.TrimSpace(in.DatosBasicos.PrimerApellido) == "" {
		return "DatosBasicos.PrimerNombre y PrimerApellido son obligatorios", false
	}
	return "", true
}

// responder escribe el envoltorio estándar del middleware (Status, Message, Data).
func (c *ProveedorNaturalController) responder(status int, mensaje string, data any) {
	c.Ctx.Output.SetStatus(status)
	c.Data["json"] = map[string]any{
		"Status":  status,
		"Message": mensaje,
		"Data":    data,
	}
	_ = c.ServeJSON()
}
