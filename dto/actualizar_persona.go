package dto

import "encoding/json"

// ActualizarPersonaRequest define una actualización parcial por servicio.
// Los mapas distinguen campos omitidos de valores enviados explícitamente.
type ActualizarPersonaRequest struct {
	TerceroID           int                        `json:"tercero_id"`
	Terceros            map[string]json.RawMessage `json:"terceros,omitempty"`
	DatosIdentificacion map[string]json.RawMessage `json:"datos_identificacion,omitempty"`
	Proveedor           map[string]json.RawMessage `json:"proveedor,omitempty"`
	ProveedorNatural    map[string]json.RawMessage `json:"proveedor_natural,omitempty"`
	ProveedorJuridico   map[string]json.RawMessage `json:"proveedor_juridico,omitempty"`
}
