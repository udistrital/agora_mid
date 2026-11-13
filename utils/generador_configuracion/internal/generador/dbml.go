package generador

import (
	"fmt"
	"regexp"
	"strings"
)

// tiposAlmacenamiento traduce los tipos del DBML al tipo de almacenamiento del
// cargador de configuración. Un tipo fuera de este mapa es un error de parseo.
var tiposAlmacenamiento = map[string]string{
	"serial":    "integer",
	"integer":   "integer",
	"varchar":   "string",
	"text":      "string",
	"numeric":   "numeric",
	"boolean":   "boolean",
	"timestamp": "timestamp",
	"uuid":      "uuid",
	"jsonb":     "jsonb",
}

// Expresiones regulares que reconocen las construcciones del DBML que el parser
// soporta. Cada línea de un bloque se prueba contra ellas en el orden de dispatch.
var (
	// reTablePartial reconoce la apertura de una TablePartial: TablePartial nombre {.
	reTablePartial = regexp.MustCompile(`^TablePartial\s+(\w+)\s*\{$`)
	// reTable reconoce la apertura de una Table con esquema: Table esquema.tabla {.
	reTable = regexp.MustCompile(`^Table\s+(\w+)\.(\w+)\s*\{$`)
	// reTableGroup reconoce un TableGroup, que el parser omite por completo.
	reTableGroup = regexp.MustCompile(`^TableGroup\b.*\{$`)
	// reRefBloque reconoce la apertura de un bloque Ref, con o sin nombre.
	reRefBloque = regexp.MustCompile(`^[Rr]ef(?:\s+\w+)?\s*\{$`)
	// reIndexes reconoce el bloque indexes anidado en una Table, que se omite.
	reIndexes = regexp.MustCompile(`^indexes\s*\{$`)
	// reSpread reconoce la inclusión ~nombre de una TablePartial dentro de una Table.
	reSpread = regexp.MustCompile(`^~(\w+)$`)
	// reColumna reconoce una columna: nombre, tipo con parámetros opcionales y, opcionalmente,
	// el contenido de los atributos entre corchetes (grupo 3).
	reColumna = regexp.MustCompile(`^(\w+)\s+(\w+)(?:\([^)]*\))?(?:\s*\[(.*)\])?$`)
	// reRelacion reconoce una relación a.b.c < d.e.f, con ? opcional y operador <, > o -.
	reRelacion = regexp.MustCompile(`^([\w.]+)\s*\??([<>-])\s*([\w.]+)$`)
)

// itemDBML es una entrada de un bloque: una columna o una parte incluida con ~.
type itemDBML struct {
	columna  Columna
	spread   string
	esSpread bool
}

// bloqueDBML acumula las líneas de un bloque de nivel superior mientras se lee.
type bloqueDBML struct {
	tipo    string // "partial", "table", "ref" o "ignorado"
	esquema string
	nombre  string
	items   []itemDBML
	refs    []string
}

// ParsearDBML lee el contenido de un DBML y devuelve el modelo de tablas y
// referencias. Las TablePartial se expanden donde aparece ~nombre; los bloques
// indexes y TableGroup se omiten. Devuelve error ante sintaxis no soportada, tipos
// fuera de tiposAlmacenamiento, partes sin declarar, tablas duplicadas o
// referencias a columnas inexistentes.
func ParsearDBML(contenido string) (*Modelo, error) {
	m := &Modelo{}
	partes := map[string][]Columna{}
	var bloques []bloqueDBML
	var actual *bloqueDBML
	anidado := 0

	for i, crudo := range strings.Split(contenido, "\n") {
		linea := strings.TrimSpace(crudo)
		n := i + 1
		if linea == "" || strings.HasPrefix(linea, "//") {
			continue
		}
		if actual == nil {
			b, err := abrirBloque(linea, n)
			if err != nil {
				return nil, err
			}
			actual = b
			continue
		}
		if anidado == 0 && linea == "}" {
			bloques = append(bloques, *actual)
			actual = nil
			continue
		}
		if anidado > 0 {
			if linea == "}" {
				anidado--
			} else if strings.HasSuffix(linea, "{") {
				anidado++
			}
			continue
		}
		if actual.tipo == "ignorado" {
			continue
		}
		if err := agregarLinea(actual, linea, n, &anidado); err != nil {
			return nil, err
		}
	}
	if actual != nil {
		return nil, fmt.Errorf("bloque %q sin cerrar", actual.nombre)
	}

	for _, b := range bloques {
		switch b.tipo {
		case "partial":
			if _, dup := partes[b.nombre]; dup {
				return nil, fmt.Errorf("TablePartial %q duplicada", b.nombre)
			}
			for _, it := range b.items {
				if it.esSpread {
					return nil, fmt.Errorf("TablePartial %q no admite ~%s", b.nombre, it.spread)
				}
				partes[b.nombre] = append(partes[b.nombre], it.columna)
			}
		case "table":
			t, err := expandirTabla(b, partes)
			if err != nil {
				return nil, err
			}
			for _, previa := range m.Tablas {
				if previa.Esquema == t.Esquema && previa.Nombre == t.Nombre {
					return nil, fmt.Errorf("tabla %s.%s duplicada", t.Esquema, t.Nombre)
				}
			}
			m.Tablas = append(m.Tablas, t)
		case "ref":
			for _, linea := range b.refs {
				r, err := parsearRelacion(linea)
				if err != nil {
					return nil, err
				}
				m.Refs = append(m.Refs, r)
			}
		}
	}

	if err := validarRefs(m); err != nil {
		return nil, err
	}
	return m, nil
}

