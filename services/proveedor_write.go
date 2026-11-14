package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/udistrital/agora_mid/clients"
	"github.com/udistrital/agora_mid/config"
	"github.com/udistrital/agora_mid/helper"
)

// ErrProveedorNoExiste indica que un PATCH referencia por Id un proveedor que el CRUD
// no tiene. El controlador lo mapea a 404 Not Found. Ningún registro se escribe.
var ErrProveedorNoExiste = errors.New("el proveedor indicado por Id no existe")

// ErrRamaDetalleIncoherente indica que el cuerpo de escritura declara una rama de
// detalle (ProveedorNatural/ProveedorJuridico) que no corresponde al tipo de
// registro del proveedor, o declara ambas a la vez. El controlador debe mapearlo
// a 400 Bad Request. Es una regla de negocio irreductible que no vive en el motor
// genérico sino en el servicio del proveedor.
var ErrRamaDetalleIncoherente = errors.New(
	"la rama de detalle del cuerpo no corresponde al tipo de registro del proveedor",
)

// ProveedorWriteService orquesta la escritura del proveedor como recurso raíz:
// aplica las reglas de negocio irreductibles (coherencia natural/jurídico), poda
// del árbol la rama de detalle que no corresponde al tipo de registro e invoca el
// motor de escritura genérico con el registro de write-resolvers del núcleo.
//
// Las reglas estructurales (orden topológico, propagación de llave, aplanado,
// idempotencia, inmutables) viven en el motor y la configuración; aquí solo vive
// lo que el motor no puede expresar de forma declarativa.
type ProveedorWriteService struct {
	// Config es la configuración declarativa de relaciones del proveedor, con su
	// bloque write.
	Config config.ServiceConfig
	// Registry es el registro de write-resolvers que el servicio inyecta al motor.
	Registry helper.WriteResolverRegistry
	// TipoRegistroNatural es el identificador de tipo de contribuyente que
	// corresponde a persona natural. Determina qué rama de detalle es válida.
	TipoRegistroNatural int
	// Client y URLs permiten leer el estado vigente del proveedor antes de componer
	// la transición de estado_proveedor.
	Client clients.HTTPClient
	URLs   config.ServiceURLs
}

// NewProveedorWriteService construye el servicio de escritura con la
// configuración de relaciones del proveedor y el registro de write-resolvers del
// núcleo resuelto contra las URLs de los CRUD.
func NewProveedorWriteService(
	client clients.HTTPClient,
	urls config.ServiceURLs,
	cfg config.ServiceConfig,
	tipoRegistroNatural int,
) ProveedorWriteService {
	raiz, hijos := NewWriteResolversNucleo(client, urls)
	registry := helper.NewWriteResolverRegistry(raiz, hijos)
	return ProveedorWriteService{
		Config:              cfg,
		Registry:            registry,
		TipoRegistroNatural: tipoRegistroNatural,
		Client:              client,
		URLs:                urls,
	}
}

