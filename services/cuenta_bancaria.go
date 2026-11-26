package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/udistrital/agora_mid/models"
)

type CuentaBancariaService struct{}

type CuentaBancariaError struct {
	StatusCode int
	Message    string
	Cause      error
	Partial    bool
	Path       string
}

func (e *CuentaBancariaError) Error() string {
	return e.Message
}

func (e *CuentaBancariaError) Unwrap() error {
	return e.Cause
}

func (s CuentaBancariaService) Registrar(ctx context.Context, terceroID int64, entrada models.RegistroCuentaBancaria) (models.CuentaBancariaData, error) {
	data := models.CuentaBancariaData{
		TerceroId:      terceroID,
		CuentaBancaria: []models.CuentaBancariaRegistrada{},
		Resultado: models.Resultado{
			Creados:      []models.EntidadId{},
			Actualizados: []models.EntidadId{},
			Reutilizados: []models.EntidadId{},
		},
	}
	fallar := func(err error) (models.CuentaBancariaData, error) {
		var cuentaErr *CuentaBancariaError
		if errors.As(err, &cuentaErr) {
			copia := *cuentaErr
			copia.Partial = copia.Partial || len(data.Resultado.Creados)+len(data.Resultado.Actualizados) > 0
			return data, &copia
		}
		return data, &CuentaBancariaError{
			StatusCode: 500,
			Message:    "Error interno al registrar la cuenta bancaria",
			Cause:      err,
			Partial:    len(data.Resultado.Creados)+len(data.Resultado.Actualizados) > 0,
		}
	}

	entrada.ContactoPago.ValorEspecifico = strings.TrimSpace(entrada.ContactoPago.ValorEspecifico)
	entrada.ContactoPago.NombreContacto = strings.TrimSpace(entrada.ContactoPago.NombreContacto)
	for i := range entrada.CuentaBancaria {
		entrada.CuentaBancaria[i].NumeroCuenta = strings.TrimSpace(entrada.CuentaBancaria[i].NumeroCuenta)
	}
	contactoExplicito, err := validarEntradaCuentaBancaria(terceroID, entrada)
	if err != nil {
		return fallar(err)
	}

	var terceroData map[string]any
	ok, status, err := GetOne(ctx, fmt.Sprintf("tercero/%d", terceroID), &terceroData)
	if err != nil {
		return fallar(errorLecturaCuenta(ctx, status, "No fue posible consultar el tercero", err))
	}
	if !ok {
		return fallar(errorLecturaCuenta(ctx, status, "No fue posible consultar el tercero", nil))
	}
	terceroData = datosCuentaBancaria(terceroData)
	if idDeMapa(terceroData) != terceroID {
		return fallar(errorDependenciaCuenta(ctx, errors.New("la respuesta del tercero no contiene el identificador esperado")))
	}
	activo, existeActivo := terceroData["Activo"].(bool)
	if !existeActivo {
		return fallar(errorDependenciaCuenta(ctx, errors.New("la respuesta del tercero no contiene Activo")))
	}
	if !activo {
		return fallar(errorCuentaBancaria(409, "El tercero está inactivo", nil))
	}

	proveedor, err := resolverProveedorCuenta(ctx, terceroID)
	if err != nil {
		return fallar(err)
	}
	proveedorID := idDeMapa(proveedor)
	data.ProveedorId = proveedorID

	contactoID, accionContacto, err := materializarContactoPago(ctx, proveedorID, entrada.ContactoPago, contactoExplicito)
	if contactoID > 0 {
		data.ContactoPagoId = contactoID
	}
	if err != nil {
		return fallar(err)
	}
	if contactoID > 0 {
		registrarResultado(&data.Resultado, "Contacto", contactoID, accionContacto)
	}

	for _, cuenta := range entrada.CuentaBancaria {
		registrada, accion, err := materializarCuentaBancaria(ctx, proveedorID, contactoID, cuenta)
		if err != nil {
			var cuentaErr *CuentaBancariaError
			if errors.As(err, &cuentaErr) {
				cuentaErr.Path = "CuentaBancaria"
			}
			if registrada.Id > 0 {
				data.CuentaBancaria = append(data.CuentaBancaria, registrada)
			}
			return fallar(err)
		}
		data.CuentaBancaria = append(data.CuentaBancaria, registrada)
		registrarResultado(&data.Resultado, "CuentaBancaria", registrada.Id, accion)
	}

	return data, nil
}

