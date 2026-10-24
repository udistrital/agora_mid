package helper

import (
	"context"
	"testing"

	"github.com/udistrital/agora_mid/config"
)

// relacionTerceroDispatch arma una relación tercero mínima para probar el
// despacho relación→resolver sin depender de la fixture compartida.
func relacionTerceroDispatch() config.RelationConfig {
	return config.RelationConfig{
		Name:           "Tercero",
		RelatedService: "tercero",
		RelatedAPI:     "terceros",
		RelatedTable:   "tercero",
		Direction:      config.DirectionLocalToRelated,
		LocalField:     "TerceroId",
		RelatedField:   "Id",
		Cardinality:    config.CardinalityOne,
		Depth:          1,
		MappedName:     "Tercero",
		Endpoint:       "/v1/tercero",
		QueryTemplate:  "Id:{ids}",
		AllowedFields:  []string{"Id", "NombreCompleto"},
	}
}

func cfgTerceroDispatch(rel config.RelationConfig) config.ServiceConfig {
	return config.ServiceConfig{
		Service:         "s",
		MaxDepth:        1,
		OnRelationError: config.OnRelationErrorNotify,
		MaxRelatedIDs:   100,
		Relations:       []config.RelationConfig{rel},
	}
}

// Cuando una relación tiene un resolver específico asociado por su nombre de
// servicio, el expansor delega a ese resolver, no consulta el genérico y no
// emite el aviso generic_resolver.
func TestDispatchResolverEspecificoNoTocaGenerico(t *testing.T) {
	especifico := &resolverFake{registros: []map[string]any{{"Id": float64(7), "NombreCompleto": "Ana"}}}
	generico := &resolverFake{registros: []map[string]any{{"Id": float64(99), "Campo": "no-debe-usarse"}}, generico: true}

	registry := NewResolverRegistry(map[string]Resolver{"tercero": especifico}, generico)

	rel := relacionTerceroDispatch()
	data := map[string]any{"Id": float64(1), "TerceroId": float64(7)}

	out, err := Expand(context.Background(), data, cfgTerceroDispatch(rel), registry, ExpandOptions{})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	// El resolver específico fue invocado una vez.
	if len(especifico.llamadas) != 1 {
		t.Errorf("se esperaba 1 consulta al resolver específico; recibidas: %d", len(especifico.llamadas))
	}
	// El genérico no fue tocado.
	if len(generico.llamadas) != 0 {
		t.Errorf("el resolver genérico no debe consultarse cuando hay uno específico; llamadas: %v", generico.llamadas)
	}
	// No hay aviso generic_resolver.
	if contieneAviso(out.Warnings, KindGenericResolver) {
		t.Errorf("no debe emitirse aviso generic_resolver con resolver específico; avisos: %+v", out.Warnings)
	}
	// El Data proviene del resolver específico.
	m := out.Data.(map[string]any)
	tercero, ok := m["Tercero"].(map[string]any)
	if !ok || tercero["NombreCompleto"] != "Ana" {
		t.Errorf("el Data debe provenir del resolver específico; recibido: %+v", m["Tercero"])
	}
}

// Cuando una relación no tiene resolver específico asociado, el expansor cae al
// genérico y emite exactamente un aviso generic_resolver.
func TestDispatchFallbackGenericoConAviso(t *testing.T) {
	generico := &resolverFake{registros: []map[string]any{{"Id": float64(7), "Campo": "x"}}, generico: true}

	// Registro sin resolver específico para "tercero": debe caer al genérico.
	registry := NewResolverRegistry(map[string]Resolver{}, generico)

	rel := relacionTerceroDispatch()
	data := map[string]any{"Id": float64(1), "TerceroId": float64(7)}

	out, err := Expand(context.Background(), data, cfgTerceroDispatch(rel), registry, ExpandOptions{})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	if len(generico.llamadas) != 1 {
		t.Errorf("se esperaba 1 consulta al resolver genérico; recibidas: %d", len(generico.llamadas))
	}
	if !contieneAviso(out.Warnings, KindGenericResolver) {
		t.Errorf("se esperaba un aviso generic_resolver; avisos: %+v", out.Warnings)
	}
	// Exactamente un aviso generic_resolver para esta ruta.
	var n int
	for _, a := range out.Warnings {
		if a.Kind == KindGenericResolver {
			n++
		}
	}
	if n != 1 {
		t.Errorf("se esperaba exactamente 1 aviso generic_resolver; recibidos: %d", n)
	}
}

// El expansor no conoce el mapeo relación→resolver salvo por inyección. Sin
// resolver específico ni genérico, la relación no puede resolverse y, en modo
// notify, se reporta como aviso relation conservando el recurso principal.
func TestDispatchExpansorNoConoceMapeoSinInyeccion(t *testing.T) {
	// Registro completamente vacío: ni específicos ni genérico.
	registry := NewResolverRegistry(map[string]Resolver{}, nil)

	rel := relacionTerceroDispatch()
	data := map[string]any{"Id": float64(1), "TerceroId": float64(7)}

	out, err := Expand(context.Background(), data, cfgTerceroDispatch(rel), registry, ExpandOptions{})
	if err != nil {
		t.Fatalf("modo notify no debe devolver error; recibido: %v", err)
	}

	// El recurso principal se conserva.
	m := out.Data.(map[string]any)
	if m["Id"] != float64(1) {
		t.Error("el recurso principal debe conservarse cuando no hay resolver inyectado")
	}
	// La relación no resuelta queda null y se reporta como aviso relation.
	if valor, existe := m["Tercero"]; !existe || valor != nil {
		t.Errorf("la relación sin resolver inyectado debe quedar null; recibido: %v", valor)
	}
	if !contieneAviso(out.Warnings, KindRelation) {
		t.Errorf("se esperaba un aviso relation cuando el expansor no conoce el mapeo; avisos: %+v", out.Warnings)
	}
}
