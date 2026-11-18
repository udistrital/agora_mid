package helper

import "testing"

// Crear un Contacto sin Proposito debe responder error con la ruta del campo.
func TestRequeridosHijoCreacionSinProposito(t *testing.T) {
	elem := map[string]any{
		"NombreContacto":      "Sin proposito",
		"ValorEspecifico":     "x@example.com",
		"TipoCanalContactoId": float64(1),
	}
	err := validarRequeridosCreacion(elem, "Contacto", "Contacto[0]")
	if err == nil {
		t.Fatal("se esperaba error por Proposito ausente")
	}
	e, ok := err.(*ErrorEscritura)
	if !ok {
		t.Fatalf("tipo de error inesperado: %T", err)
	}
	if e.Path != "Contacto[0].Proposito" {
		t.Errorf("ruta = %q, se esperaba Contacto[0].Proposito", e.Path)
	}
}

// Actualizar un Contacto por Id no exige sus campos requeridos (U05).
func TestRequeridosHijoActualizacionNoExigeCampos(t *testing.T) {
	elem := map[string]any{"Id": float64(1), "ValorEspecifico": "ana.nueva@example.com"}
	if err := validarRequeridosCreacion(elem, "Contacto", "Contacto[0]"); err != nil {
		t.Errorf("una actualización por Id no debe exigir campos: %v", err)
	}
}

// Un Id igual a 0 cuenta como ausente: el elemento es una creación.
func TestRequeridosHijoIdCeroEsCreacion(t *testing.T) {
	elem := map[string]any{"Id": float64(0), "ValorEspecifico": "x@example.com"}
	if err := validarRequeridosCreacion(elem, "Contacto", "Contacto[0]"); err == nil {
		t.Error("Id 0 debe tratarse como creación y exigir campos requeridos")
	}
}

// Crear una CuentaBancaria sin EntidadBancariaId debe responder error con esa ruta.
func TestRequeridosHijoCuentaSinEntidad(t *testing.T) {
	elem := map[string]any{"TipoCuentaId": float64(1), "NumeroCuenta": "123"}
	err := validarRequeridosCreacion(elem, "CuentaBancaria", "CuentaBancaria[0]")
	e, ok := err.(*ErrorEscritura)
	if !ok {
		t.Fatalf("se esperaba *ErrorEscritura, recibido %v", err)
	}
	if e.Path != "CuentaBancaria[0].EntidadBancariaId" {
		t.Errorf("ruta = %q", e.Path)
	}
}

// Un hijo completo pasa la validación.
func TestRequeridosHijoCompletoPasa(t *testing.T) {
	elem := map[string]any{
		"NombreContacto":      "Ana",
		"ValorEspecifico":     "ana@example.com",
		"Proposito":           "Notificaciones",
		"TipoCanalContactoId": float64(1),
	}
	if err := validarRequeridosCreacion(elem, "Contacto", "Contacto[0]"); err != nil {
		t.Errorf("un hijo completo no debe fallar: %v", err)
	}
}

// Una relación sin requeridos configurados no se valida.
func TestRequeridosHijoRelacionSinRequeridos(t *testing.T) {
	if err := validarRequeridosCreacion(map[string]any{}, "Otra", "Otra[0]"); err != nil {
		t.Errorf("una relación sin requeridos no debe fallar: %v", err)
	}
}