func validarEntradaCuentaBancaria(terceroID int64, entrada models.RegistroCuentaBancaria) (int64, error) {
	if terceroID <= 0 {
		return 0, errorCuentaBancaria(400, "terceroId debe ser un entero positivo", nil)
	}
	if len(entrada.CuentaBancaria) == 0 {
		return 0, errorCuentaBancaria(400, "Debe incluir al menos una cuenta bancaria", nil)
	}
	if entrada.ContactoPago.TipoCanalContactoId <= 0 {
		return 0, errorCuentaBancaria(400, "TipoCanalContactoId debe ser un entero positivo", nil)
	}
	if entrada.ContactoPago.ValorEspecifico == "" || entrada.ContactoPago.NombreContacto == "" {
		return 0, errorCuentaBancaria(400, "ValorEspecifico y NombreContacto son obligatorios", nil)
	}
	if strings.ContainsAny(entrada.ContactoPago.ValorEspecifico, ",:") {
		return 0, errorCuentaBancaria(400, "ValorEspecifico no puede contener coma ni dos puntos por las restricciones del filtro del CRUD", nil)
	}

	contactoExplicito := int64(0)
	clavesCuenta := make(map[string]struct{}, len(entrada.CuentaBancaria))
	for _, cuenta := range entrada.CuentaBancaria {
		if cuenta.EntidadBancariaId <= 0 || cuenta.TipoCuentaId <= 0 || cuenta.NumeroCuenta == "" {
			return 0, errorCuentaBancaria(400, "Cada cuenta requiere EntidadBancariaId, TipoCuentaId y NumeroCuenta válidos", nil)
		}
		if strings.ContainsAny(cuenta.NumeroCuenta, ",:") {
			return 0, errorCuentaBancaria(400, "NumeroCuenta no puede contener coma ni dos puntos por las restricciones del filtro del CRUD", nil)
		}
		clave := fmt.Sprintf("%d|%d|%s", cuenta.EntidadBancariaId, cuenta.TipoCuentaId, cuenta.NumeroCuenta)
		if _, existe := clavesCuenta[clave]; existe {
			return 0, errorCuentaBancaria(400, "La solicitud contiene cuentas bancarias duplicadas", nil)
		}
		clavesCuenta[clave] = struct{}{}
		if cuenta.ContactoId == nil {
			continue
		}
		if *cuenta.ContactoId <= 0 {
			return 0, errorCuentaBancaria(400, "ContactoId debe ser un entero positivo o null", nil)
		}
		if contactoExplicito != 0 && contactoExplicito != *cuenta.ContactoId {
			return 0, errorCuentaBancaria(400, "Todas las cuentas deben referenciar el mismo ContactoId", nil)
		}
		contactoExplicito = *cuenta.ContactoId
	}
	return contactoExplicito, nil
}

