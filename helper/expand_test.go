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
// constante declarada" y, recíprocamente, "toda constante declarada tiene una
// ruta de emisión que la respalda". El invariante es ABIERTO: el conjunto de Kind
// no se fija de forma cerrada, sino que se extiende declarando una constante por
// cada origen de aviso. El control no se relaja al abrirlo; se reparte por motor.
// Aquí se verifica que (1) cada constante declarada es distinta y no vacía, (2)
// para todo Kind emitido por el motor de expansión está respaldado por una constante
// declarada, (3) cada constante declarada tiene un origen asignado y cada origen
// asignado corresponde a una constante declarada, y (4) las constantes de origen
// expansión se emiten por rutas de expansión reales. La emisión de las constantes
// de origen escritura la respalda TestInvarianteKindEscritura, de modo que entre
// ambos tests ninguna constante queda declarada sin emisor.
func TestInvarianteKindsDeAviso(t *testing.T) {
	// 1. Cada constante declarada (derivada del catálogo) es distinta, no vacía y
	// tiene un origen registrado. Que toda constante tenga origen es cierto por
	// construcción —KindsDeAviso se deriva del catálogo— pero se reafirma aquí para
	// detectar una entrada vacía o duplicada.
	declarados := make(map[string]struct{})
	for _, k := range KindsDeAviso() {
		if strings.TrimSpace(k) == "" {
			t.Errorf("el catálogo de Kinds contiene una constante vacía")
		}
		if _, dup := declarados[k]; dup {
			t.Errorf("el catálogo de Kinds contiene un duplicado: %q", k)
		}
		if _, tieneOrigen := OrigenDeKind(k); !tieneOrigen {
			t.Errorf("la constante %q no tiene origen registrado en el catálogo", k)
		}
		declarados[k] = struct{}{}
	}

	// field_normalization es un origen declarado y distinto de relation (reservado
	// a fallos remotos tolerados).
	if _, ok := declarados[KindFieldNormalization]; !ok {
		t.Errorf("la constante %q debe estar declarada en el catálogo", KindFieldNormalization)
	}
	if KindFieldNormalization == KindRelation {
		t.Errorf("field_normalization no debe coincidir con relation; ambos valen %q", KindFieldNormalization)
	}

	// 2. Ejercer cada ruta de emisión del motor de expansión y recolectar los Kind
	// producidos; todos deben estar respaldados por una constante declarada.
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

	// 3. Cada constante cuyo origen registrado es expansión debe emitirse por
	// alguna ruta de expansión real (no hay constante de expansión muerta). El
	// origen se lee del catálogo de producción, no de una lista paralela en el
	// test. Las constantes de origen escritura las respalda TestInvarianteKindEscritura.
	for _, k := range KindsDeAviso() {
		origen, _ := OrigenDeKind(k)
		if origen != KindOrigenExpansion {
			continue
		}
		if _, ok := emitidos[k]; !ok {
			t.Errorf("la constante de expansión %q no fue emitida por ninguna ruta de expansión", k)
		}
	}
}

