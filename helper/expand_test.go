package helper

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/udistrital/agora_mid/config"
	"github.com/udistrital/agora_mid/responses"
)

// resolverFake simula un Resolver devolviendo registros fijos por servicio y
// contando las invocaciones para verificar deduplicación y caché.
type resolverFake struct {
	// registros son los registros que devuelve en cada llamada (filtrados luego
	// por related_field en el expansor).
	registros []map[string]any
	// err, si no es nil, se devuelve en cada llamada.
	err error
	// generico marca el resolver como genérico.
	generico bool
	// llamadas acumula las llaves recibidas en cada invocación.
	llamadas [][]string
}

func (r *resolverFake) Resolve(_ context.Context, _ config.RelationConfig, keys []string) ([]map[string]any, error) {
	r.llamadas = append(r.llamadas, append([]string{}, keys...))
	if r.err != nil {
		return nil, r.err
	}
	return r.registros, nil
}

func (r *resolverFake) IsGeneric() bool { return r.generico }

func cfgPrueba() config.ServiceConfig {
	return config.ServiceConfig{
		Service:         "registro_persona",
		MaxDepth:        3,
		OnRelationError: config.OnRelationErrorNotify,
		MaxRelatedIDs:   100,
		Relations:       relacionesPrueba(),
	}
}

// one encontrada: inserta el objeto bajo mapped_name y conserva la llave local.
func TestExpandOneEncontrada(t *testing.T) {
	res := &resolverFake{registros: []map[string]any{{"Id": float64(7), "NombreCompleto": "Ana"}}}
	registry := NewResolverRegistry(map[string]Resolver{"tercero": res}, nil)

	data := map[string]any{"Id": float64(1), "TerceroId": float64(7)}
	cfg := config.ServiceConfig{
		Service: "s", MaxDepth: 1, OnRelationError: config.OnRelationErrorNotify, MaxRelatedIDs: 100,
		Relations: []config.RelationConfig{relacionesPrueba()[0]},
	}

	out, err := Expand(context.Background(), data, cfg, registry, ExpandOptions{})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	m := out.Data.(map[string]any)
	if m["TerceroId"] != float64(7) {
		t.Errorf("se debe conservar la llave local TerceroId; recibido: %v", m["TerceroId"])
	}
	tercero, ok := m["Tercero"].(map[string]any)
	if !ok {
		t.Fatalf("Tercero debe ser un objeto; recibido: %T", m["Tercero"])
	}
	if tercero["NombreCompleto"] != "Ana" {
		t.Errorf("Tercero inesperado: %+v", tercero)
	}
}

// one sin resultado: mapped_name = null.
func TestExpandOneSinResultado(t *testing.T) {
	res := &resolverFake{registros: []map[string]any{}}
	registry := NewResolverRegistry(map[string]Resolver{"tercero": res}, nil)

	data := map[string]any{"Id": float64(1), "TerceroId": float64(9)}
	cfg := config.ServiceConfig{
		Service: "s", MaxDepth: 1, OnRelationError: config.OnRelationErrorNotify, MaxRelatedIDs: 100,
		Relations: []config.RelationConfig{relacionesPrueba()[0]},
	}

	out, err := Expand(context.Background(), data, cfg, registry, ExpandOptions{})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	m := out.Data.(map[string]any)
	if valor, existe := m["Tercero"]; !existe || valor != nil {
		t.Errorf("one sin resultado debe ser null; recibido: %v (existe=%v)", valor, existe)
	}
}

