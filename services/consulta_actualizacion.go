package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/udistrital/agora_mid/dto"
)

var (
	ErrPersonaNoEncontrada   = errors.New("no se encontró el registro de persona solicitado")
	ErrActualizacionInvalida = errors.New("solicitud de actualización inválida")
	ErrDocumentoDuplicado    = errors.New("el documento ya está asociado a otra identificación activa")
)

// PersonaIntegradaResultado reúne la identidad en Terceros y el registro propio de Agora.
// Se preserva el contrato JSON de los CRUD externos dentro de cada bloque.
type PersonaIntegradaResultado struct {
	TerceroID           int             `json:"tercero_id"`
	TipoRegistro        string          `json:"tipo_registro"`
	Terceros            json.RawMessage `json:"terceros"`
	DatosIdentificacion json.RawMessage `json:"datos_identificacion"`
	Proveedor           json.RawMessage `json:"proveedor"`
	ProveedorNatural    json.RawMessage `json:"proveedor_natural,omitempty"`
	ProveedorJuridico   json.RawMessage `json:"proveedor_juridico,omitempty"`
}

type apiEnvelopeRaw struct {
	Data json.RawMessage `json:"Data"`
}

// ConsultarPersona integra Terceros y Agora por el tercero_id y comprueba
// que el subtipo solicitado corresponda al registro existente.
func (s RegistroPersonaService) ConsultarPersona(
	ctx context.Context, terceroID int, tipo string,
) (PersonaIntegradaResultado, error) {
	resultado := PersonaIntegradaResultado{TerceroID: terceroID, TipoRegistro: tipo}
	if terceroID <= 0 {
		return resultado, fmt.Errorf("%w: tercero_id debe ser mayor que cero", ErrActualizacionInvalida)
	}
	if tipo != "natural" && tipo != "juridica" {
		return resultado, fmt.Errorf("%w: tipo de registro no válido", ErrActualizacionInvalida)
	}
	if strings.TrimSpace(s.Terceros.BaseURL) == "" {
		return resultado, errors.New("falta la URL base de Terceros CRUD")
	}
	if strings.TrimSpace(s.GestionPersonas.BaseURL) == "" {
		return resultado, errors.New("falta la URL base de gestión de personas CRUD")
	}

	tercero, status, err := obtenerDataRaw(ctx, s.Terceros.Client, basePath(s.Terceros.BaseURL, "/v1/tercero/"+strconv.Itoa(terceroID)))
	if err != nil {
		if status == http.StatusNotFound {
			return resultado, ErrPersonaNoEncontrada
		}
		return resultado, fmt.Errorf("consultando tercero %d: %w", terceroID, err)
	}
	resultado.Terceros = tercero

	identificaciones, err := s.Terceros.consultarIdentificacionesRaw(ctx, terceroID)
	if err != nil {
		return resultado, fmt.Errorf("consultando identificaciones del tercero %d: %w", terceroID, err)
	}
	resultado.DatosIdentificacion = identificaciones

	proveedor, err := s.GestionPersonas.BuscarProveedorPorTercero(ctx, terceroID)
	if err != nil {
		return resultado, err
	}
	if proveedor == nil {
		return resultado, ErrPersonaNoEncontrada
	}
	proveedorRaw, _, err := obtenerDataRaw(ctx, s.GestionPersonas.Client,
		basePath(s.GestionPersonas.BaseURL, "/v1/proveedor/"+strconv.Itoa(proveedor.Id)))
	if err != nil {
		return resultado, fmt.Errorf("consultando proveedor %d: %w", proveedor.Id, err)
	}
	resultado.Proveedor = proveedorRaw

	if tipo == "natural" {
		detalle, status, err := obtenerDataRaw(ctx, s.GestionPersonas.Client,
			basePath(s.GestionPersonas.BaseURL, "/v1/proveedor_natural/"+strconv.Itoa(proveedor.Id)))
		if err != nil {
			if status == http.StatusNotFound {
				return resultado, ErrPersonaNoEncontrada
			}
			return resultado, fmt.Errorf("consultando proveedor natural %d: %w", proveedor.Id, err)
		}
		resultado.ProveedorNatural = detalle
	} else {
		detalle, status, err := obtenerDataRaw(ctx, s.GestionPersonas.Client,
			basePath(s.GestionPersonas.BaseURL, "/v1/proveedor_juridico/"+strconv.Itoa(proveedor.Id)))
		if err != nil {
			if status == http.StatusNotFound {
				return resultado, ErrPersonaNoEncontrada
			}
			return resultado, fmt.Errorf("consultando proveedor jurídico %d: %w", proveedor.Id, err)
		}
		resultado.ProveedorJuridico = detalle
	}
	return resultado, nil
}

