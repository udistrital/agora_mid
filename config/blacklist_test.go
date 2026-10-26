package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// escribirBlacklist escribe un archivo de lista negra en un directorio temporal y
// devuelve su ruta.
func escribirBlacklist(t *testing.T, contenido string) string {
	t.Helper()
	dir := t.TempDir()
	ruta := filepath.Join(dir, "global_blacklist.yaml")
	if err := os.WriteFile(ruta, []byte(contenido), 0o600); err != nil {
		t.Fatal(err)
	}
	return ruta
}

// configsPrueba construye un conjunto mínimo de configuraciones de servicio que
// cubre las apis, tablas y campos usados por las pruebas de la lista negra.
func configsPrueba() map[string]ServiceConfig {
	return map[string]ServiceConfig{
		"proveedor": {
			Service:         "proveedor",
			MaxDepth:        4,
			OnRelationError: OnRelationErrorNotify,
			MaxRelatedIDs:   100,
			Relations: []RelationConfig{
				{
					Name:          "Tercero",
					RelatedAPI:    "terceros",
					RelatedTable:  "tercero",
					RelatedField:  "Id",
					AllowedFields: []string{"NombreCompleto", "TipoContribuyenteId"},
					NestedRelations: []RelationConfig{
						{
							Name:          "Contacto",
							RelatedAPI:    "gestion_personas_crud",
							RelatedTable:  "contacto",
							RelatedField:  "ProveedorId",
							AllowedFields: []string{"NombreContacto", "ValorEspecifico"},
						},
					},
				},
			},
		},
	}
}

// Un archivo de ejemplo con entradas exactas y comodín parsea a la estructura
// jerárquica esperada (task 4.1).
func TestLoadBlacklistArchivoValido(t *testing.T) {
	contenido := `global_blacklist:
  ANY:
    ANY:
      - AuditingField
  terceros:
    tercero:
      - NombreCompleto
  gestion_personas_crud:
    contacto:
      - ANY
`
	ruta := escribirBlacklist(t, contenido)

	bl, err := LoadBlacklist(ruta)
	if err != nil {
		t.Fatalf("error inesperado al cargar lista negra válida: %v", err)
	}

	if campos := bl[WildcardANY][WildcardANY]; len(campos) != 1 || campos[0] != "AuditingField" {
		t.Errorf("entrada ANY/ANY inesperada: %+v", bl[WildcardANY])
	}
	if campos := bl["terceros"]["tercero"]; len(campos) != 1 || campos[0] != "NombreCompleto" {
		t.Errorf("entrada terceros/tercero inesperada: %+v", bl["terceros"])
	}
	if campos := bl["gestion_personas_crud"]["contacto"]; len(campos) != 1 || campos[0] != WildcardANY {
		t.Errorf("entrada gestion_personas_crud/contacto inesperada: %+v", bl["gestion_personas_crud"])
	}
}

// La ausencia del archivo no es error: devuelve una lista negra vacía (task 4.1).
func TestLoadBlacklistArchivoAusente(t *testing.T) {
	dir := t.TempDir()
	ruta := filepath.Join(dir, "no_existe.yaml")

	bl, err := LoadBlacklist(ruta)
	if err != nil {
		t.Fatalf("un archivo ausente no debe ser error; recibido: %v", err)
	}
	if len(bl) != 0 {
		t.Errorf("se esperaba lista negra vacía; recibido: %+v", bl)
	}
}

// Un archivo vacío (o sin la clave global_blacklist) equivale a "sin supresión"
// sin error (task 4.1).
func TestLoadBlacklistArchivoVacio(t *testing.T) {
	casos := map[string]string{
		"archivo vacío":            "",
		"solo espacios":           "   \n  \n",
		"sin clave raíz":          "# solo un comentario\n",
		"clave raíz vacía":        "global_blacklist:\n",
	}
	for nombre, contenido := range casos {
		t.Run(nombre, func(t *testing.T) {
			ruta := escribirBlacklist(t, contenido)
			bl, err := LoadBlacklist(ruta)
			if err != nil {
				t.Fatalf("un archivo vacío no debe ser error; recibido: %v", err)
			}
			if len(bl) != 0 {
				t.Errorf("se esperaba lista negra vacía; recibido: %+v", bl)
			}
		})
	}
}

// Un YAML inválido en un archivo presente devuelve un error que identifica la
// ruta (task 4.1).
func TestLoadBlacklistYamlInvalido(t *testing.T) {
	ruta := escribirBlacklist(t, "global_blacklist:\n  : : :\n")
	_, err := LoadBlacklist(ruta)
	if err == nil {
		t.Fatalf("se esperaba error de YAML inválido")
	}
	if !strings.Contains(err.Error(), "global_blacklist.yaml") {
		t.Errorf("el mensaje debe identificar la ruta; mensaje: %s", err.Error())
	}
}

