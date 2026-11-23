// Package services expone la capa de servicios que consume por HTTP los CRUD
// externos de la OAS. El servicio de terceros consume el CRUD de terceros
// leyendo su URL base desde la configuración `TercerosCrudHost` y delegando la
// ejecución HTTP en github.com/udistrital/utils_oas/v2/request. Las utilidades
// HTTP compartidas con otros services viven en http_utils.go.
package services

import (
	"context"
	"net/http"
	"net/url"

	beego "github.com/beego/beego/v2/server/web"
	"github.com/udistrital/utils_oas/v2/request"
)

// claveHostTerceros es la clave de configuración (conf/app.conf) con la URL
// base del CRUD de terceros, de la forma http://host:puerto/v1.
const claveHostTerceros = "TercerosCrudHost"

// servicioTerceros identifica este service en los logs.
const servicioTerceros = "terceros"

// Mensajes públicos neutros del service de terceros. No revelan host, puerto
// ni URL interna.
const (
	msgConexionTerceros      = "No fue posible conectar con el servicio de terceros"
	msgProcesarTerceros      = "No fue posible procesar la respuesta del servicio de terceros"
	msgConfiguracionTerceros = "El servicio de terceros no está configurado correctamente"
)

// hostTerceros lee la URL base del CRUD de terceros desde la configuración.
func hostTerceros() string {
	v, _ := beego.AppConfig.String(claveHostTerceros)
	return v
}

// GetAll lista recursos del CRUD de terceros en el `endpoint` indicado y
// decodifica la respuesta en `target` (puntero provisto por el llamador). Los
// parámetros de consulta son opcionales: con nil o vacíos conserva la consulta
// original del endpoint. El llamador consume `target` solo cuando ok es true.
func GetAll(ctx context.Context, endpoint string, queryParams url.Values, target any) (ok bool, statusCode int, err error) {
	u, perr := construirURL(hostTerceros(), endpoint, queryParams)
	if perr != nil {
		registrarLogs(servicioTerceros, "GetAll", http.MethodGet, "construir-url", "", 0, perr)
		return false, http.StatusInternalServerError, &errPublico{msgConfiguracionTerceros}
	}
	status, rerr := request.GetWithContext(ctx, u, target)
	return traducir(ctx, servicioTerceros, "GetAll", http.MethodGet, u, status, rerr, msgConexionTerceros, msgProcesarTerceros)
}

// GetOne obtiene un único recurso del CRUD de terceros en el `endpoint`
// indicado y lo decodifica en `target`.
func GetOne(ctx context.Context, endpoint string, target any) (ok bool, statusCode int, err error) {
	u, perr := construirURL(hostTerceros(), endpoint, nil)
	if perr != nil {
		registrarLogs(servicioTerceros, "GetOne", http.MethodGet, "construir-url", "", 0, perr)
		return false, http.StatusInternalServerError, &errPublico{msgConfiguracionTerceros}
	}
	status, rerr := request.GetWithContext(ctx, u, target)
	return traducir(ctx, servicioTerceros, "GetOne", http.MethodGet, u, status, rerr, msgConexionTerceros, msgProcesarTerceros)
}

// Post crea un recurso en el CRUD de terceros enviando `payload` como JSON y
// decodifica la respuesta en `target`.
func Post(ctx context.Context, endpoint string, payload any, target any) (ok bool, statusCode int, err error) {
	u, perr := construirURL(hostTerceros(), endpoint, nil)
	if perr != nil {
		registrarLogs(servicioTerceros, "Post", http.MethodPost, "construir-url", "", 0, perr)
		return false, http.StatusInternalServerError, &errPublico{msgConfiguracionTerceros}
	}
	status, rerr := request.PostWithContext(ctx, u, payload, target)
	return traducir(ctx, servicioTerceros, "Post", http.MethodPost, u, status, rerr, msgConexionTerceros, msgProcesarTerceros)
}

// Put actualiza un recurso en el CRUD de terceros enviando `payload` como JSON
// y decodifica la respuesta en `target`.
func Put(ctx context.Context, endpoint string, payload any, target any) (ok bool, statusCode int, err error) {
	u, perr := construirURL(hostTerceros(), endpoint, nil)
	if perr != nil {
		registrarLogs(servicioTerceros, "Put", http.MethodPut, "construir-url", "", 0, perr)
		return false, http.StatusInternalServerError, &errPublico{msgConfiguracionTerceros}
	}
	status, rerr := request.PutWithContext(ctx, u, payload, target)
	return traducir(ctx, servicioTerceros, "Put", http.MethodPut, u, status, rerr, msgConexionTerceros, msgProcesarTerceros)
}
