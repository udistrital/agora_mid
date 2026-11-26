package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/udistrital/agora_mid/clients"
)

const (
	ttlCatalogoPositivo = 5 * time.Minute
	ttlCatalogoNegativo = 30 * time.Second
)

// CatalogosValidator valida la existencia y vigencia de catálogos externos antes
// de escribir. Se abstrae como interfaz para poder sustituirlo en pruebas.
type CatalogosValidator interface {
	TipoDocumentoExiste(ctx context.Context, id int64) (bool, error)
	LugarExiste(ctx context.Context, id int64) (bool, error)
}

// entradaCache guarda el resultado de una consulta y el momento en que vence.
type entradaCache struct {
	existe bool
	expira time.Time
}

// CatalogosService consulta terceros_crud y ubicaciones_crud con una caché en
// memoria por Id. Los resultados positivos y los registros inactivos se cachean
// por ttlCatalogoPositivo; los 404 por ttlCatalogoNegativo. Los errores de
// dependencia no se cachean.
type CatalogosService struct {
	Client             clients.HTTPClient
	TercerosBaseURL    string
	UbicacionesBaseURL string
	cache              *sync.Map
	ttlPositivo        time.Duration
	ttlNegativo        time.Duration
	ahora              func() time.Time
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
		ttlPositivo:        ttlCatalogoPositivo,
		ttlNegativo:        ttlCatalogoNegativo,
		ahora:              time.Now,
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
	if entrada, ok := s.leerCache(clave); ok {
		return entrada, nil
	}

	var raw json.RawMessage
	destino := fmt.Sprintf("%s/v1/%s/%d", base, recurso, id)
	status, err := s.Client.Get(ctx, destino, &raw)
	if err != nil {
		if status == 404 {
			s.guardar(clave, false, s.ttlNegativo)
			return false, nil
		}
		return false, err
	}

	registro, err := decodeAPIRecord(raw)
	if err != nil {
		return false, err
	}
	activo, _ := registro["Activo"].(bool)
	s.guardar(clave, activo, s.ttlPositivo)
	return activo, nil
}

// leerCache devuelve el valor vigente y descarta la entrada si venció.
func (s CatalogosService) leerCache(clave string) (bool, bool) {
	if s.cache == nil {
		return false, false
	}
	valor, ok := s.cache.Load(clave)
	if !ok {
		return false, false
	}
	entrada, ok := valor.(entradaCache)
	if !ok {
		s.cache.Delete(clave)
		return false, false
	}
	if s.reloj().Before(entrada.expira) {
		return entrada.existe, true
	}
	s.cache.Delete(clave)
	return false, false
}

func (s CatalogosService) guardar(clave string, valor bool, ttl time.Duration) {
	if s.cache == nil || ttl <= 0 {
		return
	}
	s.cache.Store(clave, entradaCache{
		existe: valor,
		expira: s.reloj().Add(ttl),
	})
}

func (s CatalogosService) reloj() time.Time {
	if s.ahora != nil {
		return s.ahora()
	}
	return time.Now()
}
