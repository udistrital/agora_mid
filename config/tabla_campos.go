package config

import (
	"fmt"
	"strings"
)

// TipoCampo es el tipo de almacenamiento de un campo de un objeto de
// configuración. Guía la lógica de negocio; no fija cómo se transmite el campo
// en el API ni cómo se documenta.
type TipoCampo string

// Vocabulario de tipos de almacenamiento admitidos para un campo.
const (
	TipoBoolean   TipoCampo = "boolean"
	TipoInteger   TipoCampo = "integer"
	TipoJSONB     TipoCampo = "jsonb"
	TipoNumeric   TipoCampo = "numeric"
	TipoString    TipoCampo = "string"
	TipoTimestamp TipoCampo = "timestamp"
	TipoUUID      TipoCampo = "uuid"
)

// CampoObjeto describe un campo de un objeto de configuración: su nombre, su tipo
// de almacenamiento y sus banderas de rol.
//
//   - Readable: el campo se incluye en la vista de lectura del objeto.
//   - Writable: el cliente puede enviar el campo en una escritura.
//   - Immutable: el campo no cambia tras la creación. Exige Writable.
//   - Required: el campo es obligatorio en una escritura. Exige Writable.
type CampoObjeto struct {
	Name      string    `yaml:"name"`
	Type      TipoCampo `yaml:"type"`
	Readable  bool      `yaml:"readable"`
	Writable  bool      `yaml:"writable"`
	Immutable bool      `yaml:"immutable"`
	Required  bool      `yaml:"required"`
}

// TablaCampos es la tabla de campos tipada de un objeto. Es la única fuente de
// verdad de sus campos para la proyección de lectura y la matriz de escritura.
// Conserva el orden de declaración.
type TablaCampos []CampoObjeto

// DecodificarTablaCampos interpreta una tabla de campos en YAML y la valida. Las
// claves desconocidas abortan la decodificación. El nombre del objeto se usa solo
// para identificar el origen en los errores. Un contenido vacío produce una tabla
// vacía.
func DecodificarTablaCampos(objeto string, contenido []byte) (TablaCampos, error) {
	var tabla TablaCampos
	if err := decodificarEstricto(contenido, &tabla); err != nil {
		return nil, fmt.Errorf("tabla de campos inválida del objeto %q: %w", objeto, err)
	}
	if err := tabla.Validar(objeto); err != nil {
		return nil, err
	}
	return tabla, nil
}

// Validar comprueba que cada campo tenga nombre único y tipo, y que las banderas
// de rol sean coherentes. Devuelve el primer error encontrado, que nombra el
// objeto y el campo afectado.
func (t TablaCampos) Validar(objeto string) error {
	vistos := make(map[string]struct{}, len(t))
	for i, campo := range t {
		if strings.TrimSpace(campo.Name) == "" {
			return fmt.Errorf("objeto %q: el campo en la posición %d no declara name", objeto, i)
		}
		if _, dup := vistos[campo.Name]; dup {
			return fmt.Errorf("objeto %q: el campo %q está declarado más de una vez", objeto, campo.Name)
		}
		vistos[campo.Name] = struct{}{}

		if strings.TrimSpace(string(campo.Type)) == "" {
			return fmt.Errorf("objeto %q, campo %q: el campo no declara type", objeto, campo.Name)
		}
		if !tipoCampoValido(campo.Type) {
			return fmt.Errorf("objeto %q, campo %q: type %q no pertenece al vocabulario cerrado de tipos", objeto, campo.Name, campo.Type)
		}
		if campo.Immutable && !campo.Writable {
			return fmt.Errorf("objeto %q, campo %q: immutable: true exige writable: true", objeto, campo.Name)
		}
		if campo.Required && !campo.Writable {
			return fmt.Errorf("objeto %q, campo %q: required: true exige writable: true", objeto, campo.Name)
		}
	}
	return nil
}

// tiposCampoValidos es el vocabulario cerrado de tipos de almacenamiento. Un tipo
// fuera de este conjunto impide el arranque.
var tiposCampoValidos = map[TipoCampo]struct{}{
	TipoBoolean:   {},
	TipoInteger:   {},
	TipoJSONB:     {},
	TipoNumeric:   {},
	TipoString:    {},
	TipoTimestamp: {},
	TipoUUID:      {},
}

// tipoCampoValido indica si el tipo pertenece al vocabulario cerrado. La
// comparación es exacta: no normaliza mayúsculas ni espacios.
func tipoCampoValido(tipo TipoCampo) bool {
	_, ok := tiposCampoValidos[tipo]
	return ok
}

// CamposLectura devuelve los nombres de los campos con rol readable, en orden de
// declaración. Es la vista de lectura del objeto.
func (t TablaCampos) CamposLectura() []string {
	return proyectarCampos(t, func(c CampoObjeto) bool { return c.Readable })
}

// CamposEscritura devuelve los nombres de los campos con rol writable, en orden de
// declaración. Es la lista de campos que el cliente puede enviar.
func (t TablaCampos) CamposEscritura() []string {
	return proyectarCampos(t, func(c CampoObjeto) bool { return c.Writable })
}

// CamposInmutables devuelve los nombres de los campos con rol immutable, en orden
// de declaración. Son un subconjunto de CamposEscritura.
func (t TablaCampos) CamposInmutables() []string {
	return proyectarCampos(t, func(c CampoObjeto) bool { return c.Immutable })
}

// proyectarCampos devuelve los nombres de los campos que cumplen el predicado,
// conservando el orden de la tabla. Devuelve una lista no nula aunque no haya
// coincidencias.
func proyectarCampos(t TablaCampos, cumple func(CampoObjeto) bool) []string {
	nombres := make([]string, 0, len(t))
	for _, campo := range t {
		if cumple(campo) {
			nombres = append(nombres, campo.Name)
		}
	}
	return nombres
}
