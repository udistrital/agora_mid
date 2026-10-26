package helper

import (
	"fmt"
	"sort"
	"strings"

	"github.com/udistrital/agora_mid/config"
)

// campoId es el nombre canónico del identificador de toda entidad. Es
// seleccionable y se conserva de forma implícita en la expansión aunque la
// configuración no lo declare en allowed_fields.
const campoId = "Id"

// NodoSeleccion representa un nodo del árbol de selección construido a partir de
// las rutas de fields. Cada nodo corresponde a un campo (base o relación) y, si
// es una relación, puede llevar hijos para los descendientes solicitados.
type NodoSeleccion struct {
	// Campos son los campos base solicitados directamente en este nivel que no
	// corresponden a una relación con descendientes seleccionados.
	Campos map[string]struct{}
	// Hijos mapea el nombre de cada relación seleccionada con su subárbol. La
	// presencia de una relación aquí implica que debe materializarse para poder
	// resolver sus descendientes.
	Hijos map[string]*NodoSeleccion
	// Completo indica que una ruta nombró esta relación como hoja (último
	// segmento), por lo que debe expandirse completa por configuración, ignorando
	// los Campos y Hijos acumulados por otras rutas (precedencia por relación de
	// la selección más amplia). No aplica al nodo raíz de la selección.
	Completo bool
}

func nuevoNodoSeleccion() *NodoSeleccion {
	return &NodoSeleccion{
		Campos: make(map[string]struct{}),
		Hijos:  make(map[string]*NodoSeleccion),
	}
}

// TieneCampo informa si el campo fue seleccionado directamente en este nivel.
func (n *NodoSeleccion) TieneCampo(campo string) bool {
	_, ok := n.Campos[campo]
	return ok
}

// Hijo devuelve el subárbol de la relación indicada y si existe.
func (n *NodoSeleccion) Hijo(nombre string) (*NodoSeleccion, bool) {
	h, ok := n.Hijos[nombre]
	return h, ok
}

// Vacio informa si el nodo no selecciona ningún campo ni relación.
func (n *NodoSeleccion) Vacio() bool {
	return len(n.Campos) == 0 && len(n.Hijos) == 0
}

// ParsearFields construye el árbol de selección a partir de las rutas de fields
// (por ejemplo "Id", "Tercero.Id", "Tercero.NombreCompleto"). Valida que cada
// ruta no esté vacía, no esté duplicada y que cada segmento esté permitido por la
// configuración jerárquica de allowed_fields y nested_relations.
//
// Al solicitar un descendiente (por ejemplo "Tercero.Id"), el objeto padre
// ("Tercero") se incluye automáticamente en el árbol para poder materializarlo.
//
// blacklist es la lista negra global de campos; puede ser nil. Pedir
// explícitamente un campo vetado produce el mismo resultado observable que pedir
// un campo no permitido (se rechaza con *ErrorFields y el mismo motivo), sin
// anunciar que el campo existe ni que está vetado.
//
// Devuelve un *ErrorFields cuando una ruta es vacía, duplicada o no permitida; el
// llamador debe mapearlo a 400 Bad Request sin iniciar la expansión.
func ParsearFields(fields []string, relaciones []config.RelationConfig, allowed []string, blacklist config.GlobalBlacklist) (*NodoSeleccion, error) {
	raiz := nuevoNodoSeleccion()
	vistas := make(map[string]struct{}, len(fields))

	for _, bruto := range fields {
		ruta := strings.TrimSpace(bruto)
		if ruta == "" {
			return nil, &ErrorFields{Path: bruto, Motivo: "la ruta de fields no puede estar vacía"}
		}
		if _, dup := vistas[ruta]; dup {
			return nil, &ErrorFields{Path: ruta, Motivo: "ruta de fields duplicada"}
		}
		vistas[ruta] = struct{}{}

		segmentos := strings.Split(ruta, ".")
		for i, seg := range segmentos {
			if strings.TrimSpace(seg) == "" {
				return nil, &ErrorFields{Path: ruta, Motivo: "la ruta de fields contiene un segmento vacío"}
			}
			segmentos[i] = strings.TrimSpace(seg)
		}

		// El nivel base no corresponde a ninguna relación resuelta: sus campos no
		// pasan por el choke-point de la lista negra. Se recorre con api/tabla
		// vacías; la lista negra solo aplica una vez que la ruta desciende a una
		// relación (que aporta su RelatedAPI/RelatedTable).
		if err := insertarRuta(raiz, ruta, segmentos, relaciones, allowed, blacklist, "", ""); err != nil {
			return nil, err
		}
	}

	return raiz, nil
}

