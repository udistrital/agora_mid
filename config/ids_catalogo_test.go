package config

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type resolvedorCatalogoFake struct {
	natural  int
	juridica int
	nit      int
	err      error

	llamadasNatural  int
	llamadasJuridica int
	llamadasNIT      int
}

func (r *resolvedorCatalogoFake) ResolverTipoContribuyente(
	_ context.Context,
	codigo string,
) (int, error) {
	if r.err != nil {
		return 0, r.err
	}
	switch codigo {
	case "P_NATURAL":
		r.llamadasNatural++
		return r.natural, nil
	case "P_JURIDICA":
		r.llamadasJuridica++
		return r.juridica, nil
	default:
		return 0, errors.New("código inesperado: " + codigo)
	}
}

func (r *resolvedorCatalogoFake) ResolverTipoDocumento(
	_ context.Context,
	codigo string,
) (int, error) {
	if r.err != nil {
		return 0, r.err
	}
	if codigo != "NIT" {
		return 0, errors.New("código inesperado: " + codigo)
	}
	r.llamadasNIT++
	return r.nit, nil
}

func entornoFake(valores map[string]string) func(string) string {
	return func(clave string) string {
		return valores[clave]
	}
}

func TestCargarIDsCatalogoUsaEntornoSinConsultar(t *testing.T) {
	resolver := &resolvedorCatalogoFake{natural: 90, juridica: 91, nit: 92}
	ids, err := CargarIDsCatalogo(context.Background(), entornoFake(map[string]string{
		EnvTipoContribuyenteNatural:  "1",
		EnvTipoContribuyenteJuridica: "2",
		EnvTipoDocumentoNIT:          "7",
	}), resolver)
	if err != nil {
		t.Fatalf("CargarIDsCatalogo no debe fallar: %v", err)
	}

	if ids.TipoContribuyenteNatural.Valor != 1 ||
		ids.TipoContribuyenteJuridica.Valor != 2 ||
		ids.TipoDocumentoNIT.Valor != 7 {
		t.Fatalf("ids inesperados: %+v", ids)
	}
	if ids.TipoContribuyenteNatural.Origen != OrigenEntorno ||
		ids.TipoContribuyenteJuridica.Origen != OrigenEntorno ||
		ids.TipoDocumentoNIT.Origen != OrigenEntorno {
		t.Fatalf("el origen debía ser entorno: %+v", ids)
	}
	if resolver.llamadasNatural+resolver.llamadasJuridica+resolver.llamadasNIT != 0 {
		t.Fatalf("no debía consultar el catálogo: %+v", resolver)
	}
}

func TestCargarIDsCatalogoConsultaCuandoFaltaEntorno(t *testing.T) {
	resolver := &resolvedorCatalogoFake{natural: 1, juridica: 2, nit: 7}
	ids, err := CargarIDsCatalogo(context.Background(), entornoFake(nil), resolver)
	if err != nil {
		t.Fatalf("CargarIDsCatalogo no debe fallar: %v", err)
	}

	if ids.TipoContribuyenteNatural.Valor != 1 ||
		ids.TipoContribuyenteJuridica.Valor != 2 ||
		ids.TipoDocumentoNIT.Valor != 7 {
		t.Fatalf("ids inesperados: %+v", ids)
	}
	if ids.TipoContribuyenteNatural.Origen != OrigenCatalogo ||
		ids.TipoContribuyenteJuridica.Origen != OrigenCatalogo ||
		ids.TipoDocumentoNIT.Origen != OrigenCatalogo {
		t.Fatalf("el origen debía ser catálogo: %+v", ids)
	}
	if resolver.llamadasNatural != 1 || resolver.llamadasJuridica != 1 || resolver.llamadasNIT != 1 {
		t.Fatalf("llamadas inesperadas: %+v", resolver)
	}
}

func TestCargarIDsCatalogoMezclaEntornoYCatalogo(t *testing.T) {
	resolver := &resolvedorCatalogoFake{natural: 90, juridica: 2, nit: 7}
	ids, err := CargarIDsCatalogo(context.Background(), entornoFake(map[string]string{
		EnvTipoContribuyenteNatural: "1",
	}), resolver)
	if err != nil {
		t.Fatalf("CargarIDsCatalogo no debe fallar: %v", err)
	}

	if ids.TipoContribuyenteNatural.Origen != OrigenEntorno {
		t.Errorf("P_NATURAL debía venir del entorno")
	}
	if ids.TipoContribuyenteJuridica.Origen != OrigenCatalogo {
		t.Errorf("P_JURIDICA debía venir del catálogo")
	}
	if ids.TipoDocumentoNIT.Origen != OrigenCatalogo {
		t.Errorf("NIT debía venir del catálogo")
	}
	if resolver.llamadasNatural != 0 || resolver.llamadasJuridica != 1 || resolver.llamadasNIT != 1 {
		t.Fatalf("llamadas inesperadas: %+v", resolver)
	}
}

func TestCargarIDsCatalogoRechazaEntornoInvalido(t *testing.T) {
	casos := []string{"abc", "0", "-1", "7.5"}
	for _, valor := range casos {
		t.Run(valor, func(t *testing.T) {
			resolver := &resolvedorCatalogoFake{natural: 1, juridica: 2, nit: 7}
			_, err := CargarIDsCatalogo(context.Background(), entornoFake(map[string]string{
				EnvTipoContribuyenteNatural: valor,
			}), resolver)
			if err == nil {
				t.Fatalf("se esperaba error para %q", valor)
			}
			if !strings.Contains(err.Error(), EnvTipoContribuyenteNatural) {
				t.Fatalf("el error debe nombrar la variable: %v", err)
			}
		})
	}
}

func TestCargarIDsCatalogoRechazaVariantesIguales(t *testing.T) {
	_, err := CargarIDsCatalogo(context.Background(), entornoFake(map[string]string{
		EnvTipoContribuyenteNatural:  "1",
		EnvTipoContribuyenteJuridica: "1",
		EnvTipoDocumentoNIT:          "7",
	}), &resolvedorCatalogoFake{})
	if err == nil {
		t.Fatal("se esperaba error cuando P_NATURAL y P_JURIDICA comparten Id")
	}
}

func TestCargarIDsCatalogoPropagaErrorDeCatalogo(t *testing.T) {
	resolver := &resolvedorCatalogoFake{err: errors.New("catálogo caído")}
	_, err := CargarIDsCatalogo(context.Background(), entornoFake(nil), resolver)
	if err == nil {
		t.Fatal("se esperaba error del catálogo")
	}
	if !strings.Contains(err.Error(), "P_NATURAL") {
		t.Fatalf("el error debe indicar el código: %v", err)
	}
}

func TestCargarIDsCatalogoRechazaIdInvalidoDelCatalogo(t *testing.T) {
	resolver := &resolvedorCatalogoFake{natural: 0, juridica: 2, nit: 7}
	if _, err := CargarIDsCatalogo(context.Background(), entornoFake(nil), resolver); err == nil {
		t.Fatal("se esperaba error por Id inválido del catálogo")
	}
}

func TestCargarIDsCatalogoRechazaResolverNulo(t *testing.T) {
	if _, err := CargarIDsCatalogo(context.Background(), entornoFake(nil), nil); err == nil {
		t.Fatal("se esperaba error por resolvedor nulo")
	}
}