func resolverProveedorCuenta(ctx context.Context, terceroID int64) (map[string]any, error) {
	filtros := url.Values{"query": {fmt.Sprintf("TerceroId:%d", terceroID)}, "limit": {"2"}}
	var proveedores []map[string]any
	ok, status, err := GetAllProveedores(ctx, "proveedor", filtros, &proveedores)
	if err != nil {
		return nil, errorDependenciaCuenta(ctx, err)
	}
	if !ok {
		return nil, errorDependenciaCuenta(ctx, fmt.Errorf("consulta de proveedor con estado %d", status))
	}
	coincidencias := make([]map[string]any, 0, 2)
	for _, proveedor := range proveedores {
		proveedor = datosCuentaBancaria(proveedor)
		if idDeMapa(proveedor) <= 0 {
			continue
		}
		if _, existeTercero := proveedor["TerceroId"]; !existeTercero {
			return nil, errorDependenciaCuenta(ctx, errors.New("la respuesta del proveedor no contiene TerceroId"))
		}
		proveedorTerceroID := idDeRelacion(proveedor["TerceroId"])
		if proveedorTerceroID <= 0 {
			return nil, errorDependenciaCuenta(ctx, errors.New("la respuesta del proveedor contiene un TerceroId inválido"))
		}
		if proveedorTerceroID == terceroID {
			coincidencias = append(coincidencias, proveedor)
		}
	}
	if len(coincidencias) == 0 {
		return nil, errorCuentaBancaria(404, "No existe un proveedor asociado al tercero", nil)
	}
	if len(coincidencias) > 1 {
		return nil, errorCuentaBancaria(409, "El tercero tiene más de un proveedor asociado; no es posible resolverlo de forma segura", nil)
	}
	activo, existeActivo := coincidencias[0]["Activo"].(bool)
	if !existeActivo {
		return nil, errorDependenciaCuenta(ctx, errors.New("la respuesta del proveedor no contiene Activo"))
	}
	if !activo {
		return nil, errorCuentaBancaria(409, "El proveedor asociado al tercero está inactivo", nil)
	}
	return coincidencias[0], nil
}

func materializarContactoPago(ctx context.Context, proveedorID int64, entrada models.ContactoPagoInput, contactoExplicito int64) (int64, string, error) {
	filtro := url.Values{
		"query": {"ProveedorId:" + strconv.FormatInt(proveedorID, 10) + ",TipoCanalContactoId:" + strconv.FormatInt(entrada.TipoCanalContactoId, 10) + ",ValorEspecifico:" + entrada.ValorEspecifico},
		"limit": {"2"},
	}
	var coincidencias []map[string]any
	ok, status, err := GetAllProveedores(ctx, "contacto", filtro, &coincidencias)
	if err != nil {
		return 0, "", errorDependenciaCuenta(ctx, err)
	}
	if !ok {
		return 0, "", errorDependenciaCuenta(ctx, fmt.Errorf("consulta de contacto con estado %d", status))
	}
	coincidencias, err = filtrarContactosCuenta(ctx, coincidencias, proveedorID, entrada)
	if err != nil {
		return 0, "", err
	}
	if len(coincidencias) > 1 {
		return 0, "", errorCuentaBancaria(409, "Hay varios contactos de pago con la misma identidad; no es posible resolverlos de forma segura", nil)
	}
	if contactoExplicito > 0 {
		for _, coincidencia := range coincidencias {
			if idDeMapa(coincidencia) != contactoExplicito {
				return 0, "", errorCuentaBancaria(409, "ContactoId contradice la identidad del contacto de pago", nil)
			}
		}
		return actualizarContactoPago(ctx, proveedorID, contactoExplicito, entrada)
	}
	if len(coincidencias) == 0 {
		payload := payloadContactoPago(proveedorID, entrada)
		payload["Activo"] = true
		var respuesta map[string]any
		ok, status, err := PostProveedores(ctx, "contacto", payload, &respuesta)
		if err != nil {
			return 0, "", errorEscrituraCuenta(ctx, status, err, "crear el contacto de pago")
		}
		if !ok {
			return 0, "", errorEscrituraCuenta(ctx, status, fmt.Errorf("el CRUD respondió con estado %d", status), "crear el contacto de pago")
		}
		id := idRespuestaCuenta(respuesta)
		if id <= 0 {
			return 0, "", errorCuentaBancaria(502,
				"La creación del contacto de pago no devolvió un Id utilizable",
				errors.New("la respuesta del CRUD no contiene Id"))
		}
		return id, "creado", nil
	}
	return actualizarContactoPago(ctx, proveedorID, idDeMapa(coincidencias[0]), entrada)
}

