package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/udistrital/agora_mid/clients"
	"github.com/udistrital/agora_mid/config"
	"github.com/udistrital/agora_mid/helper"
)

// ErrProveedorIDInvalido indica que el identificador solicitado es ausente o no
// positivo. El controlador debe mapearlo a 400 Bad Request.
var ErrProveedorIDInvalido = errors.New("el id de proveedor debe ser un entero positivo")

// ErrProveedorNoEncontrado indica que no existe un proveedor con el id
// solicitado. El controlador debe mapearlo a un estado de no encontrado.
var ErrProveedorNoEncontrado = errors.New("no existe un proveedor con el id solicitado")

// ErrProveedorConsultaInvalida indica que un filtro o parámetro de paginación
// no cumple el contrato del listado de proveedores.
var ErrProveedorConsultaInvalida = errors.New("consulta de proveedores inválida")

const (
	limiteListadoProveedorDefecto = 20
	limiteListadoProveedorMaximo  = 100
)

// Nombres de relación (mapped_name) de las ramas de detalle del proveedor. El
// servicio selecciona una u otra según el tipo de registro y poda la otra de la
// configuración antes de expandir.
const (
	relacionProveedorNatural  = "ProveedorNatural"
	relacionProveedorJuridico = "ProveedorJuridico"
)

// ProveedorService orquesta la lectura del proveedor como recurso raíz: trae el
// proveedor base por id desde gestión de personas, construye las opciones de
// expansión a partir de los fields, decide la rama de detalle (natural o
// jurídico) según el tipo de registro e invoca el expansor con el mapeo
// relación→resolver del núcleo. Es la única capa que contiene lógica de negocio;
// los resolvers no deciden ramas ni transforman datos.
type ProveedorService struct {
	// BaseURL es la URL base del CRUD de gestión de personas, sin el sufijo /v1.
	BaseURL string
	// Client ejecuta las peticiones HTTP al CRUD dueño del proveedor.
	Client clients.HTTPClient
	// Config es la configuración declarativa de relaciones del proveedor.
	Config config.ServiceConfig
	// Registry es el registro relación→resolver que el servicio inyecta al
	// expansor para resolver cada entidad del núcleo.
	Registry helper.ResolverRegistry
	// TipoRegistroNatural es el identificador de tipo de contribuyente que
	// corresponde a persona natural. Un proveedor con este tipo de registro
	// expande la rama ProveedorNatural; cualquier otro expande ProveedorJuridico.
	TipoRegistroNatural int
	// Blacklist es la lista negra global de campos que el servicio inyecta al
	// expansor para recortar los campos vetados de la salida de todo resolver y
	// para rechazar un campo vetado pedido explícitamente en fields. Puede ser nil
	// o vacía (sin supresión).
	Blacklist config.GlobalBlacklist
}

// ProveedorRecord es la vista mínima del proveedor base que el servicio de
// lectura resuelve desde el CRUD de gestión de personas antes de expandir su
// subárbol. Vive aquí porque el expansor del proveedor es su único consumidor.
type ProveedorRecord struct {
	Id             int  `json:"Id"`
	TerceroId      int  `json:"TerceroId"`
	TipoRegistroId int  `json:"TipoRegistroId"`
	Activo         bool `json:"Activo"`
}

// LeerProveedor trae el proveedor base por id y devuelve el recurso expandido
// según fields junto con los avisos de expansión acumulados.
//
// Devuelve ErrProveedorIDInvalido si id no es positivo (sin consultar el CRUD),
// ErrProveedorNoEncontrado si no existe el proveedor, *helper.ErrorFields si
// fields es inválido, o un error fatal de expansión (ver
// helper.EsErrorFatalExpansion) cuando la política de error de relación es fail.
func (s ProveedorService) LeerProveedor(
	ctx context.Context,
	id int,
	fields []string,
) (helper.ExpandResult, error) {
	if id <= 0 {
		return helper.ExpandResult{}, ErrProveedorIDInvalido
	}

	// Fallar temprano: validar y parsear fields contra la configuración completa
	// ANTES de consultar el CRUD dueño. Ambas ramas de detalle están declaradas,
	// de modo que pedir la rama que no aplica es una ruta válida (no un 400) que
	// simplemente no se materializa. Un fields inválido devuelve *helper.ErrorFields
	// sin ninguna llamada al CRUD. Este árbol es la única fuente de verdad de la
	// validación: se reutiliza en la expansión sin volver a parsear.
	var seleccion *helper.NodoSeleccion
	if len(fields) > 0 {
		arbol, err := helper.ParsearFields(fields, s.Config.Relations, nil, s.Blacklist)
		if err != nil {
			return helper.ExpandResult{}, err
		}
		seleccion = arbol
	}

	base, err := s.proveedorBase(ctx, id)
	if err != nil {
		return helper.ExpandResult{}, err
	}

	// Podar la rama de detalle que no corresponde al tipo de registro; el nodo de
	// selección de la rama descartada queda inerte porque el expansor recorre las
	// relaciones (ya podadas), no el árbol de selección.
	cfg := s.configParaTipo(base.TipoRegistroId)

	// El recurso raíz se representa como mapa genérico para que el expansor
	// pueda anidar las relaciones bajo sus mapped_name conservando las llaves
	// locales (Id, TerceroId, TipoRegistroId).
	data := map[string]any{
		"Id":             base.Id,
		"TerceroId":      base.TerceroId,
		"TipoRegistroId": base.TipoRegistroId,
		"Activo":         base.Activo,
	}

	return helper.Expand(ctx, data, cfg, s.Registry, helper.ExpandOptions{Seleccion: seleccion, Blacklist: s.Blacklist})
}

