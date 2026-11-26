package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCargarDotEnvLeeVariablesSinSobrescribir(t *testing.T) {
	const (
		claveExistente = "AGORA_MID_DOTENV_PRUEBA_EXISTENTE"
		claveNueva     = "AGORA_MID_DOTENV_PRUEBA_NUEVA"
		claveSimple    = "AGORA_MID_DOTENV_PRUEBA_SIMPLE"
		claveVacia     = "AGORA_MID_DOTENV_PRUEBA_VACIA"
		claveHash      = "AGORA_MID_DOTENV_PRUEBA_HASH"
	)
	limpiarVariables(t, claveNueva, claveSimple, claveVacia, claveHash)
	t.Setenv(claveExistente, "previo")

	ruta := filepath.Join(t.TempDir(), ".env")
	contenido := "" +
		"# comentario\n" +
		"\n" +
		claveExistente + "=nuevo\n" +
		"export " + claveNueva + "=\"valor con espacios\"\n" +
		claveSimple + "='simple'\n" +
		claveVacia + "=\n" +
		claveHash + "=valor#noescomentario\n"
	if err := os.WriteFile(ruta, []byte(contenido), 0o600); err != nil {
		t.Fatalf("escribiendo .env: %v", err)
	}

	definidas, err := CargarDotEnv(ruta)
	if err != nil {
		t.Fatalf("CargarDotEnv no debe fallar: %v", err)
	}
	if definidas != 4 {
		t.Fatalf("se esperaban 4 variables nuevas, obtuvo %d", definidas)
	}

	casos := map[string]string{
		claveExistente: "previo",
		claveNueva:     "valor con espacios",
		claveSimple:    "simple",
		claveVacia:     "",
		claveHash:      "valor#noescomentario",
	}
	for clave, esperado := range casos {
		if obtenido := os.Getenv(clave); obtenido != esperado {
			t.Errorf("%s: esperado %q, obtuvo %q", clave, esperado, obtenido)
		}
	}
}

func TestCargarDotEnvIgnoraArchivoInexistente(t *testing.T) {
	definidas, err := CargarDotEnv(filepath.Join(t.TempDir(), "no-existe.env"))
	if err != nil {
		t.Fatalf("un archivo inexistente no debe ser error: %v", err)
	}
	if definidas != 0 {
		t.Fatalf("no debía definir variables, definió %d", definidas)
	}
}

func TestCargarDotEnvRechazaLineaInvalida(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(ruta, []byte("SIN_IGUAL\n"), 0o600); err != nil {
		t.Fatalf("escribiendo .env: %v", err)
	}

	if _, err := CargarDotEnv(ruta); err == nil {
		t.Fatal("se esperaba error por línea sin '='")
	}
}

func TestCargarDotEnvContinuaTrasLineaInvalida(t *testing.T) {
	const clave = "AGORA_MID_DOTENV_PRUEBA_TRAS_ERROR"
	limpiarVariables(t, clave)

	ruta := filepath.Join(t.TempDir(), ".env")
	contenido := "SIN_IGUAL\n" + clave + "=7\n"
	if err := os.WriteFile(ruta, []byte(contenido), 0o600); err != nil {
		t.Fatalf("escribiendo .env: %v", err)
	}

	definidas, err := CargarDotEnv(ruta)
	if err == nil {
		t.Fatal("se esperaba error por la línea inválida")
	}
	if definidas != 1 {
		t.Fatalf("se esperaba una variable válida, definió %d", definidas)
	}
	if obtenido := os.Getenv(clave); obtenido != "7" {
		t.Fatalf("%s: esperado %q, obtuvo %q", clave, "7", obtenido)
	}
}

func TestCargarDotEnvSoportaBOMYCRLF(t *testing.T) {
	const clave = "AGORA_MID_DOTENV_PRUEBA_CRLF"
	limpiarVariables(t, clave)

	ruta := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(ruta, []byte("\ufeff"+clave+"=7\r\n"), 0o600); err != nil {
		t.Fatalf("escribiendo .env: %v", err)
	}

	if _, err := CargarDotEnv(ruta); err != nil {
		t.Fatalf("CargarDotEnv no debe fallar: %v", err)
	}
	if obtenido := os.Getenv(clave); obtenido != "7" {
		t.Fatalf("%s: esperado %q, obtuvo %q", clave, "7", obtenido)
	}
}

func limpiarVariables(t *testing.T, claves ...string) {
	t.Helper()
	for _, clave := range claves {
		_ = os.Unsetenv(clave)
		clave := clave
		t.Cleanup(func() {
			_ = os.Unsetenv(clave)
		})
	}
}
