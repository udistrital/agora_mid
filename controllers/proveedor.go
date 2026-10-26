package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/udistrital/agora_mid/helper"
	"github.com/udistrital/agora_mid/responses"
	"github.com/udistrital/agora_mid/services"
)

type proveedorReader interface {
	LeerProveedor(
		ctx context.Context,
		id int,
		fields []string,
	) (helper.ExpandResult, error)
}

// proveedorWriter materializa el subárbol del proveedor a partir de un cuerpo
// anidado. verb es services' WriteVerbCreate (POST) o WriteVerbUpdate (PATCH).
type proveedorWriter interface {
	EscribirProveedor(
		ctx context.Context,
		body map[string]any,
		verb string,
	) (helper.WriteResult, error)
}

type ProveedorHandler struct {
	servicio  proveedorReader
	escritura proveedorWriter
}

func NewProveedorHandler(servicio proveedorReader) *ProveedorHandler {
	return &ProveedorHandler{servicio: servicio}
}

// NewProveedorHandlerCompleto construye un handler con los servicios de lectura y
// escritura del proveedor.
func NewProveedorHandlerCompleto(
	lectura proveedorReader,
	escritura proveedorWriter,
) *ProveedorHandler {
	return &ProveedorHandler{servicio: lectura, escritura: escritura}
}

func (h *ProveedorHandler) Leer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		h.responder(w, http.StatusMethodNotAllowed, false,
			"Método no permitido", nil, nil)
		return
	}

	query := r.URL.Query()
	idTexto := query.Get("id")
	if idTexto == "" {
		h.responder(w, http.StatusBadRequest, false,
			"El parámetro id es obligatorio", nil, nil)
		return
	}

	id, err := strconv.Atoi(idTexto)
	if err != nil || id <= 0 {
		h.responder(w, http.StatusBadRequest, false,
			"id debe ser un entero positivo", nil, nil)
		return
	}

	var fields []string
	if query.Has("fields") {
		fieldsTexto := query.Get("fields")
		if strings.TrimSpace(fieldsTexto) == "" {
			h.responder(w, http.StatusBadRequest, false,
				"El parámetro fields no puede estar vacío", nil, nil)
			return
		}

		for _, field := range strings.Split(fieldsTexto, ",") {
			field = strings.TrimSpace(field)
			if field == "" {
				h.responder(w, http.StatusBadRequest, false,
					"El parámetro fields contiene un campo vacío", nil, nil)
				return
			}
			fields = append(fields, field)
		}
	}

	resultado, err := h.servicio.LeerProveedor(r.Context(), id, fields)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrProveedorIDInvalido):
			h.responder(w, http.StatusBadRequest, false,
				"El id debe ser un entero positivo", nil, nil)

		case errors.Is(err, services.ErrProveedorNoEncontrado):
			h.responder(w, http.StatusNotFound, false,
				"No existe un proveedor con el id solicitado", nil, nil)

		default:
			var errorFields *helper.ErrorFields
			if errors.As(err, &errorFields) {
				h.responder(w, http.StatusBadRequest, false,
					errorFields.Error(), nil, nil)
				return
			}

			if helper.EsErrorFatalExpansion(err) {
				status := http.StatusBadGateway
				var netErr net.Error
				if errors.Is(err, context.DeadlineExceeded) ||
					errors.As(err, &netErr) && netErr.Timeout() {
					status = http.StatusGatewayTimeout
				}
				h.responder(w, status, false,
					http.StatusText(status), nil, nil)
				return
			}

			// No exponer mensajes internos de los CRUD.
			h.responder(w, http.StatusBadGateway, false,
				"Error al consultar el proveedor o sus relaciones", nil, nil)
		}
		return
	}

	status := http.StatusOK
	message := "Proveedor consultado correctamente"
	if len(resultado.Warnings) > 0 {
		status = http.StatusPartialContent
		message = "Proveedor consultado con advertencias de expansión"
	}

	h.responder(w, status, true, message, resultado.Data, resultado.Warnings)
}

