package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/udistrital/agora_mid/clients"
	"github.com/udistrital/agora_mid/config"
	"github.com/udistrital/agora_mid/helper"
	"github.com/udistrital/agora_mid/models"
	"github.com/udistrital/agora_mid/responses"
)

// Errores de negocio del dominio 1 que el controlador mapea a códigos HTTP
// concretos.
var (
	ErrTipoContribuyenteInvalido      = errors.New("TipoContribuyenteId no corresponde a la variante natural")
	ErrTipoContribuyenteConflicto     = errors.New("el tercero existente tiene un TipoContribuyenteId distinto")
	ErrTipoContribuyenteNoConfigurado = errors.New("no está configurado el tipo de contribuyente P_NATURAL")
	ErrTerceroNoEncontrado            = errors.New("no existe el tercero indicado")
	ErrIdentificacionDeOtroTercero    = errors.New("el documento ya pertenece a otro tercero")
)

// TerceroWriteResult es el resultado del dominio 1: el subárbol materializado,
// los Id definitivos y la clasificación de entidades para la respuesta.
type TerceroWriteResult struct {
	Data                  helper.WriteResult
	TerceroId             int64
	DatosIdentificacionId int64
	Creados               []models.EntidadId
	Reutilizados          []models.EntidadId
	Actualizados          []models.EntidadId
	Avisos                []responses.RelationError
	Parcial               bool
}

// TerceroWriteService orquesta la escritura del dominio 1 sobre el motor
// declarativo del proyecto base. La raíz es Tercero y el único hijo escribible
// es DatosIdentificacion. No conoce Proveedor ni gestion_personas_crud.
type TerceroWriteService struct {
	Config                   config.ServiceConfig
	Registry                 helper.WriteResolverRegistry
	Client                   clients.HTTPClient
	URLs                     config.ServiceURLs
	TipoContribuyenteNatural int
}

// NewTerceroWriteService construye el servicio con la configuración del servicio
// tercero y el tipo de contribuyente P_NATURAL resuelto al arranque.
func NewTerceroWriteService(
	client clients.HTTPClient,
	urls config.ServiceURLs,
	cfg config.ServiceConfig,
	tipoContribuyenteNatural int,
) TerceroWriteService {
	raiz, hijos := NewTerceroWriteResolvers(client, urls)
	registry := helper.NewWriteResolverRegistry(raiz, hijos)
	return TerceroWriteService{
		Config:                   cfg,
		Registry:                 registry,
		Client:                   client,
		URLs:                     urls,
		TipoContribuyenteNatural: tipoContribuyenteNatural,
	}
}

// EscribirTercero registra o actualiza el tercero y su identificación.
//
// verb es helper.WriteVerbCreate para POST o helper.WriteVerbUpdate para PUT.
// terceroID es el Id de la ruta; en POST debe ser cero.
func (s TerceroWriteService) EscribirTercero(
	ctx context.Context,
	in models.RegistroTerceroNatural,
	terceroID int64,
	verb string,
) (TerceroWriteResult, error) {
	if s.TipoContribuyenteNatural <= 0 {
		return TerceroWriteResult{}, ErrTipoContribuyenteNoConfigurado
	}
	if in.DatosBasicos.TipoContribuyenteId <= 0 {
		return TerceroWriteResult{}, ErrTipoContribuyenteInvalido
	}
	if in.DatosBasicos.TipoContribuyenteId != int64(s.TipoContribuyenteNatural) {
		return TerceroWriteResult{}, ErrTipoContribuyenteInvalido
	}

	esActualizacion := verb == helper.WriteVerbUpdate
	if esActualizacion && terceroID <= 0 {
		return TerceroWriteResult{}, ErrTerceroNoEncontrado
	}

	terceroResuelto, identificacionID, avisos, err := s.resolverIdentificacion(ctx, in, terceroID, esActualizacion)
	if err != nil {
		return TerceroWriteResult{}, err
	}
	if terceroID == 0 {
		terceroID = terceroResuelto
	}

	// Si ya hay identificación, el tercero existe o se reutiliza. Se lee para
	// validar el TipoContribuyenteId y para conservar los nombres.
	if terceroID > 0 {
		existente, err := s.consultarTercero(ctx, terceroID)
		if err != nil {
			return TerceroWriteResult{}, err
		}
		tipoExistente := idDeRelacion(existente["TipoContribuyenteId"])
		if tipoExistente > 0 && tipoExistente != int64(s.TipoContribuyenteNatural) {
			return TerceroWriteResult{}, ErrTipoContribuyenteConflicto
		}
	}

	body := s.construirCuerpo(in, terceroID, identificacionID, !esActualizacion && terceroID == 0)
	result, err := helper.Write(ctx, body, s.Config, s.Registry, helper.WriteOptions{Verb: verb})
	if err != nil {
		return TerceroWriteResult{}, err
	}

	out := TerceroWriteResult{
		Data:    result,
		Avisos:  append(avisos, result.Warnings...),
		Parcial: len(result.Warnings) > 0,
	}
	out.TerceroId = extraerTerceroID(result.Data, terceroID)
	out.DatosIdentificacionId = extraerIdentificacionID(result.Data, identificacionID)

	// Clasificación para Resultado. Es un resumen de la intención del flujo, no
	// un diff campo a campo: POST reutiliza el tercero y actualiza la
	// identificación; PUT actualiza el tercero; el alta crea ambos.
	switch {
	case !esActualizacion && terceroID == 0:
		out.Creados = append(out.Creados, models.EntidadId{Entidad: "Tercero", Id: out.TerceroId})
	case esActualizacion:
		out.Actualizados = append(out.Actualizados, models.EntidadId{Entidad: "Tercero", Id: out.TerceroId})
	default:
		out.Reutilizados = append(out.Reutilizados, models.EntidadId{Entidad: "Tercero", Id: out.TerceroId})
	}

	if identificacionID == 0 {
		out.Creados = append(out.Creados, models.EntidadId{Entidad: "DatosIdentificacion", Id: out.DatosIdentificacionId})
	} else {
		out.Actualizados = append(out.Actualizados, models.EntidadId{Entidad: "DatosIdentificacion", Id: out.DatosIdentificacionId})
	}

	return out, nil
}