// ActualizarPersona actualiza únicamente las propiedades recibidas y las envía
// al servicio responsable. Los campos de tipo/número documental son inmutables
// en este endpoint para evitar cambios de identidad por sobrescritura accidental.
func (s RegistroPersonaService) ActualizarPersona(
	ctx context.Context, req dto.ActualizarPersonaRequest, tipo string,
) (PersonaIntegradaResultado, error) {
	if req.TerceroID <= 0 {
		return PersonaIntegradaResultado{}, fmt.Errorf("%w: tercero_id es obligatorio", ErrActualizacionInvalida)
	}
	if tipo != "natural" && tipo != "juridica" {
		return PersonaIntegradaResultado{}, fmt.Errorf("%w: tipo de registro no válido", ErrActualizacionInvalida)
	}
	if len(req.Terceros) == 0 && len(req.DatosIdentificacion) == 0 && len(req.Proveedor) == 0 && len(req.ProveedorNatural) == 0 && len(req.ProveedorJuridico) == 0 {
		return PersonaIntegradaResultado{}, fmt.Errorf("%w: no hay campos para actualizar", ErrActualizacionInvalida)
	}
	if (tipo == "natural" && len(req.ProveedorJuridico) > 0) || (tipo == "juridica" && len(req.ProveedorNatural) > 0) {
		return PersonaIntegradaResultado{}, fmt.Errorf("%w: el bloque específico no corresponde al tipo de persona", ErrActualizacionInvalida)
	}
	if err := validarSolicitudActualizacion(req); err != nil {
		return PersonaIntegradaResultado{}, err
	}
	if strings.TrimSpace(s.Terceros.BaseURL) == "" && (len(req.Terceros) > 0 || len(req.DatosIdentificacion) > 0) {
		return PersonaIntegradaResultado{}, errors.New("falta la URL base de Terceros CRUD")
	}
	if strings.TrimSpace(s.GestionPersonas.BaseURL) == "" {
		return PersonaIntegradaResultado{}, errors.New("falta la URL base de gestión de personas CRUD")
	}

	proveedor, err := s.GestionPersonas.BuscarProveedorPorTercero(ctx, req.TerceroID)
	if err != nil {
		return PersonaIntegradaResultado{TerceroID: req.TerceroID}, err
	}
	if proveedor == nil {
		return PersonaIntegradaResultado{TerceroID: req.TerceroID}, ErrPersonaNoEncontrada
	}
	perfilPath := "/v1/proveedor_natural/"
	if tipo == "juridica" {
		perfilPath = "/v1/proveedor_juridico/"
	}
	if _, status, err := obtenerDataRaw(ctx, s.GestionPersonas.Client, basePath(s.GestionPersonas.BaseURL, perfilPath+strconv.Itoa(proveedor.Id))); err != nil {
		if status == http.StatusNotFound {
			return PersonaIntegradaResultado{TerceroID: req.TerceroID}, ErrPersonaNoEncontrada
		}
		return PersonaIntegradaResultado{TerceroID: req.TerceroID}, fmt.Errorf("validando el tipo de registro antes de actualizar: %w", err)
	}
	// Verificar pertenencia de la identificación antes de modificar cualquier otro bloque.
	if len(req.DatosIdentificacion) > 0 {
		var identificacionID int
		_ = json.Unmarshal(req.DatosIdentificacion["id"], &identificacionID)
		if err := s.Terceros.verificarIdentificacionTercero(ctx, req.TerceroID, identificacionID); err != nil {
			return PersonaIntegradaResultado{TerceroID: req.TerceroID}, err
		}
		if numeroRaw, cambiaNumero := req.DatosIdentificacion["numero_documento"]; cambiaNumero {
			var numero, codigo string
			_ = json.Unmarshal(numeroRaw, &numero)
			_ = json.Unmarshal(req.DatosIdentificacion["codigo_tipo_documento"], &codigo)
			if err := s.Terceros.validarDocumentoDisponible(ctx, numero, codigo, identificacionID); err != nil {
				return PersonaIntegradaResultado{TerceroID: req.TerceroID}, err
			}
			if _, err := s.Terceros.ResolverTipoDocumento(ctx, codigo); err != nil {
				return PersonaIntegradaResultado{TerceroID: req.TerceroID}, fmt.Errorf("resolviendo el nuevo tipo documental: %w", err)
			}
		}
	}

	if len(req.Terceros) > 0 {
		if err := s.Terceros.actualizarTercero(ctx, req.TerceroID, req.Terceros); err != nil {
			return PersonaIntegradaResultado{TerceroID: req.TerceroID}, fmt.Errorf("actualizando identidad en Terceros: %w", err)
		}
	}
	if len(req.DatosIdentificacion) > 0 {
		if err := s.Terceros.actualizarIdentificacion(ctx, req.TerceroID, req.DatosIdentificacion); err != nil {
			return PersonaIntegradaResultado{TerceroID: req.TerceroID}, fmt.Errorf("actualización parcial: tercero_id=%d; otras operaciones anteriores pueden haberse aplicado; falló datos_identificacion: %w", req.TerceroID, err)
		}
	}
	if len(req.Proveedor) > 0 {
		if err := s.GestionPersonas.actualizarProveedor(ctx, proveedor.Id, req.Proveedor); err != nil {
			return PersonaIntegradaResultado{TerceroID: req.TerceroID}, fmt.Errorf("actualización parcial: tercero_id=%d; falló proveedor en Agora: %w", req.TerceroID, err)
		}
	}
	if tipo == "natural" && len(req.ProveedorNatural) > 0 {
		if err := s.GestionPersonas.actualizarProveedorNatural(ctx, proveedor.Id, req.ProveedorNatural); err != nil {
			return PersonaIntegradaResultado{TerceroID: req.TerceroID}, fmt.Errorf("actualización parcial: tercero_id=%d; falló proveedor_natural en Agora: %w", req.TerceroID, err)
		}
	}
	if tipo == "juridica" && len(req.ProveedorJuridico) > 0 {
		if err := s.GestionPersonas.actualizarProveedorJuridico(ctx, proveedor.Id, req.ProveedorJuridico); err != nil {
			return PersonaIntegradaResultado{TerceroID: req.TerceroID}, fmt.Errorf("actualización parcial: tercero_id=%d; falló proveedor_juridico en Agora: %w", req.TerceroID, err)
		}
	}
	return s.ConsultarPersona(ctx, req.TerceroID, tipo)
}

