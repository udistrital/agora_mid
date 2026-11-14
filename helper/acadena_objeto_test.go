package helper

import "testing"

// Una FK que llega como objeto con Id (CRUD con rel fk) debe compararse por su Id.
func TestACadenaDeObjetoConIdUsaElId(t *testing.T) {
	obj := map[string]any{"Id": float64(1), "TipoProveedor": float64(1)}
	if got := aCadena(obj); got != "1" {
		t.Errorf("aCadena(objeto con Id) = %q; se esperaba \"1\"", got)
	}
}
