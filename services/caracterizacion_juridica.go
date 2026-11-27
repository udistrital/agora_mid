package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/udistrital/agora_mid/helper"
	"github.com/udistrital/agora_mid/models"
	"github.com/udistrital/agora_mid/responses"
)

const (
	endpointProveedorJuridico = "/v1/proveedor_juridico"
	endpointRepresentacion    = "/v1/representacion"

	maxCodigoPostalJuridico = 16

	grupoProveedorJuridicoPorDefecto = "PROVEEDOR_JURIDICO"
	grupoProcedenciaPorDefecto       = "PROCEDENCIA"
	grupoCargoPorDefecto             = "Grupo_5"
	tipoRegimenContributivoDefecto   = "REG_CONT"
	tipoTipoConformacionDefecto      = "TIPO_CONFORMACION"
)

// ParametrosJuridica extiende la validación de parámetros con la resolución
// por código, necesaria para regímenes y tipos de conformación.
type ParametrosJuridica interface {
	ParametrosCaracterizacion
	ResolverParametroPorCodigo(
		ctx context.Context,
		codigo string,
		tipo string,
	) (id int64, existe bool, ambiguo bool, err error)
}

// RepresentanteCatalogos resuelve el tipo de documento y el tercero dueño de
// una identificación. Lo implementa CatalogosService.
type RepresentanteCatalogos interface {
	TipoDocumentoExiste(ctx context.Context, id int64) (bool, error)
	TipoDocumentoPorCodigo(
		ctx context.Context,
		codigo string,
	) (id int64, existe bool, ambiguo bool, err error)
	TipoDocumentoPorID(
		ctx context.Context,
		id int64,
	) (codigo string, existe bool, err error)
	TerceroPorDocumento(
		ctx context.Context,
		tipoDocumentoID int64,
		numero string,
	) (map[string]any, bool, error)
	IdentificacionActiva(
		ctx context.Context,
		terceroID int64,
	) (map[string]any, bool, error)
}

// CaracterizacionJuridicaResult agrupa los datos materializados y los avisos.
type CaracterizacionJuridicaResult struct {
	Data    models.CaracterizacionJuridicaData
	Avisos  []responses.RelationError
	Parcial bool
}

// CaracterizacionJuridicaService implementa el dominio 2 para persona jurídica.
// Reutiliza el núcleo genérico de caracterización del dominio natural para
// info complementaria, perfil, contacto, actividad, proveedor y errores.
type CaracterizacionJuridicaService struct {
	Base                      CaracterizacionNaturalService
	Parametros                ParametrosJuridica
	Representantes            RepresentanteCatalogos
	TipoContribuyenteJuridica int
	TipoContribuyenteNatural  int
	TipoProveedorJuridico     int
	TipoProveedorNatural      int

	TipoParametroRegimenContributivo string
	TipoParametroTipoConformacion    string
	GrupoProcedencia                 string
	GrupoCargo                       string
	GrupoProveedorJuridico           string
}

// NewCaracterizacionJuridicaService construye el servicio del dominio 2
// jurídico. Los valores vacíos usan los códigos por defecto del repositorio.
func NewCaracterizacionJuridicaService(
	catalogos CatalogosCaracterizacion,
	representantes RepresentanteCatalogos,
	parametros ParametrosJuridica,
	tipoContribuyenteJuridica int,
	tipoContribuyenteNatural int,
	tipoProveedorJuridico int,
	tipoProveedorNatural int,
	tipoParametroCIIU string,
	tipoParametroCanalContacto string,
	tipoParametroRegimenContributivo string,
	tipoParametroTipoConformacion string,
	grupoProcedencia string,
	grupoCargo string,
	grupoProveedorJuridico string,
) CaracterizacionJuridicaService {
	if tipoProveedorJuridico <= 0 {
		tipoProveedorJuridico = 2
	}
	if tipoProveedorNatural <= 0 {
		tipoProveedorNatural = 1
	}
	if strings.TrimSpace(tipoParametroRegimenContributivo) == "" {
		tipoParametroRegimenContributivo = tipoRegimenContributivoDefecto
	}
	if strings.TrimSpace(tipoParametroTipoConformacion) == "" {
		tipoParametroTipoConformacion = tipoTipoConformacionDefecto
	}
	if strings.TrimSpace(grupoProcedencia) == "" {
		grupoProcedencia = grupoProcedenciaPorDefecto
	}
	if strings.TrimSpace(grupoCargo) == "" {
		grupoCargo = grupoCargoPorDefecto
	}
	if strings.TrimSpace(grupoProveedorJuridico) == "" {
		grupoProveedorJuridico = grupoProveedorJuridicoPorDefecto
	}
	if representantes == nil {
		if catalogosConcretos, ok := catalogos.(RepresentanteCatalogos); ok {
			representantes = catalogosConcretos
		}
	}
	base := NewCaracterizacionNaturalService(
		catalogos, parametros, tipoContribuyenteNatural, tipoProveedorNatural,
		tipoParametroCIIU, tipoParametroCanalContacto,
	)
	return CaracterizacionJuridicaService{
		Base:                             base,
		Parametros:                       parametros,
		Representantes:                   representantes,
		TipoContribuyenteJuridica:        tipoContribuyenteJuridica,
		TipoContribuyenteNatural:         tipoContribuyenteNatural,
		TipoProveedorJuridico:            tipoProveedorJuridico,
		TipoProveedorNatural:             tipoProveedorNatural,
		TipoParametroRegimenContributivo: strings.TrimSpace(tipoParametroRegimenContributivo),
		TipoParametroTipoConformacion:    strings.TrimSpace(tipoParametroTipoConformacion),
		GrupoProcedencia:                 strings.TrimSpace(grupoProcedencia),
		GrupoCargo:                       strings.TrimSpace(grupoCargo),
		GrupoProveedorJuridico:           strings.TrimSpace(grupoProveedorJuridico),
	}
}

type planProveedorJuridico struct {
	procedenciaID             *int64
	codigoProcedencia         string
	ciudadOrigenID            *int64
	paisOrigenID              *int64
	codigoPostal              *string
	tipoConformacionID        *int64
	codigoTipoConformacion    string
	regimenContributivoID     *int64
	codigoRegimenContributivo string
	exclusividadProducto      *bool
	pyme                      *bool
	registroMercantil         *bool
	sujetoRetencion           *bool
	agenteRetenedor           *bool
	responsableIca            *bool
	responsableIva            *bool
}

type planRepresentante struct {
	terceroID           int64
	tercero             map[string]any
	tipoDocumentoID     int64
	codigoTipoDocumento string
	numero              string
	cargoID             int64
	codigoCargo         string
}

type planCaracterizacionJuridica struct {
	info           []planInfoComplementaria
	juridico       *planProveedorJuridico
	representante  *planRepresentante
	perfiles       []models.PerfilProveedorInput
	contactos      []models.ContactoProveedorInput
	actividades    []models.ActividadEconomicaProveedorInput
	descripcion    *string
	consentimiento *bool
	avisos         []responses.RelationError
}

// Registrar ejecuta el upsert parcial de la caracterización jurídica.
func (s CaracterizacionJuridicaService) Registrar(
	ctx context.Context,
	in models.RegistroCaracterizacionJuridica,
	terceroID int64,
	verb string,
) (CaracterizacionJuridicaResult, error) {
	if terceroID <= 0 {
		return CaracterizacionJuridicaResult{}, errorCaracterizacion(400,
			"TerceroId debe ser un entero positivo", nil)
	}
	if err := s.validarConfiguracion(); err != nil {
		return CaracterizacionJuridicaResult{}, err
	}

	plan, err := s.preparar(ctx, in)
	if err != nil {
		return CaracterizacionJuridicaResult{}, err
	}

	unlock := func() {}
	if s.Base.mutex != nil {
		unlock = s.Base.mutex.lock(strconv.FormatInt(terceroID, 10))
	}
	defer unlock()

	tercero, err := s.Base.consultarTercero(ctx, terceroID)
	if err != nil {
		return CaracterizacionJuridicaResult{}, err
	}
	if err := s.validarTerceroJuridico(ctx, tercero); err != nil {
		return CaracterizacionJuridicaResult{}, err
	}

	resultado := nuevoResultadoCaracterizacionJuridica(terceroID, plan)
	var existentesInfo *existentesInfoTercero
	if len(plan.info) > 0 {
		existentesInfo, err = s.Base.cargarInfoExistentes(ctx, terceroID)
		if err != nil {
			return resultado, err
		}
	}

	proveedor, accion, avisosProveedor, err := s.resolverProveedor(
		ctx, terceroID, verb, plan,
	)
	resultado.Avisos = append(resultado.Avisos, avisosProveedor...)
	if err != nil {
		if proveedor == nil || idDeMapa(proveedor) <= 0 {
			return resultado, err
		}
		resultado.Avisos = append(resultado.Avisos,
			avisoDeErrorCaracterizacion(err, "Proveedor", "gestion_proveedores_crud"))
		resultado.Parcial = true
	}
	proveedorID := idDeMapa(proveedor)
	if proveedorID <= 0 {
		return resultado, errorDependencia(ctx,
			errors.New("el proveedor no tiene un Id utilizable"))
	}
	resultado.Data.ProveedorId = proveedorID
	resultado.Data.Proveedor = proveedor
	registrarResultado(&resultado.Data.Resultado, "Proveedor", proveedorID, accion)

	s.escribirSecciones(ctx, terceroID, proveedorID, plan, existentesInfo, &resultado)
	return resultado, nil
}