// many con cero, uno y varios resultados.
func TestExpandManyVariosCasos(t *testing.T) {
	rel := relacionesPrueba()[0].NestedRelations[0] // InfoComplementariaTercero (many)
	rel.Depth = 1
	cfg := config.ServiceConfig{
		Service: "s", MaxDepth: 1, OnRelationError: config.OnRelationErrorNotify, MaxRelatedIDs: 100,
		Relations: []config.RelationConfig{rel},
	}

	casos := []struct {
		nombre    string
		registros []map[string]any
		esperado  int
	}{
		{"cero", []map[string]any{}, 0},
		{"uno", []map[string]any{{"Id": float64(1), "TerceroId": float64(5)}}, 1},
		{"varios", []map[string]any{
			{"Id": float64(1), "TerceroId": float64(5)},
			{"Id": float64(2), "TerceroId": float64(5)},
		}, 2},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			res := &resolverFake{registros: c.registros}
			registry := NewResolverRegistry(map[string]Resolver{"tercero": res}, nil)
			data := map[string]any{"Id": float64(5)}
			out, err := Expand(context.Background(), data, cfg, registry, ExpandOptions{})
			if err != nil {
				t.Fatalf("error inesperado: %v", err)
			}
			m := out.Data.(map[string]any)
			lista, ok := m["InfoComplementariaTercero"].([]any)
			if !ok {
				t.Fatalf("many debe ser siempre una lista; recibido: %T", m["InfoComplementariaTercero"])
			}
			if len(lista) != c.esperado {
				t.Errorf("se esperaban %d elementos; recibidos: %d", c.esperado, len(lista))
			}
		})
	}
}

// Deduplicación de llaves y caché por solicitud: varios registros con la misma
// llave producen una sola consulta.
func TestExpandDedupeYCache(t *testing.T) {
	res := &resolverFake{registros: []map[string]any{{"Id": float64(7), "NombreCompleto": "Ana"}}}
	registry := NewResolverRegistry(map[string]Resolver{"tercero": res}, nil)

	data := []any{
		map[string]any{"Id": float64(1), "TerceroId": float64(7)},
		map[string]any{"Id": float64(2), "TerceroId": float64(7)},
		map[string]any{"Id": float64(3), "TerceroId": float64(7)},
	}
	cfg := config.ServiceConfig{
		Service: "s", MaxDepth: 1, OnRelationError: config.OnRelationErrorNotify, MaxRelatedIDs: 100,
		Relations: []config.RelationConfig{relacionesPrueba()[0]},
	}

	out, err := Expand(context.Background(), data, cfg, registry, ExpandOptions{})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	lista := out.Data.([]any)
	if len(lista) != 3 {
		t.Fatalf("se esperaban 3 recursos; recibidos: %d", len(lista))
	}
	// Una sola llamada al resolver por la llave 7 (dedupe + caché).
	if len(res.llamadas) != 1 {
		t.Errorf("se esperaba 1 consulta por dedupe/caché; recibidas: %d (%v)", len(res.llamadas), res.llamadas)
	}
}

// Ciclo detectado: Persona -> Tercero -> Persona detiene la rama con aviso cycle.
func TestExpandCicloDetectado(t *testing.T) {
	// Tercero (one) con hija Persona (one) que vuelve a mapear "Tercero".
	tercero := config.RelationConfig{
		Name: "Tercero", RelatedService: "tercero", RelatedAPI: "terceros", RelatedTable: "t",
		Direction: config.DirectionLocalToRelated, LocalField: "TerceroId", RelatedField: "Id",
		Cardinality: config.CardinalityOne, Depth: 5, MappedName: "Tercero",
		Endpoint: "/v1/tercero", QueryTemplate: "Id:{ids}", AllowedFields: []string{"Id"},
		NestedRelations: []config.RelationConfig{
			{
				Name: "Tercero", RelatedService: "tercero", RelatedAPI: "terceros", RelatedTable: "t",
				Direction: config.DirectionLocalToRelated, LocalField: "TerceroId", RelatedField: "Id",
				Cardinality: config.CardinalityOne, Depth: 5, MappedName: "Tercero",
				Endpoint: "/v1/tercero", QueryTemplate: "Id:{ids}", AllowedFields: []string{"Id"},
			},
		},
	}
	res := &resolverFake{registros: []map[string]any{{"Id": float64(7), "TerceroId": float64(7)}}}
	registry := NewResolverRegistry(map[string]Resolver{"tercero": res}, nil)

	data := map[string]any{"TerceroId": float64(7)}
	cfg := config.ServiceConfig{
		Service: "s", MaxDepth: 5, OnRelationError: config.OnRelationErrorNotify, MaxRelatedIDs: 100,
		Relations: []config.RelationConfig{tercero},
	}

	out, err := Expand(context.Background(), data, cfg, registry, ExpandOptions{})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if !contieneAviso(out.Warnings, KindCycle) {
		t.Errorf("se esperaba un aviso cycle; avisos: %+v", out.Warnings)
	}
}

