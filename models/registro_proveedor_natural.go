// Package models reúne las estructuras de entrada y salida del middleware.
package models

import "time"

// RegistroProveedorNatural es el cuerpo de POST /v1/proveedores/natural.
type RegistroProveedorNatural struct {
	TipoProveedor       int64                   `json:"TipoProveedor"`
	DatosBasicos        DatosBasicosNatural     `json:"DatosBasicos"`
	DatosIdentificacion DatosIdentificacionNatural `json:"DatosIdentificacion"`
}

// DatosBasicosNatural son los datos de la persona natural en la petición.
type DatosBasicosNatural struct {
	PrimerNombre    string    `json:"PrimerNombre"`
	SegundoNombre   string    `json:"SegundoNombre"`
	PrimerApellido  string    `json:"PrimerApellido"`
	SegundoApellido string    `json:"SegundoApellido"`
	FechaNacimiento time.Time `json:"FechaNacimiento"`
}

// DatosIdentificacionNatural son los datos de identificación en la petición.
// Numero es string para conservar ceros iniciales en todo el flujo.
type DatosIdentificacionNatural struct {
	Numero          string    `json:"Numero"`
	TipoDocumentoId int64     `json:"TipoDocumentoId"`
	CiudadExpedicion int64    `json:"CiudadExpedicion"`
	FechaExpedicion time.Time `json:"FechaExpedicion"`
}

// RegistroProveedorData es el contenido de `Data` en la respuesta. DatosBasicos
// y DatosIdentificacion guardan la respuesta efectiva de los CRUD (alta o
// actualización), por eso se tipan como map para no perder campos adicionales
// como el Id.
type RegistroProveedorData struct {
	TipoProveedor       int64          `json:"TipoProveedor"`
	DatosBasicos        map[string]any `json:"DatosBasicos"`
	DatosIdentificacion map[string]any `json:"DatosIdentificacion"`
	ProveedorId         int64          `json:"ProveedorId"`
}