// RegistrarPorProveedor ejecuta la caracterización usando el Id del proveedor.
func (s CaracterizacionJuridicaService) RegistrarPorProveedor(
	ctx context.Context,
	in models.RegistroCaracterizacionJuridica,
	proveedorID int64,
	verb string,
) (CaracterizacionJuridicaResult, error) {
	if proveedorID <= 0 {
		return CaracterizacionJuridicaResult{}, errorCaracterizacion(400,
			"proveedorId debe ser un entero positivo", nil)
	}
	if err := s.validarConfiguracion(); err != nil {
		return CaracterizacionJuridicaResult{}, err
	}

	plan, err := s.preparar(ctx, in)
	if err != nil {
		return CaracterizacionJuridicaResult{}, err
	}

	proveedor, err := s.consultarProveedorPorID(ctx, proveedorID)
	if err != nil {
		return CaracterizacionJuridicaResult{}, err
	}
	terceroID := idDeRelacion(proveedor["TerceroId"])
	if terceroID <= 0 {
		return CaracterizacionJuridicaResult{}, errorDependencia(ctx,
			errors.New("el proveedor no contiene un TerceroId válido"))
	}
	if in.TerceroId != 0 && in.TerceroId != terceroID {
		return CaracterizacionJuridicaResult{}, errorCaracterizacion(400,
			"TerceroId del cuerpo no coincide con el proveedor indicado", nil)
	}

	unlock := func() {}
	if s.Base.mutex != nil {
		unlock = s.Base.mutex.lock(strconv.FormatInt(terceroID, 10))
	}
	defer unlock()

	proveedor, err = s.consultarProveedorPorID(ctx, proveedorID)
	if err != nil {
		return CaracterizacionJuridicaResult{}, err
	}
	tercero, err := s.Base.consultarTercero(ctx, terceroID)
	if err != nil {
		return CaracterizacionJuridicaResult{}, err
	}
	if err := s.validarTerceroJuridico(ctx, tercero); err != nil {
		return CaracterizacionJuridicaResult{}, err
	}

	resultado := nuevoResultadoCaracterizacionJuridica(terceroID, plan)
	var existentesInfo *existentesInfoTercero
	if len(plan.info) > 0 {
		existentesInfo, err = s.Base.cargarInfoExistentes(ctx, terceroID)
		if err != nil {
			return resultado, err
		}
	}

	planBase := planCaracterizacion{
		descripcion:    plan.descripcion,
		consentimiento: plan.consentimiento,
	}
	proveedor, accion, avisosProveedor, err := s.Base.actualizarProveedorConocido(
		ctx, proveedor, planBase,
	)
	resultado.Avisos = append(resultado.Avisos, avisosProveedor...)
	if err != nil {
		if proveedor == nil || idDeMapa(proveedor) <= 0 {
			return resultado, err
		}
		resultado.Avisos = append(resultado.Avisos,
			avisoDeErrorCaracterizacion(err, "Proveedor", "gestion_proveedores_crud"))
		resultado.Parcial = true
	}
	resultado.Data.ProveedorId = proveedorID
	resultado.Data.Proveedor = proveedor
	registrarResultado(&resultado.Data.Resultado, "Proveedor", proveedorID, accion)

	s.escribirSecciones(ctx, terceroID, proveedorID, plan, existentesInfo, &resultado)
	return resultado, nil
}

func (s CaracterizacionJuridicaService) validarConfiguracion() error {
	if s.Base.Catalogos == nil || s.Base.Parametros == nil || s.Parametros == nil {
		return errorCaracterizacion(500,
			"El servicio de caracterización jurídica no está configurado", nil)
	}
	if s.TipoContribuyenteJuridica <= 0 {
		return errorCaracterizacion(500,
			"No está configurado el tipo de contribuyente P_JURIDICA", nil)
	}
	if s.TipoContribuyenteNatural <= 0 {
		return errorCaracterizacion(500,
			"No está configurado el tipo de contribuyente P_NATURAL para el representante", nil)
	}
	if s.Representantes == nil {
		return errorCaracterizacion(500,
			"No está configurado el resolvedor de representantes", nil)
	}
	return nil
}

func nuevoResultadoCaracterizacionJuridica(
	terceroID int64,
	plan planCaracterizacionJuridica,
) CaracterizacionJuridicaResult {
	resultado := CaracterizacionJuridicaResult{
		Avisos: plan.avisos,
		Data: models.CaracterizacionJuridicaData{
			TerceroId:                   terceroID,
			InfoComplementariaTercero:   []models.InfoComplementariaTerceroRegistrada{},
			PerfilProveedor:             []models.EntidadId{},
			Contacto:                    []models.EntidadId{},
			ActividadEconomicaProveedor: []models.EntidadId{},
			Resultado: models.Resultado{
				Creados:      []models.EntidadId{},
				Actualizados: []models.EntidadId{},
				Reutilizados: []models.EntidadId{},
			},
		},
	}
	if plan.representante != nil {
		resultado.Data.Representante = respuestaRepresentante(plan.representante)
	}
	return resultado
}

func respuestaRepresentante(plan *planRepresentante) map[string]any {
	if plan == nil {
		return nil
	}
	nombre, _ := plan.tercero["NombreCompleto"].(string)
	respuesta := map[string]any{
		"TerceroId":           plan.terceroID,
		"TipoDocumentoId":     plan.tipoDocumentoID,
		"CodigoTipoDocumento": plan.codigoTipoDocumento,
		"Numero":              plan.numero,
		"CargoId":             plan.cargoID,
		"CodigoCargo":         plan.codigoCargo,
	}
	if strings.TrimSpace(nombre) != "" {
		respuesta["NombreCompleto"] = nombre
	}
	return respuesta
}

func (s CaracterizacionJuridicaService) validarTerceroJuridico(
	ctx context.Context,
	tercero map[string]any,
) error {
	activo, existeActivo := tercero["Activo"].(bool)
	if !existeActivo {
		return errorDependencia(ctx,
			errors.New("la respuesta del tercero no contiene Activo"))
	}
	if !activo {
		return errorCaracterizacion(409, "El tercero está inactivo", nil)
	}
	tipo := idDeRelacion(tercero["TipoContribuyenteId"])
	if tipo <= 0 {
		return errorDependencia(ctx,
			errors.New("la respuesta del tercero no contiene un TipoContribuyenteId válido"))
	}
	if tipo != int64(s.TipoContribuyenteJuridica) {
		return errorCaracterizacion(409,
			"El tercero no corresponde a una persona jurídica", nil)
	}
	return nil
}

func (s CaracterizacionJuridicaService) consultarProveedorPorID(
	ctx context.Context,
	proveedorID int64,
) (map[string]any, error) {
	var registro map[string]any
	ok, status, err := GetOneProveedores(ctx,
		fmt.Sprintf("%s/%d", endpointProveedor, proveedorID), &registro)
	if err != nil {
		if status == 404 {
			return nil, errorCaracterizacion(404,
				"No existe un proveedor con el Id indicado", nil)
		}
		return nil, errorLecturaCaracterizacion(ctx, status,
			"No fue posible consultar el proveedor", err)
	}
	if !ok {
		if status == 404 {
			return nil, errorCaracterizacion(404,
				"No existe un proveedor con el Id indicado", nil)
		}
		return nil, errorLecturaCaracterizacion(ctx, status,
			"No fue posible consultar el proveedor", nil)
	}
	registro = desenvolverData(registro)
	if idDeMapa(registro) != proveedorID {
		return nil, errorDependencia(ctx,
			errors.New("la respuesta del proveedor no contiene el Id esperado"))
	}
	tipo := idDeRelacion(registro["TipoProveedor"])
	if tipo <= 0 {
		return nil, errorDependencia(ctx,
			errors.New("la respuesta del proveedor no contiene TipoProveedor"))
	}
	if tipo != int64(s.TipoProveedorJuridico) {
		return nil, errorCaracterizacion(409,
			"El proveedor no corresponde a persona jurídica", nil)
	}
	activo, existe := registro["Activo"].(bool)
	if !existe {
		return nil, errorDependencia(ctx,
			errors.New("la respuesta del proveedor no contiene Activo"))
	}
	if !activo {
		return nil, errorCaracterizacion(409, "El proveedor está inactivo", nil)
	}
	return registro, nil
}

