package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/udistrital/agora_mid/clients"
)

// parametroCache guarda el estado y el tipo de un parámetro.
type parametroCache struct {
	existe bool
	tipo   string
	tipoID int64
	expira time.Time
}

type tipoParametroCache struct {
	codigo string
	expira time.Time
}

// ParametrosService valida parámetros del CRUD de parámetros con caché por Id.
// Se usa para TipoCanalContactoId y ActividadEconomicaId (CIIU).
type ParametrosService struct {
	Client      clients.HTTPClient
	BaseURL     string
	cache       *sync.Map
	tipos       *sync.Map
	ttlPositivo time.Duration
	ttlNegativo time.Duration
	ahora       func() time.Time
}

// NewParametrosService construye el validador de parámetros.
func NewParametrosService(client clients.HTTPClient, baseURL string) ParametrosService {
	return ParametrosService{
		Client:      client,
		BaseURL:     baseURL,
		cache:       &sync.Map{},
		tipos:       &sync.Map{},
		ttlPositivo: ttlCatalogoPositivo,
		ttlNegativo: ttlCatalogoNegativo,
		ahora:       time.Now,
	}
}

// ParametroExiste comprueba que el parámetro exista y esté activo.
func (s ParametrosService) ParametroExiste(ctx context.Context, id int64) (bool, error) {
	param, existe, err := s.obtener(ctx, id)
	if err != nil {
		return false, err
	}
	return existe && param.existe, nil
}

// ParametroDeTipo comprueba que el parámetro exista, esté activo y pertenezca
// al tipo indicado. Si tipo está vacío solo valida existencia.
func (s ParametrosService) ParametroDeTipo(
	ctx context.Context,
	id int64,
	tipo string,
) (bool, error) {
	tipo = strings.TrimSpace(tipo)
	if tipo == "" {
		return s.ParametroExiste(ctx, id)
	}
	param, existe, err := s.obtener(ctx, id)
	if err != nil {
		return false, err
	}
	if !existe || !param.existe {
		return false, nil
	}
	if param.tipo != "" {
		return param.tipo == tipo, nil
	}
	if param.tipoID <= 0 {
		return false, nil
	}
	codigo, err := s.codigoTipo(ctx, param.tipoID)
	if err != nil {
		return false, err
	}
	return codigo == tipo, nil
}

func (s ParametrosService) obtener(
	ctx context.Context,
	id int64,
) (parametroCache, bool, error) {
	if id <= 0 {
		return parametroCache{}, false, nil
	}
	base := strings.TrimRight(strings.TrimSpace(s.BaseURL), "/")
	if base == "" {
		return parametroCache{}, false, fmt.Errorf("no hay URL base configurada para %q", "parametro")
	}

	clave := fmt.Sprintf("parametro:%d", id)
	if valor, ok := s.cache.Load(clave); ok {
		entrada, ok := valor.(parametroCache)
		if ok && s.reloj().Before(entrada.expira) {
			return entrada, true, nil
		}
		s.cache.Delete(clave)
	}

	var raw json.RawMessage
	status, err := s.Client.Get(ctx, fmt.Sprintf("%s/v1/parametro/%d", base, id), &raw)
	if err != nil {
		if status == 404 {
			entrada := parametroCache{existe: false, expira: s.reloj().Add(s.ttlNegativo)}
			s.cache.Store(clave, entrada)
			return entrada, false, nil
		}
		return parametroCache{}, false, err
	}
	registro, err := decodeAPIRecord(raw)
	if err != nil {
		return parametroCache{}, false, err
	}
	activo, _ := registro["Activo"].(bool)
	tipo := ""
	var tipoID int64
	if tipoParametro, ok := registro["TipoParametroId"].(map[string]any); ok {
		tipo, _ = tipoParametro["CodigoAbreviacion"].(string)
		tipoID = idDeRelacion(tipoParametro["Id"])
	}
	entrada := parametroCache{
		existe: activo,
		tipo:   strings.TrimSpace(tipo),
		tipoID: tipoID,
		expira: s.reloj().Add(s.ttlPositivo),
	}
	s.cache.Store(clave, entrada)
	return entrada, true, nil
}

func (s ParametrosService) codigoTipo(ctx context.Context, id int64) (string, error) {
	if id <= 0 {
		return "", nil
	}
	clave := fmt.Sprintf("tipo_parametro:%d", id)
	if valor, ok := s.tipos.Load(clave); ok {
		entrada, ok := valor.(tipoParametroCache)
		if ok && s.reloj().Before(entrada.expira) {
			return entrada.codigo, nil
		}
		s.tipos.Delete(clave)
	}
	base := strings.TrimRight(strings.TrimSpace(s.BaseURL), "/")
	if base == "" {
		return "", fmt.Errorf("no hay URL base configurada para %q", "tipo_parametro")
	}
	var raw json.RawMessage
	status, err := s.Client.Get(ctx, fmt.Sprintf("%s/v1/tipo_parametro/%d", base, id), &raw)
	if err != nil {
		if status == 404 {
			s.tipos.Store(clave, tipoParametroCache{expira: s.reloj().Add(s.ttlNegativo)})
			return "", nil
		}
		return "", err
	}
	registro, err := decodeAPIRecord(raw)
	if err != nil {
		return "", err
	}
	codigo, _ := registro["CodigoAbreviacion"].(string)
	codigo = strings.TrimSpace(codigo)
	s.tipos.Store(clave, tipoParametroCache{
		codigo: codigo,
		expira: s.reloj().Add(s.ttlPositivo),
	})
	return codigo, nil
}

func (s ParametrosService) reloj() time.Time {
	if s.ahora != nil {
		return s.ahora()
	}
	return time.Now()
}
