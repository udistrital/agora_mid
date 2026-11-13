package generador

import (
	"os"
	"path/filepath"
	"testing"
)

// dbmlPrueba es un DBML reducido con la misma forma que el modelo real: partes
// reutilizables con ~, un bloque indexes anidado, un 1-N, una FK de terceros y
// una relación 1-1 de PK compartida.
const dbmlPrueba = `
TablePartial id {
  id serial [pk]
}

TablePartial campos_auditoria {
  activo boolean [not null, default: true]
  fecha_creacion timestamp [not null, default: now()]
  fecha_modificacion timestamp
  usuario_modificacion_id integer
}

Table terceros.tercero {
  ~id
  nombre_completo varchar(255) [not null]
  ~campos_auditoria
}

Table proveedores.proveedor {
  ~id
  tipo_proveedor integer [not null]
  tercero_id integer
  ~campos_auditoria

  indexes {
    id [name: 'pk_proveedor']
  }
}

Table proveedores.proveedor_natural {
  ~id
  pais_nacimiento_id integer [not null]
  cargo varchar(150)
  ~campos_auditoria
}

Table proveedores.contacto {
  ~id
  proveedor_id integer [not null]
  valor_especifico varchar(100) [not null]
  ~campos_auditoria
}

Ref fk_contacto_proveedor {
  proveedores.proveedor.id < proveedores.contacto.proveedor_id
}

Ref fk_proveedor_tercero {
  terceros.tercero.id < proveedores.proveedor.tercero_id
}

Ref fk_proveedor_natural_proveedor {
  proveedores.proveedor.id - proveedores.proveedor_natural.id
}
`

// rolesSemillaPrueba declara los roles de las columnas administradas por el sistema con
// comodín ANY, como lo hace el sidecar field_roles real.
const rolesSemillaPrueba = `ANY.ANY.id:
  readable: true
  writable: false
  immutable: false
ANY.ANY.activo:
  readable: true
  writable: false
  immutable: false
ANY.ANY.fecha_creacion:
  readable: true
  writable: false
  immutable: false
ANY.ANY.fecha_modificacion:
  readable: true
  writable: false
  immutable: false
ANY.ANY.usuario_modificacion_id:
  readable: true
  writable: true
  immutable: false
`

// escribirSidecarsPrueba crea una carpeta temporal con los sidecars de prueba. Cada
// argumento es el contenido YAML de un archivo; field_roles recibe la semilla.
func escribirSidecarsPrueba(t *testing.T, expand, fkInbound, blacklist, write string) string {
	t.Helper()
	dir := t.TempDir()
	archivos := map[string]string{
		"expand.yaml":      expand,
		"fk_inbound.yaml":  fkInbound,
		"blacklist.yaml":   blacklist,
		"write.yaml":       write,
		"field_roles.yaml": rolesSemillaPrueba,
	}
	for nombre, contenido := range archivos {
		if err := os.WriteFile(filepath.Join(dir, nombre), []byte(contenido), 0o644); err != nil {
			t.Fatalf("escribiendo sidecar %s: %v", nombre, err)
		}
	}
	return dir
}

// construirPrueba parsea el DBML de prueba, carga los sidecars y construye los
// objetos. Devuelve el primer error de carga o de construcción.
func construirPrueba(t *testing.T, expand, fkInbound, blacklist, write string) ([]ObjetoConfig, error) {
	t.Helper()
	m, err := ParsearDBML(dbmlPrueba)
	if err != nil {
		t.Fatalf("parseando el DBML de prueba: %v", err)
	}
	s, err := CargarSidecars(escribirSidecarsPrueba(t, expand, fkInbound, blacklist, write), m)
	if err != nil {
		return nil, err
	}
	return ConstruirObjetos(m, s)
}

// mustConstruir es construirPrueba para casos que deben construir sin error.
func mustConstruir(t *testing.T, expand, fkInbound, blacklist, write string) []ObjetoConfig {
	t.Helper()
	objetos, err := construirPrueba(t, expand, fkInbound, blacklist, write)
	if err != nil {
		t.Fatalf("construyendo objetos: %v", err)
	}
	return objetos
}

// buscarObjeto devuelve el objeto con el nombre dado o falla la prueba.
func buscarObjeto(t *testing.T, objetos []ObjetoConfig, nombre string) ObjetoConfig {
	t.Helper()
	for _, o := range objetos {
		if o.Name == nombre {
			return o
		}
	}
	t.Fatalf("no existe el objeto %q", nombre)
	return ObjetoConfig{}
}

// buscarArista devuelve la arista del objeto que apunta a objeto o falla la prueba.
func buscarArista(t *testing.T, o ObjetoConfig, objeto string) AristaConfig {
	t.Helper()
	for _, a := range o.Referencias {
		if a.Objeto == objeto {
			return a
		}
	}
	t.Fatalf("el objeto %q no tiene arista hacia %q", o.Name, objeto)
	return AristaConfig{}
}

// prepararPrueba parsea dbml, escribe archivos en una carpeta temporal y carga los
// sidecars desde ella. Devuelve el modelo, los sidecars y el error de carga.
func prepararPrueba(t *testing.T, dbml string, archivos map[string]string) (*Modelo, *Sidecars, error) {
	t.Helper()
	m, err := ParsearDBML(dbml)
	if err != nil {
		t.Fatalf("parseando el DBML de prueba: %v", err)
	}
	dir := t.TempDir()
	for nombre, contenido := range archivos {
		if err := os.WriteFile(filepath.Join(dir, nombre), []byte(contenido), 0o644); err != nil {
			t.Fatalf("escribiendo %s: %v", nombre, err)
		}
	}
	s, err := CargarSidecars(dir, m)
	return m, s, err
}

// servicioPrueba es un sidecar service válido para la raíz proveedores.proveedor.
const servicioPrueba = `proveedores.proveedor:
  service: proveedor
  max_depth: 4
  on_relation_error: notify
  max_related_ids: 100
  filterable_fields:
    - id
    - tercero_id
`
