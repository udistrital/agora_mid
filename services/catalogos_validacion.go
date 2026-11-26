package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/udistrital/agora_mid/clients"
)

// CatalogosValidator valida la existencia y vigencia de catálogos externos antes
// de escribir. Se abstrae como interfaz para poder sustituirlo en pruebas.
type CatalogosValidator interface {
	TipoDocumentoExiste(ctx context.Context, id int64) (bool, error)
	LugarExiste(ctx context.Context, id int64) (bool, error)
}

// CatalogosService consulta terceros_crud y ubicaciones_crud con una caché en
// memoria por Id. Una respuesta 404 se cachea como inexistente; los errores de
// dependencia no se cachean.
type CatalogosService struct {
	Client             clients.HTTPClient
	TercerosBaseURL    string
	UbicacionesBaseURL string
	cache              *sync.Map
}

// NewCatalogosService construye el validador de catálogos.
func NewCatalogosService(
	client clients.HTTPClient,
	tercerosBaseURL string,
	ubicacionesBaseURL string,
) CatalogosService {
	return CatalogosService{
		Client:             client,
		TercerosBaseURL:    tercerosBaseURL,
		UbicacionesBaseURL: ubicacionesBaseURL,
		cache:              &sync.Map{},
	}
}

// TipoDocumentoExiste comprueba que el tipo de documento exista y esté activo.
func (s CatalogosService) TipoDocumentoExiste(ctx context.Context, id int64) (bool, error) {
	return s.existe(ctx, "tipo_documento", s.TercerosBaseURL, id)
}

// LugarExiste comprueba que el lugar exista y esté activo en ubicaciones_crud.
func (s CatalogosService) LugarExiste(ctx context.Context, id int64) (bool, error) {
	return s.existe(ctx, "lugar", s.UbicacionesBaseURL, id)
}

func (s CatalogosService) existe(
	ctx context.Context,
	recurso string,
	base string,
	id int64,
) (bool, error) {
	if id <= 0 {
		return false, nil
	}
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		return false, fmt.Errorf("no hay URL base configurada para %q", recurso)
	}

	clave := recurso + ":" + strconv.FormatInt(id, 10)
	if s.cache != nil {
		if valor, ok := s.cache.Load(clave); ok {
			return valor.(bool), nil
		}
	}

	var raw json.RawMessage
	destino := fmt.Sprintf("%s/v1/%s/%d", base, recurso, id)
	status, err := s.Client.Get(ctx, destino, &raw)
	if err != nil {
		if status == 404 {
			s.guardar(clave, false)
			return false, nil
		}
		return false, err
	}

	registro, err := decodeAPIRecord(raw)
	if err != nil {
		return false, err
	}
	activo, _ := registro["Activo"].(bool)
	s.guardar(clave, activo)
	return activo, nil
}

func (s CatalogosService) guardar(clave string, valor bool) {
	if s.cache != nil {
		s.cache.Store(clave, valor)
	}
}
