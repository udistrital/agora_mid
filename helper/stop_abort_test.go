package helper

import (
	"context"
	"testing"

	"github.com/udistrital/agora_mid/config"
)

// Una cascada completa con canonización de un campo derivado no se detiene: el
// aviso field_normalization se conserva y todos los hijos se materializan, incluso
// con política fail.
func TestWriteCascadaCompletaConCanonizacionNoAborta(t *testing.T) {
	var ll []registroMaterializacion
	registry := registryFalso(&ll, nil)
	body := map[string]any{
		"TerceroId": float64(25),
		"Hijo": []any{
			map[string]any{"Valor": "a", "PadreId": float64(999)},
			map[string]any{"Valor": "b"},
		},
	}

	res, err := Write(context.Background(), body, cfgEscritura(), registry, WriteOptions{OnRelationError: config.OnRelationErrorFail})
	if err != nil {
		t.Fatalf("una cascada completa no debe fallar: %v", err)
	}
	if len(res.Warnings) != 1 || res.Warnings[0].Kind != KindFieldNormalization {
		t.Fatalf("se esperaba solo el aviso field_normalization; warnings=%+v", res.Warnings)
	}
	if hijos, _ := res.Data["Hijo"].([]any); len(hijos) != 2 {
		t.Errorf("ambos hijos deben materializarse; Hijo=%+v", res.Data["Hijo"])
	}
}
