package helper

import (
	"context"

	"github.com/udistrital/agora_mid/config"
	"github.com/udistrital/agora_mid/responses"
)

// GenericMarker identifica a los resolvers que entregan Data sin transformar y
// cuyo uso debe notificarse mediante un aviso generic_resolver.
type GenericMarker interface {
	// IsGeneric devuelve true para los resolvers genéricos.
	IsGeneric() bool
}

// GenericResolver resuelve relaciones que no cuentan con un adaptador
// especializado. Delega el transporte en otro Resolver y entrega el Data
// decodificado sin aplicar mapeos ni expansión interna.
type GenericResolver struct {
	inner Resolver
}

// NewGenericResolver construye un GenericResolver que delega el transporte en
// inner.
func NewGenericResolver(inner Resolver) GenericResolver {
	return GenericResolver{inner: inner}
}

// Resolve devuelve los registros decodificados por el resolver de transporte sin
// transformarlos.
func (g GenericResolver) Resolve(
	ctx context.Context,
	rel config.RelationConfig,
	keys []string,
) ([]map[string]any, error) {
	return g.inner.Resolve(ctx, rel, keys)
}

// IsGeneric devuelve true.
func (g GenericResolver) IsGeneric() bool { return true }

// EsGenerico informa si r es un resolver genérico.
func EsGenerico(r Resolver) bool {
	marker, ok := r.(GenericMarker)
	return ok && marker.IsGeneric()
}

// AvisoGenericResolver construye el aviso generic_resolver para la relación en la
// ruta indicada. El mensaje identifica solo la ruta lógica y el servicio, sin
// URLs internas, credenciales ni cuerpos remotos.
func AvisoGenericResolver(path string, rel config.RelationConfig) responses.RelationError {
	return responses.RelationError{
		Path:    path,
		Service: rel.RelatedService,
		Message: "relación resuelta por el resolver genérico; el Data se entrega sin transformar",
		Kind:    KindGenericResolver,
	}
}
