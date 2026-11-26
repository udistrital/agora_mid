package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/udistrital/agora_mid/clients"
	"github.com/udistrital/agora_mid/config"
	"github.com/udistrital/agora_mid/helper"
	"github.com/udistrital/agora_mid/models"
	"github.com/udistrital/agora_mid/responses"
)

// Errores de negocio del dominio 1 que el controlador mapea a códigos HTTP
// concretos.
var (
	ErrTipoContribuyenteInvalido              = errors.New("TipoContribuyenteId no corresponde a la variante natural")
	ErrTipoContribuyenteJuridicoInvalido      = errors.New("TipoContribuyenteId no corresponde a la variante jurídica")
	ErrTipoContribuyenteConflicto             = errors.New("el tercero existente tiene un TipoContribuyenteId distinto")
	ErrTipoContribuyenteNoConfigurado         = errors.New("no está configurado el tipo de contribuyente P_NATURAL")
	ErrTipoContribuyenteJuridicoNoConfigurado = errors.New("no está configurado el tipo de contribuyente P_JURIDICA")
	ErrTipoDocumentoNITNoConfigurado          = errors.New("no está configurado el tipo de documento NIT")
	ErrTerceroNoEncontrado                    = errors.New("no existe el tercero indicado")
	ErrTerceroInactivo                        = errors.New("el tercero está inactivo y no se puede registrar ni actualizar")
	ErrIdentificacionDeOtroTercero            = errors.New("el documento ya pertenece a otro tercero")
	ErrIdentificacionInactiva                 = errors.New("la identificación está inactiva y no se puede reutilizar ni actualizar")
	ErrIdentificacionInmutable                = errors.New("el número y el tipo de documento no se pueden cambiar en la actualización")
)

// ValidationError describe un dato de entrada inválido. El controlador lo mapea
// a 400 con el mensaje público.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

func invalid(message string) error {
	return &ValidationError{Message: message}
}

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
	Config                    config.ServiceConfig
	Registry                  helper.WriteResolverRegistry
	Client                    clients.HTTPClient
	URLs                      config.ServiceURLs
	TipoContribuyenteNatural  int
	TipoContribuyenteJuridica int
	TipoDocumentoNIT          int
	Catalogos                 CatalogosValidator
	mutex                     *keyedMutex
}

// NewTerceroWriteService construye el servicio con la configuración del servicio
// tercero y los catálogos resueltos al arranque.
func NewTerceroWriteService(
	client clients.HTTPClient,
	urls config.ServiceURLs,
	cfg config.ServiceConfig,
	tipoContribuyenteNatural int,
	tipoContribuyenteJuridica int,
	tipoDocumentoNIT int,
	catalogos CatalogosValidator,
) TerceroWriteService {
	raiz, hijos := NewTerceroWriteResolvers(client, urls)
	registry := helper.NewWriteResolverRegistry(raiz, hijos)
	return TerceroWriteService{
		Config:                    cfg,
		Registry:                  registry,
		Client:                    client,
		URLs:                      urls,
		TipoContribuyenteNatural:  tipoContribuyenteNatural,
		TipoContribuyenteJuridica: tipoContribuyenteJuridica,
		TipoDocumentoNIT:          tipoDocumentoNIT,
		Catalogos:                 catalogos,
		mutex:                     newKeyedMutex(),
	}
}

// varianteTercero distingue las reglas de construcción del tercero entre persona
// natural y persona jurídica. La identificación y la clasificación son comunes.
type varianteTercero string

const (
	varianteNatural  varianteTercero = "natural"
	varianteJuridica varianteTercero = "juridica"
)

// entradaEscrituraTercero es la vista neutral de ambos payloads. Solo una de las
// dos variantes viene diligenciada.
type entradaEscrituraTercero struct {
	tipoContribuyente int64
	variante          varianteTercero
	natural           *models.DatosBasicosTercero
	juridica          *models.DatosBasicosJuridico
	identificacion    models.DatosIdentificacionTercero
}