// Proyección estricta (grupo 2, task 2.1): con selección de fields para un nivel,
// la entidad anidada se entrega SOLO con los escalares seleccionados, sin los
// demás escalares (incluido el Id cuando no se seleccionó) ni campos fuera de la
// selección, y sin aviso en MetaData.
func TestExpandProyeccionEstrictaFields(t *testing.T) {
	// El resolver devuelve un registro con más escalares de los seleccionados.
	res := &resolverFake{registros: []map[string]any{
		{"Id": float64(7), "NombreCompleto": "Ana", "Otro": "z"},
	}}
	registry := NewResolverRegistry(map[string]Resolver{"tercero": res}, nil)

	data := map[string]any{"Id": float64(1), "TerceroId": float64(7)}
	// Depth de la relación igual al global para no emitir aviso depth y aislar
	// que la proyección por sí misma no produce avisos.
	rel := relacionesPrueba()[0]
	rel.Depth = 1
	cfg := config.ServiceConfig{
		Service: "s", MaxDepth: 1, OnRelationError: config.OnRelationErrorNotify, MaxRelatedIDs: 100,
		Relations: []config.RelationConfig{rel},
	}

	out, err := Expand(context.Background(), data, cfg, registry, ExpandOptions{
		Fields: []string{"Tercero.NombreCompleto"},
	})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	m := out.Data.(map[string]any)
	tercero, ok := m["Tercero"].(map[string]any)
	if !ok {
		t.Fatalf("se esperaba la relación Tercero; recibido: %T", m["Tercero"])
	}
	// Solo NombreCompleto debe estar presente.
	if tercero["NombreCompleto"] != "Ana" {
		t.Errorf("se esperaba NombreCompleto=Ana; recibido: %+v", tercero)
	}
	if _, existe := tercero["Id"]; existe {
		t.Errorf("Id no fue seleccionado y no debe entregarse; recibido: %+v", tercero)
	}
	if _, existe := tercero["Otro"]; existe {
		t.Errorf("Otro no fue seleccionado y no debe entregarse; recibido: %+v", tercero)
	}
	if len(tercero) != 1 {
		t.Errorf("Tercero debe tener exactamente 1 campo escalar seleccionado; recibido: %+v", tercero)
	}
	// La proyección no emite avisos.
	if len(out.Warnings) != 0 {
		t.Errorf("la proyección no debe emitir avisos; avisos: %+v", out.Warnings)
	}
}

// Proyección estricta conserva las relaciones hijas materializadas junto a los
// escalares seleccionados del nivel.
func TestExpandProyeccionConservaHijas(t *testing.T) {
	// Relación padre Tercero (servicio "tercero") con hija Info (servicio "info")
	// para que cada nivel use su propio resolver y no se mezclen registros.
	rel := config.RelationConfig{
		Name: "Tercero", RelatedService: "tercero", RelatedAPI: "terceros", RelatedTable: "tercero",
		Direction: config.DirectionLocalToRelated, LocalField: "TerceroId", RelatedField: "Id",
		Cardinality: config.CardinalityOne, Depth: 3, MappedName: "Tercero",
		Endpoint: "/v1/tercero", QueryTemplate: "Id:{ids}", AllowedFields: []string{"Id", "NombreCompleto"},
		NestedRelations: []config.RelationConfig{
			{
				Name: "InfoComplementariaTercero", RelatedService: "info", RelatedAPI: "terceros", RelatedTable: "info",
				Direction: config.DirectionRelatedToLocal, LocalField: "Id", RelatedField: "TerceroId",
				Cardinality: config.CardinalityMany, Depth: 2, MappedName: "InfoComplementariaTercero",
				Endpoint: "/v1/info", QueryTemplate: "TerceroId:{ids}", AllowedFields: []string{"Id", "Descripcion"},
			},
		},
	}
	resTercero := &resolverFake{registros: []map[string]any{{"Id": float64(7), "NombreCompleto": "Ana", "Otro": "z"}}}
	resInfo := &resolverFake{registros: []map[string]any{{"Id": float64(50), "TerceroId": float64(7), "Descripcion": "d"}}}
	registry := NewResolverRegistry(map[string]Resolver{"tercero": resTercero, "info": resInfo}, nil)

	data := map[string]any{"Id": float64(1), "TerceroId": float64(7)}
	cfg := config.ServiceConfig{
		Service: "s", MaxDepth: 3, OnRelationError: config.OnRelationErrorNotify, MaxRelatedIDs: 100,
		Relations: []config.RelationConfig{rel},
	}

	out, err := Expand(context.Background(), data, cfg, registry, ExpandOptions{
		Fields: []string{"Tercero.NombreCompleto", "Tercero.InfoComplementariaTercero.Descripcion"},
	})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	m := out.Data.(map[string]any)
	tercero, ok := m["Tercero"].(map[string]any)
	if !ok {
		t.Fatalf("se esperaba la relación Tercero; recibido: %T", m["Tercero"])
	}
	if tercero["NombreCompleto"] != "Ana" {
		t.Errorf("se esperaba NombreCompleto=Ana; recibido: %+v", tercero)
	}
	if _, existe := tercero["Otro"]; existe {
		t.Errorf("Otro no seleccionado no debe entregarse; recibido: %+v", tercero)
	}
	// La relación hija seleccionada se conserva pese a la proyección de escalares.
	if _, existe := tercero["InfoComplementariaTercero"]; !existe {
		t.Errorf("la relación hija seleccionada debe conservarse; recibido: %+v", tercero)
	}
}

