package clients

import (
	"errors"
	"net/url"
	"strings"
	"sync"
)

// ErrEscrituraBloqueada se devuelve cuando una petición no GET apunta a un CRUD
// de solo lectura. El mensaje no incluye la URL del servicio.
var ErrEscrituraBloqueada = errors.New("escritura bloqueada: el CRUD de destino es de solo lectura para el middleware")

// destinoProtegido identifica un CRUD por esquema, host y ruta base.
type destinoProtegido struct {
	esquema string
	host    string
	ruta    string
}

var (
	protegidosMu sync.RWMutex
	protegidos   []destinoProtegido
)

// ProtegerSoloLectura registra los CRUD cuyas rutas solo admiten lectura (GET).
// Reemplaza el registro anterior. Sin argumentos desactiva la protección. Una base
// vacía o sin esquema y host es un error: no se acepta un registro que fallaría abierto.
func ProtegerSoloLectura(bases ...string) error {
	nuevos := make([]destinoProtegido, 0, len(bases))
	for _, base := range bases {
		limpia := strings.TrimSpace(base)
		if limpia == "" {
			return errors.New("ProtegerSoloLectura: una URL base está vacía")
		}
		u, err := url.Parse(limpia)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return errors.New("ProtegerSoloLectura: una URL base no es válida")
		}
		nuevos = append(nuevos, destinoProtegido{
			esquema: strings.ToLower(u.Scheme),
			host:    strings.ToLower(u.Host),
			ruta:    strings.TrimRight(u.Path, "/"),
		})
	}
	protegidosMu.Lock()
	protegidos = nuevos
	protegidosMu.Unlock()
	return nil
}

// esProtegida informa si la URL pertenece a un CRUD protegido. Compara esquema y
// host completos, y la ruta por segmentos: el puerto 8121 no protege el 81210.
func esProtegida(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	esquema := strings.ToLower(u.Scheme)
	host := strings.ToLower(u.Host)
	ruta := u.Path

	protegidosMu.RLock()
	defer protegidosMu.RUnlock()
	for _, p := range protegidos {
		if p.esquema != esquema || p.host != host {
			continue
		}
		if p.ruta == "" || ruta == p.ruta || strings.HasPrefix(ruta, p.ruta+"/") {
			return true
		}
	}
	return false
}

// escrituraPermitida informa si una petición con ese método puede enviarse a la URL.
// Solo GET se permite a un CRUD protegido.
func escrituraPermitida(method, raw string) bool {
	return method == "GET" || !esProtegida(raw)
}