func validarSolicitudActualizacion(req dto.ActualizarPersonaRequest) error {
	terceros := map[string]string{
		"nombre_completo": "NombreCompleto", "primer_nombre": "PrimerNombre", "segundo_nombre": "SegundoNombre",
		"primer_apellido": "PrimerApellido", "segundo_apellido": "SegundoApellido", "fecha_nacimiento": "FechaNacimiento", "lugar_origen": "LugarOrigen",
	}
	for key, raw := range req.Terceros {
		if _, ok := terceros[key]; !ok {
			return fmt.Errorf("%w: campo de terceros no actualizable: %s; tipo y número documental no se modifican por este endpoint", ErrActualizacionInvalida, key)
		}
		switch key {
		case "nombre_completo", "primer_nombre", "segundo_nombre", "primer_apellido", "segundo_apellido":
			var value string
			if json.Unmarshal(raw, &value) != nil {
				return fmt.Errorf("%w: %s debe ser texto", ErrActualizacionInvalida, key)
			}
			if key == "nombre_completo" && strings.TrimSpace(value) == "" {
				return fmt.Errorf("%w: nombre_completo no puede estar vacío", ErrActualizacionInvalida)
			}
		case "fecha_nacimiento":
			if _, err := normalizarFechaJSON(raw); err != nil {
				return fmt.Errorf("%w: fecha_nacimiento: %v", ErrActualizacionInvalida, err)
			}
		case "lugar_origen":
			var value int
			if json.Unmarshal(raw, &value) != nil || value < 0 {
				return fmt.Errorf("%w: lugar_origen debe ser un entero no negativo", ErrActualizacionInvalida)
			}
		}
	}
	identificacion := map[string]bool{"id": true, "numero_documento": true, "codigo_tipo_documento": true, "digito_verificacion": true, "ciudad_expedicion": true, "fecha_expedicion": true}
	if len(req.DatosIdentificacion) > 0 {
		var id int
		raw, ok := req.DatosIdentificacion["id"]
		if !ok || json.Unmarshal(raw, &id) != nil || id <= 0 {
			return fmt.Errorf("%w: datos_identificacion.id es obligatorio", ErrActualizacionInvalida)
		}
		_, cambiaNumero := req.DatosIdentificacion["numero_documento"]
		_, cambiaTipo := req.DatosIdentificacion["codigo_tipo_documento"]
		if cambiaNumero != cambiaTipo {
			return fmt.Errorf("%w: para cambiar la identificación envíe juntos numero_documento y codigo_tipo_documento", ErrActualizacionInvalida)
		}
		if len(req.DatosIdentificacion) == 1 {
			return fmt.Errorf("%w: datos_identificacion requiere al menos un campo para actualizar además de id", ErrActualizacionInvalida)
		}
		for key, raw := range req.DatosIdentificacion {
			if !identificacion[key] {
				return fmt.Errorf("%w: campo de identificación no actualizable: %s", ErrActualizacionInvalida, key)
			}
			switch key {
			case "numero_documento", "codigo_tipo_documento":
				var value string
				if json.Unmarshal(raw, &value) != nil || strings.TrimSpace(value) == "" || strings.ContainsAny(value, ",:") {
					return fmt.Errorf("%w: %s no es válido", ErrActualizacionInvalida, key)
				}
			case "id":
				// Validado arriba.
			case "fecha_expedicion":
				if _, err := normalizarFechaJSON(raw); err != nil {
					return fmt.Errorf("%w: fecha_expedicion: %v", ErrActualizacionInvalida, err)
				}
			case "digito_verificacion":
				var value float64
				if json.Unmarshal(raw, &value) != nil || value < 0 {
					return fmt.Errorf("%w: digito_verificacion debe ser numérico y no negativo", ErrActualizacionInvalida)
				}
			case "ciudad_expedicion":
				var value int
				if json.Unmarshal(raw, &value) != nil || value < 0 {
					return fmt.Errorf("%w: ciudad_expedicion debe ser un entero no negativo", ErrActualizacionInvalida)
				}
			}
		}
	}
	for key, raw := range req.Proveedor {
		if key != "descripcion_portafolio" {
			return fmt.Errorf("%w: campo de proveedor no actualizable: %s", ErrActualizacionInvalida, key)
		}
		var value string
		if json.Unmarshal(raw, &value) != nil {
			return fmt.Errorf("%w: descripcion_portafolio debe ser texto", ErrActualizacionInvalida)
		}
	}
	for key, raw := range req.ProveedorNatural {
		switch key {
		case "perfil_declarado_id":
			var id int
			if json.Unmarshal(raw, &id) != nil || id <= 0 {
				return fmt.Errorf("%w: perfil_declarado_id debe ser positivo", ErrActualizacionInvalida)
			}
		case "experiencia_laboral_meses", "experiencia_profesional_meses":
			var n int
			if json.Unmarshal(raw, &n) != nil || n < 0 {
				return fmt.Errorf("%w: %s debe ser un entero no negativo", ErrActualizacionInvalida, key)
			}
		default:
			return fmt.Errorf("%w: campo natural no actualizable: %s", ErrActualizacionInvalida, key)
		}
	}
	for key, raw := range req.ProveedorJuridico {
		switch key {
		case "nombre_comercial":
			var v string
			if json.Unmarshal(raw, &v) != nil || strings.TrimSpace(v) == "" {
				return fmt.Errorf("%w: nombre_comercial no puede estar vacío", ErrActualizacionInvalida)
			}
		case "matricula_mercantil":
			var v int
			if json.Unmarshal(raw, &v) != nil || v <= 0 {
				return fmt.Errorf("%w: matricula_mercantil debe ser positivo", ErrActualizacionInvalida)
			}
		case "fecha_constitucion", "fecha_renovacion":
			var v string
			if json.Unmarshal(raw, &v) != nil {
				return fmt.Errorf("%w: %s debe ser una fecha", ErrActualizacionInvalida, key)
			}
			if _, err := parseFecha(v); err != nil {
				return fmt.Errorf("%w: %s: %v", ErrActualizacionInvalida, key, err)
			}
		case "reporta_beneficios_finales", "cotiza_bolsa", "requiere_revisor_fiscal":
			var v bool
			if json.Unmarshal(raw, &v) != nil {
				return fmt.Errorf("%w: %s debe ser booleano", ErrActualizacionInvalida, key)
			}
		default:
			return fmt.Errorf("%w: campo jurídico no actualizable: %s", ErrActualizacionInvalida, key)
		}
	}
	return nil
}