func (s CaracterizacionJuridicaService) resolverProveedor(
	ctx context.Context,
	terceroID int64,
	verb string,
	plan planCaracterizacionJuridica,
) (map[string]any, string, []responses.RelationError, error) {
	filtros := url.Values{
		"query": {fmt.Sprintf("TerceroId:%d", terceroID)},
		"limit": {"10"},
	}
	var registros []map[string]any
	ok, status, err := GetAllProveedoresMapas(ctx, endpointProveedor, filtros, &registros)
	if err != nil {
		return nil, "", nil, errorDependencia(ctx, err)
	}
	if !ok {
		return nil, "", nil, errorDependencia(ctx,
			fmt.Errorf("consulta de proveedor con estado %d", status))
	}
	activos, inactivos, err := separarRegistrosProveedor(ctx, registros)
	if err != nil {
		return nil, "", nil, err
	}
	planBase := planCaracterizacion{
		descripcion:    plan.descripcion,
		consentimiento: plan.consentimiento,
	}
	if len(activos) > 1 {
		return nil, "", nil, errorCaracterizacion(409,
			"El tercero tiene más de un proveedor activo; no es posible resolverlo de forma segura", nil)
	}
	if len(activos) == 1 {
		proveedor := activos[0]
		tipo := idDeRelacion(proveedor["TipoProveedor"])
		if tipo <= 0 {
			return nil, "", nil, errorDependencia(ctx,
				errors.New("la respuesta del proveedor no contiene TipoProveedor"))
		}
		if tipo != int64(s.TipoProveedorJuridico) {
			return nil, "", nil, errorCaracterizacion(409,
				"El proveedor existente no corresponde a persona jurídica", nil)
		}
		return s.Base.actualizarProveedorConocido(ctx, proveedor, planBase)
	}
	if len(inactivos) > 0 {
		return nil, "", nil, errorCaracterizacion(409,
			"El proveedor asociado al tercero está inactivo", nil)
	}
	if verb == helper.WriteVerbUpdate {
		return nil, "", nil, errorCaracterizacion(404,
			"No existe un proveedor asociado al tercero", nil)
	}

	consentimiento := true
	if plan.consentimiento != nil {
		consentimiento = *plan.consentimiento
	}
	payload := aplicarUsuarioModificacion(map[string]any{
		"TipoProveedor":  s.TipoProveedorJuridico,
		"TerceroId":      terceroID,
		"Consentimiento": consentimiento,
		"Activo":         true,
	}, terceroID)
	if plan.descripcion != nil {
		payload["DescripcionPortafolio"] = *plan.descripcion
	}
	var respuesta map[string]any
	ok, status, err = PostProveedores(ctx, endpointProveedor, payload, &respuesta)
	if err != nil {
		return nil, "", nil, errorEscrituraCaracterizacion(ctx, status, err, "crear el proveedor")
	}
	if !ok {
		return nil, "", nil, errorEscrituraCaracterizacion(ctx, status,
			fmt.Errorf("el CRUD respondió con estado %d", status), "crear el proveedor")
	}
	respuesta = desenvolverData(respuesta)
	proveedorID := idDeMapa(respuesta)
	if proveedorID <= 0 {
		return nil, "", nil, errorCaracterizacion(502,
			"La creación del proveedor no devolvió un Id utilizable", nil)
	}
	proveedor := clonarPayload(payload)
	proveedor["Id"] = proveedorID
	proveedor["Activo"] = true
	avisos := s.Base.avisoProveedorDuplicado(ctx, terceroID)
	return proveedor, "creado", avisos, nil
}

func (s CaracterizacionJuridicaService) preparar(
	ctx context.Context,
	in models.RegistroCaracterizacionJuridica,
) (planCaracterizacionJuridica, error) {
	plan := planCaracterizacionJuridica{
		perfiles:       in.PerfilProveedor,
		contactos:      in.Contacto,
		actividades:    in.ActividadEconomicaProveedor,
		descripcion:    in.DescripcionPortafolio,
		consentimiento: in.Consentimiento,
	}

	if plan.descripcion != nil {
		texto, err := normalizarDescripcionPortafolio(*plan.descripcion)
		if err != nil {
			return plan, err
		}
		plan.descripcion = &texto
	}

	representante, avisos, err := s.prepararRepresentante(ctx, in.Representante)
	if err != nil {
		return plan, err
	}
	plan.representante = representante
	plan.avisos = append(plan.avisos, avisos...)

	juridico, avisos, err := s.prepararProveedorJuridico(ctx, in.ProveedorJuridico)
	if err != nil {
		return plan, err
	}
	plan.juridico = juridico
	plan.avisos = append(plan.avisos, avisos...)

	idsInfo := make(map[int64]struct{}, len(in.InfoComplementariaTercero))
	gruposOpcionVistos := make(map[string]struct{}, len(in.InfoComplementariaTercero))
	for i, item := range in.InfoComplementariaTercero {
		if item.CodigoInfoComplementaria != nil &&
			strings.EqualFold(strings.TrimSpace(*item.CodigoInfoComplementaria), "CARGO_REPRESENTANTE") {
			return plan, errorCaracterizacion(400,
				fmt.Sprintf("InfoComplementariaTercero[%d]: el cargo del representante se envía en Representante.CargoId o Representante.CodigoCargo", i), nil)
		}
		preparado, err := s.Base.prepararInfoComplementaria(ctx, item, i)
		if err != nil {
			return plan, err
		}
		if preparado.ambiguo {
			plan.avisos = append(plan.avisos, responses.RelationError{
				Path:    "InfoComplementariaTercero",
				Service: "terceros_crud",
				Message: fmt.Sprintf("El código %q del grupo %q tiene varias coincidencias activas; se usó la de menor Id", preparado.codigo, preparado.codigoGrupo),
				Kind:    "ambiguous_match",
			})
		}
		if _, repetido := idsInfo[preparado.infoComplementariaID]; repetido {
			return plan, errorCaracterizacion(400,
				fmt.Sprintf("InfoComplementariaTercero[%d]: el código %q del grupo %q está repetido", i, preparado.codigo, preparado.codigoGrupo), nil)
		}
		idsInfo[preparado.infoComplementariaID] = struct{}{}
		if preparado.esOpcion {
			clave := strings.ToUpper(strings.TrimSpace(preparado.grupoOpcion))
			if _, repetido := gruposOpcionVistos[clave]; repetido {
				return plan, errorCaracterizacion(400,
					fmt.Sprintf("InfoComplementariaTercero[%d]: solo se puede enviar una opción del grupo %q", i, preparado.grupoOpcion), nil)
			}
			gruposOpcionVistos[clave] = struct{}{}
		}
		plan.info = append(plan.info, preparado)
	}

	if plan.representante != nil {
		cargo, aviso, err := s.prepararCargoRepresentante(ctx, plan.representante)
		if err != nil {
			return plan, err
		}
		if aviso != nil {
			plan.avisos = append(plan.avisos, *aviso)
		}
		plan.info = append(plan.info, cargo)
	}

	if err := s.completarOrigenDesdeDireccion(ctx, &plan); err != nil {
		return plan, err
	}
	if err := validarPrincipalEnActividades(plan.info, plan.actividades); err != nil {
		return plan, err
	}

	perfilesVistos := make(map[int64]struct{}, len(plan.perfiles))
	for i := range plan.perfiles {
		perfil := &plan.perfiles[i]
		if perfil.Id != nil && *perfil.Id <= 0 {
			return plan, errorCaracterizacion(400,
				fmt.Sprintf("PerfilProveedor[%d].Id debe ser un entero positivo", i), nil)
		}
		tienePerfilID := perfil.PerfilId > 0
		tieneCodigoPerfil := perfil.CodigoPerfil != nil && strings.TrimSpace(*perfil.CodigoPerfil) != ""
		if !tienePerfilID && !tieneCodigoPerfil {
			return plan, errorCaracterizacion(400,
				fmt.Sprintf("PerfilProveedor[%d]: debe traer PerfilId o CodigoPerfil", i), nil)
		}
		if tieneCodigoPerfil {
			codigo := strings.TrimSpace(*perfil.CodigoPerfil)
			if contieneControl(codigo) {
				return plan, errorCaracterizacion(400,
					fmt.Sprintf("PerfilProveedor[%d].CodigoPerfil no admite caracteres de control", i), nil)
			}
			detalle, ok, err := s.Base.Catalogos.ResolverInfoComplementaria(ctx, codigo, codigoGrupoPerfil)
			if err != nil {
				return plan, errorDependencia(ctx, err)
			}
			if !ok || !detalle.Activo {
				return plan, errorCaracterizacion(400,
					fmt.Sprintf("PerfilProveedor[%d].CodigoPerfil %q no existe, está inactivo o no pertenece al grupo %s", i, codigo, codigoGrupoPerfil), nil)
			}
			if tienePerfilID && perfil.PerfilId != detalle.Id {
				return plan, errorCaracterizacion(400,
					fmt.Sprintf("PerfilProveedor[%d]: PerfilId y CodigoPerfil no coinciden", i), nil)
			}
			perfil.PerfilId = detalle.Id
		} else {
			existe, err := s.Base.Catalogos.InfoComplementariaDeGrupo(ctx, perfil.PerfilId, codigoGrupoPerfil)
			if err != nil {
				return plan, errorDependencia(ctx, err)
			}
			if !existe {
				return plan, errorCaracterizacion(400,
					fmt.Sprintf("PerfilProveedor[%d].PerfilId no existe, está inactivo o no pertenece al grupo %s", i, codigoGrupoPerfil), nil)
			}
		}
		if _, repetido := perfilesVistos[perfil.PerfilId]; repetido {
			return plan, errorCaracterizacion(400,
				fmt.Sprintf("PerfilProveedor[%d].PerfilId está repetido", i), nil)
		}
		perfilesVistos[perfil.PerfilId] = struct{}{}
	}

	contactosVistos := make(map[string]struct{}, len(plan.contactos))
	for i := range plan.contactos {
		contacto := &plan.contactos[i]
		if err := s.Base.normalizarYValidarContacto(ctx, contacto, i); err != nil {
			return plan, err
		}
		clave := fmt.Sprintf("%d|%s", contacto.TipoCanalContactoId, contacto.Proposito)
		if _, repetido := contactosVistos[clave]; repetido {
			return plan, errorCaracterizacion(400,
				fmt.Sprintf("Contacto[%d]: hay otro contacto con el mismo TipoCanalContactoId y Proposito", i), nil)
		}
		contactosVistos[clave] = struct{}{}
	}

	actividadesVistas := make(map[int64]struct{}, len(plan.actividades))
	for i, actividad := range plan.actividades {
		if actividad.Id != nil && *actividad.Id <= 0 {
			return plan, errorCaracterizacion(400,
				fmt.Sprintf("ActividadEconomicaProveedor[%d].Id debe ser un entero positivo", i), nil)
		}
		if actividad.ActividadEconomicaId <= 0 {
			return plan, errorCaracterizacion(400,
				fmt.Sprintf("ActividadEconomicaProveedor[%d].ActividadEconomicaId debe ser un entero positivo", i), nil)
		}
		if _, repetido := actividadesVistas[actividad.ActividadEconomicaId]; repetido {
			return plan, errorCaracterizacion(400,
				fmt.Sprintf("ActividadEconomicaProveedor[%d].ActividadEconomicaId está repetido", i), nil)
		}
		actividadesVistas[actividad.ActividadEconomicaId] = struct{}{}
		existe, err := s.Base.Parametros.ParametroDeTipo(
			ctx, actividad.ActividadEconomicaId, s.Base.TipoParametroCIIU,
		)
		if err != nil {
			return plan, errorDependencia(ctx, err)
		}
		if !existe {
			return plan, errorCaracterizacion(400,
				fmt.Sprintf("ActividadEconomicaProveedor[%d].ActividadEconomicaId no existe, está inactivo o no es de tipo %s", i, s.Base.TipoParametroCIIU), nil)
		}
	}

	return plan, nil
}

