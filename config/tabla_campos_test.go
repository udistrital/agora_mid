package config

import (
	"strings"
	"testing"
)

// Prueba que una tabla de campos YAML se parsea con su tipo y sus cuatro banderas
// de rol, en el orden en que se declara.
func TestDecodificarTablaCampos_ParseaTipoYBanderas(t *testing.T) {
	contenido := []byte(`
- name: NombreCompleto
  type: string
  readable: true
  writable: true
  immutable: false
  required: true
- name: FechaNacimiento
  type: timestamp
  writable: true
`)

	tabla, err := DecodificarTablaCampos("Tercero", contenido)
	if err != nil {
		t.Fatalf("se esperaba una tabla válida, error: %v", err)
	}
	if len(tabla) != 2 {
		t.Fatalf("se esperaban 2 campos, hay %d", len(tabla))
	}

	nombre := tabla[0]
	if nombre.Name != "NombreCompleto" || nombre.Type != TipoString {
		t.Errorf("campo 0 = {%q, %q}, se esperaba {NombreCompleto, string}", nombre.Name, nombre.Type)
	}
	if !nombre.Readable || !nombre.Writable || nombre.Immutable || !nombre.Required {
		t.Errorf("banderas de NombreCompleto = readable:%v writable:%v immutable:%v required:%v",
			nombre.Readable, nombre.Writable, nombre.Immutable, nombre.Required)
	}

	fecha := tabla[1]
	if fecha.Name != "FechaNacimiento" || fecha.Type != TipoTimestamp {
		t.Errorf("campo 1 = {%q, %q}, se esperaba {FechaNacimiento, timestamp}", fecha.Name, fecha.Type)
	}
	if fecha.Readable || !fecha.Writable || fecha.Immutable || fecha.Required {
		t.Errorf("banderas de FechaNacimiento incorrectas: readable:%v writable:%v immutable:%v required:%v",
			fecha.Readable, fecha.Writable, fecha.Immutable, fecha.Required)
	}
}

// Prueba que un campo sin type aborta con un error que nombra el objeto y el campo.
func TestDecodificarTablaCampos_CampoSinTypeProduceErrorDeModelo(t *testing.T) {
	contenido := []byte(`
- name: NombreCompleto
  writable: true
`)

	_, err := DecodificarTablaCampos("Tercero", contenido)
	if err == nil {
		t.Fatal("se esperaba error por campo sin type")
	}
	if !strings.Contains(err.Error(), "Tercero") || !strings.Contains(err.Error(), "NombreCompleto") {
		t.Errorf("el error debe nombrar objeto y campo, obtuvo: %v", err)
	}
}

// Prueba que una bandera desconocida (por ejemplo un error tipográfico) aborta en
// vez de descartarse en silencio.
func TestDecodificarTablaCampos_BanderaDesconocidaAborta(t *testing.T) {
	contenido := []byte(`
- name: NombreCompleto
  type: string
  readble: true
`)

	if _, err := DecodificarTablaCampos("Tercero", contenido); err == nil {
		t.Fatal("se esperaba error por bandera desconocida readble")
	}
}

// Prueba la coherencia por campo: immutable y required exigen writable.
func TestDecodificarTablaCampos_CoherenciaConWritable(t *testing.T) {
	casos := []struct {
		nombre     string
		contenido  string
		esperaErr  bool
		campoError string
	}{
		{
			nombre:     "immutable sin writable aborta",
			contenido:  "- name: Numero\n  type: string\n  immutable: true\n",
			esperaErr:  true,
			campoError: "Numero",
		},
		{
			nombre:     "required sin writable aborta",
			contenido:  "- name: TerceroId\n  type: integer\n  required: true\n  readable: true\n",
			esperaErr:  true,
			campoError: "TerceroId",
		},
		{
			nombre:    "immutable con writable se acepta",
			contenido: "- name: Numero\n  type: string\n  writable: true\n  immutable: true\n",
			esperaErr: false,
		},
		{
			nombre:    "required con writable se acepta",
			contenido: "- name: TerceroId\n  type: integer\n  writable: true\n  required: true\n",
			esperaErr: false,
		},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			_, err := DecodificarTablaCampos("DatosIdentificacion", []byte(c.contenido))
			if c.esperaErr {
				if err == nil {
					t.Fatal("se esperaba error de coherencia")
				}
				if !strings.Contains(err.Error(), "DatosIdentificacion") || !strings.Contains(err.Error(), c.campoError) {
					t.Errorf("el error debe nombrar objeto y campo %q, obtuvo: %v", c.campoError, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("no se esperaba error, obtuvo: %v", err)
			}
		})
	}
}

