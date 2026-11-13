// Package generador produce la configuración de relaciones del middleware a partir
// del modelo DBML y de los sidecars de intención. Lo usa el ejecutable
// cmd/generador de forma manual; el API nunca importa este paquete.
package generador

import (
	"fmt"
	"os"
	"path/filepath"
)

// Generar ejecuta la cadena completa: lee el DBML en rutaDBML, carga y valida los
// sidecars de carpetaSidecars, construye los objetos y los archivos de servicio, y
// los escribe en salida. Antes de escribir elimina la salida generada anterior; si la
// construcción falla, la salida previa no se toca.
func Generar(rutaDBML, carpetaSidecars, salida string) error {
	contenido, err := os.ReadFile(rutaDBML)
	if err != nil {
		return fmt.Errorf("leyendo el DBML: %w", err)
	}
	m, err := ParsearDBML(string(contenido))
	if err != nil {
		return fmt.Errorf("parseando %s: %w", filepath.Base(rutaDBML), err)
	}
	s, err := CargarSidecars(carpetaSidecars, m)
	if err != nil {
		return err
	}
	objetos, servicios, err := construirTodo(m, s)
	if err != nil {
		return err
	}
	if err := limpiarSalida(salida); err != nil {
		return err
	}
	if err := EscribirObjetos(objetos, salida); err != nil {
		return err
	}
	return EscribirServicios(servicios, salida)
}

// Modelo es el resultado de leer el DBML: tablas en orden de declaración y
// referencias de llave foránea.
type Modelo struct {
	Tablas []Tabla
	Refs   []Referencia
}

// Tabla es una tabla del DBML con sus columnas ya expandidas (sin ~partes).
type Tabla struct {
	Esquema  string
	Nombre   string
	Columnas []Columna
}

// Columna es una columna del DBML con su tipo sin parámetros (p. ej. "varchar").
// NoNula es verdadera cuando la columna declara `not null` o es llave primaria, y en
// ese caso el campo generado no admite null.
type Columna struct {
	Nombre string
	Tipo   string
	NoNula bool
}

// ColumnaRef identifica una columna como esquema.tabla.columna.
type ColumnaRef struct {
	Esquema string
	Tabla   string
	Columna string
}

// Clave devuelve la columna en la forma esquema.tabla.columna usada por los sidecars.
func (c ColumnaRef) Clave() string {
	return c.Esquema + "." + c.Tabla + "." + c.Columna
}

// Referencia une una columna que porta la llave (Hijo) con la columna que
// referencia (Padre). Compartida indica una relación 1-1 de PK compartida (`-`).
type Referencia struct {
	Padre      ColumnaRef
	Hijo       ColumnaRef
	Compartida bool
}

// ObjetoConfig es un objeto direccionable generado, con las claves del cargador.
type ObjetoConfig struct {
	Name         string         `yaml:"name"`
	RelatedAPI   string         `yaml:"related_api"`
	RelatedTable string         `yaml:"related_table"`
	Endpoint     string         `yaml:"endpoint"`
	NaturalKey   []string       `yaml:"natural_key,omitempty"`
	Campos       []CampoConfig  `yaml:"campos"`
	Referencias  []AristaConfig `yaml:"referencias"`
}

// CampoConfig es una fila de la tabla de campos tipada de un objeto generado.
// Nullable indica si el campo admite null; se deriva de la ausencia de `not null` en
// la columna del DBML y es independiente de Required, que el generador no deriva.
type CampoConfig struct {
	Name      string `yaml:"name"`
	Type      string `yaml:"type"`
	Nullable  bool   `yaml:"nullable"`
	Readable  bool   `yaml:"readable"`
	Writable  bool   `yaml:"writable"`
	Immutable bool   `yaml:"immutable"`
	Required  bool   `yaml:"required"`
}

// AristaConfig es una referencia generada desde un objeto hacia otro.
type AristaConfig struct {
	Objeto         string         `yaml:"objeto"`
	Direction      string         `yaml:"direction"`
	Cardinality    string         `yaml:"cardinality"`
	LocalField     string         `yaml:"local_field"`
	RelatedField   string         `yaml:"related_field"`
	KeyPropagation string         `yaml:"key_propagation"`
	Policy         PoliticaVerbos `yaml:"policy"`
	Depth          int            `yaml:"depth"`
	MappedName     string         `yaml:"mapped_name"`
	Flatten        bool           `yaml:"flatten,omitempty"`
}

// PoliticaVerbos son los verbos de escritura admitidos por una arista u objeto.
type PoliticaVerbos struct {
	Creatable bool `yaml:"creatable"`
	Updatable bool `yaml:"updatable"`
	Deletable bool `yaml:"deletable"`
}

// Sidecars contiene la intención leída de los archivos de la carpeta.
type Sidecars struct {
	Expand     []EntradaSidecar
	FKInbound  []EntradaSidecar
	Blacklist  []string
	Write      map[string]PoliticaEscritura
	Service    map[string]ServicioSidecar
	Flatten    []string
	FieldRoles map[string]RolCampo
	Lookup     []EntradaSidecar
}

// RolCampo son los roles de un campo declarados en el sidecar field_roles. Un campo
// sin entrada toma readable y writable en true y immutable en false.
type RolCampo struct {
	Readable  bool `yaml:"readable"`
	Writable  bool `yaml:"writable"`
	Immutable bool `yaml:"immutable"`
}

// ServicioSidecar es una entrada del sidecar service, direccionada por la tabla raíz.
type ServicioSidecar struct {
	Service          string   `yaml:"service"`
	MaxDepth         int      `yaml:"max_depth"`
	OnRelationError  string   `yaml:"on_relation_error"`
	MaxRelatedIDs    int      `yaml:"max_related_ids"`
	FilterableFields []string `yaml:"filterable_fields"`
}

// PoliticaEscritura es una entrada del sidecar write para un objeto.
type PoliticaEscritura struct {
	Creatable  bool     `yaml:"creatable"`
	Updatable  bool     `yaml:"updatable"`
	Deletable  bool     `yaml:"deletable"`
	NaturalKey []string `yaml:"natural_key"`
}

// ServicioConfig es el archivo de servicio relaciones_<servicio>.yaml con las claves
// que el cargador consume. RelatedAPI y RelatedTable ubican el archivo y no se escriben.
type ServicioConfig struct {
	Service          string             `yaml:"service"`
	Raiz             string             `yaml:"raiz"`
	MaxDepth         int                `yaml:"max_depth"`
	OnRelationError  string             `yaml:"on_relation_error"`
	MaxRelatedIDs    int                `yaml:"max_related_ids"`
	FilterableFields []string           `yaml:"filterable_fields"`
	Write            *EscrituraServicio `yaml:"write,omitempty"`
	Referencias      []AristaConfig     `yaml:"referencias"`
	RelatedAPI       string             `yaml:"-"`
	RelatedTable     string             `yaml:"-"`
}

// EscrituraServicio es el bloque write de un archivo de servicio: solo la política de
// la raíz.
type EscrituraServicio struct {
	Policy PoliticaVerbos `yaml:"policy"`
}
