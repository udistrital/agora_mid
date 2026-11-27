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

// InfoComplementariaResuelta es la proyección mínima del catálogo que necesita
// el dominio 2: Id local, código, grupo, estado y si había varias filas activas.
type InfoComplementariaResuelta struct {
	Id          int64
	Codigo      string
	CodigoGrupo string
	Activo      bool
	Ambiguo     bool
}

type entradaCacheDetalle struct {
	detalle InfoComplementariaResuelta
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
	cacheDetalle       *sync.Map
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
		cacheDetalle:       &sync.Map{},
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

// TipoDocumentoPorCodigo resuelve un tipo de documento activo por su código.
// Devuelve el Id de menor valor si hay más de una coincidencia y marca ambiguo.
func (s CatalogosService) TipoDocumentoPorCodigo(
	ctx context.Context,
	codigo string,
) (id int64, existe bool, ambiguo bool, err error) {
	codigo = strings.ToUpper(strings.TrimSpace(codigo))
	if codigo == "" {
		return 0, false, false, nil
	}
	clave := "tipo_documento_codigo:" + codigo
	if id, ambiguo, ok := s.leerCacheCodigo(clave); ok {
		return id, id > 0, ambiguo, nil
	}

	base := strings.TrimRight(strings.TrimSpace(s.TercerosBaseURL), "/")
	if base == "" {
		return 0, false, false,
			fmt.Errorf("no hay URL base configurada para %q", "tipo_documento")
	}
	endpoint, err := construirURL(base, "/v1/tipo_documento", url.Values{
		"query":  {fmt.Sprintf("CodigoAbreviacion:%s,Activo:true", codigo)},
		"sortby": {"Id"},
		"order":  {"asc"},
		"limit":  {"10"},
	})
	if err != nil {
		return 0, false, false, err
	}
	registros, err := getAPIList[catalogoRespuesta](ctx, s.Client, endpoint)
	if err != nil {
		return 0, false, false, err
	}

	var elegido *catalogoRespuesta
	coincidencias := 0
	for i := range registros {
		registro := &registros[i]
		if registro.Id <= 0 || !registro.Activo ||
			!strings.EqualFold(strings.TrimSpace(registro.CodigoAbreviacion), codigo) {
			continue
		}
		coincidencias++
		if elegido == nil || registro.Id < elegido.Id {
			elegido = registro
		}
	}
	if elegido == nil {
		s.guardarCodigo(clave, 0, false, false, s.ttlNegativo)
		return 0, false, false, nil
	}
	s.guardarCodigo(clave, int64(elegido.Id), true, coincidencias > 1, s.ttlPositivo)
	return int64(elegido.Id), true, coincidencias > 1, nil
}

// TipoDocumentoPorID devuelve el código de un tipo de documento activo.
func (s CatalogosService) TipoDocumentoPorID(
	ctx context.Context,
	id int64,
) (codigo string, existe bool, err error) {
	if id <= 0 {
		return "", false, nil
	}
	clave := fmt.Sprintf("tipo_documento_id:%d", id)
	if guardado, ok := s.leerCacheDetalle(clave); ok {
		return guardado.Codigo, guardado.Id > 0, nil
	}
	base := strings.TrimRight(strings.TrimSpace(s.TercerosBaseURL), "/")
	if base == "" {
		return "", false,
			fmt.Errorf("no hay URL base configurada para %q", "tipo_documento")
	}
	var raw json.RawMessage
	status, err := s.Client.Get(ctx,
		fmt.Sprintf("%s/v1/tipo_documento/%d", base, id), &raw)
	if err != nil {
		if status == 404 {
			s.guardarDetalle(clave, InfoComplementariaResuelta{}, s.ttlNegativo)
			return "", false, nil
		}
		return "", false, err
	}
	registro, err := decodeAPIRecord(raw)
	if err != nil {
		return "", false, err
	}
	activo, _ := registro["Activo"].(bool)
	if !activo {
		return "", false, nil
	}
	codigo, _ = registro["CodigoAbreviacion"].(string)
	codigo = strings.TrimSpace(codigo)
	s.guardarDetalle(clave, InfoComplementariaResuelta{
		Id:     id,
		Codigo: codigo,
		Activo: true,
	}, s.ttlPositivo)
	return codigo, true, nil
}

// TerceroPorDocumento busca el tercero dueño de una identificación activa por
// número y tipo de documento. Si hay más de una identificación activa devuelve
// ErrCatalogoAmbiguo para que el llamador no elija arbitrariamente.
func (s CatalogosService) TerceroPorDocumento(
	ctx context.Context,
	tipoDocumentoID int64,
	numero string,
) (map[string]any, bool, error) {
	numero = strings.TrimSpace(numero)
	if tipoDocumentoID <= 0 || numero == "" {
		return nil, false, nil
	}
	base := strings.TrimRight(strings.TrimSpace(s.TercerosBaseURL), "/")
	if base == "" {
		return nil, false,
			fmt.Errorf("no hay URL base configurada para %q", "datos_identificacion")
	}
	endpoint, err := construirURL(base, "/v1/datos_identificacion", url.Values{
		"query": {
			fmt.Sprintf(
				"Numero:%s,TipoDocumentoId:%d,Activo:true",
				numero, tipoDocumentoID,
			),
		},
		"sortby": {"Id"},
		"order":  {"asc"},
		"limit":  {"10"},
	})
	if err != nil {
		return nil, false, err
	}
	var raw json.RawMessage
	status, err := s.Client.Get(ctx, endpoint, &raw)
	if err != nil {
		if status == 404 {
			return nil, false, nil
		}
		return nil, false, err
	}
	registros, err := decodeAPIList[map[string]any](raw)
	if err != nil {
		return nil, false, err
	}

	var elegido map[string]any
	coincidencias := 0
	for _, registro := range registros {
		activo, _ := registro["Activo"].(bool)
		if !activo {
			continue
		}
		numeroRegistro, _ := registro["Numero"].(string)
		if strings.TrimSpace(numeroRegistro) != numero {
			continue
		}
		if idDeRelacion(registro["TipoDocumentoId"]) != tipoDocumentoID {
			continue
		}
		coincidencias++
		if elegido == nil {
			elegido = registro
		}
	}
	if coincidencias == 0 {
		return nil, false, nil
	}
	if coincidencias > 1 {
		return nil, false, ErrCatalogoAmbiguo
	}
	terceroID := idDeRelacion(elegido["TerceroId"])
	if terceroID <= 0 {
		return nil, false,
			fmt.Errorf("la identificación no contiene un TerceroId válido")
	}

	var rawTercero json.RawMessage
	status, err = s.Client.Get(ctx,
		fmt.Sprintf("%s/v1/tercero/%d", base, terceroID), &rawTercero)
	if err != nil {
		if status == 404 {
			return nil, false, nil
		}
		return nil, false, err
	}
	tercero, err := decodeAPIRecord(rawTercero)
	if err != nil {
		return nil, false, err
	}
	tercero = desenvolverData(tercero)
	if idDeMapa(tercero) != terceroID {
		return nil, false,
			fmt.Errorf("la respuesta del tercero no contiene el Id esperado")
	}
	return tercero, true, nil
}

// IdentificacionActiva devuelve la identificación activa de un tercero. Se usa
// para enriquecer la respuesta del representante cuando llega solo su Id.
func (s CatalogosService) IdentificacionActiva(
	ctx context.Context,
	terceroID int64,
) (map[string]any, bool, error) {
	if terceroID <= 0 {
		return nil, false, nil
	}
	base := strings.TrimRight(strings.TrimSpace(s.TercerosBaseURL), "/")
	if base == "" {
		return nil, false,
			fmt.Errorf("no hay URL base configurada para %q", "datos_identificacion")
	}
	endpoint, err := construirURL(base, "/v1/datos_identificacion", url.Values{
		"query":  {fmt.Sprintf("TerceroId:%d,Activo:true", terceroID)},
		"sortby": {"Id"},
		"order":  {"asc"},
		"limit":  {"10"},
	})
	if err != nil {
		return nil, false, err
	}
	var raw json.RawMessage
	status, err := s.Client.Get(ctx, endpoint, &raw)
	if err != nil {
		if status == 404 {
			return nil, false, nil
		}
		return nil, false, err
	}
	registros, err := decodeAPIList[map[string]any](raw)
	if err != nil {
		return nil, false, err
	}
	for _, registro := range registros {
		activo, _ := registro["Activo"].(bool)
		if !activo || idDeMapa(registro) <= 0 {
			continue
		}
		return registro, true, nil
	}
	return nil, false, nil
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
	porCodigoGrupo := make(map[[2]string][]catalogoRespuesta)
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
		s.guardarDetalle(
			fmt.Sprintf("info_complementaria_detalle_id:%d", registro.Id),
			InfoComplementariaResuelta{
				Id:          int64(registro.Id),
				Codigo:      codigo,
				CodigoGrupo: codigoGrupo,
				Activo:      registro.Activo,
			},
			s.ttlPositivo,
		)
		if codigo != "" {
			porCodigo[codigo] = append(porCodigo[codigo], registro)
			porCodigoGrupo[[2]string{codigo, codigoGrupo}] = append(
				porCodigoGrupo[[2]string{codigo, codigoGrupo}], registro,
			)
		}
	}
	for clave, registros := range porCodigoGrupo {
		detalle, ok := seleccionarDetalleInfoComplementaria(registros, clave[0], clave[1])
		if ok {
			s.guardarDetalle(detalleCacheKey(clave[0], clave[1]), detalle, s.ttlPositivo)
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
		detalle, ok := seleccionarDetalleInfoComplementaria(registros, codigo, "")
		if ok {
			s.guardarDetalle(detalleCacheKey(codigo, ""), detalle, s.ttlPositivo)
		}
	}
	return nil
}

// InfoComplementariaExiste comprueba que el registro de info_complementaria
// exista y esté activo en terceros_crud.
func (s CatalogosService) InfoComplementariaExiste(ctx context.Context, id int64) (bool, error) {
	return s.existe(ctx, "info_complementaria", s.TercerosBaseURL, id)
}

// InfoComplementariaPorCodigo se conserva para compatibilidad. El dominio 2
// usa ResolverInfoComplementaria con el código del grupo para evitar que dos
// grupos compartan el mismo código (por ejemplo NO APLICA).
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
	detalle, ok, err := s.ResolverInfoComplementaria(ctx, codigo, "")
	if err != nil {
		return 0, false, false, err
	}
	if !ok {
		s.guardarCodigo(clave, 0, false, false, s.ttlNegativo)
		return 0, false, false, nil
	}
	s.guardarCodigo(clave, detalle.Id, true, detalle.Ambiguo, s.ttlPositivo)
	return detalle.Id, true, detalle.Ambiguo, nil
}