// Prueba que cada tipo del vocabulario cerrado de almacenamiento se acepta.
func TestDecodificarTablaCampos_AceptaVocabularioCerrado(t *testing.T) {
	tipos := []string{"boolean", "integer", "jsonb", "numeric", "string", "timestamp", "uuid"}

	for _, tipo := range tipos {
		t.Run(tipo, func(t *testing.T) {
			contenido := []byte("- name: Campo\n  type: " + tipo + "\n  writable: true\n")
			tabla, err := DecodificarTablaCampos("Objeto", contenido)
			if err != nil {
				t.Fatalf("el tipo %q debe aceptarse, error: %v", tipo, err)
			}
			if string(tabla[0].Type) != tipo {
				t.Errorf("tipo conservado = %q, se esperaba %q", tabla[0].Type, tipo)
			}
		})
	}
}

// Prueba que un tipo fuera del vocabulario cerrado aborta nombrando objeto, campo
// y tipo. Incluye alias de otros motores y variaciones de mayúsculas o espacios.
func TestDecodificarTablaCampos_TipoFueraDelVocabularioAborta(t *testing.T) {
	tipos := []string{"varchar", "text", "String", "bigint", "uuid "}

	for _, tipo := range tipos {
		t.Run(tipo, func(t *testing.T) {
			contenido := []byte("- name: NombreCompleto\n  type: \"" + tipo + "\"\n  writable: true\n")
			_, err := DecodificarTablaCampos("Tercero", contenido)
			if err == nil {
				t.Fatalf("el tipo %q debe abortar", tipo)
			}
			for _, esperado := range []string{"Tercero", "NombreCompleto", tipo} {
				if !strings.Contains(err.Error(), esperado) {
					t.Errorf("el error debe nombrar %q, obtuvo: %v", esperado, err)
				}
			}
		})
	}
}

// Prueba que las proyecciones de la tabla tipada reproducen, en el mismo orden, las
// listas vigentes del YAML del proveedor: allowed_fields (lectura), writable_fields
// e immutable_fields (escritura). Los objetos de ejemplo son Tercero y
// DatosIdentificacion.
func TestTablaCampos_ProyeccionesEquivalentesAListasVigentes(t *testing.T) {
	tercero := []byte(`
- name: NombreCompleto
  type: string
  readable: true
  writable: true
- name: PrimerNombre
  type: string
  writable: true
- name: SegundoNombre
  type: string
  writable: true
- name: PrimerApellido
  type: string
  writable: true
- name: SegundoApellido
  type: string
  writable: true
- name: FechaNacimiento
  type: timestamp
  writable: true
- name: TipoContribuyenteId
  type: integer
  readable: true
  writable: true
`)
	datos := []byte(`
- name: Numero
  type: string
  readable: true
  writable: true
  immutable: true
- name: TipoDocumentoId
  type: integer
  readable: true
  writable: true
  immutable: true
- name: DigitoVerificacion
  type: integer
  writable: true
- name: CiudadExpedicion
  type: integer
  readable: true
  writable: true
- name: FechaExpedicion
  type: timestamp
  writable: true
`)

	casos := []struct {
		nombre     string
		contenido  []byte
		lectura    []string
		escritura  []string
		inmutables []string
	}{
		{
			nombre:     "Tercero",
			contenido:  tercero,
			lectura:    []string{"NombreCompleto", "TipoContribuyenteId"},
			escritura:  []string{"NombreCompleto", "PrimerNombre", "SegundoNombre", "PrimerApellido", "SegundoApellido", "FechaNacimiento", "TipoContribuyenteId"},
			inmutables: []string{},
		},
		{
			nombre:     "DatosIdentificacion",
			contenido:  datos,
			lectura:    []string{"Numero", "TipoDocumentoId", "CiudadExpedicion"},
			escritura:  []string{"Numero", "TipoDocumentoId", "DigitoVerificacion", "CiudadExpedicion", "FechaExpedicion"},
			inmutables: []string{"Numero", "TipoDocumentoId"},
		},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			tabla, err := DecodificarTablaCampos(c.nombre, c.contenido)
			if err != nil {
				t.Fatalf("tabla válida rechazada: %v", err)
			}
			if got := tabla.CamposLectura(); !sonIgualesCadenas(got, c.lectura) {
				t.Errorf("CamposLectura() = %v, se esperaba %v", got, c.lectura)
			}
			if got := tabla.CamposEscritura(); !sonIgualesCadenas(got, c.escritura) {
				t.Errorf("CamposEscritura() = %v, se esperaba %v", got, c.escritura)
			}
			if got := tabla.CamposInmutables(); !sonIgualesCadenas(got, c.inmutables) {
				t.Errorf("CamposInmutables() = %v, se esperaba %v", got, c.inmutables)
			}
		})
	}
}

