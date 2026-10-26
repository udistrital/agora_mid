package helper

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/udistrital/agora_mid/config"
	"github.com/udistrital/agora_mid/responses"
)

// registroMaterializacion captura una llamada al resolver para poder verificar el
// payload aplanado, las llaves naturales y la propagación de llave recibidos.
type registroMaterializacion struct {
	servicio string
	payload  map[string]any
	keys     []string
}

// resolverFalso es un WriteResolver de prueba que asigna Ids secuenciales por
// servicio y registra cada materialización. Puede forzar un error para un
// servicio dado.
type resolverFalso struct {
	servicio    string
	siguienteID float64
	llamadas    *[]registroMaterializacion
	fallarCon   error
}

func (r *resolverFalso) Materialize(
	_ context.Context,
	rel config.RelationConfig,
	payload map[string]any,
	keys []string,
) (map[string]any, error) {
	nombre := rel.RelatedService
	if nombre == "" {
		nombre = r.servicio
	}
	*r.llamadas = append(*r.llamadas, registroMaterializacion{
		servicio: nombre,
		payload:  clonar(payload),
		keys:     append([]string{}, keys...),
	})
	if r.fallarCon != nil {
		return nil, r.fallarCon
	}
	salida := clonar(payload)
	// Respetar un Id ya propagado (shared_pk); si no, asignar uno nuevo.
	if _, ok := salida["Id"]; !ok {
		salida["Id"] = r.siguienteID
	}
	return salida, nil
}

func clonar(m map[string]any) map[string]any {
	c := make(map[string]any, len(m))
	for k, v := range m {
		c[k] = v
	}
	return c
}

// cfgEscritura construye una ServiceConfig con dos relaciones: una hija por FK
// (cardinalidad many) y una hija por PK compartida (cardinalidad one).
func cfgEscritura() config.ServiceConfig {
	return config.ServiceConfig{
		Service:         "raiz",
		MaxDepth:        4,
		OnRelationError: config.OnRelationErrorNotify,
		MaxRelatedIDs:   100,
		Write: &config.WriteConfig{
			WritableFields: []string{"TerceroId", "Descripcion"},
			NaturalKey:     []string{"TerceroId"},
			Policy:         config.WriteVerbPolicy{Creatable: true, Updatable: true},
		},
		Relations: []config.RelationConfig{
			{
				Name:           "Detalle",
				RelatedService: "detalle",
				MappedName:     "Detalle",
				RelatedField:   "Id",
				Cardinality:    config.CardinalityOne,
				Write: &config.WriteConfig{
					WritableFields: []string{"Nota", "TipoContribuyenteId"},
					KeyPropagation: config.KeyPropagationSharedPK,
					Policy:         config.WriteVerbPolicy{Creatable: true, Updatable: true},
				},
			},
			{
				Name:           "Hijo",
				RelatedService: "hijo",
				MappedName:     "Hijo",
				RelatedField:   "PadreId",
				Cardinality:    config.CardinalityMany,
				Write: &config.WriteConfig{
					WritableFields: []string{"Valor"},
					KeyPropagation: config.KeyPropagationFK,
					Policy:         config.WriteVerbPolicy{Creatable: true, Updatable: true, Deletable: true},
				},
			},
		},
	}
}

func registryFalso(llamadas *[]registroMaterializacion, fallar map[string]error) WriteResolverRegistry {
	raiz := &resolverFalso{servicio: "raiz", siguienteID: 100, llamadas: llamadas, fallarCon: fallar["raiz"]}
	esp := map[string]WriteResolver{
		"detalle": &resolverFalso{servicio: "detalle", siguienteID: 200, llamadas: llamadas, fallarCon: fallar["detalle"]},
		"hijo":    &resolverFalso{servicio: "hijo", siguienteID: 300, llamadas: llamadas, fallarCon: fallar["hijo"]},
	}
	return NewWriteResolverRegistry(raiz, esp)
}

// La materialización respeta el orden topológico (raíz antes que hijos) y propaga
// la llave del padre por FK y por PK compartida.
func TestWritePropagacionLlave(t *testing.T) {
	var llamadas []registroMaterializacion
	registry := registryFalso(&llamadas, nil)

	body := map[string]any{
		"TerceroId":   float64(25),
		"Descripcion": "principal",
		"Detalle": map[string]any{
			"Nota":              "x",
			"TipoContribuyente": map[string]any{"Id": float64(2)},
		},
		"Hijo": []any{
			map[string]any{"Valor": "a"},
			map[string]any{"Valor": "b"},
		},
	}

	res, err := Write(context.Background(), body, cfgEscritura(), registry, WriteOptions{Verb: WriteVerbCreate})
	if err != nil {
		t.Fatalf("Write no debe fallar: %v", err)
	}

	// El primer resolver invocado debe ser la raíz (orden topológico).
	if len(llamadas) == 0 || llamadas[0].servicio != "raiz" {
		t.Fatalf("la raíz debe materializarse primero; llamadas=%+v", llamadas)
	}

	// shared_pk: el detalle recibe el Id de la raíz (100) como su propio Id,
	// preservando su tipo numérico (no una cadena de texto).
	detalle, _ := res.Data["Detalle"].(map[string]any)
	if detalle == nil {
		t.Fatalf("falta el Detalle materializado; data=%+v", res.Data)
	}
	idDetalle, ok := detalle["Id"].(float64)
	if !ok {
		t.Errorf("Detalle.Id (shared_pk) debe ser numérico, no %T; detalle=%+v", detalle["Id"], detalle)
	} else if idDetalle != 100 {
		t.Errorf("Detalle (shared_pk) debe heredar el Id de la raíz 100; detalle=%+v", detalle)
	}

	// fk: cada hijo recibe PadreId = Id de la raíz (100), preservando el tipo
	// numérico; su propio Id materializado (300) también es numérico.
	hijos, _ := res.Data["Hijo"].([]any)
	if len(hijos) != 2 {
		t.Fatalf("se esperaban 2 hijos materializados; recibidos=%d", len(hijos))
	}
	for i, h := range hijos {
		hm, _ := h.(map[string]any)
		padreID, ok := hm["PadreId"].(float64)
		if !ok {
			t.Errorf("hijo %d: PadreId (fk) debe ser numérico, no %T; hijo=%+v", i, hm["PadreId"], hm)
		} else if padreID != 100 {
			t.Errorf("hijo %d debe tener PadreId=100 (fk); hijo=%+v", i, hm)
		}
		idHijo, ok := hm["Id"].(float64)
		if !ok {
			t.Errorf("hijo %d: Id debe ser numérico, no %T; hijo=%+v", i, hm["Id"], hm)
		} else if idHijo != 300 {
			t.Errorf("hijo %d debe tener Id materializado 300; hijo=%+v", i, hm)
		}
	}
}

