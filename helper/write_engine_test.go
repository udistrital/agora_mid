package helper

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/udistrital/agora_mid/config"
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
