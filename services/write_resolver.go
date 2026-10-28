package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/udistrital/agora_mid/clients"
	"github.com/udistrital/agora_mid/config"
)

// claveContexto es el tipo de las llaves de contexto propias del paquete, para
// evitar colisiones con llaves de otros paquetes.
type claveContexto string

// claveAutorModificacion es la llave de contexto bajo la que viaja el
// identificador del autor de la modificación, poblado desde la autenticación.
const claveAutorModificacion claveContexto = "autor_modificacion_id"

// campoAutorModificacion es el nombre del campo de auditoría que toda escritura
// puebla con el autor tomado del contexto, no del cuerpo del cliente.
const campoAutorModificacion = "AutorModificacionId"

// ConContextoAutor devuelve un contexto que transporta el identificador del
// autor de la modificación. El controlador lo puebla desde la autenticación
// antes de invocar la escritura.
func ConContextoAutor(ctx context.Context, autorID int) context.Context {
	return context.WithValue(ctx, claveAutorModificacion, autorID)
}

// AutorDeContexto extrae el identificador del autor de la modificación del
// contexto. Devuelve 0 y false cuando el contexto no transporta un autor válido.
// Es simétrica a ConContextoAutor y permite al llamador verificar el autor en
// curso.
func AutorDeContexto(ctx context.Context) (int, bool) {
	return autorDeContexto(ctx)
}

// autorDeContexto extrae el identificador del autor del contexto. Devuelve 0 y
// false cuando el contexto no transporta un autor válido.
func autorDeContexto(ctx context.Context) (int, bool) {
	v := ctx.Value(claveAutorModificacion)
	if v == nil {
		return 0, false
	}
	autor, ok := v.(int)
	if !ok || autor <= 0 {
		return 0, false
	}
	return autor, true
}

// EntityWriteResolver es un envoltorio fino y reutilizable sobre el cliente HTTP
// que fija el endpoint real de una entidad y materializa su escritura con upsert
// idempotente: resuelve primero si el registro ya existe (por clave natural o por
// el Id presente en el payload) y, según el caso, lo actualiza (PATCH) o lo crea
// (POST). Es el espejo de escritura de EntityResolver.
//
// No decide ramas ni reglas de negocio del subárbol: el motor de escritura
// genérico recorre el árbol y le entrega, por entidad, el payload ya aplanado y
// los valores de la clave natural declarada. El resolver solo traduce eso a la
// llamada HTTP correcta y devuelve el registro materializado con su Id.
type EntityWriteResolver struct {
	client     clients.HTTPClient
	urls       config.ServiceURLs
	relatedAPI string
	endpoint   string
	// naturalKeyQuery construye, a partir de los valores de la clave natural, el
	// parámetro query que localiza un registro existente. Si es nil, la entidad
	// no se busca por clave natural (la idempotencia se apoya en el Id del
	// payload).
	naturalKeyQuery func(keys []string) (string, bool)
}

// NewEntityWriteResolver construye un write-resolver que fija el servicio externo
// y el endpoint real de la entidad. naturalKeyQuery es opcional: cuando se
// provee, el resolver busca un registro existente por clave natural antes de
// crear, habilitando el upsert idempotente.
func NewEntityWriteResolver(
	client clients.HTTPClient,
	urls config.ServiceURLs,
	relatedAPI string,
	endpoint string,
	naturalKeyQuery func(keys []string) (string, bool),
) *EntityWriteResolver {
	return &EntityWriteResolver{
		client:          client,
		urls:            urls,
		relatedAPI:      relatedAPI,
		endpoint:        endpoint,
		naturalKeyQuery: naturalKeyQuery,
	}
}

type upsertRespuesta struct {
	Id   int `json:"Id"`
	Data struct {
		Id int `json:"Id"`
	} `json:"Data"`
}

func (r upsertRespuesta) id() int {
	if r.Id > 0 {
		return r.Id
	}
	return r.Data.Id
}

