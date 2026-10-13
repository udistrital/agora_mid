package services

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/udistrital/agora_mid/dto"
)

var ErrRegistroEspecificoIncompatible = errors.New(
	"el registro específico existente no coincide con los datos solicitados",
)

type proveedorNaturalPayload struct {
	Id                          int          `json:"Id"`
	PerfilDeclarado             referenciaID `json:"PerfilDeclarado"`
	ExperienciaLaboralMeses     int          `json:"ExperienciaLaboralMeses"`
	ExperienciaProfesionalMeses int          `json:"ExperienciaProfesionalMeses"`
	Activo                      bool         `json:"Activo"`
	FechaCreacion               time.Time    `json:"FechaCreacion"`
	FechaModificacion           time.Time    `json:"FechaModificacion"`
}

type ProveedorNaturalRecord struct {
	Id              int `json:"Id"`
	PerfilDeclarado struct {
		Id int `json:"Id"`
	} `json:"PerfilDeclarado"`
	ExperienciaLaboralMeses     int  `json:"ExperienciaLaboralMeses"`
	ExperienciaProfesionalMeses int  `json:"ExperienciaProfesionalMeses"`
	Activo                      bool `json:"Activo"`
}

type proveedorNaturalResponse struct {
	Data ProveedorNaturalRecord `json:"Data"`
}

type proveedorJuridicoPayload struct {
	Id                       int       `json:"Id"`
	NombreComercial          string    `json:"NombreComercial"`
	MatriculaMercantil       int       `json:"MatriculaMercantil"`
	FechaConstitucion        time.Time `json:"FechaConstitucion"`
	FechaRenovacion          time.Time `json:"FechaRenovacion"`
	ReportaBeneficiosFinales bool      `json:"ReportaBeneficiosFinales"`
	CotizaBolsa              bool      `json:"CotizaBolsa"`
	RequiereRevisorFiscal    bool      `json:"RequiereRevisorFiscal"`
	Activo                   bool      `json:"Activo"`
	FechaCreacion            time.Time `json:"FechaCreacion"`
	FechaModificacion        time.Time `json:"FechaModificacion"`
}

type ProveedorJuridicoRecord struct {
	Id                       int       `json:"Id"`
	NombreComercial          string    `json:"NombreComercial"`
	MatriculaMercantil       int       `json:"MatriculaMercantil"`
	FechaConstitucion        time.Time `json:"FechaConstitucion"`
	FechaRenovacion          time.Time `json:"FechaRenovacion"`
	ReportaBeneficiosFinales bool      `json:"ReportaBeneficiosFinales"`
	CotizaBolsa              bool      `json:"CotizaBolsa"`
	RequiereRevisorFiscal    bool      `json:"RequiereRevisorFiscal"`
	Activo                   bool      `json:"Activo"`
}

type proveedorJuridicoResponse struct {
	Data ProveedorJuridicoRecord `json:"Data"`
}

// CrearOReutilizarProveedorNatural usa el mismo ID del proveedor general.
// Si ya existe, lo reutiliza solamente si coincide con los datos recibidos.
func (s GestionPersonasService) CrearOReutilizarProveedorNatural(
	ctx context.Context,
	proveedorID int,
	datos dto.ProveedorNaturalDTO,
) (ProveedorNaturalRecord, error) {
	if proveedorID <= 0 || datos.PerfilDeclaradoID <= 0 {
		return ProveedorNaturalRecord{}, errors.New(
			"proveedor_id y perfil_declarado_id deben ser mayores que cero",
		)
	}
	if datos.ExperienciaLaboralMeses < 0 || datos.ExperienciaProfesionalMeses < 0 {
		return ProveedorNaturalRecord{}, errors.New("los meses de experiencia no pueden ser negativos")
	}

	base, err := s.baseURL()
	if err != nil {
		return ProveedorNaturalRecord{}, err
	}

	existente, encontrado, err := s.obtenerProveedorNatural(ctx, base, proveedorID)
	if err != nil {
		return ProveedorNaturalRecord{}, err
	}
	if encontrado {
		return validarNatural(existente, proveedorID, datos)
	}

	ahora := time.Now()
	payload := proveedorNaturalPayload{
		Id:                          proveedorID,
		PerfilDeclarado:             referenciaID{Id: datos.PerfilDeclaradoID},
		ExperienciaLaboralMeses:     datos.ExperienciaLaboralMeses,
		ExperienciaProfesionalMeses: datos.ExperienciaProfesionalMeses,
		Activo:                      true,
		FechaCreacion:               ahora,
		FechaModificacion:           ahora,
	}

	var respuesta proveedorNaturalResponse
	_, postErr := s.Client.Post(
		ctx, base+"/v1/proveedor_natural/", payload, &respuesta,
	)
	if postErr == nil && respuesta.Data.Id == proveedorID {
		return validarNatural(respuesta.Data, proveedorID, datos)
	}

	// El POST pudo haberse guardado aunque se haya perdido su respuesta.
	recuperado, existe, lookupErr := s.obtenerProveedorNatural(ctx, base, proveedorID)
	if lookupErr != nil {
		if postErr != nil {
			return ProveedorNaturalRecord{}, fmt.Errorf(
				"falló el POST natural y también la recuperación: POST: %v; GET: %w",
				postErr, lookupErr,
			)
		}
		return ProveedorNaturalRecord{}, lookupErr
	}
	if existe {
		return validarNatural(recuperado, proveedorID, datos)
	}
	if postErr != nil {
		return ProveedorNaturalRecord{}, fmt.Errorf("creando proveedor natural: %w", postErr)
	}
	return ProveedorNaturalRecord{}, errors.New(
		"el CRUD no devolvió un proveedor natural válido y no pudo recuperarse por ID",
	)
}

