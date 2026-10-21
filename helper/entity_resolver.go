package helper

import (
	"context"

	"github.com/udistrital/agora_mid/config"
)

// EntityResolver es un envoltorio fino y reutilizable sobre HTTPResolver que fija
// el endpoint real y el campo de consulta (related_field + query_template) de una
// entidad relacionada, con independencia de lo declarado en la configuración.
//
// Su propósito es encapsular el origen real de cada tabla (consulta por id
// propio, por la llave del padre o por una PK compartida) en el código, sin
// que la configuración declarativa tenga que nombrar implementaciones ni
// conocer el detalle de cómo se consulta cada CRUD. Reutiliza íntegramente el
// transporte, los timeouts y la decodificación (envoltorio OAS o lista directa)
// del HTTPResolver subyacente. No aplica ninguna lógica de negocio: anida el
// payload tal como lo entrega el CRUD.
type EntityResolver struct {
	// inner ejecuta el transporte HTTP y la decodificación.
	inner HTTPResolver
	// endpoint es la ruta real de la entidad (por ejemplo "/v1/tercero").
	endpoint string
	// relatedAPI identifica el servicio externo que resuelve la URL base.
	relatedAPI string
	// relatedField es el campo por el que se consulta y por el que el expansor
	// relaciona cada registro con su llave local (id propio, tercero_id,
	// proveedor_id o la PK compartida).
	relatedField string
	// queryTemplate codifica las llaves en el parámetro query. {ids} se
	// reemplaza por las llaves unidas por comas.
	queryTemplate string
}

// NewEntityResolver construye un resolver específico de entidad que fija el
// endpoint, el servicio externo, el campo de consulta y la plantilla de query
// sobre el HTTPResolver indicado.
func NewEntityResolver(
	inner HTTPResolver,
	relatedAPI string,
	endpoint string,
	relatedField string,
	queryTemplate string,
) EntityResolver {
	return EntityResolver{
		inner:         inner,
		endpoint:      endpoint,
		relatedAPI:    relatedAPI,
		relatedField:  relatedField,
		queryTemplate: queryTemplate,
	}
}

// Resolve consulta la entidad con el endpoint y el campo de consulta fijados por
// el resolver, ignorando los valores declarativos equivalentes de rel para que
// la configuración no determine el origen real. Conserva mapped_name,
// cardinalidad y la dirección de la relación tal como vienen en rel, ya que el
// expansor los necesita para anidar y relacionar.
//
// No transforma el payload: delega el transporte y la decodificación en el
// HTTPResolver subyacente y devuelve los registros tal como los entrega el CRUD.
func (e EntityResolver) Resolve(
	ctx context.Context,
	rel config.RelationConfig,
	keys []string,
) ([]map[string]any, error) {
	// Fijar el origen real de la entidad sin perder los campos que el expansor
	// usa para anidar y relacionar (MappedName, Cardinality, Direction,
	// LocalField).
	efectiva := rel
	efectiva.RelatedAPI = e.relatedAPI
	efectiva.Endpoint = e.endpoint
	efectiva.RelatedField = e.relatedField
	efectiva.QueryTemplate = e.queryTemplate
	return e.inner.Resolve(ctx, efectiva, keys)
}

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
// Campos de consulta por entidad (según el modelo):
//   - tercero: por su propio Id (1-1 con proveedor vía proveedor.tercero_id).
//   - datos_identificacion: por TerceroId (1-N bajo el tercero).
//   - tipo_contribuyente: por su propio Id.
//   - proveedor_natural / proveedor_juridico: por Id (PK compartida con proveedor).
//   - perfil: por su propio Id (perfil_declarado del proveedor natural).
//   - contacto: por ProveedorId (1-N).
//   - cuenta_bancaria: por ProveedorId (1-N).
func NewResolversNucleo(inner HTTPResolver) map[string]Resolver {
	return map[string]Resolver{
		ServicioTercero: NewEntityResolver(
			inner, "terceros", endpointTercero, "Id", "Id:{ids}",
		),
		ServicioDatosIdentificacion: NewEntityResolver(
			inner, "terceros", endpointDatosIdentificacion, "TerceroId", "TerceroId:{ids}",
		),
		ServicioTipoContribuyente: NewEntityResolver(
			inner, "terceros", endpointTipoContribuyente, "Id", "Id:{ids}",
		),
		ServicioProveedorNatural: NewEntityResolver(
			inner, "gestion_personas_crud", endpointProveedorNatural, "Id", "Id:{ids}",
		),
		ServicioProveedorJuridico: NewEntityResolver(
			inner, "gestion_personas_crud", endpointProveedorJuridico, "Id", "Id:{ids}",
		),
		ServicioPerfil: NewEntityResolver(
			inner, "gestion_personas_crud", endpointPerfil, "Id", "Id:{ids}",
		),
		ServicioContacto: NewEntityResolver(
			inner, "gestion_personas_crud", endpointContacto, "ProveedorId", "ProveedorId:{ids}",
		),
		ServicioCuentaBancaria: NewEntityResolver(
			inner, "gestion_personas_crud", endpointCuentaBancaria, "ProveedorId", "ProveedorId:{ids}",
		),
	}
}
