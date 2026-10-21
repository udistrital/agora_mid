// Package responses define el contrato de respuesta uniforme del API en formato
// OAS plano (Success/Status/Message/Data) con un bloque opcional MetaData para
// avisos de expansión de objetos anidados y el total de las listas.
package responses

// RelationError describe un aviso de expansión de relaciones. Nunca debe incluir
// URLs internas, credenciales ni cuerpos remotos: solo la ruta lógica.
type RelationError struct {
	Path    string `json:"path"`
	Service string `json:"service"`
	Message string `json:"message"`
	// Kind identifica el origen del aviso: relation, depth, cycle o
	// generic_resolver.
	Kind string `json:"kind"`
}

// MetaData agrupa información adicional de la respuesta. Se serializa únicamente
// cuando aporta algún dato (avisos de expansión o el total de una lista); de lo
// contrario se omite del envoltorio.
type MetaData struct {
	// Errors contiene los avisos de expansión. Es opcional: se omite cuando no
	// hay avisos.
	Errors []RelationError `json:"Errors,omitempty"`
	// Length refleja, para un Data de tipo lista, el número total de valores
	// posibles sin considerar limit u offset (no el tamaño de la página
	// devuelta). Es opcional: se omite cuando Data no es una lista o no aplica.
	Length *int `json:"Length,omitempty"`
}

// Response es el envoltorio OAS plano usado en todos los verbos del API.
//
// MetaData es opcional y se incluye solo cuando aporta datos (avisos de
// expansión o el total de una lista); en caso contrario se omite.
type Response struct {
	Success  bool      `json:"Success"`
	Status   string    `json:"Status"`
	Message  string    `json:"Message"`
	Data     any       `json:"Data"`
	MetaData *MetaData `json:"MetaData,omitempty"`
}

// New construye un envoltorio plano sin bloque MetaData.
func New(success bool, status, message string, data any) Response {
	return Response{
		Success: success,
		Status:  status,
		Message: message,
		Data:    data,
	}
}

// WithWarnings construye un envoltorio incluyendo MetaData.Errors a partir de los
// avisos de expansión. Si no hay avisos, MetaData se omite para mantener la forma
// consistente con las respuestas que no generan avisos.
func WithWarnings(success bool, status, message string, data any, warnings []RelationError) Response {
	resp := New(success, status, message, data)
	resp.setMetaData(warnings, nil)
	return resp
}

// WithLength construye un envoltorio incluyendo MetaData.Length para un Data de
// tipo lista. length representa el número total de valores posibles sin
// considerar limit u offset.
func WithLength(success bool, status, message string, data any, length int) Response {
	resp := New(success, status, message, data)
	resp.setMetaData(nil, &length)
	return resp
}

// WithMetaData construye un envoltorio con avisos y/o Length. El bloque MetaData
// se omite si no aporta ningún dato.
func WithMetaData(success bool, status, message string, data any, warnings []RelationError, length *int) Response {
	resp := New(success, status, message, data)
	resp.setMetaData(warnings, length)
	return resp
}

// setMetaData asigna el bloque MetaData solo cuando aporta algún dato (avisos o
// Length), omitiéndolo en caso contrario para mantener la forma consistente.
func (r *Response) setMetaData(warnings []RelationError, length *int) {
	if len(warnings) == 0 && length == nil {
		r.MetaData = nil
		return
	}
	meta := &MetaData{Length: length}
	if len(warnings) > 0 {
		meta.Errors = warnings
	}
	r.MetaData = meta
}
