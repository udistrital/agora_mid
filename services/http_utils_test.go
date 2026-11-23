package services

import "testing"

// TestUnirURL verifica que la unión conserva el prefijo de ruta y produce una
// sola barra, en las cuatro combinaciones de barra final/inicial.
func TestUnirURL(t *testing.T) {
	const esperado = "https://crud.example/api/v1/tercero/15"
	casos := []struct {
		base, endpoint string
	}{
		{"https://crud.example/api/v1/", "/tercero/15"},
		{"https://crud.example/api/v1/", "tercero/15"},
		{"https://crud.example/api/v1", "/tercero/15"},
		{"https://crud.example/api/v1", "tercero/15"},
	}
	for _, c := range casos {
		if got := unirURL(c.base, c.endpoint); got != esperado {
			t.Errorf("unirURL(%q, %q) = %q, se esperaba %q", c.base, c.endpoint, got, esperado)
		}
	}
}