// El objeto de parámetro anidado {X:{Id:n}} se aplana a la FK XId:n y la clave
// natural se extrae del payload aplanado para el upsert idempotente.
func TestWriteAplanadoParametroYClaveNatural(t *testing.T) {
	var llamadas []registroMaterializacion
	registry := registryFalso(&llamadas, nil)

	body := map[string]any{
		"TerceroId": float64(25),
		"Detalle": map[string]any{
			"Nota":              "x",
			"TipoContribuyente": map[string]any{"Id": float64(2)},
		},
		"Hijo": []any{map[string]any{"Valor": "a"}},
	}

	res, err := Write(context.Background(), body, cfgEscritura(), registry, WriteOptions{})
	if err != nil {
		t.Fatalf("Write no debe fallar: %v", err)
	}
	padreId := res.Data["Id"].(float64)
	if padreId == 0 {
		t.Fatalf("la raíz debe materializar su Id; data=%+v", res.Data)
	}

	// Llamada de la raíz: clave natural TerceroId=25.
	var raiz, detalle, hijo *registroMaterializacion
	for i := range llamadas {
		switch llamadas[i].servicio {
		case "raiz":
			raiz = &llamadas[i]
		case "detalle":
			detalle = &llamadas[i]
		case "hijo":
			hijo = &llamadas[i]
		}
	}
	if raiz == nil || len(raiz.keys) != 1 || raiz.keys[0] != "25" {
		t.Errorf("la raíz debe recibir la clave natural [25]; raiz=%+v", raiz)
	}

	if detalle == nil {
		t.Fatalf("el detalle debe materializarse")
	}
	// El detalle recibe el Id de la raíz (shared_pk).
	if aCadena(detalle.payload["Id"]) != aCadena(padreId) {
		t.Errorf("el detalle debe heredar el Id de la raíz (shared_pk); payload=%+v", detalle.payload)
	}
	// El parámetro anidado se aplanó a TipoContribuyenteId=2.
	if aCadena(detalle.payload["TipoContribuyenteId"]) != "2" {
		t.Errorf("el parámetro anidado debe aplanarse a TipoContribuyenteId=2; payload=%+v", detalle.payload)
	}
	if _, sigueAnidado := detalle.payload["TipoContribuyente"]; sigueAnidado {
		t.Errorf("el objeto anidado no debe pasar al resolver; payload=%+v", detalle.payload)
	}

	if hijo == nil {
		t.Fatalf("el hijo debe materializarse")
	}
	// El hijo recibe PadreId=raíz.Id (fk).
	if aCadena(hijo.payload["PadreId"]) != aCadena(padreId) {
		t.Errorf("el hijo debe recibir PadreId=raíz.Id (fk); payload=%+v", hijo.payload)
	}
}

// Un campo no escribible se rechaza con *ErrorEscritura y no materializa nada.
func TestWriteRechazoCampoNoEscribible(t *testing.T) {
	var llamadas []registroMaterializacion
	registry := registryFalso(&llamadas, nil)

	body := map[string]any{
		"TerceroId":      float64(25),
		"CampoProhibido": "x",
	}

	_, err := Write(context.Background(), body, cfgEscritura(), registry, WriteOptions{})
	var ee *ErrorEscritura
	if !errors.As(err, &ee) {
		t.Fatalf("se esperaba *ErrorEscritura; err=%v", err)
	}
	if len(llamadas) != 0 {
		t.Errorf("no debe materializarse nada ante un cuerpo inválido; llamadas=%+v", llamadas)
	}
}

// Un intento de modificar un campo inmutable en actualización se rechaza con 400
// y no materializa nada.
func TestWriteRechazoInmutableEnUpdate(t *testing.T) {
	var llamadas []registroMaterializacion
	registry := registryFalso(&llamadas, nil)

	cfg := cfgEscritura()
	cfg.Write.ImmutableFields = []string{"TerceroId"}

	body := map[string]any{"TerceroId": float64(25)}

	_, err := Write(context.Background(), body, cfg, registry, WriteOptions{Verb: WriteVerbUpdate})
	var ee *ErrorEscritura
	if !errors.As(err, &ee) {
		t.Fatalf("se esperaba *ErrorEscritura por campo inmutable; err=%v", err)
	}
	if len(llamadas) != 0 {
		t.Errorf("no debe materializarse nada; llamadas=%+v", llamadas)
	}
}

