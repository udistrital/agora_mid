package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/udistrital/agora_mid/helper"
	"github.com/udistrital/agora_mid/models"
	"github.com/udistrital/agora_mid/responses"
)

const (
	endpointInfoComplementariaTercero = "/v1/info_complementaria_tercero"
	endpointProveedor                 = "/v1/proveedor"
	endpointProveedorNatural          = "/v1/proveedor_natural"
	endpointPerfilProveedor           = "/v1/perfil_proveedor"
	endpointContacto                  = "/v1/contacto"
	endpointActividadEconomica        = "/v1/actividad_economica_proveedor"
)

const (
	codigoGrupoPerfil        = "Grupo_5"
	maxCuerpoInfoDato        = 16 * 1024
	maxDescripcionPortafolio = 2000
)

// gruposOpcionInfoComplementaria son los grupos de selección única. El valor es
// el nombre del campo del formulario y solo se usa en mensajes de error.
var gruposOpcionInfoComplementaria = map[string]string{
	"Grupo_1":            "DISCAPACIDAD",
	"Grupo_2":            "ESTADO_CIVIL",
	"Grupo_3":            "GRUPO_ETNICO",
	"Grupo_6":            "GENERO",
	"IDENTIDAD_GENERO":   "SE_RECONOCE_COMO",
	"ORIENTACION_SEXUAL": "ORIENTACION_SEXUAL",
}

func esGrupoOpcionInfoComplementaria(codigoGrupo string) bool {
	_, ok := gruposOpcionInfoComplementaria[strings.TrimSpace(codigoGrupo)]
	return ok
}

// CatalogosCaracterizacion es la porción de catálogos que usa el dominio 2.
type CatalogosCaracterizacion interface {
	LugarExiste(ctx context.Context, id int64) (bool, error)
	InfoComplementariaExiste(ctx context.Context, id int64) (bool, error)
	ResolverInfoComplementaria(ctx context.Context, codigo, codigoGrupo string) (InfoComplementariaResuelta, bool, error)
	InfoComplementariaPorID(ctx context.Context, id int64) (InfoComplementariaResuelta, bool, error)
	InfoComplementariaDeGrupo(ctx context.Context, id int64, codigoGrupo string) (bool, error)
}

// ParametrosCaracterizacion valida parámetros del CRUD de parámetros.
type ParametrosCaracterizacion interface {
	ParametroExiste(ctx context.Context, id int64) (bool, error)
	ParametroDeTipo(ctx context.Context, id int64, tipo string) (bool, error)
}

// CaracterizacionNaturalError describe un fallo del dominio 2 con su código HTTP.
type CaracterizacionNaturalError struct {
	StatusCode int
	Message    string
	Cause      error
}

func (e *CaracterizacionNaturalError) Error() string { return e.Message }
func (e *CaracterizacionNaturalError) Unwrap() error { return e.Cause }

// CaracterizacionNaturalResult agrupa los datos materializados y los avisos.
type CaracterizacionNaturalResult struct {
	Data    models.CaracterizacionNaturalData
	Avisos  []responses.RelationError
	Parcial bool
}

// CaracterizacionNaturalService implementa el dominio 2 para persona natural.
type CaracterizacionNaturalService struct {
	Catalogos                  CatalogosCaracterizacion
	Parametros                 ParametrosCaracterizacion
	TipoContribuyenteNatural   int
	TipoProveedorNatural       int
	TipoParametroCIIU          string
	TipoParametroCanalContacto string
	mutex                      *keyedMutex
}

// NewCaracterizacionNaturalService construye el servicio del dominio 2 natural.
func NewCaracterizacionNaturalService(
	catalogos CatalogosCaracterizacion,
	parametros ParametrosCaracterizacion,
	tipoContribuyenteNatural int,
	tipoProveedorNatural int,
	tipoParametroCIIU string,
	tipoParametroCanalContacto string,
) CaracterizacionNaturalService {
	if tipoProveedorNatural <= 0 {
		tipoProveedorNatural = 1
	}
	if strings.TrimSpace(tipoParametroCIIU) == "" {
		tipoParametroCIIU = "CIIU"
	}
	return CaracterizacionNaturalService{
		Catalogos:                  catalogos,
		Parametros:                 parametros,
		TipoContribuyenteNatural:   tipoContribuyenteNatural,
		TipoProveedorNatural:       tipoProveedorNatural,
		TipoParametroCIIU:          strings.TrimSpace(tipoParametroCIIU),
		TipoParametroCanalContacto: strings.TrimSpace(tipoParametroCanalContacto),
		mutex:                      newKeyedMutex(),
	}
}

type planInfoComplementaria struct {
	id                   *int64
	infoComplementariaID int64
	codigo               string
	codigoGrupo          string
	dato                 any
	datoJSON             string
	padreID              *int64
	esOpcion             bool
	grupoOpcion          string
	ambiguo              bool
}

type existentesInfoTercero struct {
	porInfo map[int64][]map[string]any
	porID   map[int64]map[string]any
}

type planCaracterizacion struct {
	info           []planInfoComplementaria
	natural        *models.ProveedorNaturalInput
	perfiles       []models.PerfilProveedorInput
	contactos      []models.ContactoProveedorInput
	actividades    []models.ActividadEconomicaProveedorInput
	descripcion    *string
	consentimiento *bool
	avisos         []responses.RelationError
}

