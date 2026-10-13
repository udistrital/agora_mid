package controllers

import (
	"encoding/json"
	"net/http"

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

func (h *RegistroPersonaHandler) RegistrarNatural(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var solicitud dto.RegistroPersonaNaturalRequest
	if err := json.NewDecoder(r.Body).Decode(&solicitud); err != nil {
		http.Error(w, "JSON de entrada inválido", http.StatusBadRequest)
		return
	}

	resultado, err := h.servicio.RegistrarPersonaNatural(r.Context(), solicitud)
	h.responder(w, resultado, err)
}

func (h *RegistroPersonaHandler) RegistrarJuridica(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

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