// consultaProveedorBody es el cuerpo JSON aceptado por el endpoint de consulta
// POST /v1/proveedor/consulta. id es el identificador del proveedor (entero
// positivo) y fields es la selección de rutas canónicas a expandir.
type consultaProveedorBody struct {
	ID     *int     `json:"id"`
	Fields []string `json:"fields"`
}

// Consultar atiende POST sobre /v1/proveedor/consulta. Decodifica el cuerpo JSON
// {id, fields}, valida que id sea un entero positivo y que fields sea un arreglo
// de rutas no vacías, y delega la lectura en el servicio. Mapea el resultado al
// contrato de respuesta:
//   - 200: lectura correcta sin advertencias.
//   - 206 + MetaData.Errors: lectura con advertencias de expansión.
//   - 400: cuerpo JSON malformado, id ausente/no positivo o fields inválido
//     (ruta vacía o no permitida), sin consultar el CRUD dueño.
//   - 404: no existe un proveedor con el id solicitado.
//   - 502/504: fallo bloqueante al consultar una relación; 504 cuando el fallo
//     es por tiempo de espera agotado.
//
// La validación estricta de cada ruta de fields la realiza el servicio
// reutilizando helper.ParsearFields, por lo que un fields inválido se rechaza
// con 400 antes de consultar el proveedor base.
func (h *ProveedorHandler) Consultar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		h.responder(w, http.StatusMethodNotAllowed, false,
			"Método no permitido", nil, nil)
		return
	}

	var body consultaProveedorBody
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&body); err != nil {
		h.responder(w, http.StatusBadRequest, false,
			"El cuerpo debe ser un objeto JSON válido", nil, nil)
		return
	}

	if body.ID == nil || *body.ID <= 0 {
		h.responder(w, http.StatusBadRequest, false,
			"id debe ser un entero positivo", nil, nil)
		return
	}

	var fields []string
	for _, field := range body.Fields {
		field = strings.TrimSpace(field)
		if field == "" {
			h.responder(w, http.StatusBadRequest, false,
				"El parámetro fields contiene un campo vacío", nil, nil)
			return
		}
		fields = append(fields, field)
	}

	resultado, err := h.servicio.LeerProveedor(r.Context(), *body.ID, fields)
	if err != nil {
		h.responderErrorLectura(w, err)
		return
	}

	status := http.StatusOK
	message := "Proveedor consultado correctamente"
	if len(resultado.Warnings) > 0 {
		status = http.StatusPartialContent
		message = "Proveedor consultado con advertencias de expansión"
	}

	h.responder(w, status, true, message, resultado.Data, resultado.Warnings)
}

// responderErrorLectura mapea el error del servicio de lectura al código HTTP del
// contrato, sin exponer mensajes internos de los CRUD.
func (h *ProveedorHandler) responderErrorLectura(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrProveedorIDInvalido):
		h.responder(w, http.StatusBadRequest, false,
			"El id debe ser un entero positivo", nil, nil)

	case errors.Is(err, services.ErrProveedorNoEncontrado):
		h.responder(w, http.StatusNotFound, false,
			"No existe un proveedor con el id solicitado", nil, nil)

	default:
		var errorFields *helper.ErrorFields
		if errors.As(err, &errorFields) {
			h.responder(w, http.StatusBadRequest, false,
				errorFields.Error(), nil, nil)
			return
		}

		if helper.EsErrorFatalExpansion(err) {
			status := http.StatusBadGateway
			var netErr net.Error
			if errors.Is(err, context.DeadlineExceeded) ||
				errors.As(err, &netErr) && netErr.Timeout() {
				status = http.StatusGatewayTimeout
			}
			h.responder(w, status, false,
				http.StatusText(status), nil, nil)
			return
		}

		// No exponer mensajes internos de los CRUD.
		h.responder(w, http.StatusBadGateway, false,
			"Error al consultar el proveedor o sus relaciones", nil, nil)
	}
}

