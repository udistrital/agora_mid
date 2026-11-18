package helper

// requeridosCreacionHijo lista, por nombre de relación hija, los campos que un hijo
// debe traer al crearse. Lista escrita a mano (solución sucia de F22). No la leen los
// sidecars del generador.
var requeridosCreacionHijo = map[string][]string{
	"Contacto":       {"ValorEspecifico", "Proposito", "TipoCanalContactoId"},
	"CuentaBancaria": {"TipoCuentaId", "NumeroCuenta", "EntidadBancariaId"},
}

// validarRequeridosCreacion valida que un hijo que se va a crear traiga sus campos
// requeridos. Un elemento con Id mayor que cero es una actualización y no se valida.
// Un Id nulo, cero o negativo cuenta como ausente, así que el elemento es una creación.
//
// Devuelve *ErrorEscritura con la ruta del campo faltante (por ejemplo
// "Contacto[0].Proposito") y nil si el elemento cumple o si la relación no tiene
// requeridos declarados.
func validarRequeridosCreacion(elem map[string]any, relacion, ruta string) error {
	if !esCreacionHijo(elem) {
		return nil
	}
	for _, campo := range requeridosCreacionHijo[relacion] {
		valor, ok := elem[campo]
		if !ok || valor == nil {
			return &ErrorEscritura{Path: ruta + "." + campo, Motivo: "campo requerido ausente en la creación"}
		}
		if texto, esTexto := valor.(string); esTexto && texto == "" {
			return &ErrorEscritura{Path: ruta + "." + campo, Motivo: "campo requerido ausente en la creación"}
		}
	}
	return nil
}

// esCreacionHijo indica si el elemento no identifica un registro existente: no trae
// Id, o su Id es nulo o no es un número mayor que cero.
func esCreacionHijo(elem map[string]any) bool {
	id, ok := elem["Id"]
	if !ok || id == nil {
		return true
	}
	switch n := id.(type) {
	case float64:
		return n <= 0
	case int:
		return n <= 0
	case int64:
		return n <= 0
	default:
		return true
	}
}
