package helper

import (
	"context"
	"errors"
	"testing"

	"github.com/udistrital/agora_mid/config"
)

// cfgNulos parte de cfgEscritura. La raíz declara Descripcion no nulable y
// Observacion nulable; TerceroId es una FK, que el motor trata como ausente cuando
// llega null. La relación Detalle declara Nota no nulable y TipoContribuyenteId como
// FK no nulable.
func cfgNulos() config.ServiceConfig {
	cfg := cfgEscritura()
	cfg.Write.WritableFields = append(cfg.Write.WritableFields, "Observacion")
	cfg.Write.NonNullableFields = []string{"Descripcion", "TerceroId"}
	for i := range cfg.Relations {
		if cfg.Relations[i].Name == "Detalle" {
			cfg.Relations[i].Write.NonNullableFields = []string{"Nota", "TipoContribuyenteId"}
		}
	}
	return cfg
}

// registroNulos devuelve un registro de resolvers que cuenta lecturas y escrituras.
func registroNulos(llamadas *[]registroMaterializacion, lecturas *int) WriteResolverRegistry {
	raiz := &resolverFalso{servicio: "raiz", siguienteID: 100, llamadas: llamadas, recuperaciones: lecturas}
	esp := map[string]WriteResolver{
		"detalle": &resolverFalso{servicio: "detalle", siguienteID: 200, llamadas: llamadas, recuperaciones: lecturas},
		"hijo":    &resolverFalso{servicio: "hijo", siguienteID: 300, llamadas: llamadas, recuperaciones: lecturas},
	}
	return NewWriteResolverRegistry(raiz, esp)
}

// null en un escalar escribible no nulable de la raíz responde ErrorEscritura antes de
// cualquier lectura o escritura.
func TestNulosRaizNoNulableRechazaAntesDeLeerOEscribir(t *testing.T) {
	var llamadas []registroMaterializacion
	var lecturas int
	body := map[string]any{"TerceroId": float64(25), "Descripcion": nil}

	_, err := Write(context.Background(), body, cfgNulos(), registroNulos(&llamadas, &lecturas), WriteOptions{})
	var ee *ErrorEscritura
	if !errors.As(err, &ee) {
		t.Fatalf("se esperaba *ErrorEscritura por null en campo no nulable; err=%v", err)
	}
	if ee.Path != "Descripcion" {
		t.Errorf("Path = %q, se esperaba Descripcion", ee.Path)
	}
	if lecturas != 0 || len(llamadas) != 0 {
		t.Errorf("no debe leer ni escribir ante un null inválido; lecturas=%d escrituras=%d", lecturas, len(llamadas))
	}
}

// null en un escalar escribible nulable se acepta: la validación no lo rechaza.
func TestNulosRaizNulableSeAcepta(t *testing.T) {
	var llamadas []registroMaterializacion
	var lecturas int
	body := map[string]any{"TerceroId": float64(25), "Descripcion": "x", "Observacion": nil}

	if _, err := Write(context.Background(), body, cfgNulos(), registroNulos(&llamadas, &lecturas), WriteOptions{}); err != nil {
		t.Fatalf("null en campo nulable no debe rechazarse; err=%v", err)
	}
}

// null en un escalar no nulable de una relación anidada responde ErrorEscritura con la
// ruta del campo anidado, antes de cualquier escritura.
func TestNulosRelacionAnidadaNoNulableRechaza(t *testing.T) {
	var llamadas []registroMaterializacion
	var lecturas int
	body := map[string]any{
		"TerceroId":   float64(25),
		"Descripcion": "x",
		"Detalle":     map[string]any{"Nota": nil},
	}

	_, err := Write(context.Background(), body, cfgNulos(), registroNulos(&llamadas, &lecturas), WriteOptions{})
	var ee *ErrorEscritura
	if !errors.As(err, &ee) {
		t.Fatalf("se esperaba *ErrorEscritura por null en Detalle.Nota; err=%v", err)
	}
	if ee.Path != "Detalle.Nota" {
		t.Errorf("Path = %q, se esperaba Detalle.Nota", ee.Path)
	}
	if lecturas != 0 || len(llamadas) != 0 {
		t.Errorf("no debe leer ni escribir ante un null inválido; lecturas=%d escrituras=%d", lecturas, len(llamadas))
	}
}

// null en una FK (sufijo Id) no nulable se trata como ausente y no se rechaza, en la
// raíz y en una relación anidada.
func TestNulosFKNoNulableSeTrataComoAusente(t *testing.T) {
	var llamadas []registroMaterializacion
	var lecturas int
	body := map[string]any{
		"TerceroId":   nil,
		"Descripcion": "x",
		"Detalle":     map[string]any{"Nota": "x", "TipoContribuyenteId": nil},
	}

	if _, err := Write(context.Background(), body, cfgNulos(), registroNulos(&llamadas, &lecturas), WriteOptions{}); err != nil {
		t.Fatalf("null en FK no debe rechazarse; err=%v", err)
	}
}