// Límite de IDs: con max_related_ids pequeño las consultas se particionan.
func TestExpandLimiteIDs(t *testing.T) {
	rel := relacionesPrueba()[0].NestedRelations[0] // many por TerceroId
	rel.Depth = 1
	res := &resolverFake{registros: []map[string]any{}}
	registry := NewResolverRegistry(map[string]Resolver{"tercero": res}, nil)

	data := []any{
		map[string]any{"Id": float64(1)},
		map[string]any{"Id": float64(2)},
		map[string]any{"Id": float64(3)},
	}
	cfg := config.ServiceConfig{
		Service: "s", MaxDepth: 1, OnRelationError: config.OnRelationErrorNotify, MaxRelatedIDs: 2,
		Relations: []config.RelationConfig{rel},
	}

	_, err := Expand(context.Background(), data, cfg, registry, ExpandOptions{})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	// 3 llaves distintas (1,2,3) con lotes de 2 => 2 particiones.
	if len(res.llamadas) != 2 {
		t.Errorf("se esperaban 2 lotes por max_related_ids=2; recibidos: %d (%v)", len(res.llamadas), res.llamadas)
	}
}

// max_depth global detiene la expansión por encima del límite.
func TestExpandMaxDepthGlobal(t *testing.T) {
	// Tercero.depth=5 pero MaxDepth global=1 => solo un nivel.
	rels := relacionesPrueba()
	rels[0].Depth = 5
	res := &resolverFake{registros: []map[string]any{{"Id": float64(7)}}}
	registry := NewResolverRegistry(map[string]Resolver{"tercero": res}, nil)

	data := map[string]any{"TerceroId": float64(7)}
	cfg := config.ServiceConfig{
		Service: "s", MaxDepth: 1, OnRelationError: config.OnRelationErrorNotify, MaxRelatedIDs: 100,
		Relations: []config.RelationConfig{rels[0]},
	}

	out, err := Expand(context.Background(), data, cfg, registry, ExpandOptions{})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	m := out.Data.(map[string]any)
	tercero, ok := m["Tercero"].(map[string]any)
	if !ok {
		t.Fatalf("Tercero debe materializarse en el primer nivel; recibido: %T", m["Tercero"])
	}
	// La hija InfoComplementariaTercero no debe expandirse (max_depth global=1).
	if _, existe := tercero["InfoComplementariaTercero"]; existe {
		t.Error("no se debe expandir por encima de max_depth global")
	}
}

// Config hija con menor profundidad reduce la rama y emite aviso depth.
func TestExpandReduccionPorConfigHija(t *testing.T) {
	// Tercero.depth=1 pero MaxDepth global=3 => efectiva=1 y aviso depth.
	rels := relacionesPrueba()
	rels[0].Depth = 1
	res := &resolverFake{registros: []map[string]any{{"Id": float64(7)}}}
	registry := NewResolverRegistry(map[string]Resolver{"tercero": res}, nil)

	data := map[string]any{"TerceroId": float64(7)}
	cfg := config.ServiceConfig{
		Service: "s", MaxDepth: 3, OnRelationError: config.OnRelationErrorNotify, MaxRelatedIDs: 100,
		Relations: []config.RelationConfig{rels[0]},
	}

	out, err := Expand(context.Background(), data, cfg, registry, ExpandOptions{})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if !contieneAviso(out.Warnings, KindDepth) {
		t.Errorf("se esperaba un aviso depth por reducción de profundidad; avisos: %+v", out.Warnings)
	}
}

