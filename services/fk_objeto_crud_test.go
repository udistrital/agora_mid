package services

import (
	"context"
	"net/http"
	"testing"

	"github.com/udistrital/agora_mid/clients"
	"github.com/udistrital/agora_mid/config"
)

// El CRUD de contacto decodifica ProveedorId como objeto (rel fk). Un entero
// responde 400. El resolver debe enviar la FK como {"Id": n} al CRUD.
func TestWriteResolverEnviaFKDeContactoComoObjeto(t *testing.T) {
	var capturas []peticionCapturada
	server := crudSimulado(t, map[string]string{
		"POST /v1/contacto/": `{"Data":{"Id":5}}`,
	}, &capturas)
	defer server.Close()

	r := NewEntityWriteResolver(
		clients.HTTPClient{Client: server.Client()},
		urlsDe(server),
		"gestion_personas_crud",
		endpointContacto,
		nil,
	)
	salida, err := r.Materialize(context.Background(), config.RelationConfig{}, map[string]any{
		"NombreContacto": "Ana",
		"ProveedorId":    float64(1),
	}, nil)
	if err != nil {
		t.Fatalf("Materialize no debe fallar: %v", err)
	}

	var post *peticionCapturada
	for i := range capturas {
		if capturas[i].metodo == http.MethodPost {
			post = &capturas[i]
		}
	}
	if post == nil {
		t.Fatalf("se esperaba un POST; capturas=%+v", capturas)
	}
	fk, ok := post.cuerpo["ProveedorId"].(map[string]any)
	if !ok {
		t.Fatalf("ProveedorId debe enviarse como objeto {Id}; cuerpo=%+v", post.cuerpo)
	}
	if idEntero(fk["Id"]) != 1 {
		t.Errorf("ProveedorId.Id debe ser 1; recibido=%+v", fk)
	}
	if salida["ProveedorId"] != float64(1) {
		t.Errorf("la salida interna debe conservar la FK entera; salida=%+v", salida)
	}
}
