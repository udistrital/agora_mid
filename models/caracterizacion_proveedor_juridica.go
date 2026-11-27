package models

// RegistroCaracterizacionJuridica es el cuerpo de POST y PUT del dominio 2 para
// persona jurídica. Cada sección es opcional: un guardado parcial solo escribe
// lo que llega. Las secciones ProveedorJuridico y Representante, cuando llegan,
// se validan completas porque alimentan columnas NOT NULL y una relación única.
type RegistroCaracterizacionJuridica struct {
	TerceroId                   int64                              `json:"TerceroId"`
	Representante               *RepresentanteInput                `json:"Representante,omitempty"`
	InfoComplementariaTercero   []InfoComplementariaTerceroInput   `json:"InfoComplementariaTercero"`
	ProveedorJuridico           *ProveedorJuridicoInput            `json:"ProveedorJuridico,omitempty"`
	PerfilProveedor             []PerfilProveedorInput             `json:"PerfilProveedor"`
	Contacto                    []ContactoProveedorInput           `json:"Contacto"`
	ActividadEconomicaProveedor []ActividadEconomicaProveedorInput `json:"ActividadEconomicaProveedor"`
	DescripcionPortafolio       *string                            `json:"DescripcionPortafolio,omitempty"`
	Consentimiento              *bool                              `json:"Consentimiento,omitempty"`
}

// RepresentanteInput identifica al representante legal. Se resuelve por
// RepresentanteId o por tipo de documento + número. El representante debe ser
// un tercero natural activo con proveedor_natural activo; el dominio 2 no lo
// crea.
type RepresentanteInput struct {
	RepresentanteId     *int64  `json:"RepresentanteId,omitempty"`
	TipoDocumentoId     *int64  `json:"TipoDocumentoId,omitempty"`
	CodigoTipoDocumento *string `json:"CodigoTipoDocumento,omitempty"`
	Numero              *string `json:"Numero,omitempty"`
	CargoId             *int64  `json:"CargoId,omitempty"`
	CodigoCargo         *string `json:"CodigoCargo,omitempty"`
	PrimerNombre        *string `json:"PrimerNombre,omitempty"`
	SegundoNombre       *string `json:"SegundoNombre,omitempty"`
	PrimerApellido      *string `json:"PrimerApellido,omitempty"`
	SegundoApellido     *string `json:"SegundoApellido,omitempty"`
}

// ProveedorJuridicoInput contiene las columnas mutables de proveedor_juridico.
// Los campos con Id y código son alternativos: el código es estable entre
// ambientes. TipoConformacionId corresponde a los tipos de entidad legal de
// TIPO_CONFORMACION; la modalidad de la cuenta viaja en
// InfoComplementariaTercero con el código TIPO_CONF_CUENTA.
type ProveedorJuridicoInput struct {
	ProcedenciaEmpresaId      *int64  `json:"ProcedenciaEmpresaId,omitempty"`
	CodigoProcedencia         *string `json:"CodigoProcedencia,omitempty"`
	CiudadOrigenId            *int64  `json:"CiudadOrigenId,omitempty"`
	PaisOrigenId              *int64  `json:"PaisOrigenId,omitempty"`
	CodigoPostal              *string `json:"CodigoPostal,omitempty"`
	TipoConformacionId        *int64  `json:"TipoConformacionId,omitempty"`
	CodigoTipoConformacion    *string `json:"CodigoTipoConformacion,omitempty"`
	RegimenContributivo       *int64  `json:"RegimenContributivo,omitempty"`
	CodigoRegimenContributivo *string `json:"CodigoRegimenContributivo,omitempty"`
	ExclusividadProducto      *bool   `json:"ExclusividadProducto,omitempty"`
	Pyme                      *bool   `json:"Pyme,omitempty"`
	RegistroMercantil         *bool   `json:"RegistroMercantil,omitempty"`
	SujetoRetencion           *bool   `json:"SujetoRetencion,omitempty"`
	AgenteRetenedor           *bool   `json:"AgenteRetenedor,omitempty"`
	ResponsableIca            *bool   `json:"ResponsableIca,omitempty"`
	ResponsableIva            *bool   `json:"ResponsableIva,omitempty"`
}

// CaracterizacionJuridicaData es el contenido de Data en la respuesta.
type CaracterizacionJuridicaData struct {
	TerceroId                   int64                                 `json:"TerceroId"`
	ProveedorId                 int64                                 `json:"ProveedorId"`
	Proveedor                   map[string]any                        `json:"Proveedor,omitempty"`
	ProveedorJuridico           map[string]any                        `json:"ProveedorJuridico,omitempty"`
	Representante               map[string]any                        `json:"Representante,omitempty"`
	Representacion              *EntidadId                            `json:"Representacion,omitempty"`
	InfoComplementariaTercero   []InfoComplementariaTerceroRegistrada `json:"InfoComplementariaTercero"`
	PerfilProveedor             []EntidadId                           `json:"PerfilProveedor"`
	Contacto                    []EntidadId                           `json:"Contacto"`
	ActividadEconomicaProveedor []EntidadId                           `json:"ActividadEconomicaProveedor"`
	Resultado                   Resultado                             `json:"Resultado"`
}