// En modo tolerante (notify), el fallo de una relación conserva lo ya
// materializado y agrega un aviso no sensible.
func TestWriteTolerante(t *testing.T) {
	var llamadas []registroMaterializacion
	registry := registryFalso(&llamadas, map[string]error{"hijo": errors.New("crud caído")})

	body := map[string]any{
		"TerceroId": float64(25),
		"Hijo":      []any{map[string]any{"Valor": "a"}},
	}

	res, err := Write(context.Background(), body, cfgEscritura(), registry, WriteOptions{OnRelationError: config.OnRelationErrorNotify})
	if err != nil {
		t.Fatalf("modo tolerante no debe devolver error fatal: %v", err)
	}
	if aCadena(res.Data["Id"]) != "100" {
		t.Errorf("la raíz debe conservar su Id materializado; data=%+v", res.Data)
	}
	if len(res.Warnings) != 1 || res.Warnings[0].Kind != KindRelation {
		t.Fatalf("se esperaba un aviso relation; warnings=%+v", res.Warnings)
	}
	w := res.Warnings[0]
	if w.Path != "Hijo[0]" || w.Service != "hijo" {
		t.Errorf("el aviso debe identificar la ruta lógica del elemento (Hijo[0]) y el servicio hijo; aviso=%+v", w)
	}
	// El aviso no debe contener el mensaje interno del CRUD.
	if contieneTexto(w.Message, "crud caído") {
		t.Errorf("el aviso no debe exponer el error interno del CRUD; aviso=%+v", w)
	}
}

// En modo estricto (fail), el fallo de una relación aborta la cascada con un
// error fatal que el controlador mapea a 502/504.
func TestWriteEstricto(t *testing.T) {
	var llamadas []registroMaterializacion
	registry := registryFalso(&llamadas, map[string]error{"hijo": errors.New("crud caído")})

	body := map[string]any{
		"TerceroId": float64(25),
		"Hijo":      []any{map[string]any{"Valor": "a"}},
	}

	_, err := Write(context.Background(), body, cfgEscritura(), registry, WriteOptions{OnRelationError: config.OnRelationErrorFail})
	if !EsErrorFatalEscritura(err) {
		t.Fatalf("modo estricto debe devolver un error fatal de escritura; err=%v", err)
	}
	if contieneTexto(err.Error(), "crud caído") {
		t.Errorf("el error fatal no debe exponer el mensaje interno del CRUD; err=%v", err)
	}
}

func contieneTexto(s, sub string) bool {
	return strings.Contains(s, sub)
}

// cfgEscrituraInbound construye una ServiceConfig cuya raíz referencia una
// entidad por FK ascendente (fk_inbound): la raíz declara la relación Referida
// con local_field RefId; la entidad referenciada es dueña de su propio Id y debe
// materializarse antes que la raíz, propagando su Id a raiz.RefId.
func cfgEscrituraInbound() config.ServiceConfig {
	return config.ServiceConfig{
		Service:         "raiz",
		MaxDepth:        4,
		OnRelationError: config.OnRelationErrorNotify,
		MaxRelatedIDs:   100,
		Write: &config.WriteConfig{
			// RefId es la FK local que la raíz recibe de la entidad referida.
			WritableFields: []string{"RefId", "Descripcion"},
			Policy:         config.WriteVerbPolicy{Creatable: true, Updatable: true},
		},
		Relations: []config.RelationConfig{
			{
				Name:           "Referida",
				RelatedService: "referida",
				MappedName:     "Referida",
				Direction:      config.DirectionLocalToRelated,
				LocalField:     "RefId",
				RelatedField:   "Id",
				Cardinality:    config.CardinalityOne,
				Write: &config.WriteConfig{
					WritableFields: []string{"Nombre"},
					KeyPropagation: config.KeyPropagationFKInbound,
					Policy:         config.WriteVerbPolicy{Creatable: true, Updatable: true},
				},
			},
		},
	}
}

