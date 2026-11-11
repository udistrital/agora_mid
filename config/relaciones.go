package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

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
	// Referencias son las referencias de objetos direccionables del servicio. El
	// cargador de composición las resuelve y las añade a Relations al arranque.
	Referencias []ReferenciaObjeto `yaml:"referencias"`
	// Write declara la configuración de escritura de la entidad raíz del
	// servicio (por ejemplo proveedor). Como raíz, no propaga llave de un padre,
	// por lo que su KeyPropagation debe quedar vacío. Es opcional: un servicio
	// solo de lectura no lo declara. Cuando Raiz está declarada, los campos
	// escribibles, inmutables y la clave natural provienen del objeto raíz, y
	// Write solo puede declarar la política.
	Write *WriteConfig `yaml:"write"`
	// Raiz nombra el objeto direccionable que representa la entidad raíz del
	// servicio. Su tabla de campos alimenta los campos de Write al arranque.
	Raiz string `yaml:"raiz"`
}

// Arista describe la semántica de una relación entre un referenciador (un servicio
// o un objeto) y un objeto referenciado. Vive en la referencia, no en la
// definición del objeto, de modo que un mismo objeto puede participar con aristas
// distintas.
//
//   - Direction, Cardinality, Depth y MappedName: igual que en RelationConfig.
//   - LocalField y RelatedField: campos del referenciador y del objeto que unen la
//     relación. Son obligatorios para que la llave derivada sea determinista.
//   - KeyPropagation: modo de propagación de la llave (fk, shared_pk o
//     fk_inbound). Es obligatorio cuando Policy admite algún verbo y puede quedar
//     vacío en una referencia de solo lectura.
//   - Policy: verbos de escritura admitidos por la arista.
type Arista struct {
	Direction      string          `yaml:"direction"`
	Cardinality    string          `yaml:"cardinality"`
	LocalField     string          `yaml:"local_field"`
	RelatedField   string          `yaml:"related_field"`
	KeyPropagation string          `yaml:"key_propagation"`
	Policy         WriteVerbPolicy `yaml:"policy"`
	Depth          int             `yaml:"depth"`
	MappedName     string          `yaml:"mapped_name"`
}

// ReferenciaObjeto es una referencia desde un servicio o desde otro objeto hacia un
// objeto direccionable. Objeto es el nombre del objeto referenciado. La arista se
// declara en línea junto a él.
type ReferenciaObjeto struct {
	Objeto string `yaml:"objeto"`
	Arista `yaml:",inline"`
}

// ObjetoDireccionable es una definición de entidad con nombre único, reutilizable
// por varios servicios. Name identifica el objeto en el catálogo, Campos es su
// tabla de campos tipada y Referencias son sus relaciones hacia otros objetos.
type ObjetoDireccionable struct {
	Name        string             `yaml:"name"`
	Campos      TablaCampos        `yaml:"campos"`
	Referencias []ReferenciaObjeto `yaml:"referencias"`
	// RelatedService es el nombre lógico del servicio/recurso del objeto. Es la
	// llave con la que el registro related_service→resolver localiza el resolver del
	// nodo compuesto; debe coincidir con esa llave y, por convención, es snake_case.
	// Es opcional: si se omite, el cargador la deriva del nombre (PascalCase a
	// snake_case). Se declara explícitamente cuando la llave lógica no coincide con
	// esa derivación (p. ej. TipoDocumento cuyo servicio lógico es "parametro").
	RelatedService string `yaml:"related_service"`
	// RelatedAPI, RelatedTable y Endpoint ubican el CRUD que almacena el objeto. Son
	// propiedades del objeto, no de la arista, y el cargador los exige al arranque.
	RelatedAPI   string `yaml:"related_api"`
	RelatedTable string `yaml:"related_table"`
	Endpoint     string `yaml:"endpoint"`
	// NaturalKey son los campos que identifican un registro existente para el upsert
	// idempotente. Deben estar entre los campos con rol writable.
	NaturalKey []string `yaml:"natural_key"`
}

// NodoArbol es un nodo del árbol de composición: un objeto con la arista de la
// referencia que lo introduce. Campos es una copia de la tabla de campos del
// objeto, de modo que mutar un nodo no altera la definición compartida ni otros
// nodos.
type NodoArbol struct {
	Objeto string
	Arista Arista
	Campos TablaCampos
}

