package config

import (
	"reflect"
	"strings"
	"testing"
)

// Objeto raíz del proveedor con su tabla de campos. Sus campos escribibles,
// inmutables y clave natural se toman de esta tabla.
const objetoProveedorRaiz = `name: Proveedor
related_api: gestion_personas_crud
related_table: proveedor
endpoint: /v1/proveedor
natural_key:
  - TerceroId
campos:
  - name: TerceroId
    type: integer
    readable: true
    writable: true
    immutable: true
  - name: DescripcionPortafolio
    type: string
    readable: true
    writable: true
referencias: []
`

// Servicio que declara su raíz por referencia al objeto y solo la política de
// escritura en línea.
const servicioProveedorRaiz = `service: proveedor
max_depth: 4
on_relation_error: notify
max_related_ids: 100
raiz: Proveedor
write:
  policy:
    creatable: true
    updatable: true
    deletable: false
`

// El bloque write de la raíz recibe sus campos escribibles, inmutables y clave
// natural desde la tabla de campos del objeto, con la política declarada en línea.
func TestCargarComposicion_RaizTomaCamposDelObjeto(t *testing.T) {
	comp, err := escribirComposicion(t, map[string]string{
		"objetos_proveedor.yaml":    objetoProveedorRaiz,
		"relaciones_proveedor.yaml": servicioProveedorRaiz,
	})
	if err != nil {
		t.Fatalf("raíz válida rechazada: %v", err)
	}
	w := comp.Servicios["proveedor"].Write
	if w == nil {
		t.Fatal("la raíz debe tener bloque write compuesto")
	}
	if !reflect.DeepEqual(w.WritableFields, []string{"TerceroId", "DescripcionPortafolio"}) {
		t.Errorf("writable_fields = %v", w.WritableFields)
	}
	if !reflect.DeepEqual(w.ImmutableFields, []string{"TerceroId"}) {
		t.Errorf("immutable_fields = %v", w.ImmutableFields)
	}
	if !reflect.DeepEqual(w.NaturalKey, []string{"TerceroId"}) {
		t.Errorf("natural_key = %v", w.NaturalKey)
	}
	if w.KeyPropagation != "" {
		t.Errorf("la raíz no propaga llave; key_propagation = %q", w.KeyPropagation)
	}
	if !w.Policy.Creatable || !w.Policy.Updatable || w.Policy.Deletable {
		t.Errorf("policy de la raíz = %+v", w.Policy)
	}
}

// La raíz se compone desde su objeto aunque ninguna referencia lo nombre, y no se
// reporta como huérfano.
func TestCargarComposicion_RaizNoEsHuerfana(t *testing.T) {
	comp, err := escribirComposicion(t, map[string]string{
		"objetos_proveedor.yaml":    objetoProveedorRaiz,
		"relaciones_proveedor.yaml": servicioProveedorRaiz,
	})
	if err != nil {
		t.Fatalf("raíz válida rechazada: %v", err)
	}
	if len(comp.Huerfanos) != 0 {
		t.Errorf("la raíz no debe reportarse como huérfana; huérfanos = %v", comp.Huerfanos)
	}
}

// Una raíz que nombra un objeto ausente aborta el arranque nombrando el objeto.
func TestCargarComposicion_RaizInexistenteAborta(t *testing.T) {
	servicio := strings.Replace(servicioProveedorRaiz, "raiz: Proveedor", "raiz: Fantasma", 1)
	_, err := escribirComposicion(t, map[string]string{
		"objetos_proveedor.yaml":    objetoProveedorRaiz,
		"relaciones_proveedor.yaml": servicio,
	})
	if err == nil {
		t.Fatal("una raíz inexistente debe abortar el arranque")
	}
	if !strings.Contains(err.Error(), "Fantasma") {
		t.Errorf("el error debe nombrar el objeto ausente; error: %v", err)
	}
}

// Declarar campos en write junto con raiz duplica la fuente de los campos y aborta.
func TestCargarComposicion_RaizConCamposEnWriteAborta(t *testing.T) {
	servicio := strings.Replace(servicioProveedorRaiz,
		"write:\n  policy:", "write:\n  writable_fields:\n    - TerceroId\n  policy:", 1)
	_, err := escribirComposicion(t, map[string]string{
		"objetos_proveedor.yaml":    objetoProveedorRaiz,
		"relaciones_proveedor.yaml": servicio,
	})
	if err == nil {
		t.Fatal("write con campos y raiz declarada debe abortar el arranque")
	}
	if !strings.Contains(err.Error(), "writable_fields") || !strings.Contains(err.Error(), "raiz") {
		t.Errorf("el error debe nombrar write y raiz; error: %v", err)
	}
}

// La raíz recibe en NonNullableFields los campos de su tabla que no admiten null, sin
// incluir los que sí admiten null.
func TestCargarComposicion_RaizProyectaCamposNoNulables(t *testing.T) {
	objeto := `name: Proveedor
related_api: gestion_personas_crud
related_table: proveedor
endpoint: /v1/proveedor
natural_key:
  - TerceroId
campos:
  - name: TerceroId
    type: integer
    nullable: false
    readable: true
    writable: true
    immutable: true
  - name: DescripcionPortafolio
    type: string
    nullable: true
    readable: true
    writable: true
referencias: []
`
	comp, err := escribirComposicion(t, map[string]string{
		"objetos_proveedor.yaml":    objeto,
		"relaciones_proveedor.yaml": servicioProveedorRaiz,
	})
	if err != nil {
		t.Fatalf("raíz válida rechazada: %v", err)
	}
	w := comp.Servicios["proveedor"].Write
	if w == nil {
		t.Fatal("la raíz debe tener bloque write compuesto")
	}
	if !reflect.DeepEqual(w.NonNullableFields, []string{"TerceroId"}) {
		t.Errorf("non_nullable_fields = %v, se esperaba [TerceroId]", w.NonNullableFields)
	}
}
