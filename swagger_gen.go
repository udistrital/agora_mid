package main

import (
	"encoding/json"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// generarSwaggerJSON deriva, de forma determinística, el documento swagger.json a
// partir de swagger.yml. Swagger 2.0 usa el mismo modelo de datos en YAML y JSON,
// por lo que la conversión es una reserialización 1:1: el YAML es la única fuente
// de verdad y el JSON se regenera en cada arranque para mantenerlo coherente.
//
// Escribe el JSON solo si su contenido cambió, para no tocar el archivo en cada
// arranque sin necesidad. Devuelve error si el YAML no existe o es inválido.
func generarSwaggerJSON(rutaYAML, rutaJSON string) error {
	datosYAML, err := os.ReadFile(rutaYAML)
	if err != nil {
		return fmt.Errorf("leyendo %s: %w", rutaYAML, err)
	}

	// yaml.v3 decodifica los mapas con claves string, compatibles directamente
	// con encoding/json; no se requiere conversión de tipos.
	var documento any
	if err := yaml.Unmarshal(datosYAML, &documento); err != nil {
		return fmt.Errorf("interpretando %s como YAML: %w", rutaYAML, err)
	}

	datosJSON, err := json.MarshalIndent(documento, "", "  ")
	if err != nil {
		return fmt.Errorf("serializando %s a JSON: %w", rutaYAML, err)
	}
	datosJSON = append(datosJSON, '\n')

	// Escribir solo si cambió, para que el arranque sea idempotente en disco.
	if actual, err := os.ReadFile(rutaJSON); err == nil && string(actual) == string(datosJSON) {
		return nil
	}

	if err := os.WriteFile(rutaJSON, datosJSON, 0o644); err != nil {
		return fmt.Errorf("escribiendo %s: %w", rutaJSON, err)
	}
	return nil
}