// Propagación fk_inbound: la entidad referenciada se materializa ANTES que la
// raíz y su Id se propaga hacia arriba a la FK local de la raíz (RefId).
func TestWriteFKInbound(t *testing.T) {
	var llamadas []registroMaterializacion
	raiz := &resolverFalso{servicio: "raiz", siguienteID: 100, llamadas: &llamadas}
	esp := map[string]WriteResolver{
		"referida": &resolverFalso{servicio: "referida", siguienteID: 200, llamadas: &llamadas},
	}
	registry := NewWriteResolverRegistry(raiz, esp)

	body := map[string]any{
		"Descripcion": "principal",
		"Referida":    map[string]any{"Nombre": "Ana"},
	}

	res, err := Write(context.Background(), body, cfgEscrituraInbound(), registry, WriteOptions{Verb: WriteVerbCreate})
	if err != nil {
		t.Fatalf("Write con fk_inbound no debe fallar: %v", err)
	}

	// La entidad referenciada se materializa primero (antes que la raíz).
	if len(llamadas) < 2 {
		t.Fatalf("se esperaban al menos 2 materializaciones; llamadas=%+v", llamadas)
	}
	if llamadas[0].servicio != "referida" {
		t.Errorf("la entidad fk_inbound debe materializarse ANTES que la raíz; orden=%v",
			[]string{llamadas[0].servicio, llamadas[1].servicio})
	}
	if llamadas[1].servicio != "raiz" {
		t.Errorf("la raíz debe materializarse después de la entidad referida; orden=%v",
			[]string{llamadas[0].servicio, llamadas[1].servicio})
	}

	// La raíz recibe RefId = Id de la entidad referenciada (200), preservando el
	// tipo numérico (no una cadena de texto).
	refID, ok := res.Data["RefId"].(float64)
	if !ok {
		t.Errorf("RefId (fk_inbound) debe ser numérico, no %T; data=%+v", res.Data["RefId"], res.Data)
	} else if refID != 200 {
		t.Errorf("la raíz debe recibir RefId=200 propagado hacia arriba; data=%+v", res.Data)
	}
	// La raíz materializó su propio Id (100), distinto del de la entidad referida.
	if aCadena(res.Data["Id"]) != "100" {
		t.Errorf("la raíz debe materializar su propio Id=100; data=%+v", res.Data)
	}
	// La entidad referenciada queda anidada con su Id materializado.
	referida, _ := res.Data["Referida"].(map[string]any)
	if referida == nil || aCadena(referida["Id"]) != "200" {
		t.Errorf("la entidad referida debe quedar anidada con Id=200; referida=%+v", referida)
	}
	// La raíz, al materializarse, ya llevaba la FK resuelta en su payload, con su
	// tipo numérico.
	payloadRef, ok := llamadas[1].payload["RefId"].(float64)
	if !ok {
		t.Errorf("RefId en el payload de la raíz debe ser numérico, no %T; payload=%+v",
			llamadas[1].payload["RefId"], llamadas[1].payload)
	} else if payloadRef != 200 {
		t.Errorf("la raíz debe materializarse con RefId=200 ya resuelto; payload=%+v", llamadas[1].payload)
	}
}

// Un campo derivado relacional reenviado (la FK descendente PadreId del hijo) no
// se rechaza como campo desconocido en la validación previa, aunque no esté en
// writable_fields del hijo.
func TestWriteValidacionReconoceCampoDerivadoFK(t *testing.T) {
	var llamadas []registroMaterializacion
	registry := registryFalso(&llamadas, nil)

	body := map[string]any{
		"TerceroId": float64(25),
		// PadreId es la FK derivada del hijo (fk -> related_field). Coincide con el
		// Id que la raíz materializará (100), así que no genera conflicto.
		"Hijo": []any{map[string]any{"Valor": "a", "PadreId": float64(100)}},
	}

	_, err := Write(context.Background(), body, cfgEscritura(), registry, WriteOptions{Verb: WriteVerbCreate})
	if err != nil {
		t.Fatalf("un campo derivado reenviado no debe rechazarse: %v", err)
	}
}

// Un campo derivado relacional en null se trata como ausente: no se rechaza y el
// motor lo completa con la llave autoritativa del padre.
func TestWriteValidacionCampoDerivadoNullEsAusencia(t *testing.T) {
	var llamadas []registroMaterializacion
	registry := registryFalso(&llamadas, nil)

	body := map[string]any{
		"TerceroId": float64(25),
		"Hijo":      []any{map[string]any{"Valor": "a", "PadreId": nil}},
	}

	res, err := Write(context.Background(), body, cfgEscritura(), registry, WriteOptions{Verb: WriteVerbCreate})
	if err != nil {
		t.Fatalf("un campo derivado null no debe rechazarse: %v", err)
	}
	hijos, _ := res.Data["Hijo"].([]any)
	if len(hijos) != 1 {
		t.Fatalf("se esperaba 1 hijo materializado; data=%+v", res.Data)
	}
	hm, _ := hijos[0].(map[string]any)
	padreID, ok := hm["PadreId"].(float64)
	if !ok || padreID != 100 {
		t.Errorf("PadreId null debe completarse con la llave autoritativa 100; hijo=%+v", hm)
	}
}

// Un campo verdaderamente desconocido (no derivado ni escribible) se rechaza con
// *ErrorEscritura y no materializa nada, incluso junto a un derivado válido.
func TestWriteValidacionRechazaDesconocidoJuntoADerivado(t *testing.T) {
	var llamadas []registroMaterializacion
	registry := registryFalso(&llamadas, nil)

	body := map[string]any{
		"TerceroId": float64(25),
		"Hijo": []any{map[string]any{
			"Valor":    "a",
			"PadreId":  float64(100), // derivado válido
			"Fantasma": "x",          // desconocido
		}},
	}

	_, err := Write(context.Background(), body, cfgEscritura(), registry, WriteOptions{Verb: WriteVerbCreate})
	var ee *ErrorEscritura
	if !errors.As(err, &ee) {
		t.Fatalf("se esperaba *ErrorEscritura por campo desconocido; err=%v", err)
	}
	if len(llamadas) != 0 {
		t.Errorf("no debe materializarse nada ante un cuerpo inválido; llamadas=%+v", llamadas)
	}
}

// --- Matriz de conciliación de campos derivados (fk, shared_pk, fk_inbound) ---
//
// La matriz común a los tres modos:
//   ausente       -> completa con la llave autoritativa, sin aviso
//   null          -> completa con la llave autoritativa, sin aviso
//   coincide      -> conserva, sin aviso
//   difiere + nuevo    -> canoniza + aviso field_normalization (206)
//   difiere + existente -> *ErrorEscritura (400), sin materializar la operación incompatible

