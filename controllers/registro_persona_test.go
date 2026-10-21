package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/udistrital/agora_mid/config"
	"github.com/udistrital/agora_mid/responses"
)

// envelopeCrudo refleja la raíz OAS plana tal como se serializa en el cable,
// incluyendo si el bloque MetaData está presente o ausente.
type envelopeCrudo struct {
	Success  bool                `json:"Success"`
	Status   string              `json:"Status"`
	Message  string              `json:"Message"`
	Data     json.RawMessage     `json:"Data"`
	MetaData *responses.MetaData `json:"MetaData"`
}

// decodificarEnvelope valida la raíz plana y devuelve el envelope junto con el
// mapa de llaves de primer nivel para verificar presencia/ausencia de MetaData.
func decodificarEnvelope(t *testing.T, cuerpo []byte) (envelopeCrudo, map[string]json.RawMessage) {
	t.Helper()

	var env envelopeCrudo
	if err := json.Unmarshal(cuerpo, &env); err != nil {
		t.Fatalf("respuesta no es JSON válido: %v; cuerpo=%s", err, cuerpo)
	}

	var llaves map[string]json.RawMessage
	if err := json.Unmarshal(cuerpo, &llaves); err != nil {
		t.Fatalf("no se pudo leer llaves de primer nivel: %v", err)
	}

	for _, requerida := range []string{"Success", "Status", "Message", "Data"} {
		if _, ok := llaves[requerida]; !ok {
			t.Errorf("falta la llave obligatoria %q en la raíz plana; cuerpo=%s", requerida, cuerpo)
		}
	}

	return env, llaves
}

// TestConsultarErrorSerializaRaizPlana ejerce el controlador real por el verbo
// GET con tercero_id inválido: no requiere backends y verifica que la respuesta
// de error usa la raíz plana Success/Status/Message/Data y omite MetaData.
func TestConsultarErrorSerializaRaizPlana(t *testing.T) {
	handler := NewRegistroPersonaHandler(config.Config{})

	req := httptest.NewRequest(http.MethodGet, "/v1/registro/persona-natural?tercero_id=0", nil)
	rec := httptest.NewRecorder()

	handler.RegistrarNatural(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("código HTTP recibido: %d; esperado: %d", rec.Code, http.StatusBadRequest)
	}

	env, llaves := decodificarEnvelope(t, rec.Body.Bytes())

	if env.Success {
		t.Errorf("Success recibido: true; esperado: false")
	}
	if env.Status != "400" {
		t.Errorf("Status recibido: %q; esperado: %q", env.Status, "400")
	}
	if strings.TrimSpace(env.Message) == "" {
		t.Errorf("Message vacío en respuesta de error")
	}
	if _, ok := llaves["MetaData"]; ok {
		t.Errorf("MetaData debe omitirse cuando no hay avisos; cuerpo=%s", rec.Body.Bytes())
	}
}

// TestConsultarExitoSerializaRaizPlanaSinMetaData ejerce el controlador por GET
// contra backends simulados y verifica una respuesta exitosa con raíz plana y
// sin bloque MetaData.
func TestConsultarExitoSerializaRaizPlanaSinMetaData(t *testing.T) {
	// Backend de terceros: devuelve el tercero base para la consulta.
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Lista vacía: suficiente para que la consulta resuelva sin encontrar
		// datos específicos, devolviendo un resultado base sin error de
		// gateway. El objetivo del test es el envoltorio, no la lógica de
		// negocio del servicio.
		_, _ = w.Write([]byte(`{"Success":true,"Status":"200","Message":"ok","Data":[]}`))
	}))
	defer backend.Close()

	handler := NewRegistroPersonaHandler(config.Config{
		TercerosCRUDURL:        backend.URL,
		GestionPersonasCRUDURL: backend.URL,
		ParametrosCRUDURL:      backend.URL,
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/registro/persona-natural?tercero_id=25", nil)
	rec := httptest.NewRecorder()

	handler.RegistrarNatural(rec, req)

	env, llaves := decodificarEnvelope(t, rec.Body.Bytes())

	// El controlador siempre emite la raíz plana; verificamos su forma con
	// independencia de que la consulta encuentre o no datos.
	if env.Status == "" {
		t.Errorf("Status vacío en la raíz plana")
	}
	if strings.TrimSpace(env.Message) == "" {
		t.Errorf("Message vacío en la raíz plana")
	}
	if _, ok := llaves["MetaData"]; ok {
		t.Errorf("MetaData debe omitirse cuando no hay avisos; cuerpo=%s", rec.Body.Bytes())
	}
}

// TestEnvelopeConAvisosSerializa206YMetaData verifica, en la capa de respuesta
// que el controlador usará, que un envoltorio con avisos de expansión incluye
// MetaData.Errors y Status "206".
func TestEnvelopeConAvisosSerializa206YMetaData(t *testing.T) {
	avisos := []responses.RelationError{
		{
			Path:    "Tercero",
			Service: "terceros",
			Message: "relación no disponible, se conserva el recurso principal",
			Kind:    "relation",
		},
	}

	resp := responses.WithWarnings(true, "206", "Expansión parcial", map[string]any{"Id": 1}, avisos)

	cuerpo, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("no se pudo serializar la respuesta: %v", err)
	}

	env, llaves := decodificarEnvelope(t, cuerpo)

	if env.Status != "206" {
		t.Errorf("Status recibido: %q; esperado: %q", env.Status, "206")
	}
	if _, ok := llaves["MetaData"]; !ok {
		t.Fatalf("MetaData debe estar presente cuando hay avisos; cuerpo=%s", cuerpo)
	}
	if env.MetaData == nil || len(env.MetaData.Errors) != 1 {
		t.Fatalf("se esperaba exactamente un aviso en MetaData.Errors; recibido=%+v", env.MetaData)
	}

	aviso := env.MetaData.Errors[0]
	if aviso.Path != "Tercero" || aviso.Service != "terceros" || aviso.Kind != "relation" {
		t.Errorf("aviso con campos inesperados: %+v", aviso)
	}
	if strings.TrimSpace(aviso.Message) == "" {
		t.Errorf("Message del aviso vacío")
	}
}

