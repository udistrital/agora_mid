package config

import "os"

// Config contiene la configuración de los servicios consumidos por Agora MID.
type Config struct {
	TercerosCRUDURL        string
	ParametrosCRUDURL      string
	GestionPersonasCRUDURL string
}

// Load obtiene la configuración desde variables de entorno.
func Load() Config {
	return Config{
		TercerosCRUDURL:        os.Getenv("TERCEROS_CRUD_BASE_URL"),
		ParametrosCRUDURL:      os.Getenv("PARAMETROS_CRUD_BASE_URL"),
		GestionPersonasCRUDURL: os.Getenv("GESTION_PERSONAS_CRUD_BASE_URL"),
	}
}