// EscribirTercero registra o actualiza el tercero persona natural y su
// identificación.
//
// verb es helper.WriteVerbCreate para POST o helper.WriteVerbUpdate para PUT.
// terceroID es el Id de la ruta; en POST debe ser cero.
func (s TerceroWriteService) EscribirTercero(
	ctx context.Context,
	in models.RegistroTerceroNatural,
	terceroID int64,
	verb string,
) (TerceroWriteResult, error) {
	in.DatosBasicos.PrimerNombre = strings.TrimSpace(in.DatosBasicos.PrimerNombre)
	in.DatosBasicos.SegundoNombre = strings.TrimSpace(in.DatosBasicos.SegundoNombre)
	in.DatosBasicos.PrimerApellido = strings.TrimSpace(in.DatosBasicos.PrimerApellido)
	in.DatosBasicos.SegundoApellido = strings.TrimSpace(in.DatosBasicos.SegundoApellido)
	in.DatosIdentificacion.Numero = strings.TrimSpace(in.DatosIdentificacion.Numero)

	return s.escribir(ctx, entradaEscrituraTercero{
		tipoContribuyente: in.DatosBasicos.TipoContribuyenteId,
		variante:          varianteNatural,
		natural:           &in.DatosBasicos,
		identificacion:    in.DatosIdentificacion,
	}, terceroID, verb)
}

// EscribirTerceroJuridico registra o actualiza el tercero persona jurídica y su
// identificación. RazonSocial se persiste en tercero.NombreCompleto.
func (s TerceroWriteService) EscribirTerceroJuridico(
	ctx context.Context,
	in models.RegistroTerceroJuridico,
	terceroID int64,
	verb string,
) (TerceroWriteResult, error) {
	in.DatosBasicos.RazonSocial = normalizarRazonSocial(in.DatosBasicos.RazonSocial)
	in.DatosIdentificacion.Numero = strings.TrimSpace(in.DatosIdentificacion.Numero)

	return s.escribir(ctx, entradaEscrituraTercero{
		tipoContribuyente: in.DatosBasicos.TipoContribuyenteId,
		variante:          varianteJuridica,
		juridica:          &in.DatosBasicos,
		identificacion:    in.DatosIdentificacion,
	}, terceroID, verb)
}

// escribir contiene el flujo compartido del dominio 1: validar la variante y el
// payload, serializar por documento, resolver la identificación, validar el
// tercero existente, materializar y clasificar el resultado.
func (s TerceroWriteService) escribir(
	ctx context.Context,
	in entradaEscrituraTercero,
	terceroID int64,
	verb string,
) (TerceroWriteResult, error) {
	tipoEsperado, errInvalido, errNoConfigurado := s.tipoEsperado(in.variante)
	if tipoEsperado <= 0 {
		return TerceroWriteResult{}, errNoConfigurado
	}
	if in.tipoContribuyente <= 0 || in.tipoContribuyente != int64(tipoEsperado) {
		return TerceroWriteResult{}, errInvalido
	}

	esActualizacion := verb == helper.WriteVerbUpdate
	if esActualizacion && terceroID <= 0 {
		return TerceroWriteResult{}, ErrTerceroNoEncontrado
	}

	if err := s.validarEntrada(ctx, in); err != nil {
		return TerceroWriteResult{}, err
	}

	// Serializa las peticiones que compiten por el mismo documento dentro de esta
	// instancia. Varias instancias del MID aún pueden duplicar; la mitigación
	// definitiva requiere una restricción única en terceros_crud.
	unlock := func() {}
	if s.mutex != nil {
		unlock = s.mutex.lock(claveDocumento(in))
	}
	defer unlock()

	terceroResuelto, identificacionID, avisos, err := s.resolverIdentificacion(
		ctx, in.identificacion, terceroID, esActualizacion,
	)
	if err != nil {
		return TerceroWriteResult{}, err
	}
	if terceroID == 0 {
		terceroID = terceroResuelto
	}

	// Si ya hay identificación, el tercero existe o se reutiliza. Se lee para
	// validar el tipo, el estado y para conservar los campos existentes.
	if terceroID > 0 {
		existente, err := s.consultarTercero(ctx, terceroID)
		if err != nil {
			return TerceroWriteResult{}, err
		}
		tipoExistente := idDeRelacion(existente["TipoContribuyenteId"])
		if tipoExistente != int64(tipoEsperado) {
			if tipoExistente <= 0 {
				return TerceroWriteResult{}, errors.New("el tercero existente no reporta un TipoContribuyenteId válido")
			}
			return TerceroWriteResult{}, ErrTipoContribuyenteConflicto
		}
		activo, ok := existente["Activo"].(bool)
		if !ok {
			return TerceroWriteResult{}, errors.New("el tercero existente no reporta Activo")
		}
		if !activo {
			return TerceroWriteResult{}, ErrTerceroInactivo
		}
	}

	esAlta := !esActualizacion && terceroID == 0
	body := s.construirCuerpo(in, terceroID, identificacionID, esAlta, esActualizacion)
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
	case esAlta:
		out.Creados = append(out.Creados, models.EntidadId{Entidad: "Tercero", Id: out.TerceroId})
	case esActualizacion:
		out.Actualizados = append(out.Actualizados, models.EntidadId{Entidad: "Tercero", Id: out.TerceroId})
	default:
		out.Reutilizados = append(out.Reutilizados, models.EntidadId{Entidad: "Tercero", Id: out.TerceroId})
	}

	if identificacionID == 0 {
		// Solo se informa como creada si el motor devolvió un Id real. En un fallo
		// parcial el aviso describe la identificación no materializada.
		if out.DatosIdentificacionId > 0 {
			out.Creados = append(out.Creados, models.EntidadId{Entidad: "DatosIdentificacion", Id: out.DatosIdentificacionId})
		}
	} else {
		out.Actualizados = append(out.Actualizados, models.EntidadId{Entidad: "DatosIdentificacion", Id: out.DatosIdentificacionId})
	}

	if esAlta {
		if aviso := s.avisoDuplicado(ctx, in.identificacion); aviso.Kind != "" {
			out.Avisos = append(out.Avisos, aviso)
		}
	}

	return out, nil
}

