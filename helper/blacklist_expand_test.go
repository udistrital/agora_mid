package helper

import (
	"context"
	"testing"

	"github.com/udistrital/agora_mid/config"
)

// Task 5.1: el choke-point recorta los campos vetados de la salida del resolver
// (específico) antes de anidar. Un campo permitido por allowed_fields pero vetado
// se omite, y un Id implícito vetado se omite, sin aviso en MetaData (la supresión
// es silenciosa).
func TestExpandRecorteListaNegraResolverEspecifico(t *testing.T) {
	// El resolver entrega Id (implícito) y NombreCompleto (permitido). La lista
	// negra veta ambos para terceros/tercero.
	res := &resolverFake{registros: []map[string]any{
		{"Id": float64(7), "NombreCompleto": "Ana"},
	}}
	registry := NewResolverRegistry(map[string]Resolver{"tercero": res}, nil)

	rel := relacionesPrueba()[0] // Tercero (terceros/tercero), allowed_fields [Id, NombreCompleto]
	rel.NestedRelations = nil
	cfg := config.ServiceConfig{
		Service: "s", MaxDepth: 1, OnRelationError: config.OnRelationErrorNotify, MaxRelatedIDs: 100,
		Relations: []config.RelationConfig{rel},
	}

	blacklist := config.GlobalBlacklist{
		"terceros": {
			"tercero": {"NombreCompleto", "Id"},
		},
	}

	data := map[string]any{"Id": float64(1), "TerceroId": float64(7)}
	out, err := Expand(context.Background(), data, cfg, registry, ExpandOptions{Blacklist: blacklist})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	m := out.Data.(map[string]any)
	tercero, ok := m["Tercero"].(map[string]any)
	if !ok {
		t.Fatalf("Tercero debe ser un objeto; recibido: %T", m["Tercero"])
	}
	if _, existe := tercero["NombreCompleto"]; existe {
		t.Errorf("un campo permitido pero vetado debe omitirse; recibido: %+v", tercero)
	}
	if _, existe := tercero["Id"]; existe {
		t.Errorf("el Id implícito vetado debe omitirse; recibido: %+v", tercero)
	}
	// Supresión silenciosa: ningún aviso en MetaData.
	if len(out.Warnings) != 0 {
		t.Errorf("la supresión por lista negra no debe emitir avisos; avisos: %+v", out.Warnings)
	}
}

// Task 5.1 (complemento): un campo no vetado sobrevive al recorte, confirmando
// que la lista negra solo quita lo listado.
func TestExpandRecorteListaNegraConservaNoVetados(t *testing.T) {
	res := &resolverFake{registros: []map[string]any{
		{"Id": float64(7), "NombreCompleto": "Ana"},
	}}
	registry := NewResolverRegistry(map[string]Resolver{"tercero": res}, nil)

	rel := relacionesPrueba()[0]
	rel.NestedRelations = nil
	cfg := config.ServiceConfig{
		Service: "s", MaxDepth: 1, OnRelationError: config.OnRelationErrorNotify, MaxRelatedIDs: 100,
		Relations: []config.RelationConfig{rel},
	}

	// Vetar solo el Id; NombreCompleto debe sobrevivir.
	blacklist := config.GlobalBlacklist{"terceros": {"tercero": {"Id"}}}

	data := map[string]any{"Id": float64(1), "TerceroId": float64(7)}
	out, err := Expand(context.Background(), data, cfg, registry, ExpandOptions{Blacklist: blacklist})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	tercero := out.Data.(map[string]any)["Tercero"].(map[string]any)
	if _, existe := tercero["Id"]; existe {
		t.Errorf("el Id vetado debe omitirse; recibido: %+v", tercero)
	}
	if tercero["NombreCompleto"] != "Ana" {
		t.Errorf("un campo no vetado debe conservarse; recibido: %+v", tercero)
	}
}

