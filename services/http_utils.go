package services

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/beego/beego/v2/core/logs"
	"github.com/udistrital/utils_oas/v2/request"
)

// errPublico es el error neutro que reciben los llamadores. No incorpora la
// causa técnica para no exponer host, puerto ni URL interna en err.Error().
type errPublico struct{ msg string }

func (e *errPublico) Error() string { return e.msg }

// unirURL une la URL base del CRUD y el endpoint relativo produciendo una única
// barra de separación y conservando cualquier prefijo de ruta de la base, sin
// importar si la base termina en barra o el endpoint empieza con barra.
func unirURL(base, endpoint string) string {
	return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(endpoint, "/")
}

// construirURL une host y endpoint conservando el prefijo de ruta, analiza el
// resultado y fusiona la consulta previa del endpoint con queryParams,
// conservando valores repetidos y codificando una sola vez. Devuelve error si
// el host está vacío o la URL no es analizable. El texto del error es neutro;
// la identidad del service la aporta el log del llamador.
func construirURL(base, endpoint string, queryParams url.Values) (string, error) {
	if base == "" {
		return "", errors.New("URL base no configurada")
	}
	u, err := url.Parse(unirURL(base, endpoint))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", errors.New("URL base inválida")
	}
	if len(queryParams) > 0 {
		q := u.Query()
		for clave, valores := range queryParams {
			for _, v := range valores {
				q.Add(clave, v)
			}
		}
		u.RawQuery = q.Encode()
	}
	return u.String(), nil
}

// traducir mapea el resultado (status, err) de la librería al contrato de
// salida del service (ok, statusCode, err) y registra la causa en los fallos.
// `servicio` identifica al service en el log; `msgConexion`/`msgProcesar` son
// los mensajes públicos neutros propios de cada service. `u` es la URL
// completa; de ella se extraen host y puerto para el log, nunca para el mensaje.
//
//   - éxito 200-226 sin error -> (true, status, nil).
//   - 204 cuyo único error es el EOF de decodificar un cuerpo vacío -> (true, 204, nil).
//   - status 0 (transporte/cancelación/serialización) -> (false, 500, errPublico conexión).
//   - status fuera de 200-226 (rechazo HTTP) -> (false, status, errPublico conexión).
//   - error de decodificación con status de éxito -> (false, status, errPublico procesar).
func traducir(ctx context.Context, servicio, op, metodo, u string, status int, rerr error, msgConexion, msgProcesar string) (bool, int, error) {
	if rerr == nil {
		return true, status, nil
	}

	// 204 sin contenido: la librería intenta decodificar y recibe EOF.
	if status == http.StatusNoContent && esEOFDecodificacion(rerr) {
		return true, status, nil
	}

	// Sin respuesta HTTP: la librería devuelve status 0. Se distingue la causa
	// (timeout, cancelación o conexión) para el log.
	if status == 0 {
		registrarLogs(servicio, op, metodo, etapaSinRespuesta(ctx, rerr), u, status, rerr)
		return false, http.StatusInternalServerError, &errPublico{msgConexion}
	}

	// Éxito HTTP pero la respuesta no se pudo decodificar en target.
	if errors.Is(rerr, request.ErrResponseDecode) {
		registrarLogs(servicio, op, metodo, "decodificacion", u, status, rerr)
		return false, status, &errPublico{msgProcesar}
	}

	// Rechazo HTTP del CRUD (status fuera de 200-226): la librería no decodifica.
	registrarLogs(servicio, op, metodo, "rechazo-http", u, status, rerr)
	return false, status, &errPublico{msgConexion}
}

// esEOFDecodificacion reconoce el error que la librería produce al intentar
// decodificar un cuerpo vacío (envuelve io.EOF en ErrResponseDecode).
func esEOFDecodificacion(err error) bool {
	return errors.Is(err, request.ErrResponseDecode) && errors.Is(err, io.EOF)
}

// etapaSinRespuesta clasifica la causa de un fallo sin respuesta HTTP para el
// log: timeout, cancelación o conexión/transporte.
func etapaSinRespuesta(ctx context.Context, rerr error) string {
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(rerr, context.DeadlineExceeded):
		return "timeout"
	case errors.Is(ctx.Err(), context.Canceled) || errors.Is(rerr, context.Canceled):
		return "cancelacion"
	default:
		return "transporte"
	}
}

// registrarLogs escribe una sola entrada de log con la causa técnica. Incluye
// service, operación, método, etapa, host, puerto y estado remoto; excluye
// credenciales, tokens, valores de filtros y payloads. No registra la URL
// completa (que llevaría el query string); solo host y puerto.
func registrarLogs(servicio, op, metodo, etapa, u string, status int, causa error) {
	host, puerto := hostPuerto(u)
	logs.Error("servicio=%s operacion=%s metodo=%s etapa=%s host=%s puerto=%s status=%d causa=%v",
		servicio, op, metodo, etapa, host, puerto, status, causa)
}

// hostPuerto extrae host y puerto de una URL para el log, sin credenciales ni
// ruta ni query. Devuelve cadenas vacías si la URL no es analizable.
func hostPuerto(u string) (host, puerto string) {
	parsed, err := url.Parse(u)
	if err != nil {
		return "", ""
	}
	return parsed.Hostname(), parsed.Port()
}
