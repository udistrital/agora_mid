package helper

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/udistrital/agora_mid/config"
	"github.com/udistrital/agora_mid/responses"
)

// tiempoAgotadoFake simula un error de red con Timeout() verdadero, como el que
// devuelve el cliente HTTP al agotar su tiempo de espera.
type tiempoAgotadoFake struct{}

func (tiempoAgotadoFake) Error() string   { return "i/o timeout" }
func (tiempoAgotadoFake) Timeout() bool   { return true }
func (tiempoAgotadoFake) Temporary() bool { return true }

// Las constantes de fallo de dependencia y de tiempo de espera deben estar
// declaradas en el catálogo con origen escritura.
func TestKindDeclaradosParaFalloDeDependenciaYTimeout(t *testing.T) {
	if KindDependencyFailure != "dependency_failure" {
		t.Errorf("KindDependencyFailure = %q; esperado dependency_failure", KindDependencyFailure)
	}
	if KindTimeout != "timeout" {
		t.Errorf("KindTimeout = %q; esperado timeout", KindTimeout)
	}
	for _, k := range []string{KindDependencyFailure, KindTimeout} {
		origen, ok := OrigenDeKind(k)
		if !ok {
			t.Errorf("la constante %q no está declarada en el catálogo", k)
			continue
		}
		if origen != KindOrigenEscritura {
			t.Errorf("la constante %q tiene origen %q; esperado escritura", k, origen)
		}
	}
}

// Un fallo tolerado de dependencia en la escritura se reporta como
// dependency_failure, sin exponer el mensaje interno del CRUD.
func TestAvisoEscrituraFalloDependenciaEmiteDependencyFailure(t *testing.T) {
	var ll []registroMaterializacion
	registry := registryFalso(&ll, map[string]error{"hijo": errors.New("503 service unavailable")})
	body := map[string]any{"TerceroId": float64(25), "Hijo": []any{map[string]any{"Valor": "a"}}}

	res, err := Write(context.Background(), body, cfgEscritura(), registry,
		WriteOptions{OnRelationError: config.OnRelationErrorNotify})
	if err != nil {
		t.Fatalf("modo tolerante no debe devolver error fatal: %v", err)
	}
	if len(res.Warnings) != 1 || res.Warnings[0].Kind != KindDependencyFailure {
		t.Fatalf("se esperaba un aviso dependency_failure; warnings=%+v", res.Warnings)
	}
	if strings.Contains(res.Warnings[0].Message, "503") {
		t.Errorf("el aviso no debe exponer el mensaje interno del CRUD; aviso=%+v", res.Warnings[0])
	}
}

// Un tiempo de espera agotado en la escritura se reporta como timeout, tanto
// si el error es context.DeadlineExceeded como si es un error de red con
// Timeout() verdadero.
func TestAvisoEscrituraTiempoAgotadoEmiteTimeout(t *testing.T) {
	casos := map[string]error{
		"contexto vencido": context.DeadlineExceeded,
		"error de red":     tiempoAgotadoFake{},
	}
	for nombre, causa := range casos {
		t.Run(nombre, func(t *testing.T) {
			var ll []registroMaterializacion
			registry := registryFalso(&ll, map[string]error{"hijo": causa})
			body := map[string]any{"TerceroId": float64(25), "Hijo": []any{map[string]any{"Valor": "a"}}}

			res, err := Write(context.Background(), body, cfgEscritura(), registry,
				WriteOptions{OnRelationError: config.OnRelationErrorNotify})
			if err != nil {
				t.Fatalf("modo tolerante no debe devolver error fatal: %v", err)
			}
			if len(res.Warnings) != 1 || res.Warnings[0].Kind != KindTimeout {
				t.Fatalf("se esperaba un aviso timeout; warnings=%+v", res.Warnings)
			}
		})
	}
}

// La clasificación del fallo se expresa solo por Kind: el aviso de MetaData.Errors
// no tiene ningún campo discriminador paralelo.
func TestClasificacionSoloPorKind(t *testing.T) {
	campos := []string{}
	tipo := reflect.TypeOf(responses.RelationError{})
	for i := 0; i < tipo.NumField(); i++ {
		campos = append(campos, tipo.Field(i).Name)
	}
	esperados := []string{"Path", "Service", "Message", "Kind"}
	if !reflect.DeepEqual(campos, esperados) {
		t.Errorf("los campos de MetaData.Errors deben ser %v; recibidos %v", esperados, campos)
	}
}
