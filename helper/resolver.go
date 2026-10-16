// Package helper resuelve y expande relaciones declaradas entre CRUD como
// objetos anidados. Expone la interfaz Resolver y sus adaptadores sobre el
// cliente HTTP, además del expansor recursivo.
package helper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/udistrital/agora_mid/config"
)

const tokenIDs = "{ids}"

// Resolver resuelve una relación declarada contra un servicio externo.
//
// Resolve recibe la configuración de la relación y las llaves locales a
// consultar, y devuelve los registros remotos decodificados como payload
// genérico. Respeta la cancelación y los timeouts de ctx. No transforma la
// semántica de los datos.
type Resolver interface {
	Resolve(ctx context.Context, rel config.RelationConfig, keys []string) ([]map[string]any, error)
}

type httpGetter interface {
	Get(ctx context.Context, url string, target any) (int, error)
}

// HTTPResolver adapta un cliente HTTP a la interfaz Resolver. Resuelve la URL
// base a partir de RelatedAPI contra URLs y codifica las llaves en el parámetro
// de consulta "query" usando el QueryTemplate de la relación.
type HTTPResolver struct {
	// Client ejecuta las peticiones GET al servicio externo.
	Client httpGetter
	// URLs mapea el valor de RelatedAPI a la URL base del servicio, sin el
	// sufijo del endpoint.
	URLs config.ServiceURLs
}

// NewHTTPResolver construye un HTTPResolver con el cliente y el mapa de URLs
// base por servicio.
func NewHTTPResolver(client httpGetter, urls config.ServiceURLs) HTTPResolver {
	return HTTPResolver{Client: client, URLs: urls}
}

// Resolve consulta el servicio externo de la relación y devuelve los registros
// decodificados. Acepta tanto el envoltorio OAS estándar
// (Success/Status/Message/Data) como una lista JSON directa.
//
// Devuelve error si RelatedAPI no tiene URL base configurada, si la petición
// falla o se cancela por ctx, o si la respuesta no contiene una lista.
func (r HTTPResolver) Resolve(
	ctx context.Context,
	rel config.RelationConfig,
	keys []string,
) ([]map[string]any, error) {
	base, ok := r.URLs[rel.RelatedAPI]
	if !ok || strings.TrimSpace(base) == "" {
		return nil, fmt.Errorf("resolver: no hay URL base para related_api %q", rel.RelatedAPI)
	}
	base = strings.TrimRight(strings.TrimSpace(base), "/")

	endpoint := "/" + strings.TrimLeft(strings.TrimSpace(rel.Endpoint), "/")
	u, err := url.Parse(base + endpoint)
	if err != nil {
		return nil, fmt.Errorf("resolver: URL inválida para %q: %w", rel.Name, err)
	}

	q := u.Query()
	q.Set("query", construirQuery(rel.QueryTemplate, keys))
	u.RawQuery = q.Encode()

	var raw json.RawMessage
	if _, err := r.Client.Get(ctx, u.String(), &raw); err != nil {
		return nil, fmt.Errorf("resolver: consulta a %q falló: %w", rel.RelatedAPI, err)
	}

	return decodeLista(raw)
}

// construirQuery reemplaza el marcador {ids} de la plantilla por las llaves
// unidas por comas. Con plantilla vacía devuelve solo las llaves unidas.
func construirQuery(plantilla string, keys []string) string {
	unidas := strings.Join(keys, ",")
	plantilla = strings.TrimSpace(plantilla)
	if plantilla == "" {
		return unidas
	}
	return strings.ReplaceAll(plantilla, tokenIDs, unidas)
}

type listaEnvelope struct {
	Data []map[string]any `json:"Data"`
}

// decodeLista decodifica una respuesta de lista aceptando tanto el envoltorio
// OAS estándar (Success/Status/Message/Data) como una lista JSON directa.
// Devuelve error si el contenido no es ninguno de los dos o no contiene lista.
func decodeLista(raw json.RawMessage) ([]map[string]any, error) {
	trimmed := []byte(strings.TrimSpace(string(raw)))

	var directa []map[string]any
	if err := json.Unmarshal(trimmed, &directa); err == nil {
		return directa, nil
	}

	var envelope listaEnvelope
	if err := json.Unmarshal(trimmed, &envelope); err != nil {
		return nil, fmt.Errorf("resolver: respuesta de lista inválida: %w", err)
	}
	if envelope.Data == nil {
		return nil, errors.New("resolver: la respuesta del CRUD no contiene una lista Data")
	}
	return envelope.Data, nil
}