func basePath(base, path string) string {
	return strings.TrimRight(strings.TrimSpace(base), "/") + path
}

func obtenerDataRaw(ctx context.Context, client interface {
	Get(context.Context, string, any) (int, error)
}, endpoint string) (json.RawMessage, int, error) {
	var respuesta apiEnvelopeRaw
	status, err := client.Get(ctx, endpoint, &respuesta)
	if err != nil {
		return nil, status, err
	}
	if len(respuesta.Data) == 0 || string(respuesta.Data) == "null" {
		return nil, status, errors.New("la respuesta del CRUD no contiene Data")
	}
	return respuesta.Data, status, nil
}

func rawObject(raw json.RawMessage) (map[string]json.RawMessage, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		return nil, fmt.Errorf("respuesta JSON no es un objeto: %w", err)
	}
	if object == nil {
		return nil, errors.New("respuesta JSON vacía")
	}
	return object, nil
}

func putRawObject(ctx context.Context, client interface {
	Put(context.Context, string, any, any) (int, error)
}, endpoint string, obj map[string]json.RawMessage) error {
	_, err := client.Put(ctx, endpoint, obj, nil)
	return err
}

func (s TercerosService) consultarIdentificacionesRaw(ctx context.Context, terceroID int) (json.RawMessage, error) {
	endpoint, err := url.Parse(basePath(s.BaseURL, "/v1/datos_identificacion"))
	if err != nil {
		return nil, err
	}
	query := endpoint.Query()
	query.Set("query", "TerceroId.Id:"+strconv.Itoa(terceroID)+",Activo:true")
	query.Set("limit", "100")
	query.Set("offset", "0")
	endpoint.RawQuery = query.Encode()
	registros, err := getAPIList[json.RawMessage](ctx, s.Client, endpoint.String())
	if err != nil {
		return nil, err
	}
	activos := make([]json.RawMessage, 0, len(registros))
	for _, registro := range registros {
		var item struct {
			Id int `json:"Id"`
		}
		if json.Unmarshal(registro, &item) == nil && item.Id > 0 {
			activos = append(activos, registro)
		}
	}
	resultado, err := json.Marshal(activos)
	if err != nil {
		return nil, err
	}
	return resultado, nil
}