func filtrarContactosCuenta(ctx context.Context, registros []map[string]any, proveedorID int64, entrada models.ContactoPagoInput) ([]map[string]any, error) {
	coincidencias := make([]map[string]any, 0, len(registros))
	for _, registro := range registros {
		registro = datosCuentaBancaria(registro)
		if idDeMapa(registro) <= 0 {
			continue
		}
		registroProveedorID := idDeRelacion(registro["ProveedorId"])
		tipoCanalID := idDeRelacion(registro["TipoCanalContactoId"])
		valorEspecifico, valorExiste := registro["ValorEspecifico"].(string)
		if registroProveedorID <= 0 || tipoCanalID <= 0 || !valorExiste {
			return nil, errorDependenciaCuenta(ctx, errors.New("la respuesta del contacto no contiene los campos de identidad"))
		}
		if registroProveedorID != proveedorID ||
			tipoCanalID != entrada.TipoCanalContactoId ||
			valorEspecifico != entrada.ValorEspecifico {
			continue
		}
		coincidencias = append(coincidencias, registro)
	}
	return coincidencias, nil
}

func actualizarContactoPago(ctx context.Context, proveedorID, contactoID int64, entrada models.ContactoPagoInput) (int64, string, error) {
	var registro map[string]any
	ok, status, err := GetOneProveedores(ctx, fmt.Sprintf("contacto/%d", contactoID), &registro)
	if err != nil {
		return 0, "", errorLecturaCuenta(ctx, status, "No fue posible consultar el contacto de pago", err)
	}
	if !ok {
		return 0, "", errorLecturaCuenta(ctx, status, "No fue posible consultar el contacto de pago", nil)
	}
	registro = datosCuentaBancaria(registro)
	if idDeMapa(registro) != contactoID {
		return 0, "", errorDependenciaCuenta(ctx, errors.New("la respuesta del contacto no contiene el identificador esperado"))
	}
	if idDeRelacion(registro["ProveedorId"]) != proveedorID {
		return 0, "", errorCuentaBancaria(409, "El contacto de pago no pertenece al proveedor asociado al tercero", nil)
	}
	activo, existeActivo := registro["Activo"].(bool)
	if !existeActivo {
		return 0, "", errorDependenciaCuenta(ctx, errors.New("la respuesta del contacto no contiene Activo"))
	}
	if !activo {
		return 0, "", errorCuentaBancaria(409, "El contacto de pago está inactivo", nil)
	}
	if contactoPagoIgual(registro, proveedorID, entrada) {
		return contactoID, "reutilizado", nil
	}
	payload := clonarPayload(registro)
	for clave, valor := range payloadContactoPago(proveedorID, entrada) {
		payload[clave] = valor
	}
	var respuesta map[string]any
	ok, status, err = PutProveedores(ctx, fmt.Sprintf("contacto/%d", contactoID), payload, &respuesta)
	if err != nil {
		return contactoID, "", errorActualizacionCuenta(ctx, status, err, "el contacto de pago", "ContactoPago")
	}
	if !ok {
		return contactoID, "", errorActualizacionCuenta(ctx, status,
			fmt.Errorf("el CRUD respondió con estado %d", status), "el contacto de pago", "ContactoPago")
	}
	if id := idRespuestaCuenta(respuesta); id > 0 && id != contactoID {
		err := errorCuentaBancaria(502,
			"La respuesta de actualización del contacto de pago contiene un Id inesperado; no es posible confirmar el resultado",
			errors.New("el Id devuelto por el CRUD no coincide con el contacto solicitado"))
		err.Partial = true
		err.Path = "ContactoPago"
		return contactoID, "", err
	}
	return contactoID, "actualizado", nil
}

func payloadContactoPago(proveedorID int64, entrada models.ContactoPagoInput) map[string]any {
	return map[string]any{
		"ProveedorId":         map[string]any{"Id": proveedorID},
		"TipoCanalContactoId": entrada.TipoCanalContactoId,
		"ValorEspecifico":     entrada.ValorEspecifico,
		"NombreContacto":      entrada.NombreContacto,
		"Extension":           entrada.Extension,
	}
}