func (s CaracterizacionJuridicaService) prepararCargoRepresentante(
	ctx context.Context,
	representante *planRepresentante,
) (planInfoComplementaria, *responses.RelationError, error) {
	detalle, ok, err := s.Base.Catalogos.ResolverInfoComplementaria(
		ctx, "CARGO_REPRESENTANTE", s.GrupoProveedorJuridico,
	)
	if err != nil {
		return planInfoComplementaria{}, nil, errorDependencia(ctx, err)
	}
	if !ok || !detalle.Activo {
		return planInfoComplementaria{}, nil, errorCaracterizacion(500,
			"No está configurado el campo CARGO_REPRESENTANTE en el grupo "+s.GrupoProveedorJuridico, nil)
	}
	var aviso *responses.RelationError
	if detalle.Ambiguo {
		aviso = &responses.RelationError{
			Path:    "InfoComplementariaTercero",
			Service: "terceros_crud",
			Message: "El código CARGO_REPRESENTANTE tiene varias coincidencias activas; se usó la de menor Id",
			Kind:    "ambiguous_match",
		}
	}
	datoJSON, err := json.Marshal(representante.cargoID)
	if err != nil {
		return planInfoComplementaria{}, nil, errorCaracterizacion(500,
			"No fue posible serializar el cargo del representante", err)
	}
	return planInfoComplementaria{
		infoComplementariaID: detalle.Id,
		codigo:               "CARGO_REPRESENTANTE",
		codigoGrupo:          s.GrupoProveedorJuridico,
		dato:                 representante.cargoID,
		datoJSON:             string(datoJSON),
		esOpcion:             false,
	}, aviso, nil
}

func (s CaracterizacionJuridicaService) prepararRepresentante(
	ctx context.Context,
	in *models.RepresentanteInput,
) (*planRepresentante, []responses.RelationError, error) {
	if in == nil {
		return nil, nil, nil
	}

	var avisos []responses.RelationError
	if in.RepresentanteId != nil && *in.RepresentanteId <= 0 {
		return nil, nil, errorCaracterizacion(400,
			"Representante.RepresentanteId debe ser un entero positivo", nil)
	}
	if in.TipoDocumentoId != nil && *in.TipoDocumentoId <= 0 {
		return nil, nil, errorCaracterizacion(400,
			"Representante.TipoDocumentoId debe ser un entero positivo", nil)
	}
	if in.CargoId != nil && *in.CargoId <= 0 {
		return nil, nil, errorCaracterizacion(400,
			"Representante.CargoId debe ser un entero positivo", nil)
	}

	tieneID := in.RepresentanteId != nil && *in.RepresentanteId > 0
	tieneCodigoDoc := in.CodigoTipoDocumento != nil && strings.TrimSpace(*in.CodigoTipoDocumento) != ""
	tieneTipoDocID := in.TipoDocumentoId != nil && *in.TipoDocumentoId > 0
	tieneNumero := in.Numero != nil && strings.TrimSpace(*in.Numero) != ""
	if !tieneID && (!tieneNumero || (!tieneCodigoDoc && !tieneTipoDocID)) {
		return nil, nil, errorCaracterizacion(400,
			"Representante requiere RepresentanteId o (TipoDocumentoId/CodigoTipoDocumento + Numero)", nil)
	}
	if tieneID && !tieneNumero && (tieneCodigoDoc || tieneTipoDocID) {
		return nil, nil, errorCaracterizacion(400,
			"Representante: si envía TipoDocumentoId o CodigoTipoDocumento también debe enviar Numero", nil)
	}
	if tieneNumero && !tieneCodigoDoc && !tieneTipoDocID {
		return nil, nil, errorCaracterizacion(400,
			"Representante: si envía Numero también debe enviar TipoDocumentoId o CodigoTipoDocumento", nil)
	}

	var err error
	var tipoDocumentoID int64
	var codigoDocumento string
	if tieneCodigoDoc || tieneTipoDocID {
		var avisoDoc *responses.RelationError
		tipoDocumentoID, codigoDocumento, avisoDoc, err = s.resolverTipoDocumentoRepresentante(ctx, in)
		if err != nil {
			return nil, nil, err
		}
		if avisoDoc != nil {
			avisos = append(avisos, *avisoDoc)
		}
	}

	var tercero map[string]any
	var terceroID int64
	if tieneID {
		terceroID = *in.RepresentanteId
		tercero, err = s.Base.consultarTercero(ctx, terceroID)
		if err != nil {
			var caracterizacionErr *CaracterizacionNaturalError
			if errors.As(err, &caracterizacionErr) && caracterizacionErr.StatusCode == 404 {
				return nil, nil, errorCaracterizacion(400,
					"No existe un tercero con el RepresentanteId indicado", nil)
			}
			return nil, nil, err
		}
		if tieneNumero && tieneTipoDocID {
			porDocumento, ok, err := s.Representantes.TerceroPorDocumento(
				ctx, tipoDocumentoID, strings.TrimSpace(*in.Numero),
			)
			if err != nil {
				if errors.Is(err, ErrCatalogoAmbiguo) {
					return nil, nil, errorCaracterizacion(409,
						"Hay más de un tercero activo con el documento del representante", nil)
				}
				return nil, nil, errorDependencia(ctx, err)
			}
			if !ok || idDeMapa(porDocumento) != terceroID {
				return nil, nil, errorCaracterizacion(400,
					"RepresentanteId no coincide con el documento indicado", nil)
			}
		}
	} else {
		porDocumento, ok, err := s.Representantes.TerceroPorDocumento(
			ctx, tipoDocumentoID, strings.TrimSpace(*in.Numero),
		)
		if err != nil {
			if errors.Is(err, ErrCatalogoAmbiguo) {
				return nil, nil, errorCaracterizacion(409,
					"Hay más de un tercero activo con el documento del representante", nil)
			}
			return nil, nil, errorDependencia(ctx, err)
		}
		if !ok {
			return nil, nil, errorCaracterizacion(400,
				"No existe un tercero natural con el documento del representante; regístrelo por /v1/proveedores/natural", nil)
		}
		tercero = porDocumento
		terceroID = idDeMapa(tercero)
	}
	if terceroID <= 0 {
		return nil, nil, errorDependencia(ctx,
			errors.New("el representante no tiene un TerceroId válido"))
	}

	activo, existeActivo := tercero["Activo"].(bool)
	if !existeActivo {
		return nil, nil, errorDependencia(ctx,
			errors.New("la respuesta del representante no contiene Activo"))
	}
	if !activo {
		return nil, nil, errorCaracterizacion(409,
			"El representante está inactivo", nil)
	}
	tipoContribuyente := idDeRelacion(tercero["TipoContribuyenteId"])
	if tipoContribuyente != int64(s.TipoContribuyenteNatural) {
		return nil, nil, errorCaracterizacion(409,
			"El representante no corresponde a una persona natural", nil)
	}

	tieneProveedorNatural, err := s.proveedorNaturalActivo(ctx, terceroID)
	if err != nil {
		return nil, nil, err
	}
	if !tieneProveedorNatural {
		return nil, nil, errorCaracterizacion(409,
			"El representante no tiene un proveedor natural activo; regístrelo por /v1/proveedores/natural/caracterizacion", nil)
	}

	cargoID, codigoCargo, avisoCargo, err := s.resolverCargoRepresentante(ctx, in)
	if err != nil {
		return nil, nil, err
	}
	if avisoCargo != nil {
		avisos = append(avisos, *avisoCargo)
	}

	if aviso := avisoNombreRepresentante(tercero, in); aviso != nil {
		avisos = append(avisos, *aviso)
	}

	numero := ""
	if in.Numero != nil {
		numero = strings.TrimSpace(*in.Numero)
	}
	if tipoDocumentoID == 0 || numero == "" {
		identificacion, ok, err := s.Representantes.IdentificacionActiva(ctx, terceroID)
		if err != nil {
			return nil, nil, errorDependencia(ctx, err)
		}
		if ok {
			if tipoDocumentoID == 0 {
				tipoDocumentoID = idDeRelacion(identificacion["TipoDocumentoId"])
			}
			if numero == "" {
				numero = strings.TrimSpace(cadenaCuenta(identificacion["Numero"]))
			}
		}
	}
	if codigoDocumento == "" && tipoDocumentoID > 0 {
		if codigo, ok, err := s.Representantes.TipoDocumentoPorID(ctx, tipoDocumentoID); err == nil && ok {
			codigoDocumento = codigo
		}
	}
	if codigoCargo == "" && cargoID > 0 {
		if detalle, ok, err := s.Base.Catalogos.InfoComplementariaPorID(ctx, cargoID); err == nil && ok {
			codigoCargo = detalle.Codigo
		}
	}
	return &planRepresentante{
		terceroID:           terceroID,
		tercero:             tercero,
		tipoDocumentoID:     tipoDocumentoID,
		codigoTipoDocumento: codigoDocumento,
		numero:              numero,
		cargoID:             cargoID,
		codigoCargo:         codigoCargo,
	}, avisos, nil
}

