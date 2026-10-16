package dto

import (
	"encoding/json"
	"errors"
	"fmt"
)

func validarContratoRegistroCompuesto(
	data []byte,
	camposRaizPermitidos map[string]bool,
	contexto string,
) error {
	var raiz map[string]json.RawMessage
	if err := json.Unmarshal(data, &raiz); err != nil {
		return fmt.Errorf("%s debe ser un objeto JSON: %w", contexto, err)
	}
	if raiz == nil {
		return fmt.Errorf("%s debe ser un objeto JSON", contexto)
	}

	for campo := range raiz {
		if !camposRaizPermitidos[campo] {
			return fmt.Errorf("campo no permitido en %s: %s", contexto, campo)
		}
	}

	rawTercero, existe := raiz["Tercero"]
	if !existe {
		return nil
	}

	var tercero map[string]json.RawMessage
	if err := json.Unmarshal(rawTercero, &tercero); err != nil {
		return fmt.Errorf("Tercero debe ser un objeto JSON: %w", err)
	}
	if tercero == nil {
		return errors.New(`el objeto "Tercero" es obligatorio`)
	}

	permitidos := map[string]bool{
		"Id":                  true,
		"NombreCompleto":      true,
		"PrimerNombre":        true,
		"SegundoNombre":       true,
		"PrimerApellido":      true,
		"SegundoApellido":     true,
		"FechaNacimiento":     true,
		"LugarOrigen":         true,
		"CodigoTipoDocumento": true,
		"NumeroDocumento":     true,
		"FechaExpedicion":     true,
		"DigitoVerificacion":  true,
		"CiudadExpedicion":    true,
		"UsuarioWSO2":         true,
	}

	for campo := range tercero {
		if !permitidos[campo] {
			return fmt.Errorf("campo no permitido en Tercero: %s", campo)
		}
	}

	return nil
}

type terceroProveedorEntrada struct {
	Id                  int      `json:"Id"`
	NombreCompleto      string   `json:"NombreCompleto"`
	PrimerNombre        string   `json:"PrimerNombre"`
	SegundoNombre       string   `json:"SegundoNombre"`
	PrimerApellido      string   `json:"PrimerApellido"`
	SegundoApellido     string   `json:"SegundoApellido"`
	FechaNacimiento     *string  `json:"FechaNacimiento"`
	LugarOrigen         *int     `json:"LugarOrigen"`
	CodigoTipoDocumento string   `json:"CodigoTipoDocumento"`
	NumeroDocumento     string   `json:"NumeroDocumento"`
	FechaExpedicion     *string  `json:"FechaExpedicion"`
	DigitoVerificacion  *float64 `json:"DigitoVerificacion"`
	CiudadExpedicion    *int     `json:"CiudadExpedicion"`
	UsuarioWSO2         string   `json:"UsuarioWSO2"`
}

func (t terceroProveedorEntrada) aDTO() DatosTercerosDTO {
	return DatosTercerosDTO{
		TerceroID:           t.Id,
		NombreCompleto:      t.NombreCompleto,
		PrimerNombre:        t.PrimerNombre,
		SegundoNombre:       t.SegundoNombre,
		PrimerApellido:      t.PrimerApellido,
		SegundoApellido:     t.SegundoApellido,
		FechaNacimiento:     t.FechaNacimiento,
		LugarOrigen:         t.LugarOrigen,
		CodigoTipoDocumento: t.CodigoTipoDocumento,
		NumeroDocumento:     t.NumeroDocumento,
		FechaExpedicion:     t.FechaExpedicion,
		DigitoVerificacion:  t.DigitoVerificacion,
		CiudadExpedicion:    t.CiudadExpedicion,
		UsuarioWSO2:         t.UsuarioWSO2,
	}
}

func (r *RegistroPersonaNaturalRequest) UnmarshalJSON(data []byte) error {
	if err := validarContratoRegistroCompuesto(data, map[string]bool{
		"Tercero":                     true,
		"DescripcionPortafolio":       true,
		"PerfilDeclarado":             true,
		"ExperienciaLaboralMeses":     true,
		"ExperienciaProfesionalMeses": true,
	}, "registro de persona natural"); err != nil {
		return err
	}

	var p struct {
		Tercero               *terceroProveedorEntrada `json:"Tercero"`
		DescripcionPortafolio string                   `json:"DescripcionPortafolio"`
		PerfilDeclarado       *struct {
			Id int `json:"Id"`
		} `json:"PerfilDeclarado"`
		ExperienciaLaboralMeses     int `json:"ExperienciaLaboralMeses"`
		ExperienciaProfesionalMeses int `json:"ExperienciaProfesionalMeses"`
	}

	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	if p.Tercero == nil {
		return errors.New(`el objeto "Tercero" es obligatorio`)
	}

	var perfilID int
	if p.PerfilDeclarado != nil {
		perfilID = p.PerfilDeclarado.Id
	}

	*r = RegistroPersonaNaturalRequest{
		DatosTerceros: p.Tercero.aDTO(),
		Proveedor: ProveedorDTO{
			DescripcionPortafolio: p.DescripcionPortafolio,
		},
		ProveedorNatural: ProveedorNaturalDTO{
			PerfilDeclaradoID:           perfilID,
			ExperienciaLaboralMeses:     p.ExperienciaLaboralMeses,
			ExperienciaProfesionalMeses: p.ExperienciaProfesionalMeses,
		},
	}
	return nil
}

func (r *RegistroPersonaJuridicaRequest) UnmarshalJSON(data []byte) error {
	if err := validarContratoRegistroCompuesto(data, map[string]bool{
		"Tercero":                  true,
		"DescripcionPortafolio":    true,
		"NombreComercial":          true,
		"MatriculaMercantil":       true,
		"FechaConstitucion":        true,
		"FechaRenovacion":          true,
		"ReportaBeneficiosFinales": true,
		"CotizaBolsa":              true,
		"RequiereRevisorFiscal":    true,
	}, "registro de persona jurídica"); err != nil {
		return err
	}

	var p struct {
		Tercero                  *terceroProveedorEntrada `json:"Tercero"`
		DescripcionPortafolio    string                   `json:"DescripcionPortafolio"`
		NombreComercial          string                   `json:"NombreComercial"`
		MatriculaMercantil       int                      `json:"MatriculaMercantil"`
		FechaConstitucion        string                   `json:"FechaConstitucion"`
		FechaRenovacion          string                   `json:"FechaRenovacion"`
		ReportaBeneficiosFinales bool                     `json:"ReportaBeneficiosFinales"`
		CotizaBolsa              bool                     `json:"CotizaBolsa"`
		RequiereRevisorFiscal    bool                     `json:"RequiereRevisorFiscal"`
	}

	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	if p.Tercero == nil {
		return errors.New(`el objeto "Tercero" es obligatorio`)
	}

	*r = RegistroPersonaJuridicaRequest{
		DatosTerceros: p.Tercero.aDTO(),
		Proveedor: ProveedorDTO{
			DescripcionPortafolio: p.DescripcionPortafolio,
		},
		ProveedorJuridico: ProveedorJuridicoDTO{
			NombreComercial:          p.NombreComercial,
			MatriculaMercantil:       p.MatriculaMercantil,
			FechaConstitucion:        p.FechaConstitucion,
			FechaRenovacion:          p.FechaRenovacion,
			ReportaBeneficiosFinales: p.ReportaBeneficiosFinales,
			CotizaBolsa:              p.CotizaBolsa,
			RequiereRevisorFiscal:    p.RequiereRevisorFiscal,
		},
	}
	return nil
}