func (s TercerosService) validarDocumentoDisponible(ctx context.Context, numero, codigo string, identificacionActualID int) error {
	base := strings.TrimRight(strings.TrimSpace(s.BaseURL), "/")
	for offset := 0; ; {
		endpoint, err := url.Parse(base + "/v1/datos_identificacion")
		if err != nil {
			return err
		}
		query := endpoint.Query()
		query.Set("query", "Numero:"+numero+",TipoDocumentoId.CodigoAbreviacion:"+codigo+",Activo:true")
		query.Set("limit", "100")
		query.Set("offset", strconv.Itoa(offset))
		endpoint.RawQuery = query.Encode()
		registros, err := getAPIList[identificacionResponse](ctx, s.Client, endpoint.String())
		if err != nil {
			return fmt.Errorf("validando unicidad del documento: %w", err)
		}
		for _, registro := range registros {
			if registro.Id > 0 && registro.Id != identificacionActualID && registro.Numero == numero && registro.TipoDocumento.Codigo == codigo && registro.Activo {
				return fmt.Errorf("%w: identificación_id=%d", ErrDocumentoDuplicado, registro.Id)
			}
		}
		if len(registros) < 100 {
			break
		}
		offset += len(registros)
	}
	return nil
}

func (s TercerosService) actualizarTercero(ctx context.Context, terceroID int, input map[string]json.RawMessage) error {
	input = clonarCampos(input)
	if raw, ok := input["fecha_nacimiento"]; ok {
		normalized, err := normalizarFechaJSON(raw)
		if err != nil {
			return fmt.Errorf("%w: fecha_nacimiento: %v", ErrActualizacionInvalida, err)
		}
		input["fecha_nacimiento"] = normalized
	}
	mapping := map[string]string{
		"nombre_completo": "NombreCompleto", "primer_nombre": "PrimerNombre", "segundo_nombre": "SegundoNombre",
		"primer_apellido": "PrimerApellido", "segundo_apellido": "SegundoApellido", "fecha_nacimiento": "FechaNacimiento", "lugar_origen": "LugarOrigen",
	}
	return actualizarMapa(ctx, s.Client, basePath(s.BaseURL, "/v1/tercero/"+strconv.Itoa(terceroID)), input, mapping)
}

