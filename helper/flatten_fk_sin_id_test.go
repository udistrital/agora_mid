package helper

import (
	"context"
	"testing"

	"github.com/udistrital/agora_mid/config"
)

// expandirContactoConFK ejecuta una expansión de Contacto cuyo registro del CRUD trae
// ProveedorId con el valor dado.
func expandirContactoConFK(t *testing.T, proveedorId any) (ExpandResult, error) {
	t.Helper()
	res := &resolverFake{registros: []map[string]any{{
		"Id":              float64(10),
		"ValorEspecifico": "ana@example.com",
		"ProveedorId":     proveedorId,
	}}}
	registry := NewResolverRegistry(map[string]Resolver{"contacto": res}, nil)
	rel := config.RelationConfig{
		Name:           "Contacto",
		RelatedService: "contacto",
		RelatedAPI:     "gestion_personas_crud",
		RelatedTable:   "contacto",
		Direction:      config.DirectionRelatedToLocal,
		LocalField:     "Id",
		RelatedField:   "ProveedorId",
		Cardinality:    config.CardinalityMany,
		Depth:          1,
		MappedName:     "Contacto",
		AllowedFields:  []string{"Id", "ValorEspecifico"},
	}
	cfg := config.ServiceConfig{
		Service: "proveedor", MaxDepth: 1, OnRelationError: config.OnRelationErrorNotify,
		MaxRelatedIDs: 100, Relations: []config.RelationConfig{rel},
	}
	return Expand(context.Background(), map[string]any{"Id": float64(1)}, cfg, registry, ExpandOptions{})
}

// tieneAvisoFKNoAplanable informa si la expansión emitió el aviso de FK no aplanable.
func tieneAvisoFKNoAplanable(res ExpandResult) bool {
	for _, a := range res.Warnings {
		if a.Kind == "fk_no_aplanable" {
			return true
		}
	}
	return false
}

// Un objeto de FK sin Id no se puede aplanar: la expansión no falla y avisa.
func TestExpandFKSinIdAvisaSinFallar(t *testing.T) {
	res, err := expandirContactoConFK(t, map[string]any{"Nombre": "sin id"})
	if err != nil {
		t.Fatalf("una FK no aplanable no debe fallar la expansión: %v", err)
	}
	if !tieneAvisoFKNoAplanable(res) {
		t.Errorf("se esperaba aviso fk_no_aplanable; warnings=%+v", res.Warnings)
	}
}

// Un objeto de FK con Id nulo tampoco se puede aplanar y debe avisar igual.
func TestExpandFKConIdNuloAvisaSinFallar(t *testing.T) {
	res, err := expandirContactoConFK(t, map[string]any{"Id": nil})
	if err != nil {
		t.Fatalf("una FK con Id nulo no debe fallar la expansión: %v", err)
	}
	if !tieneAvisoFKNoAplanable(res) {
		t.Errorf("se esperaba aviso fk_no_aplanable; warnings=%+v", res.Warnings)
	}
}

// Una FK con Id se aplana sin aviso: el caso normal no cambia.
func TestExpandFKConIdSeAplanaSinAviso(t *testing.T) {
	res, err := expandirContactoConFK(t, map[string]any{"Id": float64(1)})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if tieneAvisoFKNoAplanable(res) {
		t.Errorf("una FK con Id no debe avisar; warnings=%+v", res.Warnings)
	}
}
