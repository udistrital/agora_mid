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

type ProveedorHandler struct {
	servicio proveedorReader
}

func NewProveedorHandler(servicio proveedorReader) *ProveedorHandler {
	return &ProveedorHandler{servicio: servicio}
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