func (s TercerosService) verificarIdentificacionTercero(ctx context.Context, terceroID, identificacionID int) error {
	if identificacionID <= 0 {
		return fmt.Errorf("%w: datos_identificacion.id debe ser positivo", ErrActualizacionInvalida)
	}
	endpoint := basePath(s.BaseURL, "/v1/datos_identificacion/"+strconv.Itoa(identificacionID))
	raw, _, err := obtenerDataRaw(ctx, s.Client, endpoint)
	if err != nil {
		return fmt.Errorf("consultando identificación %d: %w", identificacionID, err)
	}
	existing, err := rawObject(raw)
	if err != nil {
		return err
	}
	var relacion struct {
		Id int `json:"Id"`
	}
	if relationRaw := existing["TerceroId"]; len(relationRaw) > 0 {
		_ = json.Unmarshal(relationRaw, &relacion)
	}
	if relacion.Id != terceroID {
		return fmt.Errorf("%w: la identificación %d no pertenece al tercero %d", ErrActualizacionInvalida, identificacionID, terceroID)
	}
	return nil
}

func (s TercerosService) actualizarIdentificacion(ctx context.Context, terceroID int, input map[string]json.RawMessage) error {
	var id int
	if raw, ok := input["id"]; !ok || json.Unmarshal(raw, &id) != nil || id <= 0 {
		return fmt.Errorf("%w: datos_identificacion.id es obligatorio", ErrActualizacionInvalida)
	}
	mapping := map[string]string{
		"digito_verificacion": "DigitoVerificacion", "ciudad_expedicion": "CiudadExpedicion", "fecha_expedicion": "FechaExpedicion",
	}
	for key := range input {
		if key == "id" || key == "numero_documento" || key == "codigo_tipo_documento" {
			continue
		}
		if _, ok := mapping[key]; !ok {
			return fmt.Errorf("%w: campo de identificación no actualizable: %s", ErrActualizacionInvalida, key)
		}
	}
	_, cambiaNumero := input["numero_documento"]
	_, cambiaTipo := input["codigo_tipo_documento"]
	if cambiaNumero != cambiaTipo {
		return fmt.Errorf("%w: para cambiar la identificación envíe juntos numero_documento y codigo_tipo_documento", ErrActualizacionInvalida)
	}
	endpoint := basePath(s.BaseURL, "/v1/datos_identificacion/"+strconv.Itoa(id))
	existingRaw, _, err := obtenerDataRaw(ctx, s.Client, endpoint)
	if err != nil {
		return fmt.Errorf("consultando identificación %d: %w", id, err)
	}
	existing, err := rawObject(existingRaw)
	if err != nil {
		return err
	}
	var relacion struct {
		Id int `json:"Id"`
	}
	if raw := existing["TerceroId"]; len(raw) > 0 {
		_ = json.Unmarshal(raw, &relacion)
	}
	if relacion.Id != terceroID {
		return fmt.Errorf("%w: la identificación %d no pertenece al tercero %d", ErrActualizacionInvalida, id, terceroID)
	}
	fields := make(map[string]json.RawMessage, len(input)-1)
	for key, value := range input {
		if key != "id" && key != "numero_documento" && key != "codigo_tipo_documento" {
			fields[key] = value
		}
	}
	if cambiaNumero && cambiaTipo {
		var numero, codigo string
		if json.Unmarshal(input["numero_documento"], &numero) != nil || strings.TrimSpace(numero) == "" || strings.ContainsAny(numero, ",:") {
			return fmt.Errorf("%w: numero_documento no es válido", ErrActualizacionInvalida)
		}
		if json.Unmarshal(input["codigo_tipo_documento"], &codigo) != nil || strings.TrimSpace(codigo) == "" || strings.ContainsAny(codigo, ",:") {
			return fmt.Errorf("%w: codigo_tipo_documento no es válido", ErrActualizacionInvalida)
		}
		if err := s.validarDocumentoDisponible(ctx, numero, codigo, id); err != nil {
			return err
		}
		tipoDocumentoID, err := s.ResolverTipoDocumento(ctx, codigo)
		if err != nil {
			return fmt.Errorf("resolviendo el nuevo tipo documental: %w", err)
		}
		numeroRaw, _ := json.Marshal(strings.TrimSpace(numero))
		tipoRaw, _ := json.Marshal(map[string]int{"Id": tipoDocumentoID})
		existing["Numero"] = numeroRaw
		existing["TipoDocumentoId"] = tipoRaw
	}
	if raw, ok := fields["fecha_expedicion"]; ok {
		normalized, err := normalizarFechaJSON(raw)
		if err != nil {
			return fmt.Errorf("%w: fecha_expedicion: %v", ErrActualizacionInvalida, err)
		}
		fields["fecha_expedicion"] = normalized
	}
	if err := mergeCampos(existing, fields, mapping); err != nil {
		return err
	}
	return putRawObject(ctx, s.Client, endpoint, existing)
}