// hijoFK devuelve el PadreId (fk) del primer hijo materializado.
func primerHijo(t *testing.T, res WriteResult) map[string]any {
	t.Helper()
	hijos, _ := res.Data["Hijo"].([]any)
	if len(hijos) != 1 {
		t.Fatalf("se esperaba 1 hijo materializado; data=%+v", res.Data)
	}
	hm, _ := hijos[0].(map[string]any)
	return hm
}

func avisosNormalizacion(res WriteResult) int {
	n := 0
	for _, w := range res.Warnings {
		if w.Kind == KindFieldNormalization {
			n++
		}
	}
	return n
}

// fk: PadreId ausente se completa con la llave autoritativa (100) sin aviso.
func TestWriteFKDerivadoAusente(t *testing.T) {
	var ll []registroMaterializacion
	body := map[string]any{"TerceroId": float64(25), "Hijo": []any{map[string]any{"Valor": "a"}}}
	res, err := Write(context.Background(), body, cfgEscritura(), registryFalso(&ll, nil), WriteOptions{})
	if err != nil {
		t.Fatalf("no debe fallar: %v", err)
	}
	if h := primerHijo(t, res); aCadena(h["PadreId"]) != "100" {
		t.Errorf("PadreId ausente debe completarse con 100; hijo=%+v", h)
	}
	if avisosNormalizacion(res) != 0 {
		t.Errorf("no debe haber aviso al completar un campo ausente; warnings=%+v", res.Warnings)
	}
}

// fk: PadreId null se trata como ausente, se completa con 100, sin aviso.
func TestWriteFKDerivadoNull(t *testing.T) {
	var ll []registroMaterializacion
	body := map[string]any{"TerceroId": float64(25), "Hijo": []any{map[string]any{"Valor": "a", "PadreId": nil}}}
	res, err := Write(context.Background(), body, cfgEscritura(), registryFalso(&ll, nil), WriteOptions{})
	if err != nil {
		t.Fatalf("no debe fallar: %v", err)
	}
	if h := primerHijo(t, res); aCadena(h["PadreId"]) != "100" {
		t.Errorf("PadreId null debe completarse con 100; hijo=%+v", h)
	}
	if avisosNormalizacion(res) != 0 {
		t.Errorf("null no debe producir aviso; warnings=%+v", res.Warnings)
	}
}

// fk: PadreId coincidente con la llave autoritativa se conserva, sin aviso, y su
// tipo numérico se preserva.
func TestWriteFKDerivadoCoincide(t *testing.T) {
	var ll []registroMaterializacion
	body := map[string]any{"TerceroId": float64(25), "Hijo": []any{map[string]any{"Valor": "a", "PadreId": float64(100)}}}
	res, err := Write(context.Background(), body, cfgEscritura(), registryFalso(&ll, nil), WriteOptions{})
	if err != nil {
		t.Fatalf("no debe fallar: %v", err)
	}
	h := primerHijo(t, res)
	if v, ok := h["PadreId"].(float64); !ok || v != 100 {
		t.Errorf("PadreId coincidente debe conservarse numérico 100; hijo=%+v", h)
	}
	if avisosNormalizacion(res) != 0 {
		t.Errorf("coincidir no debe producir aviso; warnings=%+v", res.Warnings)
	}
}

// fk: PadreId diferente en una entidad NUEVA (sin Id) se canoniza con la llave
// autoritativa y produce un aviso field_normalization.
func TestWriteFKDerivadoDiferenteNuevo(t *testing.T) {
	var ll []registroMaterializacion
	body := map[string]any{"TerceroId": float64(25), "Hijo": []any{map[string]any{"Valor": "a", "PadreId": float64(999)}}}
	res, err := Write(context.Background(), body, cfgEscritura(), registryFalso(&ll, nil), WriteOptions{})
	if err != nil {
		t.Fatalf("una entidad nueva con llave desactualizada no debe fallar: %v", err)
	}
	h := primerHijo(t, res)
	if aCadena(h["PadreId"]) != "100" {
		t.Errorf("PadreId diferente en entidad nueva debe canonizarse a 100; hijo=%+v", h)
	}
	if avisosNormalizacion(res) != 1 {
		t.Fatalf("se esperaba 1 aviso field_normalization; warnings=%+v", res.Warnings)
	}
	w := res.Warnings[0]
	if w.Kind != KindFieldNormalization || w.Path != "Hijo[0]" {
		t.Errorf("aviso inesperado; w=%+v", w)
	}
	// El aviso no debe exponer valores reenviados ni autoritativos.
	if contieneTexto(w.Message, "999") || contieneTexto(w.Message, "100") {
		t.Errorf("el aviso no debe exponer valores; w=%+v", w)
	}
}

// fk: PadreId diferente en una entidad EXISTENTE (con Id) es un conflicto:
// *ErrorEscritura y no se materializa la operación incompatible.
func TestWriteFKDerivadoDiferenteExistente(t *testing.T) {
	var ll []registroMaterializacion
	body := map[string]any{"TerceroId": float64(25), "Hijo": []any{map[string]any{"Id": float64(555), "Valor": "a", "PadreId": float64(999)}}}
	_, err := Write(context.Background(), body, cfgEscritura(), registryFalso(&ll, nil), WriteOptions{})
	var ee *ErrorEscritura
	if !errors.As(err, &ee) {
		t.Fatalf("se esperaba *ErrorEscritura por conflicto de entidad existente; err=%v", err)
	}
	// No debe haberse materializado el hijo en conflicto.
	for _, c := range ll {
		if c.servicio == "hijo" {
			t.Errorf("no debe materializarse el hijo en conflicto; llamadas=%+v", ll)
		}
	}
}