// tipoEsperado devuelve el ID configurado de la variante y los errores que
// corresponden a esa variante.
func (s TerceroWriteService) tipoEsperado(
	variante varianteTercero,
) (int, error, error) {
	if variante == varianteJuridica {
		return s.TipoContribuyenteJuridica,
			ErrTipoContribuyenteJuridicoInvalido,
			ErrTipoContribuyenteJuridicoNoConfigurado
	}
	return s.TipoContribuyenteNatural,
		ErrTipoContribuyenteInvalido,
		ErrTipoContribuyenteNoConfigurado
}

// validarEntrada aplica las reglas de negocio del dominio 1 antes de cualquier
// lectura o escritura.
func (s TerceroWriteService) validarEntrada(
	ctx context.Context,
	in entradaEscrituraTercero,
) error {
	if err := s.validarIdentificacion(ctx, in); err != nil {
		return err
	}

	switch in.variante {
	case varianteJuridica:
		if in.juridica == nil {
			return invalid("DatosBasicos es obligatorio")
		}
		if err := validarRazonSocial(in.juridica.RazonSocial); err != nil {
			return err
		}
		if err := s.validarLugar(ctx, in.juridica.LugarOrigen, "DatosBasicos.LugarOrigen"); err != nil {
			return err
		}
	default:
		if in.natural == nil {
			return invalid("DatosBasicos es obligatorio")
		}
		if err := validarNombresNaturales(in.natural); err != nil {
			return err
		}
		if err := s.validarLugar(ctx, in.natural.LugarOrigen, "DatosBasicos.LugarOrigen"); err != nil {
			return err
		}
		if err := validarFecha(in.natural.FechaNacimiento, "DatosBasicos.FechaNacimiento"); err != nil {
			return err
		}
	}

	return nil
}

