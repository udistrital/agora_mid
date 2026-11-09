package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

type apiListEnvelope[T any] struct {
	Data  []T  `json:"Data"`
	Total *int `json:"Total"`
}

// apiListResult contiene los registros de una respuesta de colección y el total
// del conjunto que el CRUD calculó antes de aplicar la página devuelta.
type apiListResult[T any] struct {
	Data  []T
	Total int
}

// decodeAPIListWithTotal decodifica una lista directa o el envoltorio OAS del
// CRUD, conservando Data y Total. Si el CRUD no entrega Total, usa el tamaño de
// Data como valor compatible con respuestas antiguas.
func decodeAPIListWithTotal[T any](raw json.RawMessage) (apiListResult[T], error) {
	var direct []T
	if err := json.Unmarshal(raw, &direct); err == nil {
		return apiListResult[T]{Data: direct, Total: len(direct)}, nil
	}

	var envelope apiListEnvelope[T]
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return apiListResult[T]{}, fmt.Errorf("respuesta de lista inválida: %w", err)
	}
	if envelope.Data == nil {
		return apiListResult[T]{}, errors.New("la respuesta del CRUD no contiene una lista Data")
	}
	total := len(envelope.Data)
	if envelope.Total != nil {
		total = *envelope.Total
	}
	return apiListResult[T]{Data: envelope.Data, Total: total}, nil
}

// decodeAPIList acepta la respuesta enveloped estándar de los CRUD OAS
// (Success/Status/Message/Data) y una lista JSON directa para compatibilidad
// con servicios o mocks que aún exponen ese formato.
func decodeAPIList[T any](raw json.RawMessage) ([]T, error) {
	resultado, err := decodeAPIListWithTotal[T](raw)
	if err != nil {
		return nil, err
	}
	return resultado.Data, nil
}

func getAPIList[T any](ctx context.Context, client interface {
	Get(context.Context, string, any) (int, error)
}, endpoint string) ([]T, error) {
	var raw json.RawMessage
	if _, err := client.Get(ctx, endpoint, &raw); err != nil {
		return nil, err
	}
	return decodeAPIList[T](bytes.TrimSpace(raw))
}