// Materialize ejecuta el upsert idempotente de la entidad. Inyecta el autor de
// la modificación desde el contexto, resuelve si el registro ya existe (por clave
// natural o por Id) y aplica PATCH sobre el existente o POST para crearlo. Devuelve
// el payload materializado con el Id resuelto.
func (r *EntityWriteResolver) Materialize(
	ctx context.Context,
	_ config.RelationConfig,
	payload map[string]any,
	keys []string,
) (map[string]any, error) {
	base, ok := r.urls[r.relatedAPI]
	if !ok || strings.TrimSpace(base) == "" {
		return nil, fmt.Errorf("write-resolver: no hay URL base para %q", r.relatedAPI)
	}
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	endpoint := "/" + strings.TrimLeft(strings.TrimSpace(r.endpoint), "/")

	// Inyectar el autor de auditoría desde el contexto, nunca desde el cuerpo.
	cuerpo := clonarPayload(payload)
	if autor, ok := autorDeContexto(ctx); ok {
		cuerpo[campoAutorModificacion] = autor
	}

	// Resolver el Id del registro existente: por el Id del payload o por la clave
	// natural declarada.
	idExistente, err := r.resolverExistente(ctx, base, endpoint, payload, keys)
	if err != nil {
		return nil, err
	}

	if idExistente > 0 {
		cuerpo["Id"] = idExistente
		destino := fmt.Sprintf("%s%s/%d", base, endpoint, idExistente)
		var resp upsertRespuesta
		if _, err := r.client.Patch(ctx, destino, cuerpo, &resp); err != nil {
			return nil, fmt.Errorf("write-resolver: actualización falló: %w", err)
		}
		salida := clonarPayload(cuerpo)
		salida["Id"] = idExistente
		return salida, nil
	}

	var resp upsertRespuesta
	if _, err := r.client.Post(ctx, base+endpoint+"/", cuerpo, &resp); err != nil {
		return nil, fmt.Errorf("write-resolver: creación falló: %w", err)
	}
	nuevoID := resp.id()
	if nuevoID <= 0 {
		return nil, errors.New("write-resolver: el CRUD no devolvió un Id válido")
	}
	salida := clonarPayload(cuerpo)
	salida["Id"] = nuevoID
	return salida, nil
}

// Recuperar resuelve el identificador autoritativo de un registro existente
// usando únicamente lecturas (GET), sin ejecutar ninguna escritura. Reutiliza la
// misma lógica de resolución del upsert (resolverExistente): prioriza el Id del
// payload y, si no lo hay, busca por la clave natural declarada. Devuelve 0
// cuando no existe un registro reutilizable.
//
// Es la operación de recuperación del contrato WriteResolver que el motor usa
// para validar antes de materializar, de modo que un 400 por conflicto no deje
// escrituras parciales en los CRUD dueños.
func (r *EntityWriteResolver) Recuperar(
	ctx context.Context,
	_ config.RelationConfig,
	payload map[string]any,
	keys []string,
) (int, error) {
	base, ok := r.urls[r.relatedAPI]
	if !ok || strings.TrimSpace(base) == "" {
		return 0, fmt.Errorf("write-resolver: no hay URL base para %q", r.relatedAPI)
	}
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	endpoint := "/" + strings.TrimLeft(strings.TrimSpace(r.endpoint), "/")
	return r.resolverExistente(ctx, base, endpoint, payload, keys)
}

// resolverExistente determina el Id de un registro existente para el upsert.
// Prioriza el Id presente en el payload (propagado por PK compartida o enviado
// por el cliente en una colección); si no hay Id y la entidad declara clave
// natural, busca por ella. Devuelve 0 cuando no existe un registro reutilizable.
func (r *EntityWriteResolver) resolverExistente(
	ctx context.Context,
	base string,
	endpoint string,
	payload map[string]any,
	keys []string,
) (int, error) {
	if id := idEntero(payload["Id"]); id > 0 {
		return id, nil
	}
	if r.naturalKeyQuery == nil || len(keys) == 0 {
		return 0, nil
	}
	consulta, ok := r.naturalKeyQuery(keys)
	if !ok {
		return 0, nil
	}

	u, err := url.Parse(base + endpoint)
	if err != nil {
		return 0, fmt.Errorf("write-resolver: URL inválida: %w", err)
	}
	q := u.Query()
	q.Set("query", consulta)
	q.Set("limit", "1")
	q.Set("offset", "0")
	u.RawQuery = q.Encode()

	var raw json.RawMessage
	if _, err := r.client.Get(ctx, u.String(), &raw); err != nil {
		return 0, fmt.Errorf("write-resolver: búsqueda por clave natural falló: %w", err)
	}
	registros, err := decodeAPIList[map[string]any](raw)
	if err != nil {
		return 0, fmt.Errorf("write-resolver: respuesta de búsqueda inválida: %w", err)
	}
	for _, reg := range registros {
		if id := idEntero(reg["Id"]); id > 0 {
			return id, nil
		}
	}
	return 0, nil
}

// clonarPayload devuelve una copia superficial del payload para no mutar la
// entrada del motor.
func clonarPayload(m map[string]any) map[string]any {
	c := make(map[string]any, len(m)+1)
	for k, v := range m {
		c[k] = v
	}
	return c
}

// idEntero normaliza a entero un valor de Id que puede venir como int o como
// float64 (decodificación JSON). Devuelve 0 para valores ausentes o no numéricos.
func idEntero(v any) int {
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	default:
		return 0
	}
}