// CatalogoObjetos guarda las definiciones de objeto validadas, indexadas por nombre.
// Se construye con NuevoCatalogoObjetos. Su valor cero no contiene objetos.
type CatalogoObjetos struct {
	objetos map[string]ObjetoDireccionable
}

// DecodificarObjetoDireccionable interpreta una definición de objeto en YAML y la
// valida. Las claves desconocidas abortan la decodificación.
func DecodificarObjetoDireccionable(contenido []byte) (ObjetoDireccionable, error) {
	var obj ObjetoDireccionable
	if err := decodificarEstricto(contenido, &obj); err != nil {
		return ObjetoDireccionable{}, fmt.Errorf("objeto direccionable inválido: %w", err)
	}
	if err := obj.Validar(); err != nil {
		return ObjetoDireccionable{}, err
	}
	return obj, nil
}

// DecodificarReferenciasServicio interpreta la lista de referencias de un servicio
// en YAML y valida cada arista. El nombre del servicio identifica el origen en los
// errores. Un contenido vacío produce una lista vacía.
func DecodificarReferenciasServicio(servicio string, contenido []byte) ([]ReferenciaObjeto, error) {
	var refs []ReferenciaObjeto
	if err := decodificarEstricto(contenido, &refs); err != nil {
		return nil, fmt.Errorf("referencias inválidas del servicio %q: %w", servicio, err)
	}
	if err := validarMappedNamesUnicos(refs); err != nil {
		return nil, fmt.Errorf("servicio %q: %w", servicio, err)
	}
	for _, ref := range refs {
		if err := ref.Validar(); err != nil {
			return nil, fmt.Errorf("servicio %q: %w", servicio, err)
		}
	}
	return refs, nil
}

// Validar comprueba el nombre, la tabla de campos y las referencias del objeto. Un
// error nombra el objeto afectado.
func (o ObjetoDireccionable) Validar() error {
	if strings.TrimSpace(o.Name) == "" {
		return fmt.Errorf("objeto direccionable sin name: el nombre es obligatorio")
	}
	if err := o.Campos.Validar(o.Name); err != nil {
		return err
	}
	if err := validarMappedNamesUnicos(o.Referencias); err != nil {
		return fmt.Errorf("objeto %q: %w", o.Name, err)
	}
	for _, ref := range o.Referencias {
		if err := ref.Validar(); err != nil {
			return fmt.Errorf("objeto %q: %w", o.Name, err)
		}
	}
	return nil
}

// Validar comprueba que la arista sea coherente: dirección, cardinalidad y
// profundidad válidas, mapped_name y campos de unión no vacíos, y una propagación
// de llave reconocida. Una arista con verbos de escritura admitidos exige
// propagación de llave. Un error nombra el objeto referenciado.
func (r ReferenciaObjeto) Validar() error {
	fallo := func(detalle string) error {
		return fmt.Errorf("referencia al objeto %q: %s", r.Objeto, detalle)
	}

	if strings.TrimSpace(r.Objeto) == "" {
		return fallo("el campo objeto es obligatorio")
	}
	if r.Direction != DirectionLocalToRelated && r.Direction != DirectionRelatedToLocal {
		return fallo(fmt.Sprintf("direction debe ser %q o %q", DirectionLocalToRelated, DirectionRelatedToLocal))
	}
	if r.Cardinality != CardinalityOne && r.Cardinality != CardinalityMany {
		return fallo(fmt.Sprintf("cardinality debe ser %q o %q", CardinalityOne, CardinalityMany))
	}
	if r.Depth <= 0 {
		return fallo("depth debe ser positivo")
	}
	if strings.TrimSpace(r.MappedName) == "" {
		return fallo("mapped_name es obligatorio")
	}
	if strings.TrimSpace(r.LocalField) == "" {
		return fallo("local_field es obligatorio")
	}
	if strings.TrimSpace(r.RelatedField) == "" {
		return fallo("related_field es obligatorio")
	}

	prop := strings.TrimSpace(r.KeyPropagation)
	if prop != "" && !propagacionLlaveValida(prop) {
		return fallo(fmt.Sprintf(
			"key_propagation debe ser %q, %q, %q o vacío, no %q",
			KeyPropagationFK, KeyPropagationSharedPK, KeyPropagationFKInbound, prop,
		))
	}
	if prop == "" && r.Policy.algunoAdmitido() {
		return fallo("policy admite escritura pero key_propagation está vacío")
	}
	return nil
}

