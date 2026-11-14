package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/udistrital/agora_mid/clients"
	"github.com/udistrital/agora_mid/config"
)

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

// Materialize escribe la entidad en el CRUD dueño. Inyecta el autor de la
// modificación desde el contexto. Un payload con Id identifica el registro existente:
// se lee ese registro completo (GET), se le superponen los valores recibidos y el
// resultado se envía por PUT. Sin Id se crea por POST. No consulta el CRUD para decidir
// la existencia: esa decisión la toma el motor en el pre-pass, con el registro
// recuperado por Recuperar, y llega aquí como Id. Devuelve el payload materializado
// con el Id resultante. Si el registro existente no se puede leer completo, devuelve
// error y no envía ninguna escritura.
func (r *EntityWriteResolver) Materialize(
	ctx context.Context,
	_ config.RelationConfig,
	payload map[string]any,
	_ []string,
) (map[string]any, error) {
	base, ok := r.urls[r.relatedAPI]
	if !ok || strings.TrimSpace(base) == "" {
		return nil, fmt.Errorf("write-resolver: no hay URL base para %q", r.relatedAPI)
	}
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	endpoint := "/" + strings.TrimLeft(strings.TrimSpace(r.endpoint), "/")

	cuerpo := clonarPayload(payload)

	// Resolver el Id del registro existente: por el Id del payload o por la clave
	// natural declarada.
	idExistente := idEntero(payload["Id"])

	if idExistente > 0 {
		cuerpo["Id"] = idExistente
		completo, existe, err := r.leerRegistroCompleto(ctx, base, endpoint, idExistente)
		if err != nil {
			return nil, fmt.Errorf("write-resolver: lectura previa a la actualización falló: %w", err)
		}
		if existe {
			destino := fmt.Sprintf("%s%s/%d", base, endpoint, idExistente)
			var resp upsertRespuesta
			if _, err := r.client.Put(ctx, destino, envolverFKRelacional(fusionarRegistro(completo, cuerpo, idExistente)), &resp); err != nil {
				return nil, fmt.Errorf("write-resolver: actualización falló: %w", err)
			}
			salida := clonarPayload(cuerpo)
			salida["Id"] = idExistente
			return salida, nil
		}
		// El registro no existe: se crea con el Id propagado (PK compartida).
	}

	var resp upsertRespuesta
	if _, err := r.client.Post(ctx, base+endpoint+"/", envolverFKRelacional(clonarPayload(cuerpo)), &resp); err != nil {
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

// Recuperar devuelve el registro existente de la entidad, con todos sus campos
// almacenados, usando únicamente lecturas (GET). Prioriza el Id presente en el
// payload (lectura por Id) y, si no lo hay, busca por la clave natural declarada.
// Devuelve nil, sin error, cuando no existe un registro reutilizable.
//
// Es la operación de recuperación del contrato WriteResolver con la que el motor
// retiene el estado de cada nodo antes de escribir: la clasificación de acción y el
// control de no-reparent necesitan los campos del registro, no solo su Id.
func (r *EntityWriteResolver) Recuperar(
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
	return r.buscarRegistroExistente(ctx, base, endpoint, payload, keys)
}

// buscarRegistroExistente localiza el registro existente de una entidad para la
// recuperación. Si el payload trae un Id propio, lee ese registro por su Id; si no,
// busca por la clave natural declarada. Devuelve nil cuando no existe un registro
// reutilizable.
func (r *EntityWriteResolver) buscarRegistroExistente(
	ctx context.Context,
	base string,
	endpoint string,
	payload map[string]any,
	keys []string,
) (map[string]any, error) {
	if id := idEntero(payload["Id"]); id > 0 {
		var raw json.RawMessage
		status, err := r.client.Get(ctx, fmt.Sprintf("%s%s/%d", base, endpoint, id), &raw)
		if status == http.StatusNotFound {
			// Sin registro con ese Id no hay nada que recuperar.
			return nil, nil
		}
		if err != nil {
			return nil, fmt.Errorf("write-resolver: lectura por Id falló: %w", err)
		}
		registro, err := decodeAPIRecord(raw)
		if err != nil {
			return nil, fmt.Errorf("write-resolver: respuesta de lectura inválida: %w", err)
		}
		// Un Id propio identifica una entidad existente aunque el CRUD no devuelva sus
		// campos: el registro mínimo contiene ese Id.
		if idEntero(registro["Id"]) <= 0 {
			return map[string]any{"Id": id}, nil
		}
		return registro, nil
	}
	return r.buscarPorClaveNatural(ctx, base, endpoint, keys)
}

// buscarPorClaveNatural devuelve el primer registro que coincide con la clave natural
// declarada. Devuelve nil cuando la entidad no declara clave natural o no hay coincidencia.
func (r *EntityWriteResolver) buscarPorClaveNatural(
	ctx context.Context,
	base string,
	endpoint string,
	keys []string,
) (map[string]any, error) {
	if r.naturalKeyQuery == nil || len(keys) == 0 {
		return nil, nil
	}
	consulta, ok := r.naturalKeyQuery(keys)
	if !ok {
		return nil, nil
	}

	u, err := url.Parse(base + endpoint)
	if err != nil {
		return nil, fmt.Errorf("write-resolver: URL inválida: %w", err)
	}
	q := u.Query()
	q.Set("query", consulta)
	q.Set("limit", "1")
	q.Set("offset", "0")
	u.RawQuery = q.Encode()

	var raw json.RawMessage
	if _, err := r.client.Get(ctx, u.String(), &raw); err != nil {
		return nil, fmt.Errorf("write-resolver: búsqueda por clave natural falló: %w", err)
	}
	registros, err := decodeAPIList[map[string]any](raw)
	if err != nil {
		return nil, fmt.Errorf("write-resolver: respuesta de búsqueda inválida: %w", err)
	}
	for _, reg := range registros {
		if idEntero(reg["Id"]) > 0 {
			return reg, nil
		}
	}
	return nil, nil
}

// Eliminar da de baja el registro con el Id dado mediante DELETE sobre el recurso del
// CRUD dueño. Devuelve error si el Id no es válido o el CRUD rechaza la baja. No
// consulta el CRUD antes de borrar: la decisión de existencia la toma el motor en el
// pre-pass.
func (r *EntityWriteResolver) Eliminar(
	ctx context.Context,
	_ config.RelationConfig,
	id any,
) error {
	base, ok := r.urls[r.relatedAPI]
	if !ok || strings.TrimSpace(base) == "" {
		return fmt.Errorf("write-resolver: no hay URL base para %q", r.relatedAPI)
	}
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	endpoint := "/" + strings.TrimLeft(strings.TrimSpace(r.endpoint), "/")

	idRegistro := idEntero(id)
	if idRegistro <= 0 {
		return errors.New("write-resolver: el registro a eliminar no tiene un Id válido")
	}
	if _, err := r.client.Delete(ctx, fmt.Sprintf("%s%s/%d", base, endpoint, idRegistro), nil); err != nil {
		return fmt.Errorf("write-resolver: eliminación falló: %w", err)
	}
	return nil
}

// decodeAPIRecord decodifica la respuesta de lectura de un único registro, acepta
// el sobre estándar de los CRUD ({"Data": {...}}) y un objeto JSON directo.
func decodeAPIRecord(raw json.RawMessage) (map[string]any, error) {
	var envelope struct {
		Data map[string]any `json:"Data"`
	}
	if err := json.Unmarshal(raw, &envelope); err == nil && envelope.Data != nil {
		return envelope.Data, nil
	}
	var directo map[string]any
	if err := json.Unmarshal(raw, &directo); err != nil {
		return nil, err
	}
	return directo, nil
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

// envolverFKRelacional convierte las FK de relación que el CRUD decodifica como
// objeto (ProveedorId, ContactoId) en {"Id": n}. Trabaja sobre una copia que se
// envía al CRUD; el payload interno conserva el entero. Un valor nulo o ya
// objeto no se toca. Solución rápida: solo cubre las dos FK conocidas.
func envolverFKRelacional(cuerpo map[string]any) map[string]any {
	for _, clave := range []string{"ProveedorId", "ContactoId"} {
		v, ok := cuerpo[clave]
		if !ok || v == nil {
			continue
		}
		if _, esObjeto := v.(map[string]any); esObjeto {
			continue

		}
		cuerpo[clave] = map[string]any{"Id": v}
	}
	return cuerpo
}

// leerRegistroCompleto lee por Id el registro existente con todos sus campos (GET).
// Devuelve existe=false, sin error, cuando el CRUD responde 404. Devuelve error si el
// CRUD falla o si trae solo el Id, para que una actualización nunca envíe un PUT que
// anule campos desconocidos.
func (r *EntityWriteResolver) leerRegistroCompleto(
	ctx context.Context,
	base string,
	endpoint string,
	id int,
) (registro map[string]any, existe bool, err error) {
	var raw json.RawMessage
	status, err := r.client.Get(ctx, fmt.Sprintf("%s%s/%d", base, endpoint, id), &raw)
	if status == http.StatusNotFound {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	registro, err = decodeAPIRecord(raw)
	if err != nil {
		return nil, false, err
	}
	if idEntero(registro["Id"]) != id || len(registro) < 2 {
		return nil, false, fmt.Errorf("el registro %d no trae sus campos", id)
	}
	return registro, true, nil
}

// fusionarRegistro devuelve el registro leído con los valores recibidos encima, con el
// Id indicado. Las FK que el CRUD devuelve como objeto completo se reducen a {"Id": n}.
// No modifica los mapas recibidos.
func fusionarRegistro(leido, recibido map[string]any, id int) map[string]any {
	salida := make(map[string]any, len(leido)+len(recibido)+1)
	for k, v := range leido {
		salida[k] = reducirFKObjeto(k, v)
	}
	for k, v := range recibido {
		salida[k] = reducirFKObjeto(k, v)
	}
	salida["Id"] = id
	return salida
}

// reducirFKObjeto convierte una FK con forma de objeto ({"Id": n, ...}) en {"Id": n}.
// Cualquier otro valor se devuelve sin cambios.
func reducirFKObjeto(clave string, valor any) any {
	if !strings.HasSuffix(clave, "Id") {
		return valor
	}
	objeto, ok := valor.(map[string]any)
	if !ok {
		return valor
	}
	id, tieneId := objeto["Id"]
	if !tieneId || id == nil {
		return valor
	}
	return map[string]any{"Id": id}
}
