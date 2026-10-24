package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func urlsPrueba() ServiceURLs {
	return ServiceURLs{
		"terceros":              "http://terceros.test",
		"parametros":            "http://parametros.test",
		"gestion_personas_crud": "http://gestion.test",
	}
}

const relacionesValidas = `service: registro_persona
max_depth: 2
on_relation_error: notify
max_related_ids: 100
relations:
  - name: Tercero
    related_service: tercero
    related_api: terceros
    related_table: tercero
    direction: local_to_related
    local_field: TerceroId
    related_field: Id
    cardinality: one
    depth: 1
    mapped_name: Tercero
    endpoint: /v1/tercero
    query_template: "Id:{ids}"
    allowed_fields:
      - Id
      - NombreCompleto
    nested_relations:
      - name: InfoComplementariaTercero
        related_service: tercero
        related_api: terceros
        related_table: info_complementaria
        direction: local_to_related
        local_field: Id
        related_field: TerceroId
        cardinality: many
        depth: 1
        mapped_name: InfoComplementariaTercero
        endpoint: /v1/info_complementaria
        query_template: "TerceroId:{ids}"
        allowed_fields:
          - Id
          - Descripcion
  - name: TipoDocumento
    related_service: parametro
    related_api: parametros
    related_table: tipo_documento
    direction: related_to_local
    local_field: TipoDocumentoId
    related_field: Id
    cardinality: one
    depth: 1
    mapped_name: TipoDocumento
    endpoint: /v1/tipo_documento
    query_template: "Id:{ids}"
`