// CrearOReutilizarProveedorJuridico usa el mismo ID del proveedor general.
func (s GestionPersonasService) CrearOReutilizarProveedorJuridico(
	ctx context.Context,
	proveedorID int,
	datos dto.ProveedorJuridicoDTO,
) (ProveedorJuridicoRecord, error) {
	if proveedorID <= 0 {
		return ProveedorJuridicoRecord{}, errors.New("proveedor_id debe ser mayor que cero")
	}

	nombre := strings.TrimSpace(datos.NombreComercial)
	if nombre == "" {
		return ProveedorJuridicoRecord{}, errors.New("nombre_comercial es obligatorio")
	}
	if datos.MatriculaMercantil <= 0 {
		return ProveedorJuridicoRecord{}, errors.New("matricula_mercantil debe ser mayor que cero")
	}

	fechaConstitucion, err := parseFecha(datos.FechaConstitucion)
	if err != nil {
		return ProveedorJuridicoRecord{}, fmt.Errorf("fecha_constitucion: %w", err)
	}
	fechaRenovacion, err := parseFecha(datos.FechaRenovacion)
	if err != nil {
		return ProveedorJuridicoRecord{}, fmt.Errorf("fecha_renovacion: %w", err)
	}

	base, err := s.baseURL()
	if err != nil {
		return ProveedorJuridicoRecord{}, err
	}

	existente, encontrado, err := s.obtenerProveedorJuridico(ctx, base, proveedorID)
	if err != nil {
		return ProveedorJuridicoRecord{}, err
	}
	if encontrado {
		return validarJuridico(existente, proveedorID, datos, fechaConstitucion, fechaRenovacion)
	}

	ahora := time.Now()
	payload := proveedorJuridicoPayload{
		Id:                       proveedorID,
		NombreComercial:          nombre,
		MatriculaMercantil:       datos.MatriculaMercantil,
		FechaConstitucion:        fechaConstitucion,
		FechaRenovacion:          fechaRenovacion,
		ReportaBeneficiosFinales: datos.ReportaBeneficiosFinales,
		CotizaBolsa:              datos.CotizaBolsa,
		RequiereRevisorFiscal:    datos.RequiereRevisorFiscal,
		Activo:                   true,
		FechaCreacion:            ahora,
		FechaModificacion:        ahora,
	}

	var respuesta proveedorJuridicoResponse
	_, postErr := s.Client.Post(
		ctx, base+"/v1/proveedor_juridico/", payload, &respuesta,
	)
	if postErr == nil && respuesta.Data.Id == proveedorID {
		return validarJuridico(
			respuesta.Data, proveedorID, datos, fechaConstitucion, fechaRenovacion,
		)
	}

	recuperado, existe, lookupErr := s.obtenerProveedorJuridico(ctx, base, proveedorID)
	if lookupErr != nil {
		if postErr != nil {
			return ProveedorJuridicoRecord{}, fmt.Errorf(
				"falló el POST jurídico y también la recuperación: POST: %v; GET: %w",
				postErr, lookupErr,
			)
		}
		return ProveedorJuridicoRecord{}, lookupErr
	}
	if existe {
		return validarJuridico(
			recuperado, proveedorID, datos, fechaConstitucion, fechaRenovacion,
		)
	}
	if postErr != nil {
		return ProveedorJuridicoRecord{}, fmt.Errorf("creando proveedor jurídico: %w", postErr)
	}
	return ProveedorJuridicoRecord{}, errors.New(
		"el CRUD no devolvió un proveedor jurídico válido y no pudo recuperarse por ID",
	)
}