// resolverIdentificacion busca la identificación que corresponde al documento y,
// en actualización, valida que no pertenezca a otro tercero. Devuelve el
// TerceroId resuelto (0 en alta nueva), el Id de la identificación seleccionada y
// los avisos de ambigüedad.
func (s TerceroWriteService) resolverIdentificacion(
	ctx context.Context,
	in models.RegistroTerceroNatural,
	terceroID int64,
	esActualizacion bool,
) (int64, int64, []responses.RelationError, error) {
	identificacionID := int64(0)
	terceroResuelto := terceroID
	var avisos []responses.RelationError

	// En actualización, la identificación activa del tercero es la fuente para el
	// PUT; si no existe, se crea.
	if esActualizacion {
		activas, err := s.buscarIdentificaciones(ctx, fmt.Sprintf("TerceroId:%d,Activo:true", terceroID))
		if err != nil {
			return 0, 0, nil, err
		}
		if len(activas) > 0 {
			identificacionID = idDeMapa(activas[0])
			if len(activas) > 1 {
				avisos = append(avisos, avisoAmbiguedad("DatosIdentificacion",
					"Se encontraron varias identificaciones activas del tercero; se usó la primera"))
			}
		}
	}

	// El documento nuevo no puede pertenecer a otro tercero. En POST, si ya
	// existe, se reutiliza el tercero y la identificación.
	documento, err := s.buscarIdentificaciones(ctx, fmt.Sprintf(
		"Numero:%s,TipoDocumentoId:%d,Activo:true",
		in.DatosIdentificacion.Numero, in.DatosIdentificacion.TipoDocumentoId,
	))
	if err != nil {
		return 0, 0, nil, err
	}
	if len(documento) > 0 {
		terceroDocumento := idDeRelacion(documento[0]["TerceroId"])
		if terceroDocumento <= 0 {
			return 0, 0, nil, errors.New("la identificación activa no trae un TerceroId válido")
		}
		if esActualizacion && terceroDocumento != terceroID {
			return 0, 0, nil, ErrIdentificacionDeOtroTercero
		}
		if !esActualizacion {
			terceroResuelto = terceroDocumento
			identificacionID = idDeMapa(documento[0])
		}
		if len(documento) > 1 {
			avisos = append(avisos, avisoAmbiguedad("DatosIdentificacion",
				"Se encontraron varias identificaciones activas para el documento; se usó la primera"))
		}
	}

	if esActualizacion && identificacionID == 0 && len(documento) > 0 {
		identificacionID = idDeMapa(documento[0])
	}

	return terceroResuelto, identificacionID, avisos, nil
}

// construirCuerpo arma el cuerpo anidado que espera el motor. En reutilización
// no se envían nombres ni LugarOrigen para que el PUT no los sobrescriba.
func (s TerceroWriteService) construirCuerpo(
	in models.RegistroTerceroNatural,
	terceroID int64,
	identificacionID int64,
	esAlta bool,
) map[string]any {
	basicos := map[string]any{
		"TipoContribuyenteId": in.DatosBasicos.TipoContribuyenteId,
	}
	if esAlta {
		basicos["PrimerNombre"] = in.DatosBasicos.PrimerNombre
		basicos["SegundoNombre"] = in.DatosBasicos.SegundoNombre
		basicos["PrimerApellido"] = in.DatosBasicos.PrimerApellido
		basicos["SegundoApellido"] = in.DatosBasicos.SegundoApellido
		basicos["NombreCompleto"] = nombreCompleto(
			in.DatosBasicos.PrimerNombre,
			in.DatosBasicos.SegundoNombre,
			in.DatosBasicos.PrimerApellido,
			in.DatosBasicos.SegundoApellido,
		)
		basicos["Activo"] = true
	}
	if in.DatosBasicos.FechaNacimiento != nil {
		basicos["FechaNacimiento"] = *in.DatosBasicos.FechaNacimiento
	}
	if esAlta && in.DatosBasicos.LugarOrigen != nil {
		basicos["LugarOrigen"] = *in.DatosBasicos.LugarOrigen
	}

	identificacion := map[string]any{
		"Numero":          in.DatosIdentificacion.Numero,
		"TipoDocumentoId": in.DatosIdentificacion.TipoDocumentoId,
	}
	if identificacionID == 0 {
		identificacion["Activo"] = true
	}
	if in.DatosIdentificacion.CiudadExpedicion != nil {
		identificacion["CiudadExpedicion"] = *in.DatosIdentificacion.CiudadExpedicion
	}
	if in.DatosIdentificacion.FechaExpedicion != nil {
		identificacion["FechaExpedicion"] = *in.DatosIdentificacion.FechaExpedicion
	}
	if in.DatosIdentificacion.DigitoVerificacion != nil {
		identificacion["DigitoVerificacion"] = *in.DatosIdentificacion.DigitoVerificacion
	}
	if identificacionID > 0 {
		identificacion["Id"] = identificacionID
	}

	if terceroID > 0 {
		basicos["Id"] = terceroID
	}
	basicos["DatosIdentificacion"] = []any{identificacion}
	return basicos
}

