package models

import "encoding/json"

// RegistroCaracterizacionNatural es el cuerpo de POST y PUT del dominio 2 para
// persona natural. Todas las secciones son opcionales: un guardado parcial solo
// escribe lo que llega.
type RegistroCaracterizacionNatural struct {
	TerceroId                   int64                              `json:"TerceroId"`
	InfoComplementariaTercero   []InfoComplementariaTerceroInput   `json:"InfoComplementariaTercero"`
	ProveedorNatural            *ProveedorNaturalInput             `json:"ProveedorNatural"`
	PerfilProveedor             []PerfilProveedorInput             `json:"PerfilProveedor"`
	Contacto                    []ContactoProveedorInput           `json:"Contacto"`
	ActividadEconomicaProveedor []ActividadEconomicaProveedorInput `json:"ActividadEconomicaProveedor"`
	DescripcionPortafolio       *string                            `json:"DescripcionPortafolio"`
	Consentimiento              *bool                              `json:"Consentimiento"`
}

// InfoComplementariaTerceroInput identifica un dato de caracterización. Debe
// traer InfoComplementariaId o CodigoInfoComplementaria, no ambos a la vez.
// Cuando se usa CodigoInfoComplementaria también se debe indicar el código del
// grupo: los Ids cambian entre ambientes, pero (código, grupo) es estable.
type InfoComplementariaTerceroInput struct {
	Id                               *int64          `json:"Id,omitempty"`
	InfoComplementariaId             *int64          `json:"InfoComplementariaId,omitempty"`
	CodigoInfoComplementaria         *string         `json:"CodigoInfoComplementaria,omitempty"`
	CodigoGrupoInfoComplementaria    *string         `json:"CodigoGrupoInfoComplementaria,omitempty"`
	Dato                             json.RawMessage `json:"Dato"`
	InfoComplementariaTerceroPadreId *int64          `json:"InfoComplementariaTerceroPadreId,omitempty"`
}

// ProveedorNaturalInput contiene los campos mutables de proveedor_natural. Los
// punteros distinguen un campo ausente de un valor cero.
type ProveedorNaturalInput struct {
	PaisNacimientoId                  *int64   `json:"PaisNacimientoId,omitempty"`
	Cargo                             *string  `json:"Cargo,omitempty"`
	MedicinaPrepagada                 *bool    `json:"MedicinaPrepagada,omitempty"`
	ValorUvtPrepagada                 *float64 `json:"ValorUvtPrepagada,omitempty"`
	InteresViviendaAfc                *float64 `json:"InteresViviendaAfc,omitempty"`
	DependienteHijoMejorEdad          *bool    `json:"DependienteHijoMejorEdad,omitempty"`
	DependienteHijoMenos23Estudiando  *bool    `json:"DependienteHijoMenos23Estudiando,omitempty"`
	DependienteHijoMas23Discapacitado *bool    `json:"DependienteHijoMas23Discapacitado,omitempty"`
	DependienteConyuge                *bool    `json:"DependienteConyuge,omitempty"`
	DependientePadreOHermano          *bool    `json:"DependientePadreOHermano,omitempty"`
	Pensionado                        *bool    `json:"Pensionado,omitempty"`
	ValorAfc                          *float64 `json:"ValorAfc,omitempty"`
	ResponsableIva                    *bool    `json:"ResponsableIva,omitempty"`
	ResideColombia                    *bool    `json:"ResideColombia,omitempty"`
	PensionVoluntaria                 *bool    `json:"PensionVoluntaria,omitempty"`
	ValorPensionVoluntaria            *float64 `json:"ValorPensionVoluntaria,omitempty"`
	ExperienciaLaboral                *int64   `json:"ExperienciaLaboral,omitempty"`
	ExperienciaProfesional            *int64   `json:"ExperienciaProfesional,omitempty"`
}

// PerfilProveedorInput referencia un perfil del catálogo Grupo_5. Se debe
// enviar exactamente uno de PerfilId o CodigoPerfil; el código es estable entre
// ambientes.
type PerfilProveedorInput struct {
	Id           *int64  `json:"Id,omitempty"`
	PerfilId     int64   `json:"PerfilId,omitempty"`
	CodigoPerfil *string `json:"CodigoPerfil,omitempty"`
}

// ContactoProveedorInput representa un contacto del proveedor.
type ContactoProveedorInput struct {
	Id                  *int64  `json:"Id,omitempty"`
	TipoCanalContactoId int64   `json:"TipoCanalContactoId"`
	Proposito           string  `json:"Proposito"`
	ValorEspecifico     string  `json:"ValorEspecifico"`
	NombreContacto      *string `json:"NombreContacto,omitempty"`
	Extension           *string `json:"Extension,omitempty"`
}

// ActividadEconomicaProveedorInput referencia una actividad CIIU.
type ActividadEconomicaProveedorInput struct {
	Id                   *int64 `json:"Id,omitempty"`
	ActividadEconomicaId int64  `json:"ActividadEconomicaId"`
}

// InfoComplementariaTerceroRegistrada es la proyección de respuesta de un dato
// de caracterización.
type InfoComplementariaTerceroRegistrada struct {
	Id                               int64  `json:"Id"`
	InfoComplementariaId             int64  `json:"InfoComplementariaId"`
	CodigoInfoComplementaria         string `json:"CodigoInfoComplementaria,omitempty"`
	CodigoGrupoInfoComplementaria    string `json:"CodigoGrupoInfoComplementaria,omitempty"`
	Dato                             any    `json:"Dato"`
	InfoComplementariaTerceroPadreId *int64 `json:"InfoComplementariaTerceroPadreId,omitempty"`
}

// CaracterizacionNaturalData es el contenido de Data en la respuesta.
type CaracterizacionNaturalData struct {
	TerceroId                   int64                                 `json:"TerceroId"`
	ProveedorId                 int64                                 `json:"ProveedorId"`
	Proveedor                   map[string]any                        `json:"Proveedor,omitempty"`
	InfoComplementariaTercero   []InfoComplementariaTerceroRegistrada `json:"InfoComplementariaTercero"`
	ProveedorNatural            map[string]any                        `json:"ProveedorNatural,omitempty"`
	PerfilProveedor             []EntidadId                           `json:"PerfilProveedor"`
	Contacto                    []EntidadId                           `json:"Contacto"`
	ActividadEconomicaProveedor []EntidadId                           `json:"ActividadEconomicaProveedor"`
	Resultado                   Resultado                             `json:"Resultado"`
}
