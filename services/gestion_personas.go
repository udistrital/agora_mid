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

var ErrProveedorAmbiguo = errors.New(
	"hay varios proveedores asociados al mismo tercero",
)

type GestionPersonasService struct {
	BaseURL string
	Client  clients.HTTPClient
}

type ProveedorRecord struct {
	Id             int  `json:"Id"`
	TerceroId      int  `json:"TerceroId"`
	TipoRegistroId int  `json:"TipoRegistroId"`
	Activo         bool `json:"Activo"`
}

type proveedorListResponse struct {
	Data []ProveedorRecord `json:"Data"`
}

// BuscarProveedorPorTercero permite reanudar el registro sin crear
// un segundo proveedor para el mismo tercero.
func (s GestionPersonasService) BuscarProveedorPorTercero(
	ctx context.Context,
	terceroID int,
) (*ProveedorRecord, error) {
	base := strings.TrimRight(strings.TrimSpace(s.BaseURL), "/")

	if base == "" {
		return nil, errors.New("falta la URL de gestión de personas CRUD")
	}
	if terceroID <= 0 {
		return nil, errors.New("el tercero_id debe ser mayor que cero")
	}

	endpoint, err := url.Parse(base + "/v1/proveedor/")
	if err != nil {
		return nil, fmt.Errorf("URL de gestión de personas inválida: %w", err)
	}

	query := endpoint.Query()
	query.Set("query", "TerceroId:"+strconv.Itoa(terceroID))
	query.Set("limit", "2")
	query.Set("offset", "0")
	endpoint.RawQuery = query.Encode()

	var respuesta proveedorListResponse
	if _, err := s.Client.Get(ctx, endpoint.String(), &respuesta); err != nil {
		return nil, fmt.Errorf("consultando proveedor por tercero_id: %w", err)
	}

	var encontrado *ProveedorRecord

	for _, registro := range respuesta.Data {
		// El CRUD puede devolver [{}] cuando no hay coincidencias.
		if registro.Id <= 0 || registro.TerceroId != terceroID {
			continue
		}

		if encontrado != nil && encontrado.Id != registro.Id {
			return nil, fmt.Errorf(
				"%w: tercero_id %d",
				ErrProveedorAmbiguo,
				terceroID,
			)
		}

		copia := registro
		encontrado = &copia
	}

	return encontrado, nil
}
