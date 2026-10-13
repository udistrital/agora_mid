package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/udistrital/agora_mid/dto"
)

type ErrorRegistroParcial struct {
	TerceroID int
	Operacion string
	Err       error
}

func (e *ErrorRegistroParcial) Error() string {
	return fmt.Sprintf(
		"el tercero %d quedó creado, pero falló %s: %v; reintentar con tercero_id=%d",
		e.TerceroID, e.Operacion, e.Err, e.TerceroID,
	)
}

func (e *ErrorRegistroParcial) Unwrap() error { return e.Err }

type referenciaID struct {
	Id int `json:"Id"`
}

type terceroPayload struct {
	NombreCompleto      string       `json:"NombreCompleto"`
	PrimerNombre        string       `json:"PrimerNombre,omitempty"`
	SegundoNombre       string       `json:"SegundoNombre,omitempty"`
	PrimerApellido      string       `json:"PrimerApellido,omitempty"`
	SegundoApellido     string       `json:"SegundoApellido,omitempty"`
	FechaNacimiento     *string      `json:"FechaNacimiento,omitempty"`
	LugarOrigen         *int         `json:"LugarOrigen,omitempty"`
	Activo              bool         `json:"Activo"`
	TipoContribuyenteId referenciaID `json:"TipoContribuyenteId"`
	UsuarioWSO2         string       `json:"UsuarioWSO2,omitempty"`
}

type terceroRespuesta struct {
	Id int `json:"Id"`
}

type identificacionPayload struct {
	Numero             string       `json:"Numero"`
	TipoDocumentoId    referenciaID `json:"TipoDocumentoId"`
	TerceroId          referenciaID `json:"TerceroId"`
	DigitoVerificacion *float64     `json:"DigitoVerificacion,omitempty"`
	CiudadExpedicion   *int         `json:"CiudadExpedicion,omitempty"`
	FechaExpedicion    *string      `json:"FechaExpedicion,omitempty"`
	Activo             bool         `json:"Activo"`
}

