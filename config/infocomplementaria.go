package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// VariablesInfoComplementaria asocia cada código de info_complementaria con la
// variable de entorno que puede fijar su Id. Sirve para resolver códigos
// duplicados o para apuntar un campo a un registro concreto sin tocar
// terceros_crud.
var VariablesInfoComplementaria = map[string]string{
	"DESCRIPCIÓN":             "MID_INFOCOMP_DESCRIPCION",
	"GENERO":                  "MID_INFOCOMP_GENERO",
	"GRUPO_ETNICO":            "MID_INFOCOMP_GRUPO_ETNICO",
	"SE_RECONOCE_COMO":        "MID_INFOCOMP_SE_RECONOCE_COMO",
	"ORIENTACION_SEXUAL":      "MID_INFOCOMP_ORIENTACION_SEXUAL",
	"ESTADO_CIVIL":            "MID_INFOCOMP_ESTADO_CIVIL",
	"MIGRANTE":                "MID_INFOCOMP_MIGRANTE",
	"EPS":                     "MID_INFOCOMP_EPS",
	"CUENTA_AFC":              "MID_INFOCOMP_CUENTA_AFC",
	"VICTIMA_CONFLICTO":       "MID_INFOCOMP_VICTIMA_CONFLICTO",
	"PERSONA_PEP":             "MID_INFOCOMP_PERSONA_PEP",
	"DIRECCIÓN":               "MID_INFOCOMP_DIRECCION",
	"LUGAR_RESIDENCIA":        "MID_INFOCOMP_LUGAR_RESIDENCIA",
	"HIJOS":                   "MID_INFOCOMP_HIJOS",
	"NUMERO_HIJOS":            "MID_INFOCOMP_NUMERO_HIJOS",
	"CABEZA_FAMILIA":          "MID_INFOCOMP_CABEZA_FAMILIA",
	"PERSONAS_A_CARGO":        "MID_INFOCOMP_PERSONAS_A_CARGO",
	"DECLARANTE_RENTA":        "MID_INFOCOMP_DECLARANTE_RENTA",
	"DISCAPACIDAD":            "MID_INFOCOMP_DISCAPACIDAD",
	"SITIO_WEB":               "MID_INFOCOMP_SITIO_WEB",
	"AUTORIZACION_GIRO":       "MID_INFOCOMP_AUTORIZACION_GIRO",
	"APLICA_RUP":              "MID_INFOCOMP_APLICA_RUP",
	"ACTIVIDAD_PRINCIPAL":     "MID_INFOCOMP_ACTIVIDAD_PRINCIPAL",
	"PAIS_EXPEDICION":         "MID_INFOCOMP_PAIS_EXPEDICION",
	"DEPARTAMENTO_EXPEDICION": "MID_INFOCOMP_DEPARTAMENTO_EXPEDICION",
	"AFP":                     "MID_INFOCOMP_AFP",
	"CCF":                     "MID_INFOCOMP_CCF",
}

// CargarOverridesInfoComplementaria lee las variables MID_INFOCOMP_* definidas
// y devuelve un mapa código→Id. Las variables ausentes o vacías se ignoran. Un
// valor inválido devuelve error para no arrancar con IDs incorrectos.
func CargarOverridesInfoComplementaria(getenv func(string) string) (map[string]int64, error) {
	if getenv == nil {
		getenv = os.Getenv
	}
	overrides := make(map[string]int64)
	for codigo, variable := range VariablesInfoComplementaria {
		valor := strings.TrimSpace(getenv(variable))
		if valor == "" {
			continue
		}
		id, err := strconv.ParseInt(valor, 10, 64)
		if err != nil || id <= 0 {
			return nil, fmt.Errorf("%s debe ser un entero positivo: %q", variable, valor)
		}
		overrides[codigo] = id
	}
	return overrides, nil
}
