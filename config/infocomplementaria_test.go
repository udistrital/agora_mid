package config

import (
	"strings"
	"testing"
)

func TestCargarOverridesInfoComplementaria(t *testing.T) {
	overrides, err := CargarOverridesInfoComplementaria(func(clave string) string {
		switch clave {
		case "MID_INFOCOMP_DESCRIPCION":
			return " 53 "
		case "MID_INFOCOMP_EPS":
			return "308"
		default:
			return ""
		}
	})
	if err != nil {
		t.Fatalf("CargarOverridesInfoComplementaria no debe fallar: %v", err)
	}
	if overrides["DESCRIPCIÓN"] != 53 || overrides["EPS"] != 308 {
		t.Fatalf("overrides inesperados: %+v", overrides)
	}
	if _, existe := overrides["MIGRANTE"]; existe {
		t.Fatalf("MIGRANTE no estaba definido: %+v", overrides)
	}
}

func TestCargarOverridesInfoComplementariaRechazaValorInvalido(t *testing.T) {
	_, err := CargarOverridesInfoComplementaria(func(clave string) string {
		if clave == "MID_INFOCOMP_MIGRANTE" {
			return "abc"
		}
		return ""
	})
	if err == nil {
		t.Fatal("se esperaba error para un override inválido")
	}
	if !strings.Contains(err.Error(), "MID_INFOCOMP_MIGRANTE") {
		t.Fatalf("el error debe nombrar la variable: %v", err)
	}
}