func (s CaracterizacionJuridicaService) resolverTipoDocumentoRepresentante(
	ctx context.Context,
	in *models.RepresentanteInput,
) (int64, string, *responses.RelationError, error) {
	var tipoID int64
	var codigo string
	var aviso *responses.RelationError
	if in.CodigoTipoDocumento != nil && strings.TrimSpace(*in.CodigoTipoDocumento) != "" {
		codigo = strings.ToUpper(strings.TrimSpace(*in.CodigoTipoDocumento))
		if codigo == "PA" {
			codigo = "PAS"
		}
		id, existe, ambiguo, err := s.Representantes.TipoDocumentoPorCodigo(ctx, codigo)
		if err != nil {
			return 0, "", nil, errorDependencia(ctx, err)
		}
		if !existe {
			return 0, "", nil, errorCaracterizacion(400,
				fmt.Sprintf("Representante.CodigoTipoDocumento %q no existe o está inactivo", codigo), nil)
		}
		if ambiguo {
			aviso = &responses.RelationError{
				Path:    "Representante",
				Service: "terceros_crud",
				Message: fmt.Sprintf("El tipo de documento %q tiene varias coincidencias activas; se usó la de menor Id", codigo),
				Kind:    "ambiguous_match",
			}
		}
		tipoID = id
	}
	if in.TipoDocumentoId != nil && *in.TipoDocumentoId > 0 {
		existe, err := s.Representantes.TipoDocumentoExiste(ctx, *in.TipoDocumentoId)
		if err != nil {
			return 0, "", nil, errorDependencia(ctx, err)
		}
		if !existe {
			return 0, "", nil, errorCaracterizacion(400,
				"Representante.TipoDocumentoId no existe o está inactivo", nil)
		}
		if tipoID > 0 && tipoID != *in.TipoDocumentoId {
			return 0, "", nil, errorCaracterizacion(400,
				"Representante.TipoDocumentoId y CodigoTipoDocumento no coinciden", nil)
		}
		tipoID = *in.TipoDocumentoId
	}
	if tipoID <= 0 {
		return 0, "", nil, errorCaracterizacion(400,
			"Representante requiere TipoDocumentoId o CodigoTipoDocumento", nil)
	}
	return tipoID, codigo, aviso, nil
}

func (s CaracterizacionJuridicaService) resolverCargoRepresentante(
	ctx context.Context,
	in *models.RepresentanteInput,
) (int64, string, *responses.RelationError, error) {
	tieneID := in.CargoId != nil && *in.CargoId > 0
	tieneCodigo := in.CodigoCargo != nil && strings.TrimSpace(*in.CodigoCargo) != ""
	if !tieneID && !tieneCodigo {
		return 0, "", nil, errorCaracterizacion(400,
			"Representante requiere CargoId o CodigoCargo", nil)
	}

	var cargoID int64
	var codigo string
	var aviso *responses.RelationError
	if tieneCodigo {
		codigo = strings.TrimSpace(*in.CodigoCargo)
		if contieneControl(codigo) {
			return 0, "", nil, errorCaracterizacion(400,
				"Representante.CodigoCargo no admite caracteres de control", nil)
		}
		detalle, ok, err := s.Base.Catalogos.ResolverInfoComplementaria(ctx, codigo, s.GrupoCargo)
		if err != nil {
			return 0, "", nil, errorDependencia(ctx, err)
		}
		if !ok || !detalle.Activo {
			return 0, "", nil, errorCaracterizacion(400,
				fmt.Sprintf("Representante.CodigoCargo %q no existe, está inactivo o no pertenece al grupo %s", codigo, s.GrupoCargo), nil)
		}
		if detalle.Ambiguo {
			aviso = &responses.RelationError{
				Path:    "Representante",
				Service: "terceros_crud",
				Message: fmt.Sprintf("El cargo %q tiene varias coincidencias activas; se usó la de menor Id", codigo),
				Kind:    "ambiguous_match",
			}
		}
		cargoID = detalle.Id
	}
	if tieneID {
		pertenece, err := s.Base.Catalogos.InfoComplementariaDeGrupo(ctx, *in.CargoId, s.GrupoCargo)
		if err != nil {
			return 0, "", nil, errorDependencia(ctx, err)
		}
		if !pertenece {
			return 0, "", nil, errorCaracterizacion(400,
				fmt.Sprintf("Representante.CargoId no existe, está inactivo o no pertenece al grupo %s", s.GrupoCargo), nil)
		}
		if cargoID > 0 && cargoID != *in.CargoId {
			return 0, "", nil, errorCaracterizacion(400,
				"Representante.CargoId y CodigoCargo no coinciden", nil)
		}
		cargoID = *in.CargoId
	}
	return cargoID, codigo, aviso, nil
}

func (s CaracterizacionJuridicaService) proveedorNaturalActivo(
	ctx context.Context,
	terceroID int64,
) (bool, error) {
	filtros := url.Values{
		"query": {fmt.Sprintf("TerceroId:%d,Activo:true", terceroID)},
		"limit": {"10"},
	}
	var registros []map[string]any
	ok, status, err := GetAllProveedoresMapas(ctx, endpointProveedor, filtros, &registros)
	if err != nil {
		return false, errorDependencia(ctx, err)
	}
	if !ok {
		return false, errorDependencia(ctx,
			fmt.Errorf("consulta de proveedor con estado %d", status))
	}
	activos, _, err := separarRegistrosProveedor(ctx, registros)
	if err != nil {
		return false, err
	}
	naturales := make([]map[string]any, 0, len(activos))
	for _, registro := range activos {
		if idDeRelacion(registro["TipoProveedor"]) == int64(s.TipoProveedorNatural) {
			naturales = append(naturales, registro)
		}
	}
	if len(naturales) > 1 {
		return false, errorCaracterizacion(409,
			"El representante tiene más de un proveedor natural activo; no es posible resolverlo de forma segura", nil)
	}
	if len(naturales) == 0 {
		return false, nil
	}
	proveedorID := idDeMapa(naturales[0])
	if proveedorID <= 0 {
		return false, errorDependencia(ctx,
			errors.New("el proveedor natural del representante no tiene Id"))
	}
	var detalle map[string]any
	ok, status, err = GetOneProveedores(ctx,
		fmt.Sprintf("%s/%d", endpointProveedorNatural, proveedorID), &detalle)
	if err != nil {
		if status == 404 {
			return false, nil
		}
		return false, errorLecturaCaracterizacion(ctx, status,
			"No fue posible consultar el proveedor natural del representante", err)
	}
	if !ok {
		if status == 404 {
			return false, nil
		}
		return false, errorLecturaCaracterizacion(ctx, status,
			"No fue posible consultar el proveedor natural del representante", nil)
	}
	detalle = desenvolverData(detalle)
	activo, existe := detalle["Activo"].(bool)
	if !existe {
		return false, errorDependencia(ctx,
			errors.New("el proveedor natural del representante no contiene Activo"))
	}
	return activo, nil
}

func avisoNombreRepresentante(
	tercero map[string]any,
	in *models.RepresentanteInput,
) *responses.RelationError {
	proporcionado := nombreRepresentanteProporcionado(in)
	if strings.TrimSpace(proporcionado) == "" {
		return nil
	}
	existente, _ := tercero["NombreCompleto"].(string)
	if strings.TrimSpace(existente) == "" {
		existente = nombreRepresentanteTercero(tercero)
	}
	if strings.TrimSpace(existente) == "" {
		return nil
	}
	if normalizarNombre(existente) == normalizarNombre(proporcionado) {
		return nil
	}
	return &responses.RelationError{
		Path:    "Representante",
		Service: "terceros_crud",
		Message: "Los nombres enviados no coinciden con el tercero resuelto por documento; se usó el tercero del catálogo",
		Kind:    "name_mismatch",
	}
}

func nombreRepresentanteProporcionado(in *models.RepresentanteInput) string {
	partes := []*string{
		in.PrimerNombre, in.SegundoNombre, in.PrimerApellido, in.SegundoApellido,
	}
	return unirPartes(partes)
}

func nombreRepresentanteTercero(tercero map[string]any) string {
	partes := []*string{}
	for _, campo := range []string{
		"PrimerNombre", "SegundoNombre", "PrimerApellido", "SegundoApellido",
	} {
		if valor, ok := tercero[campo].(string); ok {
			copia := valor
			partes = append(partes, &copia)
		} else {
			partes = append(partes, nil)
		}
	}
	return unirPartes(partes)
}

func unirPartes(partes []*string) string {
	salida := make([]string, 0, len(partes))
	for _, parte := range partes {
		if parte == nil {
			continue
		}
		if texto := strings.TrimSpace(*parte); texto != "" {
			salida = append(salida, texto)
		}
	}
	return strings.Join(salida, " ")
}

func normalizarNombre(valor string) string {
	return strings.ToLower(strings.Join(strings.Fields(valor), " "))
}