func contactoPagoIgual(registro map[string]any, proveedorID int64, entrada models.ContactoPagoInput) bool {
	if idDeRelacion(registro["ProveedorId"]) != proveedorID ||
		idDeRelacion(registro["TipoCanalContactoId"]) != entrada.TipoCanalContactoId ||
		cadenaCuenta(registro["ValorEspecifico"]) != entrada.ValorEspecifico ||
		cadenaCuenta(registro["NombreContacto"]) != entrada.NombreContacto {
		return false
	}
	if entrada.Extension == nil {
		return registro["Extension"] == nil
	}
	return cadenaCuenta(registro["Extension"]) == *entrada.Extension
}

func materializarCuentaBancaria(ctx context.Context, proveedorID, contactoID int64, entrada models.CuentaBancariaInput) (models.CuentaBancariaRegistrada, string, error) {
	var resultado models.CuentaBancariaRegistrada
	filtro := url.Values{
		"query": {"ProveedorId:" + strconv.FormatInt(proveedorID, 10) + ",EntidadBancariaId:" + strconv.FormatInt(entrada.EntidadBancariaId, 10) + ",TipoCuentaId:" + strconv.FormatInt(entrada.TipoCuentaId, 10) + ",NumeroCuenta:" + entrada.NumeroCuenta},
		"limit": {"2"},
	}
	var coincidencias []map[string]any
	ok, status, err := GetAllProveedores(ctx, "cuenta_bancaria", filtro, &coincidencias)
	if err != nil {
		return resultado, "", errorDependenciaCuenta(ctx, err)
	}
	if !ok {
		return resultado, "", errorDependenciaCuenta(ctx, fmt.Errorf("consulta de cuenta bancaria con estado %d", status))
	}
	coincidencias, err = filtrarCuentasBancarias(ctx, coincidencias, proveedorID, entrada)
	if err != nil {
		return resultado, "", err
	}
	if len(coincidencias) > 1 {
		return resultado, "", errorCuentaBancaria(409, "Hay varias cuentas bancarias con la misma identidad; no es posible resolverlas de forma segura", nil)
	}
	if len(coincidencias) == 0 {
		payload := payloadCuentaBancaria(proveedorID, contactoID, entrada)
		payload["Activo"] = true
		var respuesta map[string]any
		ok, status, err := PostProveedores(ctx, "cuenta_bancaria", payload, &respuesta)
		if err != nil {
			return resultado, "", errorEscrituraCuenta(ctx, status, err, "crear la cuenta bancaria")
		}
		if !ok {
			return resultado, "", errorEscrituraCuenta(ctx, status, fmt.Errorf("el CRUD respondió con estado %d", status), "crear la cuenta bancaria")
		}
		id := idRespuestaCuenta(respuesta)
		if id <= 0 {
			return resultado, "", errorCuentaBancaria(502,
				"La creación de la cuenta bancaria no devolvió un Id utilizable",
				errors.New("la respuesta del CRUD no contiene Id"))
		}
		return cuentaBancariaRegistrada(id, contactoID, entrada), "creado", nil
	}

	id := idDeMapa(coincidencias[0])
	var registro map[string]any
	ok, status, err = GetOneProveedores(ctx, fmt.Sprintf("cuenta_bancaria/%d", id), &registro)
	if err != nil {
		return resultado, "", errorLecturaCuenta(ctx, status, "No fue posible consultar la cuenta bancaria", err)
	}
	if !ok {
		return resultado, "", errorLecturaCuenta(ctx, status, "No fue posible consultar la cuenta bancaria", nil)
	}
	registro = datosCuentaBancaria(registro)
	if idDeMapa(registro) != id {
		return resultado, "", errorDependenciaCuenta(ctx, errors.New("la respuesta de la cuenta bancaria no contiene el identificador esperado"))
	}
	if idDeRelacion(registro["ProveedorId"]) != proveedorID {
		return resultado, "", errorCuentaBancaria(409, "La cuenta bancaria no pertenece al proveedor asociado al tercero", nil)
	}
	activo, existeActivo := registro["Activo"].(bool)
	if !existeActivo {
		return resultado, "", errorDependenciaCuenta(ctx, errors.New("la respuesta de la cuenta bancaria no contiene Activo"))
	}
	if !activo {
		return resultado, "", errorCuentaBancaria(409, "La cuenta bancaria existente está inactiva", nil)
	}
	if cuentaBancariaIgual(registro, proveedorID, contactoID, entrada) {
		return cuentaBancariaRegistrada(id, contactoID, entrada), "reutilizado", nil
	}
	payload := clonarPayload(registro)
	for clave, valor := range payloadCuentaBancaria(proveedorID, contactoID, entrada) {
		payload[clave] = valor
	}
	ok, status, err = PutProveedores(ctx, fmt.Sprintf("cuenta_bancaria/%d", id), payload, &map[string]any{})
	if err != nil {
		return cuentaBancariaRegistrada(id, contactoID, entrada), "", errorActualizacionCuenta(ctx, status, err, "la cuenta bancaria", "CuentaBancaria")
	}
	if !ok {
		return cuentaBancariaRegistrada(id, contactoID, entrada), "", errorActualizacionCuenta(ctx, status,
			fmt.Errorf("el CRUD respondió con estado %d", status), "la cuenta bancaria", "CuentaBancaria")
	}
	return cuentaBancariaRegistrada(id, contactoID, entrada), "actualizado", nil
}