// ListarProveedores valida filtros de primer nivel, consulta la página al CRUD
// dueño y expande cada proveedor conservando el total del conjunto filtrado.
// limit y offset iguales a cero aplican los valores por defecto.
func (s ProveedorService) ListarProveedores(
	ctx context.Context,
	filtros []string,
	fields []string,
	limit, offset int,
) (helper.CollectionResult, error) {
	query, err := validarFiltrosProveedor(filtros, s.Config.FilterableFields)
	if err != nil {
		return helper.CollectionResult{}, err
	}
	// Una colección sin fields no expande relaciones: usar una selección vacía
	// (no nil) evita la expansión completa por elemento, costosa para una lista.
	seleccion := &helper.NodoSeleccion{}
	if len(fields) > 0 {
		seleccion, err = helper.ParsearFields(fields, s.Config.Relations, nil, s.Blacklist)
		if err != nil {
			return helper.CollectionResult{}, err
		}
	}
	if limit < 0 || offset < 0 {
		return helper.CollectionResult{}, fmt.Errorf("%w: limit y offset deben ser no negativos", ErrProveedorConsultaInvalida)
	}
	if limit == 0 {
		limit = limiteListadoProveedorDefecto
	}
	if limit > limiteListadoProveedorMaximo {
		limit = limiteListadoProveedorMaximo
	}

	endpoint, err := url.Parse(strings.TrimRight(strings.TrimSpace(s.BaseURL), "/") + "/v1/proveedor/")
	if err != nil {
		return helper.CollectionResult{}, fmt.Errorf("URL de gestión de personas inválida: %w", err)
	}
	params := endpoint.Query()
	params.Set("query", query)
	params.Set("limit", strconv.Itoa(limit))
	params.Set("offset", strconv.Itoa(offset))
	endpoint.RawQuery = params.Encode()

	var raw json.RawMessage
	if _, err := s.Client.Get(ctx, endpoint.String(), &raw); err != nil {
		return helper.CollectionResult{}, fmt.Errorf("consultando listado de proveedores: %w", err)
	}
	lista, err := decodeAPIListWithTotal[ProveedorRecord](raw)
	if err != nil {
		return helper.CollectionResult{}, fmt.Errorf("consultando listado de proveedores: %w", err)
	}
	records := make([]map[string]any, 0, len(lista.Data))
	for _, registro := range lista.Data {
		records = append(records, map[string]any{
			"Id": registro.Id, "TerceroId": registro.TerceroId,
			"TipoRegistroId": registro.TipoRegistroId, "Activo": registro.Activo,
		})
	}
	return helper.ListCollection(ctx, records, lista.Total, s.configParaTipo(s.TipoRegistroNatural), s.Registry,
		helper.ExpandOptions{Seleccion: seleccion, Blacklist: s.Blacklist})
}

