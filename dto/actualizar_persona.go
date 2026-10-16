package dto

import "encoding/json"

// ActualizarPersonaRequest define una actualización parcial por servicio.
// Los mapas distinguen campos omitidos de valores enviados explícitamente.
type ActualizarPersonaRequest struct {
	ProveedorID         int                        `json:"-"`
	TerceroID           int                        `json:"-"`
	Terceros            map[string]json.RawMessage `json:"-"`
	DatosIdentificacion map[string]json.RawMessage `json:"-"`
	Proveedor           map[string]json.RawMessage `json:"-"`
	ProveedorNatural    map[string]json.RawMessage `json:"-"`
	ProveedorJuridico   map[string]json.RawMessage `json:"-"`
}
