// Package services — service de gestión de proveedores. Consume el CRUD de
// gestion_proveedores_crud leyendo su URL base desde la configuración
// `GestionProveedoresCrudHost` y reutilizando las utilidades HTTP compartidas
// en http_utils.go. Mantiene el mismo contrato que el service de terceros.
package services

import (
	"context"
	"net/http"
	"net/url"

	beego "github.com/beego/beego/v2/server/web"
	"github.com/udistrital/utils_oas/v2/request"
)

// claveHostProveedores es la clave de configuración (conf/app.conf) con la URL
// base del CRUD de gestión de proveedores, de la forma http://host:puerto/v1.
const claveHostProveedores = "GestionProveedoresCrudHost"

// servicioProveedores identifica este service en los logs.
const servicioProveedores = "gestion_proveedores"

// Mensajes públicos neutros del service de proveedores. No revelan host,
// puerto ni URL interna.
const (
	msgConexionProveedores      = "No fue posible conectar con el servicio de gestión de proveedores"
	msgProcesarProveedores      = "No fue posible procesar la respuesta del servicio de gestión de proveedores"
	msgConfiguracionProveedores = "El servicio de gestión de proveedores no está configurado correctamente"
)

// hostProveedores lee la URL base del CRUD de proveedores desde la configuración.
func hostProveedores() string {
	v, _ := beego.AppConfig.String(claveHostProveedores)
	return v
}

// GetAllProveedores lista recursos del CRUD de proveedores en el `endpoint`
// indicado y decodifica la respuesta en `target`. Los parámetros de consulta
// son opcionales; con nil o vacíos conserva la consulta original del endpoint.
// El llamador consume `target` solo cuando ok es true.
func GetAllProveedores(ctx context.Context, endpoint string, queryParams url.Values, target any) (ok bool, statusCode int, err error) {
	u, perr := construirURL(hostProveedores(), endpoint, queryParams)
	if perr != nil {
		registrarLogs(servicioProveedores, "GetAll", http.MethodGet, "construir-url", "", 0, perr)
		return false, http.StatusInternalServerError, &errPublico{msgConfiguracionProveedores}
	}
	status, rerr := request.GetWithContext(ctx, u, target)
	return traducir(ctx, servicioProveedores, "GetAll", http.MethodGet, u, status, rerr, msgConexionProveedores, msgProcesarProveedores)
}

// GetOneProveedores obtiene un único recurso del CRUD de proveedores en el
// `endpoint` del recurso individual (por ejemplo "proveedor/15") y lo
// decodifica en `target`.
func GetOneProveedores(ctx context.Context, endpoint string, target any) (ok bool, statusCode int, err error) {
	u, perr := construirURL(hostProveedores(), endpoint, nil)
	if perr != nil {
		registrarLogs(servicioProveedores, "GetOne", http.MethodGet, "construir-url", "", 0, perr)
		return false, http.StatusInternalServerError, &errPublico{msgConfiguracionProveedores}
	}
	status, rerr := request.GetWithContext(ctx, u, target)
	return traducir(ctx, servicioProveedores, "GetOne", http.MethodGet, u, status, rerr, msgConexionProveedores, msgProcesarProveedores)
}

// PostProveedores crea un recurso en el CRUD de proveedores enviando `payload`
// como JSON y decodifica la respuesta en `target`.
func PostProveedores(ctx context.Context, endpoint string, payload any, target any) (ok bool, statusCode int, err error) {
	u, perr := construirURL(hostProveedores(), endpoint, nil)
	if perr != nil {
		registrarLogs(servicioProveedores, "Post", http.MethodPost, "construir-url", "", 0, perr)
		return false, http.StatusInternalServerError, &errPublico{msgConfiguracionProveedores}
	}
	status, rerr := request.PostWithContext(ctx, u, payload, target)
	return traducir(ctx, servicioProveedores, "Post", http.MethodPost, u, status, rerr, msgConexionProveedores, msgProcesarProveedores)
}

// PutProveedores actualiza un recurso en el CRUD de proveedores enviando
// `payload` como JSON y decodifica la respuesta en `target`.
func PutProveedores(ctx context.Context, endpoint string, payload any, target any) (ok bool, statusCode int, err error) {
	u, perr := construirURL(hostProveedores(), endpoint, nil)
	if perr != nil {
		registrarLogs(servicioProveedores, "Put", http.MethodPut, "construir-url", "", 0, perr)
		return false, http.StatusInternalServerError, &errPublico{msgConfiguracionProveedores}
	}
	status, rerr := request.PutWithContext(ctx, u, payload, target)
	return traducir(ctx, servicioProveedores, "Put", http.MethodPut, u, status, rerr, msgConexionProveedores, msgProcesarProveedores)
}
