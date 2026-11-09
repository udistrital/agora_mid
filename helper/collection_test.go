package helper

import (
	"context"
	"errors"
	"testing"

	"github.com/udistrital/agora_mid/config"
)

// cfgFija devuelve un cfgDe que aplica la misma configuración a todo registro.
func cfgFija(cfg config.ServiceConfig) func(map[string]any) config.ServiceConfig {
	return func(map[string]any) config.ServiceConfig { return cfg }
}

// Expande todos los elementos de una colección, conserva el total remoto y
// acumula los avisos producidos por cada expansión.
func TestListCollectionExpandeElementosYPropagaTotal(t *testing.T) {
	res := &resolverFake{
		registros: []map[string]any{{"Id": float64(7), "NombreCompleto": "Ana"}},
		err:       errors.New("crud no disponible"),
	}
	registry := NewResolverRegistry(map[string]Resolver{"tercero": res}, nil)
	registros := []map[string]any{
		{"Id": float64(1), "TerceroId": float64(7)},
		{"Id": float64(2), "TerceroId": float64(7)},
	}

	resultado, err := ListCollection(
		context.Background(),
		registros,
		9,
		cfgFija(cfgPrueba()),
		registry,
		ExpandOptions{},
	)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if resultado.Total != 9 || len(resultado.Data) != 2 {
		t.Fatalf("resultado inesperado: total=%d data=%+v", resultado.Total, resultado.Data)
	}
	uno, err := Expand(context.Background(), registros[0], cfgPrueba(), registry, ExpandOptions{})
	if err != nil {
		t.Fatalf("error inesperado al calcular avisos de referencia: %v", err)
	}
	if len(resultado.Warnings) != len(uno.Warnings)*len(registros) {
		t.Fatalf("se deben acumular los avisos de todos los elementos: %+v", resultado.Warnings)
	}
	for _, elemento := range resultado.Data {
		if elemento.(map[string]any)["Tercero"] != nil {
			t.Errorf("la rama fallida debe conservarse como null: %+v", elemento)
		}
	}
}

// Una colección sin registros produce una lista vacía y conserva su total.
func TestListCollectionVacia(t *testing.T) {
	resultado, err := ListCollection(
		context.Background(),
		[]map[string]any{},
		0,
		cfgFija(cfgPrueba()),
		NewResolverRegistry(nil, nil),
		ExpandOptions{},
	)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if resultado.Total != 0 || resultado.Data == nil || len(resultado.Data) != 0 {
		t.Fatalf("lista vacía inesperada: %+v", resultado)
	}
}

// Cada elemento de la colección se expande con la configuración que le
// corresponde según su propio registro, no con una única configuración para
// toda la página.
func TestListCollectionUsaCfgPorElemento(t *testing.T) {
	res := &resolverFake{registros: []map[string]any{{"Id": float64(7), "NombreCompleto": "Ana"}}}
	registry := NewResolverRegistry(map[string]Resolver{"tercero": res}, nil)

	conRelacion := cfgPrueba()
	sinRelacion := cfgPrueba()
	sinRelacion.Relations = nil

	registros := []map[string]any{
		{"Id": float64(1), "TerceroId": float64(7), "Tipo": "con"},
		{"Id": float64(2), "TerceroId": float64(7), "Tipo": "sin"},
	}
	cfgDe := func(record map[string]any) config.ServiceConfig {
		if record["Tipo"] == "con" {
			return conRelacion
		}
		return sinRelacion
	}

	resultado, err := ListCollection(
		context.Background(),
		registros,
		2,
		cfgDe,
		registry,
		ExpandOptions{},
	)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if len(resultado.Data) != 2 {
		t.Fatalf("se esperaban 2 elementos; recibido: %+v", resultado.Data)
	}
	primero := resultado.Data[0].(map[string]any)
	if _, ok := primero["Tercero"].(map[string]any); !ok {
		t.Errorf("el primer elemento debe expandir Tercero con su cfg; recibido: %+v", primero)
	}
	segundo := resultado.Data[1].(map[string]any)
	if _, presente := segundo["Tercero"]; presente {
		t.Errorf("el segundo elemento no debe expandir Tercero con su cfg; recibido: %+v", segundo)
	}
}