func filtrarCuentasBancarias(ctx context.Context, registros []map[string]any, proveedorID int64, entrada models.CuentaBancariaInput) ([]map[string]any, error) {
	coincidencias := make([]map[string]any, 0, len(registros))
	for _, registro := range registros {
		registro = datosCuentaBancaria(registro)
		if idDeMapa(registro) <= 0 {
			continue
		}
		registroProveedorID := idDeRelacion(registro["ProveedorId"])
		entidadBancariaID := idDeRelacion(registro["EntidadBancariaId"])
		tipoCuentaID := idDeRelacion(registro["TipoCuentaId"])
		numeroCuenta, numeroExiste := registro["NumeroCuenta"].(string)
		if registroProveedorID <= 0 || entidadBancariaID <= 0 || tipoCuentaID <= 0 || !numeroExiste {
			return nil, errorDependenciaCuenta(ctx, errors.New("la respuesta de la cuenta bancaria no contiene los campos de identidad"))
		}
		if registroProveedorID != proveedorID ||
			entidadBancariaID != entrada.EntidadBancariaId ||
			tipoCuentaID != entrada.TipoCuentaId ||
			numeroCuenta != entrada.NumeroCuenta {
			continue
		}
		coincidencias = append(coincidencias, registro)
	}
	return coincidencias, nil
}

func payloadCuentaBancaria(proveedorID, contactoID int64, entrada models.CuentaBancariaInput) map[string]any {
	return map[string]any{
		"ProveedorId":            map[string]any{"Id": proveedorID},
		"ContactoId":             map[string]any{"Id": contactoID},
		"EntidadBancariaId":      entrada.EntidadBancariaId,
		"TipoCuentaId":           entrada.TipoCuentaId,
		"NumeroCuenta":           entrada.NumeroCuenta,
		"MontoCapitalAutorizado": entrada.MontoCapitalAutorizado,
	}
}

func cuentaBancariaIgual(registro map[string]any, proveedorID, contactoID int64, entrada models.CuentaBancariaInput) bool {
	return idDeRelacion(registro["ProveedorId"]) == proveedorID &&
		idDeRelacion(registro["ContactoId"]) == contactoID &&
		idDeRelacion(registro["EntidadBancariaId"]) == entrada.EntidadBancariaId &&
		idDeRelacion(registro["TipoCuentaId"]) == entrada.TipoCuentaId &&
		cadenaCuenta(registro["NumeroCuenta"]) == entrada.NumeroCuenta &&
		numeroCuentaIgual(registro["MontoCapitalAutorizado"], entrada.MontoCapitalAutorizado)
}

