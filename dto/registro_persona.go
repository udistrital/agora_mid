package dto

// DatosTercerosDTO contiene los datos de identidad que administra Terceros.
// Los códigos de catálogo se resuelven en el MID; no se reciben IDs fijos.
type DatosTercerosDTO struct {
	NombreCompleto      string   `json:"nombre_completo"`
	TerceroID           int      `json:"tercero_id,omitempty"`
	PrimerNombre        string   `json:"primer_nombre,omitempty"`
	SegundoNombre       string   `json:"segundo_nombre,omitempty"`
	PrimerApellido      string   `json:"primer_apellido,omitempty"`
	SegundoApellido     string   `json:"segundo_apellido,omitempty"`
	FechaNacimiento     *string  `json:"fecha_nacimiento,omitempty"`
	LugarOrigen         *int     `json:"lugar_origen,omitempty"`
	CodigoTipoDocumento string   `json:"codigo_tipo_documento"`
	NumeroDocumento     string   `json:"numero_documento"`
	FechaExpedicion     *string  `json:"fecha_expedicion,omitempty"`
	DigitoVerificacion  *float64 `json:"digito_verificacion,omitempty"`
	CiudadExpedicion    *int     `json:"ciudad_expedicion,omitempty"`
	UsuarioWSO2         string   `json:"usuario_wso2,omitempty"`
}

// RegistroPersonaNaturalRequest agrupa los datos para una persona natural.
type RegistroPersonaNaturalRequest struct {
	DatosTerceros    DatosTercerosDTO    `json:"-"`
	Proveedor        ProveedorDTO        `json:"-"`
	ProveedorNatural ProveedorNaturalDTO `json:"-"`
}

// RegistroPersonaJuridicaRequest agrupa los datos para una persona jurídica.
type RegistroPersonaJuridicaRequest struct {
	DatosTerceros     DatosTercerosDTO     `json:"-"`
	Proveedor         ProveedorDTO         `json:"-"`
	ProveedorJuridico ProveedorJuridicoDTO `json:"-"`
}

// ProveedorDTO contiene los campos generales propios de ÁGORA.
// tipo_registro_id se resolverá desde Terceros, no desde la entrada.
type ProveedorDTO struct {
	DescripcionPortafolio string `json:"descripcion_portafolio,omitempty"`
}

// ProveedorNaturalDTO contiene los datos específicos de persona natural.
type ProveedorNaturalDTO struct {
	PerfilDeclaradoID           int `json:"perfil_declarado_id"`
	ExperienciaLaboralMeses     int `json:"experiencia_laboral_meses"`
	ExperienciaProfesionalMeses int `json:"experiencia_profesional_meses"`
}

// ProveedorJuridicoDTO contiene los datos específicos de persona jurídica.
type ProveedorJuridicoDTO struct {
	NombreComercial          string `json:"nombre_comercial"`
	MatriculaMercantil       int    `json:"matricula_mercantil"`
	FechaConstitucion        string `json:"fecha_constitucion"`
	FechaRenovacion          string `json:"fecha_renovacion"`
	ReportaBeneficiosFinales bool   `json:"reporta_beneficios_finales"`
	CotizaBolsa              bool   `json:"cotiza_bolsa"`
	RequiereRevisorFiscal    bool   `json:"requiere_revisor_fiscal"`
}