func (s CaracterizacionJuridicaService) prepararProveedorJuridico(
	ctx context.Context,
	in *models.ProveedorJuridicoInput,
) (*planProveedorJuridico, []responses.RelationError, error) {
	if in == nil {
		return nil, nil, nil
	}
	var avisos []responses.RelationError

	tieneProcedencia := (in.ProcedenciaEmpresaId != nil && *in.ProcedenciaEmpresaId > 0) ||
		(in.CodigoProcedencia != nil && strings.TrimSpace(*in.CodigoProcedencia) != "")
	tieneTipoConformacion := (in.TipoConformacionId != nil && *in.TipoConformacionId > 0) ||
		(in.CodigoTipoConformacion != nil && strings.TrimSpace(*in.CodigoTipoConformacion) != "")
	tieneRegimen := (in.RegimenContributivo != nil && *in.RegimenContributivo > 0) ||
		(in.CodigoRegimenContributivo != nil && strings.TrimSpace(*in.CodigoRegimenContributivo) != "")

	faltantes := make([]string, 0, 10)
	if !tieneProcedencia {
		faltantes = append(faltantes, "ProcedenciaEmpresaId o CodigoProcedencia")
	}
	if !tieneTipoConformacion {
		faltantes = append(faltantes, "TipoConformacionId o CodigoTipoConformacion")
	}
	if !tieneRegimen {
		faltantes = append(faltantes, "RegimenContributivo o CodigoRegimenContributivo")
	}
	for nombre, valor := range map[string]*bool{
		"ExclusividadProducto": in.ExclusividadProducto,
		"Pyme":                 in.Pyme,
		"RegistroMercantil":    in.RegistroMercantil,
		"SujetoRetencion":      in.SujetoRetencion,
		"AgenteRetenedor":      in.AgenteRetenedor,
		"ResponsableIca":       in.ResponsableIca,
		"ResponsableIva":       in.ResponsableIva,
	} {
		if valor == nil {
			faltantes = append(faltantes, nombre)
		}
	}
	if len(faltantes) > 0 {
		return nil, nil, errorCaracterizacion(400,
			"ProveedorJuridico requiere: "+strings.Join(faltantes, ", "), nil)
	}

	plan := &planProveedorJuridico{
		ciudadOrigenID:       in.CiudadOrigenId,
		paisOrigenID:         in.PaisOrigenId,
		exclusividadProducto: in.ExclusividadProducto,
		pyme:                 in.Pyme,
		registroMercantil:    in.RegistroMercantil,
		sujetoRetencion:      in.SujetoRetencion,
		agenteRetenedor:      in.AgenteRetenedor,
		responsableIca:       in.ResponsableIca,
		responsableIva:       in.ResponsableIva,
	}
	if in.CodigoPostal != nil {
		codigo := strings.TrimSpace(*in.CodigoPostal)
		if utf8.RuneCountInString(codigo) > maxCodigoPostalJuridico {
			return nil, nil, errorCaracterizacion(400,
				fmt.Sprintf("ProveedorJuridico.CodigoPostal no puede superar %d caracteres", maxCodigoPostalJuridico), nil)
		}
		if contieneControl(codigo) {
			return nil, nil, errorCaracterizacion(400,
				"ProveedorJuridico.CodigoPostal no admite caracteres de control", nil)
		}
		plan.codigoPostal = &codigo
	}
	for nombre, valor := range map[string]*int64{
		"CiudadOrigenId": in.CiudadOrigenId,
		"PaisOrigenId":   in.PaisOrigenId,
	} {
		if valor == nil {
			continue
		}
		if *valor <= 0 {
			return nil, nil, errorCaracterizacion(400,
				"ProveedorJuridico."+nombre+" debe ser un entero positivo", nil)
		}
		existe, err := s.Base.Catalogos.LugarExiste(ctx, *valor)
		if err != nil {
			return nil, nil, errorDependencia(ctx, err)
		}
		if !existe {
			return nil, nil, errorCaracterizacion(400,
				"ProveedorJuridico."+nombre+" no existe o está inactivo", nil)
		}
	}

	procedenciaID, codigoProcedencia, avisoProcedencia, err := s.resolverProcedencia(ctx, in)
	if err != nil {
		return nil, nil, err
	}
	if avisoProcedencia != nil {
		avisos = append(avisos, *avisoProcedencia)
	}
	plan.procedenciaID = &procedenciaID
	plan.codigoProcedencia = codigoProcedencia

	tipoConformacionID, codigoTipo, avisoTipo, err := s.resolverParametroDeProveedor(
		ctx, in.TipoConformacionId, in.CodigoTipoConformacion,
		s.TipoParametroTipoConformacion, "TipoConformacionId", "CodigoTipoConformacion",
	)
	if err != nil {
		return nil, nil, err
	}
	if avisoTipo != nil {
		avisos = append(avisos, *avisoTipo)
	}
	plan.tipoConformacionID = &tipoConformacionID
	plan.codigoTipoConformacion = codigoTipo

	regimenID, codigoRegimen, avisoRegimen, err := s.resolverParametroDeProveedor(
		ctx, in.RegimenContributivo, in.CodigoRegimenContributivo,
		s.TipoParametroRegimenContributivo, "RegimenContributivo", "CodigoRegimenContributivo",
	)
	if err != nil {
		return nil, nil, err
	}
	if avisoRegimen != nil {
		avisos = append(avisos, *avisoRegimen)
	}
	plan.regimenContributivoID = &regimenID
	plan.codigoRegimenContributivo = codigoRegimen

	return plan, avisos, nil
}

func (s CaracterizacionJuridicaService) resolverProcedencia(
	ctx context.Context,
	in *models.ProveedorJuridicoInput,
) (int64, string, *responses.RelationError, error) {
	var id int64
	var codigo string
	var aviso *responses.RelationError
	if in.CodigoProcedencia != nil && strings.TrimSpace(*in.CodigoProcedencia) != "" {
		codigo = strings.TrimSpace(*in.CodigoProcedencia)
		detalle, ok, err := s.Base.Catalogos.ResolverInfoComplementaria(ctx, codigo, s.GrupoProcedencia)
		if err != nil {
			return 0, "", nil, errorDependencia(ctx, err)
		}
		if !ok || !detalle.Activo {
			return 0, "", nil, errorCaracterizacion(400,
				fmt.Sprintf("ProveedorJuridico.CodigoProcedencia %q no existe, está inactivo o no pertenece al grupo %s", codigo, s.GrupoProcedencia), nil)
		}
		if detalle.Ambiguo {
			aviso = &responses.RelationError{
				Path:    "ProveedorJuridico",
				Service: "terceros_crud",
				Message: fmt.Sprintf("La procedencia %q tiene varias coincidencias activas; se usó la de menor Id", codigo),
				Kind:    "ambiguous_match",
			}
		}
		id = detalle.Id
	}
	if in.ProcedenciaEmpresaId != nil && *in.ProcedenciaEmpresaId > 0 {
		pertenece, err := s.Base.Catalogos.InfoComplementariaDeGrupo(
			ctx, *in.ProcedenciaEmpresaId, s.GrupoProcedencia,
		)
		if err != nil {
			return 0, "", nil, errorDependencia(ctx, err)
		}
		if !pertenece {
			return 0, "", nil, errorCaracterizacion(400,
				fmt.Sprintf("ProveedorJuridico.ProcedenciaEmpresaId no existe, está inactivo o no pertenece al grupo %s", s.GrupoProcedencia), nil)
		}
		if id > 0 && id != *in.ProcedenciaEmpresaId {
			return 0, "", nil, errorCaracterizacion(400,
				"ProveedorJuridico.ProcedenciaEmpresaId y CodigoProcedencia no coinciden", nil)
		}
		id = *in.ProcedenciaEmpresaId
	}
	return id, codigo, aviso, nil
}

func (s CaracterizacionJuridicaService) resolverParametroDeProveedor(
	ctx context.Context,
	id *int64,
	codigo *string,
	tipo string,
	campoID string,
	campoCodigo string,
) (int64, string, *responses.RelationError, error) {
	var resuelto int64
	var codigoResuelto string
	var aviso *responses.RelationError
	if codigo != nil && strings.TrimSpace(*codigo) != "" {
		codigoResuelto = strings.TrimSpace(*codigo)
		encontrado, existe, ambiguo, err := s.Parametros.ResolverParametroPorCodigo(
			ctx, codigoResuelto, tipo,
		)
		if err != nil {
			return 0, "", nil, errorDependencia(ctx, err)
		}
		if !existe {
			return 0, "", nil, errorCaracterizacion(400,
				fmt.Sprintf("ProveedorJuridico.%s %q no existe o está inactivo para el tipo %s", campoCodigo, codigoResuelto, tipo), nil)
		}
		if ambiguo {
			aviso = &responses.RelationError{
				Path:    "ProveedorJuridico",
				Service: "parametros_crud",
				Message: fmt.Sprintf("El código %q del tipo %s tiene varias coincidencias activas; se usó la de menor Id", codigoResuelto, tipo),
				Kind:    "ambiguous_match",
			}
		}
		resuelto = encontrado
	}
	if id != nil && *id > 0 {
		existe, err := s.Parametros.ParametroDeTipo(ctx, *id, tipo)
		if err != nil {
			return 0, "", nil, errorDependencia(ctx, err)
		}
		if !existe {
			return 0, "", nil, errorCaracterizacion(400,
				fmt.Sprintf("ProveedorJuridico.%s no existe, está inactivo o no es de tipo %s", campoID, tipo), nil)
		}
		if resuelto > 0 && resuelto != *id {
			return 0, "", nil, errorCaracterizacion(400,
				fmt.Sprintf("ProveedorJuridico.%s y %s no coinciden", campoID, campoCodigo), nil)
		}
		resuelto = *id
	}
	return resuelto, codigoResuelto, aviso, nil
}

