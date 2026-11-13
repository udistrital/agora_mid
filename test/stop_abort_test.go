package test

import (
	"net/http"
	"testing"
	"time"

	"github.com/udistrital/agora_mid/config"
)

// Fallo de dependencia en un nodo tras materializar la raíz, con política fail: la
// cascada se detiene, responde 206 con el subárbol materializado y nombra el nodo
// en MetaData.Errors; ningún nodo posterior se envía a su CRUD.
func TestStopAbortFalloDependenciaResponde206ConDataParcial(t *testing.T) {
	crud := NuevoCRUDSimulado(t, nil)
	crud.FallarPostDe("contacto", http.StatusServiceUnavailable)
	router := routerProveedorConPolitica(t, crud, config.OnRelationErrorFail)

	rec, env := ejecutar(t, router, http.MethodPost, comoJSON(t, cuerpoProveedorNaturalConTerceroID(nil, false)))
	if rec.Code != http.StatusPartialContent {
		t.Fatalf("status = %d; esperado 206 de stop-abort. body=%s", rec.Code, rec.Body.String())
	}
	if env.Data == nil || env.Data["Id"] == nil {
		t.Errorf("Data debe incluir el proveedor ya materializado; data=%+v", env.Data)
	}
	if env.MetaData == nil || len(env.MetaData.Errors) != 1 {
		t.Fatalf("se esperaba una entrada en MetaData.Errors; env=%+v", env)
	}
	aviso := env.MetaData.Errors[0]
	if p, _ := aviso["path"].(string); p != "Contacto[0]" {
		t.Errorf("path = %q; esperado Contacto[0] (nodo donde se detuvo)", p)
	}
	if s, _ := aviso["service"].(string); s != "contacto" {
		t.Errorf("service = %q; esperado contacto", s)
	}
	if k, _ := aviso["kind"].(string); k != "dependency_failure" {
		t.Errorf("kind = %q; esperado dependency_failure", k)
	}

	// Ningún nodo posterior al fallido se envía: nada no-GET después del POST fallido.
	capturas := crud.Capturas()
	idxFallido := -1
	for i, c := range capturas {
		if c.Metodo == http.MethodPost && recursoDePath(c.Path) == "contacto" {
			idxFallido = i
			break
		}
	}
	if idxFallido < 0 {
		t.Fatal("el POST a contacto debió intentarse")
	}
	for _, c := range capturas[idxFallido+1:] {
		if c.Metodo != http.MethodGet {
			t.Errorf("no debe enviarse ningún nodo posterior al fallido; captura=%+v", c)
		}
	}
}

// Tiempo de espera en un nodo tras materializar la raíz, con política fail: 206 con
// Kind timeout y sin nodos posteriores.
func TestStopAbortTiempoAgotadoResponde206ConKindTimeout(t *testing.T) {
	crud := NuevoCRUDSimulado(t, nil)
	crud.ConTimeoutCliente(200 * time.Millisecond)
	crud.RetardarMetodoDe(http.MethodPost, "contacto", 5*time.Second)
	router := routerProveedorConPolitica(t, crud, config.OnRelationErrorFail)

	rec, env := ejecutar(t, router, http.MethodPost, comoJSON(t, cuerpoProveedorNaturalConTerceroID(nil, false)))
	if rec.Code != http.StatusPartialContent {
		t.Fatalf("status = %d; esperado 206 de stop-abort. body=%s", rec.Code, rec.Body.String())
	}
	if env.MetaData == nil || len(env.MetaData.Errors) != 1 {
		t.Fatalf("se esperaba una entrada en MetaData.Errors; env=%+v", env)
	}
	if k, _ := env.MetaData.Errors[0]["kind"].(string); k != "timeout" {
		t.Errorf("kind = %q; esperado timeout", k)
	}
}
