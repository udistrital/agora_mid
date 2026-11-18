package services

import "github.com/udistrital/agora_mid/helper"

// Nombres lógicos de servicio (RelatedService) de las entidades del árbol del proveedor.
// El ProveedorService usa estos nombres como llaves del registro relación→resolver que
// inyecta al expansor; deben coincidir con el related_service que el cargador deriva de
// cada objeto de config/.
const (
	ServicioTercero                   = "tercero"
	ServicioDatosIdentificacion       = "datos_identificacion"
	ServicioTipoContribuyente         = "tipo_contribuyente"
	ServicioProveedorNatural          = "proveedor_natural"
	ServicioProveedorJuridico         = "proveedor_juridico"
	ServicioPerfil                    = "perfil"
	ServicioContacto                  = "contacto"
	ServicioCuentaBancaria            = "cuenta_bancaria"
	ServicioEstadoProveedor           = "estado_proveedor"
	ServicioDocumento                 = "documento"
	ServicioPerfilProveedor           = "perfil_proveedor"
	ServicioTituloProveedorNatural    = "titulo_proveedor_natural"
	ServicioActividadEconomica        = "actividad_economica_proveedor"
	ServicioRepresentacion            = "representacion"
	ServicioPersonaJuridicaTipoEnt    = "persona_juridica_tipo_entidad"
	ServicioTerceroTipoTercero        = "tercero_tipo_tercero"
	ServicioInfoComplementariaTercero = "info_complementaria_tercero"
	ServicioSeguridadSocialTercero    = "seguridad_social_tercero"
	ServicioParametro                 = "parametro"
	ServicioLugar                     = "lugar"
	ServicioTipoLugar                 = "tipo_lugar"
	ServicioTipoTercero               = "tipo_tercero"
	ServicioInfoComplementaria        = "info_complementaria"
	ServicioTipoDocumento             = "tipo_documento"
)

// Endpoints reales de cada entidad en su CRUD dueño.
const (
	endpointTercero                   = "/v1/tercero"
	endpointDatosIdentificacion       = "/v1/datos_identificacion"
	endpointTipoContribuyente         = "/v1/tipo_contribuyente"
	endpointProveedorNatural          = "/v1/proveedor_natural"
	endpointProveedorJuridico         = "/v1/proveedor_juridico"
	endpointPerfil                    = "/v1/perfil"
	endpointContacto                  = "/v1/contacto"
	endpointCuentaBancaria            = "/v1/cuenta_bancaria"
	endpointEstadoProveedor           = "/v1/estado_proveedor"
	endpointDocumento                 = "/v1/documento"
	endpointPerfilProveedor           = "/v1/perfil_proveedor"
	endpointTituloProveedorNatural    = "/v1/titulo_proveedor_natural"
	endpointActividadEconomica        = "/v1/actividad_economica_proveedor"
	endpointRepresentacion            = "/v1/representacion"
	endpointPersonaJuridicaTipoEnt    = "/v1/persona_juridica_tipo_entidad"
	endpointTerceroTipoTercero        = "/v1/tercero_tipo_tercero"
	endpointInfoComplementariaTercero = "/v1/info_complementaria_tercero"
	endpointSeguridadSocialTercero    = "/v1/seguridad_social_tercero"
	endpointParametro                 = "/v1/parametro"
	endpointLugar                     = "/v1/lugar"
	endpointTipoLugar                 = "/v1/tipo_lugar"
	endpointTipoTercero               = "/v1/tipo_tercero"
	endpointInfoComplementaria        = "/v1/info_complementaria"
	endpointTipoDocumento             = "/v1/tipo_documento"
)

// Campos de consulta comunes a los resolvedores de lectura del árbol.
const (
	campoID                = "Id"
	campoProveedorID       = "ProveedorId"
	campoTerceroID         = "TerceroId"
	campoPersonaJuridicaID = "PersonaJuridicaId"
)

// APIs de los CRUD dueños que resuelven cada entidad.
const (
	apiGestion    = "gestion_personas_crud"
	apiTerceros   = "terceros"
	apiParametros = "parametros"
	apiUbicacion  = "ubicaciones"
)