// Sin selección de fields (grupo 2, task 2.2): no se aplica recorte proyectivo;
// la entidad conserva los campos que entrega el resolver.
func TestExpandSinFieldsNoRecorta(t *testing.T) {
	res := &resolverFake{registros: []map[string]any{
		{"Id": float64(7), "NombreCompleto": "Ana", "Otro": "z"},
	}}
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
	tercero, ok := m["Tercero"].(map[string]any)
	if !ok {
		t.Fatalf("se esperaba la relación Tercero; recibido: %T", m["Tercero"])
	}
	// Sin fields, el registro conserva todos sus campos (sin recorte).
	for _, campo := range []string{"Id", "NombreCompleto", "Otro"} {
		if _, existe := tercero[campo]; !existe {
			t.Errorf("sin fields, el campo %q debe conservarse; recibido: %+v", campo, tercero)
		}
	}
}

// matrizRelaciones es el fixture compartido de la matriz de selección: Tercero
// (one) con dos relaciones hijas, Datos (many) y Perfil (one), y una hermana de
// nivel raíz Documento (one) para probar la precedencia por relación.
func matrizRelaciones() []config.RelationConfig {
	return []config.RelationConfig{
		{
			Name: "Tercero", RelatedService: "tercero", RelatedAPI: "terceros", RelatedTable: "tercero",
			Direction: config.DirectionLocalToRelated, LocalField: "TerceroId", RelatedField: "Id",
			Cardinality: config.CardinalityOne, Depth: 3, MappedName: "Tercero",
			AllowedFields: []string{"Id", "NombreCompleto"},
			NestedRelations: []config.RelationConfig{
				{
					Name: "Datos", RelatedService: "datos", RelatedAPI: "terceros", RelatedTable: "datos",
					Direction: config.DirectionRelatedToLocal, LocalField: "Id", RelatedField: "TerceroId",
					Cardinality: config.CardinalityMany, Depth: 2, MappedName: "Datos",
					AllowedFields: []string{"Id", "Numero"},
				},
				{
					Name: "Perfil", RelatedService: "perfil", RelatedAPI: "terceros", RelatedTable: "perfil",
					Direction: config.DirectionLocalToRelated, LocalField: "PerfilId", RelatedField: "Id",
					Cardinality: config.CardinalityOne, Depth: 2, MappedName: "Perfil",
					AllowedFields: []string{"Id", "Nombre"},
				},
			},
		},
		{
			Name: "Documento", RelatedService: "documento", RelatedAPI: "terceros", RelatedTable: "documento",
			Direction: config.DirectionLocalToRelated, LocalField: "DocumentoId", RelatedField: "Id",
			Cardinality: config.CardinalityOne, Depth: 2, MappedName: "Documento",
			AllowedFields: []string{"Id", "Serie"},
		},
	}
}

// matrizRegistry construye un registro con resolvers independientes por servicio
// para el fixture matrizRelaciones, de modo que cada nivel se resuelva por
// separado y se pueda contar cuándo una relación no debe consultarse.
func matrizRegistry() (ResolverRegistry, map[string]*resolverFake) {
	fakes := map[string]*resolverFake{
		"tercero":   {registros: []map[string]any{{"Id": float64(7), "NombreCompleto": "Ana", "PerfilId": float64(9)}}},
		"datos":     {registros: []map[string]any{{"Id": float64(50), "TerceroId": float64(7), "Numero": "123"}}},
		"perfil":    {registros: []map[string]any{{"Id": float64(9), "Nombre": "P"}}},
		"documento": {registros: []map[string]any{{"Id": float64(3), "Serie": "A"}}},
	}
	esp := make(map[string]Resolver, len(fakes))
	for k, v := range fakes {
		esp[k] = v
	}
	return NewResolverRegistry(esp, nil), fakes
}