func (s CaracterizacionJuridicaService) completarOrigenDesdeDireccion(
	ctx context.Context,
	plan *planCaracterizacionJuridica,
) error {
	if plan == nil || plan.juridico == nil {
		return nil
	}
	var ciudad, pais int64
	for _, item := range plan.info {
		switch strings.ToUpper(strings.TrimSpace(item.codigo)) {
		case "DIRECCIÓN", "DIRECCION":
			direccion, ok := item.dato.(map[string]any)
			if !ok {
				continue
			}
			if valor, ok := datoEntero(direccion["CiudadId"]); ok {
				ciudad = valor
			}
			if valor, ok := datoEntero(direccion["PaisId"]); ok {
				pais = valor
			}
		}
	}
	if plan.juridico.ciudadOrigenID != nil && ciudad > 0 && *plan.juridico.ciudadOrigenID != ciudad {
		return errorCaracterizacion(400,
			"ProveedorJuridico.CiudadOrigenId y DIRECCIÓN.CiudadId deben coincidir", nil)
	}
	if plan.juridico.paisOrigenID != nil && pais > 0 && *plan.juridico.paisOrigenID != pais {
		return errorCaracterizacion(400,
			"ProveedorJuridico.PaisOrigenId y DIRECCIÓN.PaisId deben coincidir", nil)
	}
	if plan.juridico.ciudadOrigenID == nil && ciudad > 0 {
		plan.juridico.ciudadOrigenID = &ciudad
	}
	if plan.juridico.paisOrigenID == nil && pais > 0 {
		plan.juridico.paisOrigenID = &pais
	}
	if plan.juridico.ciudadOrigenID != nil {
		existe, err := s.Base.Catalogos.LugarExiste(ctx, *plan.juridico.ciudadOrigenID)
		if err != nil {
			return errorDependencia(ctx, err)
		}
		if !existe {
			return errorCaracterizacion(400,
				"ProveedorJuridico.CiudadOrigenId no existe o está inactivo", nil)
		}
	}
	if plan.juridico.paisOrigenID != nil {
		existe, err := s.Base.Catalogos.LugarExiste(ctx, *plan.juridico.paisOrigenID)
		if err != nil {
			return errorDependencia(ctx, err)
		}
		if !existe {
			return errorCaracterizacion(400,
				"ProveedorJuridico.PaisOrigenId no existe o está inactivo", nil)
		}
	}
	return nil
}

func validarPrincipalEnActividades(
	info []planInfoComplementaria,
	actividades []models.ActividadEconomicaProveedorInput,
) error {
	if len(actividades) == 0 {
		return nil
	}
	var principal int64
	for _, item := range info {
		if !strings.EqualFold(strings.TrimSpace(item.codigo), "ACTIVIDAD_PRINCIPAL") {
			continue
		}
		if id, ok := datoEntero(item.dato); ok {
			principal = id
		}
	}
	if principal <= 0 {
		return nil
	}
	for _, actividad := range actividades {
		if actividad.ActividadEconomicaId == principal {
			return nil
		}
	}
	return errorCaracterizacion(400,
		"ACTIVIDAD_PRINCIPAL debe estar entre las actividades enviadas en ActividadEconomicaProveedor", nil)
}

func (s CaracterizacionJuridicaService) escribirSecciones(
	ctx context.Context,
	terceroID int64,
	proveedorID int64,
	plan planCaracterizacionJuridica,
	existentesInfo *existentesInfoTercero,
	resultado *CaracterizacionJuridicaResult,
) {
	for _, item := range plan.info {
		registrada, accionInfo, aviso, err := s.Base.escribirInfoComplementaria(
			ctx, terceroID, item, existentesInfo,
		)
		if aviso != nil {
			resultado.Avisos = append(resultado.Avisos, *aviso)
		}
		if err != nil {
			resultado.Avisos = append(resultado.Avisos,
				avisoDeErrorCaracterizacion(err, "InfoComplementariaTercero", "terceros_crud"))
			resultado.Parcial = true
			continue
		}
		resultado.Data.InfoComplementariaTercero = append(
			resultado.Data.InfoComplementariaTercero, registrada,
		)
		registrarResultado(&resultado.Data.Resultado,
			"InfoComplementariaTercero", registrada.Id, accionInfo)
	}

	if plan.juridico != nil {
		juridico, accion, err := s.escribirProveedorJuridico(ctx, terceroID, proveedorID, plan.juridico)
		if err != nil {
			resultado.Avisos = append(resultado.Avisos,
				avisoDeErrorCaracterizacion(err, "ProveedorJuridico", "gestion_proveedores_crud"))
			resultado.Parcial = true
		} else {
			resultado.Data.ProveedorJuridico = juridico
			registrarResultado(&resultado.Data.Resultado,
				"ProveedorJuridico", proveedorID, accion)
		}
	}

	if plan.representante != nil {
		id, accion, aviso, err := s.escribirRepresentacion(
			ctx, terceroID, proveedorID, plan.representante,
		)
		if aviso != nil {
			resultado.Avisos = append(resultado.Avisos, *aviso)
		}
		if err != nil {
			resultado.Avisos = append(resultado.Avisos,
				avisoDeErrorCaracterizacion(err, "Representacion", "gestion_proveedores_crud"))
			resultado.Parcial = true
		} else {
			resultado.Data.Representacion = &models.EntidadId{
				Entidad: "Representacion",
				Id:      id,
			}
			registrarResultado(&resultado.Data.Resultado,
				"Representacion", id, accion)
		}
	}

	for _, perfil := range plan.perfiles {
		id, accion, aviso, err := s.Base.escribirPerfil(ctx, terceroID, proveedorID, perfil)
		if aviso != nil {
			resultado.Avisos = append(resultado.Avisos, *aviso)
		}
		if err != nil {
			resultado.Avisos = append(resultado.Avisos,
				avisoDeErrorCaracterizacion(err, "PerfilProveedor", "gestion_proveedores_crud"))
			resultado.Parcial = true
			continue
		}
		resultado.Data.PerfilProveedor = append(
			resultado.Data.PerfilProveedor, models.EntidadId{Entidad: "PerfilProveedor", Id: id},
		)
		registrarResultado(&resultado.Data.Resultado, "PerfilProveedor", id, accion)
	}

	for _, contacto := range plan.contactos {
		id, accion, aviso, err := s.Base.escribirContacto(ctx, terceroID, proveedorID, contacto)
		if aviso != nil {
			resultado.Avisos = append(resultado.Avisos, *aviso)
		}
		if err != nil {
			resultado.Avisos = append(resultado.Avisos,
				avisoDeErrorCaracterizacion(err, "Contacto", "gestion_proveedores_crud"))
			resultado.Parcial = true
			continue
		}
		resultado.Data.Contacto = append(
			resultado.Data.Contacto, models.EntidadId{Entidad: "Contacto", Id: id},
		)
		registrarResultado(&resultado.Data.Resultado, "Contacto", id, accion)
	}

	for _, actividad := range plan.actividades {
		id, accion, aviso, err := s.Base.escribirActividad(ctx, terceroID, proveedorID, actividad)
		if aviso != nil {
			resultado.Avisos = append(resultado.Avisos, *aviso)
		}
		if err != nil {
			resultado.Avisos = append(resultado.Avisos,
				avisoDeErrorCaracterizacion(err, "ActividadEconomicaProveedor", "gestion_proveedores_crud"))
			resultado.Parcial = true
			continue
		}
		resultado.Data.ActividadEconomicaProveedor = append(
			resultado.Data.ActividadEconomicaProveedor,
			models.EntidadId{Entidad: "ActividadEconomicaProveedor", Id: id},
		)
		registrarResultado(&resultado.Data.Resultado,
			"ActividadEconomicaProveedor", id, accion)
	}
}

func (s CaracterizacionJuridicaService) escribirProveedorJuridico(
	ctx context.Context,
	terceroID int64,
	proveedorID int64,
	plan *planProveedorJuridico,
) (map[string]any, string, error) {
	if plan == nil {
		return nil, "", nil
	}
	var registro map[string]any
	ok, status, err := GetOneProveedores(ctx,
		fmt.Sprintf("%s/%d", endpointProveedorJuridico, proveedorID), &registro)
	if err != nil && status != 404 {
		return nil, "", errorLecturaCaracterizacion(ctx, status,
			"No fue posible consultar proveedor_juridico", err)
	}
	if ok {
		registro = desenvolverData(registro)
		if idDeMapa(registro) != proveedorID {
			return nil, "", errorDependencia(ctx,
				errors.New("la respuesta de proveedor_juridico no contiene el Id esperado"))
		}
		activo, existe := registro["Activo"].(bool)
		if !existe {
			return nil, "", errorDependencia(ctx,
				errors.New("la respuesta de proveedor_juridico no contiene Activo"))
		}
		if !activo {
			return nil, "", errorCaracterizacion(409,
				"El detalle de proveedor jurídico está inactivo", nil)
		}
		if proveedorJuridicoIgual(registro, plan) &&
			idDeRelacion(registro["UsuarioModificacionId"]) == terceroID {
			return respuestaProveedorJuridico(registro, plan), "reutilizado", nil
		}
		payload := clonarPayload(registro)
		aplicarCamposProveedorJuridico(payload, plan)
		aplicarUsuarioModificacion(payload, terceroID)
		ok, status, err = PutProveedores(ctx,
			fmt.Sprintf("%s/%d", endpointProveedorJuridico, proveedorID),
			payload, &map[string]any{})
		if err != nil {
			return nil, "", errorEscrituraCaracterizacion(ctx, status, err, "actualizar proveedor_juridico")
		}
		if !ok {
			return nil, "", errorEscrituraCaracterizacion(ctx, status,
				fmt.Errorf("el CRUD respondió con estado %d", status), "actualizar proveedor_juridico")
		}
		return respuestaProveedorJuridico(payload, plan), "actualizado", nil
	}

	payload := map[string]any{
		"Id":     proveedorID,
		"Activo": true,
	}
	aplicarCamposProveedorJuridico(payload, plan)
	aplicarUsuarioModificacion(payload, terceroID)
	ok, status, err = PostProveedores(ctx, endpointProveedorJuridico, payload, &map[string]any{})
	if err != nil {
		return nil, "", errorEscrituraCaracterizacion(ctx, status, err, "crear proveedor_juridico")
	}
	if !ok {
		return nil, "", errorEscrituraCaracterizacion(ctx, status,
			fmt.Errorf("el CRUD respondió con estado %d", status), "crear proveedor_juridico")
	}
	return respuestaProveedorJuridico(payload, plan), "creado", nil
}

