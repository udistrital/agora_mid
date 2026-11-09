package services

import (
	"encoding/json"
	"testing"
)

// Verifica que la decodificación de una lista conserva el total del conjunto
// entregado por el CRUD además de sus registros.
func TestDecodeAPIListCapturaTotal(t *testing.T) {
	raw := json.RawMessage(`{"Data":[{"Id":1}],"Total":7}`)

	resultado, err := decodeAPIListWithTotal[map[string]any](raw)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if resultado.Total != 7 {
		t.Fatalf("total inesperado: got=%d want=7", resultado.Total)
	}
	if len(resultado.Data) != 1 || resultado.Data[0]["Id"] != float64(1) {
		t.Fatalf("data inesperada: %+v", resultado.Data)
	}
}