// insertarRuta inserta una ruta en el árbol validando cada segmento contra las
// relaciones y los allowed_fields del nivel correspondiente. nivelAPI y
// nivelTabla identifican la relación cuyo nivel se está recorriendo (vacías en el
// nivel base) y se usan para emparejar un campo escalar terminal contra la lista
// negra.
func insertarRuta(
	nodo *NodoSeleccion,
	rutaCompleta string,
	segmentos []string,
	relaciones []config.RelationConfig,
	allowed []string,
	blacklist config.GlobalBlacklist,
	nivelAPI string,
	nivelTabla string,
) error {
	seg := segmentos[0]
	rel, esRelacion := buscarRelacion(relaciones, seg)

	// Último segmento de la ruta.
	if len(segmentos) == 1 {
		if esRelacion {
			// Seleccionar la relación completa; se materializa como hijo y se
			// marca Completo para que se expanda por configuración (precedencia
			// por relación: domina sobre cualquier descendiente del mismo nodo).
			hijo, ok := nodo.Hijos[seg]
			if !ok {
				hijo = nuevoNodoSeleccion()
				nodo.Hijos[seg] = hijo
			}
			hijo.Completo = true
			return nil
		}
		// Un campo vetado por la lista negra se rechaza igual que un campo no
		// permitido: mismo *ErrorFields y mismo motivo, sin distinguir el caso.
		if !campoPermitido(allowed, seg) || blacklist.CampoVetado(nivelAPI, nivelTabla, seg) {
			return &ErrorFields{Path: rutaCompleta, Motivo: fmt.Sprintf("el campo %q no está permitido", seg)}
		}
		nodo.Campos[seg] = struct{}{}
		return nil
	}

	// Segmento intermedio: debe ser una relación para poder descender.
	if !esRelacion {
		return &ErrorFields{
			Path:   rutaCompleta,
			Motivo: fmt.Sprintf("el segmento %q no es una relación y no admite descendientes", seg),
		}
	}

	// Incluir automáticamente el padre y descender al nivel de la relación, que
	// aporta su RelatedAPI/RelatedTable para emparejar la lista negra en los
	// segmentos siguientes.
	hijo, ok := nodo.Hijos[seg]
	if !ok {
		hijo = nuevoNodoSeleccion()
		nodo.Hijos[seg] = hijo
	}
	return insertarRuta(hijo, rutaCompleta, segmentos[1:], rel.NestedRelations, rel.AllowedFields, blacklist, rel.RelatedAPI, rel.RelatedTable)
}

// buscarRelacion devuelve la relación con el MappedName indicado y si existe.
func buscarRelacion(relaciones []config.RelationConfig, nombre string) (config.RelationConfig, bool) {
	for i := range relaciones {
		if relaciones[i].MappedName == nombre {
			return relaciones[i], true
		}
	}
	return config.RelationConfig{}, false
}

// campoPermitido informa si el campo está dentro de allowed. Un allowed nil
// significa que no hay lista de restricción declarada en ese nivel (los campos
// base provienen del esquema del recurso), por lo que se acepta cualquier campo.
// Un allowed declarado (no nil, incluido vacío) restringe estrictamente a sus
// elementos, con una excepción: el campo "Id" es seleccionable de forma implícita
// en cualquier nivel aunque no esté declarado en allowed_fields, de modo que toda
// entidad anidada sea identificable por defecto.
func campoPermitido(allowed []string, campo string) bool {
	if allowed == nil {
		return true
	}
	if campo == campoId {
		return true
	}
	for _, a := range allowed {
		if a == campo {
			return true
		}
	}
	return false
}

// ErrorFields representa un error de validación del parámetro fields. El llamador
// debe mapearlo a 400 Bad Request. Identifica la ruta inválida sin exponer datos
// sensibles.
type ErrorFields struct {
	Path   string
	Motivo string
}

func (e *ErrorFields) Error() string {
	return fmt.Sprintf("fields inválido en %q: %s", e.Path, e.Motivo)
}

// camposOrdenados devuelve los campos del nodo en orden determinista; útil para
// pruebas y mensajes estables.
func (n *NodoSeleccion) camposOrdenados() []string {
	salida := make([]string, 0, len(n.Campos))
	for c := range n.Campos {
		salida = append(salida, c)
	}
	sort.Strings(salida)
	return salida
}