// Una clave repetida en el mismo nivel del mapa (related_api o related_table)
// es un error de parseo del YAML, no "la última gana": la unicidad por api/tabla
// la impone el propio YAML al ser un mapa plano con KnownFields activado. Un
// campo repetido dentro de la lista, en cambio, es una secuencia y no es error
// (el duplicado es redundante) (task 4.1).
func TestLoadBlacklistClaveRepetidaMismoNivel(t *testing.T) {
	casos := []struct {
		nombre    string
		contenido string
	}{
		{
			nombre: "related_api repetida",
			contenido: `global_blacklist:
  terceros:
    tercero:
      - NombreCompleto
  terceros:
    datos_identificacion:
      - Numero
`,
		},
		{
			nombre: "related_table repetida bajo la misma api",
			contenido: `global_blacklist:
  terceros:
    tercero:
      - NombreCompleto
    tercero:
      - TipoContribuyenteId
`,
		},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			ruta := escribirBlacklist(t, c.contenido)
			_, err := LoadBlacklist(ruta)
			if err == nil {
				t.Fatalf("una clave repetida en el mismo nivel debe ser error de YAML")
			}
			if !strings.Contains(err.Error(), "already defined") {
				t.Errorf("el mensaje debe señalar la clave duplicada del YAML; mensaje: %s", err.Error())
			}
			if !strings.Contains(err.Error(), "global_blacklist.yaml") {
				t.Errorf("el mensaje debe identificar la ruta; mensaje: %s", err.Error())
			}
		})
	}
}

// Un campo repetido dentro de la lista de una tabla no es error: una secuencia
// YAML admite elementos repetidos y el duplicado queda como redundante (task 4.1).
func TestLoadBlacklistCampoRepetidoEnListaNoFalla(t *testing.T) {
	ruta := escribirBlacklist(t, `global_blacklist:
  terceros:
    tercero:
      - NombreCompleto
      - NombreCompleto
`)
	bl, err := LoadBlacklist(ruta)
	if err != nil {
		t.Fatalf("un campo repetido en la lista no debe ser error; recibido: %v", err)
	}
	if campos := bl["terceros"]["tercero"]; len(campos) != 2 || campos[0] != "NombreCompleto" || campos[1] != "NombreCompleto" {
		t.Errorf("el duplicado debe conservarse como redundante en el slice; recibido: %+v", campos)
	}
}
func TestValidarBlacklistValida(t *testing.T) {
	bl := GlobalBlacklist{
		"terceros": {
			"tercero": {"NombreCompleto", "Id"},
		},
		WildcardANY: {
			WildcardANY: {"NombreCompleto"},
		},
		"gestion_personas_crud": {
			"contacto": {WildcardANY},
		},
	}
	if err := ValidarBlacklist(bl, configsPrueba()); err != nil {
		t.Fatalf("una lista negra válida no debe abortar; recibido: %v", err)
	}
}

// Un componente exacto inexistente en el modelo aborta la validación (task 4.2).
func TestValidarBlacklistComponenteInexistente(t *testing.T) {
	casos := []struct {
		nombre       string
		bl           GlobalBlacklist
		debeContener []string
	}{
		{
			nombre:       "related_api inexistente",
			bl:           GlobalBlacklist{"inexistente": {"tercero": {"NombreCompleto"}}},
			debeContener: []string{"related_api", "inexistente"},
		},
		{
			nombre:       "related_table inexistente",
			bl:           GlobalBlacklist{"terceros": {"tabla_fantasma": {"NombreCompleto"}}},
			debeContener: []string{"related_table", "tabla_fantasma"},
		},
		{
			nombre:       "campo inexistente",
			bl:           GlobalBlacklist{"terceros": {"tercero": {"CampoFantasma"}}},
			debeContener: []string{"CampoFantasma"},
		},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			err := ValidarBlacklist(c.bl, configsPrueba())
			if err == nil {
				t.Fatalf("se esperaba error de validación")
			}
			for _, frag := range c.debeContener {
				if !strings.Contains(err.Error(), frag) {
					t.Errorf("el mensaje debe contener %q; mensaje: %s", frag, err.Error())
				}
			}
		})
	}
}

// ANY/ANY/ANY (comodín en los tres niveles) se rechaza (task 4.2).
func TestValidarBlacklistAnyAnyAnyProhibido(t *testing.T) {
	bl := GlobalBlacklist{
		WildcardANY: {
			WildcardANY: {WildcardANY},
		},
	}
	err := ValidarBlacklist(bl, configsPrueba())
	if err == nil {
		t.Fatalf("ANY/ANY/ANY debe abortar")
	}
	if !strings.Contains(err.Error(), "ANY/ANY/ANY") && !strings.Contains(err.Error(), "comodín en los tres niveles") {
		t.Errorf("el mensaje debe señalar el comodín total; mensaje: %s", err.Error())
	}
}

// Una entrada con ANY en un nivel no falla por inexistencia de los demás
// componentes: el comodín no se verifica contra el modelo (task 4.2).
func TestValidarBlacklistComodinNoSeVerifica(t *testing.T) {
	casos := map[string]GlobalBlacklist{
		"ANY en api con tabla/campo reales": {
			WildcardANY: {"tercero": {"NombreCompleto"}},
		},
		"ANY en tabla con api/campo reales": {
			"terceros": {WildcardANY: {"NombreCompleto"}},
		},
		"ANY en campo con api/tabla reales": {
			"terceros": {"tercero": {WildcardANY}},
		},
		"ANY en api y tabla, campo real": {
			WildcardANY: {WildcardANY: {"NombreCompleto"}},
		},
	}
	for nombre, bl := range casos {
		t.Run(nombre, func(t *testing.T) {
			if err := ValidarBlacklist(bl, configsPrueba()); err != nil {
				t.Errorf("un comodín no debe fallar por inexistencia; recibido: %v", err)
			}
		})
	}
}