// ResolverInfoComplementaria resuelve (código, grupo) contra el catálogo. Los
// Ids cambian entre ambientes; el par de códigos no. Si hay varias filas
// activas, usa la de menor Id y marca Ambiguo.
func (s CatalogosService) ResolverInfoComplementaria(
	ctx context.Context,
	codigo string,
	codigoGrupo string,
) (InfoComplementariaResuelta, bool, error) {
	codigo = strings.TrimSpace(codigo)
	codigoGrupo = strings.TrimSpace(codigoGrupo)
	if codigo == "" {
		return InfoComplementariaResuelta{}, false, nil
	}
	clave := detalleCacheKey(codigo, codigoGrupo)
	if detalle, ok := s.leerCacheDetalle(clave); ok {
		return detalle, detalle.Id > 0, nil
	}

	base := strings.TrimRight(strings.TrimSpace(s.TercerosBaseURL), "/")
	if base == "" {
		return InfoComplementariaResuelta{}, false,
			fmt.Errorf("no hay URL base configurada para %q", "info_complementaria")
	}
	endpoint, err := url.Parse(base + "/v1/info_complementaria")
	if err != nil {
		return InfoComplementariaResuelta{}, false,
			fmt.Errorf("URL de info_complementaria inválida: %w", err)
	}
	query := endpoint.Query()
	filtro := "CodigoAbreviacion:" + codigo
	if codigoGrupo != "" {
		filtro += ",GrupoInfoComplementariaId.CodigoAbreviacion:" + codigoGrupo
	}
	query.Set("query", filtro)
	query.Set("sortby", "Id")
	query.Set("order", "asc")
	query.Set("limit", "20")
	endpoint.RawQuery = query.Encode()

	registros, err := getAPIList[catalogoRespuesta](ctx, s.Client, endpoint.String())
	if err != nil {
		return InfoComplementariaResuelta{}, false, err
	}
	detalle, ok := seleccionarDetalleInfoComplementaria(registros, codigo, codigoGrupo)
	if !ok {
		s.guardarDetalle(clave, InfoComplementariaResuelta{}, s.ttlNegativo)
		return InfoComplementariaResuelta{}, false, nil
	}
	s.guardarDetalle(clave, detalle, s.ttlPositivo)
	return detalle, true, nil
}

