package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// generarSwaggerJSON debe producir un JSON cuyo contenido semántico coincida con
// el YAML de origen, y hacerlo de forma determinística (dos ejecuciones generan
// bytes idénticos).
func TestGenerarSwaggerJSONDeterministaYEquivalente(t *testing.T) {
	dir := t.TempDir()
	rutaYAML := filepath.Join(dir, "swagger.yml")
	rutaJSON := filepath.Join(dir, "swagger.json")

	yamlFuente := `swagger: "2.0"
info:
  title: prueba
  version: 0.0.1
paths:
  /proveedor:
    get:
      parameters:
        - name: id
          in: query
          required: true
          type: integer
        - name: fields
          in: query
          required: false
          type: string
      responses:
        "200":
          description: ok
`
	if err := os.WriteFile(rutaYAML, []byte(yamlFuente), 0o644); err != nil {
		t.Fatalf("no se pudo escribir el YAML de prueba: %v", err)
	}

	if err := generarSwaggerJSON(rutaYAML, rutaJSON); err != nil {
		t.Fatalf("primera generación falló: %v", err)
	}
	primero, err := os.ReadFile(rutaJSON)
	if err != nil {
		t.Fatalf("no se pudo leer el JSON generado: %v", err)
	}

	// Determinismo: regenerar produce exactamente los mismos bytes.
	if err := generarSwaggerJSON(rutaYAML, rutaJSON); err != nil {
		t.Fatalf("segunda generación falló: %v", err)
	}
	segundo, err := os.ReadFile(rutaJSON)
	if err != nil {
		t.Fatalf("no se pudo releer el JSON generado: %v", err)
	}
	if string(primero) != string(segundo) {
		t.Fatalf("la generación no es determinística:\nprimero=%s\nsegundo=%s", primero, segundo)
	}

	// Equivalencia semántica: el JSON representa el mismo modelo que el YAML.
	var desdeYAML, desdeJSON any
	if err := yaml.Unmarshal([]byte(yamlFuente), &desdeYAML); err != nil {
		t.Fatalf("no se pudo interpretar el YAML fuente: %v", err)
	}
	if err := json.Unmarshal(segundo, &desdeJSON); err != nil {
		t.Fatalf("el JSON generado no es válido: %v", err)
	}
	// Reserializar ambos a JSON canónico para comparar estructura y valores.
	a, _ := json.Marshal(desdeYAML)
	b, _ := json.Marshal(desdeJSON)
	if string(a) != string(b) {
		t.Fatalf("el JSON no es equivalente al YAML:\nyaml->json=%s\njson=%s", a, b)
	}
}

// generarSwaggerJSON debe fallar de forma explícita si el YAML no existe.
func TestGenerarSwaggerJSONYAMLInexistente(t *testing.T) {
	dir := t.TempDir()
	err := generarSwaggerJSON(filepath.Join(dir, "no_existe.yml"), filepath.Join(dir, "out.json"))
	if err == nil {
		t.Fatal("se esperaba error por YAML inexistente")
	}
}

// generarSwaggerJSON debe fallar de forma explícita si el YAML no es un
// documento válido, sin producir un JSON a partir de contenido corrupto.
func TestGenerarSwaggerJSONYAMLInvalido(t *testing.T) {
	dir := t.TempDir()
	rutaYAML := filepath.Join(dir, "swagger.yml")
	rutaJSON := filepath.Join(dir, "swagger.json")

	// YAML sintácticamente inválido (indentación inconsistente de un mapeo).
	if err := os.WriteFile(rutaYAML, []byte("swagger: \"2.0\"\n  mal: [sin cerrar\n"), 0o644); err != nil {
		t.Fatalf("no se pudo escribir el YAML inválido: %v", err)
	}

	err := generarSwaggerJSON(rutaYAML, rutaJSON)
	if err == nil {
		t.Fatal("se esperaba error por YAML inválido")
	}
	// No debe haber publicado un JSON a partir de un fuente corrupto.
	if _, statErr := os.Stat(rutaJSON); statErr == nil {
		t.Fatal("no se debe escribir swagger.json cuando el YAML es inválido")
	}
}

// generarSwaggerJSON no debe reescribir el archivo cuando el YAML no cambió
// desde la última generación (idempotencia en disco).
func TestGenerarSwaggerJSONIdempotenteEnDisco(t *testing.T) {
	dir := t.TempDir()
	rutaYAML := filepath.Join(dir, "swagger.yml")
	rutaJSON := filepath.Join(dir, "swagger.json")

	if err := os.WriteFile(rutaYAML, []byte("swagger: \"2.0\"\ninfo:\n  title: prueba\n"), 0o644); err != nil {
		t.Fatalf("no se pudo escribir el YAML de prueba: %v", err)
	}

	if err := generarSwaggerJSON(rutaYAML, rutaJSON); err != nil {
		t.Fatalf("primera generación falló: %v", err)
	}
	info1, err := os.Stat(rutaJSON)
	if err != nil {
		t.Fatalf("no se pudo consultar el JSON generado: %v", err)
	}

	// Esperar para que un posible cambio de mtime sea detectable.
	time.Sleep(10 * time.Millisecond)

	if err := generarSwaggerJSON(rutaYAML, rutaJSON); err != nil {
		t.Fatalf("segunda generación falló: %v", err)
	}
	info2, err := os.Stat(rutaJSON)
	if err != nil {
		t.Fatalf("no se pudo reconsultar el JSON generado: %v", err)
	}

	if !info1.ModTime().Equal(info2.ModTime()) {
		t.Fatalf("el archivo se reescribió sin cambios en el YAML: mtime %v -> %v",
			info1.ModTime(), info2.ModTime())
	}
}