// TestEnvelopeSinAvisosOmiteMetaData verifica que WithWarnings sin avisos no
// agrega el bloque MetaData, de forma consistente con New.
func TestEnvelopeSinAvisosOmiteMetaData(t *testing.T) {
	resp := responses.WithWarnings(true, "200", "ok", map[string]any{"Id": 1}, nil)

	cuerpo, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("no se pudo serializar la respuesta: %v", err)
	}

	_, llaves := decodificarEnvelope(t, cuerpo)
	if _, ok := llaves["MetaData"]; ok {
		t.Errorf("MetaData debe omitirse sin avisos; cuerpo=%s", cuerpo)
	}
}

// TestEnvelopeConLengthSerializaMetaDataLength verifica que un Data de tipo
// lista pueda reportar MetaData.Length (total de valores posibles) sin incluir
// Errors cuando no hay avisos.
func TestEnvelopeConLengthSerializaMetaDataLength(t *testing.T) {
	resp := responses.WithLength(true, "200", "ok", []map[string]any{{"Id": 1}, {"Id": 2}}, 57)

	cuerpo, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("no se pudo serializar la respuesta: %v", err)
	}

	env, llaves := decodificarEnvelope(t, cuerpo)

	if _, ok := llaves["MetaData"]; !ok {
		t.Fatalf("MetaData debe estar presente cuando hay Length; cuerpo=%s", cuerpo)
	}
	if env.MetaData == nil || env.MetaData.Length == nil {
		t.Fatalf("se esperaba MetaData.Length presente; recibido=%+v", env.MetaData)
	}
	if *env.MetaData.Length != 57 {
		t.Errorf("Length recibido: %d; esperado: %d", *env.MetaData.Length, 57)
	}
	if len(env.MetaData.Errors) != 0 {
		t.Errorf("Errors debe omitirse cuando no hay avisos; recibido=%+v", env.MetaData.Errors)
	}
	// Errors optional: su llave no debe aparecer dentro de MetaData sin avisos.
	var metaLlaves struct {
		MetaData map[string]json.RawMessage `json:"MetaData"`
	}
	if err := json.Unmarshal(cuerpo, &metaLlaves); err != nil {
		t.Fatalf("no se pudo leer llaves de MetaData: %v", err)
	}
	if _, ok := metaLlaves.MetaData["Errors"]; ok {
		t.Errorf("Errors debe omitirse del JSON cuando no hay avisos; cuerpo=%s", cuerpo)
	}
}

// TestEnvelopeConAvisosYLength verifica que avisos y Length coexistan en
// MetaData cuando ambos aplican.
func TestEnvelopeConAvisosYLength(t *testing.T) {
	avisos := []responses.RelationError{{Path: "Tercero", Service: "terceros", Message: "aviso", Kind: "relation"}}
	length := 3

	resp := responses.WithMetaData(true, "206", "parcial", []map[string]any{{"Id": 1}}, avisos, &length)

	cuerpo, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("no se pudo serializar la respuesta: %v", err)
	}

	env, _ := decodificarEnvelope(t, cuerpo)
	if env.MetaData == nil {
		t.Fatalf("MetaData debe estar presente")
	}
	if len(env.MetaData.Errors) != 1 {
		t.Errorf("se esperaba un aviso; recibido=%+v", env.MetaData.Errors)
	}
	if env.MetaData.Length == nil || *env.MetaData.Length != 3 {
		t.Errorf("se esperaba Length=3; recibido=%+v", env.MetaData.Length)
	}
}

// TestEnvelopeSinAvisosNiLengthOmiteMetaData verifica que MetaData se omita por
// completo cuando no hay ni avisos ni Length.
func TestEnvelopeSinAvisosNiLengthOmiteMetaData(t *testing.T) {
	resp := responses.WithMetaData(true, "200", "ok", map[string]any{"Id": 1}, nil, nil)

	cuerpo, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("no se pudo serializar la respuesta: %v", err)
	}

	_, llaves := decodificarEnvelope(t, cuerpo)
	if _, ok := llaves["MetaData"]; ok {
		t.Errorf("MetaData debe omitirse sin avisos ni Length; cuerpo=%s", cuerpo)
	}
}
