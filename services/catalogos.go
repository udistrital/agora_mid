package services

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

var (
	ErrCatalogoNoEncontrado = errors.New("no se encontró un registro activo del catálogo")
	ErrCatalogoAmbiguo      = errors.New("el código del catálogo tiene múltiples coincidencias activas")
)

type catalogoRespuesta struct {
	Id                int    `json:"Id"`
	CodigoAbreviacion string `json:"CodigoAbreviacion"`
	Activo            bool   `json:"Activo"`
}

// ResolverTipoContribuyente resuelve P_NATURAL o P_JURIDICA por código.
func (s TercerosService) ResolverTipoContribuyente(
	ctx context.Context,
	codigo string,
) (int, error) {
	return s.resolverCatalogo(ctx, "/v1/tipo_contribuyente", codigo)
}

// ResolverTipoDocumento resuelve CC, NIT u otro tipo documental por código.
func (s TercerosService) ResolverTipoDocumento(
	ctx context.Context,
	codigo string,
) (int, error) {
	return s.resolverCatalogo(ctx, "/v1/tipo_documento", codigo)
}

func (s TercerosService) resolverCatalogo(
	ctx context.Context,
	recurso string,
	codigo string,
) (int, error) {
	base := strings.TrimRight(strings.TrimSpace(s.BaseURL), "/")
	codigo = strings.TrimSpace(codigo)

	if base == "" {
		return 0, errors.New("falta la URL base de Terceros CRUD")
	}
	if codigo == "" || strings.ContainsAny(codigo, ",:") {
		return 0, errors.New("el código de catálogo está vacío o no es válido")
	}

	endpoint, err := url.Parse(base + recurso)
	if err != nil {
		return 0, fmt.Errorf("URL de catálogo inválida: %w", err)
	}

	query := endpoint.Query()
	query.Set("query", "CodigoAbreviacion:"+codigo+",Activo:true")
	query.Set("limit", "2")
	endpoint.RawQuery = query.Encode()

	var registros []catalogoRespuesta
	if _, err := s.Client.Get(ctx, endpoint.String(), &registros); err != nil {
		return 0, fmt.Errorf("consultando catálogo %s: %w", recurso, err)
	}

	idEncontrado := 0
	coincidencias := 0

	for _, registro := range registros {
		if registro.Id <= 0 ||
			!registro.Activo ||
			registro.CodigoAbreviacion != codigo {
			continue
		}

		coincidencias++
		idEncontrado = registro.Id

		if coincidencias > 1 {
			return 0, fmt.Errorf("%w: %s", ErrCatalogoAmbiguo, codigo)
		}
	}

	if coincidencias == 0 {
		return 0, fmt.Errorf("%w: %s", ErrCatalogoNoEncontrado, codigo)
	}

	return idEncontrado, nil
}
