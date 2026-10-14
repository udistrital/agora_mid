package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadLeeURLDesdeAppConf(t *testing.T) {
	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	temp := t.TempDir()
	if err := os.Mkdir(filepath.Join(temp, "conf"), 0o755); err != nil {
		t.Fatal(err)
	}
	content := "appname = test\nterceros_crud_url = http://terceros.test\ngestion_personas_crud_url = http://agora.test\nparametros_crud_url = http://parametros.test\n"
	if err := os.WriteFile(filepath.Join(temp, "conf", "app.conf"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(temp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(original) })

	got := Load()
	if got.TercerosCRUDURL != "http://terceros.test" || got.GestionPersonasCRUDURL != "http://agora.test" || got.ParametrosCRUDURL != "http://parametros.test" {
		t.Fatalf("configuración incorrecta: %+v", got)
	}
}

func TestLoadExpandeVariablesDeEntorno(t *testing.T) {
	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	temp := t.TempDir()
	if err := os.Mkdir(filepath.Join(temp, "conf"), 0o755); err != nil {
		t.Fatal(err)
	}

	content := "terceros_crud_url = ${TEST_TERCEROS_CRUD_URL}\n"
	if err := os.WriteFile(filepath.Join(temp, "conf", "app.conf"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("TEST_TERCEROS_CRUD_URL", "https://terceros.test")

	if err := os.Chdir(temp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(original) })

	got := Load()
	if got.TercerosCRUDURL != "https://terceros.test" {
		t.Fatalf("URL no expandida correctamente: %q", got.TercerosCRUDURL)
	}
}