// sonIgualesCadenas compara dos listas de cadenas en orden y contenido. Una lista
// nil y una vacía se consideran iguales.
func sonIgualesCadenas(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Prueba los casos límite de la tabla: nombre de campo duplicado o vacío aborta
// nombrando el objeto y el campo; un contenido vacío produce una tabla sin campos
// y proyecciones no nulas.
func TestDecodificarTablaCampos_CasosLimite(t *testing.T) {
	t.Run("nombre duplicado aborta nombrando el campo", func(t *testing.T) {
		contenido := []byte("- name: Numero\n  type: string\n  writable: true\n- name: Numero\n  type: string\n  writable: true\n")
		_, err := DecodificarTablaCampos("DatosIdentificacion", contenido)
		if err == nil || !strings.Contains(err.Error(), "Numero") || !strings.Contains(err.Error(), "DatosIdentificacion") {
			t.Fatalf("se esperaba error que nombre objeto y campo duplicado, obtuvo: %v", err)
		}
	})

	t.Run("nombre vacío aborta", func(t *testing.T) {
		contenido := []byte("- name: \"  \"\n  type: string\n")
		_, err := DecodificarTablaCampos("Tercero", contenido)
		if err == nil || !strings.Contains(err.Error(), "Tercero") {
			t.Fatalf("se esperaba error que nombre el objeto, obtuvo: %v", err)
		}
	})

	t.Run("contenido vacío produce tabla sin campos", func(t *testing.T) {
		tabla, err := DecodificarTablaCampos("Tercero", []byte(""))
		if err != nil {
			t.Fatalf("contenido vacío no debe fallar, error: %v", err)
		}
		if len(tabla) != 0 {
			t.Errorf("se esperaba tabla vacía, hay %d campos", len(tabla))
		}
		if tabla.CamposLectura() == nil || tabla.CamposEscritura() == nil || tabla.CamposInmutables() == nil {
			t.Error("las proyecciones de una tabla vacía deben ser listas no nulas")
		}
	})
}

// Prueba que la bandera nullable se decodifica por campo, que CamposNoNulables devuelve
// solo los campos que no admiten null en orden de declaración, y que un campo sin la
// clave no admite null (valor por omisión seguro).
func TestDecodificarTablaCampos_NullableYCamposNoNulables(t *testing.T) {
	contenido := []byte(`
- name: Numero
  type: string
  nullable: false
  writable: true
- name: Observacion
  type: string
  nullable: true
  writable: true
- name: Sin
  type: string
  writable: true
- name: Activo
  type: boolean
  nullable: false
  readable: true
`)

	tabla, err := DecodificarTablaCampos("Demo", contenido)
	if err != nil {
		t.Fatalf("se esperaba una tabla válida, error: %v", err)
	}
	esperados := map[string]bool{"Numero": false, "Observacion": true, "Sin": false, "Activo": false}
	for _, c := range tabla {
		if c.Nullable != esperados[c.Name] {
			t.Errorf("campo %s: Nullable = %v, se esperaba %v", c.Name, c.Nullable, esperados[c.Name])
		}
	}
	if got, want := tabla.CamposNoNulables(), []string{"Numero", "Sin", "Activo"}; !sonIgualesCadenas(got, want) {
		t.Errorf("CamposNoNulables() = %v, se esperaba %v", got, want)
	}
}