// shared_pk: Detalle.Id ausente hereda el Id del padre (100), sin aviso.
func TestWriteSharedPKAusente(t *testing.T) {
	var ll []registroMaterializacion
	body := map[string]any{"TerceroId": float64(25), "Detalle": map[string]any{"Nota": "x"}}
	res, err := Write(context.Background(), body, cfgEscritura(), registryFalso(&ll, nil), WriteOptions{})
	if err != nil {
		t.Fatalf("no debe fallar: %v", err)
	}
	d, _ := res.Data["Detalle"].(map[string]any)
	if v, ok := d["Id"].(float64); !ok || v != 100 {
		t.Errorf("Detalle.Id ausente debe heredar 100 numérico; detalle=%+v", d)
	}
	if avisosNormalizacion(res) != 0 {
		t.Errorf("no debe haber aviso; warnings=%+v", res.Warnings)
	}
}

// shared_pk: Detalle.Id coincidente con el del padre se conserva, sin conflicto
// ni aviso.
func TestWriteSharedPKCoincide(t *testing.T) {
	var ll []registroMaterializacion
	body := map[string]any{"TerceroId": float64(25), "Detalle": map[string]any{"Id": float64(100), "Nota": "x"}}
	res, err := Write(context.Background(), body, cfgEscritura(), registryFalso(&ll, nil), WriteOptions{})
	if err != nil {
		t.Fatalf("coincidir no debe fallar: %v", err)
	}
	if avisosNormalizacion(res) != 0 {
		t.Errorf("coincidir no debe producir aviso; warnings=%+v", res.Warnings)
	}
}

// shared_pk: Detalle.Id distinto del padre es un conflicto de entidad existente
// (el Id propio es a la vez el campo derivado): *ErrorEscritura.
func TestWriteSharedPKDiferenteExistente(t *testing.T) {
	var ll []registroMaterializacion
	body := map[string]any{"TerceroId": float64(25), "Detalle": map[string]any{"Id": float64(777), "Nota": "x"}}
	_, err := Write(context.Background(), body, cfgEscritura(), registryFalso(&ll, nil), WriteOptions{})
	var ee *ErrorEscritura
	if !errors.As(err, &ee) {
		t.Fatalf("se esperaba *ErrorEscritura por Id de shared_pk divergente; err=%v", err)
	}
	for _, c := range ll {
		if c.servicio == "detalle" {
			t.Errorf("no debe materializarse el detalle en conflicto; llamadas=%+v", ll)
		}
	}
}

// registryInbound construye el registro de la configuración fk_inbound con Ids
// secuenciales (raíz 100, referida 200).
func registryInbound(ll *[]registroMaterializacion) WriteResolverRegistry {
	raiz := &resolverFalso{servicio: "raiz", siguienteID: 100, llamadas: ll}
	esp := map[string]WriteResolver{"referida": &resolverFalso{servicio: "referida", siguienteID: 200, llamadas: ll}}
	return NewWriteResolverRegistry(raiz, esp)
}

// fk_inbound: RefId ausente se completa con el Id de la entidad referenciada
// (200), sin aviso.
func TestWriteInboundAusente(t *testing.T) {
	var ll []registroMaterializacion
	body := map[string]any{"Descripcion": "p", "Referida": map[string]any{"Nombre": "Ana"}}
	res, err := Write(context.Background(), body, cfgEscrituraInbound(), registryInbound(&ll), WriteOptions{})
	if err != nil {
		t.Fatalf("no debe fallar: %v", err)
	}
	if aCadena(res.Data["RefId"]) != "200" {
		t.Errorf("RefId ausente debe completarse con 200; data=%+v", res.Data)
	}
	if avisosNormalizacion(res) != 0 {
		t.Errorf("no debe haber aviso; warnings=%+v", res.Warnings)
	}
}

// fk_inbound: RefId null se trata como ausente y se completa con 200, sin aviso.
func TestWriteInboundNull(t *testing.T) {
	var ll []registroMaterializacion
	body := map[string]any{"Descripcion": "p", "RefId": nil, "Referida": map[string]any{"Nombre": "Ana"}}
	res, err := Write(context.Background(), body, cfgEscrituraInbound(), registryInbound(&ll), WriteOptions{})
	if err != nil {
		t.Fatalf("no debe fallar: %v", err)
	}
	if aCadena(res.Data["RefId"]) != "200" {
		t.Errorf("RefId null debe completarse con 200; data=%+v", res.Data)
	}
	if avisosNormalizacion(res) != 0 {
		t.Errorf("null no debe producir aviso; warnings=%+v", res.Warnings)
	}
}

// fk_inbound: RefId coincidente con el Id de la entidad referenciada se conserva,
// sin aviso, numérico.
func TestWriteInboundCoincide(t *testing.T) {
	var ll []registroMaterializacion
	body := map[string]any{"Descripcion": "p", "RefId": float64(200), "Referida": map[string]any{"Nombre": "Ana"}}
	res, err := Write(context.Background(), body, cfgEscrituraInbound(), registryInbound(&ll), WriteOptions{})
	if err != nil {
		t.Fatalf("coincidir no debe fallar: %v", err)
	}
	if v, ok := res.Data["RefId"].(float64); !ok || v != 200 {
		t.Errorf("RefId coincidente debe conservarse numérico 200; data=%+v", res.Data)
	}
	if avisosNormalizacion(res) != 0 {
		t.Errorf("coincidir no debe producir aviso; warnings=%+v", res.Warnings)
	}
}