// NewResolversNucleo construye los resolvers de lectura de las entidades del árbol del
// proveedor, cada uno con su endpoint y campo de consulta reales, reutilizando el mismo
// transporte HTTP. Devuelve el mapeo related_service→Resolver listo para inyectarse al
// expansor.
//
// Campo de consulta por entidad:
//   - Catálogos y lugares (lookup, cardinalidad one): por su propio Id.
//   - Colecciones del proveedor: por ProveedorId (1-N).
//   - Colecciones bajo el tercero: por TerceroId (1-N).
//   - Colecciones bajo persona jurídica: por PersonaJuridicaId (1-N).
//   - Especializaciones (proveedor_natural / proveedor_juridico): por Id (PK compartida).
func NewResolversNucleo(inner helper.HTTPResolver) map[string]helper.Resolver {
	porID := func(api, endpoint string) helper.Resolver {
		return helper.NewEntityResolver(inner, api, endpoint, campoID, "Id:{ids}")
	}
	porProveedor := func(api, endpoint string) helper.Resolver {
		return helper.NewEntityResolver(inner, api, endpoint, campoProveedorID, "ProveedorId:{ids}")
	}
	porTercero := func(api, endpoint string) helper.Resolver {
		return helper.NewEntityResolver(inner, api, endpoint, campoTerceroID, "TerceroId:{ids}")
	}
	porPersonaJuridica := func(api, endpoint string) helper.Resolver {
		return helper.NewEntityResolver(inner, api, endpoint, campoPersonaJuridicaID, "PersonaJuridicaId:{ids}")
	}

	return map[string]helper.Resolver{
		ServicioTercero:                   porID(apiTerceros, endpointTercero),
		ServicioDatosIdentificacion:       porTercero(apiTerceros, endpointDatosIdentificacion),
		ServicioTipoContribuyente:         porID(apiTerceros, endpointTipoContribuyente),
		ServicioProveedorNatural:          porID(apiGestion, endpointProveedorNatural),
		ServicioProveedorJuridico:         porID(apiGestion, endpointProveedorJuridico),
		ServicioPerfil:                    porID(apiGestion, endpointPerfil),
		ServicioContacto:                  porProveedor(apiGestion, endpointContacto),
		ServicioCuentaBancaria:            porProveedor(apiGestion, endpointCuentaBancaria),
		ServicioEstadoProveedor:           porProveedor(apiGestion, endpointEstadoProveedor),
		ServicioDocumento:                 porProveedor(apiGestion, endpointDocumento),
		ServicioPerfilProveedor:           porProveedor(apiGestion, endpointPerfilProveedor),
		ServicioTituloProveedorNatural:    porProveedor(apiGestion, endpointTituloProveedorNatural),
		ServicioActividadEconomica:        porProveedor(apiGestion, endpointActividadEconomica),
		ServicioRepresentacion:            porProveedor(apiGestion, endpointRepresentacion),
		ServicioPersonaJuridicaTipoEnt:    porPersonaJuridica(apiGestion, endpointPersonaJuridicaTipoEnt),
		ServicioTerceroTipoTercero:        porTercero(apiTerceros, endpointTerceroTipoTercero),
		ServicioInfoComplementariaTercero: porTercero(apiTerceros, endpointInfoComplementariaTercero),
		ServicioSeguridadSocialTercero:    porTercero(apiTerceros, endpointSeguridadSocialTercero),
		ServicioParametro:                 porID(apiParametros, endpointParametro),
		ServicioLugar:                     porID(apiUbicacion, endpointLugar),
		ServicioTipoLugar:                 porID(apiUbicacion, endpointTipoLugar),
		ServicioTipoTercero:               porID(apiTerceros, endpointTipoTercero),
		ServicioInfoComplementaria:        porID(apiTerceros, endpointInfoComplementaria),
		ServicioTipoDocumento:             porID(apiTerceros, endpointTipoDocumento),
	}
}