func cuentaBancariaRegistrada(id, contactoID int64, entrada models.CuentaBancariaInput) models.CuentaBancariaRegistrada {
	return models.CuentaBancariaRegistrada{
		Id:                     id,
		EntidadBancariaId:      entrada.EntidadBancariaId,
		TipoCuentaId:           entrada.TipoCuentaId,
		NumeroCuenta:           entrada.NumeroCuenta,
		MontoCapitalAutorizado: entrada.MontoCapitalAutorizado,
		ContactoId:             contactoID,
	}
}

func registrarResultado(resultado *models.Resultado, entidad string, id int64, accion string) {
	elemento := models.EntidadId{Entidad: entidad, Id: id}
	switch accion {
	case "creado":
		resultado.Creados = append(resultado.Creados, elemento)
	case "actualizado":
		resultado.Actualizados = append(resultado.Actualizados, elemento)
	default:
		resultado.Reutilizados = append(resultado.Reutilizados, elemento)
	}
}

func datosCuentaBancaria(registro map[string]any) map[string]any {
	if data, ok := registro["Data"].(map[string]any); ok {
		return data
	}
	return registro
}

func idRespuestaCuenta(registro map[string]any) int64 {
	registro = datosCuentaBancaria(registro)
	return idDeMapa(registro)
}

func cadenaCuenta(valor any) string {
	if texto, ok := valor.(string); ok {
		return texto
	}
	return ""
}

func numeroCuentaIgual(valor any, esperado float64) bool {
	switch numero := valor.(type) {
	case float64:
		return numero == esperado
	case float32:
		return float64(numero) == esperado
	case int:
		return float64(numero) == esperado
	case int64:
		return float64(numero) == esperado
	case json.Number:
		resultado, err := strconv.ParseFloat(string(numero), 64)
		return err == nil && resultado == esperado
	default:
		return false
	}
}

func errorCuentaBancaria(status int, mensaje string, causa error) *CuentaBancariaError {
	return &CuentaBancariaError{StatusCode: status, Message: mensaje, Cause: causa}
}

func errorLecturaCuenta(ctx context.Context, status int, mensaje string, causa error) *CuentaBancariaError {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return errorCuentaBancaria(504, "La consulta al servicio CRUD excedió el tiempo de espera", ctx.Err())
	}
	if status == 404 {
		return errorCuentaBancaria(404, mensaje, causa)
	}
	if causa == nil {
		causa = fmt.Errorf("servicio CRUD respondió con estado %d", status)
	}
	return errorCuentaBancaria(502, mensaje, causa)
}

func errorDependenciaCuenta(ctx context.Context, causa error) *CuentaBancariaError {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return errorCuentaBancaria(504, "La consulta al servicio CRUD excedió el tiempo de espera", ctx.Err())
	}
	return errorCuentaBancaria(502, "No fue posible completar la consulta al servicio CRUD", causa)
}

func errorEscrituraCuenta(ctx context.Context, status int, causa error, operacion string) *CuentaBancariaError {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return errorCuentaBancaria(504, "La operación para "+operacion+" excedió el tiempo de espera", ctx.Err())
	}
	if status == 400 || status == 409 {
		return errorCuentaBancaria(status, "El servicio CRUD rechazó la operación para "+operacion+": "+causa.Error(), causa)
	}
	return errorCuentaBancaria(502, "No fue posible completar la operación para "+operacion+": "+causa.Error(), causa)
}

func errorActualizacionCuenta(ctx context.Context, status int, causa error, entidad, path string) *CuentaBancariaError {
	err := errorEscrituraCuenta(ctx, status, causa, "actualizar "+entidad)
	err.Partial = true
	err.Path = path
	return err
}
