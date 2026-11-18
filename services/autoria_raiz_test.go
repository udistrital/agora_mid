package services

import (
	"testing"

	"github.com/udistrital/agora_mid/config"
)

// TestPropagarAutorDeRaizNoModificaElCuerpoOriginal verifica que la propagación
// devuelve una copia: el cuerpo recibido conserva los objetos anidados sin autor, y
// la copia lo recibe.
func TestPropagarAutorDeRaizNoModificaElCuerpoOriginal(t *testing.T) {
	relaciones := []config.RelationConfig{{
		MappedName: "Contacto",
		Write:      &config.WriteConfig{WritableFields: []string{"UsuarioModificacionId"}},
	}}
	cuerpo := map[string]any{
		"UsuarioModificacionId": float64(9),
		"Contacto":              []any{map[string]any{"ValorEspecifico": "a@b.co"}},
	}

	salida := propagarAutorDeRaiz(cuerpo, relaciones)

	original := cuerpo["Contacto"].([]any)[0].(map[string]any)
	if _, trae := original["UsuarioModificacionId"]; trae {
		t.Errorf("el cuerpo original no debe modificarse; contacto=%+v", original)
	}
	copiado := salida["Contacto"].([]any)[0].(map[string]any)
	if copiado["UsuarioModificacionId"] != float64(9) {
		t.Errorf("la copia debe llevar UsuarioModificacionId=9 en el contacto; contacto=%+v", copiado)
	}
}
