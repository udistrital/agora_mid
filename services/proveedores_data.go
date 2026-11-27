package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
)

// GetAllProveedoresMapas lista recursos de gestion_proveedores_crud y acepta
// tanto una lista directa como el envelope { "Data": [...] } que devuelve el
// CRUD. Deja el resultado en target y conserva el contrato (ok, status, error)
// de GetAllProveedores.
func GetAllProveedoresMapas(
	ctx context.Context,
	endpoint string,
	queryParams url.Values,
	target *[]map[string]any,
) (ok bool, statusCode int, err error) {
	var raw json.RawMessage
	ok, statusCode, err = GetAllProveedores(ctx, endpoint, queryParams, &raw)
	if statusCode == 404 {
		// gestion_proveedores_crud responde 404 cuando el listado filtrado no
		// tiene filas y se ordena; para el MID eso es una lista vacía.
		*target = []map[string]any{}
		return true, statusCode, nil
	}
	if err != nil || !ok {
		return ok, statusCode, err
	}
	registros, err := decodeAPIList[map[string]any](bytes.TrimSpace(raw))
	if err != nil {
		return false, statusCode, fmt.Errorf("respuesta de lista inválida: %w", err)
	}
	sort.SliceStable(registros, func(i, j int) bool {
		return idDeMapa(registros[i]) < idDeMapa(registros[j])
	})
	*target = registros
	return true, statusCode, nil
}
