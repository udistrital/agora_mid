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

// Modos de propagación de la llave del padre al hijo durante la escritura en
// cascada.
const (
	// KeyPropagationFK propaga la llave del padre fijando una llave foránea en el
	// hijo (por ejemplo contacto.proveedor_id = proveedor.Id).
	KeyPropagationFK = "fk"
	// KeyPropagationSharedPK propaga la llave del padre como la propia llave
	// primaria del hijo (por ejemplo proveedor_natural.Id = proveedor.Id).
	KeyPropagationSharedPK = "shared_pk"
	// KeyPropagationFKInbound modela una FK ascendente: la entidad referenciada
	// es dueña de su propio Id y el nodo que la declara la referencia por una FK
	// local (por ejemplo proveedor.tercero_id = tercero.Id). El motor materializa
	// la entidad referenciada ANTES que el nodo y propaga su Id hacia arriba, a la
	// FK local del nodo (RelationConfig.LocalField).
	KeyPropagationFKInbound = "fk_inbound"
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
	// Write declara la configuración de escritura de la relación. Es opcional:
	// una relación sin bloque write solo participa en lectura.
	Write *WriteConfig `yaml:"write"`
}

// WriteVerbPolicy declara, por verbo HTTP, si la relación admite esa operación
// de escritura.
type WriteVerbPolicy struct {
	// Creatable habilita la creación de la entidad (POST/PATCH la crea si falta).
	Creatable bool `yaml:"creatable"`
	// Updatable habilita la actualización de la entidad existente.
	Updatable bool `yaml:"updatable"`
	// Deletable habilita la eliminación de la entidad en cascada de baja.
	Deletable bool `yaml:"deletable"`
}

// WriteConfig describe cómo se materializa una relación en escritura: qué campos
// se aceptan, cuáles son inmutables, qué clave natural identifica un registro
// existente para el upsert idempotente, cómo se propaga la llave del padre y qué
// verbos admite. Es el espejo de escritura del árbol de lectura y se valida
// contra él al arranque.
type WriteConfig struct {
	// WritableFields son los campos que el cliente puede enviar para esta
	// relación. Un campo fuera de esta lista se rechaza con 400. La lista vacía o
	// ausente indica que la relación no acepta campos propios.
	WritableFields []string `yaml:"writable_fields"`
	// ImmutableFields son campos que, una vez creado el registro, no pueden
	// modificarse; enviarlos en una actualización se rechaza con 400. Deben estar
	// contenidos en WritableFields.
	ImmutableFields []string `yaml:"immutable_fields"`
	// NaturalKey son los campos que identifican de forma única un registro
	// existente para el upsert idempotente (por ejemplo numero + tipo_documento_id
	// en datos_identificacion). Vacío indica que la idempotencia se apoya en el Id
	// del cuerpo. Deben estar contenidos en WritableFields.
	NaturalKey []string `yaml:"natural_key"`
	// KeyPropagation declara cómo se propaga la llave entre el nodo y la relación:
	// KeyPropagationFK (llave foránea del hijo), KeyPropagationSharedPK (PK
	// compartida) o KeyPropagationFKInbound (FK ascendente: la entidad referenciada
	// se materializa antes y su Id se propaga a la FK local del nodo).
	// Obligatorio para toda relación escribible que no sea la raíz.
	KeyPropagation string `yaml:"key_propagation"`
	// Policy declara los verbos admitidos por la relación.
	Policy WriteVerbPolicy `yaml:"policy"`
}

// CampoDerivado identifica de forma determinista el nombre del campo relacional
// derivado por la relación según su modo de propagación de llave, y si la
// relación tiene uno. El campo derivado es la columna que el motor de escritura
// completa o canoniza con la llave autoritativa del padre (o de la entidad
// referenciada en fk_inbound), aunque no pertenezca a writable_fields:
//
//   - fk:         related_field del hijo (la FK descendente, por ejemplo ProveedorId);
//   - shared_pk:  "Id" del detalle (la PK compartida con el padre);
//   - fk_inbound: local_field del nodo (la FK ascendente, por ejemplo TerceroId).
//
// Devuelve ("", false) para una relación sin bloque write, sin modo de
// propagación o con un modo no reconocido: en esos casos no hay campo derivado
// que reconocer ni comparar.
//
// Es la única fuente de verdad del nombre del campo derivado, compartida por la
// validación de configuración y por el motor de escritura, para que ambos
// coincidan sin reimplementar la regla.
func (rc RelationConfig) CampoDerivado() (string, bool) {
	if rc.Write == nil {
		return "", false
	}
	switch rc.Write.KeyPropagation {
	case KeyPropagationFK:
		return rc.RelatedField, rc.RelatedField != ""
	case KeyPropagationSharedPK:
		return "Id", true
	case KeyPropagationFKInbound:
		return rc.LocalField, rc.LocalField != ""
	default:
		return "", false
	}
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
	// FilterableFields contiene los nombres de columnas de primer nivel que el
	// servicio permite utilizar en filtros de consultas de colección.
	FilterableFields []string `yaml:"filterable_fields"`
	// Write declara la configuración de escritura de la entidad raíz del
	// servicio (por ejemplo proveedor). Como raíz, no propaga llave de un padre,
	// por lo que su KeyPropagation debe quedar vacío. Es opcional: un servicio
	// solo de lectura no lo declara.
	Write *WriteConfig `yaml:"write"`
}
