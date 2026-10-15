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
    endpoint: /v1/terceros
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
