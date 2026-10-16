package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

type apiListEnvelope[T any] struct {
	Data []T `json:"Data"`
}

// decodeAPIList acepta la respuesta enveloped estándar de los CRUD OAS
// (Success/Status/Message/Data) y una lista JSON directa para compatibilidad
// con servicios o mocks que aún exponen ese formato.
func decodeAPIList[T any](raw json.RawMessage) ([]T, error) {
	var direct []T
	if err := json.Unmarshal(raw, &direct); err == nil {
		return direct, nil
	}
	var envelope apiListEnvelope[T]
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("respuesta de lista inválida: %w", err)
	}
	if envelope.Data == nil {
		return nil, errors.New("la respuesta del CRUD no contiene una lista Data")
	}
	return envelope.Data, nil
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
