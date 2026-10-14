package controllers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/udistrital/agora_mid/clients"
	"github.com/udistrital/agora_mid/config"
	"github.com/udistrital/agora_mid/dto"
	"github.com/udistrital/agora_mid/services"
)

type RegistroPersonaHandler struct {
	servicio services.RegistroPersonaService
}

type respuestaRegistro struct {
	Success bool                              `json:"success"`
	Data    services.RegistroPersonaResultado `json:"data"`
	Error   string                            `json:"error,omitempty"`
}

type respuestaOperacion struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

func NewRegistroPersonaHandler(cfg config.Config) *RegistroPersonaHandler {
	client := clients.HTTPClient{}

	return &RegistroPersonaHandler{
		servicio: services.RegistroPersonaService{
			Terceros: services.TercerosService{
				BaseURL: cfg.TercerosCRUDURL,
				Client:  client,
			},
			GestionPersonas: services.GestionPersonasService{
				BaseURL: cfg.GestionPersonasCRUDURL,
				Client:  client,
			},
		},
	}
}

func (h *RegistroPersonaHandler) RegistrarNatural(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.registrarNatural(w, r)
	case http.MethodGet:
		h.consultar(w, r, "natural")
	case http.MethodPut:
		h.actualizar(w, r, "natural")
	default:
		w.Header().Set("Allow", "GET, POST, PUT")
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
	}
}

func (h *RegistroPersonaHandler) registrarNatural(
	w http.ResponseWriter,
	r *http.Request,
) {

	var solicitud dto.RegistroPersonaNaturalRequest
	if err := json.NewDecoder(r.Body).Decode(&solicitud); err != nil {
		http.Error(w, "JSON de entrada inválido", http.StatusBadRequest)
		return
	}

	resultado, err := h.servicio.RegistrarPersonaNatural(r.Context(), solicitud)
	h.responder(w, resultado, err)
}

func (h *RegistroPersonaHandler) RegistrarJuridica(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.registrarJuridica(w, r)
	case http.MethodGet:
		h.consultar(w, r, "juridica")
	case http.MethodPut:
		h.actualizar(w, r, "juridica")
	default:
		w.Header().Set("Allow", "GET, POST, PUT")
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
	}
}

func (h *RegistroPersonaHandler) registrarJuridica(
	w http.ResponseWriter,
	r *http.Request,
) {

	var solicitud dto.RegistroPersonaJuridicaRequest
	if err := json.NewDecoder(r.Body).Decode(&solicitud); err != nil {
		http.Error(w, "JSON de entrada inválido", http.StatusBadRequest)
		return
	}

	resultado, err := h.servicio.RegistrarPersonaJuridica(r.Context(), solicitud)
	h.responder(w, resultado, err)
}

func (h *RegistroPersonaHandler) responder(
	w http.ResponseWriter,
	resultado services.RegistroPersonaResultado,
	err error,
) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(respuestaRegistro{
			Success: false,
			Data:    resultado,
			Error:   err.Error(),
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(respuestaRegistro{
		Success: true,
		Data:    resultado,
	})
}

func (h *RegistroPersonaHandler) consultar(w http.ResponseWriter, r *http.Request, tipo string) {
	id, err := strconv.Atoi(r.URL.Query().Get("tercero_id"))
	if err != nil || id <= 0 {
		h.responderOperacion(w, nil, errors.New("tercero_id debe ser un entero mayor que cero"), http.StatusBadRequest)
		return
	}

	resultado, err := h.servicio.ConsultarPersona(r.Context(), id, tipo)
	h.responderOperacion(w, resultado, err, 0)
}

func (h *RegistroPersonaHandler) actualizar(w http.ResponseWriter, r *http.Request, tipo string) {
	var solicitud dto.ActualizarPersonaRequest
	if err := json.NewDecoder(r.Body).Decode(&solicitud); err != nil {
		h.responderOperacion(w, nil, errors.New("JSON de entrada inválido"), http.StatusBadRequest)
		return
	}
	if queryID := r.URL.Query().Get("tercero_id"); queryID != "" {
		id, err := strconv.Atoi(queryID)
		if err != nil || id <= 0 {
			h.responderOperacion(w, nil, errors.New("tercero_id debe ser un entero mayor que cero"), http.StatusBadRequest)
			return
		}
		if solicitud.TerceroID != 0 && solicitud.TerceroID != id {
			h.responderOperacion(w, nil, errors.New("tercero_id del cuerpo no coincide con el de la URL"), http.StatusBadRequest)
			return
		}
		solicitud.TerceroID = id
	}
	if solicitud.TerceroID <= 0 {
		h.responderOperacion(w, nil, errors.New("tercero_id es obligatorio en el cuerpo o la consulta"), http.StatusBadRequest)
		return
	}

	resultado, err := h.servicio.ActualizarPersona(r.Context(), solicitud, tipo)
	h.responderOperacion(w, resultado, err, 0)
}

func (h *RegistroPersonaHandler) responderOperacion(w http.ResponseWriter, data any, err error, status int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err != nil {
		if status == 0 {
			switch {
			case errors.Is(err, services.ErrActualizacionInvalida):
				status = http.StatusBadRequest
			case errors.Is(err, services.ErrPersonaNoEncontrada):
				status = http.StatusNotFound
			case errors.Is(err, services.ErrProveedorAmbiguo), errors.Is(err, services.ErrTerceroAmbiguo), errors.Is(err, services.ErrDocumentoDuplicado):
				status = http.StatusConflict
			default:
				status = http.StatusBadGateway
			}
		}
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(respuestaOperacion{Success: false, Data: data, Error: err.Error()})
		return
	}
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(respuestaOperacion{Success: true, Data: data})
}
