package helper

import (
	"context"
	"testing"

	"github.com/udistrital/agora_mid/config"
)

// El CRUD gestion_personas devuelve la FK descendente de una colección (por
// ejemplo Contacto.ProveedorId) como objeto anidado con el registro padre y no
// como entero. El motor debe aplanarla a su Id antes de comparar con la llave
// local, de lo contrario la relación 1-N nunca coincide y la lista sale vacía.
func TestExpandAplanaFKAnidadaDeRegistroCRUD(t *testing.T) {
	res := &resolverFake{registros: []map[string]any{{
		"Id":              float64(10),
		"ValorEspecifico": "ana@example.com",
		"ProveedorId": map[string]any{
			"Id":        float64(1),
			"TerceroId": float64(5),
			"Activo":    true,
		},
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

	out, err := Expand(context.Background(), map[string]any{"Id": float64(1)}, cfg, registry, ExpandOptions{})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	m := out.Data.(map[string]any)
	contactos, ok := m["Contacto"].([]any)
	if !ok {
		t.Fatalf("Contacto debe ser una lista; recibido: %T", m["Contacto"])
	}
	if len(contactos) != 1 {
		t.Fatalf("se esperaba 1 contacto del proveedor 1; recibido: %d", len(contactos))
	}
	contacto, ok := contactos[0].(map[string]any)
	if !ok {
		t.Fatalf("el contacto debe ser un objeto; recibido: %T", contactos[0])
	}
	if contacto["ProveedorId"] != float64(1) {
		t.Errorf("ProveedorId debe aplanarse a su Id entero 1; recibido: %#v", contacto["ProveedorId"])
	}
}
