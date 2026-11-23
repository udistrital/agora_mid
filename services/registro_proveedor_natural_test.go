package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	beego "github.com/beego/beego/v2/server/web"

	"github.com/udistrital/agora_mid/models"
)

// crudDoble monta dos httptest (terceros y proveedores) y los fija en config.
func crudDoble(t *testing.T, terceros, proveedores http.HandlerFunc) {
	t.Helper()
	st := httptest.NewServer(terceros)
	sp := httptest.NewServer(proveedores)
	t.Cleanup(st.Close)
	t.Cleanup(sp.Close)
	if err := beego.AppConfig.Set(claveHostTerceros, st.URL); err != nil {
		t.Fatal(err)
	}
	if err := beego.AppConfig.Set(claveHostProveedores, sp.URL); err != nil {
		t.Fatal(err)
	}
}

func entradaBase() models.RegistroProveedorNatural {
	return models.RegistroProveedorNatural{
		TipoProveedor: 1,
		DatosBasicos: models.DatosBasicosNatural{
			PrimerNombre: "Carlos", PrimerApellido: "González",
			FechaNacimiento: time.Date(1990, 5, 15, 0, 0, 0, 0, time.UTC),
		},
		DatosIdentificacion: models.DatosIdentificacionNatural{
			Numero: "1023456789", TipoDocumentoId: 1, CiudadExpedicion: 123,
			FechaExpedicion: time.Date(2008, 6, 20, 0, 0, 0, 0, time.UTC),
		},
	}
}

// Caso 2: varias identificaciones activas -> warning + un solo Put por el Id
// del primer registro, usando ese Id.
func TestCaso2VariasIdentificaciones(t *testing.T) {
	leerLogs := capturarLogs(t)
	var putsIdent int
	var pathPutIdent string
	crudDoble(t,
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch {
			case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/datos_identificacion"):
				w.WriteHeader(200)
				// dos coincidencias activas; se usa la primera (Id 88).
				_, _ = w.Write([]byte(`[{"Id":88,"TerceroId":{"Id":55}},{"Id":89,"TerceroId":{"Id":55}}]`))
			case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/datos_identificacion/"):
				putsIdent++
				pathPutIdent = r.URL.Path
				w.WriteHeader(200)
				_, _ = w.Write([]byte(`{"Id":88}`))
			case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/tercero/"):
				w.WriteHeader(200)
				_, _ = w.Write([]byte(`{"Id":55,"PrimerNombre":"Carlos"}`))
			case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/tercero/"):
				w.WriteHeader(200)
				_, _ = w.Write([]byte(`{"Id":55}`))
			default:
				w.WriteHeader(404)
				_, _ = w.Write([]byte(`{}`))
			}
		},
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`[{"Id":42}]`)) // proveedor existente
		},
	)

	_, creado, err := RegistrarProveedorNatural(context.Background(), entradaBase())
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if creado {
		t.Errorf("creado = true, se esperaba false (todo existía)")
	}
	if putsIdent != 1 {
		t.Errorf("Put identificación = %d, se esperaba exactamente 1", putsIdent)
	}
	if !strings.HasSuffix(pathPutIdent, "/datos_identificacion/88") {
		t.Errorf("Put sobre %q, se esperaba el Id 88 (primer registro)", pathPutIdent)
	}
	// Warning con cantidad e Id seleccionado, documento enmascarado.
	var warnings []entradaLog
	for _, e := range leerLogs() {
		if strings.Contains(e.linea, "identificacion-multiple") {
			warnings = append(warnings, e)
		}
	}
	if len(warnings) != 1 {
		t.Fatalf("se esperaba 1 warning de identificacion-multiple, got %d", len(warnings))
	}
	w := warnings[0].linea
	if !strings.Contains(w, "coincidencias=2") || !strings.Contains(w, "id_seleccionado=88") {
		t.Errorf("warning incompleto: %q", w)
	}
	if strings.Contains(w, "1023456789") {
		t.Errorf("el warning no debe contener el documento sin enmascarar: %q", w)
	}
	if !strings.Contains(w, "****6789") {
		t.Errorf("el documento debería ir enmascarado: %q", w)
	}
}

// Guarda: una consulta exitosa pero con TerceroId.Id ausente es error, no
// autorización para operar con Id 0.
func TestGuardaTerceroIdCero(t *testing.T) {
	crudDoble(t,
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/datos_identificacion") {
				w.WriteHeader(200)
				_, _ = w.Write([]byte(`[{"Id":88}]`)) // objeto no vacío pero sin TerceroId
				return
			}
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{}`))
		},
		func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200); _, _ = w.Write([]byte(`[]`)) },
	)

	_, _, err := RegistrarProveedorNatural(context.Background(), entradaBase())
	if err == nil {
		t.Fatal("se esperaba error por TerceroId.Id ausente, no operar con Id 0")
	}
	if !EsErrorRegistro(err) {
		t.Errorf("se esperaba errRegistro, got %T", err)
	}
}

// El error público no revela el host del CRUD.
func TestErrorPublicoSinHost(t *testing.T) {
	var host string
	crudDoble(t,
		func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500); _, _ = w.Write([]byte(`{}`)) },
		func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200); _, _ = w.Write([]byte(`[]`)) },
	)
	host, _ = beego.AppConfig.String(claveHostTerceros)

	_, _, err := RegistrarProveedorNatural(context.Background(), entradaBase())
	if err == nil {
		t.Fatal("se esperaba error")
	}
	if strings.Contains(err.Error(), host) || strings.Contains(err.Error(), "127.0.0.1") {
		t.Errorf("el error público no debe contener el host %q: %q", host, err.Error())
	}
}