func escribirRelaciones(t *testing.T, servicio, contenido string) string {
	t.Helper()
	dir := t.TempDir()
	ruta := filepath.Join(dir, "relaciones_"+servicio+".yaml")
	if err := os.WriteFile(ruta, []byte(contenido), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

// Un archivo válido se carga correctamente, incluyendo relaciones anidadas, y
// related_api resuelve contra las URLs.
func TestLoadRelacionesArchivoValido(t *testing.T) {
	dir := escribirRelaciones(t, "registro_persona", relacionesValidas)

	configs, err := LoadRelaciones(dir, urlsPrueba())
	if err != nil {
		t.Fatalf("error inesperado al cargar configuración válida: %v", err)
	}

	sc, ok := configs["registro_persona"]
	if !ok {
		t.Fatalf("no se cargó el servicio registro_persona; configs=%+v", configs)
	}
	if sc.MaxDepth != 2 || sc.OnRelationError != OnRelationErrorNotify || sc.MaxRelatedIDs != 100 {
		t.Errorf("campos de servicio inesperados: %+v", sc)
	}
	if len(sc.Relations) != 2 {
		t.Fatalf("se esperaban 2 relaciones; recibidas: %d", len(sc.Relations))
	}
	tercero := sc.Relations[0]
	if tercero.Name != "Tercero" || tercero.RelatedAPI != "terceros" || tercero.RelatedService != "tercero" || tercero.Cardinality != CardinalityOne {
		t.Errorf("relación Tercero inesperada: %+v", tercero)
	}
	if len(tercero.NestedRelations) != 1 || tercero.NestedRelations[0].Name != "InfoComplementariaTercero" {
		t.Errorf("relación anidada inesperada: %+v", tercero.NestedRelations)
	}
}

// Las configuraciones inválidas abortan con un mensaje que identifica el
// archivo y, cuando aplica, la relación.
func TestLoadRelacionesInvalidas(t *testing.T) {
	base := func(mut func(string) string) string {
		return mut(relacionesValidas)
	}

	casos := []struct {
		nombre       string
		servicio     string
		contenido    string
		debeContener []string
	}{
		{
			nombre:   "max_depth no positivo",
			servicio: "svc",
			contenido: base(func(s string) string {
				return strings.Replace(s, "max_depth: 2", "max_depth: 0", 1)
			}),
			debeContener: []string{"relaciones_svc.yaml", "max_depth"},
		},
		{
			nombre:   "on_relation_error inválido",
			servicio: "svc",
			contenido: base(func(s string) string {
				return strings.Replace(s, "on_relation_error: notify", "on_relation_error: quizas", 1)
			}),
			debeContener: []string{"relaciones_svc.yaml", "on_relation_error"},
		},
		{
			nombre:   "direction inválida",
			servicio: "svc",
			contenido: base(func(s string) string {
				return strings.Replace(s, "direction: local_to_related", "direction: hacia_un_lado", 1)
			}),
			debeContener: []string{"relaciones_svc.yaml", "Tercero", "direction"},
		},
		{
			nombre:   "cardinality inválida",
			servicio: "svc",
			contenido: base(func(s string) string {
				return strings.Replace(s, "cardinality: one", "cardinality: ninguna", 1)
			}),
			debeContener: []string{"relaciones_svc.yaml", "cardinality"},
		},
		{
			nombre:   "related_api inexistente",
			servicio: "svc",
			contenido: base(func(s string) string {
				return strings.Replace(s, "related_api: terceros", "related_api: inexistente", 1)
			}),
			debeContener: []string{"relaciones_svc.yaml", "related_api", "inexistente"},
		},
		{
			nombre:   "related_api vacío",
			servicio: "svc",
			contenido: base(func(s string) string {
				return strings.Replace(s, "related_api: terceros", `related_api: ""`, 1)
			}),
			debeContener: []string{"relaciones_svc.yaml", "related_api"},
		},
		{
			nombre:   "mapped_name duplicado",
			servicio: "svc",
			contenido: base(func(s string) string {
				return strings.Replace(s, "mapped_name: TipoDocumento", "mapped_name: Tercero", 1)
			}),
			debeContener: []string{"relaciones_svc.yaml", "mapped_name", "Tercero"},
		},
		{
			nombre:   "name vacío",
			servicio: "svc",
			contenido: base(func(s string) string {
				return strings.Replace(s, "name: Tercero", `name: ""`, 1)
			}),
			debeContener: []string{"relaciones_svc.yaml", "name"},
		},
		{
			nombre:       "yaml inválido",
			servicio:     "svc",
			contenido:    "service: svc\n  : : :\n",
			debeContener: []string{"relaciones_svc.yaml"},
		},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			dir := escribirRelaciones(t, c.servicio, c.contenido)
			_, err := LoadRelaciones(dir, urlsPrueba())
			if err == nil {
				t.Fatalf("se esperaba error de validación")
			}
			msg := err.Error()
			for _, frag := range c.debeContener {
				if !strings.Contains(msg, frag) {
					t.Errorf("el mensaje de error debe contener %q; mensaje: %s", frag, msg)
				}
			}
		})
	}
}

// Un directorio sin archivos de relaciones devuelve un mapa vacío sin error.
func TestLoadRelacionesSinArchivos(t *testing.T) {
	dir := t.TempDir()
	configs, err := LoadRelaciones(dir, urlsPrueba())
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if len(configs) != 0 {
		t.Errorf("se esperaba mapa vacío; recibido: %+v", configs)
	}
}

// relacionesConWrite declara un servicio coherente que incluye bloques write en
// la raíz y en una relación hija, para ejercitar la validación de coherencia
// lectura/escritura del loader.
const relacionesConWrite = `service: proveedor
max_depth: 2
on_relation_error: notify
max_related_ids: 100
write:
  writable_fields:
    - TerceroId
    - TipoRegistroId
  immutable_fields:
    - TerceroId
  natural_key:
    - TerceroId
  policy:
    creatable: true
    updatable: true
    deletable: false
relations:
  - name: Tercero
    related_service: tercero
    related_api: terceros
    related_table: tercero
    direction: local_to_related
    local_field: TerceroId
    related_field: Id
    cardinality: one
    depth: 1
    mapped_name: Tercero
    endpoint: /v1/tercero
    query_template: "Id:{ids}"
    allowed_fields:
      - Id
      - NombreCompleto
    write:
      writable_fields:
        - NombreCompleto
        - TipoContribuyenteId
      immutable_fields: []
      natural_key: []
      key_propagation: shared_pk
      policy:
        creatable: true
        updatable: true
        deletable: false
`

