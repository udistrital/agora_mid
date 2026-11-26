package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/udistrital/agora_mid/clients"
	"github.com/udistrital/agora_mid/config"
	"github.com/udistrital/agora_mid/helper"
	"github.com/udistrital/agora_mid/models"
)

type capturaPeticion struct {
	metodo string
	path   string
	query  string
	cuerpo map[string]any
}

type crudTercerosFake struct {
	terceros             map[int]map[string]any
	identificaciones     map[int]map[string]any
	siguienteTercero     int
	siguienteIdent       int
	capturas             []capturaPeticion
	vacioComoObjeto      bool
	fallarIdentificacion bool
}

func nuevoCrudFake() *crudTercerosFake {
	return &crudTercerosFake{
		terceros:         map[int]map[string]any{},
		identificaciones: map[int]map[string]any{},
		siguienteTercero: 101,
		siguienteIdent:   140,
	}
}

func (f *crudTercerosFake) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var cuerpo map[string]any
	if r.Body != nil {
		data, _ := io.ReadAll(r.Body)
		if len(data) > 0 {
			_ = json.Unmarshal(data, &cuerpo)
		}
	}
	f.capturas = append(f.capturas, capturaPeticion{
		metodo: r.Method,
		path:   r.URL.Path,
		query:  r.URL.Query().Get("query"),
		cuerpo: cuerpo,
	})

	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v1/tipo_documento/"):
		id, _ := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/v1/tipo_documento/"))
		if id == 9999 {
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"Message":"no encontrado"}`)
			return
		}
		codigo := "DOC"
		if id == 7 {
			codigo = "NIT"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"Id": id, "Activo": true, "CodigoAbreviacion": codigo,
		})

	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v1/lugar/"):
		id, _ := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/v1/lugar/"))
		if id == 9999 {
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"Message":"no encontrado"}`)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"Id": id, "Activo": true, "Nombre": "Lugar",
		})

	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v1/tercero/"):
		id, _ := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/v1/tercero/"))
		registro, ok := f.terceros[id]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"Message":"no encontrado"}`)
			return
		}
		_ = json.NewEncoder(w).Encode(registro)

	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v1/datos_identificacion/"):
		id, _ := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/v1/datos_identificacion/"))
		registro, ok := f.identificaciones[id]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"Message":"no encontrado"}`)
			return
		}
		_ = json.NewEncoder(w).Encode(registro)

	case r.Method == http.MethodGet && r.URL.Path == "/v1/datos_identificacion":
		_ = json.NewEncoder(w).Encode(f.buscarIdentificaciones(r.URL.Query().Get("query")))

	case r.Method == http.MethodPost && (r.URL.Path == "/v1/tercero" || r.URL.Path == "/v1/tercero/"):
		id := f.siguienteTercero
		f.siguienteTercero++
		cuerpo["Id"] = id
		f.terceros[id] = cuerpo
		_ = json.NewEncoder(w).Encode(cuerpo)

	case r.Method == http.MethodPost && (r.URL.Path == "/v1/datos_identificacion" || r.URL.Path == "/v1/datos_identificacion/"):
		if f.fallarIdentificacion {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = io.WriteString(w, `{"Message":"fallo simulado"}`)
			return
		}
		id := f.siguienteIdent
		f.siguienteIdent++
		cuerpo["Id"] = id
		f.identificaciones[id] = cuerpo
		_ = json.NewEncoder(w).Encode(cuerpo)

	case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/v1/tercero/"):
		id, _ := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/v1/tercero/"))
		cuerpo["Id"] = id
		f.terceros[id] = cuerpo
		_ = json.NewEncoder(w).Encode(cuerpo)

	case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/v1/datos_identificacion/"):
		id, _ := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/v1/datos_identificacion/"))
		cuerpo["Id"] = id
		f.identificaciones[id] = cuerpo
		_ = json.NewEncoder(w).Encode(cuerpo)

	default:
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"Message":"ruta no simulada"}`)
	}
}

func (f *crudTercerosFake) buscarIdentificaciones(consulta string) []map[string]any {
	salida := []map[string]any{}
	for _, registro := range f.identificaciones {
		if !coincideTercero(registro, consulta) {
			continue
		}
		if !coincideNumero(registro, consulta) {
			continue
		}
		if !coincideTipoDocumento(registro, consulta) {
			continue
		}
		salida = append(salida, registro)
	}
	if len(salida) == 0 && f.vacioComoObjeto {
		return []map[string]any{{}}
	}
	return salida
}

func coincideTercero(registro map[string]any, consulta string) bool {
	if !strings.Contains(consulta, "TerceroId:") {
		return true
	}
	id := idDeRelacion(registro["TerceroId"])
	return strings.Contains(consulta, fmt.Sprintf("TerceroId:%d", id))
}

func coincideNumero(registro map[string]any, consulta string) bool {
	if !strings.Contains(consulta, "Numero:") {
		return true
	}
	numero, _ := registro["Numero"].(string)
	return strings.Contains(consulta, "Numero:"+numero)
}

func coincideTipoDocumento(registro map[string]any, consulta string) bool {
	if !strings.Contains(consulta, "TipoDocumentoId:") {
		return true
	}
	id := idDeRelacion(registro["TipoDocumentoId"])
	return strings.Contains(consulta, fmt.Sprintf("TipoDocumentoId:%d", id))
}

func servicioTerceroPrueba(t *testing.T, fake *crudTercerosFake) TerceroWriteService {
	t.Helper()
	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)

	urls := config.ServiceURLs{
		"terceros":              server.URL,
		"parametros":            server.URL,
		"gestion_personas_crud": server.URL,
		"ubicaciones":           server.URL,
	}
	relaciones, err := config.LoadRelaciones("../config", urls)
	if err != nil {
		t.Fatalf("cargando relaciones: %v", err)
	}
	cfg, ok := relaciones["tercero"]
	if !ok {
		t.Fatal("no se cargó la configuración del servicio tercero")
	}
	catalogos := NewCatalogosService(
		clients.HTTPClient{Client: server.Client()},
		server.URL,
		server.URL,
	)
	return NewTerceroWriteService(
		clients.HTTPClient{Client: server.Client()},
		urls,
		cfg,
		1,
		2,
		7,
		catalogos,
	)
}

func entradaPrueba() models.RegistroTerceroNatural {
	return models.RegistroTerceroNatural{
		DatosBasicos: models.DatosBasicosTercero{
			PrimerNombre:        "Ana",
			SegundoNombre:       "",
			PrimerApellido:      "Pérez",
			SegundoApellido:     "",
			TipoContribuyenteId: 1,
		},
		DatosIdentificacion: models.DatosIdentificacionTercero{
			Numero:          "1012345678",
			TipoDocumentoId: 3,
		},
	}
}

func TestEscribirTerceroCreaTerceroEIdentificacion(t *testing.T) {
	fake := nuevoCrudFake()
	servicio := servicioTerceroPrueba(t, fake)

	resultado, err := servicio.EscribirTercero(
		context.Background(), entradaPrueba(), 0, helper.WriteVerbCreate,
	)
	if err != nil {
		t.Fatalf("EscribirTercero no debe fallar: %v", err)
	}
	if resultado.TerceroId != 101 || resultado.DatosIdentificacionId != 140 {
		t.Fatalf("ids inesperados: tercero=%d ident=%d", resultado.TerceroId, resultado.DatosIdentificacionId)
	}
	if len(resultado.Creados) != 2 {
		t.Fatalf("se esperaban dos creaciones, obtuvo: %+v", resultado.Creados)
	}

	postTercero := buscarCaptura(fake.capturas, http.MethodPost, "/v1/tercero/")
	if postTercero == nil {
		t.Fatalf("no se envió POST de tercero; capturas=%+v", fake.capturas)
	}
	if postTercero.cuerpo["Activo"] != true {
		t.Errorf("el tercero debe crearse activo: %+v", postTercero.cuerpo)
	}
	if idDeRelacion(postTercero.cuerpo["TipoContribuyenteId"]) != 1 {
		t.Errorf("TipoContribuyenteId debe viajar como objeto Id=1: %+v", postTercero.cuerpo["TipoContribuyenteId"])
	}
	if postTercero.cuerpo["NombreCompleto"] != "Ana Pérez" {
		t.Errorf("NombreCompleto inesperado: %v", postTercero.cuerpo["NombreCompleto"])
	}

	postIdent := buscarCaptura(fake.capturas, http.MethodPost, "/v1/datos_identificacion/")
	if postIdent == nil {
		t.Fatalf("no se envió POST de identificación; capturas=%+v", fake.capturas)
	}
	if postIdent.cuerpo["Activo"] != true {
		t.Errorf("la identificación debe crearse activa: %+v", postIdent.cuerpo)
	}
	if idDeRelacion(postIdent.cuerpo["TipoDocumentoId"]) != 3 {
		t.Errorf("TipoDocumentoId debe viajar como objeto Id=3: %+v", postIdent.cuerpo["TipoDocumentoId"])
	}
	if idDeRelacion(postIdent.cuerpo["TerceroId"]) != 101 {
		t.Errorf("TerceroId debe viajar como objeto Id=101: %+v", postIdent.cuerpo["TerceroId"])
	}
}

func TestEscribirTerceroActualizaConservandoCampos(t *testing.T) {
	fake := nuevoCrudFake()
	fake.terceros[101] = map[string]any{
		"Id":                  101,
		"NombreCompleto":      "Ana Pérez",
		"PrimerNombre":        "Ana",
		"SegundoNombre":       "",
		"PrimerApellido":      "Pérez",
		"SegundoApellido":     "",
		"LugarOrigen":         149,
		"Activo":              true,
		"TipoContribuyenteId": map[string]any{"Id": 1},
	}
	fake.identificaciones[140] = map[string]any{
		"Id":                 140,
		"TerceroId":          map[string]any{"Id": 101},
		"TipoDocumentoId":    map[string]any{"Id": 3},
		"Numero":             "1012345678",
		"Activo":             true,
		"DigitoVerificacion": 0,
	}

	entrada := entradaPrueba()
	fecha := time.Date(1990, 5, 10, 0, 0, 0, 0, time.UTC)
	entrada.DatosBasicos.FechaNacimiento = &fecha
	digito := float64(1)
	entrada.DatosIdentificacion.DigitoVerificacion = &digito

	resultado, err := servicioTerceroPrueba(t, fake).EscribirTercero(
		context.Background(), entrada, 101, helper.WriteVerbUpdate,
	)
	if err != nil {
		t.Fatalf("EscribirTercero no debe fallar: %v", err)
	}
	if resultado.DatosIdentificacionId != 140 {
		t.Fatalf("se esperaba reutilizar la identificación 140; obtuvo %d", resultado.DatosIdentificacionId)
	}

	putTercero := buscarCaptura(fake.capturas, http.MethodPut, "/v1/tercero/101")
	if putTercero == nil {
		t.Fatalf("no se envió PUT de tercero; capturas=%+v", fake.capturas)
	}
	if putTercero.cuerpo["NombreCompleto"] != "Ana Pérez" {
		t.Errorf("el PUT debe conservar NombreCompleto: %+v", putTercero.cuerpo)
	}
	if putTercero.cuerpo["Activo"] != true {
		t.Errorf("el PUT debe conservar Activo: %+v", putTercero.cuerpo)
	}
	if idDeRelacion(putTercero.cuerpo["TipoContribuyenteId"]) != 1 {
		t.Errorf("el PUT debe conservar TipoContribuyenteId: %+v", putTercero.cuerpo)
	}

	putIdent := buscarCaptura(fake.capturas, http.MethodPut, "/v1/datos_identificacion/140")
	if putIdent == nil {
		t.Fatalf("no se envió PUT de identificación; capturas=%+v", fake.capturas)
	}
	if putIdent.cuerpo["Activo"] != true {
		t.Errorf("el PUT debe conservar Activo de la identificación: %+v", putIdent.cuerpo)
	}
	if idDeRelacion(putIdent.cuerpo["TerceroId"]) != 101 {
		t.Errorf("el PUT debe conservar TerceroId: %+v", putIdent.cuerpo)
	}
	if putIdent.cuerpo["Numero"] != "1012345678" {
		t.Errorf("Numero inesperado: %v", putIdent.cuerpo["Numero"])
	}
}

func TestEscribirTerceroRechazaCambioDeIdentificacion(t *testing.T) {
	fake := nuevoCrudFake()
	fake.terceros[101] = map[string]any{
		"Id":                  101,
		"NombreCompleto":      "Ana Pérez",
		"Activo":              true,
		"TipoContribuyenteId": map[string]any{"Id": 1},
	}
	fake.identificaciones[140] = map[string]any{
		"Id":              140,
		"TerceroId":       map[string]any{"Id": 101},
		"TipoDocumentoId": map[string]any{"Id": 3},
		"Numero":          "1012345678",
		"Activo":          true,
	}

	entrada := entradaPrueba()
	entrada.DatosIdentificacion.Numero = "1099999999"

	_, err := servicioTerceroPrueba(t, fake).EscribirTercero(
		context.Background(), entrada, 101, helper.WriteVerbUpdate,
	)
	if !errors.Is(err, ErrIdentificacionInmutable) {
		t.Fatalf("se esperaba ErrIdentificacionInmutable; obtuvo %v", err)
	}
	for _, captura := range fake.capturas {
		if captura.metodo == http.MethodPut || captura.metodo == http.MethodPost {
			t.Fatalf("no debe escribir si cambia la identidad; capturas=%+v", fake.capturas)
		}
	}
}

func TestEscribirTerceroRechazaDocumentoDeOtroTercero(t *testing.T) {
	fake := nuevoCrudFake()
	fake.terceros[101] = map[string]any{"Id": 101, "Activo": true}
	fake.identificaciones[140] = map[string]any{
		"Id":              140,
		"TerceroId":       map[string]any{"Id": 999},
		"TipoDocumentoId": map[string]any{"Id": 3},
		"Numero":          "1012345678",
		"Activo":          true,
	}

	_, err := servicioTerceroPrueba(t, fake).EscribirTercero(
		context.Background(), entradaPrueba(), 101, helper.WriteVerbUpdate,
	)
	if !errors.Is(err, ErrIdentificacionDeOtroTercero) {
		t.Fatalf("se esperaba ErrIdentificacionDeOtroTercero; obtuvo %v", err)
	}
}

func TestEscribirTerceroRechazaTipoContribuyenteInvalido(t *testing.T) {
	fake := nuevoCrudFake()
	entrada := entradaPrueba()
	entrada.DatosBasicos.TipoContribuyenteId = 2

	_, err := servicioTerceroPrueba(t, fake).EscribirTercero(
		context.Background(), entrada, 0, helper.WriteVerbCreate,
	)
	if !errors.Is(err, ErrTipoContribuyenteInvalido) {
		t.Fatalf("se esperaba ErrTipoContribuyenteInvalido; obtuvo %v", err)
	}
	if len(fake.capturas) != 0 {
		t.Fatalf("no debe llamar al CRUD con un tipo inválido; capturas=%+v", fake.capturas)
	}
}

// El CRUD de terceros puede responder [{}] cuando no hay coincidencias.
// Esa respuesta debe tratarse como ausencia, no como una identificación inválida.
func TestEscribirTerceroCreaConRespuestaVaciaObjeto(t *testing.T) {
	fake := nuevoCrudFake()
	fake.vacioComoObjeto = true

	resultado, err := servicioTerceroPrueba(t, fake).EscribirTercero(
		context.Background(), entradaPrueba(), 0, helper.WriteVerbCreate,
	)
	if err != nil {
		t.Fatalf("EscribirTercero no debe fallar con [{}]: %v", err)
	}
	if resultado.TerceroId != 101 || resultado.DatosIdentificacionId != 140 {
		t.Fatalf("ids inesperados: tercero=%d ident=%d", resultado.TerceroId, resultado.DatosIdentificacionId)
	}
}

// Un PUT con un tercero inexistente debe llegar al 404, no fallar antes por [{}].
func TestEscribirTerceroPutInexistenteConRespuestaVaciaObjeto(t *testing.T) {
	fake := nuevoCrudFake()
	fake.vacioComoObjeto = true

	_, err := servicioTerceroPrueba(t, fake).EscribirTercero(
		context.Background(), entradaPrueba(), 999999, helper.WriteVerbUpdate,
	)
	if !errors.Is(err, ErrTerceroNoEncontrado) {
		t.Fatalf("se esperaba ErrTerceroNoEncontrado; obtuvo %v", err)
	}
	for _, captura := range fake.capturas {
		if captura.metodo == "POST" || captura.metodo == "PUT" {
			t.Fatalf("no debe escribir con un tercero inexistente; capturas=%+v", fake.capturas)
		}
	}
}

func buscarCaptura(capturas []capturaPeticion, metodo, path string) *capturaPeticion {
	for i := range capturas {
		if capturas[i].metodo == metodo && capturas[i].path == path {
			return &capturas[i]
		}
	}
	return nil
}

func TestEscribirTerceroRechazaNumeroConComa(t *testing.T) {
	fake := nuevoCrudFake()
	entrada := entradaPrueba()
	entrada.DatosIdentificacion.Numero = "1012,345"

	_, err := servicioTerceroPrueba(t, fake).EscribirTercero(
		context.Background(), entrada, 0, helper.WriteVerbCreate,
	)
	var validacion *ValidationError
	if !errors.As(err, &validacion) {
		t.Fatalf("se esperaba ValidationError; obtuvo %v", err)
	}
	sinEscrituras(t, fake)
}

func TestEscribirTerceroRechazaNombreLargo(t *testing.T) {
	fake := nuevoCrudFake()
	entrada := entradaPrueba()
	entrada.DatosBasicos.PrimerNombre = strings.Repeat("A", 101)

	_, err := servicioTerceroPrueba(t, fake).EscribirTercero(
		context.Background(), entrada, 0, helper.WriteVerbCreate,
	)
	var validacion *ValidationError
	if !errors.As(err, &validacion) {
		t.Fatalf("se esperaba ValidationError; obtuvo %v", err)
	}
	sinEscrituras(t, fake)
}

func TestEscribirTerceroRechazaFechaNacimientoFutura(t *testing.T) {
	fake := nuevoCrudFake()
	entrada := entradaPrueba()
	futuro := time.Now().Add(24 * time.Hour)
	entrada.DatosBasicos.FechaNacimiento = &futuro

	_, err := servicioTerceroPrueba(t, fake).EscribirTercero(
		context.Background(), entrada, 0, helper.WriteVerbCreate,
	)
	var validacion *ValidationError
	if !errors.As(err, &validacion) {
		t.Fatalf("se esperaba ValidationError; obtuvo %v", err)
	}
	sinEscrituras(t, fake)
}

func TestEscribirTerceroRechazaTipoDocumentoInexistente(t *testing.T) {
	fake := nuevoCrudFake()
	entrada := entradaPrueba()
	entrada.DatosIdentificacion.TipoDocumentoId = 9999

	_, err := servicioTerceroPrueba(t, fake).EscribirTercero(
		context.Background(), entrada, 0, helper.WriteVerbCreate,
	)
	var validacion *ValidationError
	if !errors.As(err, &validacion) {
		t.Fatalf("se esperaba ValidationError; obtuvo %v", err)
	}
	sinEscrituras(t, fake)
}
