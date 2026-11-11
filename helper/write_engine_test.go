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
	// fallarRecuperarCon, si no es nil, hace que Recuperar devuelva este error,
	// para ejercitar la política on_relation_error ante un fallo de recuperación.
	fallarRecuperarCon error
	// idRecuperado es el Id autoritativo que Recuperar devuelve cuando el payload
	// no trae Id propio. 0 indica que no existe un registro reutilizable; en ese
	// caso Recuperar cae en el Id de materialización (siguienteID) simulando que la
	// entidad se creará. Permite simular la recuperación por lectura del Id
	// autoritativo de una entidad referenciada por fk_inbound.
	idRecuperado float64
	// recuperaciones cuenta cuántas veces se invocó Recuperar, para verificar que
	// el pre-pass recupera por lectura antes de materializar.
	recuperaciones *int
	// eventos, si no es nil, registra en orden cada lectura ("R:<servicio>") y
	// cada escritura ("M:<servicio>") para verificar el orden del pre-pass.
	eventos *[]string
	// sinRegistro, si es true, hace que Recuperar no encuentre registro existente
	// cuando el payload no trae Id: la entidad se creará.
	sinRegistro bool
	// registro, si no es nil, es el registro existente que Recuperar devuelve.
	registro map[string]any
	// eliminaciones, si no es nil, registra el Id de cada baja solicitada a Eliminar.
	eliminaciones *[]string
}

// Eliminar registra la baja solicitada y devuelve fallarCon si está configurado.
func (r *resolverFalso) Eliminar(_ context.Context, _ config.RelationConfig, id any) error {
	if r.eliminaciones != nil {
		*r.eliminaciones = append(*r.eliminaciones, aCadena(id))
	}
	return r.fallarCon
}

func (r *resolverFalso) Recuperar(
	_ context.Context,
	_ config.RelationConfig,
	payload map[string]any,
	_ []string,
) (map[string]any, error) {
	if r.recuperaciones != nil {
		*r.recuperaciones++
	}
	if r.eventos != nil {
		*r.eventos = append(*r.eventos, "R:"+r.servicio)
	}
	if r.fallarRecuperarCon != nil {
		return nil, r.fallarRecuperarCon
	}
	if r.registro != nil {
		return r.registro, nil
	}
	// Un Id propio en el payload es el identificador autoritativo del registro
	// existente.
	if v, ok := payload["Id"]; ok && v != nil {
		return map[string]any{"Id": aFloat(v)}, nil
	}
	if r.sinRegistro {
		return nil, nil
	}
	// Sin Id propio, el Id autoritativo es el que el registro existente tendría al
	// resolverse por clave natural. Si no se configuró uno, se usa el Id que la
	// materialización asignaría, simulando la creación idempotente.
	if r.idRecuperado > 0 {
		return map[string]any{"Id": r.idRecuperado}, nil
	}
	return map[string]any{"Id": r.siguienteID}, nil
}