// Task 5.2: el resolver genérico obedece la lista negra. Una relación resuelta
// por el genérico cuyo registro trae un campo vetado lo entrega recortado, y el
// aviso generic_resolver se conserva intacto.
func TestExpandRecorteListaNegraResolverGenerico(t *testing.T) {
	// El genérico entrega un registro con un campo que la lista negra veta.
	generico := &resolverFake{registros: []map[string]any{
		{"Id": float64(7), "NombreCompleto": "Ana", "CampoVetado": "secreto"},
	}, generico: true}
	// Sin resolver específico para "tercero": cae al genérico.
	registry := NewResolverRegistry(map[string]Resolver{}, generico)

	rel := relacionTerceroDispatch() // terceros/tercero
	cfg := cfgTerceroDispatch(rel)

	blacklist := config.GlobalBlacklist{"terceros": {"tercero": {"CampoVetado"}}}

	data := map[string]any{"Id": float64(1), "TerceroId": float64(7)}
	out, err := Expand(context.Background(), data, cfg, registry, ExpandOptions{Blacklist: blacklist})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	// El genérico fue consultado.
	if len(generico.llamadas) != 1 {
		t.Errorf("se esperaba 1 consulta al genérico; recibidas: %d", len(generico.llamadas))
	}
	// El campo vetado se recortó pese a venir del genérico.
	tercero := out.Data.(map[string]any)["Tercero"].(map[string]any)
	if _, existe := tercero["CampoVetado"]; existe {
		t.Errorf("el genérico debe obedecer la lista negra; recibido: %+v", tercero)
	}
	// Los campos no vetados sobreviven.
	if tercero["NombreCompleto"] != "Ana" {
		t.Errorf("un campo no vetado debe conservarse en la salida del genérico; recibido: %+v", tercero)
	}
	// El aviso generic_resolver se conserva intacto.
	if !contieneAviso(out.Warnings, KindGenericResolver) {
		t.Errorf("debe conservarse el aviso generic_resolver; avisos: %+v", out.Warnings)
	}
}

// Un campo vetado se omite por defecto cuando NO se solicita ningún fields
// (expansión completa): el recorte por lista negra se aplica siempre, sin
// depender de una selección. No hay error y no se emite ningún aviso; los campos
// no vetados sobreviven intactos. Complementa el caso de pedir el campo
// explícitamente (que se rechaza): aquí la omisión es silenciosa.
func TestExpandCampoVetadoSeOmitePorDefectoSinErrorNiAviso(t *testing.T) {
	// El resolver entrega un campo vetado y otros no vetados. No se pasan Fields
	// ni Seleccion: expansión por defecto de toda la relación.
	res := &resolverFake{registros: []map[string]any{
		{"Id": float64(7), "NombreCompleto": "Ana", "CampoVetado": "secreto"},
	}}
	registry := NewResolverRegistry(map[string]Resolver{"tercero": res}, nil)

	rel := relacionesPrueba()[0] // Tercero (terceros/tercero)
	rel.NestedRelations = nil
	cfg := config.ServiceConfig{
		Service: "s", MaxDepth: 1, OnRelationError: config.OnRelationErrorNotify, MaxRelatedIDs: 100,
		Relations: []config.RelationConfig{rel},
	}

	blacklist := config.GlobalBlacklist{"terceros": {"tercero": {"CampoVetado"}}}

	data := map[string]any{"Id": float64(1), "TerceroId": float64(7)}
	// ExpandOptions sin Fields ni Seleccion: no se solicita nada en particular.
	out, err := Expand(context.Background(), data, cfg, registry, ExpandOptions{Blacklist: blacklist})

	// No hay error por no solicitar el campo (ni por ningún otro motivo).
	if err != nil {
		t.Fatalf("no debe haber error en la expansión por defecto; err=%v", err)
	}

	tercero, ok := out.Data.(map[string]any)["Tercero"].(map[string]any)
	if !ok {
		t.Fatalf("Tercero debe ser un objeto; recibido: %T", out.Data.(map[string]any)["Tercero"])
	}
	// El campo vetado se omite aunque no se haya pedido (ni excluido) nada.
	if _, existe := tercero["CampoVetado"]; existe {
		t.Errorf("el campo vetado debe omitirse por defecto; recibido: %+v", tercero)
	}
	// Los campos no vetados se conservan: el default no recorta nada más.
	if tercero["NombreCompleto"] != "Ana" {
		t.Errorf("un campo no vetado debe conservarse por defecto; recibido: %+v", tercero)
	}
	if _, existe := tercero["Id"]; !existe {
		t.Errorf("el Id implícito (no vetado) debe conservarse por defecto; recibido: %+v", tercero)
	}
	// La omisión es silenciosa: ningún aviso en MetaData.
	if len(out.Warnings) != 0 {
		t.Errorf("omitir un campo vetado por defecto no debe emitir avisos; avisos: %+v", out.Warnings)
	}
}