// Modo notify: error de relación conserva el principal y agrega aviso relation.
func TestExpandNotify(t *testing.T) {
	res := &resolverFake{err: errors.New("503 service unavailable")}
	registry := NewResolverRegistry(map[string]Resolver{"tercero": res}, nil)

	data := map[string]any{"Id": float64(1), "TerceroId": float64(7)}
	cfg := config.ServiceConfig{
		Service: "s", MaxDepth: 1, OnRelationError: config.OnRelationErrorNotify, MaxRelatedIDs: 100,
		Relations: []config.RelationConfig{relacionesPrueba()[0]},
	}

	out, err := Expand(context.Background(), data, cfg, registry, ExpandOptions{})
	if err != nil {
		t.Fatalf("notify no debe devolver error; recibido: %v", err)
	}
	m := out.Data.(map[string]any)
	if m["Id"] != float64(1) {
		t.Error("el recurso principal debe conservarse en modo notify")
	}
	if valor, existe := m["Tercero"]; !existe || valor != nil {
		t.Errorf("la rama fallida one debe ser null; recibido: %v", valor)
	}
	if !contieneAviso(out.Warnings, KindRelation) {
		t.Errorf("se esperaba un aviso relation; avisos: %+v", out.Warnings)
	}
	// El aviso no debe exponer el cuerpo remoto.
	for _, a := range out.Warnings {
		if a.Kind == KindRelation && strings.Contains(a.Message, "503 service unavailable") {
			t.Errorf("el aviso no debe exponer el cuerpo remoto: %q", a.Message)
		}
	}
}

// Modo fail: error de relación aborta toda la expansión.
func TestExpandFail(t *testing.T) {
	res := &resolverFake{err: errors.New("504 timeout")}
	registry := NewResolverRegistry(map[string]Resolver{"tercero": res}, nil)

	data := map[string]any{"Id": float64(1), "TerceroId": float64(7)}
	cfg := config.ServiceConfig{
		Service: "s", MaxDepth: 1, OnRelationError: config.OnRelationErrorFail, MaxRelatedIDs: 100,
		Relations: []config.RelationConfig{relacionesPrueba()[0]},
	}

	_, err := Expand(context.Background(), data, cfg, registry, ExpandOptions{})
	if err == nil {
		t.Fatal("modo fail debe abortar la expansión con error")
	}
	if !EsErrorFatalExpansion(err) {
		t.Errorf("se esperaba *errorFatalExpansion; recibido: %T", err)
	}
}

// Errores de configuración (on_relation_error inválido) son error normal.
func TestExpandOnRelationErrorInvalido(t *testing.T) {
	registry := NewResolverRegistry(map[string]Resolver{}, nil)
	data := map[string]any{"Id": float64(1)}
	cfg := config.ServiceConfig{
		Service: "s", MaxDepth: 1, OnRelationError: "invalido", MaxRelatedIDs: 100,
		Relations: nil,
	}
	_, err := Expand(context.Background(), data, cfg, registry, ExpandOptions{})
	if err == nil {
		t.Fatal("on_relation_error inválido debe ser error de configuración")
	}
	if EsErrorFatalExpansion(err) {
		t.Error("un error de configuración no debe ser error fatal de expansión")
	}
}

// Resolver genérico de respaldo: Data sin transformar + aviso generic_resolver.
func TestExpandResolverGenerico(t *testing.T) {
	generico := &resolverFake{
		registros: []map[string]any{{"Id": float64(7), "Campo": "x"}},
		generico:  true,
	}
	// Sin resolver especializado para "terceros": usa el genérico de respaldo.
	registry := NewResolverRegistry(map[string]Resolver{}, generico)

	data := map[string]any{"TerceroId": float64(7)}
	cfg := config.ServiceConfig{
		Service: "s", MaxDepth: 1, OnRelationError: config.OnRelationErrorNotify, MaxRelatedIDs: 100,
		Relations: []config.RelationConfig{relacionesPrueba()[0]},
	}

	out, err := Expand(context.Background(), data, cfg, registry, ExpandOptions{})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if !contieneAviso(out.Warnings, KindGenericResolver) {
		t.Errorf("se esperaba un aviso generic_resolver; avisos: %+v", out.Warnings)
	}
	m := out.Data.(map[string]any)
	tercero, ok := m["Tercero"].(map[string]any)
	if !ok || tercero["Campo"] != "x" {
		t.Errorf("el resolver genérico debe entregar Data sin transformar; recibido: %+v", m["Tercero"])
	}
}