func (s GestionPersonasService) actualizarProveedor(ctx context.Context, id int, input map[string]json.RawMessage) error {
	return actualizarMapa(ctx, s.Client, basePath(s.BaseURL, "/v1/proveedor/"+strconv.Itoa(id)), input,
		map[string]string{"descripcion_portafolio": "DescripcionPortafolio"})
}

func (s GestionPersonasService) actualizarProveedorNatural(ctx context.Context, id int, input map[string]json.RawMessage) error {
	allowed := map[string]string{
		"experiencia_laboral_meses": "ExperienciaLaboralMeses", "experiencia_profesional_meses": "ExperienciaProfesionalMeses",
	}
	existingRaw, _, err := obtenerDataRaw(ctx, s.Client, basePath(s.BaseURL, "/v1/proveedor_natural/"+strconv.Itoa(id)))
	if err != nil {
		return err
	}
	existing, err := rawObject(existingRaw)
	if err != nil {
		return err
	}
	for key, value := range input {
		switch key {
		case "perfil_declarado_id":
			var profileID int
			if err := json.Unmarshal(value, &profileID); err != nil || profileID <= 0 {
				return fmt.Errorf("%w: perfil_declarado_id debe ser positivo", ErrActualizacionInvalida)
			}
			nested, _ := json.Marshal(map[string]int{"Id": profileID})
			existing["PerfilDeclarado"] = nested
		case "experiencia_laboral_meses", "experiencia_profesional_meses":
			var months int
			if err := json.Unmarshal(value, &months); err != nil || months < 0 {
				return fmt.Errorf("%w: %s debe ser un entero no negativo", ErrActualizacionInvalida, key)
			}
			existing[allowed[key]] = value
		default:
			return fmt.Errorf("%w: campo natural no actualizable: %s", ErrActualizacionInvalida, key)
		}
	}
	return putRawObject(ctx, s.Client, basePath(s.BaseURL, "/v1/proveedor_natural/"+strconv.Itoa(id)), existing)
}

