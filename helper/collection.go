package helper

import (
	"context"

	"github.com/udistrital/agora_mid/config"
	"github.com/udistrital/agora_mid/responses"
)

// CollectionResult contiene la página expandida de una colección, el total del
// conjunto remoto y los avisos acumulados de sus elementos.
type CollectionResult struct {
	Data     []any
	Total    int
	Warnings []responses.RelationError
}

// ListCollection expande cada registro de una colección con la misma
// configuración y opciones de Expand, conservando elementos cuya relación
// falle bajo la política tolerante y propagando sus avisos.
func ListCollection(
	ctx context.Context,
	records []map[string]any,
	total int,
	cfg config.ServiceConfig,
	registry ResolverRegistry,
	options ExpandOptions,
) (CollectionResult, error) {
	resultado := CollectionResult{
		Data:  make([]any, 0, len(records)),
		Total: total,
	}
	for _, record := range records {
		expandido, err := Expand(ctx, record, cfg, registry, options)
		if err != nil {
			return CollectionResult{}, err
		}
		resultado.Data = append(resultado.Data, expandido.Data)
		resultado.Warnings = append(resultado.Warnings, expandido.Warnings...)
	}
	return resultado, nil
}
