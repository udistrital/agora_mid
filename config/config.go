package config

import (
	"bufio"
	"os"
	"strings"
)

// Config contiene la configuración de los servicios consumidos por Agora MID.
type Config struct {
	TercerosCRUDURL        string
	ParametrosCRUDURL      string
	GestionPersonasCRUDURL string
}

// Load lee la configuración desde conf/app.conf y expande variables de entorno.
func Load() Config {
	values := make(map[string]string)
	file, err := os.Open("conf/app.conf")
	if err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
				continue
			}
			key, value, found := strings.Cut(line, "=")
			if !found {
				continue
			}
			values[strings.ToLower(strings.TrimSpace(key))] = os.ExpandEnv(strings.Trim(strings.TrimSpace(value), "\"'"))
		}
	}

	return Config{
		TercerosCRUDURL:        values["terceros_crud_url"],
		ParametrosCRUDURL:      values["parametros_crud_url"],
		GestionPersonasCRUDURL: values["gestion_personas_crud_url"],
	}
}