// Selección por fields: solo se expanden las relaciones solicitadas.
func TestExpandConFieldsSeleccionados(t *testing.T) {
	resTerceros := &resolverFake{registros: []map[string]any{{"Id": float64(7), "NombreCompleto": "Ana"}}}
	resParam := &resolverFake{registros: []map[string]any{{"Id": float64(3), "Nombre": "CC"}}}
	registry := NewResolverRegistry(map[string]Resolver{
		"tercero":   resTerceros,
		"parametro": resParam,
	}, nil)

	data := map[string]any{"Id": float64(1), "TerceroId": float64(7), "TipoDocumentoId": float64(3)}
	cfg := cfgPrueba()

	// Solo se pide Tercero; TipoDocumento no debe consultarse.
	out, err := Expand(context.Background(), data, cfg, registry, ExpandOptions{
		Fields: []string{"Id", "Tercero.Id", "Tercero.NombreCompleto"},
	})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	m := out.Data.(map[string]any)
	if _, existe := m["Tercero"]; !existe {
		t.Error("se esperaba la relación Tercero seleccionada")
	}
	if _, existe := m["TipoDocumento"]; existe {
		t.Error("TipoDocumento no fue seleccionado y no debe expandirse")
	}
	if len(resParam.llamadas) != 0 {
		t.Errorf("no se debe consultar parametros si no se seleccionó; llamadas: %v", resParam.llamadas)
	}
}

// Fields inválido devuelve error (que el controlador mapea a 400).
func TestExpandFieldsInvalido(t *testing.T) {
	registry := NewResolverRegistry(map[string]Resolver{}, nil)
	data := map[string]any{"Id": float64(1)}
	cfg := cfgPrueba()

	_, err := Expand(context.Background(), data, cfg, registry, ExpandOptions{
		Fields: []string{"Tercero.Secreto"},
	})
	if err == nil {
		t.Fatal("se esperaba error por campo no permitido")
	}
	if _, ok := err.(*ErrorFields); !ok {
		t.Errorf("se esperaba *ErrorFields; recibido: %T", err)
	}
}

func contieneAviso(avisos []responses.RelationError, kind string) bool {
	for _, a := range avisos {
		if a.Kind == kind {
			return true
		}
	}
	return false
}