// Escribir atiende POST y PATCH sobre /v1/proveedor. Decodifica el cuerpo
// anidado, inyecta el autor de la modificación desde la autenticación y delega
// la materialización en el servicio de escritura, mapeando el resultado al
// contrato de respuesta:
//   - 200: creación/actualización completa sin avisos.
//   - 206 + MetaData.Errors: materialización parcial tolerada.
//   - 400: cuerpo o campo inválido (estructura, campo no escribible, inmutable o
//     rama de detalle incoherente), sin materializar.
//   - 502/504: fallo bloqueante de una relación (política estricta); 504 cuando
//     el fallo es por tiempo de espera agotado.
func (h *ProveedorHandler) Escribir(w http.ResponseWriter, r *http.Request) {
	verb, ok := verboEscritura(r.Method)
	if !ok {
		w.Header().Set("Allow", http.MethodPost+", "+http.MethodPatch)
		h.responder(w, http.StatusMethodNotAllowed, false,
			"Método no permitido", nil, nil)
		return
	}

	var body map[string]any
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&body); err != nil || body == nil {
		h.responder(w, http.StatusBadRequest, false,
			"El cuerpo debe ser un objeto JSON válido", nil, nil)
		return
	}

	ctx := r.Context()
	if autor, ok := autorDePeticion(r); ok {
		ctx = services.ConContextoAutor(ctx, autor)
	}

	resultado, err := h.escritura.EscribirProveedor(ctx, body, verb)
	if err != nil {
		h.responderErrorEscritura(w, err)
		return
	}

	status := http.StatusOK
	message := "Proveedor registrado correctamente"
	if len(resultado.Warnings) > 0 {
		status = http.StatusPartialContent
		message = "Proveedor registrado con advertencias de materialización"
	}
	h.responder(w, status, true, message, resultado.Data, resultado.Warnings)
}

// responderErrorEscritura mapea el error del servicio de escritura al código HTTP
// del contrato, sin exponer mensajes internos de los CRUD.
func (h *ProveedorHandler) responderErrorEscritura(w http.ResponseWriter, err error) {
	// 400: cuerpo o campo inválido, o rama de detalle incoherente.
	var errEscritura *helper.ErrorEscritura
	if errors.As(err, &errEscritura) {
		h.responder(w, http.StatusBadRequest, false, errEscritura.Error(), nil, nil)
		return
	}
	if errors.Is(err, services.ErrRamaDetalleIncoherente) {
		h.responder(w, http.StatusBadRequest, false, err.Error(), nil, nil)
		return
	}

	// 502/504: fallo bloqueante de una relación bajo política estricta.
	if helper.EsErrorFatalEscritura(err) {
		status := http.StatusBadGateway
		var netErr net.Error
		if errors.Is(err, context.DeadlineExceeded) ||
			errors.As(err, &netErr) && netErr.Timeout() {
			status = http.StatusGatewayTimeout
		}
		h.responder(w, status, false, http.StatusText(status), nil, nil)
		return
	}

	// No exponer mensajes internos de los CRUD ante un error no clasificado.
	h.responder(w, http.StatusBadGateway, false,
		"Error al registrar el proveedor o sus relaciones", nil, nil)
}

// verboEscritura traduce el método HTTP al verbo de escritura del motor. Devuelve
// false para métodos no soportados por el endpoint de escritura.
func verboEscritura(metodo string) (string, bool) {
	switch metodo {
	case http.MethodPost:
		return helper.WriteVerbCreate, true
	case http.MethodPatch:
		return helper.WriteVerbUpdate, true
	default:
		return "", false
	}
}

// autorDePeticion extrae el identificador del autor de la modificación desde la
// cabecera X-Autor-Id, poblada por la capa de autenticación. Devuelve 0 y false
// cuando la cabecera está ausente o no es un entero positivo.
func autorDePeticion(r *http.Request) (int, bool) {
	valor := strings.TrimSpace(r.Header.Get("X-Autor-Id"))
	if valor == "" {
		return 0, false
	}
	autor, err := strconv.Atoi(valor)
	if err != nil || autor <= 0 {
		return 0, false
	}
	return autor, true
}

func (h *ProveedorHandler) responder(
	w http.ResponseWriter,
	status int,
	success bool,
	message string,
	data any,
	warnings []responses.RelationError,
) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	response := responses.WithWarnings(
		success,
		strconv.Itoa(status),
		message,
		data,
		warnings,
	)
	_ = json.NewEncoder(w).Encode(response)
}