// validarFiltrosProveedor valida la gramática de filtros aceptada por beego y
// los serializa como una conjunción para el CRUD dueño.
func validarFiltrosProveedor(filtros, campos []string) (string, error) {
	permitidos := make(map[string]struct{}, len(campos))
	for _, campo := range campos {
		permitidos[campo] = struct{}{}
	}
	const operadores = "exact|iexact|contains|icontains|gt|gte|lt|lte|startswith|istartswith|endswith|iendswith|in|isnull"
	partes := make([]string, 0, len(filtros))
	for _, filtro := range filtros {
		filtro = strings.TrimSpace(filtro)
		segmentos := strings.SplitN(filtro, ":", 2)
		if len(segmentos) != 2 {
			return "", fmt.Errorf("%w: condición inválida %q", ErrProveedorConsultaInvalida, filtro)
		}
		campo, valor := segmentos[0], segmentos[1]
		nombre, operador := campo, "exact"
		if idx := strings.Index(campo, "__"); idx >= 0 {
			nombre, operador = campo[:idx], campo[idx+2:]
		}
		if _, ok := permitidos[nombre]; !ok || strings.ContainsAny(nombre, ".") || nombre == "" {
			return "", fmt.Errorf("%w: campo no filtrable %q", ErrProveedorConsultaInvalida, nombre)
		}
		if !strings.Contains("|"+operadores+"|", "|"+operador+"|") {
			return "", fmt.Errorf("%w: operador no soportado %q", ErrProveedorConsultaInvalida, operador)
		}
		if operador == "in" {
			if valor == "" || !strings.Contains(valor, "|") {
				return "", fmt.Errorf("%w: __in requiere una lista separada por |", ErrProveedorConsultaInvalida)
			}
		} else if strings.Contains(valor, "|") {
			return "", fmt.Errorf("%w: el operador %s requiere un valor único", ErrProveedorConsultaInvalida, operador)
		}
		if operador == "isnull" && valor != "true" && valor != "false" {
			return "", fmt.Errorf("%w: isnull requiere true o false", ErrProveedorConsultaInvalida)
		}
		partes = append(partes, campo+":"+valor)
	}
	return strings.Join(partes, ","), nil
}

// proveedorBase consulta el proveedor por su id contra el CRUD dueño y devuelve
// el registro encontrado. Devuelve ErrProveedorNoEncontrado si la consulta no
// arroja un proveedor con el id solicitado.
func (s ProveedorService) proveedorBase(ctx context.Context, id int) (ProveedorRecord, error) {
	base := strings.TrimRight(strings.TrimSpace(s.BaseURL), "/")
	if base == "" {
		return ProveedorRecord{}, errors.New("falta la URL base de gestión de personas CRUD")
	}

	endpoint, err := url.Parse(base + "/v1/proveedor/")
	if err != nil {
		return ProveedorRecord{}, fmt.Errorf("URL de gestión de personas inválida: %w", err)
	}

	query := endpoint.Query()
	query.Set("query", "Id:"+strconv.Itoa(id))
	query.Set("limit", "1")
	query.Set("offset", "0")
	endpoint.RawQuery = query.Encode()

	var raw json.RawMessage
	if _, err := s.Client.Get(ctx, endpoint.String(), &raw); err != nil {
		return ProveedorRecord{}, fmt.Errorf("consultando proveedor por id: %w", err)
	}

	registros, err := decodeAPIList[ProveedorRecord](raw)
	if err != nil {
		return ProveedorRecord{}, fmt.Errorf("consultando proveedor por id: %w", err)
	}

	for _, registro := range registros {
		// El CRUD puede devolver [{}] cuando no hay coincidencias.
		if registro.Id == id {
			return registro, nil
		}
	}

	return ProveedorRecord{}, ErrProveedorNoEncontrado
}

// configParaTipo devuelve la configuración de relaciones con la rama de detalle
// que no corresponde al tipo de registro podada. Un proveedor natural conserva
// ProveedorNatural y descarta ProveedorJuridico, y viceversa.
func (s ProveedorService) configParaTipo(tipoRegistro int) config.ServiceConfig {
	descartar := relacionProveedorJuridico
	if tipoRegistro != s.TipoRegistroNatural {
		descartar = relacionProveedorNatural
	}

	cfg := s.Config
	relaciones := make([]config.RelationConfig, 0, len(cfg.Relations))
	for _, rel := range cfg.Relations {
		if rel.MappedName == descartar {
			continue
		}
		relaciones = append(relaciones, rel)
	}
	cfg.Relations = relaciones
	return cfg
}

// NewProveedorService construye un ProveedorService con el transporte HTTP, la
// configuración de relaciones del proveedor y el registro relación→resolver del
// núcleo resuelto contra las URLs de los CRUD. generic es el resolver genérico
// de respaldo para relaciones sin resolver específico; puede ser nil. blacklist
// es la lista negra global de campos; puede ser nil o vacía (sin supresión).
func NewProveedorService(
	client clients.HTTPClient,
	urls config.ServiceURLs,
	cfg config.ServiceConfig,
	tipoRegistroNatural int,
	generic helper.Resolver,
	blacklist config.GlobalBlacklist,
) ProveedorService {
	inner := helper.NewHTTPResolver(client, urls)
	registry := helper.NewResolverRegistry(NewResolversNucleo(inner), generic)
	return ProveedorService{
		BaseURL:             urls["gestion_personas_crud"],
		Client:              client,
		Config:              cfg,
		Registry:            registry,
		TipoRegistroNatural: tipoRegistroNatural,
		Blacklist:           blacklist,
	}
}