func (s TerceroWriteService) validarIdentificacion(
	ctx context.Context,
	in entradaEscrituraTercero,
) error {
	id := in.identificacion
	numero := strings.TrimSpace(id.Numero)
	if numero == "" {
		return invalid("DatosIdentificacion.Numero es obligatorio")
	}
	if utf8.RuneCountInString(numero) > 50 {
		return invalid("DatosIdentificacion.Numero no puede superar 50 caracteres")
	}
	if strings.ContainsAny(numero, ",:") {
		return invalid("DatosIdentificacion.Numero no admite ',' ni ':'")
	}
	for _, r := range numero {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return invalid("DatosIdentificacion.Numero no admite espacios ni caracteres de control")
		}
	}

	if id.TipoDocumentoId <= 0 {
		return invalid("DatosIdentificacion.TipoDocumentoId es obligatorio")
	}

	esNIT := s.TipoDocumentoNIT > 0 && id.TipoDocumentoId == int64(s.TipoDocumentoNIT)
	if in.variante == varianteJuridica {
		if s.TipoDocumentoNIT <= 0 {
			return ErrTipoDocumentoNITNoConfigurado
		}
		if !esNIT {
			return invalid("para persona jurídica el TipoDocumentoId debe ser NIT")
		}
	} else if s.Catalogos != nil && !esNIT {
		// Si el NIT viene configurado por entorno, se confía en ese Id y no se
		// consulta tipo_documento. El resto de tipos de documento sí se valida.
		existe, err := s.Catalogos.TipoDocumentoExiste(ctx, id.TipoDocumentoId)
		if err != nil {
			return err
		}
		if !existe {
			return invalid("DatosIdentificacion.TipoDocumentoId no existe o está inactivo")
		}
	}

	if esNIT && !soloDigitos(numero) {
		return invalid("DatosIdentificacion.Numero de NIT debe contener solo dígitos")
	}
	if esNIT && id.DigitoVerificacion == nil {
		return invalid("DatosIdentificacion.DigitoVerificacion es obligatorio para NIT")
	}
	if id.DigitoVerificacion != nil {
		if !esEntero(*id.DigitoVerificacion) || *id.DigitoVerificacion < 0 || *id.DigitoVerificacion > 9 {
			return invalid("DatosIdentificacion.DigitoVerificacion debe ser un entero entre 0 y 9")
		}
	}

	if err := s.validarLugar(ctx, id.CiudadExpedicion, "DatosIdentificacion.CiudadExpedicion"); err != nil {
		return err
	}
	if err := validarFecha(id.FechaExpedicion, "DatosIdentificacion.FechaExpedicion"); err != nil {
		return err
	}
	return nil
}

func (s TerceroWriteService) validarLugar(
	ctx context.Context,
	id *int64,
	campo string,
) error {
	if id == nil {
		return nil
	}
	if *id < 0 {
		return invalid(campo + " no puede ser negativo")
	}
	if *id == 0 || s.Catalogos == nil {
		return nil
	}
	existe, err := s.Catalogos.LugarExiste(ctx, *id)
	if err != nil {
		return err
	}
	if !existe {
		return invalid(campo + " no existe o está inactivo")
	}
	return nil
}

func validarRazonSocial(razonSocial string) error {
	razonSocial = normalizarRazonSocial(razonSocial)
	if razonSocial == "" {
		return invalid("DatosBasicos.RazonSocial es obligatorio")
	}
	if utf8.RuneCountInString(razonSocial) > 255 {
		return invalid("DatosBasicos.RazonSocial no puede superar 255 caracteres")
	}
	for _, r := range razonSocial {
		if unicode.IsControl(r) {
			return invalid("DatosBasicos.RazonSocial no admite caracteres de control")
		}
	}
	return nil
}

func validarNombresNaturales(in *models.DatosBasicosTercero) error {
	if strings.TrimSpace(in.PrimerNombre) == "" || strings.TrimSpace(in.PrimerApellido) == "" {
		return invalid("DatosBasicos.PrimerNombre y PrimerApellido son obligatorios")
	}
	nombres := []struct {
		valor string
		campo string
	}{
		{in.PrimerNombre, "DatosBasicos.PrimerNombre"},
		{in.SegundoNombre, "DatosBasicos.SegundoNombre"},
		{in.PrimerApellido, "DatosBasicos.PrimerApellido"},
		{in.SegundoApellido, "DatosBasicos.SegundoApellido"},
	}
	for _, nombre := range nombres {
		if utf8.RuneCountInString(nombre.valor) > 100 {
			return invalid(nombre.campo + " no puede superar 100 caracteres")
		}
	}
	completo := nombreCompleto(in.PrimerNombre, in.SegundoNombre, in.PrimerApellido, in.SegundoApellido)
	if utf8.RuneCountInString(completo) > 255 {
		return invalid("El NombreCompleto derivado no puede superar 255 caracteres")
	}
	return nil
}

func validarFecha(fecha *time.Time, campo string) error {
	if fecha == nil {
		return nil
	}
	if fecha.After(time.Now()) {
		return invalid(campo + " no puede ser futura")
	}
	return nil
}