// TestInvarianteKindsDeAviso afirma el invariante "todo Kind emitido tiene una
// constante declarada": el conjunto declarado KindsDeAviso es exactamente
// {relation, depth, cycle, generic_resolver}, cada ruta de emisión produce un
// Kind dentro de ese conjunto y cada constante declarada se emite por alguna
// ruta (no hay constante muerta ni Kind fuera del conjunto).
func TestInvarianteKindsDeAviso(t *testing.T) {
	// 1. El conjunto declarado coincide exactamente con los cuatro canónicos.
	declarados := make(map[string]struct{}, len(KindsDeAviso))
	for _, k := range KindsDeAviso {
		if _, dup := declarados[k]; dup {
			t.Errorf("KindsDeAviso contiene un duplicado: %q", k)
		}
		declarados[k] = struct{}{}
	}
	canonicos := map[string]struct{}{
		KindRelation:        {},
		KindDepth:           {},
		KindCycle:           {},
		KindGenericResolver: {},
	}
	if len(declarados) != len(canonicos) {
		t.Fatalf("KindsDeAviso = %v; esperado exactamente %d constantes", KindsDeAviso, len(canonicos))
	}
	for k := range canonicos {
		if _, ok := declarados[k]; !ok {
			t.Errorf("la constante canónica %q no está en KindsDeAviso", k)
		}
	}

	// 2. Ejercer cada ruta de emisión y recolectar los Kind producidos.
	emitidos := make(map[string]struct{})
	recolectar := func(avisos []responses.RelationError) {
		for _, a := range avisos {
			if _, ok := declarados[a.Kind]; !ok {
				t.Errorf("Kind emitido %q sin constante declarada (aviso %+v)", a.Kind, a)
			}
			emitidos[a.Kind] = struct{}{}
		}
	}

	// relation: error de relación tolerado en modo notify.
	{
		res := &resolverFake{err: errors.New("503 service unavailable")}
		registry := NewResolverRegistry(map[string]Resolver{"tercero": res}, nil)
		data := map[string]any{"Id": float64(1), "TerceroId": float64(7)}
		cfg := config.ServiceConfig{
			Service: "s", MaxDepth: 1, OnRelationError: config.OnRelationErrorNotify, MaxRelatedIDs: 100,
			Relations: []config.RelationConfig{relacionesPrueba()[0]},
		}
		out, err := Expand(context.Background(), data, cfg, registry, ExpandOptions{})
		if err != nil {
			t.Fatalf("notify no debe devolver error: %v", err)
		}
		recolectar(out.Warnings)
	}

	// depth: config hija con menor profundidad que el límite global.
	{
		rels := relacionesPrueba()
		rels[0].Depth = 1
		res := &resolverFake{registros: []map[string]any{{"Id": float64(7)}}}
		registry := NewResolverRegistry(map[string]Resolver{"tercero": res}, nil)
		data := map[string]any{"TerceroId": float64(7)}
		cfg := config.ServiceConfig{
			Service: "s", MaxDepth: 3, OnRelationError: config.OnRelationErrorNotify, MaxRelatedIDs: 100,
			Relations: []config.RelationConfig{rels[0]},
		}
		out, err := Expand(context.Background(), data, cfg, registry, ExpandOptions{})
		if err != nil {
			t.Fatalf("expansión con reducción de profundidad no debe fallar: %v", err)
		}
		recolectar(out.Warnings)
	}

	// cycle: una relación cuya hija vuelve a mapear el mismo mapped_name.
	{
		tercero := config.RelationConfig{
			Name: "Tercero", RelatedService: "tercero", RelatedAPI: "terceros", RelatedTable: "t",
			Direction: config.DirectionLocalToRelated, LocalField: "TerceroId", RelatedField: "Id",
			Cardinality: config.CardinalityOne, Depth: 5, MappedName: "Tercero",
			Endpoint: "/v1/tercero", QueryTemplate: "Id:{ids}", AllowedFields: []string{"Id"},
			NestedRelations: []config.RelationConfig{
				{
					Name: "Tercero", RelatedService: "tercero", RelatedAPI: "terceros", RelatedTable: "t",
					Direction: config.DirectionLocalToRelated, LocalField: "TerceroId", RelatedField: "Id",
					Cardinality: config.CardinalityOne, Depth: 5, MappedName: "Tercero",
					Endpoint: "/v1/tercero", QueryTemplate: "Id:{ids}", AllowedFields: []string{"Id"},
				},
			},
		}
		res := &resolverFake{registros: []map[string]any{{"Id": float64(7), "TerceroId": float64(7)}}}
		registry := NewResolverRegistry(map[string]Resolver{"tercero": res}, nil)
		data := map[string]any{"TerceroId": float64(7)}
		cfg := config.ServiceConfig{
			Service: "s", MaxDepth: 5, OnRelationError: config.OnRelationErrorNotify, MaxRelatedIDs: 100,
			Relations: []config.RelationConfig{tercero},
		}
		out, err := Expand(context.Background(), data, cfg, registry, ExpandOptions{})
		if err != nil {
			t.Fatalf("expansión con ciclo no debe fallar: %v", err)
		}
		recolectar(out.Warnings)
	}

	// generic_resolver: relación sin resolver especializado usa el genérico.
	{
		generico := &resolverFake{registros: []map[string]any{{"Id": float64(7), "Campo": "x"}}, generico: true}
		registry := NewResolverRegistry(map[string]Resolver{}, generico)
		data := map[string]any{"TerceroId": float64(7)}
		cfg := config.ServiceConfig{
			Service: "s", MaxDepth: 1, OnRelationError: config.OnRelationErrorNotify, MaxRelatedIDs: 100,
			Relations: []config.RelationConfig{relacionesPrueba()[0]},
		}
		out, err := Expand(context.Background(), data, cfg, registry, ExpandOptions{})
		if err != nil {
			t.Fatalf("expansión con resolver genérico no debe fallar: %v", err)
		}
		recolectar(out.Warnings)
	}

	// 3. Cada constante declarada se emitió por alguna ruta (no hay constante muerta).
	for k := range declarados {
		if _, ok := emitidos[k]; !ok {
			t.Errorf("la constante declarada %q no fue emitida por ninguna ruta de expansión", k)
		}
	}
}
