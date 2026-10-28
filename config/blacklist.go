package config

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// WildcardANY es la palabra reservada comodín de la lista negra global. Puede
// aparecer en cualquier nivel (related_api, related_table o field) y, como único
// elemento de la lista de campos, significa "todos los campos de esa tabla". Se
// escribe en mayúsculas para no colisionar con nombres de api/tabla (minúsculas)
// ni de campo (PascalCase).
const WildcardANY = "ANY"

// GlobalBlacklist es la lista negra global de campos suprimidos en la salida de
// expansión, cargada desde config/global_blacklist.yaml. Se estructura como un
// mapa jerárquico related_api -> related_table -> [field]. La palabra reservada
// WildcardANY actúa como comodín en cualquier nivel. Un valor nil o vacío
// significa "sin supresión".
//
// El mapa plano (en vez de una lista de mapas de una sola clave) deja que el
// propio YAML imponga la unicidad de clave por api/tabla: un bloque duplicado es
// un error de parseo del YAML.
type GlobalBlacklist map[string]map[string][]string

// archivoBlacklist es la raíz del documento YAML de la lista negra. El contenido
// efectivo cuelga de la clave global_blacklist; un archivo ausente o vacío deja
// este campo nil (sin supresión).
type archivoBlacklist struct {
	GlobalBlacklist GlobalBlacklist `yaml:"global_blacklist"`
}

// LoadBlacklist carga la lista negra global desde la ruta indicada. Un archivo
// ausente o vacío no es error: devuelve una lista negra vacía ("sin supresión").
// Un archivo presente pero con YAML inválido devuelve un error que identifica la
// ruta; el llamador debe abortar el arranque ante ese error.
func LoadBlacklist(ruta string) (GlobalBlacklist, error) {
	contenido, err := os.ReadFile(ruta)
	if err != nil {
		if os.IsNotExist(err) {
			// Archivo ausente: sin supresión.
			return GlobalBlacklist{}, nil
		}
		return nil, fmt.Errorf("no se pudo leer la lista negra %q: %w", ruta, err)
	}

	if strings.TrimSpace(string(contenido)) == "" {
		// Archivo vacío: sin supresión.
		return GlobalBlacklist{}, nil
	}

	var doc archivoBlacklist
	dec := yaml.NewDecoder(strings.NewReader(string(contenido)))
	dec.KnownFields(true)
	if err := dec.Decode(&doc); err != nil {
		// Un documento sin nodos efectivos (por ejemplo solo comentarios)
		// decodifica a io.EOF: equivale a "sin supresión", no es un error.
		if errors.Is(err, io.EOF) {
			return GlobalBlacklist{}, nil
		}
		return nil, fmt.Errorf("lista negra inválida %q: %w", ruta, err)
	}

	if doc.GlobalBlacklist == nil {
		// Documento sin entradas bajo global_blacklist: sin supresión.
		return GlobalBlacklist{}, nil
	}

	return doc.GlobalBlacklist, nil
}

// ModeloConocido es el conjunto de apis, tablas y campos derivados de las
// configuraciones de servicio cargadas. Se usa para validar que los componentes
// exactos (sin comodín) de la lista negra existen en el modelo.
type ModeloConocido struct {
	// apis son los related_api declarados.
	apis map[string]struct{}
	// tablas son los related_table declarados.
	tablas map[string]struct{}
	// campos son los nombres de campo conocidos: la unión de allowed_fields, el
	// Id implícito y los related_field de las relaciones.
	campos map[string]struct{}
}

// nuevoModeloConocido construye el modelo conocido recorriendo todas las
// relaciones (incluidas las anidadas) de las configuraciones de servicio. El Id
// implícito se agrega siempre al conjunto de campos conocidos.
func nuevoModeloConocido(configs map[string]ServiceConfig) ModeloConocido {
	m := ModeloConocido{
		apis:   make(map[string]struct{}),
		tablas: make(map[string]struct{}),
		campos: map[string]struct{}{"Id": {}},
	}
	for _, sc := range configs {
		for i := range sc.Relations {
			m.agregarRelacion(sc.Relations[i])
		}
	}
	return m
}