// buscarIdentificaciones consulta datos_identificacion con la query indicada y
// orden determinista por Id.
func (s TerceroWriteService) buscarIdentificaciones(
	ctx context.Context,
	consulta string,
) ([]map[string]any, error) {
	base := strings.TrimRight(strings.TrimSpace(s.URLs["terceros"]), "/")
	if base == "" {
		return nil, errors.New("no hay URL base configurada para terceros")
	}
	u, err := url.Parse(base + endpointDatosIdentificacion)
	if err != nil {
		return nil, fmt.Errorf("URL de identificaciones inválida: %w", err)
	}
	q := u.Query()
	q.Set("query", consulta)
	q.Set("sortby", "Id")
	q.Set("order", "asc")
	q.Set("limit", "10")
	q.Set("offset", "0")
	u.RawQuery = q.Encode()

	var raw json.RawMessage
	if _, err := s.Client.Get(ctx, u.String(), &raw); err != nil {
		return nil, err
	}
	registros, err := decodeAPIList[map[string]any](bytes.TrimSpace(raw))
	if err != nil {
		return nil, err
	}
	// El CRUD puede responder [{}] cuando no hay coincidencias. Esa entrada no
	// representa una identificación y no debe tratarse como una coincidencia.
	validos := make([]map[string]any, 0, len(registros))
	for _, registro := range registros {
		if idDeMapa(registro) <= 0 {
			continue
		}
		validos = append(validos, registro)
	}
	return validos, nil
}

// consultarTercero obtiene el tercero por Id; traduce 404 a ErrTerceroNoEncontrado.
func (s TerceroWriteService) consultarTercero(
	ctx context.Context,
	terceroID int64,
) (map[string]any, error) {
	base := strings.TrimRight(strings.TrimSpace(s.URLs["terceros"]), "/")
	if base == "" {
		return nil, errors.New("no hay URL base configurada para terceros")
	}
	destino := fmt.Sprintf("%s%s/%d", base, endpointTercero, terceroID)
	var raw json.RawMessage
	status, err := s.Client.Get(ctx, destino, &raw)
	if err != nil {
		if status == 404 {
			return nil, ErrTerceroNoEncontrado
		}
		return nil, err
	}
	return decodeAPIRecord(raw)
}

func avisoAmbiguedad(path, mensaje string) responses.RelationError {
	return responses.RelationError{
		Path:    path,
		Service: "terceros_crud",
		Message: mensaje,
		Kind:    "ambiguous_match",
	}
}

func nombreCompleto(primerNombre, segundoNombre, primerApellido, segundoApellido string) string {
	partes := []string{primerNombre, segundoNombre, primerApellido, segundoApellido}
	var limpio []string
	for _, parte := range partes {
		if parte = strings.TrimSpace(parte); parte != "" {
			limpio = append(limpio, parte)
		}
	}
	return strings.Join(limpio, " ")
}

func idDeMapa(m map[string]any) int64 {
	return int64(idEntero(m["Id"]))
}

func idDeRelacion(v any) int64 {
	switch valor := v.(type) {
	case map[string]any:
		return idDeMapa(valor)
	default:
		return int64(idEntero(valor))
	}
}

func extraerTerceroID(data map[string]any, fallback int64) int64 {
	if id := idDeMapa(data); id > 0 {
		return id
	}
	return fallback
}

func extraerIdentificacionID(data map[string]any, fallback int64) int64 {
	valor, ok := data["DatosIdentificacion"]
	if !ok {
		return fallback
	}
	switch v := valor.(type) {
	case []any:
		for _, elem := range v {
			if m, ok := elem.(map[string]any); ok {
				if id := idDeMapa(m); id > 0 {
					return id
				}
			}
		}
	case []map[string]any:
		for _, m := range v {
			if id := idDeMapa(m); id > 0 {
				return id
			}
		}
	case map[string]any:
		if id := idDeMapa(v); id > 0 {
			return id
		}
	}
	return fallback
}
