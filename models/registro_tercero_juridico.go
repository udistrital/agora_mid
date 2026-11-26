// Package models reúne las estructuras de entrada y salida del middleware.
package models

// RegistroTerceroJuridico es el cuerpo de POST /v1/proveedores/juridica y
// PUT /v1/proveedores/juridica/{terceroId} para el dominio 1.
type RegistroTerceroJuridico struct {
	DatosBasicos        DatosBasicosJuridico       `json:"DatosBasicos"`
	DatosIdentificacion DatosIdentificacionTercero `json:"DatosIdentificacion"`
}

// DatosBasicosJuridico contiene los datos de la persona jurídica que pertenecen
// a tercero. RazonSocial y TipoContribuyenteId son obligatorios; LugarOrigen es
// opcional. No admite nombres de persona natural.
type DatosBasicosJuridico struct {
	RazonSocial         string `json:"RazonSocial"`
	LugarOrigen         *int64 `json:"LugarOrigen,omitempty"`
	TipoContribuyenteId int64  `json:"TipoContribuyenteId"`
}
