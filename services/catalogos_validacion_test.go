package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/udistrital/agora_mid/clients"
)

func servidorCatalogoPrueba(
	t *testing.T,
	estado int,
	activo bool,
) (*httptest.Server, *atomic.Int32) {
	t.Helper()

	var llamadas atomic.Int32
	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		llamadas.Add(1)
		if estado != 0 && estado != http.StatusOK {
			w.WriteHeader(estado)
			_, _ = w.Write([]byte(`{"Message":"error simulado"}`))
			return
		}

		id, _ := strconv.Atoi(strings.TrimPrefix(strings.TrimPrefix(
			r.URL.Path, "/v1/tipo_documento/"), "/v1/lugar/"))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"Id":     id,
			"Activo": activo,
		})
	}))
	t.Cleanup(servidor.Close)
	return servidor, &llamadas
}

func servicioCatalogosPrueba(
	t *testing.T,
	servidor *httptest.Server,
	ahora *time.Time,
) CatalogosService {
	t.Helper()
	servicio := NewCatalogosService(
		clients.HTTPClient{Client: servidor.Client()},
		servidor.URL,
		servidor.URL,
	)
	servicio.ahora = func() time.Time { return *ahora }
	return servicio
}

func TestCatalogosServiceCacheaPositivoYExpira(t *testing.T) {
	servidor, llamadas := servidorCatalogoPrueba(t, http.StatusOK, true)
	ahora := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	servicio := servicioCatalogosPrueba(t, servidor, &ahora)

	for i := 0; i < 2; i++ {
		existe, err := servicio.TipoDocumentoExiste(context.Background(), 7)
		if err != nil {
			t.Fatalf("TipoDocumentoExiste no debe fallar: %v", err)
		}
		if !existe {
			t.Fatal("el tipo de documento debía existir")
		}
	}
	if obtenidas := llamadas.Load(); obtenidas != 1 {
		t.Fatalf("se esperaba una sola consulta, hubo %d", obtenidas)
	}

	ahora = ahora.Add(ttlCatalogoPositivo + time.Second)
	if _, err := servicio.TipoDocumentoExiste(context.Background(), 7); err != nil {
		t.Fatalf("TipoDocumentoExiste no debe fallar: %v", err)
	}
	if obtenidas := llamadas.Load(); obtenidas != 2 {
		t.Fatalf("la caché debía vencer y consultar de nuevo, hubo %d consultas", obtenidas)
	}
}

func TestCatalogosServiceCacheaNegativoYExpira(t *testing.T) {
	servidor, llamadas := servidorCatalogoPrueba(t, http.StatusNotFound, false)
	ahora := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	servicio := servicioCatalogosPrueba(t, servidor, &ahora)

	for i := 0; i < 2; i++ {
		existe, err := servicio.TipoDocumentoExiste(context.Background(), 999)
		if err != nil {
			t.Fatalf("un 404 no debe ser error de dependencia: %v", err)
		}
		if existe {
			t.Fatal("el tipo de documento no debía existir")
		}
	}
	if obtenidas := llamadas.Load(); obtenidas != 1 {
		t.Fatalf("se esperaba una sola consulta, hubo %d", obtenidas)
	}

	ahora = ahora.Add(ttlCatalogoNegativo + time.Second)
	if _, err := servicio.TipoDocumentoExiste(context.Background(), 999); err != nil {
		t.Fatalf("TipoDocumentoExiste no debe fallar: %v", err)
	}
	if obtenidas := llamadas.Load(); obtenidas != 2 {
		t.Fatalf("el negativo debía vencer y consultar de nuevo, hubo %d consultas", obtenidas)
	}
}

func TestCatalogosServiceCacheaRegistroInactivo(t *testing.T) {
	servidor, llamadas := servidorCatalogoPrueba(t, http.StatusOK, false)
	ahora := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	servicio := servicioCatalogosPrueba(t, servidor, &ahora)

	for i := 0; i < 2; i++ {
		existe, err := servicio.TipoDocumentoExiste(context.Background(), 8)
		if err != nil {
			t.Fatalf("TipoDocumentoExiste no debe fallar: %v", err)
		}
		if existe {
			t.Fatal("un registro inactivo no debe considerarse existente")
		}
	}
	if obtenidas := llamadas.Load(); obtenidas != 1 {
		t.Fatalf("se esperaba una sola consulta, hubo %d", obtenidas)
	}

	ahora = ahora.Add(ttlCatalogoPositivo + time.Second)
	if _, err := servicio.TipoDocumentoExiste(context.Background(), 8); err != nil {
		t.Fatalf("TipoDocumentoExiste no debe fallar: %v", err)
	}
	if obtenidas := llamadas.Load(); obtenidas != 2 {
		t.Fatalf("el resultado debía vencer y consultar de nuevo, hubo %d consultas", obtenidas)
	}
}

func TestCatalogosServiceNoCacheaErroresDeDependencia(t *testing.T) {
	servidor, llamadas := servidorCatalogoPrueba(t, http.StatusInternalServerError, false)
	ahora := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	servicio := servicioCatalogosPrueba(t, servidor, &ahora)

	for i := 0; i < 2; i++ {
		if _, err := servicio.TipoDocumentoExiste(context.Background(), 7); err == nil {
			t.Fatal("se esperaba error de dependencia")
		}
	}
	if obtenidas := llamadas.Load(); obtenidas != 2 {
		t.Fatalf("un error de dependencia no debe cachearse, hubo %d consultas", obtenidas)
	}
}

func TestCatalogosServiceSeparaCachePorRecurso(t *testing.T) {
	servidor, llamadas := servidorCatalogoPrueba(t, http.StatusOK, true)
	ahora := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	servicio := servicioCatalogosPrueba(t, servidor, &ahora)

	if _, err := servicio.TipoDocumentoExiste(context.Background(), 7); err != nil {
		t.Fatalf("TipoDocumentoExiste no debe fallar: %v", err)
	}
	if _, err := servicio.LugarExiste(context.Background(), 7); err != nil {
		t.Fatalf("LugarExiste no debe fallar: %v", err)
	}
	if obtenidas := llamadas.Load(); obtenidas != 2 {
		t.Fatalf("tipo_documento y lugar deben tener claves distintas, hubo %d consultas", obtenidas)
	}

	if _, err := servicio.TipoDocumentoExiste(context.Background(), 7); err != nil {
		t.Fatalf("TipoDocumentoExiste no debe fallar: %v", err)
	}
	if _, err := servicio.LugarExiste(context.Background(), 7); err != nil {
		t.Fatalf("LugarExiste no debe fallar: %v", err)
	}
	if obtenidas := llamadas.Load(); obtenidas != 2 {
		t.Fatalf("las segundas consultas debían salir de caché, hubo %d consultas", obtenidas)
	}
}