// Registrar ejecuta el upsert parcial de la caracterización natural.
func (s CaracterizacionNaturalService) Registrar(
	ctx context.Context,
	in models.RegistroCaracterizacionNatural,
	terceroID int64,
	verb string,
) (CaracterizacionNaturalResult, error) {
	if terceroID <= 0 {
		return CaracterizacionNaturalResult{}, errorCaracterizacion(400,
			"TerceroId debe ser un entero positivo", nil)
	}
	if s.Catalogos == nil || s.Parametros == nil {
		return CaracterizacionNaturalResult{}, errorCaracterizacion(500,
			"El servicio de caracterización no está configurado", nil)
	}
	if s.TipoContribuyenteNatural <= 0 {
		return CaracterizacionNaturalResult{}, errorCaracterizacion(500,
			"No está configurado el tipo de contribuyente P_NATURAL", nil)
	}

	plan, err := s.preparar(ctx, in)
	if err != nil {
		return CaracterizacionNaturalResult{}, err
	}

	unlock := func() {}
	if s.mutex != nil {
		unlock = s.mutex.lock(strconv.FormatInt(terceroID, 10))
	}
	defer unlock()

	tercero, err := s.consultarTercero(ctx, terceroID)
	if err != nil {
		return CaracterizacionNaturalResult{}, err
	}
	if err := s.validarTerceroNatural(ctx, tercero); err != nil {
		return CaracterizacionNaturalResult{}, err
	}

	resultado := nuevoResultadoCaracterizacion(terceroID, plan)

	var existentesInfo *existentesInfoTercero
	if len(plan.info) > 0 {
		existentesInfo, err = s.cargarInfoExistentes(ctx, terceroID)
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
// Es la vía recomendada para los PUT de los pasos 2 y 4: evita resolver el
// proveedor por TerceroId y elimina ambigüedades si hay más de un registro.
func (s CaracterizacionNaturalService) RegistrarPorProveedor(
	ctx context.Context,
	in models.RegistroCaracterizacionNatural,
	proveedorID int64,
	verb string,
) (CaracterizacionNaturalResult, error) {
	if proveedorID <= 0 {
		return CaracterizacionNaturalResult{}, errorCaracterizacion(400,
			"proveedorId debe ser un entero positivo", nil)
	}
	if s.Catalogos == nil || s.Parametros == nil {
		return CaracterizacionNaturalResult{}, errorCaracterizacion(500,
			"El servicio de caracterización no está configurado", nil)
	}
	if s.TipoContribuyenteNatural <= 0 {
		return CaracterizacionNaturalResult{}, errorCaracterizacion(500,
			"No está configurado el tipo de contribuyente P_NATURAL", nil)
	}

	plan, err := s.preparar(ctx, in)
	if err != nil {
		return CaracterizacionNaturalResult{}, err
	}

	proveedor, err := s.consultarProveedorPorID(ctx, proveedorID)
	if err != nil {
		return CaracterizacionNaturalResult{}, err
	}
	terceroID := idDeRelacion(proveedor["TerceroId"])
	if terceroID <= 0 {
		return CaracterizacionNaturalResult{}, errorDependencia(ctx,
			errors.New("el proveedor no contiene un TerceroId válido"))
	}
	if in.TerceroId != 0 && in.TerceroId != terceroID {
		return CaracterizacionNaturalResult{}, errorCaracterizacion(400,
			"TerceroId del cuerpo no coincide con el proveedor indicado", nil)
	}

	unlock := func() {}
	if s.mutex != nil {
		unlock = s.mutex.lock(strconv.FormatInt(terceroID, 10))
	}
	defer unlock()

	// Se relee dentro del lock para usar el estado vigente.
	proveedor, err = s.consultarProveedorPorID(ctx, proveedorID)
	if err != nil {
		return CaracterizacionNaturalResult{}, err
	}
	tercero, err := s.consultarTercero(ctx, terceroID)
	if err != nil {
		return CaracterizacionNaturalResult{}, err
	}
	if err := s.validarTerceroNatural(ctx, tercero); err != nil {
		return CaracterizacionNaturalResult{}, err
	}

	resultado := nuevoResultadoCaracterizacion(terceroID, plan)
	var existentesInfo *existentesInfoTercero
	if len(plan.info) > 0 {
		existentesInfo, err = s.cargarInfoExistentes(ctx, terceroID)
		if err != nil {
			return resultado, err
		}
	}

	proveedor, accion, avisosProveedor, err := s.actualizarProveedorConocido(ctx, proveedor, plan)
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

func nuevoResultadoCaracterizacion(
	terceroID int64,
	plan planCaracterizacion,
) CaracterizacionNaturalResult {
	return CaracterizacionNaturalResult{
		Avisos: plan.avisos,
		Data: models.CaracterizacionNaturalData{
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
}

func (s CaracterizacionNaturalService) consultarProveedorPorID(
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
	if tipo != int64(s.TipoProveedorNatural) {
		return nil, errorCaracterizacion(409,
			"El proveedor no corresponde a persona natural", nil)
	}
	activo, existe := registro["Activo"].(bool)
	if !existe {
		return nil, errorDependencia(ctx,
			errors.New("la respuesta del proveedor no contiene Activo"))
	}
	if !activo {
		return nil, errorCaracterizacion(409,
			"El proveedor está inactivo", nil)
	}
	return registro, nil
}

func (s CaracterizacionNaturalService) escribirSecciones(
	ctx context.Context,
	terceroID int64,
	proveedorID int64,
	plan planCaracterizacion,
	existentesInfo *existentesInfoTercero,
	resultado *CaracterizacionNaturalResult,
) {
	// Sección InfoComplementariaTercero (terceros_crud).
	for _, item := range plan.info {
		registrada, accionInfo, aviso, err := s.escribirInfoComplementaria(
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

	// Sección ProveedorNatural.
	if plan.natural != nil {
		natural, accionNatural, err := s.escribirProveedorNatural(
			ctx, terceroID, proveedorID, plan.natural,
		)
		if err != nil {
			resultado.Avisos = append(resultado.Avisos,
				avisoDeErrorCaracterizacion(err, "ProveedorNatural", "gestion_proveedores_crud"))
			resultado.Parcial = true
		} else {
			resultado.Data.ProveedorNatural = natural
			registrarResultado(&resultado.Data.Resultado,
				"ProveedorNatural", proveedorID, accionNatural)
		}
	}

	// Sección PerfilProveedor.
	for _, perfil := range plan.perfiles {
		id, accionPerfil, aviso, err := s.escribirPerfil(ctx, terceroID, proveedorID, perfil)
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
		registrarResultado(&resultado.Data.Resultado, "PerfilProveedor", id, accionPerfil)
	}

	// Sección Contacto.
	for _, contacto := range plan.contactos {
		id, accionContacto, aviso, err := s.escribirContacto(ctx, terceroID, proveedorID, contacto)
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
		registrarResultado(&resultado.Data.Resultado, "Contacto", id, accionContacto)
	}

	// Sección ActividadEconomicaProveedor.
	for _, actividad := range plan.actividades {
		id, accionActividad, aviso, err := s.escribirActividad(ctx, terceroID, proveedorID, actividad)
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
			"ActividadEconomicaProveedor", id, accionActividad)
	}

}

func (s CaracterizacionNaturalService) preparar(
	ctx context.Context,
	in models.RegistroCaracterizacionNatural,
) (planCaracterizacion, error) {
	plan := planCaracterizacion{
		natural:        in.ProveedorNatural,
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
	if plan.natural != nil {
		if err := s.validarProveedorNatural(ctx, plan.natural); err != nil {
			return plan, err
		}
	}

	idsInfo := make(map[int64]struct{}, len(in.InfoComplementariaTercero))
	gruposOpcionVistos := make(map[string]struct{}, len(in.InfoComplementariaTercero))

	for i, item := range in.InfoComplementariaTercero {
		preparado, err := s.prepararInfoComplementaria(ctx, item, i)
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

	if err := s.validarCoherenciaDireccion(plan.info); err != nil {
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
			detalle, ok, err := s.Catalogos.ResolverInfoComplementaria(ctx, codigo, codigoGrupoPerfil)
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
			existe, err := s.Catalogos.InfoComplementariaDeGrupo(ctx, perfil.PerfilId, codigoGrupoPerfil)
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
		if err := s.normalizarYValidarContacto(ctx, contacto, i); err != nil {
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
		existe, err := s.Parametros.ParametroDeTipo(
			ctx, actividad.ActividadEconomicaId, s.TipoParametroCIIU,
		)
		if err != nil {
			return plan, errorDependencia(ctx, err)
		}
		if !existe {
			return plan, errorCaracterizacion(400,
				fmt.Sprintf("ActividadEconomicaProveedor[%d].ActividadEconomicaId no existe, está inactivo o no es de tipo %s", i, s.TipoParametroCIIU), nil)
		}
	}

	return plan, nil
}

func (s CaracterizacionNaturalService) prepararInfoComplementaria(
	ctx context.Context,
	item models.InfoComplementariaTerceroInput,
	indice int,
) (planInfoComplementaria, error) {
	if len(item.Dato) > maxCuerpoInfoDato {
		return planInfoComplementaria{}, errorCaracterizacion(400,
			fmt.Sprintf("InfoComplementariaTercero[%d].Dato supera el tamaño máximo permitido", indice), nil)
	}
	if item.InfoComplementariaTerceroPadreId != nil && *item.InfoComplementariaTerceroPadreId <= 0 {
		return planInfoComplementaria{}, errorCaracterizacion(400,
			fmt.Sprintf("InfoComplementariaTercero[%d].InfoComplementariaTerceroPadreId debe ser un entero positivo", indice), nil)
	}
	if item.Id != nil && *item.Id <= 0 {
		return planInfoComplementaria{}, errorCaracterizacion(400,
			fmt.Sprintf("InfoComplementariaTercero[%d].Id debe ser un entero positivo", indice), nil)
	}

	tieneID := item.InfoComplementariaId != nil
	tieneCodigo := item.CodigoInfoComplementaria != nil && strings.TrimSpace(*item.CodigoInfoComplementaria) != ""
	if tieneID == tieneCodigo {
		return planInfoComplementaria{}, errorCaracterizacion(400,
			fmt.Sprintf("InfoComplementariaTercero[%d]: debe traer exactamente uno de InfoComplementariaId o CodigoInfoComplementaria", indice), nil)
	}

	var detalle InfoComplementariaResuelta
	if tieneID {
		if *item.InfoComplementariaId <= 0 {
			return planInfoComplementaria{}, errorCaracterizacion(400,
				fmt.Sprintf("InfoComplementariaTercero[%d].InfoComplementariaId debe ser un entero positivo", indice), nil)
		}
		resuelto, ok, err := s.Catalogos.InfoComplementariaPorID(ctx, *item.InfoComplementariaId)
		if err != nil {
			return planInfoComplementaria{}, errorDependencia(ctx, err)
		}
		if !ok {
			return planInfoComplementaria{}, errorCaracterizacion(400,
				fmt.Sprintf("InfoComplementariaTercero[%d].InfoComplementariaId no existe", indice), nil)
		}
		detalle = resuelto
	} else {
		codigo := strings.TrimSpace(*item.CodigoInfoComplementaria)
		if contieneControl(codigo) {
			return planInfoComplementaria{}, errorCaracterizacion(400,
				fmt.Sprintf("InfoComplementariaTercero[%d].CodigoInfoComplementaria no admite caracteres de control", indice), nil)
		}
		if item.CodigoGrupoInfoComplementaria == nil ||
			strings.TrimSpace(*item.CodigoGrupoInfoComplementaria) == "" {
			return planInfoComplementaria{}, errorCaracterizacion(400,
				fmt.Sprintf("InfoComplementariaTercero[%d].CodigoGrupoInfoComplementaria es obligatorio cuando se usa CodigoInfoComplementaria", indice), nil)
		}
		codigoGrupo := strings.TrimSpace(*item.CodigoGrupoInfoComplementaria)
		if contieneControl(codigoGrupo) {
			return planInfoComplementaria{}, errorCaracterizacion(400,
				fmt.Sprintf("InfoComplementariaTercero[%d].CodigoGrupoInfoComplementaria no admite caracteres de control", indice), nil)
		}
		resuelto, ok, err := s.Catalogos.ResolverInfoComplementaria(ctx, codigo, codigoGrupo)
		if err != nil {
			return planInfoComplementaria{}, errorDependencia(ctx, err)
		}
		if !ok {
			return planInfoComplementaria{}, errorCaracterizacion(400,
				fmt.Sprintf("InfoComplementariaTercero[%d].CodigoInfoComplementaria %q no existe en el grupo %q", indice, codigo, codigoGrupo), nil)
		}
		if !strings.EqualFold(strings.TrimSpace(resuelto.CodigoGrupo), codigoGrupo) {
			return planInfoComplementaria{}, errorCaracterizacion(400,
				fmt.Sprintf("InfoComplementariaTercero[%d]: el código %q no pertenece al grupo %q", indice, codigo, codigoGrupo), nil)
		}
		detalle = resuelto
	}

	if item.CodigoGrupoInfoComplementaria != nil &&
		strings.TrimSpace(*item.CodigoGrupoInfoComplementaria) != "" &&
		!strings.EqualFold(strings.TrimSpace(detalle.CodigoGrupo), strings.TrimSpace(*item.CodigoGrupoInfoComplementaria)) {
		return planInfoComplementaria{}, errorCaracterizacion(400,
			fmt.Sprintf("InfoComplementariaTercero[%d]: el grupo enviado no coincide con el del catálogo", indice), nil)
	}

	if esGrupoOpcionInfoComplementaria(detalle.CodigoGrupo) {
		if !detalle.Activo {
			return planInfoComplementaria{}, errorCaracterizacion(400,
				fmt.Sprintf("InfoComplementariaTercero[%d]: la opción %q no existe o está inactiva en el catálogo", indice, detalle.Codigo), nil)
		}
		var datoOpcion any
		if len(item.Dato) > 0 {
			if err := json.Unmarshal(item.Dato, &datoOpcion); err != nil {
				return planInfoComplementaria{}, errorCaracterizacion(400,
					fmt.Sprintf("InfoComplementariaTercero[%d].Dato no es un JSON válido", indice), nil)
			}
			if datoOpcion != nil {
				texto, ok := datoOpcion.(string)
				if !ok || strings.TrimSpace(texto) != "" {
					return planInfoComplementaria{}, errorCaracterizacion(400,
						fmt.Sprintf("InfoComplementariaTercero[%d]: para un campo de opción Dato debe ir vacío", indice), nil)
				}
			}
		}
		return planInfoComplementaria{
			id:                   item.Id,
			infoComplementariaID: detalle.Id,
			codigo:               detalle.Codigo,
			codigoGrupo:          detalle.CodigoGrupo,
			dato:                 "",
			datoJSON:             `""`,
			padreID:              item.InfoComplementariaTerceroPadreId,
			esOpcion:             true,
			grupoOpcion:          detalle.CodigoGrupo,
			ambiguo:              detalle.Ambiguo,
		}, nil
	}

	if len(item.Dato) == 0 {
		return planInfoComplementaria{}, errorCaracterizacion(400,
			fmt.Sprintf("InfoComplementariaTercero[%d].Dato es obligatorio", indice), nil)
	}
	var dato any
	if err := json.Unmarshal(item.Dato, &dato); err != nil {
		return planInfoComplementaria{}, errorCaracterizacion(400,
			fmt.Sprintf("InfoComplementariaTercero[%d].Dato no es un JSON válido", indice), nil)
	}
	if !detalle.Activo {
		return planInfoComplementaria{}, errorCaracterizacion(400,
			fmt.Sprintf("InfoComplementariaTercero[%d].CodigoInfoComplementaria %q no existe o está inactivo", indice, detalle.Codigo), nil)
	}
	if err := s.validarDatoPorCodigo(ctx, detalle.Codigo, dato, indice); err != nil {
		return planInfoComplementaria{}, err
	}
	datoJSON, err := json.Marshal(dato)
	if err != nil {
		return planInfoComplementaria{}, errorCaracterizacion(400,
			fmt.Sprintf("InfoComplementariaTercero[%d].Dato no se pudo serializar", indice), err)
	}
	return planInfoComplementaria{
		id:                   item.Id,
		infoComplementariaID: detalle.Id,
		codigo:               detalle.Codigo,
		codigoGrupo:          detalle.CodigoGrupo,
		dato:                 dato,
		datoJSON:             string(datoJSON),
		padreID:              item.InfoComplementariaTerceroPadreId,
		ambiguo:              detalle.Ambiguo,
	}, nil
}

func (s CaracterizacionNaturalService) validarDatoPorCodigo(
	ctx context.Context,
	codigo string,
	dato any,
	indice int,
) error {
	switch strings.ToUpper(strings.TrimSpace(codigo)) {
	case "GENERO":
		return s.validarOpcionInfoComplementaria(ctx, dato, "Grupo_6", indice, codigo)
	case "GRUPO_ETNICO":
		return s.validarOpcionInfoComplementaria(ctx, dato, "Grupo_3", indice, codigo)
	case "SE_RECONOCE_COMO":
		return s.validarOpcionInfoComplementaria(ctx, dato, "IDENTIDAD_GENERO", indice, codigo)
	case "ORIENTACION_SEXUAL":
		return s.validarOpcionInfoComplementaria(ctx, dato, "ORIENTACION_SEXUAL", indice, codigo)
	case "ESTADO_CIVIL":
		return s.validarOpcionInfoComplementaria(ctx, dato, "Grupo_2", indice, codigo)
	case "DISCAPACIDAD":
		return s.validarOpcionInfoComplementaria(ctx, dato, "Grupo_1", indice, codigo)
	case "HIJOS", "CABEZA_FAMILIA", "PERSONAS_A_CARGO", "DECLARANTE_RENTA",
		"MIGRANTE", "VICTIMA_CONFLICTO", "PERSONA_PEP", "CUENTA_AFC",
		"AUTORIZACION_GIRO", "APLICA_RUP":
		if _, ok := dato.(bool); !ok {
			return errorCaracterizacion(400,
				fmt.Sprintf("InfoComplementariaTercero[%d].Dato para %s debe ser booleano", indice, codigo), nil)
		}
	case "NUMERO_HIJOS":
		numero, ok := datoEntero(dato)
		if !ok || numero < 0 {
			return errorCaracterizacion(400,
				fmt.Sprintf("InfoComplementariaTercero[%d].Dato para %s debe ser un entero no negativo", indice, codigo), nil)
		}
	case "EPS":
		return s.validarParametroDeTipo(ctx, dato, "EPS", indice, codigo)
	case "AFP":
		return s.validarParametroDeTipo(ctx, dato, "EFP", indice, codigo)
	case "CCF":
		return s.validarParametroDeTipo(ctx, dato, "CCF", indice, codigo)
	case "ACTIVIDAD_PRINCIPAL":
		id, ok := datoEntero(dato)
		if !ok || id <= 0 {
			return errorCaracterizacion(400,
				fmt.Sprintf("InfoComplementariaTercero[%d].Dato para %s debe ser un Id de parámetro CIIU", indice, codigo), nil)
		}
		existe, err := s.Parametros.ParametroDeTipo(ctx, id, s.TipoParametroCIIU)
		if err != nil {
			return errorDependencia(ctx, err)
		}
		if !existe {
			return errorCaracterizacion(400,
				fmt.Sprintf("InfoComplementariaTercero[%d].Dato para %s no es un CIIU activo", indice, codigo), nil)
		}
	case "PAIS_EXPEDICION", "DEPARTAMENTO_EXPEDICION", "DEPTO_EXPEDICION", "LUGAR_RESIDENCIA":
		id, ok := datoEntero(dato)
		if !ok || id <= 0 {
			return errorCaracterizacion(400,
				fmt.Sprintf("InfoComplementariaTercero[%d].Dato para %s debe ser un Id de lugar", indice, codigo), nil)
		}
		existe, err := s.Catalogos.LugarExiste(ctx, id)
		if err != nil {
			return errorDependencia(ctx, err)
		}
		if !existe {
			return errorCaracterizacion(400,
				fmt.Sprintf("InfoComplementariaTercero[%d].Dato para %s no existe o está inactivo", indice, codigo), nil)
		}
	case "DIRECCIÓN", "DIRECCION":
		return s.validarDireccionDian(ctx, dato, indice)
	case "TIPO_CONF_CUENTA":
		texto, ok := dato.(string)
		if !ok || !esTipoConformacionCuentaValido(texto) {
			return errorCaracterizacion(400,
				fmt.Sprintf("InfoComplementariaTercero[%d].Dato para %s debe ser INDIVIDUAL_PRINCIPAL, CONJUNTA_MANCOMUNADA o CONSORCIO_UT", indice, codigo), nil)
		}
	case "CALIDAD_GRAN_CONTRIB":
		texto, ok := dato.(string)
		if !ok || !esCalidadGranContribuyenteValida(texto) {
			return errorCaracterizacion(400,
				fmt.Sprintf("InfoComplementariaTercero[%d].Dato para %s debe ser NO, SI_CALIFICADO o AUTORRETENEDOR", indice, codigo), nil)
		}
	case "SITIO_WEB":
		texto, ok := dato.(string)
		if !ok || strings.TrimSpace(texto) == "" {
			return errorCaracterizacion(400,
				fmt.Sprintf("InfoComplementariaTercero[%d].Dato para %s debe ser una cadena no vacía", indice, codigo), nil)
		}
		parsed, err := url.Parse(strings.TrimSpace(texto))
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return errorCaracterizacion(400,
				fmt.Sprintf("InfoComplementariaTercero[%d].Dato para %s debe ser una URL http o https válida", indice, codigo), nil)
		}
	}
	return nil
}

func esTipoConformacionCuentaValido(valor string) bool {
	switch strings.ToUpper(strings.TrimSpace(valor)) {
	case "INDIVIDUAL_PRINCIPAL", "CONJUNTA_MANCOMUNADA", "CONSORCIO_UT":
		return true
	default:
		return false
	}
}

func esCalidadGranContribuyenteValida(valor string) bool {
	switch strings.ToUpper(strings.TrimSpace(valor)) {
	case "NO", "SI_CALIFICADO", "AUTORRETENEDOR":
		return true
	default:
		return false
	}
}

func (s CaracterizacionNaturalService) validarParametroDeTipo(
	ctx context.Context,
	dato any,
	tipo string,
	indice int,
	codigo string,
) error {
	id, ok := datoEntero(dato)
	if !ok || id <= 0 {
		return errorCaracterizacion(400,
			fmt.Sprintf("InfoComplementariaTercero[%d].Dato para %s debe ser un Id de parámetro", indice, codigo), nil)
	}
	existe, err := s.Parametros.ParametroDeTipo(ctx, id, tipo)
	if err != nil {
		return errorDependencia(ctx, err)
	}
	if !existe {
		return errorCaracterizacion(400,
			fmt.Sprintf("InfoComplementariaTercero[%d].Dato para %s no existe, está inactivo o no es de tipo %s", indice, codigo, tipo), nil)
	}
	return nil
}

func (s CaracterizacionNaturalService) validarDireccionDian(
	ctx context.Context,
	dato any,
	indice int,
) error {
	direccion, ok := dato.(map[string]any)
	if !ok {
		return errorCaracterizacion(400,
			fmt.Sprintf("InfoComplementariaTercero[%d].Dato para DIRECCIÓN debe ser un objeto", indice), nil)
	}
	requeridos := []string{
		"DepartamentoId", "CiudadId", "ViaId",
		"DetalleVia", "NumeroCruce", "PlacaPuerta", "Direccion",
	}
	for _, campo := range requeridos {
		valor, existe := direccion[campo]
		if !existe || valor == nil {
			return errorCaracterizacion(400,
				fmt.Sprintf("InfoComplementariaTercero[%d].Dato para DIRECCIÓN requiere %s", indice, campo), nil)
		}
	}
	for _, campo := range []string{"PaisId", "DepartamentoId", "CiudadId", "ViaId", "InteriorId"} {
		valor, existe := direccion[campo]
		if !existe || valor == nil {
			continue
		}
		id, ok := datoEntero(valor)
		if !ok || id <= 0 {
			return errorCaracterizacion(400,
				fmt.Sprintf("InfoComplementariaTercero[%d].Dato para DIRECCIÓN.%s debe ser un Id de lugar", indice, campo), nil)
		}
		existeLugar, err := s.Catalogos.LugarExiste(ctx, id)
		if err != nil {
			return errorDependencia(ctx, err)
		}
		if !existeLugar {
			return errorCaracterizacion(400,
				fmt.Sprintf("InfoComplementariaTercero[%d].Dato para DIRECCIÓN.%s no existe o está inactivo", indice, campo), nil)
		}
	}
	for _, campo := range []string{
		"Via", "DetalleVia", "NumeroCruce", "PlacaPuerta",
		"Interior", "DetalleInterior", "Direccion",
	} {
		valor, existe := direccion[campo]
		if !existe || valor == nil {
			continue
		}
		texto, ok := valor.(string)
		if !ok {
			return errorCaracterizacion(400,
				fmt.Sprintf("InfoComplementariaTercero[%d].Dato para DIRECCIÓN.%s debe ser una cadena", indice, campo), nil)
		}
		if contieneControl(texto) {
			return errorCaracterizacion(400,
				fmt.Sprintf("InfoComplementariaTercero[%d].Dato para DIRECCIÓN.%s no admite caracteres de control", indice, campo), nil)
		}
		if campo == "Direccion" && strings.TrimSpace(texto) == "" {
			return errorCaracterizacion(400,
				fmt.Sprintf("InfoComplementariaTercero[%d].Dato para DIRECCIÓN.Direccion no puede estar vacío", indice), nil)
		}
	}
	return nil
}

func (s CaracterizacionNaturalService) validarCoherenciaDireccion(
	info []planInfoComplementaria,
) error {
	var ciudadDireccion int64
	var ciudadResidencia int64
	for _, item := range info {
		switch strings.ToUpper(strings.TrimSpace(item.codigo)) {
		case "DIRECCIÓN", "DIRECCION":
			if mapa, ok := item.dato.(map[string]any); ok {
				if id, ok := datoEntero(mapa["CiudadId"]); ok {
					ciudadDireccion = id
				}
			}
		case "LUGAR_RESIDENCIA":
			if id, ok := datoEntero(item.dato); ok {
				ciudadResidencia = id
			}
		}
	}
	if ciudadDireccion > 0 && ciudadResidencia > 0 && ciudadDireccion != ciudadResidencia {
		return errorCaracterizacion(400,
			"DIRECCIÓN.CiudadId y LUGAR_RESIDENCIA deben coincidir", nil)
	}
	return nil
}

func (s CaracterizacionNaturalService) validarOpcionInfoComplementaria(
	ctx context.Context,
	dato any,
	grupo string,
	indice int,
	codigo string,
) error {
	id, ok := datoEntero(dato)
	if !ok || id <= 0 {
		return errorCaracterizacion(400,
			fmt.Sprintf("InfoComplementariaTercero[%d].Dato para %s debe ser un Id de opción", indice, codigo), nil)
	}
	pertenece, err := s.Catalogos.InfoComplementariaDeGrupo(ctx, id, grupo)
	if err != nil {
		return errorDependencia(ctx, err)
	}
	if !pertenece {
		return errorCaracterizacion(400,
			fmt.Sprintf("InfoComplementariaTercero[%d].Dato para %s no pertenece al grupo %s", indice, codigo, grupo), nil)
	}
	return nil
}

func datoEntero(dato any) (int64, bool) {
	switch valor := dato.(type) {
	case float64:
		if math.Trunc(valor) != valor {
			return 0, false
		}
		return int64(valor), true
	case float32:
		if math.Trunc(float64(valor)) != float64(valor) {
			return 0, false
		}
		return int64(valor), true
	case int:
		return int64(valor), true
	case int64:
		return valor, true
	case json.Number:
		numero, err := valor.Int64()
		return numero, err == nil
	default:
		return 0, false
	}
}

func (s CaracterizacionNaturalService) validarProveedorNatural(
	ctx context.Context,
	in *models.ProveedorNaturalInput,
) error {
	if in.PaisNacimientoId != nil {
		if *in.PaisNacimientoId <= 0 {
			return errorCaracterizacion(400, "ProveedorNatural.PaisNacimientoId debe ser un entero positivo", nil)
		}
		existe, err := s.Catalogos.LugarExiste(ctx, *in.PaisNacimientoId)
		if err != nil {
			return errorDependencia(ctx, err)
		}
		if !existe {
			return errorCaracterizacion(400, "ProveedorNatural.PaisNacimientoId no existe o está inactivo", nil)
		}
	}
	if in.Cargo != nil {
		cargo := strings.TrimSpace(*in.Cargo)
		if utf8.RuneCountInString(cargo) > 150 {
			return errorCaracterizacion(400, "ProveedorNatural.Cargo no puede superar 150 caracteres", nil)
		}
		if contieneControl(cargo) {
			return errorCaracterizacion(400, "ProveedorNatural.Cargo no admite caracteres de control", nil)
		}
		*in.Cargo = cargo
	}
	for nombre, valor := range map[string]*float64{
		"ValorUvtPrepagada":      in.ValorUvtPrepagada,
		"ValorAfc":               in.ValorAfc,
		"ValorPensionVoluntaria": in.ValorPensionVoluntaria,
	} {
		if valor != nil && *valor < 0 {
			return errorCaracterizacion(400, "ProveedorNatural."+nombre+" no puede ser negativo", nil)
		}
	}
	if in.InteresViviendaAfc != nil && (*in.InteresViviendaAfc < 0 || *in.InteresViviendaAfc > 1) {
		return errorCaracterizacion(400, "ProveedorNatural.InteresViviendaAfc debe estar entre 0 y 1", nil)
	}
	for nombre, valor := range map[string]*int64{
		"ExperienciaLaboral":     in.ExperienciaLaboral,
		"ExperienciaProfesional": in.ExperienciaProfesional,
	} {
		if valor != nil && *valor < 0 {
			return errorCaracterizacion(400, "ProveedorNatural."+nombre+" no puede ser negativo", nil)
		}
	}
	return nil
}

func (s CaracterizacionNaturalService) normalizarYValidarContacto(
	ctx context.Context,
	contacto *models.ContactoProveedorInput,
	indice int,
) error {
	if contacto.TipoCanalContactoId <= 0 {
		return errorCaracterizacion(400,
			fmt.Sprintf("Contacto[%d].TipoCanalContactoId debe ser un entero positivo", indice), nil)
	}
	existe, err := s.Parametros.ParametroDeTipo(
		ctx, contacto.TipoCanalContactoId, s.TipoParametroCanalContacto,
	)
	if err != nil {
		return errorDependencia(ctx, err)
	}
	if !existe {
		return errorCaracterizacion(400,
			fmt.Sprintf("Contacto[%d].TipoCanalContactoId no existe, está inactivo o no es del tipo esperado", indice), nil)
	}
	contacto.Proposito = strings.ToUpper(strings.TrimSpace(contacto.Proposito))
	if contacto.Proposito == "" || utf8.RuneCountInString(contacto.Proposito) > 50 {
		return errorCaracterizacion(400,
			fmt.Sprintf("Contacto[%d].Proposito es obligatorio y no puede superar 50 caracteres", indice), nil)
	}
	if strings.ContainsAny(contacto.Proposito, ",:") || contieneControl(contacto.Proposito) {
		return errorCaracterizacion(400,
			fmt.Sprintf("Contacto[%d].Proposito no admite ',' ni ':' ni caracteres de control", indice), nil)
	}
	contacto.ValorEspecifico = strings.TrimSpace(contacto.ValorEspecifico)
	if contacto.ValorEspecifico == "" || utf8.RuneCountInString(contacto.ValorEspecifico) > 100 {
		return errorCaracterizacion(400,
			fmt.Sprintf("Contacto[%d].ValorEspecifico es obligatorio y no puede superar 100 caracteres", indice), nil)
	}
	if strings.ContainsAny(contacto.ValorEspecifico, ",:") || contieneControl(contacto.ValorEspecifico) {
		return errorCaracterizacion(400,
			fmt.Sprintf("Contacto[%d].ValorEspecifico no admite ',' ni ':' ni caracteres de control", indice), nil)
	}
	if contacto.NombreContacto != nil {
		nombre := strings.TrimSpace(*contacto.NombreContacto)
		if utf8.RuneCountInString(nombre) > 100 {
			return errorCaracterizacion(400,
				fmt.Sprintf("Contacto[%d].NombreContacto no puede superar 100 caracteres", indice), nil)
		}
		contacto.NombreContacto = &nombre
	}
	if contacto.Extension != nil {
		extension := strings.TrimSpace(*contacto.Extension)
		if utf8.RuneCountInString(extension) > 10 {
			return errorCaracterizacion(400,
				fmt.Sprintf("Contacto[%d].Extension no puede superar 10 caracteres", indice), nil)
		}
		contacto.Extension = &extension
	}
	if contacto.Id != nil && *contacto.Id <= 0 {
		return errorCaracterizacion(400,
			fmt.Sprintf("Contacto[%d].Id debe ser un entero positivo", indice), nil)
	}
	return nil
}

func normalizarDescripcionPortafolio(valor string) (string, error) {
	texto := strings.TrimSpace(valor)
	if utf8.RuneCountInString(texto) > maxDescripcionPortafolio {
		return "", errorCaracterizacion(400,
			fmt.Sprintf("DescripcionPortafolio no puede superar %d caracteres", maxDescripcionPortafolio), nil)
	}
	if contieneControlDescripcion(texto) {
		return "", errorCaracterizacion(400,
			"DescripcionPortafolio no admite caracteres de control", nil)
	}
	return texto, nil
}

func contieneControl(valor string) bool {
	for _, r := range valor {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}

func contieneControlDescripcion(valor string) bool {
	for _, r := range valor {
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			return true
		}
	}
	return false
}

func (s CaracterizacionNaturalService) consultarTercero(
	ctx context.Context,
	terceroID int64,
) (map[string]any, error) {
	var registro map[string]any
	ok, status, err := GetOne(ctx, fmt.Sprintf("%s/%d", endpointTercero, terceroID), &registro)
	if err != nil {
		return nil, errorLecturaCaracterizacion(ctx, status,
			"No fue posible consultar el tercero", err)
	}
	if !ok {
		return nil, errorLecturaCaracterizacion(ctx, status,
			"No fue posible consultar el tercero", nil)
	}
	registro = desenvolverData(registro)
	if idDeMapa(registro) != terceroID {
		return nil, errorDependencia(ctx,
			errors.New("la respuesta del tercero no contiene el identificador esperado"))
	}
	return registro, nil
}

func (s CaracterizacionNaturalService) validarTerceroNatural(
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
	if tipo != int64(s.TipoContribuyenteNatural) {
		return errorCaracterizacion(409,
			"El tercero no corresponde a una persona natural", nil)
	}
	return nil
}

func (s CaracterizacionNaturalService) resolverProveedor(
	ctx context.Context,
	terceroID int64,
	verb string,
	plan planCaracterizacion,
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
		if tipo != int64(s.TipoProveedorNatural) {
			return nil, "", nil, errorCaracterizacion(409,
				"El proveedor existente no corresponde a persona natural", nil)
		}
		return s.actualizarProveedorConocido(ctx, proveedor, plan)
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
	payload := map[string]any{
		"TipoProveedor":         s.TipoProveedorNatural,
		"TerceroId":             terceroID,
		"Consentimiento":        consentimiento,
		"Activo":                true,
		"UsuarioModificacionId": terceroID,
	}
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
	avisos := s.avisoProveedorDuplicado(ctx, terceroID)
	return proveedor, "creado", avisos, nil
}

func (s CaracterizacionNaturalService) actualizarProveedorConocido(
	ctx context.Context,
	proveedor map[string]any,
	plan planCaracterizacion,
) (map[string]any, string, []responses.RelationError, error) {
	proveedorID := idDeMapa(proveedor)
	if proveedorID <= 0 {
		return nil, "", nil, errorDependencia(ctx,
			errors.New("el proveedor no contiene un Id válido"))
	}
	cambios := map[string]any{}
	if plan.descripcion != nil {
		existente, _ := proveedor["DescripcionPortafolio"].(string)
		if existente != *plan.descripcion {
			cambios["DescripcionPortafolio"] = *plan.descripcion
		}
	}
	if plan.consentimiento != nil {
		existente, _ := proveedor["Consentimiento"].(bool)
		if existente != *plan.consentimiento {
			cambios["Consentimiento"] = *plan.consentimiento
		}
	}
	// El actor se toma del propio proveedor: en autoregistro el tercero que
	// escribe es el mismo proveedor. Si el registro histórico quedó en cero, se
	// actualiza en la primera escritura.
	if actor := idDeRelacion(proveedor["TerceroId"]); actor > 0 &&
		idDeRelacion(proveedor["UsuarioModificacionId"]) != actor {
		cambios["UsuarioModificacionId"] = actor
	}
	if len(cambios) == 0 {
		return proveedor, "reutilizado", nil, nil
	}
	payload := clonarPayload(proveedor)
	for clave, valor := range cambios {
		payload[clave] = valor
	}
	var respuesta map[string]any
	ok, status, err := PutProveedores(ctx,
		fmt.Sprintf("%s/%d", endpointProveedor, proveedorID), payload, &respuesta)
	if err != nil {
		return proveedor, "reutilizado", nil, errorEscrituraCaracterizacion(ctx, status, err, "actualizar el proveedor")
	}
	if !ok {
		return proveedor, "reutilizado", nil, errorEscrituraCaracterizacion(ctx, status,
			fmt.Errorf("el CRUD respondió con estado %d", status), "actualizar el proveedor")
	}
	respuesta = desenvolverData(respuesta)
	if idDeMapa(respuesta) > 0 {
		proveedor = respuesta
	} else {
		proveedor = payload
	}
	return proveedor, "actualizado", nil, nil
}

func (s CaracterizacionNaturalService) avisoProveedorDuplicado(
	ctx context.Context,
	terceroID int64,
) []responses.RelationError {
	filtros := url.Values{
		"query": {fmt.Sprintf("TerceroId:%d,Activo:true", terceroID)},
		"limit": {"5"},
	}
	var registros []map[string]any
	ok, _, err := GetAllProveedoresMapas(ctx, endpointProveedor, filtros, &registros)
	if err != nil || !ok || len(registros) <= 1 {
		return nil
	}
	return []responses.RelationError{{
		Path:    "Proveedor",
		Service: "gestion_proveedores_crud",
		Message: "Se encontraron varios proveedores activos para el tercero",
		Kind:    "duplicate_match",
	}}
}

func separarRegistrosProveedor(
	ctx context.Context,
	registros []map[string]any,
) ([]map[string]any, []map[string]any, error) {
	activos := make([]map[string]any, 0, len(registros))
	inactivos := make([]map[string]any, 0, len(registros))
	for _, registro := range registros {
		registro = desenvolverData(registro)
		if idDeMapa(registro) <= 0 {
			continue
		}
		activo, existe := registro["Activo"].(bool)
		if !existe {
			return nil, nil, errorDependencia(ctx,
				errors.New("la respuesta del proveedor no contiene Activo"))
		}
		if activo {
			activos = append(activos, registro)
		} else {
			inactivos = append(inactivos, registro)
		}
	}
	return activos, inactivos, nil
}

func (s CaracterizacionNaturalService) cargarInfoExistentes(
	ctx context.Context,
	terceroID int64,
) (*existentesInfoTercero, error) {
	filtros := url.Values{
		"query": {fmt.Sprintf("TerceroId:%d", terceroID)},
		"limit": {"1000"},
	}
	var registros []map[string]any
	ok, status, err := GetAll(ctx, endpointInfoComplementariaTercero, filtros, &registros)
	if err != nil && status != 404 {
		return nil, errorDependencia(ctx, err)
	}
	if !ok && status != 404 {
		return nil, errorDependencia(ctx,
			fmt.Errorf("consulta de info_complementaria_tercero con estado %d", status))
	}
	existentes := &existentesInfoTercero{
		porInfo: make(map[int64][]map[string]any),
		porID:   make(map[int64]map[string]any),
	}
	for _, registro := range registros {
		registro = desenvolverData(registro)
		id := idDeMapa(registro)
		if id <= 0 {
			continue
		}
		infoID := idDeRelacion(registro["InfoComplementariaId"])
		if infoID <= 0 {
			continue
		}
		existentes.porID[id] = registro
		existentes.porInfo[infoID] = append(existentes.porInfo[infoID], registro)
	}
	for _, lista := range existentes.porInfo {
		sort.SliceStable(lista, func(i, j int) bool {
			return idDeMapa(lista[i]) < idDeMapa(lista[j])
		})
	}
	return existentes, nil
}

func (s CaracterizacionNaturalService) escribirInfoComplementaria(
	ctx context.Context,
	terceroID int64,
	item planInfoComplementaria,
	existentes *existentesInfoTercero,
) (models.InfoComplementariaTerceroRegistrada, string, *responses.RelationError, error) {
	if item.padreID != nil {
		if err := s.validarPadreInfoComplementaria(ctx, terceroID, *item.padreID); err != nil {
			return models.InfoComplementariaTerceroRegistrada{}, "", nil, err
		}
	}
	if item.esOpcion {
		return s.escribirOpcionInfoComplementaria(ctx, terceroID, item, existentes)
	}
	if item.id != nil {
		return s.escribirInfoComplementariaPorID(ctx, terceroID, item, existentes)
	}
	var registros []map[string]any
	if existentes != nil {
		registros = existentes.porInfo[item.infoComplementariaID]
	} else {
		filtros := url.Values{
			"query": {
				fmt.Sprintf("TerceroId:%d,InfoComplementariaId:%d", terceroID, item.infoComplementariaID),
			},
			"sortby": {"Id"},
			"order":  {"asc"},
			"limit":  {"10"},
		}
		ok, status, err := GetAll(ctx, endpointInfoComplementariaTercero, filtros, &registros)
		if err != nil {
			return models.InfoComplementariaTerceroRegistrada{}, "",
				nil, errorDependencia(ctx, err)
		}
		if !ok {
			return models.InfoComplementariaTerceroRegistrada{}, "",
				nil, errorDependencia(ctx, fmt.Errorf("consulta de info_complementaria_tercero con estado %d", status))
		}
	}
	activos, inactivos := separarRegistrosInfo(registros)
	var status int
	var err error
	var ok bool
	var aviso *responses.RelationError
	if len(activos) > 1 {
		aviso = &responses.RelationError{
			Path:    "InfoComplementariaTercero",
			Service: "terceros_crud",
			Message: "Se encontraron varios registros activos para el mismo InfoComplementariaId; se usó el primero",
			Kind:    "ambiguous_match",
		}
	}
	registrada := models.InfoComplementariaTerceroRegistrada{
		InfoComplementariaId:             item.infoComplementariaID,
		CodigoInfoComplementaria:         item.codigo,
		CodigoGrupoInfoComplementaria:    item.codigoGrupo,
		Dato:                             item.dato,
		InfoComplementariaTerceroPadreId: item.padreID,
	}
	if len(activos) == 0 {
		if len(inactivos) > 0 {
			return registrada, "", aviso, errorCaracterizacion(409,
				"El registro de info_complementaria_tercero está inactivo y no se puede reutilizar", nil)
		}
		payload := map[string]any{
			"TerceroId":            map[string]any{"Id": terceroID},
			"InfoComplementariaId": map[string]any{"Id": item.infoComplementariaID},
			"Dato":                 item.datoJSON,
			"Activo":               true,
		}
		if item.padreID != nil {
			payload["InfoComplementariaTerceroPadreId"] = map[string]any{"Id": *item.padreID}
		}
		var respuesta map[string]any
		ok, status, err := Post(ctx, endpointInfoComplementariaTercero, payload, &respuesta)
		if err != nil {
			return registrada, "", aviso, errorEscrituraCaracterizacion(ctx, status, err, "crear info_complementaria_tercero")
		}
		if !ok {
			return registrada, "", aviso, errorEscrituraCaracterizacion(ctx, status,
				fmt.Errorf("el CRUD respondió con estado %d", status), "crear info_complementaria_tercero")
		}
		respuesta = desenvolverData(respuesta)
		registrada.Id = idDeMapa(respuesta)
		if registrada.Id <= 0 {
			return registrada, "", aviso, errorCaracterizacion(502,
				"La creación de info_complementaria_tercero no devolvió un Id utilizable", nil)
		}
		return registrada, "creado", aviso, nil
	}

	existente := activos[0]
	id := idDeMapa(existente)
	if id <= 0 {
		return registrada, "", aviso, errorDependencia(ctx,
			errors.New("el registro de info_complementaria_tercero no contiene Id"))
	}
	var registro map[string]any
	if existentes != nil {
		registro = existente
	} else {
		ok, status, err = GetOne(ctx, fmt.Sprintf("%s/%d", endpointInfoComplementariaTercero, id), &registro)
		if err != nil {
			return registrada, "", aviso, errorLecturaCaracterizacion(ctx, status,
				"No fue posible consultar info_complementaria_tercero", err)
		}
		if !ok {
			return registrada, "", aviso, errorLecturaCaracterizacion(ctx, status,
				"No fue posible consultar info_complementaria_tercero", nil)
		}
		registro = desenvolverData(registro)
	}
	if idDeMapa(registro) != id {
		return registrada, "", aviso, errorDependencia(ctx,
			errors.New("la respuesta de info_complementaria_tercero no contiene el Id esperado"))
	}
	datoExistente := decodificarDato(registro["Dato"])
	datoFinal := fusionarDato(datoExistente, item.dato)
	datoJSON := item.datoJSON
	if !datoIgual(datoFinal, item.dato) {
		bytes, err := json.Marshal(datoFinal)
		if err != nil {
			return registrada, "", aviso, errorCaracterizacion(400,
				"No fue posible serializar el dato de caracterización", err)
		}
		datoJSON = string(bytes)
	}
	padreIgual := idDeRelacion(registro["InfoComplementariaTerceroPadreId"]) == idDeRelacionOpcional(item.padreID)
	if datoIgual(datoExistente, datoFinal) && padreIgual {
		registrada.Id = id
		registrada.Dato = datoExistente
		registrada.InfoComplementariaTerceroPadreId = item.padreID
		return registrada, "reutilizado", aviso, nil
	}

	payload := clonarPayload(registro)
	payload["Dato"] = datoJSON
	payload["TerceroId"] = map[string]any{"Id": terceroID}
	payload["InfoComplementariaId"] = map[string]any{"Id": item.infoComplementariaID}
	if item.padreID != nil {
		payload["InfoComplementariaTerceroPadreId"] = map[string]any{"Id": *item.padreID}
	}
	ok, status, err = Put(ctx, fmt.Sprintf("%s/%d", endpointInfoComplementariaTercero, id), payload, &map[string]any{})
	if err != nil {
		return registrada, "", aviso, errorEscrituraCaracterizacion(ctx, status, err, "actualizar info_complementaria_tercero")
	}
	if !ok {
		return registrada, "", aviso, errorEscrituraCaracterizacion(ctx, status,
			fmt.Errorf("el CRUD respondió con estado %d", status), "actualizar info_complementaria_tercero")
	}
	registrada.Id = id
	registrada.Dato = datoFinal
	return registrada, "actualizado", aviso, nil
}

func (s CaracterizacionNaturalService) escribirOpcionInfoComplementaria(
	ctx context.Context,
	terceroID int64,
	item planInfoComplementaria,
	existentes *existentesInfoTercero,
) (models.InfoComplementariaTerceroRegistrada, string, *responses.RelationError, error) {
	registrada := models.InfoComplementariaTerceroRegistrada{
		InfoComplementariaId:             item.infoComplementariaID,
		CodigoInfoComplementaria:         item.codigo,
		CodigoGrupoInfoComplementaria:    item.codigoGrupo,
		Dato:                             "",
		InfoComplementariaTerceroPadreId: item.padreID,
	}
	if item.padreID != nil {
		if err := s.validarPadreInfoComplementaria(ctx, terceroID, *item.padreID); err != nil {
			return registrada, "", nil, err
		}
	}
	if existentes == nil {
		cargados, err := s.cargarInfoExistentes(ctx, terceroID)
		if err != nil {
			return registrada, "", nil, err
		}
		existentes = cargados
	}

	// Una sola opción activa por grupo: se desactivan las demás. Los registros
	// DOC_* comparten grupo con las opciones, pero no son opciones.
	for id, registro := range existentes.porID {
		if id <= 0 {
			continue
		}
		if activo, ok := registro["Activo"].(bool); !ok || !activo {
			continue
		}
		icID := idDeRelacion(registro["InfoComplementariaId"])
		if icID <= 0 || icID == item.infoComplementariaID {
			continue
		}
		grupo, codigo, err := s.grupoYCodigoDeRegistro(ctx, registro)
		if err != nil {
			return registrada, "", nil, err
		}
		if !strings.EqualFold(strings.TrimSpace(grupo), strings.TrimSpace(item.grupoOpcion)) {
			continue
		}
		if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(codigo)), "DOC_") {
			continue
		}
		if err := s.desactivarInfoComplementaria(ctx, terceroID, id, registro); err != nil {
			return registrada, "", nil, err
		}
	}

	seleccionados := existentes.porInfo[item.infoComplementariaID]
	activos, inactivos := separarRegistrosInfo(seleccionados)
	if len(activos) > 0 {
		registro := activos[0]
		id := idDeMapa(registro)
		if id <= 0 {
			return registrada, "", nil, errorDependencia(ctx,
				errors.New("el registro de info_complementaria_tercero no contiene Id"))
		}
		for _, extra := range activos[1:] {
			if err := s.desactivarInfoComplementaria(ctx, terceroID, idDeMapa(extra), extra); err != nil {
				return registrada, "", nil, err
			}
		}
		accion := "reutilizado"
		if datoActual, ok := decodificarDato(registro["Dato"]).(string); !ok || datoActual != "" {
			if err := s.actualizarDatoInfoComplementaria(ctx, terceroID, id, item, registro); err != nil {
				return registrada, "", nil, err
			}
			accion = "actualizado"
		}
		registrada.Id = id
		return registrada, accion, nil, nil
	}
	if len(inactivos) > 0 {
		registro := inactivos[0]
		id := idDeMapa(registro)
		if id <= 0 {
			return registrada, "", nil, errorDependencia(ctx,
				errors.New("el registro de info_complementaria_tercero no contiene Id"))
		}
		if err := s.reactivarOpcionInfoComplementaria(ctx, terceroID, id, item, registro); err != nil {
			return registrada, "", nil, err
		}
		registrada.Id = id
		return registrada, "actualizado", nil, nil
	}

	payload := map[string]any{
		"TerceroId":            map[string]any{"Id": terceroID},
		"InfoComplementariaId": map[string]any{"Id": item.infoComplementariaID},
		"Dato":                 item.datoJSON,
		"Activo":               true,
	}
	if item.padreID != nil {
		payload["InfoComplementariaTerceroPadreId"] = map[string]any{"Id": *item.padreID}
	}
	var respuesta map[string]any
	ok, status, err := Post(ctx, endpointInfoComplementariaTercero, payload, &respuesta)
	if err != nil {
		return registrada, "", nil,
			errorEscrituraCaracterizacion(ctx, status, err, "crear info_complementaria_tercero")
	}
	if !ok {
		return registrada, "", nil, errorEscrituraCaracterizacion(ctx, status,
			fmt.Errorf("el CRUD respondió con estado %d", status), "crear info_complementaria_tercero")
	}
	respuesta = desenvolverData(respuesta)
	registrada.Id = idDeMapa(respuesta)
	if registrada.Id <= 0 {
		return registrada, "", nil, errorCaracterizacion(502,
			"La creación de info_complementaria_tercero no devolvió un Id utilizable", nil)
	}
	return registrada, "creado", nil, nil
}

func (s CaracterizacionNaturalService) grupoYCodigoDeRegistro(
	ctx context.Context,
	registro map[string]any,
) (string, string, error) {
	icID := idDeRelacion(registro["InfoComplementariaId"])
	codigo := ""
	grupo := ""
	if info, ok := registro["InfoComplementariaId"].(map[string]any); ok {
		codigo, _ = info["CodigoAbreviacion"].(string)
		if infoGrupo, ok := info["GrupoInfoComplementariaId"].(map[string]any); ok {
			grupo, _ = infoGrupo["CodigoAbreviacion"].(string)
		}
	}
	if (codigo == "" || grupo == "") && icID > 0 {
		detalle, ok, err := s.Catalogos.InfoComplementariaPorID(ctx, icID)
		if err != nil {
			return "", "", errorDependencia(ctx, err)
		}
		if ok {
			if codigo == "" {
				codigo = detalle.Codigo
			}
			if grupo == "" {
				grupo = detalle.CodigoGrupo
			}
		}
	}
	return strings.TrimSpace(grupo), strings.TrimSpace(codigo), nil
}

func (s CaracterizacionNaturalService) desactivarInfoComplementaria(
	ctx context.Context,
	terceroID int64,
	id int64,
	registro map[string]any,
) error {
	payload := clonarPayload(registro)
	payload["Activo"] = false
	payload["TerceroId"] = map[string]any{"Id": terceroID}
	payload["InfoComplementariaId"] = map[string]any{
		"Id": idDeRelacion(registro["InfoComplementariaId"]),
	}
	ok, status, err := Put(ctx,
		fmt.Sprintf("%s/%d", endpointInfoComplementariaTercero, id), payload, &map[string]any{})
	if err != nil {
		return errorEscrituraCaracterizacion(ctx, status, err, "desactivar info_complementaria_tercero")
	}
	if !ok {
		return errorEscrituraCaracterizacion(ctx, status,
			fmt.Errorf("el CRUD respondió con estado %d", status), "desactivar info_complementaria_tercero")
	}
	return nil
}

func (s CaracterizacionNaturalService) actualizarDatoInfoComplementaria(
	ctx context.Context,
	terceroID int64,
	id int64,
	item planInfoComplementaria,
	registro map[string]any,
) error {
	payload := clonarPayload(registro)
	payload["Dato"] = item.datoJSON
	payload["TerceroId"] = map[string]any{"Id": terceroID}
	payload["InfoComplementariaId"] = map[string]any{"Id": item.infoComplementariaID}
	ok, status, err := Put(ctx,
		fmt.Sprintf("%s/%d", endpointInfoComplementariaTercero, id), payload, &map[string]any{})
	if err != nil {
		return errorEscrituraCaracterizacion(ctx, status, err, "actualizar info_complementaria_tercero")
	}
	if !ok {
		return errorEscrituraCaracterizacion(ctx, status,
			fmt.Errorf("el CRUD respondió con estado %d", status), "actualizar info_complementaria_tercero")
	}
	return nil
}

func (s CaracterizacionNaturalService) reactivarOpcionInfoComplementaria(
	ctx context.Context,
	terceroID int64,
	id int64,
	item planInfoComplementaria,
	registro map[string]any,
) error {
	payload := clonarPayload(registro)
	payload["Activo"] = true
	payload["Dato"] = item.datoJSON
	payload["TerceroId"] = map[string]any{"Id": terceroID}
	payload["InfoComplementariaId"] = map[string]any{"Id": item.infoComplementariaID}
	ok, status, err := Put(ctx,
		fmt.Sprintf("%s/%d", endpointInfoComplementariaTercero, id), payload, &map[string]any{})
	if err != nil {
		return errorEscrituraCaracterizacion(ctx, status, err, "reactivar info_complementaria_tercero")
	}
	if !ok {
		return errorEscrituraCaracterizacion(ctx, status,
			fmt.Errorf("el CRUD respondió con estado %d", status), "reactivar info_complementaria_tercero")
	}
	return nil
}

func (s CaracterizacionNaturalService) validarPadreInfoComplementaria(
	ctx context.Context,
	terceroID int64,
	padreID int64,
) error {
	if padreID <= 0 {
		return errorCaracterizacion(400,
			"InfoComplementariaTerceroPadreId debe ser un entero positivo", nil)
	}
	var registro map[string]any
	ok, status, err := GetOne(ctx,
		fmt.Sprintf("%s/%d", endpointInfoComplementariaTercero, padreID), &registro)
	if err != nil {
		return errorLecturaCaracterizacion(ctx, status,
			"No fue posible consultar el registro padre de info_complementaria_tercero", err)
	}
	if !ok {
		return errorCaracterizacion(400,
			"El registro padre de info_complementaria_tercero no existe", nil)
	}
	registro = desenvolverData(registro)
	if idDeRelacion(registro["TerceroId"]) != terceroID {
		return errorCaracterizacion(409,
			"El registro padre de info_complementaria_tercero no pertenece al tercero", nil)
	}
	if activo, existe := registro["Activo"].(bool); !existe || !activo {
		return errorCaracterizacion(409,
			"El registro padre de info_complementaria_tercero está inactivo", nil)
	}
	return nil
}

func (s CaracterizacionNaturalService) escribirInfoComplementariaPorID(
	ctx context.Context,
	terceroID int64,
	item planInfoComplementaria,
	existentes *existentesInfoTercero,
) (models.InfoComplementariaTerceroRegistrada, string, *responses.RelationError, error) {
	registrada := models.InfoComplementariaTerceroRegistrada{
		Id:                               *item.id,
		InfoComplementariaId:             item.infoComplementariaID,
		CodigoInfoComplementaria:         item.codigo,
		CodigoGrupoInfoComplementaria:    item.codigoGrupo,
		Dato:                             item.dato,
		InfoComplementariaTerceroPadreId: item.padreID,
	}
	var registro map[string]any
	var ok bool
	var status int
	var err error
	if existentes != nil {
		registro = existentes.porID[*item.id]
	}
	if registro == nil {
		ok, status, err = GetOne(ctx,
			fmt.Sprintf("%s/%d", endpointInfoComplementariaTercero, *item.id), &registro)
		if err != nil {
			return registrada, "", nil, errorLecturaCaracterizacion(ctx, status,
				"No fue posible consultar info_complementaria_tercero", err)
		}
		if !ok {
			return registrada, "", nil, errorLecturaCaracterizacion(ctx, status,
				"No fue posible consultar info_complementaria_tercero", nil)
		}
	}
	registro = desenvolverData(registro)
	if idDeMapa(registro) != *item.id {
		return registrada, "", nil, errorDependencia(ctx,
			errors.New("la respuesta de info_complementaria_tercero no contiene el Id esperado"))
	}
	if idDeRelacion(registro["TerceroId"]) != terceroID {
		return registrada, "", nil, errorCaracterizacion(409,
			"El registro de info_complementaria_tercero no pertenece al tercero", nil)
	}
	if item.infoComplementariaID > 0 &&
		idDeRelacion(registro["InfoComplementariaId"]) != item.infoComplementariaID {
		return registrada, "", nil, errorCaracterizacion(409,
			"El registro de info_complementaria_tercero no corresponde al InfoComplementariaId enviado", nil)
	}
	activo, existe := registro["Activo"].(bool)
	if !existe {
		return registrada, "", nil, errorDependencia(ctx,
			errors.New("la respuesta de info_complementaria_tercero no contiene Activo"))
	}
	if !activo {
		return registrada, "", nil, errorCaracterizacion(409,
			"El registro de info_complementaria_tercero está inactivo", nil)
	}
	datoFinal := fusionarDato(decodificarDato(registro["Dato"]), item.dato)
	datoJSON := item.datoJSON
	if !datoIgual(datoFinal, item.dato) {
		bytes, err := json.Marshal(datoFinal)
		if err != nil {
			return registrada, "", nil, errorCaracterizacion(400,
				"No fue posible serializar el dato de caracterización", err)
		}
		datoJSON = string(bytes)
	}
	payload := clonarPayload(registro)
	payload["Dato"] = datoJSON
	payload["TerceroId"] = map[string]any{"Id": terceroID}
	payload["InfoComplementariaId"] = map[string]any{"Id": item.infoComplementariaID}
	if item.padreID != nil {
		payload["InfoComplementariaTerceroPadreId"] = map[string]any{"Id": *item.padreID}
	}
	ok, status, err = Put(ctx,
		fmt.Sprintf("%s/%d", endpointInfoComplementariaTercero, *item.id),
		payload, &map[string]any{})
	if err != nil {
		return registrada, "", nil, errorEscrituraCaracterizacion(ctx, status, err,
			"actualizar info_complementaria_tercero")
	}
	if !ok {
		return registrada, "", nil, errorEscrituraCaracterizacion(ctx, status,
			fmt.Errorf("el CRUD respondió con estado %d", status),
			"actualizar info_complementaria_tercero")
	}
	registrada.Dato = datoFinal
	return registrada, "actualizado", nil, nil
}

func separarRegistrosInfo(
	registros []map[string]any,
) ([]map[string]any, []map[string]any) {
	activos := make([]map[string]any, 0, len(registros))
	inactivos := make([]map[string]any, 0, len(registros))
	for _, registro := range registros {
		registro = desenvolverData(registro)
		if idDeMapa(registro) <= 0 {
			continue
		}
		if activo, ok := registro["Activo"].(bool); ok && activo {
			activos = append(activos, registro)
		} else {
			inactivos = append(inactivos, registro)
		}
	}
	return activos, inactivos
}

// datoIgual compara valores decodificados de JSON. Los números pueden llegar
// como float64 desde el CRUD y como int64 desde el payload; se comparan por
// valor para que la idempotencia no dependa del tipo Go.
func datoIgual(a any, b any) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	if af, ok := numeroFloat(a); ok {
		if bf, ok := numeroFloat(b); ok {
			return floatIgual(af, bf)
		}
	}
	if mapaA, ok := a.(map[string]any); ok {
		mapaB, ok := b.(map[string]any)
		if !ok || len(mapaA) != len(mapaB) {
			return false
		}
		for clave, valorA := range mapaA {
			valorB, existe := mapaB[clave]
			if !existe || !datoIgual(valorA, valorB) {
				return false
			}
		}
		return true
	}
	if listaA, ok := a.([]any); ok {
		listaB, ok := b.([]any)
		if !ok || len(listaA) != len(listaB) {
			return false
		}
		for i := range listaA {
			if !datoIgual(listaA[i], listaB[i]) {
				return false
			}
		}
		return true
	}
	return reflect.DeepEqual(a, b)
}

// fusionarDato conserva las claves existentes de un objeto y superpone las
// nuevas, de forma recursiva para los objetos anidados. Para valores escalares
// reemplaza el valor completo.
func fusionarDato(existente any, nuevo any) any {
	mapaNuevo, esMapaNuevo := nuevo.(map[string]any)
	if !esMapaNuevo {
		return nuevo
	}
	resultado := map[string]any{}
	if mapaExistente, ok := existente.(map[string]any); ok {
		for clave, valor := range mapaExistente {
			resultado[clave] = valor
		}
	}
	for clave, valor := range mapaNuevo {
		if existenteValor, ok := resultado[clave]; ok {
			resultado[clave] = fusionarDato(existenteValor, valor)
			continue
		}
		resultado[clave] = valor
	}
	return resultado
}

func decodificarDato(valor any) any {
	texto, ok := valor.(string)
	if !ok {
		return valor
	}
	var dato any
	if err := json.Unmarshal([]byte(texto), &dato); err == nil {
		return dato
	}
	return texto
}

func idDeRelacionOpcional(valor *int64) int64 {
	if valor == nil {
		return 0
	}
	return *valor
}

func (s CaracterizacionNaturalService) escribirProveedorNatural(
	ctx context.Context,
	terceroID int64,
	proveedorID int64,
	in *models.ProveedorNaturalInput,
) (map[string]any, string, error) {
	var registro map[string]any
	ok, status, err := GetOneProveedores(ctx,
		fmt.Sprintf("%s/%d", endpointProveedorNatural, proveedorID), &registro)
	if err != nil && status != 404 {
		return nil, "", errorLecturaCaracterizacion(ctx, status,
			"No fue posible consultar proveedor_natural", err)
	}
	if ok {
		registro = desenvolverData(registro)
		if idDeMapa(registro) != proveedorID {
			return nil, "", errorDependencia(ctx,
				errors.New("la respuesta de proveedor_natural no contiene el Id esperado"))
		}
		activo, existe := registro["Activo"].(bool)
		if !existe {
			return nil, "", errorDependencia(ctx,
				errors.New("la respuesta de proveedor_natural no contiene Activo"))
		}
		if !activo {
			return nil, "", errorCaracterizacion(409,
				"El detalle de proveedor natural está inactivo", nil)
		}
		if proveedorNaturalIgual(registro, in) &&
			idDeRelacion(registro["UsuarioModificacionId"]) == terceroID {
			return registro, "reutilizado", nil
		}
		payload := clonarPayload(registro)
		aplicarCamposProveedorNatural(payload, in)
		aplicarUsuarioModificacion(payload, terceroID)
		ok, status, err = PutProveedores(ctx,
			fmt.Sprintf("%s/%d", endpointProveedorNatural, proveedorID), payload, &map[string]any{})
		if err != nil {
			return nil, "", errorEscrituraCaracterizacion(ctx, status, err, "actualizar proveedor_natural")
		}
		if !ok {
			return nil, "", errorEscrituraCaracterizacion(ctx, status,
				fmt.Errorf("el CRUD respondió con estado %d", status), "actualizar proveedor_natural")
		}
		return payload, "actualizado", nil
	}

	if in.PaisNacimientoId == nil {
		return nil, "", errorCaracterizacion(400,
			"ProveedorNatural.PaisNacimientoId es obligatorio al crear el detalle", nil)
	}
	payload := map[string]any{
		"Id":     proveedorID,
		"Activo": true,
	}
	aplicarCamposProveedorNatural(payload, in)
	aplicarUsuarioModificacion(payload, terceroID)
	ok, status, err = PostProveedores(ctx, endpointProveedorNatural, payload, &map[string]any{})
	if err != nil {
		return nil, "", errorEscrituraCaracterizacion(ctx, status, err, "crear proveedor_natural")
	}
	if !ok {
		return nil, "", errorEscrituraCaracterizacion(ctx, status,
			fmt.Errorf("el CRUD respondió con estado %d", status), "crear proveedor_natural")
	}
	return payload, "creado", nil
}

func proveedorNaturalIgual(
	registro map[string]any,
	in *models.ProveedorNaturalInput,
) bool {
	if in.PaisNacimientoId != nil && idDeRelacion(registro["PaisNacimientoId"]) != *in.PaisNacimientoId {
		return false
	}
	if in.Cargo != nil && cadenaCuenta(registro["Cargo"]) != *in.Cargo {
		return false
	}
	booleanos := []struct {
		campo string
		valor *bool
	}{
		{"MedicinaPrepagada", in.MedicinaPrepagada},
		{"DependienteHijoMejorEdad", in.DependienteHijoMejorEdad},
		{"DependienteHijoMenos23Estudiando", in.DependienteHijoMenos23Estudiando},
		{"DependienteHijoMas23Discapacitado", in.DependienteHijoMas23Discapacitado},
		{"DependienteConyuge", in.DependienteConyuge},
		{"DependientePadreOHermano", in.DependientePadreOHermano},
		{"Pensionado", in.Pensionado},
		{"ResponsableIva", in.ResponsableIva},
		{"ResideColombia", in.ResideColombia},
		{"PensionVoluntaria", in.PensionVoluntaria},
	}
	for _, item := range booleanos {
		if item.valor == nil {
			continue
		}
		existente, _ := registro[item.campo].(bool)
		if existente != *item.valor {
			return false
		}
	}
	numeros := []struct {
		campo string
		valor *float64
	}{
		{"ValorUvtPrepagada", in.ValorUvtPrepagada},
		{"InteresViviendaAfc", in.InteresViviendaAfc},
		{"ValorAfc", in.ValorAfc},
		{"ValorPensionVoluntaria", in.ValorPensionVoluntaria},
	}
	for _, item := range numeros {
		if item.valor == nil {
			continue
		}
		existente, ok := numeroFloat(registro[item.campo])
		if !ok || !floatIgual(existente, *item.valor) {
			return false
		}
	}
	enteros := []struct {
		campo string
		valor *int64
	}{
		{"ExperienciaLaboral", in.ExperienciaLaboral},
		{"ExperienciaProfesional", in.ExperienciaProfesional},
	}
	for _, item := range enteros {
		if item.valor == nil {
			continue
		}
		if idDeRelacion(registro[item.campo]) != *item.valor {
			return false
		}
	}
	return true
}

func numeroFloat(valor any) (float64, bool) {
	switch v := valor.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case json.Number:
		n, err := v.Float64()
		return n, err == nil
	default:
		return 0, false
	}
}

func floatIgual(a, b float64) bool {
	const epsilon = 1e-9
	diferencia := a - b
	if diferencia < 0 {
		diferencia = -diferencia
	}
	return diferencia <= epsilon
}

func aplicarCamposProveedorNatural(
	payload map[string]any,
	in *models.ProveedorNaturalInput,
) {
	if in.PaisNacimientoId != nil {
		payload["PaisNacimientoId"] = *in.PaisNacimientoId
	}
	if in.Cargo != nil {
		payload["Cargo"] = *in.Cargo
	}
	if in.MedicinaPrepagada != nil {
		payload["MedicinaPrepagada"] = *in.MedicinaPrepagada
	}
	if in.ValorUvtPrepagada != nil {
		payload["ValorUvtPrepagada"] = *in.ValorUvtPrepagada
	}
	if in.InteresViviendaAfc != nil {
		payload["InteresViviendaAfc"] = *in.InteresViviendaAfc
	}
	if in.DependienteHijoMejorEdad != nil {
		payload["DependienteHijoMejorEdad"] = *in.DependienteHijoMejorEdad
	}
	if in.DependienteHijoMenos23Estudiando != nil {
		payload["DependienteHijoMenos23Estudiando"] = *in.DependienteHijoMenos23Estudiando
	}
	if in.DependienteHijoMas23Discapacitado != nil {
		payload["DependienteHijoMas23Discapacitado"] = *in.DependienteHijoMas23Discapacitado
	}
	if in.DependienteConyuge != nil {
		payload["DependienteConyuge"] = *in.DependienteConyuge
	}
	if in.DependientePadreOHermano != nil {
		payload["DependientePadreOHermano"] = *in.DependientePadreOHermano
	}
	if in.Pensionado != nil {
		payload["Pensionado"] = *in.Pensionado
	}
	if in.ValorAfc != nil {
		payload["ValorAfc"] = *in.ValorAfc
	}
	if in.ResponsableIva != nil {
		payload["ResponsableIva"] = *in.ResponsableIva
	}
	if in.ResideColombia != nil {
		payload["ResideColombia"] = *in.ResideColombia
	}
	if in.PensionVoluntaria != nil {
		payload["PensionVoluntaria"] = *in.PensionVoluntaria
	}
	if in.ValorPensionVoluntaria != nil {
		payload["ValorPensionVoluntaria"] = *in.ValorPensionVoluntaria
	}
	if in.ExperienciaLaboral != nil {
		payload["ExperienciaLaboral"] = *in.ExperienciaLaboral
	}
	if in.ExperienciaProfesional != nil {
		payload["ExperienciaProfesional"] = *in.ExperienciaProfesional
	}
}

func (s CaracterizacionNaturalService) escribirPerfil(
	ctx context.Context,
	terceroID int64,
	proveedorID int64,
	in models.PerfilProveedorInput,
) (int64, string, *responses.RelationError, error) {
	if in.Id != nil {
		return s.escribirPerfilPorID(ctx, terceroID, proveedorID, in)
	}
	filtros := url.Values{
		"query": {
			fmt.Sprintf("ProveedorId:%d,PerfilId:%d", proveedorID, in.PerfilId),
		},
		"limit": {"10"},
	}
	var registros []map[string]any
	ok, status, err := GetAllProveedoresMapas(ctx, endpointPerfilProveedor, filtros, &registros)
	if err != nil {
		return 0, "", nil, errorDependencia(ctx, err)
	}
	if !ok {
		return 0, "", nil, errorDependencia(ctx,
			fmt.Errorf("consulta de perfil_proveedor con estado %d", status))
	}
	activos, inactivos := separarActivosProveedor(registros)
	var aviso *responses.RelationError
	if len(activos) > 1 {
		aviso = &responses.RelationError{
			Path:    "PerfilProveedor",
			Service: "gestion_proveedores_crud",
			Message: "Se encontraron varios perfiles activos con el mismo PerfilId; se usó el primero",
			Kind:    "ambiguous_match",
		}
	}
	if len(activos) > 0 {
		existente := activos[0]
		id := idDeMapa(existente)
		if idDeRelacion(existente["UsuarioModificacionId"]) == terceroID {
			return id, "reutilizado", aviso, nil
		}
		payload := clonarPayload(existente)
		aplicarUsuarioModificacion(payload, terceroID)
		ok, status, err = PutProveedores(ctx,
			fmt.Sprintf("%s/%d", endpointPerfilProveedor, id), payload, &map[string]any{})
		if err != nil {
			return id, "", aviso, errorEscrituraCaracterizacion(ctx, status, err, "actualizar perfil_proveedor")
		}
		if !ok {
			return id, "", aviso, errorEscrituraCaracterizacion(ctx, status,
				fmt.Errorf("el CRUD respondió con estado %d", status), "actualizar perfil_proveedor")
		}
		return id, "actualizado", aviso, nil
	}
	if len(inactivos) > 0 {
		return 0, "", aviso, errorCaracterizacion(409,
			"El perfil del proveedor está inactivo y no se puede reutilizar", nil)
	}
	payload := aplicarUsuarioModificacion(map[string]any{
		"ProveedorId": map[string]any{"Id": proveedorID},
		"PerfilId":    in.PerfilId,
		"Activo":      true,
	}, terceroID)
	var respuesta map[string]any
	ok, status, err = PostProveedores(ctx, endpointPerfilProveedor, payload, &respuesta)
	if err != nil {
		return 0, "", aviso, errorEscrituraCaracterizacion(ctx, status, err, "crear perfil_proveedor")
	}
	if !ok {
		return 0, "", aviso, errorEscrituraCaracterizacion(ctx, status,
			fmt.Errorf("el CRUD respondió con estado %d", status), "crear perfil_proveedor")
	}
	id := idDeMapa(desenvolverData(respuesta))
	if id <= 0 {
		return 0, "", aviso, errorCaracterizacion(502,
			"La creación de perfil_proveedor no devolvió un Id utilizable", nil)
	}
	return id, "creado", aviso, nil
}

func (s CaracterizacionNaturalService) escribirContacto(
	ctx context.Context,
	terceroID int64,
	proveedorID int64,
	in models.ContactoProveedorInput,
) (int64, string, *responses.RelationError, error) {
	if in.Id != nil {
		return s.actualizarContactoPorID(ctx, terceroID, proveedorID, in)
	}
	filtros := url.Values{
		"query": {
			fmt.Sprintf("ProveedorId:%d,TipoCanalContactoId:%d,Proposito:%s",
				proveedorID, in.TipoCanalContactoId, in.Proposito),
		},
		"limit": {"10"},
	}
	var registros []map[string]any
	ok, status, err := GetAllProveedoresMapas(ctx, endpointContacto, filtros, &registros)
	if err != nil {
		return 0, "", nil, errorDependencia(ctx, err)
	}
	if !ok {
		return 0, "", nil, errorDependencia(ctx,
			fmt.Errorf("consulta de contacto con estado %d", status))
	}
	activos, inactivos := separarActivosProveedor(registros)
	var aviso *responses.RelationError
	if len(activos) > 1 {
		aviso = &responses.RelationError{
			Path:    "Contacto",
			Service: "gestion_proveedores_crud",
			Message: "Se encontraron varios contactos activos con el mismo canal y propósito; se usó el primero",
			Kind:    "ambiguous_match",
		}
	}
	if len(activos) > 0 {
		return s.actualizarContacto(ctx, terceroID, activos[0], proveedorID, in, aviso)
	}
	if len(inactivos) > 0 {
		return 0, "", aviso, errorCaracterizacion(409,
			"El contacto del proveedor está inactivo y no se puede reutilizar", nil)
	}
	payload := payloadContacto(terceroID, proveedorID, in)
	payload["Activo"] = true
	var respuesta map[string]any
	ok, status, err = PostProveedores(ctx, endpointContacto, payload, &respuesta)
	if err != nil {
		return 0, "", aviso, errorEscrituraCaracterizacion(ctx, status, err, "crear contacto")
	}
	if !ok {
		return 0, "", aviso, errorEscrituraCaracterizacion(ctx, status,
			fmt.Errorf("el CRUD respondió con estado %d", status), "crear contacto")
	}
	id := idDeMapa(desenvolverData(respuesta))
	if id <= 0 {
		return 0, "", aviso, errorCaracterizacion(502,
			"La creación del contacto no devolvió un Id utilizable", nil)
	}
	return id, "creado", aviso, nil
}

func (s CaracterizacionNaturalService) actualizarContactoPorID(
	ctx context.Context,
	terceroID int64,
	proveedorID int64,
	in models.ContactoProveedorInput,
) (int64, string, *responses.RelationError, error) {
	var registro map[string]any
	ok, status, err := GetOneProveedores(ctx, fmt.Sprintf("%s/%d", endpointContacto, *in.Id), &registro)
	if err != nil {
		return 0, "", nil, errorLecturaCaracterizacion(ctx, status,
			"No fue posible consultar el contacto", err)
	}
	if !ok {
		return 0, "", nil, errorLecturaCaracterizacion(ctx, status,
			"No fue posible consultar el contacto", nil)
	}
	registro = desenvolverData(registro)
	if idDeMapa(registro) != *in.Id {
		return 0, "", nil, errorDependencia(ctx,
			errors.New("la respuesta del contacto no contiene el Id esperado"))
	}
	if idDeRelacion(registro["ProveedorId"]) != proveedorID {
		return 0, "", nil, errorCaracterizacion(409,
			"El contacto no pertenece al proveedor del tercero", nil)
	}
	return s.actualizarContacto(ctx, terceroID, registro, proveedorID, in, nil)
}

func (s CaracterizacionNaturalService) actualizarContacto(
	ctx context.Context,
	terceroID int64,
	existente map[string]any,
	proveedorID int64,
	in models.ContactoProveedorInput,
	aviso *responses.RelationError,
) (int64, string, *responses.RelationError, error) {
	id := idDeMapa(existente)
	if id <= 0 {
		return 0, "", aviso, errorDependencia(ctx,
			errors.New("el contacto no contiene Id"))
	}
	activo, existe := existente["Activo"].(bool)
	if !existe {
		return 0, "", aviso, errorDependencia(ctx,
			errors.New("el contacto no contiene Activo"))
	}
	if !activo {
		return 0, "", aviso, errorCaracterizacion(409,
			"El contacto del proveedor está inactivo", nil)
	}
	if contactoIgual(existente, proveedorID, in) &&
		idDeRelacion(existente["UsuarioModificacionId"]) == terceroID {
		return id, "reutilizado", aviso, nil
	}
	payload := clonarPayload(existente)
	for clave, valor := range payloadContacto(terceroID, proveedorID, in) {
		payload[clave] = valor
	}
	aplicarUsuarioModificacion(payload, terceroID)
	ok, status, err := PutProveedores(ctx, fmt.Sprintf("%s/%d", endpointContacto, id), payload, &map[string]any{})
	if err != nil {
		return id, "", aviso, errorEscrituraCaracterizacion(ctx, status, err, "actualizar contacto")
	}
	if !ok {
		return id, "", aviso, errorEscrituraCaracterizacion(ctx, status,
			fmt.Errorf("el CRUD respondió con estado %d", status), "actualizar contacto")
	}
	return id, "actualizado", aviso, nil
}

func payloadContacto(
	terceroID int64,
	proveedorID int64,
	in models.ContactoProveedorInput,
) map[string]any {
	payload := map[string]any{
		"ProveedorId":         map[string]any{"Id": proveedorID},
		"TipoCanalContactoId": in.TipoCanalContactoId,
		"Proposito":           in.Proposito,
		"ValorEspecifico":     in.ValorEspecifico,
	}
	if in.NombreContacto != nil {
		payload["NombreContacto"] = *in.NombreContacto
	}
	if in.Extension != nil {
		payload["Extension"] = *in.Extension
	}
	return aplicarUsuarioModificacion(payload, terceroID)
}

func contactoIgual(
	registro map[string]any,
	proveedorID int64,
	in models.ContactoProveedorInput,
) bool {
	if idDeRelacion(registro["ProveedorId"]) != proveedorID ||
		idDeRelacion(registro["TipoCanalContactoId"]) != in.TipoCanalContactoId ||
		cadenaCuenta(registro["Proposito"]) != in.Proposito ||
		cadenaCuenta(registro["ValorEspecifico"]) != in.ValorEspecifico {
		return false
	}
	if in.NombreContacto != nil && cadenaCuenta(registro["NombreContacto"]) != *in.NombreContacto {
		return false
	}
	if in.Extension != nil && cadenaCuenta(registro["Extension"]) != *in.Extension {
		return false
	}
	return true
}

func (s CaracterizacionNaturalService) escribirActividad(
	ctx context.Context,
	terceroID int64,
	proveedorID int64,
	in models.ActividadEconomicaProveedorInput,
) (int64, string, *responses.RelationError, error) {
	if in.Id != nil {
		return s.escribirActividadPorID(ctx, terceroID, proveedorID, in)
	}
	filtros := url.Values{
		"query": {
			fmt.Sprintf("ProveedorId:%d,ActividadEconomicaId:%d",
				proveedorID, in.ActividadEconomicaId),
		},
		"limit": {"10"},
	}
	var registros []map[string]any
	ok, status, err := GetAllProveedoresMapas(ctx, endpointActividadEconomica, filtros, &registros)
	if err != nil {
		return 0, "", nil, errorDependencia(ctx, err)
	}
	if !ok {
		return 0, "", nil, errorDependencia(ctx,
			fmt.Errorf("consulta de actividad_economica_proveedor con estado %d", status))
	}
	activos, inactivos := separarActivosProveedor(registros)
	var aviso *responses.RelationError
	if len(activos) > 1 {
		aviso = &responses.RelationError{
			Path:    "ActividadEconomicaProveedor",
			Service: "gestion_proveedores_crud",
			Message: "Se encontraron varias actividades activas con el mismo ActividadEconomicaId; se usó la primera",
			Kind:    "ambiguous_match",
		}
	}
	if len(activos) > 0 {
		existente := activos[0]
		id := idDeMapa(existente)
		if idDeRelacion(existente["UsuarioModificacionId"]) == terceroID {
			return id, "reutilizado", aviso, nil
		}
		payload := clonarPayload(existente)
		aplicarUsuarioModificacion(payload, terceroID)
		ok, status, err = PutProveedores(ctx,
			fmt.Sprintf("%s/%d", endpointActividadEconomica, id), payload, &map[string]any{})
		if err != nil {
			return id, "", aviso, errorEscrituraCaracterizacion(ctx, status, err, "actualizar actividad_economica_proveedor")
		}
		if !ok {
			return id, "", aviso, errorEscrituraCaracterizacion(ctx, status,
				fmt.Errorf("el CRUD respondió con estado %d", status), "actualizar actividad_economica_proveedor")
		}
		return id, "actualizado", aviso, nil
	}
	if len(inactivos) > 0 {
		return 0, "", aviso, errorCaracterizacion(409,
			"La actividad económica del proveedor está inactiva y no se puede reutilizar", nil)
	}
	payload := aplicarUsuarioModificacion(map[string]any{
		"ProveedorId":          map[string]any{"Id": proveedorID},
		"ActividadEconomicaId": in.ActividadEconomicaId,
		"Activo":               true,
	}, terceroID)
	var respuesta map[string]any
	ok, status, err = PostProveedores(ctx, endpointActividadEconomica, payload, &respuesta)
	if err != nil {
		return 0, "", aviso, errorEscrituraCaracterizacion(ctx, status, err, "crear actividad_economica_proveedor")
	}
	if !ok {
		return 0, "", aviso, errorEscrituraCaracterizacion(ctx, status,
			fmt.Errorf("el CRUD respondió con estado %d", status), "crear actividad_economica_proveedor")
	}
	id := idDeMapa(desenvolverData(respuesta))
	if id <= 0 {
		return 0, "", aviso, errorCaracterizacion(502,
			"La creación de actividad_economica_proveedor no devolvió un Id utilizable", nil)
	}
	return id, "creado", aviso, nil
}

func (s CaracterizacionNaturalService) escribirPerfilPorID(
	ctx context.Context,
	terceroID int64,
	proveedorID int64,
	in models.PerfilProveedorInput,
) (int64, string, *responses.RelationError, error) {
	var registro map[string]any
	ok, status, err := GetOneProveedores(ctx,
		fmt.Sprintf("%s/%d", endpointPerfilProveedor, *in.Id), &registro)
	if err != nil {
		return 0, "", nil, errorLecturaCaracterizacion(ctx, status,
			"No fue posible consultar perfil_proveedor", err)
	}
	if !ok {
		return 0, "", nil, errorLecturaCaracterizacion(ctx, status,
			"No fue posible consultar perfil_proveedor", nil)
	}
	registro = desenvolverData(registro)
	if idDeMapa(registro) != *in.Id {
		return 0, "", nil, errorDependencia(ctx,
			errors.New("la respuesta de perfil_proveedor no contiene el Id esperado"))
	}
	if idDeRelacion(registro["ProveedorId"]) != proveedorID {
		return 0, "", nil, errorCaracterizacion(409,
			"El perfil no pertenece al proveedor del tercero", nil)
	}
	activo, existe := registro["Activo"].(bool)
	if !existe {
		return 0, "", nil, errorDependencia(ctx,
			errors.New("la respuesta de perfil_proveedor no contiene Activo"))
	}
	if !activo {
		return 0, "", nil, errorCaracterizacion(409,
			"El perfil del proveedor está inactivo", nil)
	}
	mismoPerfil := idDeRelacion(registro["PerfilId"]) == in.PerfilId
	if mismoPerfil && idDeRelacion(registro["UsuarioModificacionId"]) == terceroID {
		return *in.Id, "reutilizado", nil, nil
	}
	if !mismoPerfil {
		if err := s.validarPerfilLibre(ctx, proveedorID, in.PerfilId, *in.Id); err != nil {
			return *in.Id, "", nil, err
		}
	}
	payload := clonarPayload(registro)
	payload["ProveedorId"] = map[string]any{"Id": proveedorID}
	payload["PerfilId"] = in.PerfilId
	aplicarUsuarioModificacion(payload, terceroID)
	ok, status, err = PutProveedores(ctx,
		fmt.Sprintf("%s/%d", endpointPerfilProveedor, *in.Id), payload, &map[string]any{})
	if err != nil {
		return *in.Id, "", nil, errorEscrituraCaracterizacion(ctx, status, err, "actualizar perfil_proveedor")
	}
	if !ok {
		return *in.Id, "", nil, errorEscrituraCaracterizacion(ctx, status,
			fmt.Errorf("el CRUD respondió con estado %d", status), "actualizar perfil_proveedor")
	}
	return *in.Id, "actualizado", nil, nil
}

func (s CaracterizacionNaturalService) validarPerfilLibre(
	ctx context.Context,
	proveedorID int64,
	perfilID int64,
	exceptoID int64,
) error {
	filtros := url.Values{
		"query": {fmt.Sprintf("ProveedorId:%d,PerfilId:%d,Activo:true", proveedorID, perfilID)},
		"limit": {"10"},
	}
	var registros []map[string]any
	ok, status, err := GetAllProveedoresMapas(ctx, endpointPerfilProveedor, filtros, &registros)
	if err != nil {
		return errorDependencia(ctx, err)
	}
	if !ok {
		return errorDependencia(ctx, fmt.Errorf("consulta de perfil_proveedor con estado %d", status))
	}
	for _, registro := range registros {
		if idDeMapa(desenvolverData(registro)) != exceptoID {
			return errorCaracterizacion(409,
				"El PerfilId ya está asociado al proveedor en otro registro", nil)
		}
	}
	return nil
}

func (s CaracterizacionNaturalService) escribirActividadPorID(
	ctx context.Context,
	terceroID int64,
	proveedorID int64,
	in models.ActividadEconomicaProveedorInput,
) (int64, string, *responses.RelationError, error) {
	var registro map[string]any
	ok, status, err := GetOneProveedores(ctx,
		fmt.Sprintf("%s/%d", endpointActividadEconomica, *in.Id), &registro)
	if err != nil {
		return 0, "", nil, errorLecturaCaracterizacion(ctx, status,
			"No fue posible consultar actividad_economica_proveedor", err)
	}
	if !ok {
		return 0, "", nil, errorLecturaCaracterizacion(ctx, status,
			"No fue posible consultar actividad_economica_proveedor", nil)
	}
	registro = desenvolverData(registro)
	if idDeMapa(registro) != *in.Id {
		return 0, "", nil, errorDependencia(ctx,
			errors.New("la respuesta de actividad_economica_proveedor no contiene el Id esperado"))
	}
	if idDeRelacion(registro["ProveedorId"]) != proveedorID {
		return 0, "", nil, errorCaracterizacion(409,
			"La actividad económica no pertenece al proveedor del tercero", nil)
	}
	activo, existe := registro["Activo"].(bool)
	if !existe {
		return 0, "", nil, errorDependencia(ctx,
			errors.New("la respuesta de actividad_economica_proveedor no contiene Activo"))
	}
	if !activo {
		return 0, "", nil, errorCaracterizacion(409,
			"La actividad económica del proveedor está inactiva", nil)
	}
	mismaActividad := idDeRelacion(registro["ActividadEconomicaId"]) == in.ActividadEconomicaId
	if mismaActividad && idDeRelacion(registro["UsuarioModificacionId"]) == terceroID {
		return *in.Id, "reutilizado", nil, nil
	}
	if !mismaActividad {
		if err := s.validarActividadLibre(ctx, proveedorID, in.ActividadEconomicaId, *in.Id); err != nil {
			return *in.Id, "", nil, err
		}
	}
	payload := clonarPayload(registro)
	payload["ProveedorId"] = map[string]any{"Id": proveedorID}
	payload["ActividadEconomicaId"] = in.ActividadEconomicaId
	aplicarUsuarioModificacion(payload, terceroID)
	ok, status, err = PutProveedores(ctx,
		fmt.Sprintf("%s/%d", endpointActividadEconomica, *in.Id), payload, &map[string]any{})
	if err != nil {
		return *in.Id, "", nil, errorEscrituraCaracterizacion(ctx, status, err, "actualizar actividad_economica_proveedor")
	}
	if !ok {
		return *in.Id, "", nil, errorEscrituraCaracterizacion(ctx, status,
			fmt.Errorf("el CRUD respondió con estado %d", status), "actualizar actividad_economica_proveedor")
	}
	return *in.Id, "actualizado", nil, nil
}

func (s CaracterizacionNaturalService) validarActividadLibre(
	ctx context.Context,
	proveedorID int64,
	actividadID int64,
	exceptoID int64,
) error {
	filtros := url.Values{
		"query": {fmt.Sprintf("ProveedorId:%d,ActividadEconomicaId:%d,Activo:true", proveedorID, actividadID)},
		"limit": {"10"},
	}
	var registros []map[string]any
	ok, status, err := GetAllProveedoresMapas(ctx, endpointActividadEconomica, filtros, &registros)
	if err != nil {
		return errorDependencia(ctx, err)
	}
	if !ok {
		return errorDependencia(ctx, fmt.Errorf("consulta de actividad_economica_proveedor con estado %d", status))
	}
	for _, registro := range registros {
		if idDeMapa(desenvolverData(registro)) != exceptoID {
			return errorCaracterizacion(409,
				"El ActividadEconomicaId ya está asociado al proveedor en otro registro", nil)
		}
	}
	return nil
}

func separarActivosProveedor(
	registros []map[string]any,
) ([]map[string]any, []map[string]any) {
	activos := make([]map[string]any, 0, len(registros))
	inactivos := make([]map[string]any, 0, len(registros))
	for _, registro := range registros {
		registro = desenvolverData(registro)
		if idDeMapa(registro) <= 0 {
			continue
		}
		if activo, ok := registro["Activo"].(bool); ok && activo {
			activos = append(activos, registro)
		} else {
			inactivos = append(inactivos, registro)
		}
	}
	return activos, inactivos
}

func desenvolverData(registro map[string]any) map[string]any {
	if data, ok := registro["Data"].(map[string]any); ok {
		return data
	}
	return registro
}

func errorCaracterizacion(status int, mensaje string, causa error) *CaracterizacionNaturalError {
	return &CaracterizacionNaturalError{StatusCode: status, Message: mensaje, Cause: causa}
}

func errorDependencia(ctx context.Context, causa error) *CaracterizacionNaturalError {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return errorCaracterizacion(504,
			"La consulta a un servicio externo excedió el tiempo de espera", ctx.Err())
	}
	return errorCaracterizacion(502,
		"No fue posible completar la consulta a un servicio externo", causa)
}

func errorLecturaCaracterizacion(
	ctx context.Context,
	status int,
	mensaje string,
	causa error,
) *CaracterizacionNaturalError {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return errorCaracterizacion(504,
			"La consulta a un servicio externo excedió el tiempo de espera", ctx.Err())
	}
	if status == 404 {
		return errorCaracterizacion(404, mensaje, causa)
	}
	if causa == nil {
		causa = fmt.Errorf("el servicio externo respondió con estado %d", status)
	}
	return errorCaracterizacion(502, mensaje, causa)
}

func errorEscrituraCaracterizacion(
	ctx context.Context,
	status int,
	causa error,
	operacion string,
) *CaracterizacionNaturalError {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return errorCaracterizacion(504,
			"La operación para "+operacion+" excedió el tiempo de espera", ctx.Err())
	}
	if status == 400 || status == 409 {
		return errorCaracterizacion(status,
			"El servicio externo rechazó la operación para "+operacion, causa)
	}
	return errorCaracterizacion(502,
		"No fue posible completar la operación para "+operacion, causa)
}

func avisoDeErrorCaracterizacion(
	err error,
	path string,
	service string,
) responses.RelationError {
	kind := "dependency_failure"
	message := "No fue posible materializar la sección; se conserva lo ya materializado"
	var caracterizacionErr *CaracterizacionNaturalError
	if errors.As(err, &caracterizacionErr) {
		message = caracterizacionErr.Message
		switch caracterizacionErr.StatusCode {
		case 409:
			kind = "conflict"
		case 504:
			kind = "timeout"
		case 400:
			kind = "validation"
		}
	}
	return responses.RelationError{
		Path:    path,
		Service: service,
		Message: message,
		Kind:    kind,
	}
}