func aplicarCamposProveedorJuridico(
	payload map[string]any,
	plan *planProveedorJuridico,
) {
	if plan.procedenciaID != nil {
		payload["ProcedenciaEmpresaId"] = *plan.procedenciaID
	}
	if plan.ciudadOrigenID != nil {
		payload["CiudadOrigenId"] = *plan.ciudadOrigenID
	}
	if plan.paisOrigenID != nil {
		payload["PaisOrigenId"] = *plan.paisOrigenID
	}
	if plan.codigoPostal != nil {
		payload["CodigoPostal"] = *plan.codigoPostal
	}
	if plan.tipoConformacionID != nil {
		payload["TipoConformacionId"] = *plan.tipoConformacionID
	}
	if plan.regimenContributivoID != nil {
		payload["RegimenContributivo"] = *plan.regimenContributivoID
	}
	if plan.exclusividadProducto != nil {
		payload["ExclusividadProducto"] = *plan.exclusividadProducto
	}
	if plan.pyme != nil {
		payload["Pyme"] = *plan.pyme
	}
	if plan.registroMercantil != nil {
		payload["RegistroMercantil"] = *plan.registroMercantil
	}
	if plan.sujetoRetencion != nil {
		payload["SujetoRetencion"] = *plan.sujetoRetencion
	}
	if plan.agenteRetenedor != nil {
		payload["AgenteRetenedor"] = *plan.agenteRetenedor
	}
	if plan.responsableIca != nil {
		payload["ResponsableIca"] = *plan.responsableIca
	}
	if plan.responsableIva != nil {
		payload["ResponsableIva"] = *plan.responsableIva
	}
}

func proveedorJuridicoIgual(
	registro map[string]any,
	plan *planProveedorJuridico,
) bool {
	if plan.procedenciaID != nil && idDeRelacion(registro["ProcedenciaEmpresaId"]) != *plan.procedenciaID {
		return false
	}
	if plan.ciudadOrigenID != nil && idDeRelacion(registro["CiudadOrigenId"]) != *plan.ciudadOrigenID {
		return false
	}
	if plan.paisOrigenID != nil && idDeRelacion(registro["PaisOrigenId"]) != *plan.paisOrigenID {
		return false
	}
	if plan.codigoPostal != nil {
		existente, _ := registro["CodigoPostal"].(string)
		if existente != *plan.codigoPostal {
			return false
		}
	}
	if plan.tipoConformacionID != nil && idDeRelacion(registro["TipoConformacionId"]) != *plan.tipoConformacionID {
		return false
	}
	if plan.regimenContributivoID != nil && idDeRelacion(registro["RegimenContributivo"]) != *plan.regimenContributivoID {
		return false
	}
	for nombre, valor := range map[string]*bool{
		"ExclusividadProducto": plan.exclusividadProducto,
		"Pyme":                 plan.pyme,
		"RegistroMercantil":    plan.registroMercantil,
		"SujetoRetencion":      plan.sujetoRetencion,
		"AgenteRetenedor":      plan.agenteRetenedor,
		"ResponsableIca":       plan.responsableIca,
		"ResponsableIva":       plan.responsableIva,
	} {
		if valor == nil {
			continue
		}
		existente, _ := registro[nombre].(bool)
		if existente != *valor {
			return false
		}
	}
	return true
}

func respuestaProveedorJuridico(
	registro map[string]any,
	plan *planProveedorJuridico,
) map[string]any {
	respuesta := clonarPayload(registro)
	if plan.codigoProcedencia != "" {
		respuesta["CodigoProcedencia"] = plan.codigoProcedencia
	}
	if plan.codigoTipoConformacion != "" {
		respuesta["CodigoTipoConformacion"] = plan.codigoTipoConformacion
	}
	if plan.codigoRegimenContributivo != "" {
		respuesta["CodigoRegimenContributivo"] = plan.codigoRegimenContributivo
	}
	return respuesta
}

func (s CaracterizacionJuridicaService) escribirRepresentacion(
	ctx context.Context,
	terceroID int64,
	proveedorID int64,
	representante *planRepresentante,
) (int64, string, *responses.RelationError, error) {
	if representante == nil {
		return 0, "", nil, nil
	}
	filtros := url.Values{
		"query": {fmt.Sprintf("ProveedorId:%d", proveedorID)},
		"limit": {"100"},
	}
	var registros []map[string]any
	ok, status, err := GetAllProveedoresMapas(ctx, endpointRepresentacion, filtros, &registros)
	if err != nil {
		return 0, "", nil, errorDependencia(ctx, err)
	}
	if !ok {
		return 0, "", nil, errorDependencia(ctx,
			fmt.Errorf("consulta de representacion con estado %d", status))
	}

	var mismoActivo map[string]any
	var mismoInactivo map[string]any
	activos := make([]map[string]any, 0, len(registros))
	for _, registro := range registros {
		registro = desenvolverData(registro)
		if idDeMapa(registro) <= 0 {
			continue
		}
		activo, _ := registro["Activo"].(bool)
		representanteID := idDeRelacion(registro["RepresentanteId"])
		if activo {
			activos = append(activos, registro)
			if representanteID == representante.terceroID && mismoActivo == nil {
				mismoActivo = registro
			}
			continue
		}
		if representanteID == representante.terceroID && mismoInactivo == nil {
			mismoInactivo = registro
		}
	}

	if mismoActivo != nil {
		for _, activo := range activos {
			if idDeMapa(activo) == idDeMapa(mismoActivo) {
				continue
			}
			if err := s.desactivarRepresentacion(ctx, terceroID, activo); err != nil {
				return 0, "", nil, err
			}
		}
		id := idDeMapa(mismoActivo)
		if idDeRelacion(mismoActivo["UsuarioModificacionId"]) == terceroID {
			return id, "reutilizado", nil, nil
		}
		payload := clonarPayload(mismoActivo)
		aplicarUsuarioModificacion(payload, terceroID)
		ok, status, err := PutProveedores(ctx,
			fmt.Sprintf("%s/%d", endpointRepresentacion, id),
			payload, &map[string]any{})
		if err != nil {
			return 0, "", nil, errorEscrituraCaracterizacion(ctx, status, err, "actualizar representacion")
		}
		if !ok {
			return 0, "", nil, errorEscrituraCaracterizacion(ctx, status,
				fmt.Errorf("el CRUD respondió con estado %d", status), "actualizar representacion")
		}
		return id, "actualizado", nil, nil
	}

	for _, activo := range activos {
		if err := s.desactivarRepresentacion(ctx, terceroID, activo); err != nil {
			return 0, "", nil, err
		}
	}
	if mismoInactivo != nil {
		payload := clonarPayload(mismoInactivo)
		payload["Activo"] = true
		aplicarUsuarioModificacion(payload, terceroID)
		ok, status, err := PutProveedores(ctx,
			fmt.Sprintf("%s/%d", endpointRepresentacion, idDeMapa(mismoInactivo)),
			payload, &map[string]any{})
		if err != nil {
			return 0, "", nil, errorEscrituraCaracterizacion(ctx, status, err, "reactivar representacion")
		}
		if !ok {
			return 0, "", nil, errorEscrituraCaracterizacion(ctx, status,
				fmt.Errorf("el CRUD respondió con estado %d", status), "reactivar representacion")
		}
		return idDeMapa(mismoInactivo), "actualizado", nil, nil
	}

	payload := aplicarUsuarioModificacion(map[string]any{
		"ProveedorId":     map[string]any{"Id": proveedorID},
		"RepresentanteId": representante.terceroID,
		"Activo":          true,
	}, terceroID)
	var respuesta map[string]any
	ok, status, err = PostProveedores(ctx, endpointRepresentacion, payload, &respuesta)
	if err != nil {
		return 0, "", nil, errorEscrituraCaracterizacion(ctx, status, err, "crear representacion")
	}
	if !ok {
		return 0, "", nil, errorEscrituraCaracterizacion(ctx, status,
			fmt.Errorf("el CRUD respondió con estado %d", status), "crear representacion")
	}
	respuesta = desenvolverData(respuesta)
	id := idDeMapa(respuesta)
	if id <= 0 {
		return 0, "", nil, errorCaracterizacion(502,
			"La creación de representacion no devolvió un Id utilizable", nil)
	}
	return id, "creado", nil, nil
}

func (s CaracterizacionJuridicaService) desactivarRepresentacion(
	ctx context.Context,
	terceroID int64,
	registro map[string]any,
) error {
	id := idDeMapa(registro)
	if id <= 0 {
		return errorDependencia(ctx, errors.New("la representación no contiene Id"))
	}
	payload := clonarPayload(registro)
	payload["Activo"] = false
	aplicarUsuarioModificacion(payload, terceroID)
	ok, status, err := PutProveedores(ctx,
		fmt.Sprintf("%s/%d", endpointRepresentacion, id),
		payload, &map[string]any{})
	if err != nil {
		return errorEscrituraCaracterizacion(ctx, status, err, "desactivar representacion")
	}
	if !ok {
		return errorEscrituraCaracterizacion(ctx, status,
			fmt.Errorf("el CRUD respondió con estado %d", status), "desactivar representacion")
	}
	return nil
}
