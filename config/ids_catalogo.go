package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	// EnvTipoContribuyenteNatural identifica el Id esperado de P_NATURAL.
	EnvTipoContribuyenteNatural = "MID_TIPO_CONTRIBUYENTE_NATURAL"
	// EnvTipoContribuyenteJuridica identifica el Id esperado de P_JURIDICA.
	EnvTipoContribuyenteJuridica = "MID_TIPO_CONTRIBUYENTE_JURIDICA"
	// EnvTipoDocumentoNIT identifica el Id del tipo de documento NIT.
	EnvTipoDocumentoNIT = "MID_TIPO_DOCUMENTO_NIT"
)

// OrigenID indica de dónde se obtuvo un Id de catálogo.
type OrigenID string

const (
	OrigenEntorno  OrigenID = "entorno"
	OrigenCatalogo OrigenID = "catálogo"
)

// IDCatalogo es un Id resuelto junto con su origen.
type IDCatalogo struct {
	Valor  int
	Origen OrigenID
}

func (id IDCatalogo) String() string {
	return fmt.Sprintf("%d (%s)", id.Valor, id.Origen)
}

// CatalogoResolver abstrae las consultas de catálogo que usa el arranque. La
// implementa services.TercerosService.
type CatalogoResolver interface {
	ResolverTipoContribuyente(ctx context.Context, codigo string) (int, error)
	ResolverTipoDocumento(ctx context.Context, codigo string) (int, error)
}

// IDsCatalogo agrupa los tres identificadores que el dominio 1 necesita.
type IDsCatalogo struct {
	TipoContribuyenteNatural  IDCatalogo
	TipoContribuyenteJuridica IDCatalogo
	TipoDocumentoNIT          IDCatalogo
}

// CargarIDsCatalogo resuelve los Id de P_NATURAL, P_JURIDICA y NIT. Si la
// variable de entorno correspondiente está definida, se usa su valor sin
// consultar el catálogo; si no, se consulta el CRUD de terceros.
//
// getenv permite inyectar el lector de entorno en las pruebas. Si es nil se usa
// os.Getenv.
func CargarIDsCatalogo(
	ctx context.Context,
	getenv func(string) string,
	resolver CatalogoResolver,
) (IDsCatalogo, error) {
	if resolver == nil {
		return IDsCatalogo{}, errors.New("no hay resolvedor de catálogos configurado")
	}
	if getenv == nil {
		getenv = os.Getenv
	}

	natural, err := resolverID(
		ctx, getenv, EnvTipoContribuyenteNatural, "P_NATURAL",
		resolver.ResolverTipoContribuyente,
	)
	if err != nil {
		return IDsCatalogo{}, err
	}

	juridica, err := resolverID(
		ctx, getenv, EnvTipoContribuyenteJuridica, "P_JURIDICA",
		resolver.ResolverTipoContribuyente,
	)
	if err != nil {
		return IDsCatalogo{}, err
	}

	nit, err := resolverID(
		ctx, getenv, EnvTipoDocumentoNIT, "NIT",
		resolver.ResolverTipoDocumento,
	)
	if err != nil {
		return IDsCatalogo{}, err
	}

	if natural.Valor == juridica.Valor {
		return IDsCatalogo{}, fmt.Errorf(
			"%s y %s no pueden tener el mismo Id (%d)",
			EnvTipoContribuyenteNatural, EnvTipoContribuyenteJuridica, natural.Valor,
		)
	}

	return IDsCatalogo{
		TipoContribuyenteNatural:  natural,
		TipoContribuyenteJuridica: juridica,
		TipoDocumentoNIT:          nit,
	}, nil
}

func resolverID(
	ctx context.Context,
	getenv func(string) string,
	variable string,
	codigo string,
	porCatalogo func(context.Context, string) (int, error),
) (IDCatalogo, error) {
	if valor := strings.TrimSpace(getenv(variable)); valor != "" {
		id, err := strconv.Atoi(valor)
		if err != nil || id <= 0 {
			return IDCatalogo{}, fmt.Errorf("%s debe ser un entero positivo: %q", variable, valor)
		}
		return IDCatalogo{Valor: id, Origen: OrigenEntorno}, nil
	}

	id, err := porCatalogo(ctx, codigo)
	if err != nil {
		return IDCatalogo{}, fmt.Errorf("resolviendo %s (%s): %w", codigo, variable, err)
	}
	if id <= 0 {
		return IDCatalogo{}, fmt.Errorf("el catálogo %s devolvió un Id inválido: %d", codigo, id)
	}
	return IDCatalogo{Valor: id, Origen: OrigenCatalogo}, nil
}