// InfoComplementariaPorID resuelve una fila por Id. Se conserva para el
// contrato anterior y para leer el grupo de filas ya existentes.
func (s CatalogosService) InfoComplementariaPorID(
	ctx context.Context,
	id int64,
) (InfoComplementariaResuelta, bool, error) {
	if id <= 0 {
		return InfoComplementariaResuelta{}, false, nil
	}
	clave := fmt.Sprintf("info_complementaria_detalle_id:%d", id)
	if detalle, ok := s.leerCacheDetalle(clave); ok {
		return detalle, detalle.Id > 0, nil
	}
	base := strings.TrimRight(strings.TrimSpace(s.TercerosBaseURL), "/")
	if base == "" {
		return InfoComplementariaResuelta{}, false,
			fmt.Errorf("no hay URL base configurada para %q", "info_complementaria")
	}
	var raw json.RawMessage
	status, err := s.Client.Get(ctx, fmt.Sprintf("%s/v1/info_complementaria/%d", base, id), &raw)
	if err != nil {
		if status == 404 {
			s.guardarDetalle(clave, InfoComplementariaResuelta{}, s.ttlNegativo)
			return InfoComplementariaResuelta{}, false, nil
		}
		return InfoComplementariaResuelta{}, false, err
	}
	registro, err := decodeAPIRecord(raw)
	if err != nil {
		return InfoComplementariaResuelta{}, false, err
	}
	codigo, _ := registro["CodigoAbreviacion"].(string)
	activo, _ := registro["Activo"].(bool)
	codigoGrupo := ""
	if grupo, ok := registro["GrupoInfoComplementariaId"].(map[string]any); ok {
		codigoGrupo, _ = grupo["CodigoAbreviacion"].(string)
		codigoGrupo = strings.TrimSpace(codigoGrupo)
		if codigoGrupo == "" {
			grupoID := idDeRelacion(grupo["Id"])
			if grupoID > 0 {
				codigoGrupo, err = s.codigoGrupoPorID(ctx, base, grupoID)
				if err != nil {
					return InfoComplementariaResuelta{}, false, err
				}
			}
		}
	}
	detalle := InfoComplementariaResuelta{
		Id:          id,
		Codigo:      strings.TrimSpace(codigo),
		CodigoGrupo: codigoGrupo,
		Activo:      activo,
	}
	s.guardarDetalle(clave, detalle, s.ttlPositivo)
	return detalle, true, nil
}