// agregarRelacion incorpora una relación y, recursivamente, sus anidadas al
// modelo conocido.
func (m ModeloConocido) agregarRelacion(rel RelationConfig) {
	m.apis[rel.RelatedAPI] = struct{}{}
	m.tablas[rel.RelatedTable] = struct{}{}
	if rel.RelatedField != "" {
		m.campos[rel.RelatedField] = struct{}{}
	}
	for _, campo := range rel.AllowedFields {
		m.campos[campo] = struct{}{}
	}
	for i := range rel.NestedRelations {
		m.agregarRelacion(rel.NestedRelations[i])
	}
}

// ValidarBlacklist valida la lista negra contra el modelo conocido. Las reglas:
//   - Un componente exacto (api, tabla o campo distinto de WildcardANY) debe
//     existir en el modelo conocido; si no, la entrada es inválida.
//   - WildcardANY en cualquier nivel no se verifica contra el modelo (un comodín
//     es intencionalmente amplio).
//   - La entrada ANY/ANY/ANY (comodín en los tres niveles) se rechaza.
//
// Devuelve un error que identifica la entrada inconsistente; el llamador debe
// abortar el arranque ante ese error.
func ValidarBlacklist(bl GlobalBlacklist, configs map[string]ServiceConfig) error {
	modelo := nuevoModeloConocido(configs)

	for api, tablas := range bl {
		apiANY := api == WildcardANY
		if !apiANY {
			if _, ok := modelo.apis[api]; !ok {
				return fmt.Errorf(
					"lista negra inválida: related_api %q no existe en el modelo conocido",
					api,
				)
			}
		}
		for tabla, campos := range tablas {
			tablaANY := tabla == WildcardANY
			if !tablaANY {
				if _, ok := modelo.tablas[tabla]; !ok {
					return fmt.Errorf(
						"lista negra inválida: related_table %q (bajo related_api %q) no existe en el modelo conocido",
						tabla, api,
					)
				}
			}
			for _, campo := range campos {
				campoANY := campo == WildcardANY
				if apiANY && tablaANY && campoANY {
					return fmt.Errorf(
						"lista negra inválida: la entrada %s/%s/%s (comodín en los tres niveles) está prohibida",
						WildcardANY, WildcardANY, WildcardANY,
					)
				}
				if campoANY {
					continue
				}
				if _, ok := modelo.campos[campo]; !ok {
					return fmt.Errorf(
						"lista negra inválida: el campo %q (bajo related_api %q, related_table %q) no existe en el modelo conocido",
						campo, api, tabla,
					)
				}
			}
		}
	}

	return nil
}

// CampoVetado informa si el campo está vetado por la lista negra para la
// relación identificada por (relatedAPI, relatedTable). El emparejamiento
// considera la palabra reservada WildcardANY como comodín en cualquiera de los
// tres niveles:
//   - La api se empareja con una entrada exacta o con WildcardANY.
//   - La tabla se empareja con una entrada exacta o con WildcardANY bajo esa api.
//   - El campo está vetado si la lista de campos lo contiene explícitamente o
//     contiene WildcardANY (que significa "todos los campos de esa tabla").
//
// Una lista negra nil o vacía no veta ningún campo.
func (bl GlobalBlacklist) CampoVetado(relatedAPI, relatedTable, campo string) bool {
	if len(bl) == 0 {
		return false
	}
	for _, apiClave := range []string{relatedAPI, WildcardANY} {
		tablas, ok := bl[apiClave]
		if !ok {
			continue
		}
		for _, tablaClave := range []string{relatedTable, WildcardANY} {
			campos, ok := tablas[tablaClave]
			if !ok {
				continue
			}
			for _, c := range campos {
				if c == WildcardANY || c == campo {
					return true
				}
			}
		}
	}
	return false
}

// MustLoadBlacklist carga y valida la lista negra global contra las
// configuraciones de servicio y aborta el arranque del proceso si el archivo es
// inválido o contiene una entrada inconsistente, imprimiendo el mensaje que
// identifica la entrada. Un archivo ausente o vacío no aborta: equivale a "sin
// supresión".
func MustLoadBlacklist(ruta string, configs map[string]ServiceConfig) GlobalBlacklist {
	bl, err := LoadBlacklist(ruta)
	if err != nil {
		log.Fatalf("no se pudo cargar la lista negra global: %v", err)
	}
	if err := ValidarBlacklist(bl, configs); err != nil {
		log.Fatalf("no se pudo validar la lista negra global: %v", err)
	}
	return bl
}
