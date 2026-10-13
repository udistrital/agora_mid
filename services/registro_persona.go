package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/udistrital/agora_mid/dto"
)

// RegistroPersonaService coordina el registro sin convertirse en dueño de los datos.
type RegistroPersonaService struct {
	Terceros        TercerosService
	GestionPersonas GestionPersonasService
}

// RegistroPersonaResultado permite recuperar los identificadores si una etapa falla.
type RegistroPersonaResultado struct {
	TerceroID         int                      `json:"tercero_id"`
	ProveedorID       int                      `json:"proveedor_id,omitempty"`
	ProveedorNatural  *ProveedorNaturalRecord  `json:"proveedor_natural,omitempty"`
	ProveedorJuridico *ProveedorJuridicoRecord `json:"proveedor_juridico,omitempty"`
}

// RegistrarPersonaNatural registra o reutiliza el tercero y el proveedor natural.
func (s RegistroPersonaService) RegistrarPersonaNatural(
	ctx context.Context,
	req dto.RegistroPersonaNaturalRequest,
) (RegistroPersonaResultado, error) {
	resultado, err := s.registrarBase(
		ctx,
		req.DatosTerceros,
		req.Proveedor,
		"P_NATURAL",
	)
	if err != nil {
		return resultado, err
	}

	natural, err := s.GestionPersonas.CrearOReutilizarProveedorNatural(
		ctx,
		resultado.ProveedorID,
		req.ProveedorNatural,
	)
	if err != nil {
		return resultado, fmt.Errorf(
			"registro específico natural; tercero_id=%d, proveedor_id=%d: %w",
			resultado.TerceroID,
			resultado.ProveedorID,
			err,
		)
	}

	resultado.ProveedorNatural = &natural
	return resultado, nil
}

// RegistrarPersonaJuridica registra o reutiliza el tercero y el proveedor jurídico.
func (s RegistroPersonaService) RegistrarPersonaJuridica(
	ctx context.Context,
	req dto.RegistroPersonaJuridicaRequest,
) (RegistroPersonaResultado, error) {
	resultado, err := s.registrarBase(
		ctx,
		req.DatosTerceros,
		req.Proveedor,
		"P_JURIDICA",
	)
	if err != nil {
		return resultado, err
	}

	juridico, err := s.GestionPersonas.CrearOReutilizarProveedorJuridico(
		ctx,
		resultado.ProveedorID,
		req.ProveedorJuridico,
	)
	if err != nil {
		return resultado, fmt.Errorf(
			"registro específico jurídico; tercero_id=%d, proveedor_id=%d: %w",
			resultado.TerceroID,
			resultado.ProveedorID,
			err,
		)
	}

	resultado.ProveedorJuridico = &juridico
	return resultado, nil
}

// registrarBase coordina las etapas comunes y preserva los IDs para reintentar.
func (s RegistroPersonaService) registrarBase(
	ctx context.Context,
	datos dto.DatosTercerosDTO,
	proveedorDTO dto.ProveedorDTO,
	codigoContribuyente string,
) (RegistroPersonaResultado, error) {
	resultado := RegistroPersonaResultado{
		TerceroID: datos.TerceroID,
	}

	terceroID, err := s.Terceros.RegistrarOReutilizar(
		ctx,
		datos,
		codigoContribuyente,
	)
	if err != nil {
		var parcial *ErrorRegistroParcial
		if errors.As(err, &parcial) && parcial.TerceroID > 0 {
			resultado.TerceroID = parcial.TerceroID
		}

		return resultado, fmt.Errorf("registrando o reutilizando tercero: %w", err)
	}

	resultado.TerceroID = terceroID

	// El tipo de registro se resuelve dinámicamente; nunca se toma un ID fijo
	// enviado por el cliente.
	tipoRegistroID, err := s.Terceros.ResolverTipoContribuyente(
		ctx,
		codigoContribuyente,
	)
	if err != nil {
		return resultado, fmt.Errorf(
			"resolviendo tipo de registro %s para tercero_id=%d: %w",
			codigoContribuyente,
			terceroID,
			err,
		)
	}

	proveedor, err := s.GestionPersonas.CrearOReutilizarProveedor(
		ctx,
		terceroID,
		tipoRegistroID,
		proveedorDTO.DescripcionPortafolio,
	)
	if err != nil {
		return resultado, fmt.Errorf(
			"registrando proveedor general para tercero_id=%d: %w",
			terceroID,
			err,
		)
	}

	resultado.ProveedorID = proveedor.Id
	return resultado, nil
}
