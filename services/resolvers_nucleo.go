package services

import "github.com/udistrital/agora_mid/helper"

// Nombres lógicos de servicio (RelatedService) de las entidades del núcleo. El
// ProveedorService usa estos nombres como llaves del registro relación→resolver
// que inyecta al expansor; deben coincidir con el related_service declarado en
// config/relaciones_proveedor.yaml.
const (
	ServicioTercero             = "tercero"
	ServicioDatosIdentificacion = "datos_identificacion"
	ServicioTipoContribuyente   = "tipo_contribuyente"
	ServicioProveedorNatural    = "proveedor_natural"
	ServicioProveedorJuridico   = "proveedor_juridico"
	ServicioPerfil              = "perfil"
	ServicioContacto            = "contacto"
	ServicioCuentaBancaria      = "cuenta_bancaria"
)

// Endpoints reales de cada entidad del núcleo en su CRUD dueño.
const (
	endpointTercero             = "/v1/tercero"
	endpointDatosIdentificacion = "/v1/datos_identificacion"
	endpointTipoContribuyente   = "/v1/tipo_contribuyente"
	endpointProveedorNatural    = "/v1/proveedor_natural"
	endpointProveedorJuridico   = "/v1/proveedor_juridico"
	endpointPerfil              = "/v1/perfil"
	endpointContacto            = "/v1/contacto"
	endpointCuentaBancaria      = "/v1/cuenta_bancaria"
)

// NewResolversNucleo construye los resolvers específicos de las entidades del
// núcleo del proveedor, cada uno con su endpoint y campo de consulta reales,
// reutilizando el mismo transporte HTTP. Devuelve el mapeo
// related_service→Resolver listo para inyectarse al expansor.
//
// El cableado concreto del núcleo (qué entidades existen y su endpoint/campo de
// consulta real) vive en services/ porque es específico del servicio proveedor;
// helper/ solo aporta la primitiva reutilizable EntityResolver.
//
// Campos de consulta por entidad (según el modelo):
//   - tercero: por su propio Id (1-1 con proveedor vía proveedor.tercero_id).
//   - datos_identificacion: por TerceroId (1-N bajo el tercero).
//   - tipo_contribuyente: por su propio Id.
//   - proveedor_natural / proveedor_juridico: por Id (PK compartida con proveedor).
//   - perfil: por su propio Id (perfil_declarado del proveedor natural).
//   - contacto: por ProveedorId (1-N).
//   - cuenta_bancaria: por ProveedorId (1-N).
func NewResolversNucleo(inner helper.HTTPResolver) map[string]helper.Resolver {
	return map[string]helper.Resolver{
		ServicioTercero: helper.NewEntityResolver(
			inner, "terceros", endpointTercero, "Id", "Id:{ids}",
		),
		ServicioDatosIdentificacion: helper.NewEntityResolver(
			inner, "terceros", endpointDatosIdentificacion, "TerceroId", "TerceroId:{ids}",
		),
		ServicioTipoContribuyente: helper.NewEntityResolver(
			inner, "terceros", endpointTipoContribuyente, "Id", "Id:{ids}",
		),
		ServicioProveedorNatural: helper.NewEntityResolver(
			inner, "gestion_personas_crud", endpointProveedorNatural, "Id", "Id:{ids}",
		),
		ServicioProveedorJuridico: helper.NewEntityResolver(
			inner, "gestion_personas_crud", endpointProveedorJuridico, "Id", "Id:{ids}",
		),
		ServicioPerfil: helper.NewEntityResolver(
			inner, "gestion_personas_crud", endpointPerfil, "Id", "Id:{ids}",
		),
		ServicioContacto: helper.NewEntityResolver(
			inner, "gestion_personas_crud", endpointContacto, "ProveedorId", "ProveedorId:{ids}",
		),
		ServicioCuentaBancaria: helper.NewEntityResolver(
			inner, "gestion_personas_crud", endpointCuentaBancaria, "ProveedorId", "ProveedorId:{ids}",
		),
	}
}
