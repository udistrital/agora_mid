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

