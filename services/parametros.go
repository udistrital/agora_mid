package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
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

// parametroCodigoCache guarda el Id resuelto por (código, tipo).
type parametroCodigoCache struct {
	id      int64
	existe  bool
	ambiguo bool
	expira  time.Time
}

// parametroCodigoRespuesta es la proyección mínima para resolver un parámetro
// por código y tipo.
type parametroCodigoRespuesta struct {
	Id                int    `json:"Id"`
	CodigoAbreviacion string `json:"CodigoAbreviacion"`
	Activo            bool   `json:"Activo"`
	TipoParametroId   struct {
		Id                int    `json:"Id"`
		CodigoAbreviacion string `json:"CodigoAbreviacion"`
	} `json:"TipoParametroId"`
}

// ParametrosService valida parámetros del CRUD de parámetros con caché por Id.
// Se usa para TipoCanalContactoId y ActividadEconomicaId (CIIU).
type ParametrosService struct {
	Client      clients.HTTPClient
	BaseURL     string
	cache       *sync.Map
	tipos       *sync.Map
	codigos     *sync.Map
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
		codigos:     &sync.Map{},
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

// ResolverParametroPorCodigo resuelve un parámetro activo por su código y el
// código de su tipo. Devuelve el Id de menor valor si hay más de una
// coincidencia activa y marca ambiguo. El par (código, tipo) es estable entre
// ambientes, a diferencia del Id.
func (s ParametrosService) ResolverParametroPorCodigo(
	ctx context.Context,
	codigo string,
	tipo string,
) (id int64, existe bool, ambiguo bool, err error) {
	codigo = strings.TrimSpace(codigo)
	tipo = strings.TrimSpace(tipo)
	if codigo == "" || tipo == "" {
		return 0, false, false, nil
	}
	if strings.ContainsAny(codigo, ",:") || strings.ContainsAny(tipo, ",:") {
		return 0, false, false,
			fmt.Errorf("el código de parámetro o su tipo no es válido")
	}

	clave := "parametro_codigo:" + tipo + "|" + codigo
	if cacheado, ok := s.leerCacheParametroCodigo(clave); ok {
		return cacheado.id, cacheado.existe, cacheado.ambiguo, nil
	}

	base := strings.TrimRight(strings.TrimSpace(s.BaseURL), "/")
	if base == "" {
		return 0, false, false,
			fmt.Errorf("no hay URL base configurada para %q", "parametro")
	}
	endpoint, err := construirURL(base, "/v1/parametro", url.Values{
		"query": {
			fmt.Sprintf(
				"CodigoAbreviacion:%s,TipoParametroId.CodigoAbreviacion:%s,Activo:true",
				codigo, tipo,
			),
		},
		"sortby": {"Id"},
		"order":  {"asc"},
		"limit":  {"10"},
	})
	if err != nil {
		return 0, false, false, err
	}

	registros, err := getAPIList[parametroCodigoRespuesta](ctx, s.Client, endpoint)
	if err != nil {
		return 0, false, false, err
	}

	var elegido *parametroCodigoRespuesta
	coincidencias := 0
	for i := range registros {
		registro := &registros[i]
		if registro.Id <= 0 || !registro.Activo {
			continue
		}
		if registro.CodigoAbreviacion != codigo {
			continue
		}
		if !strings.EqualFold(
			strings.TrimSpace(registro.TipoParametroId.CodigoAbreviacion), tipo,
		) {
			continue
		}
		coincidencias++
		if elegido == nil || registro.Id < elegido.Id {
			elegido = registro
		}
	}
	if elegido == nil {
		s.guardarCacheParametroCodigo(clave, parametroCodigoCache{
			existe: false,
			expira: s.reloj().Add(s.ttlNegativo),
		})
		return 0, false, false, nil
	}

	entrada := parametroCodigoCache{
		id:      int64(elegido.Id),
		existe:  true,
		ambiguo: coincidencias > 1,
		expira:  s.reloj().Add(s.ttlPositivo),
	}
	s.guardarCacheParametroCodigo(clave, entrada)
	return entrada.id, true, entrada.ambiguo, nil
}

func (s ParametrosService) leerCacheParametroCodigo(
	clave string,
) (parametroCodigoCache, bool) {
	if s.codigos == nil {
		return parametroCodigoCache{}, false
	}
	valor, ok := s.codigos.Load(clave)
	if !ok {
		return parametroCodigoCache{}, false
	}
	entrada, ok := valor.(parametroCodigoCache)
	if !ok {
		s.codigos.Delete(clave)
		return parametroCodigoCache{}, false
	}
	if s.reloj().Before(entrada.expira) {
		return entrada, true
	}
	s.codigos.Delete(clave)
	return parametroCodigoCache{}, false
}

func (s ParametrosService) guardarCacheParametroCodigo(
	clave string,
	entrada parametroCodigoCache,
) {
	if s.codigos == nil || entrada.expira.IsZero() {
		return
	}
	s.codigos.Store(clave, entrada)
}

func (s ParametrosService) reloj() time.Time {
	if s.ahora != nil {
		return s.ahora()
	}
	return time.Now()
}