func (s GestionPersonasService) baseURL() (string, error) {
	base := strings.TrimRight(strings.TrimSpace(s.BaseURL), "/")
	if base == "" {
		return "", errors.New("falta la URL base de gestión de personas CRUD")
	}
	return base, nil
}

func (s GestionPersonasService) obtenerProveedorNatural(
	ctx context.Context,
	base string,
	id int,
) (ProveedorNaturalRecord, bool, error) {
	var respuesta proveedorNaturalResponse
	status, err := s.Client.Get(
		ctx, fmt.Sprintf("%s/v1/proveedor_natural/%d", base, id), &respuesta,
	)
	if err != nil {
		if status == http.StatusNotFound {
			return ProveedorNaturalRecord{}, false, nil
		}
		return ProveedorNaturalRecord{}, false, fmt.Errorf("consultando proveedor natural: %w", err)
	}
	if respuesta.Data.Id != id {
		return ProveedorNaturalRecord{}, false, errors.New(
			"GET de proveedor natural respondió sin un ID coincidente",
		)
	}
	return respuesta.Data, true, nil
}

func (s GestionPersonasService) obtenerProveedorJuridico(
	ctx context.Context,
	base string,
	id int,
) (ProveedorJuridicoRecord, bool, error) {
	var respuesta proveedorJuridicoResponse
	status, err := s.Client.Get(
		ctx, fmt.Sprintf("%s/v1/proveedor_juridico/%d", base, id), &respuesta,
	)
	if err != nil {
		if status == http.StatusNotFound {
			return ProveedorJuridicoRecord{}, false, nil
		}
		return ProveedorJuridicoRecord{}, false, fmt.Errorf("consultando proveedor jurídico: %w", err)
	}
	if respuesta.Data.Id != id {
		return ProveedorJuridicoRecord{}, false, errors.New(
			"GET de proveedor jurídico respondió sin un ID coincidente",
		)
	}
	return respuesta.Data, true, nil
}

func validarNatural(
	existente ProveedorNaturalRecord,
	proveedorID int,
	datos dto.ProveedorNaturalDTO,
) (ProveedorNaturalRecord, error) {
	if existente.Id != proveedorID || !existente.Activo ||
		existente.PerfilDeclarado.Id != datos.PerfilDeclaradoID ||
		existente.ExperienciaLaboralMeses != datos.ExperienciaLaboralMeses ||
		existente.ExperienciaProfesionalMeses != datos.ExperienciaProfesionalMeses {
		return ProveedorNaturalRecord{}, fmt.Errorf(
			"%w: proveedor natural %d",
			ErrRegistroEspecificoIncompatible, proveedorID,
		)
	}
	return existente, nil
}

func validarJuridico(
	existente ProveedorJuridicoRecord,
	proveedorID int,
	datos dto.ProveedorJuridicoDTO,
	fechaConstitucion time.Time,
	fechaRenovacion time.Time,
) (ProveedorJuridicoRecord, error) {
	mismoDia := func(a, b time.Time) bool {
		return a.Format("2006-01-02") == b.Format("2006-01-02")
	}

	if existente.Id != proveedorID || !existente.Activo ||
		existente.NombreComercial != strings.TrimSpace(datos.NombreComercial) ||
		existente.MatriculaMercantil != datos.MatriculaMercantil ||
		!mismoDia(existente.FechaConstitucion, fechaConstitucion) ||
		!mismoDia(existente.FechaRenovacion, fechaRenovacion) ||
		existente.ReportaBeneficiosFinales != datos.ReportaBeneficiosFinales ||
		existente.CotizaBolsa != datos.CotizaBolsa ||
		existente.RequiereRevisorFiscal != datos.RequiereRevisorFiscal {
		return ProveedorJuridicoRecord{}, fmt.Errorf(
			"%w: proveedor jurídico %d",
			ErrRegistroEspecificoIncompatible, proveedorID,
		)
	}
	return existente, nil
}

func parseFecha(valor string) (time.Time, error) {
	valor = strings.TrimSpace(valor)
	if valor == "" {
		return time.Time{}, errors.New("el valor de fecha es obligatorio")
	}
	if fecha, err := time.Parse("2006-01-02", valor); err == nil {
		return fecha, nil
	}
	if fecha, err := time.Parse(time.RFC3339, valor); err == nil {
		return time.Date(fecha.Year(), fecha.Month(), fecha.Day(), 0, 0, 0, 0, time.UTC), nil
	}
	return time.Time{}, fmt.Errorf("formato inválido %q; use YYYY-MM-DD o RFC3339", valor)
}