// claves devuelve el conjunto ordenado de claves de un mapa, para aserciones
// estables sobre qué campos quedaron tras la proyección.
func claves(m map[string]any) map[string]struct{} {
	s := make(map[string]struct{}, len(m))
	for k := range m {
		s[k] = struct{}{}
	}
	return s
}

func tieneExactamente(m map[string]any, esperados ...string) bool {
	if len(m) != len(esperados) {
		return false
	}
	c := claves(m)
	for _, e := range esperados {
		if _, ok := c[e]; !ok {
			return false
		}
	}
	return true
}

// TestExpandMatrizSeleccion cubre la matriz de selección en el mismo nivel con el
// fixture compartido matrizRelaciones, verificando la regla del último segmento y
// la precedencia por relación de la selección más amplia.
func TestExpandMatrizSeleccion(t *testing.T) {
	cfg := config.ServiceConfig{
		Service: "s", MaxDepth: 3, OnRelationError: config.OnRelationErrorNotify, MaxRelatedIDs: 100,
		Relations: matrizRelaciones(),
	}
	base := func() map[string]any {
		return map[string]any{"Id": float64(1), "TerceroId": float64(7), "DocumentoId": float64(3)}
	}

	t.Run("1 [escalar] solo ese escalar", func(t *testing.T) {
		reg, _ := matrizRegistry()
		out, err := Expand(context.Background(), base(), cfg, reg, ExpandOptions{Fields: []string{"Tercero.NombreCompleto"}})
		if err != nil {
			t.Fatal(err)
		}
		tercero := out.Data.(map[string]any)["Tercero"].(map[string]any)
		if !tieneExactamente(tercero, "NombreCompleto") {
			t.Errorf("esperado solo NombreCompleto; recibido: %+v", tercero)
		}
	})

	t.Run("2 [relacion hoja] completa", func(t *testing.T) {
		reg, _ := matrizRegistry()
		out, err := Expand(context.Background(), base(), cfg, reg, ExpandOptions{Fields: []string{"Tercero"}})
		if err != nil {
			t.Fatal(err)
		}
		tercero := out.Data.(map[string]any)["Tercero"].(map[string]any)
		// Completa: escalares + relaciones hijas expandidas.
		for _, c := range []string{"Id", "NombreCompleto", "Datos", "Perfil"} {
			if _, ok := tercero[c]; !ok {
				t.Errorf("relación completa debe incluir %q; recibido: %+v", c, tercero)
			}
		}
	})

	t.Run("3 [escalar,escalar] solo esos", func(t *testing.T) {
		reg, _ := matrizRegistry()
		out, err := Expand(context.Background(), base(), cfg, reg, ExpandOptions{Fields: []string{"Tercero.Id", "Tercero.NombreCompleto"}})
		if err != nil {
			t.Fatal(err)
		}
		tercero := out.Data.(map[string]any)["Tercero"].(map[string]any)
		if !tieneExactamente(tercero, "Id", "NombreCompleto") {
			t.Errorf("esperado solo Id y NombreCompleto; recibido: %+v", tercero)
		}
	})

	t.Run("4 [escalar,relacion-hoja misma] completa subsume escalar", func(t *testing.T) {
		reg, _ := matrizRegistry()
		out, err := Expand(context.Background(), base(), cfg, reg, ExpandOptions{Fields: []string{"Tercero.Id", "Tercero"}})
		if err != nil {
			t.Fatal(err)
		}
		tercero := out.Data.(map[string]any)["Tercero"].(map[string]any)
		for _, c := range []string{"Id", "NombreCompleto", "Datos", "Perfil"} {
			if _, ok := tercero[c]; !ok {
				t.Errorf("la relación completa debe subsumir el escalar; falta %q en %+v", c, tercero)
			}
		}
	})

	t.Run("5 [relacion hoja, relacion hoja] distintas coexisten completas", func(t *testing.T) {
		reg, _ := matrizRegistry()
		out, err := Expand(context.Background(), base(), cfg, reg, ExpandOptions{Fields: []string{"Tercero", "Documento"}})
		if err != nil {
			t.Fatal(err)
		}
		m := out.Data.(map[string]any)
		tercero, okT := m["Tercero"].(map[string]any)
		documento, okD := m["Documento"].(map[string]any)
		if !okT || !okD {
			t.Fatalf("ambas relaciones deben expandirse; recibido: %+v", m)
		}
		if _, ok := tercero["Datos"]; !ok {
			t.Errorf("Tercero completo debe incluir Datos; recibido: %+v", tercero)
		}
		if _, ok := documento["Serie"]; !ok {
			t.Errorf("Documento completo debe incluir Serie; recibido: %+v", documento)
		}
	})

	t.Run("6 [relacion hoja, descendiente misma] completa subsume descendiente", func(t *testing.T) {
		reg, _ := matrizRegistry()
		out, err := Expand(context.Background(), base(), cfg, reg, ExpandOptions{Fields: []string{"Tercero", "Tercero.NombreCompleto"}})
		if err != nil {
			t.Fatal(err)
		}
		tercero := out.Data.(map[string]any)["Tercero"].(map[string]any)
		// Completa, no proyectada a solo NombreCompleto.
		for _, c := range []string{"Id", "NombreCompleto", "Datos", "Perfil"} {
			if _, ok := tercero[c]; !ok {
				t.Errorf("la relación completa domina al descendiente; falta %q en %+v", c, tercero)
			}
		}
	})

	t.Run("7 [descendiente,descendiente] misma relacion conviven como esqueleto", func(t *testing.T) {
		reg, _ := matrizRegistry()
		out, err := Expand(context.Background(), base(), cfg, reg, ExpandOptions{Fields: []string{"Tercero.Datos.Numero", "Tercero.Perfil.Nombre"}})
		if err != nil {
			t.Fatal(err)
		}
		tercero := out.Data.(map[string]any)["Tercero"].(map[string]any)
		// Esqueleto: solo las dos ramas, sin escalares del padre.
		if !tieneExactamente(tercero, "Datos", "Perfil") {
			t.Errorf("esperado esqueleto con Datos y Perfil; recibido: %+v", tercero)
		}
		datos := tercero["Datos"].([]any)
		if len(datos) != 1 || !tieneExactamente(datos[0].(map[string]any), "Numero") {
			t.Errorf("Datos debe proyectarse a solo Numero; recibido: %+v", tercero["Datos"])
		}
		perfil := tercero["Perfil"].(map[string]any)
		if !tieneExactamente(perfil, "Nombre") {
			t.Errorf("Perfil debe proyectarse a solo Nombre; recibido: %+v", perfil)
		}
	})

	t.Run("8 [relacion hoja, descendiente de hermana] trato distinto por relacion", func(t *testing.T) {
		reg, fakes := matrizRegistry()
		// Tercero completo (hoja) + Documento solo por un descendiente (esqueleto).
		out, err := Expand(context.Background(), base(), cfg, reg, ExpandOptions{Fields: []string{"Tercero", "Documento.Serie"}})
		if err != nil {
			t.Fatal(err)
		}
		m := out.Data.(map[string]any)
		tercero := m["Tercero"].(map[string]any)
		documento := m["Documento"].(map[string]any)
		// Tercero completo no suprime a la hermana Documento (precedencia por relación).
		if _, ok := tercero["Datos"]; !ok {
			t.Errorf("Tercero completo debe incluir su subárbol; recibido: %+v", tercero)
		}
		if !tieneExactamente(documento, "Serie") {
			t.Errorf("Documento hermana debe ser esqueleto con solo Serie; recibido: %+v", documento)
		}
		_ = fakes
	})

	t.Run("relacion no alcanzada no se consulta", func(t *testing.T) {
		reg, fakes := matrizRegistry()
		// Solo se pide Tercero.NombreCompleto: Documento (hermana raíz) no se toca.
		_, err := Expand(context.Background(), base(), cfg, reg, ExpandOptions{Fields: []string{"Tercero.NombreCompleto"}})
		if err != nil {
			t.Fatal(err)
		}
		if len(fakes["documento"].llamadas) != 0 {
			t.Errorf("Documento no fue seleccionado y no debe consultarse; llamadas: %v", fakes["documento"].llamadas)
		}
		// Perfil tampoco (no se pidió, y Tercero es parcial, no completo).
		if len(fakes["perfil"].llamadas) != 0 {
			t.Errorf("Perfil no seleccionado no debe consultarse; llamadas: %v", fakes["perfil"].llamadas)
		}
	})
}

