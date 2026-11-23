package services

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/beego/beego/v2/core/logs"
)

// entradaLog representa una línea de log capturada, ya parseada en sus campos
// clave=valor. Permite a las pruebas asertar servicio, etapa y host sin
// depender de una variable global mutable en el código de producción.
type entradaLog struct {
	linea                                    string
	servicio, operacion, metodo, etapa, host string
	status                                   string
}

// capturador acumula las líneas que emite registrarLogs a través del logger de
// Beego. Es un adaptador de logs registrado solo durante la prueba.
type capturador struct {
	mu      sync.Mutex
	lineas  []string
	formato logs.LogFormatter
}

func (c *capturador) Init(string) error              { return nil }
func (c *capturador) Destroy()                        {}
func (c *capturador) Flush()                          {}
func (c *capturador) SetFormatter(f logs.LogFormatter) { c.formato = f }

func (c *capturador) WriteMsg(lm *logs.LogMsg) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lineas = append(c.lineas, fmt.Sprintf(lm.Msg, lm.Args...))
	return nil
}

// instancia compartida: el adaptador se registra una vez por nombre y cada
// prueba lo limpia al empezar.
var capturadorCompartido = &capturador{}

func init() {
	logs.Register("captura_test", func() logs.Logger { return capturadorCompartido })
}

// capturarLogs redirige el logger de Beego al capturador durante la prueba y
// devuelve una función que entrega las entradas parseadas. Restaura el logger
// por defecto al terminar. No usa ninguna variable global mutable del paquete
// de producción.
func capturarLogs(t *testing.T) func() []entradaLog {
	t.Helper()
	capturadorCompartido.mu.Lock()
	capturadorCompartido.lineas = nil
	capturadorCompartido.mu.Unlock()

	if err := logs.SetLogger("captura_test"); err != nil {
		t.Fatalf("no se pudo registrar el capturador de logs: %v", err)
	}
	// Evitar que la consola duplique la salida durante la prueba.
	_ = logs.GetBeeLogger().DelLogger(logs.AdapterConsole)

	t.Cleanup(func() {
		_ = logs.SetLogger(logs.AdapterConsole)
		_ = logs.GetBeeLogger().DelLogger("captura_test")
	})

	return func() []entradaLog {
		logs.GetBeeLogger().Flush()
		capturadorCompartido.mu.Lock()
		defer capturadorCompartido.mu.Unlock()
		var entradas []entradaLog
		for _, l := range capturadorCompartido.lineas {
			entradas = append(entradas, parsearEntrada(l))
		}
		return entradas
	}
}

// parsearEntrada extrae los pares clave=valor que escribe registrarLogs.
func parsearEntrada(linea string) entradaLog {
	e := entradaLog{linea: linea}
	for _, campo := range strings.Fields(linea) {
		kv := strings.SplitN(campo, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "servicio":
			e.servicio = kv[1]
		case "operacion":
			e.operacion = kv[1]
		case "metodo":
			e.metodo = kv[1]
		case "etapa":
			e.etapa = kv[1]
		case "host":
			e.host = kv[1]
		case "status":
			e.status = kv[1]
		}
	}
	return e
}