func (s GestionPersonasService) actualizarProveedorJuridico(ctx context.Context, id int, input map[string]json.RawMessage) error {
	mapping := map[string]string{
		"nombre_comercial": "NombreComercial", "matricula_mercantil": "MatriculaMercantil",
		"reporta_beneficios_finales": "ReportaBeneficiosFinales", "cotiza_bolsa": "CotizaBolsa", "requiere_revisor_fiscal": "RequiereRevisorFiscal",
	}
	existingRaw, _, err := obtenerDataRaw(ctx, s.Client, basePath(s.BaseURL, "/v1/proveedor_juridico/"+strconv.Itoa(id)))
	if err != nil {
		return err
	}
	existing, err := rawObject(existingRaw)
	if err != nil {
		return err
	}
	for key, value := range input {
		if key == "fecha_constitucion" || key == "fecha_renovacion" {
			var stringValue string
			if err := json.Unmarshal(value, &stringValue); err != nil {
				return fmt.Errorf("%w: %s debe ser una fecha", ErrActualizacionInvalida, key)
			}
			fecha, err := parseFecha(stringValue)
			if err != nil {
				return fmt.Errorf("%w: %s: %v", ErrActualizacionInvalida, key, err)
			}
			marshaled, _ := json.Marshal(fecha)
			if key == "fecha_constitucion" {
				existing["FechaConstitucion"] = marshaled
			} else {
				existing["FechaRenovacion"] = marshaled
			}
			continue
		}
		if _, ok := mapping[key]; !ok {
			return fmt.Errorf("%w: campo jurídico no actualizable: %s", ErrActualizacionInvalida, key)
		}
		if key == "nombre_comercial" {
			var v string
			if json.Unmarshal(value, &v) != nil || strings.TrimSpace(v) == "" {
				return fmt.Errorf("%w: nombre_comercial no puede estar vacío", ErrActualizacionInvalida)
			}
		}
		if key == "matricula_mercantil" {
			var v int
			if json.Unmarshal(value, &v) != nil || v <= 0 {
				return fmt.Errorf("%w: matricula_mercantil debe ser positivo", ErrActualizacionInvalida)
			}
		}
		if strings.HasPrefix(key, "reporta_") || key == "cotiza_bolsa" || key == "requiere_revisor_fiscal" {
			var v bool
			if json.Unmarshal(value, &v) != nil {
				return fmt.Errorf("%w: %s debe ser booleano", ErrActualizacionInvalida, key)
			}
		}
		existing[mapping[key]] = value
	}
	return putRawObject(ctx, s.Client, basePath(s.BaseURL, "/v1/proveedor_juridico/"+strconv.Itoa(id)), existing)
}

func actualizarMapa(ctx context.Context, client interface {
	Get(context.Context, string, any) (int, error)
	Put(context.Context, string, any, any) (int, error)
}, endpoint string, input map[string]json.RawMessage, mapping map[string]string) error {
	for key := range input {
		if _, ok := mapping[key]; !ok {
			return fmt.Errorf("%w: campo no actualizable: %s", ErrActualizacionInvalida, key)
		}
	}
	raw, _, err := obtenerDataRaw(ctx, client, endpoint)
	if err != nil {
		return err
	}
	obj, err := rawObject(raw)
	if err != nil {
		return err
	}
	if err := mergeCampos(obj, input, mapping); err != nil {
		return err
	}
	return putRawObject(ctx, client, endpoint, obj)
}

func clonarCampos(input map[string]json.RawMessage) map[string]json.RawMessage {
	copy := make(map[string]json.RawMessage, len(input))
	for key, value := range input {
		copy[key] = value
	}
	return copy
}

func normalizarFechaJSON(raw json.RawMessage) (json.RawMessage, error) {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, errors.New("debe ser una cadena de fecha")
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return json.RawMessage("null"), nil
	}
	fecha, err := parseFecha(value)
	if err != nil {
		return nil, err
	}
	return json.Marshal(fecha)
}

func mergeCampos(target map[string]json.RawMessage, input map[string]json.RawMessage, mapping map[string]string) error {
	for key, value := range input {
		external, ok := mapping[key]
		if !ok {
			return fmt.Errorf("%w: campo no actualizable: %s", ErrActualizacionInvalida, key)
		}
		target[external] = value
	}
	return nil
}