// Id implícito en expansión (grupo 3, task 3.2): sin fields, la entidad expandida
// conserva su Id aunque allowed_fields de la relación no lo liste.
func TestExpandIdImplicitoSinFields(t *testing.T) {
	// Tercero con allowed_fields SIN Id; el resolver sí devuelve Id.
	rel := config.RelationConfig{
		Name: "Tercero", RelatedService: "tercero", RelatedAPI: "terceros", RelatedTable: "tercero",
		Direction: config.DirectionLocalToRelated, LocalField: "TerceroId", RelatedField: "Id",
		Cardinality: config.CardinalityOne, Depth: 1, MappedName: "Tercero",
		AllowedFields: []string{"NombreCompleto"},
	}
	res := &resolverFake{registros: []map[string]any{{"Id": float64(7), "NombreCompleto": "Ana"}}}
	registry := NewResolverRegistry(map[string]Resolver{"tercero": res}, nil)

	data := map[string]any{"Id": float64(1), "TerceroId": float64(7)}
	cfg := config.ServiceConfig{
		Service: "s", MaxDepth: 1, OnRelationError: config.OnRelationErrorNotify, MaxRelatedIDs: 100,
		Relations: []config.RelationConfig{rel},
	}

	out, err := Expand(context.Background(), data, cfg, registry, ExpandOptions{})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	tercero, ok := out.Data.(map[string]any)["Tercero"].(map[string]any)
	if !ok {
		t.Fatalf("se esperaba la relación Tercero; recibido: %T", out.Data.(map[string]any)["Tercero"])
	}
	if _, existe := tercero["Id"]; !existe {
		t.Errorf("el Id implícito debe conservarse sin fields aunque allowed_fields no lo liste; recibido: %+v", tercero)
	}
}