// NuevoCatalogoObjetos valida cada definición y construye el catálogo. Un nombre
// duplicado o una definición inválida aborta, nombrando el objeto afectado. El
// catálogo guarda copias de las definiciones, independientes de la entrada.
func NuevoCatalogoObjetos(objetos []ObjetoDireccionable) (CatalogoObjetos, error) {
	porNombre := make(map[string]ObjetoDireccionable, len(objetos))
	for _, obj := range objetos {
		if err := obj.Validar(); err != nil {
			return CatalogoObjetos{}, err
		}
		if _, dup := porNombre[obj.Name]; dup {
			return CatalogoObjetos{}, fmt.Errorf("nombre de objeto %q duplicado", obj.Name)
		}
		porNombre[obj.Name] = copiarObjeto(obj)
	}
	return CatalogoObjetos{objetos: porNombre}, nil
}

// Objeto devuelve una copia de la definición con el nombre indicado y true si
// existe. Si no existe devuelve la definición vacía y false.
func (c CatalogoObjetos) Objeto(nombre string) (ObjetoDireccionable, bool) {
	obj, ok := c.objetos[nombre]
	if !ok {
		return ObjetoDireccionable{}, false
	}
	return copiarObjeto(obj), true
}

// Nodo valida la referencia y construye el nodo del árbol para el objeto
// referenciado. El nodo lleva una copia de la arista de la referencia y una copia de
// la tabla de campos del objeto. Devuelve error si la referencia es inválida o si el
// objeto no está en el catálogo, nombrando el objeto.
func (c CatalogoObjetos) Nodo(ref ReferenciaObjeto) (NodoArbol, error) {
	if err := ref.Validar(); err != nil {
		return NodoArbol{}, err
	}
	obj, ok := c.objetos[ref.Objeto]
	if !ok {
		return NodoArbol{}, fmt.Errorf("referencia al objeto %q: el objeto no está definido", ref.Objeto)
	}
	return NodoArbol{
		Objeto: ref.Objeto,
		Arista: ref.Arista,
		Campos: append(TablaCampos(nil), obj.Campos...),
	}, nil
}

// decodificarEstricto decodifica YAML en destino con claves desconocidas
// rechazadas. Un contenido vacío no es error: deja destino en su valor cero.
func decodificarEstricto(contenido []byte, destino any) error {
	dec := yaml.NewDecoder(bytes.NewReader(contenido))
	dec.KnownFields(true)
	if err := dec.Decode(destino); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}
	return nil
}

// propagacionLlaveValida indica si el modo pertenece a los tres modos de propagación
// reconocidos.
func propagacionLlaveValida(modo string) bool {
	switch modo {
	case KeyPropagationFK, KeyPropagationSharedPK, KeyPropagationFKInbound:
		return true
	}
	return false
}

// algunoAdmitido indica si la política admite al menos un verbo de escritura.
func (p WriteVerbPolicy) algunoAdmitido() bool {
	return p.Creatable || p.Updatable || p.Deletable
}

// validarMappedNamesUnicos comprueba que los mapped_name de las referencias
// hermanas no se repitan.
func validarMappedNamesUnicos(refs []ReferenciaObjeto) error {
	vistos := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		if _, dup := vistos[ref.MappedName]; dup {
			return fmt.Errorf("mapped_name %q duplicado entre referencias", ref.MappedName)
		}
		vistos[ref.MappedName] = struct{}{}
	}
	return nil
}

// copiarObjeto devuelve una copia de la definición con sus slices duplicados, de
// modo que la copia no comparte memoria con el original.
func copiarObjeto(o ObjetoDireccionable) ObjetoDireccionable {
	c := o
	c.Campos = append(TablaCampos(nil), o.Campos...)
	c.Referencias = append([]ReferenciaObjeto(nil), o.Referencias...)
	return c
}

// Nombres devuelve los nombres de los objetos del catálogo en orden alfabético.
func (c CatalogoObjetos) Nombres() []string {
	nombres := make([]string, 0, len(c.objetos))
	for nombre := range c.objetos {
		nombres = append(nombres, nombre)
	}
	sort.Strings(nombres)
	return nombres
}