// EscribirProveedor materializa el subárbol del proveedor a partir del cuerpo
// anidado. verb es WriteVerbCreate (POST) o WriteVerbUpdate (PATCH). Aplica la
// coherencia natural/jurídico, poda la rama de detalle que no corresponde y
// delega la cascada en el motor de escritura.
//
// Devuelve ErrRamaDetalleIncoherente o *helper.ErrorEscritura (ambos 400) ante un
// cuerpo inválido, ErrProveedorNoExiste (404) si un PATCH referencia por Id un
// proveedor inexistente, un error fatal de escritura (ver helper.EsErrorFatalEscritura)
// cuando la política de relación es fail, o el WriteResult con los identificadores
// materializados en el caso tolerante. Ante un PATCH con Id, ningún registro se
// escribe si la verificación de existencia falla.
func (s ProveedorWriteService) EscribirProveedor(
	ctx context.Context,
	body map[string]any,
	verb string,
) (helper.WriteResult, error) {
	if body == nil {
		return helper.WriteResult{}, &helper.ErrorEscritura{
			Path: "(raíz)", Motivo: "el cuerpo de escritura es obligatorio",
		}
	}

	tipoRegistro := idEntero(body["TipoProveedor"])
	if tipoRegistro <= 0 {
		return helper.WriteResult{}, &helper.ErrorEscritura{
			Path: "TipoProveedor", Motivo: "es obligatorio y debe ser un entero positivo",
		}
	}

	// Regla irreductible: la rama de detalle debe corresponder al tipo de
	// registro, y no pueden declararse ambas ramas a la vez.
	if err := s.validarCoherenciaRama(body, tipoRegistro); err != nil {
		return helper.WriteResult{}, err
	}

	// Un PATCH debe identificar el proveedor existente por Id o por su clave natural
	// TerceroId. Se verifica antes de cualquier escritura.
	if verb == helper.WriteVerbUpdate {
		if err := s.verificarIdentidadPatch(ctx, body); err != nil {
			return helper.WriteResult{}, err
		}
	}

	// Podar del árbol la rama de detalle que no corresponde al tipo de registro,
	// igual que en lectura; la rama ausente del cuerpo tampoco se recorre.
	// Un alta con TerceroId suelto (sin objeto Tercero) debe referir un tercero existente en
	// el CRUD de terceros. Con objeto Tercero, el motor crea o actualiza el tercero en la misma
	// escritura y propaga su Id.
	if verb == helper.WriteVerbCreate && body["Tercero"] == nil {
		if terceroID := idEntero(body["TerceroId"]); terceroID > 0 {
			if err := s.verificarTerceroExistente(ctx, terceroID); err != nil {
				return helper.WriteResult{}, err
			}
		}
	}

	cfg := s.configParaTipo(tipoRegistro)

	// La transición de estado se compone antes que la propagación de autor, para que
	// el estado nuevo también reciba el autor de la raíz.
	body, err := s.componerTransicionEstado(ctx, body)
	if err != nil {
		return helper.WriteResult{}, err
	}

	// El autor de la raíz se propaga a los objetos anidados que lo declaran. El
	// motor recibe el árbol ya compuesto y no conoce el campo de auditoría.
	body = propagarAutorDeRaiz(body, cfg.Relations)

	return helper.Write(ctx, body, cfg, s.Registry, helper.WriteOptions{Verb: verb})
}

// validarCoherenciaRama verifica que el cuerpo declare únicamente la rama de
// detalle que corresponde al tipo de registro. Un proveedor natural no puede
// traer ProveedorJuridico y viceversa, ni ambas a la vez.
func (s ProveedorWriteService) validarCoherenciaRama(body map[string]any, tipoRegistro int) error {
	_, traeNatural := body[relacionProveedorNatural]
	_, traeJuridico := body[relacionProveedorJuridico]

	if traeNatural && traeJuridico {
		return ErrRamaDetalleIncoherente
	}

	esNatural := tipoRegistro == s.TipoRegistroNatural
	if esNatural && traeJuridico {
		return ErrRamaDetalleIncoherente
	}
	if !esNatural && traeNatural {
		return ErrRamaDetalleIncoherente
	}
	return nil
}