// Omisión explícita del Id por fields: cuando la selección de una relación no
// menciona su Id, el Id NO se incluye para esa solicitud y no se emite aviso. Es
// la contraparte del Id implícito: implícito por defecto (sin fields), pero
// omitible como cualquier escalar cuando hay selección parcial que no lo nombra.
func TestExpandIdOmitidoSiNoEstaEnFields(t *testing.T) {
	// allowed_fields SÍ incluye Id; el resolver lo devuelve. Aun así, al pedir
	// solo NombreCompleto, el Id no debe entregarse.
	rel := config.RelationConfig{
		Name: "Tercero", RelatedService: "tercero", RelatedAPI: "terceros", RelatedTable: "tercero",
		Direction: config.DirectionLocalToRelated, LocalField: "TerceroId", RelatedField: "Id",
		Cardinality: config.CardinalityOne, Depth: 1, MappedName: "Tercero",
		AllowedFields: []string{"Id", "NombreCompleto"},
	}
	res := &resolverFake{registros: []map[string]any{{"Id": float64(7), "NombreCompleto": "Ana"}}}
	registry := NewResolverRegistry(map[string]Resolver{"tercero": res}, nil)

	data := map[string]any{"Id": float64(1), "TerceroId": float64(7)}
	cfg := config.ServiceConfig{
		Service: "s", MaxDepth: 1, OnRelationError: config.OnRelationErrorNotify, MaxRelatedIDs: 100,
		Relations: []config.RelationConfig{rel},
	}

	out, err := Expand(context.Background(), data, cfg, registry, ExpandOptions{
		Fields: []string{"Tercero.NombreCompleto"},
	})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	tercero := out.Data.(map[string]any)["Tercero"].(map[string]any)
	if _, existe := tercero["Id"]; existe {
		t.Errorf("Id no fue seleccionado en fields y no debe incluirse; recibido: %+v", tercero)
	}
	if !tieneExactamente(tercero, "NombreCompleto") {
		t.Errorf("debe entregarse solo NombreCompleto; recibido: %+v", tercero)
	}
	// La omisión del Id es silenciosa: no emite aviso.
	if len(out.Warnings) != 0 {
		t.Errorf("la omisión del Id no debe emitir aviso; avisos: %+v", out.Warnings)
	}
}
