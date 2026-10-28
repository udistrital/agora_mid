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
		rels, []string{"Id"}, nil,
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
		rels, nil, nil,
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
	_, err := ParsearFields([]string{"Id", ""}, rels, []string{"Id"}, nil)
	if err == nil {
		t.Fatal("se esperaba error por ruta vacía")
	}
	if _, ok := err.(*ErrorFields); !ok {
		t.Errorf("se esperaba *ErrorFields; recibido: %T", err)
	}
}

func TestParsearFieldsRutaDuplicada(t *testing.T) {
	rels := relacionesPrueba()
	_, err := ParsearFields([]string{"Tercero.Id", "Tercero.Id"}, rels, []string{"Id"}, nil)
	if err == nil {
		t.Fatal("se esperaba error por ruta duplicada")
	}
}

func TestParsearFieldsCampoNoPermitido(t *testing.T) {
	rels := relacionesPrueba()
	_, err := ParsearFields([]string{"Tercero.Secreto"}, rels, []string{"Id"}, nil)
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
	_, err := ParsearFields([]string{"Id.Algo"}, rels, []string{"Id"}, nil)
	if err == nil {
		t.Fatal("se esperaba error: un campo base no admite descendientes")
	}
}

func TestParsearFieldsRelacionCompleta(t *testing.T) {
	rels := relacionesPrueba()
	arbol, err := ParsearFields([]string{"Tercero"}, rels, nil, nil)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if _, ok := arbol.Hijo("Tercero"); !ok {
		t.Error("seleccionar una relación completa debe materializarla")
	}
}

// Id implícito en selección (grupo 3, task 3.1): la ruta del Id de una relación
// es válida aunque allowed_fields de esa relación no liste Id; un campo no-Id
// fuera de allowed_fields sigue rechazándose.
func TestParsearFieldsIdImplicito(t *testing.T) {
	// Relación Tercero con allowed_fields que NO incluye Id.
	rels := []config.RelationConfig{
		{
			Name: "Tercero", RelatedService: "tercero", RelatedAPI: "terceros", RelatedTable: "tercero",
			Direction: config.DirectionLocalToRelated, LocalField: "TerceroId", RelatedField: "Id",
			Cardinality: config.CardinalityOne, Depth: 2, MappedName: "Tercero",
			AllowedFields: []string{"NombreCompleto"},
		},
	}

	// Tercero.Id se acepta pese a que allowed_fields no lista Id.
	arbol, err := ParsearFields([]string{"Tercero.Id"}, rels, nil, nil)
	if err != nil {
		t.Fatalf("Tercero.Id debe aceptarse por Id implícito; err=%v", err)
	}
	hijo, ok := arbol.Hijo("Tercero")
	if !ok || !hijo.TieneCampo("Id") {
		t.Errorf("el árbol debe contener Tercero.Id; recibido: %+v", hijo)
	}

	// Un campo no-Id fuera de allowed_fields sigue rechazándose.
	_, err = ParsearFields([]string{"Tercero.Secreto"}, rels, nil, nil)
	if err == nil {
		t.Fatal("un campo no-Id fuera de allowed_fields debe rechazarse")
	}
	if _, ok := err.(*ErrorFields); !ok {
		t.Errorf("se esperaba *ErrorFields; recibido: %T", err)
	}
}

// Task 5.3: pedir explícitamente en fields un campo vetado por la lista negra
// produce el mismo resultado observable que pedir un campo no permitido: se
// rechaza con *ErrorFields y el mismo motivo, sin distinguir el caso ni emitir
// aviso.
func TestParsearFieldsCampoVetadoSeRechazaComoNoPermitido(t *testing.T) {
	rels := relacionesPrueba() // Tercero -> terceros/tercero, allowed_fields [Id, NombreCompleto]

	// NombreCompleto está permitido por allowed_fields, pero la lista negra lo veta
	// para terceros/tercero.
	blacklist := config.GlobalBlacklist{"terceros": {"tercero": {"NombreCompleto"}}}

	_, errVetado := ParsearFields([]string{"Tercero.NombreCompleto"}, rels, nil, blacklist)
	if errVetado == nil {
		t.Fatal("un campo vetado pedido en fields debe rechazarse")
	}
	efVetado, ok := errVetado.(*ErrorFields)
	if !ok {
		t.Fatalf("se esperaba *ErrorFields para un campo vetado; recibido: %T", errVetado)
	}

	// Un campo no permitido (fuera de allowed_fields) produce el mismo tipo y el
	// mismo motivo observable. Se usa el MISMO nombre de campo vetado para que, si
	// ambos caminos comparten plantilla de motivo, las cadenas sean idénticas: así
	// el motivo no revela si el campo existe pero está vetado o si no está
	// permitido. "NombreCompleto" sí está en allowed_fields, por lo que sin lista
	// negra se acepta; para forzar el rechazo "no permitido" sobre el mismo nombre
	// se usa un allowed vacío (ninguna lista negra).
	_, errNoPermitido := ParsearFields([]string{"Tercero.NombreCompleto"}, relsSinPermiso(), nil, nil)
	efNoPermitido, ok := errNoPermitido.(*ErrorFields)
	if !ok {
		t.Fatalf("se esperaba *ErrorFields para un campo no permitido; recibido: %T", errNoPermitido)
	}

	// El motivo no debe distinguir el caso vetado del no permitido (política
	// anti-sondeo): para el mismo nombre de campo, ambos producen el motivo
	// idéntico.
	if efVetado.Motivo != efNoPermitido.Motivo {
		t.Errorf(
			"el motivo de un campo vetado debe ser idéntico al de uno no permitido; vetado=%q no_permitido=%q",
			efVetado.Motivo, efNoPermitido.Motivo,
		)
	}
}

// relsSinPermiso devuelve la fixture con Tercero restringido a allowed_fields que
// NO incluye NombreCompleto, de modo que pedir Tercero.NombreCompleto se rechace
// por no permitido (sin lista negra). Se usa para comparar el motivo de ese
// rechazo con el de un campo vetado del mismo nombre.
func relsSinPermiso() []config.RelationConfig {
	rels := relacionesPrueba()
	rels[0].AllowedFields = []string{"Id"}
	return rels
}
