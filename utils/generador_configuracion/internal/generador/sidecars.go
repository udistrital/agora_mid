package generador

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Nombres de los sidecars dentro de la carpeta de entradas.
const (
	archivoExpand     = "expand.yaml"
	archivoFKInbound  = "fk_inbound.yaml"
	archivoBlacklist  = "blacklist.yaml"
	archivoWrite      = "write.yaml"
	archivoService    = "service.yaml"
	archivoFlatten    = "flatten.yaml"
	archivoFieldRoles = "field_roles.yaml"
	archivoLookup     = "lookup.yaml"
)

// comodinANY es el segmento que coincide con cualquier esquema, tabla o columna en las
// claves de field_roles.
const comodinANY = "ANY"

// profundidadPorOmision es la profundidad de una arista cuando la entrada no la fija.
const profundidadPorOmision = 1

// EntradaSidecar es una entrada de expand, fk_inbound o lookup: una FK direccionada por
// su columna portadora con la profundidad de su arista y, opcionalmente, el nombre de
// la arista. Se lee como cadena simple (profundidad por omisión) o como mapa con
// entrada, depth y mapped_name opcionales.
type EntradaSidecar struct {
	Clave      string
	Depth      int
	MappedName string
}

// UnmarshalYAML decodifica una cadena simple o un mapa con entrada, depth y
// mapped_name. Rechaza claves desconocidas, falta de entrada, depth que no sea un
// entero positivo y mapped_name que no sea una cadena no vacía; cada error nombra la
// entrada cuando puede identificarla.
func (e *EntradaSidecar) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind == yaml.ScalarNode {
		e.Clave = n.Value
		e.Depth = profundidadPorOmision
		return nil
	}
	if n.Kind != yaml.MappingNode {
		return fmt.Errorf("línea %d: la entrada debe ser una cadena o un mapa", n.Line)
	}
	var campos map[string]any
	if err := n.Decode(&campos); err != nil {
		return err
	}
	clave, _ := campos["entrada"].(string)
	for k := range campos {
		if k != "entrada" && k != "depth" && k != "mapped_name" {
			return fmt.Errorf("entrada %q: clave desconocida %q", clave, k)
		}
	}
	if clave == "" {
		return errors.New("entrada de mapa sin campo entrada")
	}
	e.Clave = clave
	e.Depth = profundidadPorOmision
	if valor, ok := campos["depth"]; ok {
		profundidad, ok := valor.(int)
		if !ok || profundidad < 1 {
			return fmt.Errorf("entrada %q: depth debe ser un entero positivo", clave)
		}
		e.Depth = profundidad
	}
	if valor, ok := campos["mapped_name"]; ok {
		nombre, ok := valor.(string)
		if !ok || strings.TrimSpace(nombre) == "" {
			return fmt.Errorf("entrada %q: mapped_name debe ser una cadena no vacía", clave)
		}
		e.MappedName = nombre
	}
	return nil
}

// CargarSidecars lee los sidecars de dir y valida cada entrada contra el modelo.
// Un archivo ausente o vacío equivale a una lista vacía. Las claves desconocidas en
// un sidecar son error. Cada error nombra la entrada responsable.
func CargarSidecars(dir string, m *Modelo) (*Sidecars, error) {
	s := &Sidecars{Write: map[string]PoliticaEscritura{}, Service: map[string]ServicioSidecar{}}
	var err error
	if err = decodificarSidecar(dir, archivoExpand, &s.Expand); err != nil {
		return nil, err
	}
	if err = decodificarSidecar(dir, archivoFKInbound, &s.FKInbound); err != nil {
		return nil, err
	}
	if err = decodificarSidecar(dir, archivoBlacklist, &s.Blacklist); err != nil {
		return nil, err
	}
	if err = decodificarSidecar(dir, archivoWrite, &s.Write); err != nil {
		return nil, err
	}
	if err = decodificarSidecar(dir, archivoService, &s.Service); err != nil {
		return nil, err
	}
	if err = decodificarSidecar(dir, archivoFlatten, &s.Flatten); err != nil {
		return nil, err
	}
	if err = decodificarSidecar(dir, archivoFieldRoles, &s.FieldRoles); err != nil {
		return nil, err
	}
	if err = decodificarSidecar(dir, archivoLookup, &s.Lookup); err != nil {
		return nil, err
	}
	if err := validarSidecars(s, m); err != nil {
		return nil, err
	}
	return s, nil
}

// decodificarSidecar lee dir/nombre en destino, rechazando claves desconocidas.
// Un archivo inexistente o sin contenido deja destino sin cambios.
func decodificarSidecar(dir, nombre string, destino any) error {
	f, err := os.Open(filepath.Join(dir, nombre))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("sidecar %s: %w", nombre, err)
	}
	defer f.Close()
	dec := yaml.NewDecoder(f)
	dec.KnownFields(true)
	if err := dec.Decode(destino); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("sidecar %s: %w", nombre, err)
	}
	return nil
}

