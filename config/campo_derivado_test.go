package config

import (
	"strings"
	"testing"
)

// El campo derivado se identifica de forma determinista según el modo de
// propagación de llave declarado: fk -> related_field, shared_pk -> Id,
// fk_inbound -> local_field. Una relación sin write, sin modo o con un modo
// desconocido no tiene campo derivado.
func TestCampoDerivadoPorModo(t *testing.T) {
	casos := []struct {
		nombre   string
		rel      RelationConfig
		esperado string
		ok       bool
	}{
		{
			nombre: "fk usa related_field del hijo",
			rel: RelationConfig{
				LocalField:   "ProveedorId",
				RelatedField: "ProveedorId",
				Write:        &WriteConfig{KeyPropagation: KeyPropagationFK},
			},
			esperado: "ProveedorId",
			ok:       true,
		},
		{
			nombre: "shared_pk usa el Id del detalle",
			rel: RelationConfig{
				LocalField:   "Id",
				RelatedField: "Id",
				Write:        &WriteConfig{KeyPropagation: KeyPropagationSharedPK},
			},
			esperado: "Id",
			ok:       true,
		},
		{
			nombre: "fk_inbound usa local_field del nodo",
			rel: RelationConfig{
				LocalField:   "TerceroId",
				RelatedField: "Id",
				Write:        &WriteConfig{KeyPropagation: KeyPropagationFKInbound},
			},
			esperado: "TerceroId",
			ok:       true,
		},
		{
			nombre:   "sin bloque write no hay campo derivado",
			rel:      RelationConfig{LocalField: "X", RelatedField: "Y"},
			esperado: "",
			ok:       false,
		},
		{
			nombre: "modo desconocido no identifica campo",
			rel: RelationConfig{
				LocalField:   "X",
				RelatedField: "Y",
				Write:        &WriteConfig{KeyPropagation: "otro"},
			},
			esperado: "",
			ok:       false,
		},
		{
			nombre: "fk sin related_field no identifica campo",
			rel: RelationConfig{
				LocalField: "X",
				Write:      &WriteConfig{KeyPropagation: KeyPropagationFK},
			},
			esperado: "",
			ok:       false,
		},
		{
			nombre: "fk_inbound sin local_field no identifica campo",
			rel: RelationConfig{
				RelatedField: "Id",
				Write:        &WriteConfig{KeyPropagation: KeyPropagationFKInbound},
			},
			esperado: "",
			ok:       false,
		},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			campo, ok := c.rel.CampoDerivado()
			if ok != c.ok {
				t.Fatalf("CampoDerivado ok = %v; esperado %v", ok, c.ok)
			}
			if campo != c.esperado {
				t.Errorf("CampoDerivado = %q; esperado %q", campo, c.esperado)
			}
		})
	}
}

// relacionesInboundWrite declara una raíz que referencia a una entidad por
// fk_inbound (su FK local es TerceroId), para verificar que el loader identifica
// el campo derivado ascendente y conserva la configuración.
const relacionesInboundWrite = `service: proveedor
max_depth: 2
on_relation_error: notify
max_related_ids: 100
write:
  writable_fields:
    - TerceroId
  policy:
    creatable: true
    updatable: true
relations:
  - name: Tercero
    related_service: tercero
    related_api: terceros
    related_table: tercero
    direction: local_to_related
    local_field: TerceroId
    related_field: Id
    cardinality: one
    depth: 1
    mapped_name: Tercero
    endpoint: /v1/tercero
    query_template: "Id:{ids}"
    write:
      writable_fields:
        - NombreCompleto
      key_propagation: fk_inbound
      policy:
        creatable: true
        updatable: true
`

// El loader carga una relación fk_inbound e identifica su campo derivado como la
// FK local del nodo (local_field).
func TestLoadRelacionesInboundIdentificaCampoDerivado(t *testing.T) {
	dir := escribirRelaciones(t, "proveedor", relacionesInboundWrite)

	configs, err := LoadRelaciones(dir, urlsPrueba())
	if err != nil {
		t.Fatalf("configuración fk_inbound coherente debe cargar sin error: %v", err)
	}
	rel := configs["proveedor"].Relations[0]
	campo, ok := rel.CampoDerivado()
	if !ok || campo != "TerceroId" {
		t.Errorf("fk_inbound debe derivar el campo local_field TerceroId; campo=%q ok=%v", campo, ok)
	}
}

// Una relación escribible cuyo modo no identifica un campo derivado determinista
// (fk_inbound cuyo local_field quedó vacío) se rechaza al cargar con un mensaje
// que identifica el archivo y la relación.
func TestLoadRelacionesInboundSinCampoDerivado(t *testing.T) {
	contenido := strings.Replace(
		relacionesInboundWrite,
		"    local_field: TerceroId",
		`    local_field: ""`,
		1,
	)
	if contenido == relacionesInboundWrite {
		t.Fatalf("el caso no modificó la configuración base")
	}
	dir := escribirRelaciones(t, "proveedor", contenido)

	_, err := LoadRelaciones(dir, urlsPrueba())
	if err == nil {
		t.Fatalf("se esperaba error de validación por campo derivado indeterminado")
	}
	for _, frag := range []string{"relaciones_proveedor.yaml", "Tercero", "local_field"} {
		if !strings.Contains(err.Error(), frag) {
			t.Errorf("el mensaje debe contener %q; mensaje: %s", frag, err.Error())
		}
	}
}