// abrirBloque interpreta una línea de nivel superior y devuelve el bloque que abre.
func abrirBloque(linea string, n int) (*bloqueDBML, error) {
	switch {
	case reTablePartial.MatchString(linea):
		return &bloqueDBML{tipo: "partial", nombre: reTablePartial.FindStringSubmatch(linea)[1]}, nil
	case reTable.MatchString(linea):
		g := reTable.FindStringSubmatch(linea)
		return &bloqueDBML{tipo: "table", esquema: g[1], nombre: g[2]}, nil
	case reTableGroup.MatchString(linea):
		return &bloqueDBML{tipo: "ignorado"}, nil
	case reRefBloque.MatchString(linea):
		return &bloqueDBML{tipo: "ref"}, nil
	case strings.HasPrefix(linea, "Ref") || strings.HasPrefix(linea, "ref"):
		return nil, fmt.Errorf("línea %d: forma de Ref no soportada: %q", n, linea)
	}
	return nil, fmt.Errorf("línea %d no reconocida: %q", n, linea)
}

// agregarLinea procesa una línea dentro de un bloque abierto. Actualiza anidado
// cuando la línea abre un bloque interno como indexes.
func agregarLinea(b *bloqueDBML, linea string, n int, anidado *int) error {
	if b.tipo == "ref" {
		b.refs = append(b.refs, linea)
		return nil
	}
	if strings.HasPrefix(linea, "note:") {
		return nil
	}
	if b.tipo == "table" && reIndexes.MatchString(linea) {
		*anidado = 1
		return nil
	}
	if g := reSpread.FindStringSubmatch(linea); g != nil {
		b.items = append(b.items, itemDBML{spread: g[1], esSpread: true})
		return nil
	}
	if g := reColumna.FindStringSubmatch(linea); g != nil {
		if _, ok := tiposAlmacenamiento[g[2]]; !ok {
			return fmt.Errorf("línea %d: tipo %q no soportado en %s", n, g[2], b.nombre)
		}
		b.items = append(b.items, itemDBML{columna: Columna{Nombre: g[1], Tipo: g[2], NoNula: esNoNula(g[3])}})
		return nil
	}
	return fmt.Errorf("línea %d no reconocida dentro de %s: %q", n, b.nombre, linea)
}

// expandirTabla construye la tabla reemplazando cada ~parte por sus columnas.
func expandirTabla(b bloqueDBML, partes map[string][]Columna) (Tabla, error) {
	t := Tabla{Esquema: b.esquema, Nombre: b.nombre}
	for _, it := range b.items {
		if !it.esSpread {
			t.Columnas = append(t.Columnas, it.columna)
			continue
		}
		cols, ok := partes[it.spread]
		if !ok {
			return Tabla{}, fmt.Errorf("%s.%s usa ~%s, que no es una TablePartial declarada", b.esquema, b.nombre, it.spread)
		}
		t.Columnas = append(t.Columnas, cols...)
	}
	return t, nil
}

// parsearRelacion interpreta `a.b.c < d.e.f` y las variantes `?<`, `>` y `-`.
// Para `<` el lado derecho porta la llave; para `>` el izquierdo; para `-` el
// derecho es el hijo de PK compartida.
func parsearRelacion(linea string) (Referencia, error) {
	g := reRelacion.FindStringSubmatch(linea)
	if g == nil {
		return Referencia{}, fmt.Errorf("relación no soportada: %q", linea)
	}
	izq, err := parsearColumnaRef(g[1])
	if err != nil {
		return Referencia{}, err
	}
	der, err := parsearColumnaRef(g[3])
	if err != nil {
		return Referencia{}, err
	}
	switch g[2] {
	case "<":
		return Referencia{Padre: izq, Hijo: der}, nil
	case ">":
		return Referencia{Padre: der, Hijo: izq}, nil
	default:
		return Referencia{Padre: izq, Hijo: der, Compartida: true}, nil
	}
}

// parsearColumnaRef separa esquema.tabla.columna.
func parsearColumnaRef(texto string) (ColumnaRef, error) {
	partes := strings.Split(texto, ".")
	if len(partes) != 3 {
		return ColumnaRef{}, fmt.Errorf("columna %q debe tener forma esquema.tabla.columna", texto)
	}
	return ColumnaRef{Esquema: partes[0], Tabla: partes[1], Columna: partes[2]}, nil
}

// validarRefs comprueba que cada columna de una referencia exista en su tabla.
func validarRefs(m *Modelo) error {
	columnas := map[string]bool{}
	for _, t := range m.Tablas {
		for _, c := range t.Columnas {
			columnas[t.Esquema+"."+t.Nombre+"."+c.Nombre] = true
		}
	}
	for _, r := range m.Refs {
		for _, c := range []ColumnaRef{r.Padre, r.Hijo} {
			if !columnas[c.Clave()] {
				return fmt.Errorf("referencia a %s, que no existe en el DBML", c.Clave())
			}
		}
	}
	return nil
}

// esNoNula indica si el contenido de los atributos de una columna declara que la
// columna no admite null: el atributo `not null` o la llave primaria (`pk` o
// `primary key`). Los atributos se comparan como tokens separados por coma, de modo
// que el texto de una nota no produce coincidencias falsas.
func esNoNula(atributos string) bool {
	for _, tok := range strings.Split(atributos, ",") {
		switch strings.TrimSpace(tok) {
		case "not null", "pk", "primary key":
			return true
		}
	}
	return false
}
