package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
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

// entradaCacheCodigo guarda un InfoComplementariaId resuelto por código.
type entradaCacheCodigo struct {
	id      int64
	existe  bool
	ambiguo bool
	expira  time.Time
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
	cacheCodigos       *sync.Map
	// InfoComplementariaOverrides permite fijar por entorno el Id de un código
	// duplicado o dudoso, por ejemplo MID_INFOCOMP_DESCRIPCION=53.
	InfoComplementariaOverrides map[string]int64
	ttlPositivo                 time.Duration
	ttlNegativo                 time.Duration
	ahora                       func() time.Time
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
		cacheCodigos:       &sync.Map{},
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

// PrecargarInfoComplementaria carga el catálogo completo en la caché para
// evitar una consulta HTTP por cada campo de caracterización. Si falla, el
// llamador puede continuar: las consultas puntuales seguirán funcionando.
func (s CatalogosService) PrecargarInfoComplementaria(ctx context.Context) error {
	base := strings.TrimRight(strings.TrimSpace(s.TercerosBaseURL), "/")
	if base == "" {
		return fmt.Errorf("no hay URL base configurada para %q", "info_complementaria")
	}
	limit := 1000
	var todos []catalogoRespuesta
	for offset := 0; ; {
		endpoint, err := url.Parse(base + "/v1/info_complementaria")
		if err != nil {
			return fmt.Errorf("URL de info_complementaria inválida: %w", err)
		}
		query := endpoint.Query()
		query.Set("limit", strconv.Itoa(limit))
		query.Set("offset", strconv.Itoa(offset))
		query.Set("sortby", "Id")
		query.Set("order", "asc")
		endpoint.RawQuery = query.Encode()
		registros, err := getAPIList[catalogoRespuesta](ctx, s.Client, endpoint.String())
		if err != nil {
			return err
		}
		todos = append(todos, registros...)
		if len(registros) < limit {
			break
		}
		offset += len(registros)
	}

	porCodigo := make(map[string][]catalogoRespuesta)
	for _, registro := range todos {
		if registro.Id <= 0 {
			continue
		}
		s.guardar("info_complementaria:"+strconv.Itoa(registro.Id),
			registro.Activo, s.ttlPositivo)
		codigoGrupo := strings.TrimSpace(registro.GrupoInfoComplementariaId.CodigoAbreviacion)
		if registro.Activo && codigoGrupo != "" {
			s.guardar(fmt.Sprintf("info_complementaria_grupo:%d:%s", registro.Id, codigoGrupo),
				true, s.ttlPositivo)
		}
		codigo := strings.TrimSpace(registro.CodigoAbreviacion)
		if codigo != "" {
			porCodigo[codigo] = append(porCodigo[codigo], registro)
		}
	}
	for codigo, registros := range porCodigo {
		activoID := int64(0)
		activos := 0
		inactivoID := int64(0)
		for _, registro := range registros {
			if registro.Activo {
				if activoID == 0 || int64(registro.Id) < activoID {
					activoID = int64(registro.Id)
				}
				activos++
				continue
			}
			if inactivoID == 0 || int64(registro.Id) < inactivoID {
				inactivoID = int64(registro.Id)
			}
		}
		clave := "info_complementaria_codigo:" + codigo
		if activos > 0 {
			s.guardarCodigo(clave, activoID, true, activos > 1, s.ttlPositivo)
		} else if inactivoID > 0 {
			s.guardarCodigo(clave, inactivoID, true, false, s.ttlPositivo)
		}
	}
	return nil
}

// InfoComplementariaExiste comprueba que el registro de info_complementaria
// exista y esté activo en terceros_crud.
func (s CatalogosService) InfoComplementariaExiste(ctx context.Context, id int64) (bool, error) {
	return s.existe(ctx, "info_complementaria", s.TercerosBaseURL, id)
}

// InfoComplementariaPorCodigo resuelve un código de info_complementaria a su
// Id. Devuelve existe=false cuando el código no está registrado. Si hay varios
// activos, usa el primero por Id y devuelve ambiguo=true para que el llamador
// emita un aviso. Si solo hay inactivos, devuelve el Id y el llamador debe
// comprobarlo con InfoComplementariaExiste.
func (s CatalogosService) InfoComplementariaPorCodigo(
	ctx context.Context,
	codigo string,
) (id int64, existe bool, ambiguo bool, err error) {
	codigo = strings.TrimSpace(codigo)
	if codigo == "" {
		return 0, false, false, nil
	}
	if override, ok := s.InfoComplementariaOverrides[codigo]; ok && override > 0 {
		return override, true, false, nil
	}
	clave := "info_complementaria_codigo:" + codigo
	if id, ambiguo, ok := s.leerCacheCodigo(clave); ok {
		return id, id > 0, ambiguo, nil
	}

	base := strings.TrimRight(strings.TrimSpace(s.TercerosBaseURL), "/")
	if base == "" {
		return 0, false, false, fmt.Errorf("no hay URL base configurada para %q", "info_complementaria")
	}
	endpoint, err := url.Parse(base + "/v1/info_complementaria")
	if err != nil {
		return 0, false, false, fmt.Errorf("URL de info_complementaria inválida: %w", err)
	}
	query := endpoint.Query()
	query.Set("query", "CodigoAbreviacion:"+codigo)
	query.Set("sortby", "Id")
	query.Set("order", "asc")
	query.Set("limit", "10")
	endpoint.RawQuery = query.Encode()

	registros, err := getAPIList[catalogoRespuesta](ctx, s.Client, endpoint.String())
	if err != nil {
		return 0, false, false, err
	}

	activoID := int64(0)
	activos := 0
	inactivoID := int64(0)
	for _, registro := range registros {
		if registro.Id <= 0 || registro.CodigoAbreviacion != codigo {
			continue
		}
		if registro.Activo {
			if activoID == 0 || int64(registro.Id) < activoID {
				activoID = int64(registro.Id)
			}
			activos++
			continue
		}
		if inactivoID == 0 || int64(registro.Id) < inactivoID {
			inactivoID = int64(registro.Id)
		}
	}
	if activos > 0 {
		ambiguo = activos > 1
		s.guardarCodigo(clave, activoID, true, ambiguo, s.ttlPositivo)
		return activoID, true, ambiguo, nil
	}
	if inactivoID > 0 {
		// El llamador valida Activo con InfoComplementariaExiste para poder
		// distinguir un código inactivo de uno inexistente.
		s.guardarCodigo(clave, inactivoID, true, false, s.ttlPositivo)
		return inactivoID, true, false, nil
	}
	s.guardarCodigo(clave, 0, false, false, s.ttlNegativo)
	return 0, false, false, nil
}

// InfoComplementariaDeGrupo valida que un registro de info_complementaria
// exista, esté activo y pertenezca al grupo indicado. Se usa para validar
// PerfilId contra Grupo_5.
func (s CatalogosService) InfoComplementariaDeGrupo(
	ctx context.Context,
	id int64,
	codigoGrupo string,
) (bool, error) {
	if id <= 0 {
		return false, nil
	}
	codigoGrupo = strings.TrimSpace(codigoGrupo)
	if codigoGrupo == "" {
		return s.InfoComplementariaExiste(ctx, id)
	}
	clave := fmt.Sprintf("info_complementaria_grupo:%d:%s", id, codigoGrupo)
	if existe, ok := s.leerCache(clave); ok {
		return existe, nil
	}

	base := strings.TrimRight(strings.TrimSpace(s.TercerosBaseURL), "/")
	if base == "" {
		return false, fmt.Errorf("no hay URL base configurada para %q", "info_complementaria")
	}
	var raw json.RawMessage
	status, err := s.Client.Get(ctx, fmt.Sprintf("%s/v1/info_complementaria/%d", base, id), &raw)
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
	grupo := idDeRelacion(registro["GrupoInfoComplementariaId"])
	if grupo <= 0 {
		s.guardar(clave, false, s.ttlPositivo)
		return false, nil
	}
	// El registro del grupo se consulta por Id para comparar el código.
	var rawGrupo json.RawMessage
	status, err = s.Client.Get(ctx, fmt.Sprintf("%s/v1/grupo_info_complementaria/%d", base, grupo), &rawGrupo)
	if err != nil {
		if status == 404 {
			s.guardar(clave, false, s.ttlNegativo)
			return false, nil
		}
		return false, err
	}
	registroGrupo, err := decodeAPIRecord(rawGrupo)
	if err != nil {
		return false, err
	}
	codigo, _ := registroGrupo["CodigoAbreviacion"].(string)
	pertenece := activo && strings.TrimSpace(codigo) == codigoGrupo
	s.guardar(clave, pertenece, s.ttlPositivo)
	return pertenece, nil
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

// leerCacheCodigo devuelve el Id vigente de un código y si era ambiguo, y
// descarta la entrada si venció.
func (s CatalogosService) leerCacheCodigo(clave string) (int64, bool, bool) {
	if s.cacheCodigos == nil {
		return 0, false, false
	}
	valor, ok := s.cacheCodigos.Load(clave)
	if !ok {
		return 0, false, false
	}
	entrada, ok := valor.(entradaCacheCodigo)
	if !ok {
		s.cacheCodigos.Delete(clave)
		return 0, false, false
	}
	if s.reloj().Before(entrada.expira) {
		return entrada.id, entrada.ambiguo, true
	}
	s.cacheCodigos.Delete(clave)
	return 0, false, false
}

func (s CatalogosService) guardarCodigo(
	clave string,
	id int64,
	existe bool,
	ambiguo bool,
	ttl time.Duration,
) {
	if s.cacheCodigos == nil || ttl <= 0 {
		return
	}
	s.cacheCodigos.Store(clave, entradaCacheCodigo{
		id:      id,
		existe:  existe,
		ambiguo: ambiguo,
		expira:  s.reloj().Add(ttl),
	})
}

func (s CatalogosService) reloj() time.Time {
	if s.ahora != nil {
		return s.ahora()
	}
	return time.Now()
}