func soloDigitos(valor string) bool {
	if valor == "" {
		return false
	}
	for _, r := range valor {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func esEntero(valor float64) bool {
	return math.Trunc(valor) == valor
}

// resolverIdentificacion busca la identificación que corresponde al documento,
// incluyendo inactivas para rechazarlas, y en actualización valida que no
// pertenezca a otro tercero.
func (s TerceroWriteService) resolverIdentificacion(
	ctx context.Context,
	identificacion models.DatosIdentificacionTercero,
	terceroID int64,
	esActualizacion bool,
) (int64, int64, []responses.RelationError, error) {
	identificacionID := int64(0)
	terceroResuelto := terceroID
	var avisos []responses.RelationError
	var activa map[string]any

	if esActualizacion {
		registros, err := s.buscarIdentificaciones(ctx, fmt.Sprintf("TerceroId:%d", terceroID))
		if err != nil {
			return 0, 0, nil, err
		}
		activas, inactivas := separarIdentificaciones(registros)
		switch {
		case len(activas) > 0:
			activa = activas[0]
			identificacionID = idDeMapa(activa)
			if len(activas) > 1 {
				avisos = append(avisos, avisoAmbiguedad("DatosIdentificacion",
					"Se encontraron varias identificaciones activas del tercero; se usó la primera"))
			}
		case len(inactivas) > 0:
			return 0, 0, nil, ErrIdentificacionInactiva
		}
	}

	documentos, err := s.buscarIdentificaciones(ctx, fmt.Sprintf(
		"Numero:%s,TipoDocumentoId:%d",
		identificacion.Numero, identificacion.TipoDocumentoId,
	))
	if err != nil {
		return 0, 0, nil, err
	}
	activos, inactivos := separarIdentificaciones(documentos)
	switch {
	case len(activos) > 0:
		terceroDocumento := idDeRelacion(activos[0]["TerceroId"])
		if terceroDocumento <= 0 {
			return 0, 0, nil, errors.New("la identificación activa no trae un TerceroId válido")
		}
		if esActualizacion && terceroDocumento != terceroID {
			return 0, 0, nil, ErrIdentificacionDeOtroTercero
		}
		if !esActualizacion {
			terceroResuelto = terceroDocumento
			identificacionID = idDeMapa(activos[0])
		}
		if len(activos) > 1 {
			avisos = append(avisos, avisoAmbiguedad("DatosIdentificacion",
				"Se encontraron varias identificaciones activas para el documento; se usó la primera"))
		}
	case len(inactivos) > 0:
		return 0, 0, nil, ErrIdentificacionInactiva
	}

	// La identidad del tercero no se puede reasignar en una actualización. Si el
	// payload cambia Numero o TipoDocumentoId respecto de la identificación
	// activa, se rechaza antes de escribir.
	if esActualizacion && activa != nil && !mismaIdentidad(activa, identificacion) {
		return 0, 0, nil, ErrIdentificacionInmutable
	}

	if esActualizacion && identificacionID == 0 && len(activos) > 0 &&
		idDeRelacion(activos[0]["TerceroId"]) == terceroID {
		identificacionID = idDeMapa(activos[0])
	}

	return terceroResuelto, identificacionID, avisos, nil
}

func separarIdentificaciones(
	registros []map[string]any,
) (activas []map[string]any, inactivas []map[string]any) {
	for _, registro := range registros {
		if idDeMapa(registro) <= 0 {
			continue
		}
		if activo, _ := registro["Activo"].(bool); activo {
			activas = append(activas, registro)
		} else {
			inactivas = append(inactivas, registro)
		}
	}
	return activas, inactivas
}

// mismaIdentidad compara el Numero y el TipoDocumentoId de una identificación
// almacenada con los valores del payload.
func mismaIdentidad(registro map[string]any, in models.DatosIdentificacionTercero) bool {
	numero, ok := registro["Numero"].(string)
	if !ok {
		return false
	}
	return strings.TrimSpace(numero) == strings.TrimSpace(in.Numero) &&
		idDeRelacion(registro["TipoDocumentoId"]) == in.TipoDocumentoId
}

// avisoDuplicado detecta duplicados creados por otra instancia del MID. No altera
// el resultado de la escritura; solo informa en MetaData.Errors.
func (s TerceroWriteService) avisoDuplicado(
	ctx context.Context,
	id models.DatosIdentificacionTercero,
) responses.RelationError {
	registros, err := s.buscarIdentificaciones(ctx, fmt.Sprintf(
		"Numero:%s,TipoDocumentoId:%d", id.Numero, id.TipoDocumentoId,
	))
	if err != nil {
		return responses.RelationError{}
	}
	activas, _ := separarIdentificaciones(registros)
	if len(activas) <= 1 {
		return responses.RelationError{}
	}
	return responses.RelationError{
		Path:    "DatosIdentificacion",
		Service: "terceros_crud",
		Message: "Se encontraron varias identificaciones activas para el documento",
		Kind:    "duplicate_match",
	}
}

// construirCuerpo arma el cuerpo anidado que espera el motor. En reutilización
// de persona natural no se envían nombres ni LugarOrigen para no sobrescribirlos.
// En persona jurídica, el PUT sí actualiza NombreCompleto y LugarOrigen.
func (s TerceroWriteService) construirCuerpo(
	in entradaEscrituraTercero,
	terceroID int64,
	identificacionID int64,
	esAlta bool,
	esActualizacion bool,
) map[string]any {
	basicos := map[string]any{
		"TipoContribuyenteId": in.tipoContribuyente,
	}

	switch in.variante {
	case varianteJuridica:
		if esAlta {
			basicos["Activo"] = true
		}
		if (esAlta || esActualizacion) && in.juridica != nil {
			basicos["NombreCompleto"] = normalizarRazonSocial(in.juridica.RazonSocial)
			if in.juridica.LugarOrigen != nil {
				basicos["LugarOrigen"] = *in.juridica.LugarOrigen
			}
		}

	default:
		if esAlta && in.natural != nil {
			basicos["PrimerNombre"] = in.natural.PrimerNombre
			basicos["SegundoNombre"] = in.natural.SegundoNombre
			basicos["PrimerApellido"] = in.natural.PrimerApellido
			basicos["SegundoApellido"] = in.natural.SegundoApellido
			basicos["NombreCompleto"] = nombreCompleto(
				in.natural.PrimerNombre,
				in.natural.SegundoNombre,
				in.natural.PrimerApellido,
				in.natural.SegundoApellido,
			)
			basicos["Activo"] = true
		}
		if in.natural != nil && in.natural.FechaNacimiento != nil {
			basicos["FechaNacimiento"] = *in.natural.FechaNacimiento
		}
		if esAlta && in.natural != nil && in.natural.LugarOrigen != nil {
			basicos["LugarOrigen"] = *in.natural.LugarOrigen
		}
	}

	identificacion := map[string]any{
		"Numero":          in.identificacion.Numero,
		"TipoDocumentoId": in.identificacion.TipoDocumentoId,
	}
	if identificacionID == 0 {
		identificacion["Activo"] = true
	}
	if in.identificacion.CiudadExpedicion != nil {
		identificacion["CiudadExpedicion"] = *in.identificacion.CiudadExpedicion
	}
	if in.identificacion.FechaExpedicion != nil {
		identificacion["FechaExpedicion"] = *in.identificacion.FechaExpedicion
	}
	if in.identificacion.DigitoVerificacion != nil {
		identificacion["DigitoVerificacion"] = *in.identificacion.DigitoVerificacion
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
// orden determinista por Id. La query decide si incluye o no Activo.
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

// normalizarRazonSocial recorta y colapsa los espacios de la razón social.
func normalizarRazonSocial(razonSocial string) string {
	return strings.Join(strings.Fields(razonSocial), " ")
}

func claveDocumento(in entradaEscrituraTercero) string {
	return fmt.Sprintf("%s|%s|%d", in.variante, in.identificacion.Numero, in.identificacion.TipoDocumentoId)
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

// keyedMutex serializa operaciones por clave y libera la entrada cuando no hay
// más esperas.
type keyedMutex struct {
	mu      sync.Mutex
	entries map[string]*entradaMutex
}

type entradaMutex struct {
	mu   sync.Mutex
	refs int
}

func newKeyedMutex() *keyedMutex {
	return &keyedMutex{entries: make(map[string]*entradaMutex)}
}

func (k *keyedMutex) lock(clave string) func() {
	k.mu.Lock()
	entrada := k.entries[clave]
	if entrada == nil {
		entrada = &entradaMutex{}
		k.entries[clave] = entrada
	}
	entrada.refs++
	k.mu.Unlock()

	entrada.mu.Lock()
	return func() {
		entrada.mu.Unlock()
		k.mu.Lock()
		entrada.refs--
		if entrada.refs == 0 {
			delete(k.entries, clave)
		}
		k.mu.Unlock()
	}
}
