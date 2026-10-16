package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var ErrTipoProveedorIncompatible = errors.New(
	"el proveedor existente tiene un tipo de registro diferente",
)

type proveedorCreatePayload struct {
	TerceroId             int    `json:"TerceroId"`
	TipoRegistroId        int    `json:"TipoRegistroId"`
	DescripcionPortafolio string `json:"DescripcionPortafolio,omitempty"`
	Activo                bool   `json:"Activo"`
}

type proveedorCreateResponse struct {
	Data ProveedorRecord `json:"Data"`
}

// CrearOReutilizarProveedor crea el proveedor general o reutiliza el existente.
// No crea un segundo proveedor para el mismo tercero.
func (s GestionPersonasService) CrearOReutilizarProveedor(
	ctx context.Context,
	terceroID int,
	tipoRegistroID int,
	descripcion string,
) (ProveedorRecord, error) {
	base := strings.TrimRight(strings.TrimSpace(s.BaseURL), "/")
	if base == "" {
		return ProveedorRecord{}, errors.New(
			"falta la URL base de gestión de personas CRUD",
		)
	}
	if terceroID <= 0 || tipoRegistroID <= 0 {
		return ProveedorRecord{}, errors.New(
			"tercero_id y tipo_registro_id deben ser mayores que cero",
		)
	}

	existente, err := s.BuscarProveedorPorTercero(ctx, terceroID)
	if err != nil {
		return ProveedorRecord{}, err
	}
	if existente != nil {
		return validarProveedorExistente(*existente, terceroID, tipoRegistroID)
	}

	payload := proveedorCreatePayload{
		TerceroId:             terceroID,
		TipoRegistroId:        tipoRegistroID,
		DescripcionPortafolio: strings.TrimSpace(descripcion),
		Activo:                true,
	}

	var respuesta proveedorCreateResponse
	_, postErr := s.Client.Post(
		ctx, base+"/v1/proveedor/", payload, &respuesta,
	)
	if postErr != nil {
		// El POST pudo guardarse aunque la respuesta se haya perdido.
		recuperado, lookupErr := s.BuscarProveedorPorTercero(ctx, terceroID)
		if lookupErr != nil {
			return ProveedorRecord{}, fmt.Errorf(
				"falló el POST del proveedor y también su recuperación: POST: %v; consulta: %w",
				postErr, lookupErr,
			)
		}
		if recuperado != nil {
			return validarProveedorExistente(*recuperado, terceroID, tipoRegistroID)
		}
		return ProveedorRecord{}, fmt.Errorf(
			"creando proveedor general en ÁGORA CRUD: %w", postErr,
		)
	}

	creado := respuesta.Data
	if creado.Id <= 0 {
		recuperado, lookupErr := s.BuscarProveedorPorTercero(ctx, terceroID)
		if lookupErr == nil && recuperado != nil {
			return validarProveedorExistente(*recuperado, terceroID, tipoRegistroID)
		}
		return ProveedorRecord{}, errors.New(
			"ÁGORA CRUD no devolvió un ID válido para el proveedor",
		)
	}

	return validarProveedorExistente(creado, terceroID, tipoRegistroID)
}

func validarProveedorExistente(
	proveedor ProveedorRecord,
	terceroID int,
	tipoRegistroID int,
) (ProveedorRecord, error) {
	if proveedor.Id <= 0 || proveedor.TerceroId != terceroID {
		return ProveedorRecord{}, errors.New(
			"el proveedor devuelto no corresponde al tercero solicitado",
		)
	}
	if proveedor.TipoRegistroId != tipoRegistroID {
		return ProveedorRecord{}, fmt.Errorf(
			"%w: proveedor_id %d",
			ErrTipoProveedorIncompatible, proveedor.Id,
		)
	}
	if !proveedor.Activo {
		return ProveedorRecord{}, fmt.Errorf(
			"el proveedor %d está inactivo; no se creará otro", proveedor.Id,
		)
	}
	return proveedor, nil
}