// validarSidecars comprueba que cada entrada de los sidecars resuelva contra el DBML.
func validarSidecars(s *Sidecars, m *Modelo) error {
	fks := map[string]bool{}
	for _, r := range m.Refs {
		fks[r.Hijo.Clave()] = true
	}
	for _, nombre := range []struct {
		archivo  string
		entradas []EntradaSidecar
	}{{archivoExpand, s.Expand}, {archivoFKInbound, s.FKInbound}, {archivoLookup, s.Lookup}} {
		for _, e := range nombre.entradas {
			if !fks[e.Clave] {
				return fmt.Errorf("sidecar %s: la entrada %q no corresponde a una FK declarada en el DBML", nombre.archivo, e.Clave)
			}
		}
	}

	otros := map[string]string{}
	for _, e := range s.Expand {
		otros[e.Clave] = archivoExpand
	}
	for _, e := range s.FKInbound {
		otros[e.Clave] = archivoFKInbound
	}
	for _, e := range s.Flatten {
		otros[e] = archivoFlatten
	}
	for _, e := range s.Blacklist {
		otros[e] = archivoBlacklist
	}
	for _, e := range s.Lookup {
		if archivo, ok := otros[e.Clave]; ok {
			return fmt.Errorf("sidecar %s: la entrada %q también está en %s", archivoLookup, e.Clave, archivo)
		}
	}

	compartidas := map[string]bool{}
	for _, r := range m.Refs {
		if r.Compartida {
			compartidas[r.Hijo.Clave()] = true
		}
	}
	inbound := conjunto(entradasClaves(s.FKInbound))
	for _, e := range s.Flatten {
		if !compartidas[e] {
			return fmt.Errorf("sidecar %s: la entrada %q no es una columna de PK compartida declarada en el DBML", archivoFlatten, e)
		}
		if inbound[e] {
			return fmt.Errorf("sidecar %s: la entrada %q también está en %s", archivoFlatten, e, archivoFKInbound)
		}
	}

	columnas := map[string]bool{}
	tablas := map[string]Tabla{}
	for _, t := range m.Tablas {
		tablas[t.Esquema+"."+t.Nombre] = t
		for _, c := range t.Columnas {
			columnas[t.Esquema+"."+t.Nombre+"."+c.Nombre] = true
		}
	}
	for _, e := range s.Blacklist {
		if !columnas[e] {
			return fmt.Errorf("sidecar %s: la entrada %q no existe en el DBML", archivoBlacklist, e)
		}
	}

	for _, k := range clavesOrdenadas(s.Write) {
		t, ok := tablas[k]
		if !ok {
			return fmt.Errorf("sidecar %s: la tabla %q no existe en el DBML", archivoWrite, k)
		}
		for _, col := range s.Write[k].NaturalKey {
			if !columnas[k+"."+col] {
				return fmt.Errorf("sidecar %s: natural_key %q de %s no existe en la tabla %s", archivoWrite, col, k, t.Nombre)
			}
		}
	}

	for _, k := range clavesOrdenadas(s.Service) {
		if _, ok := tablas[k]; !ok {
			return fmt.Errorf("sidecar %s: la entrada %q no es una tabla del DBML", archivoService, k)
		}
		servicio := s.Service[k]
		if servicio.Service == "" {
			return fmt.Errorf("sidecar %s: la entrada %q no declara service", archivoService, k)
		}
		for _, f := range servicio.FilterableFields {
			if !columnas[k+"."+f] {
				return fmt.Errorf("sidecar %s: filterable_fields %q de %s no es columna de la tabla raíz", archivoService, f, k)
			}
		}
	}

	for _, k := range clavesOrdenadas(s.FieldRoles) {
		partes := strings.Split(k, ".")
		if len(partes) != 3 {
			return fmt.Errorf("sidecar %s: la entrada %q debe tener la forma esquema.tabla.columna", archivoFieldRoles, k)
		}
		rol := s.FieldRoles[k]
		if rol.Immutable && !rol.Writable {
			return fmt.Errorf("sidecar %s: la entrada %q declara immutable: true con writable: false", archivoFieldRoles, k)
		}
		if partes[2] == comodinANY {
			return fmt.Errorf("sidecar %s: la entrada %q usa ANY en el segmento de columna; ANY solo aplica al esquema y a la tabla", archivoFieldRoles, k)
		}
		if !coincideAlgunaColumna(partes, m) {
			return fmt.Errorf("sidecar %s: la entrada %q no coincide con ninguna columna del DBML", archivoFieldRoles, k)
		}
	}
	return nil
}

// coincideAlgunaColumna indica si alguna columna del modelo coincide con la clave
// partida en esquema, tabla y columna, donde el segmento ANY coincide con cualquier
// valor.
func coincideAlgunaColumna(partes []string, m *Modelo) bool {
	for _, t := range m.Tablas {
		for _, c := range t.Columnas {
			if coincide(partes, t.Esquema, t.Nombre, c.Nombre) {
				return true
			}
		}
	}
	return false
}

// coincide indica si la clave partida coincide con esquema.tabla.columna. El segmento
// ANY coincide con cualquier valor.
func coincide(partes []string, esquema, tabla, columna string) bool {
	valores := []string{esquema, tabla, columna}
	for i, p := range partes {
		if p != comodinANY && p != valores[i] {
			return false
		}
	}
	return true
}

// clavesOrdenadas devuelve las claves de un mapa en orden lexicográfico, para que los
// errores y la salida no dependan del orden de iteración.
func clavesOrdenadas[V any](mapa map[string]V) []string {
	claves := make([]string, 0, len(mapa))
	for k := range mapa {
		claves = append(claves, k)
	}
	sort.Strings(claves)
	return claves
}

// entradasClaves devuelve la clave de cada entrada de un sidecar de FK.
func entradasClaves(entradas []EntradaSidecar) []string {
	claves := make([]string, 0, len(entradas))
	for _, e := range entradas {
		claves = append(claves, e.Clave)
	}
	return claves
}