// fk_inbound: RefId diferente en un nodo NUEVO (sin Id propio) se canoniza a 200
// y produce aviso field_normalization.
func TestWriteInboundDiferenteNuevo(t *testing.T) {
	var ll []registroMaterializacion
	body := map[string]any{"Descripcion": "p", "RefId": float64(999), "Referida": map[string]any{"Nombre": "Ana"}}
	res, err := Write(context.Background(), body, cfgEscrituraInbound(), registryInbound(&ll), WriteOptions{})
	if err != nil {
		t.Fatalf("nodo nuevo con FK local desactualizada no debe fallar: %v", err)
	}
	if aCadena(res.Data["RefId"]) != "200" {
		t.Errorf("RefId diferente en nodo nuevo debe canonizarse a 200; data=%+v", res.Data)
	}
	if avisosNormalizacion(res) != 1 {
		t.Errorf("se esperaba 1 aviso field_normalization; warnings=%+v", res.Warnings)
	}
}

// fk_inbound: RefId diferente en un nodo EXISTENTE (con Id propio) es conflicto:
// *ErrorEscritura y no se materializa la raíz en conflicto.
func TestWriteInboundDiferenteExistente(t *testing.T) {
	var ll []registroMaterializacion
	body := map[string]any{"Id": float64(100), "Descripcion": "p", "RefId": float64(999), "Referida": map[string]any{"Nombre": "Ana"}}
	_, err := Write(context.Background(), body, cfgEscrituraInbound(), registryInbound(&ll), WriteOptions{Verb: WriteVerbUpdate})
	var ee *ErrorEscritura
	if !errors.As(err, &ee) {
		t.Fatalf("se esperaba *ErrorEscritura por conflicto fk_inbound de nodo existente; err=%v", err)
	}
	// La raíz en conflicto no debe materializarse (solo pudo materializarse la
	// entidad referenciada en el pre-pass idempotente).
	for _, c := range ll {
		if c.servicio == "raiz" {
			t.Errorf("no debe materializarse la raíz en conflicto; llamadas=%+v", ll)
		}
	}
}

// --- Conservación de campos derivados internos en el payload del resolver ---

// Una FK derivada (PadreId), que NO está en writable_fields del hijo, llega al
// payload del resolver con su tipo numérico, separada de los campos escribibles
// por el cliente.
func TestWriteAplanadoConservaCampoDerivadoNoEscribible(t *testing.T) {
	var ll []registroMaterializacion
	body := map[string]any{"TerceroId": float64(25), "Hijo": []any{map[string]any{"Valor": "a"}}}
	_, err := Write(context.Background(), body, cfgEscritura(), registryFalso(&ll, nil), WriteOptions{})
	if err != nil {
		t.Fatalf("no debe fallar: %v", err)
	}
	var hijo *registroMaterializacion
	for i := range ll {
		if ll[i].servicio == "hijo" {
			hijo = &ll[i]
		}
	}
	if hijo == nil {
		t.Fatalf("el hijo debe materializarse; llamadas=%+v", ll)
	}
	// PadreId no es escribible por el cliente pero debe llegar al resolver.
	v, ok := hijo.payload["PadreId"].(float64)
	if !ok {
		t.Fatalf("PadreId (fk derivada) debe llegar al resolver como número; payload=%+v", hijo.payload)
	}
	if v != 100 {
		t.Errorf("PadreId debe ser la llave autoritativa 100; payload=%+v", hijo.payload)
	}
}

// La llave canonizada de una entidad nueva se preserva numérica tanto en el
// payload del resolver como en Data, y genera el aviso field_normalization.
func TestWriteCanonizacionPreservaNumericoEnPayloadYData(t *testing.T) {
	var ll []registroMaterializacion
	body := map[string]any{"TerceroId": float64(25), "Hijo": []any{map[string]any{"Valor": "a", "PadreId": float64(999)}}}
	res, err := Write(context.Background(), body, cfgEscritura(), registryFalso(&ll, nil), WriteOptions{})
	if err != nil {
		t.Fatalf("no debe fallar: %v", err)
	}
	// Payload del resolver: PadreId canonizado a 100 numérico.
	var hijo *registroMaterializacion
	for i := range ll {
		if ll[i].servicio == "hijo" {
			hijo = &ll[i]
		}
	}
	if hijo == nil {
		t.Fatalf("el hijo debe materializarse; llamadas=%+v", ll)
	}
	if v, ok := hijo.payload["PadreId"].(float64); !ok || v != 100 {
		t.Errorf("el payload del resolver debe llevar PadreId=100 numérico; payload=%+v", hijo.payload)
	}
	// Data: PadreId canonizado a 100 numérico.
	h := primerHijo(t, res)
	if v, ok := h["PadreId"].(float64); !ok || v != 100 {
		t.Errorf("Data debe llevar PadreId=100 numérico; hijo=%+v", h)
	}
	if avisosNormalizacion(res) != 1 {
		t.Errorf("se esperaba 1 aviso field_normalization; warnings=%+v", res.Warnings)
	}
}

