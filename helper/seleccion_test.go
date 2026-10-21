package helper

import (
	"testing"

	"github.com/udistrital/agora_mid/config"
)

// relacionesPrueba construye un conjunto de relaciones jerárquicas para pruebas
// de selección y expansión: Tercero (one) con hija InfoComplementariaTercero
// (many), y TipoDocumento (one).
func relacionesPrueba() []config.RelationConfig {
	return []config.RelationConfig{
		{
			Name:           "Tercero",
			RelatedService: "tercero",
			RelatedAPI:     "terceros",
			RelatedTable:   "tercero",
			Direction:      config.DirectionLocalToRelated,
			LocalField:     "TerceroId",
			RelatedField:   "Id",
			Cardinality:    config.CardinalityOne,
			Depth:          2,
			MappedName:     "Tercero",
			Endpoint:       "/v1/tercero",
			QueryTemplate:  "Id:{ids}",
			AllowedFields:  []string{"Id", "NombreCompleto"},
			NestedRelations: []config.RelationConfig{
				{
					Name:           "InfoComplementariaTercero",
					RelatedService: "tercero",
					RelatedAPI:     "terceros",
					RelatedTable:   "info_complementaria",
					Direction:      config.DirectionRelatedToLocal,
					LocalField:     "Id",
					RelatedField:   "TerceroId",
					Cardinality:    config.CardinalityMany,
					Depth:          1,
					MappedName:     "InfoComplementariaTercero",
					Endpoint:       "/v1/info_complementaria",
					QueryTemplate:  "TerceroId:{ids}",
					AllowedFields:  []string{"Id", "Descripcion"},
				},
			},
		},
		{
			Name:           "TipoDocumento",
			RelatedService: "parametro",
			RelatedAPI:     "parametros",
			RelatedTable:   "tipo_documento",
			Direction:      config.DirectionLocalToRelated,
			LocalField:     "TipoDocumentoId",
			RelatedField:   "Id",
			Cardinality:    config.CardinalityOne,
			Depth:          1,
			MappedName:     "TipoDocumento",
			Endpoint:       "/v1/tipo_documento",
			QueryTemplate:  "Id:{ids}",
			AllowedFields:  []string{"Id", "Nombre"},
		},
	}
}

func TestParsearFieldsBaseYAnidado(t *testing.T) {
	rels := relacionesPrueba()
	arbol, err := ParsearFields(
		[]string{"Id", "Tercero.Id", "Tercero.NombreCompleto"},
		rels, []string{"Id"},
	)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if !arbol.TieneCampo("Id") {
		t.Error("se esperaba el campo base Id")
	}
	hijoTercero, ok := arbol.Hijo("Tercero")
	if !ok {
		t.Fatal("se esperaba la relación Tercero en el árbol")
	}
	if !hijoTercero.TieneCampo("Id") || !hijoTercero.TieneCampo("NombreCompleto") {
		t.Errorf("Tercero debe tener Id y NombreCompleto; campos: %v", hijoTercero.camposOrdenados())
	}
}

// Solicitar un descendiente incluye automáticamente el padre.
func TestParsearFieldsInclusionAutomaticaDePadres(t *testing.T) {
	rels := relacionesPrueba()
	arbol, err := ParsearFields(
		[]string{"Tercero.InfoComplementariaTercero.Descripcion"},
		rels, nil,
	)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	tercero, ok := arbol.Hijo("Tercero")
	if !ok {
		t.Fatal("se esperaba el padre Tercero incluido automáticamente")
	}
	info, ok := tercero.Hijo("InfoComplementariaTercero")
	if !ok {
		t.Fatal("se esperaba el padre InfoComplementariaTercero incluido automáticamente")
	}
	if !info.TieneCampo("Descripcion") {
		t.Error("se esperaba el campo Descripcion en InfoComplementariaTercero")
	}
}

func TestParsearFieldsRutaVacia(t *testing.T) {
	rels := relacionesPrueba()
	_, err := ParsearFields([]string{"Id", ""}, rels, []string{"Id"})
	if err == nil {
		t.Fatal("se esperaba error por ruta vacía")
	}
	if _, ok := err.(*ErrorFields); !ok {
		t.Errorf("se esperaba *ErrorFields; recibido: %T", err)
	}
}

func TestParsearFieldsRutaDuplicada(t *testing.T) {
	rels := relacionesPrueba()
	_, err := ParsearFields([]string{"Tercero.Id", "Tercero.Id"}, rels, []string{"Id"})
	if err == nil {
		t.Fatal("se esperaba error por ruta duplicada")
	}
}

func TestParsearFieldsCampoNoPermitido(t *testing.T) {
	rels := relacionesPrueba()
	_, err := ParsearFields([]string{"Tercero.Secreto"}, rels, []string{"Id"})
	if err == nil {
		t.Fatal("se esperaba error por campo no permitido")
	}
	ef, ok := err.(*ErrorFields)
	if !ok {
		t.Fatalf("se esperaba *ErrorFields; recibido: %T", err)
	}
	if ef.Path != "Tercero.Secreto" {
		t.Errorf("el error debe identificar la ruta inválida; path: %q", ef.Path)
	}
}

func TestParsearFieldsSegmentoIntermedioNoRelacion(t *testing.T) {
	rels := relacionesPrueba()
	// "Id" es campo base, no relación: no admite descendientes.
	_, err := ParsearFields([]string{"Id.Algo"}, rels, []string{"Id"})
	if err == nil {
		t.Fatal("se esperaba error: un campo base no admite descendientes")
	}
}

func TestParsearFieldsRelacionCompleta(t *testing.T) {
	rels := relacionesPrueba()
	arbol, err := ParsearFields([]string{"Tercero"}, rels, nil)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if _, ok := arbol.Hijo("Tercero"); !ok {
		t.Error("seleccionar una relación completa debe materializarla")
	}
}
