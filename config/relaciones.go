package config

// Direcciones y cardinalidades válidas para una relación declarada.
const (
	DirectionLocalToRelated = "local_to_related"
	DirectionRelatedToLocal = "related_to_local"

	CardinalityOne  = "one"
	CardinalityMany = "many"

	OnRelationErrorNotify = "notify"
	OnRelationErrorFail   = "fail"
)

// RelationConfig describe una relación declarada hacia otro CRUD. Las relaciones
// anidadas se declaran jerárquicamente en NestedRelations y cada una lleva su
// propio AllowedFields y NestedRelations.
//
// RelatedService es el nombre lógico del recurso/servicio relacionado dentro del
// dominio y no resuelve URLs; sirve para nombrar la relación en avisos y lógica
// interna. RelatedAPI identifica el servicio externo (la API que resuelve la URL
// base contra conf/app.conf): terceros, parametros o gestion_personas_crud.
type RelationConfig struct {
	Name            string           `yaml:"name"`
	RelatedService  string           `yaml:"related_service"`
	RelatedAPI      string           `yaml:"related_api"`
	RelatedTable    string           `yaml:"related_table"`
	Direction       string           `yaml:"direction"`
	LocalField      string           `yaml:"local_field"`
	RelatedField    string           `yaml:"related_field"`
	Cardinality     string           `yaml:"cardinality"`
	Depth           int              `yaml:"depth"`
	MappedName      string           `yaml:"mapped_name"`
	Endpoint        string           `yaml:"endpoint"`
	QueryTemplate   string           `yaml:"query_template"`
	AllowedFields   []string         `yaml:"allowed_fields"`
	NestedRelations []RelationConfig `yaml:"nested_relations"`
}

// ServiceConfig es la configuración de relaciones de un servicio, cargada desde
// config/relaciones_<servicio>.yaml. BaseURL se resuelve al cargar contra las
// URLs de conf/app.conf y no se declara en el archivo YAML.
type ServiceConfig struct {
	Service         string           `yaml:"service"`
	MaxDepth        int              `yaml:"max_depth"`
	OnRelationError string           `yaml:"on_relation_error"`
	MaxRelatedIDs   int              `yaml:"max_related_ids"`
	Relations       []RelationConfig `yaml:"relations"`
}