// configParaTipo devuelve la configuración con la rama de detalle que no
// corresponde al tipo de registro podada, de modo que el motor no intente
// materializar la rama incorrecta.
func (s ProveedorWriteService) configParaTipo(tipoRegistro int) config.ServiceConfig {
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

// motivoTerceroInexistente es el motivo del 400 cuando un alta referencia un TerceroId
// que el CRUD de terceros no tiene. No expone valores del CRUD.
const motivoTerceroInexistente = "el TerceroId no existe en el CRUD de terceros"

// verificarTerceroExistente consulta por GET el tercero con el TerceroId dado en el CRUD
// de terceros. Devuelve *helper.ErrorEscritura (400) si el CRUD responde 404 y un error
// distinto ante cualquier otro fallo de lectura. No escribe nada.
func (s ProveedorWriteService) verificarTerceroExistente(ctx context.Context, terceroID int) error {
	base, ok := s.URLs["terceros"]
	if !ok || strings.TrimSpace(base) == "" {
		return fmt.Errorf("verificando el tercero %d: no hay URL base del CRUD de terceros", terceroID)
	}
	destino := fmt.Sprintf("%s/v1/tercero/%d", strings.TrimRight(strings.TrimSpace(base), "/"), terceroID)

	var raw json.RawMessage
	status, err := s.Client.Get(ctx, destino, &raw)
	if status == http.StatusNotFound {
		return &helper.ErrorEscritura{Path: "TerceroId", Motivo: motivoTerceroInexistente}
	}
	if err != nil {
		return fmt.Errorf("verificando el tercero %d: %w", terceroID, err)
	}
	return nil
}

// motivoPatchSinIdentidad es el motivo del 400 cuando un PATCH no trae Id ni TerceroId
// válido. No expone valores del cuerpo.
const motivoPatchSinIdentidad = "un PATCH requiere Id o TerceroId para identificar el proveedor"

// verificarIdentidadPatch exige que un PATCH identifique un proveedor existente. Con
// Id, el proveedor debe existir por Id; sin Id, TerceroId (clave natural) debe
// devolver un proveedor. Sin Id ni TerceroId mayor que cero devuelve *helper.ErrorEscritura
// (400). Si el Id o la clave natural no devuelven ningún proveedor, devuelve
// ErrProveedorNoExiste (404). No escribe nada.
func (s ProveedorWriteService) verificarIdentidadPatch(ctx context.Context, body map[string]any) error {
	if id := idEntero(body["Id"]); id > 0 {
		registro, err := s.verificarProveedorExistente(ctx, id)
		if err != nil {
			return err
		}
		// Un TerceroId distinto del registro reasignaría el proveedor. Un TerceroId
		// ausente, 0 o null no es conflicto, y un registro sin tercero no bloquea.
		terceroBody := idEntero(body["TerceroId"])
		terceroRegistro := idEntero(registro["TerceroId"])
		if terceroBody > 0 && terceroRegistro > 0 && terceroBody != terceroRegistro {
			return &helper.ErrorEscritura{Path: "TerceroId", Motivo: motivoTerceroEnConflicto}
		}
		return nil
	}
	terceroID := idEntero(body["TerceroId"])
	if terceroID <= 0 {
		return &helper.ErrorEscritura{Path: "(raíz)", Motivo: motivoPatchSinIdentidad}
	}
	return s.verificarProveedorPorTercero(ctx, terceroID)
}

// motivoTerceroEnConflicto es el motivo del 400 cuando un PATCH reenvía un TerceroId
// distinto del que tiene el proveedor existente. No expone valores del CRUD.
const motivoTerceroEnConflicto = "el TerceroId no coincide con el del proveedor existente; no se reasigna"

// verificarProveedorExistente consulta por GET el proveedor con el Id dado en el CRUD
// gestion_personas_crud y devuelve su registro. Devuelve ErrProveedorNoExiste si el CRUD
// responde 404. Ante cualquier otro fallo de lectura devuelve un error distinto.
func (s ProveedorWriteService) verificarProveedorExistente(ctx context.Context, id int) (map[string]any, error) {
	base, err := s.baseProveedor()
	if err != nil {
		return nil, err
	}
	destino := fmt.Sprintf("%s/v1/proveedor/%d", base, id)

	var raw json.RawMessage
	status, err := s.Client.Get(ctx, destino, &raw)
	if status == http.StatusNotFound {
		return nil, ErrProveedorNoExiste
	}
	if err != nil {
		return nil, fmt.Errorf("verificando el proveedor %d: %w", id, err)
	}
	registro, err := decodeAPIRecord(raw)
	if err != nil {
		return nil, fmt.Errorf("verificando el proveedor %d: respuesta inválida: %w", id, err)
	}
	return registro, nil
}

// verificarProveedorPorTercero busca en el CRUD gestion_personas_crud un proveedor activo
// con el TerceroId dado, con el mismo filtro que usa el motor en su clave natural. Devuelve
// ErrProveedorNoExiste si la lista no tiene coincidencias.
func (s ProveedorWriteService) verificarProveedorPorTercero(ctx context.Context, terceroID int) error {
	base, err := s.baseProveedor()
	if err != nil {
		return err
	}
	consulta, ok := queryPorTerceroId([]string{strconv.Itoa(terceroID)})
	if !ok {
		return fmt.Errorf("verificando el proveedor por TerceroId %d: consulta inválida", terceroID)
	}
	params := url.Values{}
	params.Set("query", consulta)
	params.Set("limit", "1")
	params.Set("offset", "0")
	destino := base + "/v1/proveedor?" + params.Encode()

	var raw json.RawMessage
	if _, err := s.Client.Get(ctx, destino, &raw); err != nil {
		return fmt.Errorf("verificando el proveedor por TerceroId %d: %w", terceroID, err)
	}
	registros, err := decodeAPIList[map[string]any](raw)
	if err != nil {
		return fmt.Errorf("verificando el proveedor por TerceroId %d: respuesta inválida: %w", terceroID, err)
	}
	for _, reg := range registros {
		if idEntero(reg["Id"]) > 0 {
			return nil
		}
	}
	return ErrProveedorNoExiste
}

// baseProveedor devuelve la URL base del CRUD gestion_personas_crud sin barra final.
func (s ProveedorWriteService) baseProveedor() (string, error) {
	base, ok := s.URLs["gestion_personas_crud"]
	if !ok || strings.TrimSpace(base) == "" {
		return "", fmt.Errorf("verificando el proveedor: no hay URL base del CRUD gestion_personas")
	}
	return strings.TrimRight(strings.TrimSpace(base), "/"), nil
}