// RegistrarOReutilizar busca un tercero por tipo y número de documento.
// Si no existe, lo crea y registra la identificación.
//
// codigoContribuyente debe ser P_NATURAL o P_JURIDICA.
// Si falla la identificación después de crear el tercero, devuelve
// ErrorRegistroParcial con el ID que debe conservarse para el reintento.
func (s TercerosService) RegistrarOReutilizar(
	ctx context.Context,
	datos dto.DatosTercerosDTO,
	codigoContribuyente string,
) (int, error) {
	base := strings.TrimRight(strings.TrimSpace(s.BaseURL), "/")
	numero := strings.TrimSpace(datos.NumeroDocumento)
	codigoDocumento := strings.TrimSpace(datos.CodigoTipoDocumento)

	if base == "" {
		return 0, errors.New("falta la URL base de Terceros CRUD")
	}
	if numero == "" || codigoDocumento == "" {
		return 0, errors.New("numero_documento y codigo_tipo_documento son obligatorios")
	}
	if strings.ContainsAny(numero, ",:") {
		return 0, errors.New("el número documental contiene caracteres incompatibles con el filtro de Terceros")
	}

	tipoDocumentoID, err := s.ResolverTipoDocumento(ctx, codigoDocumento)
	if err != nil {
		return 0, fmt.Errorf("resolviendo tipo documental %s: %w", codigoDocumento, err)
	}

	encontradoID, encontrado, err := s.BuscarPorDocumento(
		ctx, numero, codigoDocumento,
	)
	if err != nil {
		return 0, err
	}

	// Reintento de una operación parcialmente completada.
	if datos.TerceroID > 0 {
		if encontrado && encontradoID != datos.TerceroID {
			return 0, fmt.Errorf(
				"el documento ya está asociado al tercero %d, no al tercero_id recibido %d",
				encontradoID, datos.TerceroID,
			)
		}
		if encontrado {
			return encontradoID, nil
		}

		// Verificar que el tercero previo existe antes de anexarle el documento.
		var previo terceroRespuesta
		if _, err := s.Client.Get(
			ctx, fmt.Sprintf("%s/v1/tercero/%d", base, datos.TerceroID), &previo,
		); err != nil {
			return 0, fmt.Errorf(
				"no se pudo validar el tercero_id %d para reintentar: %w",
				datos.TerceroID, err,
			)
		}
		if previo.Id != datos.TerceroID {
			return 0, fmt.Errorf("Terceros CRUD no confirmó el tercero_id %d", datos.TerceroID)
		}

		if err := s.crearIdentificacion(
			ctx, base, datos, numero, tipoDocumentoID, datos.TerceroID,
		); err != nil {
			return 0, &ErrorRegistroParcial{
				TerceroID: datos.TerceroID,
				Operacion: "registrar la identificación",
				Err:       err,
			}
		}
		return datos.TerceroID, nil
	}

	// La persona ya existe: se reutiliza sin sobrescribir su identidad.
	if encontrado {
		return encontradoID, nil
	}

	tipoContribuyenteID, err := s.ResolverTipoContribuyente(
		ctx, codigoContribuyente,
	)
	if err != nil {
		return 0, fmt.Errorf(
			"resolviendo tipo de contribuyente %s: %w",
			codigoContribuyente, err,
		)
	}

	nombreCompleto := strings.TrimSpace(datos.NombreCompleto)
	if nombreCompleto == "" {
		nombreCompleto = strings.Join(
			filtrarVacios(
				datos.PrimerNombre,
				datos.SegundoNombre,
				datos.PrimerApellido,
				datos.SegundoApellido,
			),
			" ",
		)
	}
	if nombreCompleto == "" {
		return 0, errors.New("terceros.nombre_completo es obligatorio si no se proporcionan nombres")
	}

	payload := terceroPayload{
		NombreCompleto:      nombreCompleto,
		PrimerNombre:        datos.PrimerNombre,
		SegundoNombre:       datos.SegundoNombre,
		PrimerApellido:      datos.PrimerApellido,
		SegundoApellido:     datos.SegundoApellido,
		FechaNacimiento:     datos.FechaNacimiento,
		LugarOrigen:         datos.LugarOrigen,
		Activo:              true,
		TipoContribuyenteId: referenciaID{Id: tipoContribuyenteID},
		UsuarioWSO2:         datos.UsuarioWSO2,
	}

	var creado terceroRespuesta
	if _, err := s.Client.Post(
		ctx, base+"/v1/tercero/", payload, &creado,
	); err != nil {
		return 0, fmt.Errorf("creando tercero en Terceros CRUD: %w", err)
	}
	if creado.Id <= 0 {
		return 0, errors.New("Terceros CRUD no devolvió un Id válido al crear el tercero")
	}

	if err := s.crearIdentificacion(
		ctx, base, datos, numero, tipoDocumentoID, creado.Id,
	); err != nil {
		return creado.Id, &ErrorRegistroParcial{
			TerceroID: creado.Id,
			Operacion: "registrar la identificación",
			Err:       err,
		}
	}

	return creado.Id, nil
}

func (s TercerosService) crearIdentificacion(
	ctx context.Context,
	base string,
	datos dto.DatosTercerosDTO,
	numero string,
	tipoDocumentoID int,
	terceroID int,
) error {
	payload := identificacionPayload{
		Numero:             numero,
		TipoDocumentoId:    referenciaID{Id: tipoDocumentoID},
		TerceroId:          referenciaID{Id: terceroID},
		DigitoVerificacion: datos.DigitoVerificacion,
		CiudadExpedicion:   datos.CiudadExpedicion,
		FechaExpedicion:    datos.FechaExpedicion,
		Activo:             true,
	}

	var creada map[string]interface{}
	if _, err := s.Client.Post(
		ctx, base+"/v1/datos_identificacion/", payload, &creada,
	); err != nil {
		return fmt.Errorf("creando datos_identificacion: %w", err)
	}
	return nil
}

func filtrarVacios(valores ...string) []string {
	resultado := make([]string, 0, len(valores))
	for _, valor := range valores {
		valor = strings.TrimSpace(valor)
		if valor != "" {
			resultado = append(resultado, valor)
		}
	}
	return resultado
}
