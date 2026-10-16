package services

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/udistrital/agora_mid/clients"
)

var ErrTerceroAmbiguo = errors.New("multiples terceros para el mismo documento")

type TercerosService struct {
	BaseURL string
	Client  clients.HTTPClient
}

type identificacionResponse struct {
	Id      int    `json:"Id"`
	Numero  string `json:"Numero"`
	Activo  bool   `json:"Activo"`
	Tercero struct {
		Id int `json:"Id"`
	} `json:"TerceroId"`
	TipoDocumento struct {
		Codigo string `json:"CodigoAbreviacion"`
	} `json:"TipoDocumentoId"`
}

func (s TercerosService) BuscarPorDocumento(
	ctx context.Context,
	numero string,
	codigo string,
) (int, bool, error) {
	base := strings.TrimRight(strings.TrimSpace(s.BaseURL), "/")
	numero = strings.TrimSpace(numero)
	codigo = strings.TrimSpace(codigo)

	if base == "" {
		return 0, false, errors.New("falta la URL de Terceros CRUD")
	}
	if numero == "" || codigo == "" {
		return 0, false, errors.New("documento y tipo documental son obligatorios")
	}
	if strings.Contains(numero, ",") || strings.Contains(codigo, ",") {
		return 0, false, errors.New("valor incompatible con el filtro de Terceros")
	}

	encontrados := make(map[int]bool)

	for offset := 0; ; {
		u, err := url.Parse(base + "/v1/datos_identificacion")
		if err != nil {
			return 0, false, fmt.Errorf("URL inválida: %w", err)
		}

		q := u.Query()
		q.Set("query", "Numero:"+numero+
			",TipoDocumentoId.CodigoAbreviacion:"+codigo+
			",Activo:true")
		q.Set("limit", "100")
		q.Set("offset", strconv.Itoa(offset))
		u.RawQuery = q.Encode()

		registros, err := getAPIList[identificacionResponse](ctx, s.Client, u.String())
		if err != nil {
			return 0, false, fmt.Errorf("consulta de Terceros: %w", err)
		}

		for _, r := range registros {
			if r.Id == 0 || r.Tercero.Id == 0 ||
				!r.Activo || r.Numero != numero ||
				r.TipoDocumento.Codigo != codigo {
				continue
			}

			encontrados[r.Tercero.Id] = true
			if len(encontrados) > 1 {
				return 0, false, fmt.Errorf(
					"%w: tipo %s", ErrTerceroAmbiguo, codigo,
				)
			}
		}

		if len(registros) < 100 {
			break
		}
		offset += len(registros)
	}

	if len(encontrados) == 0 {
		return 0, false, nil
	}
	for id := range encontrados {
		return id, true, nil
	}
	return 0, false, nil
}