// Un archivo con bloques write coherentes (raíz sin key_propagation, hija con
// shared_pk, inmutables y clave natural contenidos en writable_fields) se carga
// sin error y conserva la configuración de escritura.
func TestLoadRelacionesWriteCoherente(t *testing.T) {
	dir := escribirRelaciones(t, "proveedor", relacionesConWrite)

	configs, err := LoadRelaciones(dir, urlsPrueba())
	if err != nil {
		t.Fatalf("configuración write coherente debe cargar sin error: %v", err)
	}
	sc := configs["proveedor"]
	if sc.Write == nil || sc.Write.KeyPropagation != "" {
		t.Errorf("la raíz debe tener write sin key_propagation; write=%+v", sc.Write)
	}
	if len(sc.Relations) == 0 || sc.Relations[0].Write == nil {
		t.Fatalf("la relación Tercero debe conservar su bloque write")
	}
	if sc.Relations[0].Write.KeyPropagation != KeyPropagationSharedPK {
		t.Errorf("Tercero debe propagar por shared_pk; recibido %q", sc.Relations[0].Write.KeyPropagation)
	}
}

// Configuraciones write incoherentes abortan con un mensaje que identifica el
// archivo y la relación afectada.
func TestLoadRelacionesWriteIncoherente(t *testing.T) {
	casos := []struct {
		nombre       string
		contenido    string
		debeContener []string
	}{
		{
			nombre: "raíz con key_propagation",
			contenido: strings.Replace(
				relacionesConWrite,
				"max_related_ids: 100\nwrite:\n  writable_fields:",
				"max_related_ids: 100\nwrite:\n  key_propagation: fk\n  writable_fields:",
				1,
			),
			debeContener: []string{"relaciones_proveedor.yaml", "write", "key_propagation"},
		},
		{
			nombre: "hija sin key_propagation",
			contenido: strings.Replace(
				relacionesConWrite,
				"      key_propagation: shared_pk\n",
				"",
				1,
			),
			debeContener: []string{"relaciones_proveedor.yaml", "Tercero", "key_propagation"},
		},
		{
			nombre: "hija con key_propagation inválido",
			contenido: strings.Replace(
				relacionesConWrite,
				"key_propagation: shared_pk",
				"key_propagation: magia",
				1,
			),
			debeContener: []string{"relaciones_proveedor.yaml", "Tercero", "key_propagation"},
		},
		{
			nombre: "inmutable fuera de writable_fields",
			contenido: strings.Replace(
				relacionesConWrite,
				"  immutable_fields:\n    - TerceroId",
				"  immutable_fields:\n    - CampoFantasma",
				1,
			),
			debeContener: []string{"relaciones_proveedor.yaml", "immutable_fields", "CampoFantasma"},
		},
		{
			nombre: "natural_key fuera de writable_fields",
			contenido: strings.Replace(
				relacionesConWrite,
				"  natural_key:\n    - TerceroId",
				"  natural_key:\n    - CampoFantasma",
				1,
			),
			debeContener: []string{"relaciones_proveedor.yaml", "natural_key", "CampoFantasma"},
		},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			if c.contenido == relacionesConWrite {
				t.Fatalf("el caso %q no modificó la configuración base", c.nombre)
			}
			dir := escribirRelaciones(t, "proveedor", c.contenido)
			_, err := LoadRelaciones(dir, urlsPrueba())
			if err == nil {
				t.Fatalf("se esperaba error de validación write")
			}
			msg := err.Error()
			for _, frag := range c.debeContener {
				if !strings.Contains(msg, frag) {
					t.Errorf("el mensaje debe contener %q; mensaje: %s", frag, msg)
				}
			}
		})
	}
}