func (s CatalogosService) codigoGrupoPorID(
	ctx context.Context,
	base string,
	id int64,
) (string, error) {
	var raw json.RawMessage
	status, err := s.Client.Get(ctx,
		fmt.Sprintf("%s/v1/grupo_info_complementaria/%d", base, id), &raw)
	if err != nil {
		if status == 404 {
			return "", nil
		}
		return "", err
	}
	registro, err := decodeAPIRecord(raw)
	if err != nil {
		return "", err
	}
	codigo, _ := registro["CodigoAbreviacion"].(string)
	return strings.TrimSpace(codigo), nil
}

func seleccionarDetalleInfoComplementaria(
	registros []catalogoRespuesta,
	codigo string,
	codigoGrupo string,
) (InfoComplementariaResuelta, bool) {
	var activo *catalogoRespuesta
	var inactivo *catalogoRespuesta
	activos := 0
	for i := range registros {
		registro := &registros[i]
		if registro.Id <= 0 || registro.CodigoAbreviacion != codigo {
			continue
		}
		if codigoGrupo != "" &&
			!strings.EqualFold(strings.TrimSpace(registro.GrupoInfoComplementariaId.CodigoAbreviacion), codigoGrupo) {
			continue
		}
		if registro.Activo {
			if activo == nil || int64(registro.Id) < int64(activo.Id) {
				activo = registro
			}
			activos++
			continue
		}
		if inactivo == nil || int64(registro.Id) < int64(inactivo.Id) {
			inactivo = registro
		}
	}
	elegido := activo
	if elegido == nil {
		elegido = inactivo
	}
	if elegido == nil {
		return InfoComplementariaResuelta{}, false
	}
	return InfoComplementariaResuelta{
		Id:          int64(elegido.Id),
		Codigo:      elegido.CodigoAbreviacion,
		CodigoGrupo: strings.TrimSpace(elegido.GrupoInfoComplementariaId.CodigoAbreviacion),
		Activo:      elegido.Activo,
		Ambiguo:     activos > 1,
	}, true
}

func detalleCacheKey(codigo, codigoGrupo string) string {
	return "info_complementaria_detalle:" + codigo + "|" + codigoGrupo
}

func (s CatalogosService) leerCacheDetalle(clave string) (InfoComplementariaResuelta, bool) {
	if s.cacheDetalle == nil {
		return InfoComplementariaResuelta{}, false
	}
	valor, ok := s.cacheDetalle.Load(clave)
	if !ok {
		return InfoComplementariaResuelta{}, false
	}
	entrada, ok := valor.(entradaCacheDetalle)
	if !ok {
		s.cacheDetalle.Delete(clave)
		return InfoComplementariaResuelta{}, false
	}
	if s.reloj().Before(entrada.expira) {
		return entrada.detalle, true
	}
	s.cacheDetalle.Delete(clave)
	return InfoComplementariaResuelta{}, false
}

func (s CatalogosService) guardarDetalle(
	clave string,
	detalle InfoComplementariaResuelta,
	ttl time.Duration,
) {
	if s.cacheDetalle == nil || ttl <= 0 {
		return
	}
	s.cacheDetalle.Store(clave, entradaCacheDetalle{
		detalle: detalle,
		expira:  s.reloj().Add(ttl),
	})
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
