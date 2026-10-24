package services

import (
	"github.com/udistrital/agora_mid/clients"
)

// TercerosService expone las consultas contra el CRUD de Terceros que el
// middleware necesita al margen del motor declarativo. Hoy solo resuelve
// catálogos por código (ver catalogos.go); `main.go` lo usa al arranque para
// resolver el tipo de contribuyente `P_NATURAL`.
type TercerosService struct {
	BaseURL string
	Client  clients.HTTPClient
}
