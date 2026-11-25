// Package models reúne las estructuras de entrada y salida del middleware.
package models

import "time"

// RegistroTerceroNatural es el cuerpo de POST /v1/proveedores/natural y
// PUT /v1/proveedores/natural/{terceroId} para el dominio 1.
type RegistroTerceroNatural struct {
	DatosBasicos        DatosBasicosTercero        `json:"DatosBasicos"`
	DatosIdentificacion DatosIdentificacionTercero `json:"DatosIdentificacion"`
}

// DatosBasicosTercero contiene los datos de la persona natural que pertenecen a
// tercero. Las fechas y el lugar son opcionales para permitir guardados
// parciales; TipoContribuyenteId es obligatorio y lo valida el servicio.
type DatosBasicosTercero struct {
	PrimerNombre        string     `json:"PrimerNombre"`
	SegundoNombre       string     `json:"SegundoNombre"`
	PrimerApellido      string     `json:"PrimerApellido"`
	SegundoApellido     string     `json:"SegundoApellido"`
	FechaNacimiento     *time.Time `json:"FechaNacimiento,omitempty"`
	LugarOrigen         *int64     `json:"LugarOrigen,omitempty"`
	TipoContribuyenteId int64      `json:"TipoContribuyenteId"`
}

// DatosIdentificacionTercero contiene la identificación activa del tercero.
// Numero se conserva como string para no perder ceros iniciales.
type DatosIdentificacionTercero struct {
	Numero             string     `json:"Numero"`
	TipoDocumentoId    int64      `json:"TipoDocumentoId"`
	CiudadExpedicion   *int64     `json:"CiudadExpedicion,omitempty"`
	FechaExpedicion    *time.Time `json:"FechaExpedicion,omitempty"`
	DigitoVerificacion *float64   `json:"DigitoVerificacion,omitempty"`
}

// RegistroTerceroData es el contenido de Data en la respuesta del dominio 1.
type RegistroTerceroData struct {
	TerceroId             int64          `json:"TerceroId"`
	DatosBasicos          map[string]any `json:"DatosBasicos"`
	DatosIdentificacion   map[string]any `json:"DatosIdentificacion"`
	DatosIdentificacionId int64          `json:"DatosIdentificacionId"`
	Resultado             Resultado      `json:"Resultado"`
}

// Resultado resume las entidades materializadas durante la operación.
type Resultado struct {
	Creados      []EntidadId `json:"Creados"`
	Reutilizados []EntidadId `json:"Reutilizados"`
	Actualizados []EntidadId `json:"Actualizados"`
}

// EntidadId identifica una entidad materializada y su Id.
type EntidadId struct {
	Entidad string `json:"Entidad"`
	Id      int64  `json:"Id"`
}
