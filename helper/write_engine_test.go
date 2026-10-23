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
	servicio   string
	siguienteID float64
	llamadas   *[]registroMaterializacion
	fallarCon  error
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
				Name:         "Detalle",
				RelatedService: "detalle",
				MappedName:   "Detalle",
				RelatedField: "Id",
				Cardinality:  config.CardinalityOne,
				Write: &config.WriteConfig{
					WritableFields: []string{"Nota", "TipoContribuyenteId"},
					KeyPropagation: config.KeyPropagationSharedPK,
					Policy:         config.WriteVerbPolicy{Creatable: true, Updatable: true},
				},
			},
			{
				Name:         "Hijo",
				RelatedService: "hijo",
				MappedName:   "Hijo",
				RelatedField: "PadreId",
				Cardinality:  config.CardinalityMany,
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
			"Nota":             "x",
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

	// shared_pk: el detalle recibe el Id de la raíz (100) como su propio Id.
	detalle, _ := res.Data["Detalle"].(map[string]any)
	if detalle == nil || aCadena(detalle["Id"]) != "100" {
		t.Errorf("Detalle (shared_pk) debe heredar el Id de la raíz 100; detalle=%+v", detalle)
	}

	// fk: cada hijo recibe PadreId = Id de la raíz (100).
	hijos, _ := res.Data["Hijo"].([]any)
	if len(hijos) != 2 {
		t.Fatalf("se esperaban 2 hijos materializados; recibidos=%d", len(hijos))
	}
	for i, h := range hijos {
		hm, _ := h.(map[string]any)
		if aCadena(hm["PadreId"]) != "100" {
			t.Errorf("hijo %d debe tener PadreId=100 (fk); hijo=%+v", i, hm)
		}
		if aCadena(hm["Id"]) != "300" {
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
			"Nota":             "x",
			"TipoContribuyente": map[string]any{"Id": float64(2)},
		},
	}

	if _, err := Write(context.Background(), body, cfgEscritura(), registry, WriteOptions{}); err != nil {
		t.Fatalf("Write no debe fallar: %v", err)
	}

	// Llamada de la raíz: clave natural TerceroId=25.
	var raiz, detalle *registroMaterializacion
	for i := range llamadas {
		switch llamadas[i].servicio {
		case "raiz":
			raiz = &llamadas[i]
		case "detalle":
			detalle = &llamadas[i]
		}
	}
	if raiz == nil || len(raiz.keys) != 1 || raiz.keys[0] != "25" {
		t.Errorf("la raíz debe recibir la clave natural [25]; raiz=%+v", raiz)
	}
	if detalle == nil {
		t.Fatalf("el detalle debe materializarse")
	}
	// El parámetro anidado se aplanó a TipoContribuyenteId=2.
	if aCadena(detalle.payload["TipoContribuyenteId"]) != "2" {
		t.Errorf("el parámetro anidado debe aplanarse a TipoContribuyenteId=2; payload=%+v", detalle.payload)
	}
	if _, sigueAnidado := detalle.payload["TipoContribuyente"]; sigueAnidado {
		t.Errorf("el objeto anidado no debe pasar al resolver; payload=%+v", detalle.payload)
	}
}

// Un campo no escribible se rechaza con *ErrorEscritura y no materializa nada.
func TestWriteRechazoCampoNoEscribible(t *testing.T) {
	var llamadas []registroMaterializacion
	registry := registryFalso(&llamadas, nil)

	body := map[string]any{
		"TerceroId": float64(25),
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
