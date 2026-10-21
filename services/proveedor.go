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
}

type proveedorReadResponse struct {
	Data []ProveedorRecord `json:"Data"`
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

	base, err := s.proveedorBase(ctx, id)
	if err != nil {
		return helper.ExpandResult{}, err
	}

	// Validar los fields contra la configuración completa: ambas ramas de
	// detalle están declaradas, de modo que pedir la rama que no aplica es una
	// ruta válida (no un 400) que simplemente no se materializa.
	if len(fields) > 0 {
		if _, err := helper.ParsearFields(fields, s.Config.Relations, nil); err != nil {
			return helper.ExpandResult{}, err
		}
	}

	// Podar la rama de detalle que no corresponde al tipo de registro y descartar
	// los fields que la seleccionaban, dejando que el servicio decida la rama.
	cfg, descartada := s.configParaTipo(base.TipoRegistroId)
	fieldsEfectivos := filtrarFieldsDeRama(fields, descartada)

	// El recurso raíz se representa como mapa genérico para que el expansor
	// pueda anidar las relaciones bajo sus mapped_name conservando las llaves
	// locales (Id, TerceroId, TipoRegistroId).
	data := map[string]any{
		"Id":             base.Id,
		"TerceroId":      base.TerceroId,
		"TipoRegistroId": base.TipoRegistroId,
		"Activo":         base.Activo,
	}

	return helper.Expand(ctx, data, cfg, s.Registry, helper.ExpandOptions{Fields: fieldsEfectivos})
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
// que no corresponde al tipo de registro podada, junto con el mapped_name de la
// rama descartada. Un proveedor natural conserva ProveedorNatural y descarta
// ProveedorJuridico, y viceversa.
func (s ProveedorService) configParaTipo(tipoRegistro int) (config.ServiceConfig, string) {
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
	return cfg, descartar
}

// filtrarFieldsDeRama descarta las rutas de fields cuyo primer segmento es la
// rama de detalle podada, de modo que la validación contra la configuración
// reducida no las rechace.
func filtrarFieldsDeRama(fields []string, rama string) []string {
	if len(fields) == 0 {
		return fields
	}
	salida := make([]string, 0, len(fields))
	for _, f := range fields {
		primer := strings.SplitN(strings.TrimSpace(f), ".", 2)[0]
		if primer == rama {
			continue
		}
		salida = append(salida, f)
	}
	return salida
}

// NewProveedorService construye un ProveedorService con el transporte HTTP, la
// configuración de relaciones del proveedor y el registro relación→resolver del
// núcleo resuelto contra las URLs de los CRUD. generic es el resolver genérico
// de respaldo para relaciones sin resolver específico; puede ser nil.
func NewProveedorService(
	client clients.HTTPClient,
	urls config.ServiceURLs,
	cfg config.ServiceConfig,
	tipoRegistroNatural int,
	generic helper.Resolver,
) ProveedorService {
	inner := helper.NewHTTPResolver(client, urls)
	registry := helper.NewResolverRegistry(helper.NewResolversNucleo(inner), generic)
	return ProveedorService{
		BaseURL:             urls["gestion_personas_crud"],
		Client:              client,
		Config:              cfg,
		Registry:            registry,
		TipoRegistroNatural: tipoRegistroNatural,
	}
}