// aFloat normaliza a float64 un valor numérico de Id para el resolver de prueba.
func aFloat(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case int64:
		return float64(t)
	default:
		return 0
	}
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
	if r.eventos != nil {
		*r.eventos = append(*r.eventos, "M:"+nombre)
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
// *ErrorEscritura. Bajo el invariante "validar antes de escribir", el conflicto
// se detecta tras recuperar por lectura el Id autoritativo de la entidad
// referenciada y conciliar la FK local, ANTES de materializar: ni la entidad
// referenciada ni la raíz deben pasar por Materialize.
func TestWriteInboundDiferenteExistente(t *testing.T) {
	var ll []registroMaterializacion
	body := map[string]any{"Id": float64(100), "Descripcion": "p", "RefId": float64(999), "Referida": map[string]any{"Nombre": "Ana"}}
	_, err := Write(context.Background(), body, cfgEscrituraInbound(), registryInbound(&ll), WriteOptions{Verb: WriteVerbUpdate})
	var ee *ErrorEscritura
	if !errors.As(err, &ee) {
		t.Fatalf("se esperaba *ErrorEscritura por conflicto fk_inbound de nodo existente; err=%v", err)
	}
	// Ninguna entidad debe materializarse ante el conflicto: ni la referenciada
	// (el pre-pass ya no escribe antes de conciliar) ni la raíz en conflicto.
	for _, c := range ll {
		t.Errorf("no debe materializarse ninguna entidad en conflicto; se materializó %q; llamadas=%+v", c.servicio, ll)
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

	// Nodo marcado para baja sin registro que borrar -> eliminacion_omitida.
	{
		var ll []registroMaterializacion
		cfg := cfgEscritura()
		cfg.Relations[1].Write.NaturalKey = []string{"Valor"}
		hijo := &resolverFalso{servicio: "hijo", siguienteID: 300, llamadas: &ll, sinRegistro: true}
		body := map[string]any{"TerceroId": float64(25), "Hijo": []any{map[string]any{"Valor": "a", "_Eliminar": true}}}
		res, err := Write(context.Background(), body, cfg, registryConHijo(&ll, hijo), WriteOptions{})
		if err != nil {
			t.Fatalf("la baja omitida de un nodo sin registro no debe fallar: %v", err)
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

// Ante un conflicto de fk_inbound en un nodo EXISTENTE (con Id propio), el motor
// debe recuperar por lectura el Id autoritativo de la entidad referenciada,
// conciliar la FK local del nodo y detectar el conflicto ANTES de materializar:
// ni la entidad referenciada ni el nodo deben pasar por Materialize, y se
// devuelve *ErrorEscritura. Hoy el pre-pass materializa la entidad referenciada
// antes de conciliar, por lo que este test falla (se observa una materialización
// de "referida").
func TestWriteInboundConflictoNoMaterializaReferidaNiNodo(t *testing.T) {
	var ll []registroMaterializacion
	var recuperaciones int
	raiz := &resolverFalso{servicio: "raiz", siguienteID: 100, llamadas: &ll, recuperaciones: &recuperaciones}
	// La entidad referenciada existe con Id autoritativo 200 (recuperable por
	// lectura); el nodo reenvía RefId=999, incompatible, en un nodo con Id propio.
	referida := &resolverFalso{servicio: "referida", siguienteID: 200, llamadas: &ll, idRecuperado: 200, recuperaciones: &recuperaciones}
	registry := NewWriteResolverRegistry(raiz, map[string]WriteResolver{"referida": referida})

	body := map[string]any{
		"Id":          float64(100),
		"Descripcion": "p",
		"RefId":       float64(999),
		"Referida":    map[string]any{"Id": float64(200), "Nombre": "Ana"},
	}

	_, err := Write(context.Background(), body, cfgEscrituraInbound(), registry, WriteOptions{Verb: WriteVerbUpdate})
	var ee *ErrorEscritura
	if !errors.As(err, &ee) {
		t.Fatalf("se esperaba *ErrorEscritura por conflicto fk_inbound de nodo existente; err=%v", err)
	}

	// Invariante "validar antes de escribir": ninguna materialización (ni de la
	// entidad referenciada ni del nodo) debe ocurrir ante el conflicto.
	for _, c := range ll {
		t.Errorf("no debe materializarse ninguna entidad ante el conflicto; se materializó %q; llamadas=%+v", c.servicio, ll)
	}
	// La validación previa exige haber recuperado por lectura el Id autoritativo
	// de la entidad referenciada.
	if recuperaciones == 0 {
		t.Errorf("el pre-pass debe recuperar por lectura el Id autoritativo antes de conciliar")
	}
}

// Un fallo del GET de recuperación de una entidad fk_inbound se trata con la
// misma política de error de relación que un fallo de escritura, sin introducir
// una ruta de error distinta: en modo tolerante (notify) se agrega un aviso y la
// cascada continúa (la raíz se materializa igual); en modo estricto (fail) se
// aborta con *errorFatalEscritura.

// registryInboundRecuperarFalla construye el registro fk_inbound con un resolver
// de la entidad referenciada cuya recuperación por lectura falla.
func registryInboundRecuperarFalla(ll *[]registroMaterializacion) WriteResolverRegistry {
	raiz := &resolverFalso{servicio: "raiz", siguienteID: 100, llamadas: ll}
	referida := &resolverFalso{
		servicio:           "referida",
		siguienteID:        200,
		llamadas:           ll,
		fallarRecuperarCon: errors.New("crud caído al recuperar"),
	}
	return NewWriteResolverRegistry(raiz, map[string]WriteResolver{"referida": referida})
}

// notify: el fallo de la recuperación agrega un aviso no sensible y la cascada
// continúa (la raíz se materializa).
func TestWriteInboundFalloRecuperacionTolerante(t *testing.T) {
	var ll []registroMaterializacion
	body := map[string]any{"Descripcion": "p", "Referida": map[string]any{"Nombre": "Ana"}}

	res, err := Write(context.Background(), body, cfgEscrituraInbound(), registryInboundRecuperarFalla(&ll),
		WriteOptions{OnRelationError: config.OnRelationErrorNotify})
	if err != nil {
		t.Fatalf("modo tolerante no debe devolver error fatal ante un fallo de recuperación: %v", err)
	}
	if len(res.Warnings) != 1 || res.Warnings[0].Kind != KindRelation {
		t.Fatalf("se esperaba un aviso relation por el fallo de recuperación; warnings=%+v", res.Warnings)
	}
	if res.Warnings[0].Path != "Referida" || res.Warnings[0].Service != "referida" {
		t.Errorf("el aviso debe identificar la relación Referida; aviso=%+v", res.Warnings[0])
	}
	// El aviso no debe exponer el error interno del CRUD.
	if contieneTexto(res.Warnings[0].Message, "crud caído") {
		t.Errorf("el aviso no debe exponer el error interno del CRUD; aviso=%+v", res.Warnings[0])
	}
	// La cascada continúa: la raíz se materializa pese al fallo tolerado.
	if aCadena(res.Data["Id"]) != "100" {
		t.Errorf("la raíz debe materializarse pese al fallo tolerado; data=%+v", res.Data)
	}
}

// fail: el fallo de la recuperación aborta la cascada con *errorFatalEscritura.
func TestWriteInboundFalloRecuperacionEstricto(t *testing.T) {
	var ll []registroMaterializacion
	body := map[string]any{"Descripcion": "p", "Referida": map[string]any{"Nombre": "Ana"}}

	_, err := Write(context.Background(), body, cfgEscrituraInbound(), registryInboundRecuperarFalla(&ll),
		WriteOptions{OnRelationError: config.OnRelationErrorFail})
	if !EsErrorFatalEscritura(err) {
		t.Fatalf("modo estricto debe abortar con *errorFatalEscritura ante un fallo de recuperación; err=%v", err)
	}
	if contieneTexto(err.Error(), "crud caído") {
		t.Errorf("el error fatal no debe exponer el mensaje interno del CRUD; err=%v", err)
	}
}

// cfgEscrituraInboundConHijo extiende la configuración fk_inbound con una relación
// hija descendente por FK (Hijo, con PadreId) para ejercitar nodos posteriores.
func cfgEscrituraInboundConHijo() config.ServiceConfig {
	cfg := cfgEscrituraInbound()
	cfg.Relations = append(cfg.Relations, config.RelationConfig{
		Name:           "Hijo",
		RelatedService: "hijo",
		MappedName:     "Hijo",
		RelatedField:   "PadreId",
		Cardinality:    config.CardinalityMany,
		Write: &config.WriteConfig{
			WritableFields: []string{"Valor"},
			KeyPropagation: config.KeyPropagationFK,
			Policy:         config.WriteVerbPolicy{Creatable: true, Updatable: true},
		},
	})
	return cfg
}

// registryInboundConHijo construye el registro de raíz (100), referida (200) e
// hijo (300), registrando las materializaciones en ll.
func registryInboundConHijo(ll *[]registroMaterializacion, eventos *[]string) WriteResolverRegistry {
	raiz := &resolverFalso{servicio: "raiz", siguienteID: 100, llamadas: ll, eventos: eventos}
	esp := map[string]WriteResolver{
		"referida": &resolverFalso{servicio: "referida", siguienteID: 200, llamadas: ll, eventos: eventos},
		"hijo":     &resolverFalso{servicio: "hijo", siguienteID: 300, llamadas: ll, eventos: eventos},
	}
	return NewWriteResolverRegistry(raiz, esp)
}

// Una llave incompatible en un nodo hijo posterior del recorrido debe rechazar la
// escritura antes de materializar cualquier entidad, incluida la referida que el
// nodo raíz recupera antes que el hijo.
func TestWriteViolacionEnNodoPosteriorNoEscribeNada(t *testing.T) {
	var ll []registroMaterializacion
	registry := registryInboundConHijo(&ll, nil)

	body := map[string]any{
		"Id":          float64(100),
		"Descripcion": "p",
		"Referida":    map[string]any{"Nombre": "Ana"},
		"Hijo":        []any{map[string]any{"Id": float64(300), "PadreId": float64(999), "Valor": "x"}},
	}

	_, err := Write(context.Background(), body, cfgEscrituraInboundConHijo(), registry, WriteOptions{Verb: WriteVerbUpdate})
	var ee *ErrorEscritura
	if !errors.As(err, &ee) {
		t.Fatalf("se esperaba *ErrorEscritura por la llave incompatible del hijo; err=%v", err)
	}
	if len(ll) != 0 {
		t.Errorf("un 400 no debe dejar escrituras previas; se materializó: %+v", ll)
	}
}

// El pre-pass de solo lectura recupera cada nodo del árbol (raíz, referida fk_inbound
// e hijo) exactamente una vez, y todas esas lecturas ocurren antes de la primera
// escritura.
func TestWritePrePassRecuperaTodoElArbolAntesDeEscribir(t *testing.T) {
	var ll []registroMaterializacion
	var eventos []string
	registry := registryInboundConHijo(&ll, &eventos)

	body := map[string]any{
		"Descripcion": "p",
		"Referida":    map[string]any{"Nombre": "Ana"},
		"Hijo":        []any{map[string]any{"Valor": "x"}},
	}

	if _, err := Write(context.Background(), body, cfgEscrituraInboundConHijo(), registry, WriteOptions{}); err != nil {
		t.Fatalf("no debe fallar: %v", err)
	}

	primeraEscritura := -1
	for i, e := range eventos {
		if strings.HasPrefix(e, "M:") {
			primeraEscritura = i
			break
		}
	}
	if primeraEscritura < 0 {
		t.Fatalf("se esperaban escrituras; eventos=%v", eventos)
	}
	for _, servicio := range []string{"raiz", "referida", "hijo"} {
		n := 0
		for _, e := range eventos[:primeraEscritura] {
			if e == "R:"+servicio {
				n++
			}
		}
		if n != 1 {
			t.Errorf("el pre-pass debe recuperar %q exactamente una vez antes de la primera escritura; eventos=%v", servicio, eventos)
		}
	}
}

// El pre-pass clasifica la acción de cada nodo a partir del estado retenido: crear
// cuando no resuelve a un registro existente y actualizar cuando sí, por Id propio o
// por clave natural.
func TestPrePassClasificaCrearOActualizarPorResolucion(t *testing.T) {
	var ll []registroMaterializacion
	raiz := &resolverFalso{servicio: "raiz", siguienteID: 100, llamadas: &ll, sinRegistro: true}
	referida := &resolverFalso{servicio: "referida", siguienteID: 200, llamadas: &ll, idRecuperado: 200}
	hijo := &resolverFalso{servicio: "hijo", siguienteID: 300, llamadas: &ll, sinRegistro: true}
	registry := NewWriteResolverRegistry(raiz, map[string]WriteResolver{"referida": referida, "hijo": hijo})

	body := map[string]any{
		"Descripcion": "p",
		"Referida":    map[string]any{"Nombre": "Ana"},
		"Hijo": []any{
			map[string]any{"Id": float64(300), "Valor": "x"},
			map[string]any{"Valor": "y"},
		},
	}
	me := prepassDe(t, body, cfgEscrituraInboundConHijo(), registry)

	hijos := body["Hijo"].([]any)
	casos := []struct {
		nombre string
		nodo   map[string]any
		want   AccionEscritura
	}{
		{"raíz sin registro", body, AccionCrear},
		{"referida resuelta por clave natural", body["Referida"].(map[string]any), AccionActualizar},
		{"hijo con Id propio", hijos[0].(map[string]any), AccionActualizar},
		{"hijo sin registro", hijos[1].(map[string]any), AccionCrear},
	}
	for _, c := range casos {
		if got := me.accionDe(c.nodo); got != c.want {
			t.Errorf("%s: acción=%q, se esperaba %q", c.nombre, got, c.want)
		}
	}
}

// Un reenvío cuyos campos escribibles coinciden con el registro retenido no se
// clasifica como actualización; los campos derivados no se comparan.
func TestPrePassReenvioSinCambiosNoEsActualizacion(t *testing.T) {
	var ll []registroMaterializacion
	raiz := &resolverFalso{servicio: "raiz", siguienteID: 100, llamadas: &ll,
		registro: map[string]any{"Id": float64(100), "Descripcion": "p", "RefId": float64(200)}}
	hijo := &resolverFalso{servicio: "hijo", siguienteID: 300, llamadas: &ll,
		registro: map[string]any{"Id": float64(300), "Valor": "x"}}
	registry := NewWriteResolverRegistry(raiz, map[string]WriteResolver{
		"referida": &resolverFalso{servicio: "referida", siguienteID: 200, llamadas: &ll},
		"hijo":     hijo,
	})

	body := map[string]any{
		"Id":          float64(100),
		"Descripcion": "p",
		"Hijo":        []any{map[string]any{"Id": float64(300), "PadreId": float64(100), "Valor": "x"}},
	}
	me := prepassDe(t, body, cfgEscrituraInboundConHijo(), registry)
	hijoNodo := body["Hijo"].([]any)[0].(map[string]any)
	if got := me.accionDe(hijoNodo); got != AccionSinCambios {
		t.Errorf("reenvío sin cambios: acción=%q, se esperaba %q", got, AccionSinCambios)
	}
	if got := me.accionDe(body); got != AccionSinCambios {
		t.Errorf("raíz reenviada sin cambios: acción=%q, se esperaba %q", got, AccionSinCambios)
	}
}

// Un campo escribible que difiere del registro retenido convierte el reenvío en una
// actualización con cambios.
func TestPrePassCambioEnCampoEscribibleEsActualizacion(t *testing.T) {
	var ll []registroMaterializacion
	hijo := &resolverFalso{servicio: "hijo", siguienteID: 300, llamadas: &ll,
		registro: map[string]any{"Id": float64(300), "Valor": "x"}}
	registry := NewWriteResolverRegistry(
		&resolverFalso{servicio: "raiz", siguienteID: 100, llamadas: &ll},
		map[string]WriteResolver{"referida": &resolverFalso{servicio: "referida", siguienteID: 200, llamadas: &ll}, "hijo": hijo},
	)
	body := map[string]any{
		"Id":   float64(100),
		"Hijo": []any{map[string]any{"Id": float64(300), "PadreId": float64(100), "Valor": "z"}},
	}
	me := prepassDe(t, body, cfgEscrituraInboundConHijo(), registry)
	hijoNodo := body["Hijo"].([]any)[0].(map[string]any)
	if got := me.accionDe(hijoNodo); got != AccionActualizar {
		t.Errorf("campo escribible cambiado: acción=%q, se esperaba %q", got, AccionActualizar)
	}
}

// La materialización usa el Id del registro retenido por el pre-pass en lugar de
// volver a buscarlo: la entidad existente se envía con su Id.
func TestWriteMaterializaConIdDelRegistroRetenido(t *testing.T) {
	var ll []registroMaterializacion
	hijo := &resolverFalso{servicio: "hijo", siguienteID: 300, llamadas: &ll,
		registro: map[string]any{"Id": float64(555), "Valor": "x"}}
	registry := NewWriteResolverRegistry(
		&resolverFalso{servicio: "raiz", siguienteID: 100, llamadas: &ll},
		map[string]WriteResolver{"referida": &resolverFalso{servicio: "referida", siguienteID: 200, llamadas: &ll}, "hijo": hijo},
	)
	// Valor distinto del registro retenido: el hijo es una actualización y se escribe.
	body := map[string]any{
		"Descripcion": "p",
		"Hijo":        []any{map[string]any{"Valor": "y"}},
	}
	if _, err := Write(context.Background(), body, cfgEscrituraInboundConHijo(), registry, WriteOptions{}); err != nil {
		t.Fatalf("no debe fallar: %v", err)
	}
	var enviado *registroMaterializacion
	for i := range ll {
		if ll[i].servicio == "hijo" {
			enviado = &ll[i]
		}
	}
	if enviado == nil {
		t.Fatalf("se esperaba materializar el hijo; llamadas=%+v", ll)
	}
	if aFloat(enviado.payload["Id"]) != 555 {
		t.Errorf("la materialización debe usar el Id retenido 555; payload=%+v", enviado.payload)
	}
}

// prepassDe ejecuta solo el pre-pass sobre el cuerpo y devuelve el motor con el estado
// retenido, para verificar la clasificación de cada nodo antes de cualquier escritura.
func prepassDe(t *testing.T, body map[string]any, cfg config.ServiceConfig, registry WriteResolverRegistry) *motorEscritura {
	t.Helper()
	me := &motorEscritura{
		ctx:          context.Background(),
		registry:     registry,
		onRelError:   config.OnRelationErrorNotify,
		servicioRaiz: cfg.Service,
		retenido:     make(map[string]nodoRetenido),
	}
	if err := me.prepass(body, registry.resolverRaiz(), config.RelationConfig{}, *cfg.Write, cfg.Relations, "", nil); err != nil {
		t.Fatalf("el pre-pass no debe fallar: %v", err)
	}
	return me
}

// Un hijo existente cuya asociación almacenada apunta a otro padre no puede ser
// reasociado por el motor aunque el reenvío omita la FK: el 400 se decide antes de
// cualquier escritura.
func TestWriteReparentPorAsociacionAlmacenadaSinFKReenviada(t *testing.T) {
	var ll []registroMaterializacion
	registry := NewWriteResolverRegistry(
		&resolverFalso{servicio: "raiz", siguienteID: 100, llamadas: &ll, registro: map[string]any{"Id": float64(100), "Descripcion": "p"}},
		map[string]WriteResolver{
			"referida": &resolverFalso{servicio: "referida", siguienteID: 200, llamadas: &ll},
			"hijo": &resolverFalso{servicio: "hijo", siguienteID: 300, llamadas: &ll,
				registro: map[string]any{"Id": float64(300), "Valor": "x", "PadreId": float64(999)}},
		},
	)
	body := map[string]any{
		"Id":          float64(100),
		"Descripcion": "p",
		"Hijo":        []any{map[string]any{"Id": float64(300), "Valor": "z"}},
	}
	_, err := Write(context.Background(), body, cfgEscrituraInboundConHijo(), registry, WriteOptions{Verb: WriteVerbUpdate})
	var ee *ErrorEscritura
	if !errors.As(err, &ee) {
		t.Fatalf("se esperaba *ErrorEscritura por reparent de hijo existente; err=%v", err)
	}
	if len(ll) != 0 {
		t.Errorf("un 400 por reparent no debe escribir; se materializó: %+v", ll)
	}
}

// Un reenvío con FK propagada distinta de la asociación almacenada de un hijo
// existente es un reparent, aunque la FK reenviada coincida con la del padre
// materializado.
func TestWriteReparentPorFKReenviadaDistintaDeAlmacenada(t *testing.T) {
	var ll []registroMaterializacion
	registry := NewWriteResolverRegistry(
		&resolverFalso{servicio: "raiz", siguienteID: 100, llamadas: &ll, registro: map[string]any{"Id": float64(100), "Descripcion": "p"}},
		map[string]WriteResolver{
			"referida": &resolverFalso{servicio: "referida", siguienteID: 200, llamadas: &ll},
			"hijo": &resolverFalso{servicio: "hijo", siguienteID: 300, llamadas: &ll,
				registro: map[string]any{"Id": float64(300), "Valor": "x", "PadreId": float64(999)}},
		},
	)
	body := map[string]any{
		"Id":          float64(100),
		"Descripcion": "p",
		"Hijo":        []any{map[string]any{"Id": float64(300), "PadreId": float64(100), "Valor": "z"}},
	}
	_, err := Write(context.Background(), body, cfgEscrituraInboundConHijo(), registry, WriteOptions{Verb: WriteVerbUpdate})
	var ee *ErrorEscritura
	if !errors.As(err, &ee) {
		t.Fatalf("se esperaba *ErrorEscritura por FK reenviada distinta de la almacenada; err=%v", err)
	}
	if len(ll) != 0 {
		t.Errorf("un 400 por reparent no debe escribir; se materializó: %+v", ll)
	}
}

// Regresión: un hijo existente que ya pertenece al padre y reenvía sin FK se actualiza
// con normalidad, sin tratarse como reparent.
func TestWriteHijoExistenteEnSuPadreNoEsReparent(t *testing.T) {
	var ll []registroMaterializacion
	registry := NewWriteResolverRegistry(
		&resolverFalso{servicio: "raiz", siguienteID: 100, llamadas: &ll, registro: map[string]any{"Id": float64(100), "Descripcion": "p"}},
		map[string]WriteResolver{
			"referida": &resolverFalso{servicio: "referida", siguienteID: 200, llamadas: &ll},
			"hijo": &resolverFalso{servicio: "hijo", siguienteID: 300, llamadas: &ll,
				registro: map[string]any{"Id": float64(300), "Valor": "x", "PadreId": float64(100)}},
		},
	)
	body := map[string]any{
		"Id":          float64(100),
		"Descripcion": "p",
		"Hijo":        []any{map[string]any{"Id": float64(300), "Valor": "z"}},
	}
	if _, err := Write(context.Background(), body, cfgEscrituraInboundConHijo(), registry, WriteOptions{Verb: WriteVerbUpdate}); err != nil {
		t.Fatalf("un hijo existente en su padre no debe rechazarse: %v", err)
	}
}

// Un nodo clasificado sin cambios no se escribe: ningún PATCH ni POST, pero el árbol
// resultante conserva su registro existente con su Id.
func TestWriteReenvioSinCambiosNoMaterializaYConservaRegistro(t *testing.T) {
	var ll []registroMaterializacion
	registry := NewWriteResolverRegistry(
		&resolverFalso{servicio: "raiz", siguienteID: 100, llamadas: &ll, registro: map[string]any{"Id": float64(100), "Descripcion": "p"}},
		map[string]WriteResolver{
			"referida": &resolverFalso{servicio: "referida", siguienteID: 200, llamadas: &ll},
			"hijo": &resolverFalso{servicio: "hijo", siguienteID: 300, llamadas: &ll,
				registro: map[string]any{"Id": float64(300), "Valor": "x", "PadreId": float64(100)}},
		},
	)
	body := map[string]any{
		"Id":          float64(100),
		"Descripcion": "p",
		"Hijo":        []any{map[string]any{"Id": float64(300), "PadreId": float64(100), "Valor": "x"}},
	}
	res, err := Write(context.Background(), body, cfgEscrituraInboundConHijo(), registry, WriteOptions{Verb: WriteVerbUpdate})
	if err != nil {
		t.Fatalf("un reenvío sin cambios no debe fallar: %v", err)
	}
	if len(ll) != 0 {
		t.Errorf("un reenvío sin cambios no debe escribirse; materializaciones=%+v", ll)
	}
	if aFloat(res.Data["Id"]) != 100 {
		t.Errorf("la raíz sin cambios debe conservar su registro con Id 100; data=%+v", res.Data)
	}
	hijos := res.Data["Hijo"].([]any)
	if aFloat(hijos[0].(map[string]any)["Id"]) != 300 {
		t.Errorf("el hijo sin cambios debe conservar su registro con Id 300; hijo=%+v", hijos[0])
	}
}

// cfgConPoliticaHijo devuelve la configuración de cfgEscritura con la política de la
// arista Hijo reemplazada por pol.
func cfgConPoliticaHijo(pol config.WriteVerbPolicy) config.ServiceConfig {
	cfg := cfgEscritura()
	cfg.Relations[1].Write.Policy = pol
	return cfg
}

// registryConHijo construye el registro de cfgEscritura usando el resolver de Hijo
// que entrega el llamador, para controlar lo que Recuperar devuelve para esa arista.
func registryConHijo(llamadas *[]registroMaterializacion, hijo WriteResolver) WriteResolverRegistry {
	raiz := &resolverFalso{servicio: "raiz", siguienteID: 100, llamadas: llamadas}
	esp := map[string]WriteResolver{
		"detalle": &resolverFalso{servicio: "detalle", siguienteID: 200, llamadas: llamadas},
		"hijo":    hijo,
	}
	return NewWriteResolverRegistry(raiz, esp)
}

// comoErrorEscritura devuelve el *ErrorEscritura contenido en err, o nil si err no lo
// contiene.
func comoErrorEscritura(err error) *ErrorEscritura {
	var ee *ErrorEscritura
	if errors.As(err, &ee) {
		return ee
	}
	return nil
}

// Una arista con creatable: false que recibe una entidad sin registro existente
// (crearía una nueva) se rechaza con *ErrorEscritura que nombra la arista, y no se
// envía ninguna escritura a los CRUD.
func TestWriteCreatableFalseRechazaCreacion(t *testing.T) {
	var llamadas []registroMaterializacion
	hijo := &resolverFalso{servicio: "hijo", siguienteID: 300, llamadas: &llamadas, sinRegistro: true}
	body := map[string]any{"TerceroId": float64(25), "Hijo": []any{map[string]any{"Valor": "a"}}}

	_, err := Write(context.Background(), body,
		cfgConPoliticaHijo(config.WriteVerbPolicy{Creatable: false, Updatable: true}),
		registryConHijo(&llamadas, hijo), WriteOptions{Verb: WriteVerbCreate})

	ee := comoErrorEscritura(err)
	if ee == nil {
		t.Fatalf("se esperaba *ErrorEscritura por creación no permitida; err=%v", err)
	}
	if !contieneTexto(ee.Path, "Hijo") {
		t.Errorf("el error debe nombrar la arista Hijo; path=%q", ee.Path)
	}
	if len(llamadas) != 0 {
		t.Errorf("no debe enviarse ninguna escritura a los CRUD; llamadas=%+v", llamadas)
	}
}

// Una arista con creatable: false también rechaza la creación de una entidad
// referenciada por fk_inbound, antes de escribir la raíz.
func TestWriteCreatableFalseRechazaReferidaInbound(t *testing.T) {
	var llamadas []registroMaterializacion
	cfg := cfgEscrituraInbound()
	cfg.Relations[0].Write.Policy = config.WriteVerbPolicy{Creatable: false, Updatable: true}
	raiz := &resolverFalso{servicio: "raiz", siguienteID: 100, llamadas: &llamadas}
	referida := &resolverFalso{servicio: "referida", siguienteID: 200, llamadas: &llamadas, sinRegistro: true}
	registry := NewWriteResolverRegistry(raiz, map[string]WriteResolver{"referida": referida})
	body := map[string]any{"Descripcion": "p", "Referida": map[string]any{"Nombre": "Ana"}}

	_, err := Write(context.Background(), body, cfg, registry, WriteOptions{Verb: WriteVerbCreate})

	ee := comoErrorEscritura(err)
	if ee == nil {
		t.Fatalf("se esperaba *ErrorEscritura por creación fk_inbound no permitida; err=%v", err)
	}
	if !contieneTexto(ee.Path, "Referida") {
		t.Errorf("el error debe nombrar la arista Referida; path=%q", ee.Path)
	}
	if len(llamadas) != 0 {
		t.Errorf("no debe enviarse ninguna escritura a los CRUD; llamadas=%+v", llamadas)
	}
}

// Una arista con updatable: false que recibe una entidad existente con cambios en
// sus campos escribibles se rechaza con *ErrorEscritura, sin escrituras.
func TestWriteUpdatableFalseRechazaCambioExistente(t *testing.T) {
	var llamadas []registroMaterializacion
	hijo := &resolverFalso{
		servicio: "hijo", siguienteID: 300, llamadas: &llamadas,
		registro: map[string]any{"Id": float64(555), "Valor": "viejo", "PadreId": float64(100)},
	}
	body := map[string]any{"TerceroId": float64(25), "Hijo": []any{map[string]any{"Id": float64(555), "Valor": "nuevo"}}}

	_, err := Write(context.Background(), body,
		cfgConPoliticaHijo(config.WriteVerbPolicy{Creatable: true, Updatable: false}),
		registryConHijo(&llamadas, hijo), WriteOptions{Verb: WriteVerbUpdate})

	ee := comoErrorEscritura(err)
	if ee == nil {
		t.Fatalf("se esperaba *ErrorEscritura por actualización no permitida; err=%v", err)
	}
	if !contieneTexto(ee.Path, "Hijo") {
		t.Errorf("el error debe nombrar la arista Hijo; path=%q", ee.Path)
	}
	if len(llamadas) != 0 {
		t.Errorf("no debe enviarse ninguna escritura a los CRUD; llamadas=%+v", llamadas)
	}
}

// Con updatable: false, una entidad que resuelve por clave natural a un registro
// existente con un campo escribible distinto se rechaza sin escrituras, aunque el
// payload no traiga Id.
func TestWriteUpdatableFalseRechazaCambioPorClaveNatural(t *testing.T) {
	var llamadas []registroMaterializacion
	cfg := cfgConPoliticaHijo(config.WriteVerbPolicy{Creatable: true, Updatable: false})
	cfg.Relations[1].Write.WritableFields = []string{"Codigo", "Valor"}
	cfg.Relations[1].Write.NaturalKey = []string{"Codigo"}
	hijo := &resolverFalso{
		servicio: "hijo", siguienteID: 300, llamadas: &llamadas,
		registro: map[string]any{"Id": float64(555), "Codigo": "A", "Valor": "viejo", "PadreId": float64(100)},
	}
	body := map[string]any{"TerceroId": float64(25), "Hijo": []any{map[string]any{"Codigo": "A", "Valor": "nuevo"}}}

	_, err := Write(context.Background(), body, cfg, registryConHijo(&llamadas, hijo), WriteOptions{Verb: WriteVerbUpdate})

	if ee := comoErrorEscritura(err); ee == nil {
		t.Fatalf("se esperaba *ErrorEscritura por actualización por clave natural no permitida; err=%v", err)
	}
	if len(llamadas) != 0 {
		t.Errorf("no debe enviarse ninguna escritura a los CRUD; llamadas=%+v", llamadas)
	}
}

// Con creatable: true, una entidad sin registro existente se crea normalmente.
func TestWriteCreatableTrueCreaEntidadNueva(t *testing.T) {
	var llamadas []registroMaterializacion
	hijo := &resolverFalso{servicio: "hijo", siguienteID: 300, llamadas: &llamadas, sinRegistro: true}
	body := map[string]any{"TerceroId": float64(25), "Hijo": []any{map[string]any{"Valor": "a"}}}

	_, err := Write(context.Background(), body,
		cfgConPoliticaHijo(config.WriteVerbPolicy{Creatable: true, Updatable: true}),
		registryConHijo(&llamadas, hijo), WriteOptions{Verb: WriteVerbCreate})

	if err != nil {
		t.Fatalf("una creación permitida no debe fallar: %v", err)
	}
	if !huboMaterializacion(llamadas, "hijo") {
		t.Errorf("la entidad hijo debe materializarse con creatable: true; llamadas=%+v", llamadas)
	}
}

// Con updatable: true, una entidad existente con cambios se actualiza normalmente.
func TestWriteUpdatableTrueActualizaEntidadExistente(t *testing.T) {
	var llamadas []registroMaterializacion
	hijo := &resolverFalso{
		servicio: "hijo", siguienteID: 300, llamadas: &llamadas,
		registro: map[string]any{"Id": float64(555), "Valor": "viejo", "PadreId": float64(100)},
	}
	body := map[string]any{"TerceroId": float64(25), "Hijo": []any{map[string]any{"Id": float64(555), "Valor": "nuevo"}}}

	_, err := Write(context.Background(), body,
		cfgConPoliticaHijo(config.WriteVerbPolicy{Creatable: true, Updatable: true}),
		registryConHijo(&llamadas, hijo), WriteOptions{Verb: WriteVerbUpdate})

	if err != nil {
		t.Fatalf("una actualización permitida no debe fallar: %v", err)
	}
	if !huboMaterializacion(llamadas, "hijo") {
		t.Errorf("la entidad hijo debe actualizarse con updatable: true; llamadas=%+v", llamadas)
	}
}

// Con updatable: false, el reenvío de una entidad existente sin cambios en sus campos
// escribibles no se rechaza y no se reescribe.
func TestWriteUpdatableFalseReenvioSinCambiosNoSeRechaza(t *testing.T) {
	var llamadas []registroMaterializacion
	hijo := &resolverFalso{
		servicio: "hijo", siguienteID: 300, llamadas: &llamadas,
		registro: map[string]any{"Id": float64(555), "Valor": "a", "PadreId": float64(100)},
	}
	body := map[string]any{"TerceroId": float64(25), "Hijo": []any{map[string]any{"Id": float64(555), "Valor": "a"}}}

	_, err := Write(context.Background(), body,
		cfgConPoliticaHijo(config.WriteVerbPolicy{Creatable: true, Updatable: false}),
		registryConHijo(&llamadas, hijo), WriteOptions{Verb: WriteVerbUpdate})

	if err != nil {
		t.Fatalf("un reenvío sin cambios no debe rechazarse con updatable: false: %v", err)
	}
	if huboMaterializacion(llamadas, "hijo") {
		t.Errorf("un reenvío sin cambios no debe reescribir la entidad hijo; llamadas=%+v", llamadas)
	}
}

// Con creatable: false, una entidad existente con cambios en sus campos se actualiza
// si updatable: true: la política de creación no bloquea la actualización.
func TestWriteCreatableFalseNoBloqueaActualizacion(t *testing.T) {
	var llamadas []registroMaterializacion
	hijo := &resolverFalso{
		servicio: "hijo", siguienteID: 300, llamadas: &llamadas,
		registro: map[string]any{"Id": float64(555), "Valor": "viejo", "PadreId": float64(100)},
	}
	body := map[string]any{"TerceroId": float64(25), "Hijo": []any{map[string]any{"Id": float64(555), "Valor": "nuevo"}}}

	_, err := Write(context.Background(), body,
		cfgConPoliticaHijo(config.WriteVerbPolicy{Creatable: false, Updatable: true}),
		registryConHijo(&llamadas, hijo), WriteOptions{Verb: WriteVerbUpdate})

	if err != nil {
		t.Fatalf("creatable: false no debe bloquear una actualización permitida: %v", err)
	}
	if !huboMaterializacion(llamadas, "hijo") {
		t.Errorf("la actualización permitida debe materializarse; llamadas=%+v", llamadas)
	}
}

// huboMaterializacion informa si alguna escritura de llamadas corresponde al servicio.
func huboMaterializacion(llamadas []registroMaterializacion, servicio string) bool {
	for _, c := range llamadas {
		if c.servicio == servicio {
			return true
		}
	}
	return false
}

// Con una raíz nueva (sin Id, sin llave que propagar), un hijo con Id propio que ya
// pertenece a otro padre se rechaza con *ErrorEscritura que nombra la arista, sin
// escrituras.
func TestWriteReparentConRaizNuevaRechazaSinEscribir(t *testing.T) {
	var llamadas []registroMaterializacion
	raiz := &resolverFalso{servicio: "raiz", siguienteID: 100, llamadas: &llamadas, sinRegistro: true}
	hijo := &resolverFalso{
		servicio: "hijo", siguienteID: 300, llamadas: &llamadas,
		registro: map[string]any{"Id": float64(300), "Valor": "x", "PadreId": float64(999)},
	}
	registry := NewWriteResolverRegistry(raiz, map[string]WriteResolver{
		"detalle": &resolverFalso{servicio: "detalle", siguienteID: 200, llamadas: &llamadas},
		"hijo":    hijo,
	})
	body := map[string]any{
		"TerceroId":   float64(25),
		"Descripcion": "p",
		"Hijo":        []any{map[string]any{"Id": float64(300), "Valor": "x"}},
	}

	_, err := Write(context.Background(), body, cfgEscritura(), registry, WriteOptions{Verb: WriteVerbCreate})

	ee := comoErrorEscritura(err)
	if ee == nil {
		t.Fatalf("se esperaba *ErrorEscritura por reparent con raíz nueva; err=%v", err)
	}
	if !contieneTexto(ee.Path, "Hijo") {
		t.Errorf("el error debe nombrar la arista Hijo; path=%q", ee.Path)
	}
	if len(llamadas) != 0 {
		t.Errorf("un 400 por reparent no debe escribir en ningún CRUD; llamadas=%+v", llamadas)
	}
}

// Un hijo con Id propio bajo el padre que ya lo posee, con la llave propagada igual a
// su asociación almacenada, no es reparent: se valida y se actualiza con normalidad,
// y conserva la llave del padre.
func TestWriteHijoBajoSuPadreConLlaveCoincidenteNoEsReparent(t *testing.T) {
	var llamadas []registroMaterializacion
	raiz := &resolverFalso{
		servicio: "raiz", siguienteID: 100, llamadas: &llamadas,
		registro: map[string]any{"Id": float64(100), "Descripcion": "p"},
	}
	hijo := &resolverFalso{
		servicio: "hijo", siguienteID: 300, llamadas: &llamadas,
		registro: map[string]any{"Id": float64(300), "Valor": "x", "PadreId": float64(100)},
	}
	registry := NewWriteResolverRegistry(raiz, map[string]WriteResolver{
		"detalle": &resolverFalso{servicio: "detalle", siguienteID: 200, llamadas: &llamadas},
		"hijo":    hijo,
	})
	body := map[string]any{
		"Id":          float64(100),
		"Descripcion": "p",
		"Hijo":        []any{map[string]any{"Id": float64(300), "Valor": "z"}},
	}

	_, err := Write(context.Background(), body, cfgEscritura(), registry, WriteOptions{Verb: WriteVerbUpdate})

	if err != nil {
		t.Fatalf("un hijo bajo su padre legítimo con llave coincidente no debe rechazarse: %v", err)
	}
	var payloadHijo map[string]any
	for _, c := range llamadas {
		if c.servicio == "hijo" {
			payloadHijo = c.payload
		}
	}
	if payloadHijo == nil {
		t.Fatalf("el hijo con cambios debe actualizarse; llamadas=%+v", llamadas)
	}
	if v, ok := payloadHijo["PadreId"].(float64); !ok || v != 100 {
		t.Errorf("el hijo debe conservar la llave de su padre (100) numérica; payload=%+v", payloadHijo)
	}
}

// Un nodo marcado con _Eliminar bajo una arista con deletable: false se rechaza por la
// política de la arista, antes de cualquier baja o escritura.
func TestWriteEliminarBajoDeletableFalseRechazaSinEscribir(t *testing.T) {
	var llamadas []registroMaterializacion
	var bajas []string
	hijo := &resolverFalso{
		servicio: "hijo", siguienteID: 300, llamadas: &llamadas, eliminaciones: &bajas,
		registro: map[string]any{"Id": float64(555), "Valor": "a", "PadreId": float64(100)},
	}
	body := map[string]any{"TerceroId": float64(25), "Hijo": []any{map[string]any{"Id": float64(555), "_Eliminar": true}}}

	_, err := Write(context.Background(), body,
		cfgConPoliticaHijo(config.WriteVerbPolicy{Creatable: true, Updatable: true, Deletable: false}),
		registryConHijo(&llamadas, hijo), WriteOptions{Verb: WriteVerbUpdate})

	ee := comoErrorEscritura(err)
	if ee == nil {
		t.Fatalf("se esperaba *ErrorEscritura por eliminación no permitida; err=%v", err)
	}
	if !contieneTexto(ee.Path, "Hijo") || !contieneTexto(ee.Motivo, "deletable") {
		t.Errorf("el error debe nombrar la arista Hijo y la política deletable; path=%q motivo=%q", ee.Path, ee.Motivo)
	}
	if len(llamadas) != 0 || len(bajas) != 0 {
		t.Errorf("no debe haber escrituras ni bajas; llamadas=%+v bajas=%v", llamadas, bajas)
	}
}

// Un nodo marcado con _Eliminar bajo una arista con deletable: true se da de baja con
// la operación de borrado del CRUD dueño, y no se materializa como creación ni
// actualización.
func TestWriteEliminarBajoDeletableTrueBorraPorCRUD(t *testing.T) {
	var llamadas []registroMaterializacion
	var bajas []string
	hijo := &resolverFalso{
		servicio: "hijo", siguienteID: 300, llamadas: &llamadas, eliminaciones: &bajas,
		registro: map[string]any{"Id": float64(555), "Valor": "a", "PadreId": float64(100)},
	}
	body := map[string]any{"TerceroId": float64(25), "Hijo": []any{map[string]any{"Id": float64(555), "_Eliminar": true}}}

	_, err := Write(context.Background(), body,
		cfgConPoliticaHijo(config.WriteVerbPolicy{Creatable: true, Updatable: true, Deletable: true}),
		registryConHijo(&llamadas, hijo), WriteOptions{Verb: WriteVerbUpdate})

	if err != nil {
		t.Fatalf("una baja permitida no debe fallar: %v", err)
	}
	if len(bajas) != 1 || bajas[0] != "555" {
		t.Errorf("la baja debe solicitarse al CRUD dueño para el Id 555; bajas=%v", bajas)
	}
	if huboMaterializacion(llamadas, "hijo") {
		t.Errorf("el nodo marcado no debe materializarse como creación o actualización; llamadas=%+v", llamadas)
	}
}

// Sin nodos marcados, el motor no infiere ninguna baja aunque el CRUD tenga registros
// que el cuerpo no trae.
func TestWriteSinMarcaNoInfiereEliminacion(t *testing.T) {
	var llamadas []registroMaterializacion
	var bajas []string
	hijo := &resolverFalso{
		servicio: "hijo", siguienteID: 300, llamadas: &llamadas, eliminaciones: &bajas,
		registro: map[string]any{"Id": float64(555), "Valor": "a", "PadreId": float64(100)},
	}
	body := map[string]any{"TerceroId": float64(25)}

	_, err := Write(context.Background(), body, cfgEscritura(), registryConHijo(&llamadas, hijo), WriteOptions{Verb: WriteVerbUpdate})

	if err != nil {
		t.Fatalf("un cuerpo sin marcas no debe fallar: %v", err)
	}
	if len(bajas) != 0 {
		t.Errorf("el motor no debe inferir bajas sin marca; bajas=%v", bajas)
	}
}

// La marca _Eliminar debe ser booleana; otro tipo se rechaza con 400 antes de escribir.
func TestWriteEliminarNoBooleanaRechaza(t *testing.T) {
	var llamadas []registroMaterializacion
	var bajas []string
	hijo := &resolverFalso{
		servicio: "hijo", siguienteID: 300, llamadas: &llamadas, eliminaciones: &bajas,
		registro: map[string]any{"Id": float64(555), "Valor": "a", "PadreId": float64(100)},
	}
	body := map[string]any{"TerceroId": float64(25), "Hijo": []any{map[string]any{"Id": float64(555), "_Eliminar": "si"}}}

	_, err := Write(context.Background(), body, cfgEscritura(), registryConHijo(&llamadas, hijo), WriteOptions{Verb: WriteVerbUpdate})

	ee := comoErrorEscritura(err)
	if ee == nil || !contieneTexto(ee.Motivo, "booleana") {
		t.Fatalf("se esperaba *ErrorEscritura por marca no booleana; err=%v", err)
	}
	if len(llamadas) != 0 || len(bajas) != 0 {
		t.Errorf("no debe haber escrituras ni bajas; llamadas=%+v bajas=%v", llamadas, bajas)
	}
}

// La raíz no es una arista con política de eliminación: marcarla se rechaza con 400.
func TestWriteEliminarEnRaizRechaza(t *testing.T) {
	var llamadas []registroMaterializacion
	body := map[string]any{"TerceroId": float64(25), "_Eliminar": true}

	_, err := Write(context.Background(), body, cfgEscritura(), registryFalso(&llamadas, nil), WriteOptions{Verb: WriteVerbCreate})

	ee := comoErrorEscritura(err)
	if ee == nil || !contieneTexto(ee.Motivo, "raíz") {
		t.Fatalf("se esperaba *ErrorEscritura por marca en la raíz; err=%v", err)
	}
	if len(llamadas) != 0 {
		t.Errorf("no debe haber escrituras; llamadas=%+v", llamadas)
	}
}

// Sin Id, si la clave natural incluye un campo derivado de la relación padre, la
// búsqueda puede fallar y la baja quedaría mal identificada: se rechaza con 400.
func TestWriteEliminarSinIdConClaveDerivadaRechaza(t *testing.T) {
	var llamadas []registroMaterializacion
	var bajas []string
	cfg := cfgConPoliticaHijo(config.WriteVerbPolicy{Creatable: true, Updatable: true, Deletable: true})
	cfg.Relations[1].Write.NaturalKey = []string{"Valor", "PadreId"}
	hijo := &resolverFalso{servicio: "hijo", siguienteID: 300, llamadas: &llamadas, eliminaciones: &bajas, sinRegistro: true}
	body := map[string]any{"TerceroId": float64(25), "Hijo": []any{map[string]any{"Valor": "a", "_Eliminar": true}}}

	_, err := Write(context.Background(), body, cfg, registryConHijo(&llamadas, hijo), WriteOptions{Verb: WriteVerbUpdate})

	ee := comoErrorEscritura(err)
	if ee == nil || !contieneTexto(ee.Motivo, "clave") {
		t.Fatalf("se esperaba *ErrorEscritura por clave derivada sin Id; err=%v", err)
	}
	if len(llamadas) != 0 || len(bajas) != 0 {
		t.Errorf("no debe haber escrituras ni bajas; llamadas=%+v bajas=%v", llamadas, bajas)
	}
}

// Sin Id y sin coincidencia por una clave natural propia del nodo, no hay nada que
// borrar: la baja se omite con un aviso no bloqueante, sin error ni escritura.
func TestWriteEliminarSinRegistroEsAvisoNoBloqueante(t *testing.T) {
	var llamadas []registroMaterializacion
	var bajas []string
	cfg := cfgConPoliticaHijo(config.WriteVerbPolicy{Creatable: true, Updatable: true, Deletable: true})
	cfg.Relations[1].Write.NaturalKey = []string{"Valor"}
	hijo := &resolverFalso{servicio: "hijo", siguienteID: 300, llamadas: &llamadas, eliminaciones: &bajas, sinRegistro: true}
	body := map[string]any{"TerceroId": float64(25), "Hijo": []any{map[string]any{"Valor": "a", "_Eliminar": true}}}

	res, err := Write(context.Background(), body, cfg, registryConHijo(&llamadas, hijo), WriteOptions{Verb: WriteVerbUpdate})

	if err != nil {
		t.Fatalf("un nodo marcado sin registro no debe bloquear la escritura: %v", err)
	}
	if len(bajas) != 0 || huboMaterializacion(llamadas, "hijo") {
		t.Errorf("no debe haber baja ni escritura del nodo; bajas=%v llamadas=%+v", bajas, llamadas)
	}
	avisos := 0
	for _, w := range res.Warnings {
		if w.Kind == KindEliminacionOmitida && w.Path == "Hijo[0]" {
			avisos++
		}
	}
	if avisos != 1 {
		t.Errorf("se esperaba 1 aviso eliminacion_omitida en Hijo[0]; warnings=%+v", res.Warnings)
	}
}

// fk_inbound referencia una entidad que se materializa antes que la raíz; su baja no
// se admite con la marca, porque dejaría la FK local de la raíz apuntando a nada.
func TestWriteEliminarEnFKInboundRechaza(t *testing.T) {
	var llamadas []registroMaterializacion
	cfg := cfgEscrituraInbound()
	cfg.Relations[0].Write.Policy = config.WriteVerbPolicy{Creatable: true, Updatable: true, Deletable: true}
	body := map[string]any{"Descripcion": "p", "Referida": map[string]any{"Id": float64(200), "Nombre": "Ana", "_Eliminar": true}}

	_, err := Write(context.Background(), body, cfg, registryInbound(&llamadas), WriteOptions{Verb: WriteVerbUpdate})

	ee := comoErrorEscritura(err)
	if ee == nil || !contieneTexto(ee.Motivo, "ascendente") {
		t.Fatalf("se esperaba *ErrorEscritura por marca en una relación fk_inbound; err=%v", err)
	}
	if len(llamadas) != 0 {
		t.Errorf("no debe haber escrituras; llamadas=%+v", llamadas)
	}
}

// resolverConRegistros entrega, por Id, el registro existente que Recuperar devuelve, y
// registra cada lectura en eventos. Un Id que no está en porID no tiene registro en el CRUD;
// sin Id no hay registro.
type resolverConRegistros struct {
	*resolverFalso
	porID map[float64]map[string]any
}

func (r *resolverConRegistros) Recuperar(
	_ context.Context,
	_ config.RelationConfig,
	payload map[string]any,
	_ []string,
) (map[string]any, error) {
	if r.eventos != nil {
		*r.eventos = append(*r.eventos, "R:"+r.servicio)
	}
	v, ok := payload["Id"]
	if !ok || v == nil {
		return nil, nil
	}
	return r.porID[aFloat(v)], nil
}

// escenarioTardio construye el registro de un árbol raíz (100) con hijos existentes 300, 301
// (de otro padre) y 302, junto con las llamadas, lecturas y bajas capturadas.
func escenarioTardio(llamadas *[]registroMaterializacion, eventos *[]string, bajas *[]string) WriteResolverRegistry {
	raiz := &resolverConRegistros{
		resolverFalso: &resolverFalso{servicio: "raiz", siguienteID: 100, llamadas: llamadas, eventos: eventos},
		porID: map[float64]map[string]any{
			100: {"Id": float64(100), "Descripcion": "p", "TerceroId": float64(25)},
		},
	}
	hijo := &resolverConRegistros{
		resolverFalso: &resolverFalso{servicio: "hijo", siguienteID: 300, llamadas: llamadas, eventos: eventos, eliminaciones: bajas},
		porID: map[float64]map[string]any{
			300: {"Id": float64(300), "Valor": "a", "PadreId": float64(100)},
			301: {"Id": float64(301), "Valor": "b", "PadreId": float64(999)},
			302: {"Id": float64(302), "Valor": "c", "PadreId": float64(100)},
		},
	}
	return NewWriteResolverRegistry(raiz, map[string]WriteResolver{
		"detalle": &resolverFalso{servicio: "detalle", siguienteID: 200, llamadas: llamadas, eventos: eventos},
		"hijo":    hijo,
	})
}

// Cada control de escritura (política de crear, actualizar y eliminar, no-reparent,
// inmutable y campo derivado) rechaza con 400 cuando la violación está en un hijo posterior
// del recorrido, después de nodos que sí se escribirían. El rechazo llega como
// *ErrorEscritura sobre el árbol completo, sin creaciones, actualizaciones ni bajas en los
// CRUD. Las lecturas esperadas indican que el pre-pass recupera cada nodo una sola vez; los
// rechazos previos al pre-pass (inmutable) no leen nada.
func TestWriteViolacionTardiaNoEscribeNada(t *testing.T) {
	todas := config.WriteVerbPolicy{Creatable: true, Updatable: true, Deletable: true}
	casos := []struct {
		nombre    string
		pol       config.WriteVerbPolicy
		inmutable bool
		hijos     []any
		lecturas  int
	}{
		{
			nombre:   "creacion no permitida en el ultimo hijo",
			pol:      config.WriteVerbPolicy{Creatable: false, Updatable: true, Deletable: true},
			hijos:    []any{map[string]any{"Id": float64(300), "Valor": "z"}, map[string]any{"Valor": "nuevo"}},
			lecturas: 3,
		},
		{
			nombre:   "actualizacion no permitida en el ultimo hijo",
			pol:      config.WriteVerbPolicy{Creatable: true, Updatable: false, Deletable: true},
			hijos:    []any{map[string]any{"Valor": "nuevo"}, map[string]any{"Id": float64(300), "Valor": "z"}},
			lecturas: 3,
		},
		{
			nombre:   "baja no permitida en el ultimo hijo",
			pol:      config.WriteVerbPolicy{Creatable: true, Updatable: true, Deletable: false},
			hijos:    []any{map[string]any{"Id": float64(300), "Valor": "z"}, map[string]any{"Id": float64(302), "_Eliminar": true}},
			lecturas: 3,
		},
		{
			nombre:   "reparent del ultimo hijo",
			pol:      todas,
			hijos:    []any{map[string]any{"Id": float64(300), "Valor": "z"}, map[string]any{"Id": float64(301), "Valor": "b"}},
			lecturas: 3,
		},
		{
			nombre:   "campo derivado en conflicto en el ultimo hijo",
			pol:      todas,
			hijos:    []any{map[string]any{"Id": float64(300), "Valor": "z", "PadreId": float64(555)}},
			lecturas: 2,
		},
		{
			nombre:    "campo inmutable en el ultimo hijo",
			pol:       todas,
			inmutable: true,
			hijos:     []any{map[string]any{"Id": float64(300)}, map[string]any{"Id": float64(302), "Valor": "w"}},
			lecturas:  3,
		},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			var llamadas []registroMaterializacion
			var eventos []string
			var bajas []string
			registry := escenarioTardio(&llamadas, &eventos, &bajas)
			cfg := cfgConPoliticaHijo(caso.pol)
			if caso.inmutable {
				cfg.Relations[1].Write.ImmutableFields = []string{"Valor"}
			}
			body := map[string]any{
				"Id":          float64(100),
				"TerceroId":   float64(25),
				"Descripcion": "p cambiada",
				"Hijo":        caso.hijos,
			}

			_, err := Write(context.Background(), body, cfg, registry, WriteOptions{Verb: WriteVerbUpdate})

			ee := comoErrorEscritura(err)
			if ee == nil {
				t.Fatalf("se esperaba *ErrorEscritura por la violación del último nodo; err=%v", err)
			}
			if !contieneTexto(ee.Path, "Hijo") {
				t.Errorf("el error debe nombrar la arista Hijo; path=%q", ee.Path)
			}
			lecturas := 0
			for _, ev := range eventos {
				if strings.HasPrefix(ev, "R:") {
					lecturas++
				}
				if strings.HasPrefix(ev, "M:") {
					t.Errorf("un 400 no debe materializar ninguna entidad; eventos=%v", eventos)
				}
			}
			if lecturas != caso.lecturas {
				t.Errorf("el pre-pass debe recuperar cada nodo visitado exactamente una vez (esperadas %d); lecturas=%d eventos=%v",
					caso.lecturas, lecturas, eventos)
			}
			if len(llamadas) != 0 {
				t.Errorf("un 400 no debe enviar creaciones, actualizaciones ni bajas; llamadas=%+v", llamadas)
			}
			if len(bajas) != 0 {
				t.Errorf("un 400 no debe enviar bajas; bajas=%v", bajas)
			}
		})
	}
}

// En un árbol válido con creaciones, actualizaciones y nodos sin cambios, todas las lecturas
// del pre-pass (una por nodo visitado) ocurren antes de la primera escritura a un CRUD.
func TestWriteTodasLasLecturasAntesDeEscribir(t *testing.T) {
	var llamadas []registroMaterializacion
	var eventos []string
	var bajas []string
	registry := escenarioTardio(&llamadas, &eventos, &bajas)
	body := map[string]any{
		"Id":          float64(100),
		"TerceroId":   float64(25),
		"Descripcion": "p cambiada",
		"Hijo": []any{
			map[string]any{"Id": float64(300), "Valor": "z"},
			map[string]any{"Valor": "nuevo"},
			map[string]any{"Id": float64(302), "Valor": "c"},
		},
	}

	_, err := Write(context.Background(), body,
		cfgConPoliticaHijo(config.WriteVerbPolicy{Creatable: true, Updatable: true, Deletable: true}),
		registry, WriteOptions{Verb: WriteVerbUpdate})
	if err != nil {
		t.Fatalf("un árbol válido no debe fallar: %v", err)
	}

	ultimaLectura, primeraEscritura, lecturas := -1, -1, 0
	for i, ev := range eventos {
		switch {
		case strings.HasPrefix(ev, "R:"):
			lecturas++
			ultimaLectura = i
		case strings.HasPrefix(ev, "M:") && primeraEscritura < 0:
			primeraEscritura = i
		}
	}
	if lecturas != 4 {
		t.Errorf("la raíz y sus tres hijos deben recuperarse una vez cada uno; lecturas=%d eventos=%v", lecturas, eventos)
	}
	if primeraEscritura < 0 {
		t.Fatalf("un árbol válido con cambios debe materializar escrituras; eventos=%v", eventos)
	}
	if ultimaLectura > primeraEscritura {
		t.Errorf("todas las lecturas deben preceder a la primera escritura; eventos=%v", eventos)
	}
}

// Un campo inmutable reenviado con el mismo valor que el registro existente no es una
// modificación: la escritura continúa y no se rechaza.
func TestWriteInmutableReenvioIgualAlAlmacenadoSeAcepta(t *testing.T) {
	var llamadas []registroMaterializacion
	var eventos []string
	var bajas []string
	registry := escenarioTardio(&llamadas, &eventos, &bajas)
	cfg := cfgConPoliticaHijo(config.WriteVerbPolicy{Creatable: true, Updatable: true, Deletable: true})
	cfg.Relations[1].Write.ImmutableFields = []string{"Valor"}
	body := map[string]any{
		"Id":          float64(100),
		"TerceroId":   float64(25),
		"Descripcion": "p cambiada",
		"Hijo":        []any{map[string]any{"Id": float64(300), "Valor": "a"}},
	}

	if _, err := Write(context.Background(), body, cfg, registry, WriteOptions{Verb: WriteVerbUpdate}); err != nil {
		t.Fatalf("un inmutable con el valor almacenado no debe rechazarse: %v", err)
	}
	if len(llamadas) == 0 {
		t.Errorf("la escritura debe continuar y materializar el cambio de la raíz")
	}
}

// El motor emite un evento por fase y nodo. En un árbol válido, todas las lecturas y decisiones
// ocurren antes de la primera escritura, y cada nodo se recupera y decide exactamente una vez.
func TestWriteObservaFasesAntesDeEscribir(t *testing.T) {
	var llamadas []registroMaterializacion
	var eventos []string
	var bajas []string
	registry := escenarioTardio(&llamadas, &eventos, &bajas)
	var observados []eventoEscritura
	body := map[string]any{
		"Id":          float64(100),
		"TerceroId":   float64(25),
		"Descripcion": "p cambiada",
		"Hijo": []any{
			map[string]any{"Id": float64(300), "Valor": "z"},
			map[string]any{"Valor": "nuevo"},
			map[string]any{"Id": float64(302), "Valor": "c"},
		},
	}

	_, err := writeObservado(context.Background(), body,
		cfgConPoliticaHijo(config.WriteVerbPolicy{Creatable: true, Updatable: true, Deletable: true}),
		registry, WriteOptions{Verb: WriteVerbUpdate}, func(e eventoEscritura) { observados = append(observados, e) })
	if err != nil {
		t.Fatalf("un árbol válido no debe fallar: %v", err)
	}

	if n := contarFase(observados, faseLectura); n != 4 {
		t.Errorf("cada nodo debe recuperarse una vez (4 nodos); lecturas=%d", n)
	}
	if n := contarFase(observados, faseDecision); n != 4 {
		t.Errorf("cada nodo debe decidirse una vez (4 nodos); decisiones=%d", n)
	}
	primeraEscritura := primerIndiceFase(observados, faseEscritura)
	if primeraEscritura < 0 {
		t.Fatalf("un árbol válido con cambios debe materializar escrituras; eventos=%+v", observados)
	}
	if ultimaDecision := ultimoIndiceFase(observados, faseDecision); ultimaDecision > primeraEscritura {
		t.Errorf("toda decisión debe preceder a la primera escritura; eventos=%+v", observados)
	}
	if ultimaLectura := ultimoIndiceFase(observados, faseLectura); ultimaLectura > primeraEscritura {
		t.Errorf("toda lectura debe preceder a la primera escritura; eventos=%+v", observados)
	}
}

// Con una violación en un nodo posterior, el motor decide sobre todos los nodos antes de
// rechazar y no emite ninguna escritura.
func TestWriteObservaViolacionTardiaSinEscrituras(t *testing.T) {
	var llamadas []registroMaterializacion
	var eventos []string
	var bajas []string
	registry := escenarioTardio(&llamadas, &eventos, &bajas)
	var observados []eventoEscritura
	body := map[string]any{
		"Id":          float64(100),
		"TerceroId":   float64(25),
		"Descripcion": "p cambiada",
		"Hijo": []any{
			map[string]any{"Id": float64(300), "Valor": "z"},
			map[string]any{"Id": float64(301), "Valor": "b"},
		},
	}

	_, err := writeObservado(context.Background(), body,
		cfgConPoliticaHijo(config.WriteVerbPolicy{Creatable: true, Updatable: true, Deletable: true}),
		registry, WriteOptions{Verb: WriteVerbUpdate}, func(e eventoEscritura) { observados = append(observados, e) })
	if comoErrorEscritura(err) == nil {
		t.Fatalf("se esperaba *ErrorEscritura por el reparent; err=%v", err)
	}
	if n := contarFase(observados, faseEscritura); n != 0 {
		t.Errorf("un 400 no debe emitir escrituras; escrituras=%d eventos=%+v", n, observados)
	}
	if n := contarFase(observados, faseDecision); n != 2 {
		t.Errorf("solo los nodos validados con éxito emiten decisión (raíz y primer hijo); decisiones=%d", n)
	}
}

// contarFase cuenta los eventos de una fase.
func contarFase(eventos []eventoEscritura, fase string) int {
	n := 0
	for _, e := range eventos {
		if e.Fase == fase {
			n++
		}
	}
	return n
}

// primerIndiceFase devuelve la posición del primer evento de la fase, o -1.
func primerIndiceFase(eventos []eventoEscritura, fase string) int {
	for i, e := range eventos {
		if e.Fase == fase {
			return i
		}
	}
	return -1
}

// ultimoIndiceFase devuelve la posición del último evento de la fase, o -1.
func ultimoIndiceFase(eventos []eventoEscritura, fase string) int {
	ultimo := -1
	for i, e := range eventos {
		if e.Fase == fase {
			ultimo = i
		}
	}
	return ultimo
}
