package utils

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestResponseMetadataOmitempty verifica que Metadata se omite del JSON cuando
// es nil y se incluye cuando está presente.
func TestResponseMetadataOmitempty(t *testing.T) {
	sinMetadata := Response{Success: true, Status: 200, Message: "ok", Data: nil}
	b, err := json.Marshal(sinMetadata)
	if err != nil {
		t.Fatalf("marshal sin metadata: %v", err)
	}
	if strings.Contains(string(b), "Metadata") {
		t.Errorf("Metadata no debería aparecer cuando es nil, JSON: %s", b)
	}

	conMetadata := Response{
		Success:  true,
		Status:   200,
		Message:  "ok",
		Data:     []string{"a"},
		Metadata: &Metadata{Total: 1, Limit: 10, Offset: 0, CurrentPage: 1, TotalPages: 1},
	}
	b, err = json.Marshal(conMetadata)
	if err != nil {
		t.Fatalf("marshal con metadata: %v", err)
	}
	if !strings.Contains(string(b), "Metadata") {
		t.Errorf("Metadata debería aparecer cuando está presente, JSON: %s", b)
	}
	if !strings.Contains(string(b), `"total":1`) {
		t.Errorf("los campos de paginación deberían serializarse, JSON: %s", b)
	}
}
