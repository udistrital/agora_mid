package config

import (
	"os"
	"reflect"
	"testing"
)

// Árbol inline de referencia del servicio proveedor, previo a la composición por
// objetos direccionables. Es la base con la que se compara el árbol compuesto.
const archivoBaseInlineProveedor = "testdata/relaciones_proveedor_inline.yaml"

// cargarBaseInlineProveedor carga el árbol inline de referencia como único archivo
// de relaciones y devuelve el servicio proveedor.
func cargarBaseInlineProveedor(t *testing.T) ServiceConfig {
	t.Helper()
	contenido, err := os.ReadFile(archivoBaseInlineProveedor)
	if err != nil {
		t.Fatalf("no se pudo leer la base inline del proveedor: %v", err)
	}
	comp, err := escribirComposicion(t, map[string]string{
		"relaciones_proveedor.yaml": string(contenido),
	})
	if err != nil {
		t.Fatalf("la base inline del proveedor debe cargar sin error: %v", err)
	}
	return comp.Servicios["proveedor"]
}

// listaNoNula devuelve el slice recibido o una lista vacía cuando es nulo, para
// que nil y lista vacía se comparen como iguales.
func listaNoNula(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// normalizarWrite copia un bloque write con sus listas no nulas.
func normalizarWrite(w *WriteConfig) *WriteConfig {
	if w == nil {
		return nil
	}
	c := *w
	c.WritableFields = listaNoNula(w.WritableFields)
	c.ImmutableFields = listaNoNula(w.ImmutableFields)
	c.NaturalKey = listaNoNula(w.NaturalKey)
	return &c
}

// normalizarRelaciones copia recursivamente las relaciones con sus listas no nulas.
func normalizarRelaciones(rels []RelationConfig) []RelationConfig {
	out := make([]RelationConfig, 0, len(rels))
	for _, r := range rels {
		r.AllowedFields = listaNoNula(r.AllowedFields)
		r.Write = normalizarWrite(r.Write)
		r.NestedRelations = normalizarRelaciones(r.NestedRelations)
		out = append(out, r)
	}
	return out
}

// compararRelaciones informa cada diferencia entre dos árboles de relaciones,
// nombrando la ruta del nodo y el campo que difiere.
func compararRelaciones(t *testing.T, ruta string, esperado, obtenido []RelationConfig) {
	t.Helper()
	if len(esperado) != len(obtenido) {
		t.Errorf("%s: cantidad de relaciones: esperado %d, obtenido %d", ruta, len(esperado), len(obtenido))
		return
	}
	for i := range esperado {
		e, o := esperado[i], obtenido[i]
		nodo := ruta + "/" + e.MappedName
		if e.Name != o.Name {
			t.Errorf("%s: name esperado %q, obtenido %q", nodo, e.Name, o.Name)
		}
		campos := []struct {
			nombre       string
			esperado, ob any
		}{
			{"related_service", e.RelatedService, o.RelatedService},
			{"related_api", e.RelatedAPI, o.RelatedAPI},
			{"related_table", e.RelatedTable, o.RelatedTable},
			{"direction", e.Direction, o.Direction},
			{"local_field", e.LocalField, o.LocalField},
			{"related_field", e.RelatedField, o.RelatedField},
			{"cardinality", e.Cardinality, o.Cardinality},
			{"depth", e.Depth, o.Depth},
			{"mapped_name", e.MappedName, o.MappedName},
			{"endpoint", e.Endpoint, o.Endpoint},
			{"query_template", e.QueryTemplate, o.QueryTemplate},
			{"allowed_fields", listaNoNula(e.AllowedFields), listaNoNula(o.AllowedFields)},
			{"write", normalizarWrite(e.Write), normalizarWrite(o.Write)},
		}
		for _, c := range campos {
			if !reflect.DeepEqual(c.esperado, c.ob) {
				t.Errorf("%s: %s esperado %+v, obtenido %+v", nodo, c.nombre, c.esperado, c.ob)
			}
		}
		compararRelaciones(t, nodo, e.NestedRelations, o.NestedRelations)
	}
}

// El árbol efectivo del proveedor que produce el cargador desde el directorio real
// de configuración equivale, nodo a nodo, al árbol inline de referencia: mismas
// relaciones, llaves de propagación, campos de lectura y escritura, políticas,
// profundidades y la escritura de la raíz. Además, el proveedor se compone desde
// referencias a objetos.
func TestProveedorArbolCompuestoEquivalenteAlInline(t *testing.T) {
	base := cargarBaseInlineProveedor(t)

	comp, err := CargarComposicion(".", urlsPrueba())
	if err != nil {
		t.Fatalf("la configuración real debe componer sin error: %v", err)
	}
	obtenido, ok := comp.Servicios["proveedor"]
	if !ok {
		t.Fatal("el servicio proveedor debe existir en la composición real")
	}

	if len(obtenido.Referencias) == 0 {
		t.Fatal("el proveedor debe componerse desde referencias a objetos direccionables")
	}

	if obtenido.MaxDepth != base.MaxDepth || obtenido.MaxRelatedIDs != base.MaxRelatedIDs ||
		obtenido.OnRelationError != base.OnRelationError {
		t.Errorf("parámetros del servicio: esperado depth=%d ids=%d error=%q, obtenido depth=%d ids=%d error=%q",
			base.MaxDepth, base.MaxRelatedIDs, base.OnRelationError,
			obtenido.MaxDepth, obtenido.MaxRelatedIDs, obtenido.OnRelationError)
	}
	if !reflect.DeepEqual(normalizarWrite(base.Write), normalizarWrite(obtenido.Write)) {
		t.Errorf("write de la raíz: esperado %+v, obtenido %+v", normalizarWrite(base.Write), normalizarWrite(obtenido.Write))
	}
	if !reflect.DeepEqual(listaNoNula(base.FilterableFields), listaNoNula(obtenido.FilterableFields)) {
		t.Errorf("filterable_fields: esperado %v, obtenido %v", base.FilterableFields, obtenido.FilterableFields)
	}

	compararRelaciones(t, "", normalizarRelaciones(base.Relations), normalizarRelaciones(obtenido.Relations))
}