// --- Ancla del invariante de Kind para el motor de escritura ---

// TestInvarianteKindEscritura cierra, del lado del motor de escritura, el
// invariante "todo Kind emitido tiene una constante declarada": garantiza que el
// motor de escritura emite todas las constantes cuyo origen registrado en el
// catálogo es escritura, y que todo Kind que emite está respaldado por una
// constante declarada. Es la contraparte de TestInvarianteKindsDeAviso (que cubre
// las rutas de expansión): entre ambos, cada constante del catálogo queda
// conectada a una ruta de emisión real, impidiendo que una constante quede
// declarada sin emisor que la respalde.
//
// Si se agrega un Kind de origen escritura al catálogo pero ninguna ruta
// ejercida aquí lo emite, este test falla señalando que falta ejercer su ruta.
func TestInvarianteKindEscritura(t *testing.T) {
	declarados := make(map[string]struct{})
	for _, k := range KindsDeAviso() {
		declarados[k] = struct{}{}
	}

	// Recolectar los Kind emitidos por las rutas de escritura ejercidas. Hoy la
	// única ruta que emite un aviso es la canonización de un campo derivado en una
	// entidad nueva; agregar más orígenes de escritura exige ejercer su ruta aquí.
	emitidos := make(map[string]struct{})
	recolectar := func(ws []responses.RelationError) {
		for _, w := range ws {
			if _, ok := declarados[w.Kind]; !ok {
				t.Errorf("el motor de escritura emitió el Kind %q sin constante declarada; aviso=%+v", w.Kind, w)
			}
			emitidos[w.Kind] = struct{}{}
		}
	}

	// Canonización de campo derivado en una entidad nueva -> field_normalization.
	{
		var ll []registroMaterializacion
		body := map[string]any{"TerceroId": float64(25), "Hijo": []any{map[string]any{"Valor": "a", "PadreId": float64(999)}}}
		res, err := Write(context.Background(), body, cfgEscritura(), registryFalso(&ll, nil), WriteOptions{})
		if err != nil {
			t.Fatalf("la canonización de una entidad nueva no debe fallar: %v", err)
		}
		recolectar(res.Warnings)
	}

	// Toda constante de origen escritura debe haber sido emitida por alguna ruta
	// ejercida; de lo contrario quedaría declarada pero muerta.
	for _, k := range KindsDeAviso() {
		origen, _ := OrigenDeKind(k)
		if origen != KindOrigenEscritura {
			continue
		}
		if _, ok := emitidos[k]; !ok {
			t.Errorf("la constante de escritura %q no fue emitida por ninguna ruta ejercida; falta ejercer su ruta", k)
		}
	}
}

// --- Inmutabilidad de un campo derivado reenviado ---
//
// Cuando un campo es a la vez inmutable y derivado (la FK ascendente RefId, que
// es el local_field de una relación fk_inbound), la inmutabilidad no se aplica en
// la pre-validación: protege contra el cambio de valor, no contra el reenvío. La
// conciliación posterior decide según la matriz.

// Un reenvío coincidente de una FK inmutable derivada en una actualización no se
// rechaza por inmutable: coincide con la llave autoritativa y se conserva (sin
// aviso), permitiendo reenviar el Data materializado.
func TestWriteInmutableDerivadoReenvioCoincide(t *testing.T) {
	var ll []registroMaterializacion
	cfg := cfgEscrituraInbound()
	cfg.Write.ImmutableFields = []string{"RefId"}

	// Nodo existente (con Id) que reenvía RefId = Id autoritativo de la entidad
	// referenciada (200), tal como vendría en un Data materializado.
	body := map[string]any{
		"Id":          float64(100),
		"Descripcion": "p",
		"RefId":       float64(200),
		"Referida":    map[string]any{"Id": float64(200), "Nombre": "Ana"},
	}

	res, err := Write(context.Background(), body, cfg, registryInbound(&ll), WriteOptions{Verb: WriteVerbUpdate})
	if err != nil {
		t.Fatalf("un reenvío coincidente de una FK inmutable derivada no debe fallar: %v", err)
	}
	if v, ok := res.Data["RefId"].(float64); !ok || v != 200 {
		t.Errorf("RefId coincidente debe conservarse numérico 200; data=%+v", res.Data)
	}
	if avisosNormalizacion(res) != 0 {
		t.Errorf("un reenvío coincidente no debe producir aviso; warnings=%+v", res.Warnings)
	}
}

// Una FK inmutable derivada que DIFIERE en una entidad existente sigue siendo un
// conflicto (400), independientemente de la inmutabilidad: la matriz lo rechaza
// por entidad existente, no por regla de inmutabilidad.
func TestWriteInmutableDerivadoDifiereExistenteConflicto(t *testing.T) {
	var ll []registroMaterializacion
	cfg := cfgEscrituraInbound()
	cfg.Write.ImmutableFields = []string{"RefId"}

	body := map[string]any{
		"Id":          float64(100),
		"Descripcion": "p",
		"RefId":       float64(999),
		"Referida":    map[string]any{"Id": float64(200), "Nombre": "Ana"},
	}

	_, err := Write(context.Background(), body, cfg, registryInbound(&ll), WriteOptions{Verb: WriteVerbUpdate})
	var ee *ErrorEscritura
	if !errors.As(err, &ee) {
		t.Fatalf("se esperaba *ErrorEscritura por conflicto de entidad existente; err=%v", err)
	}
}
