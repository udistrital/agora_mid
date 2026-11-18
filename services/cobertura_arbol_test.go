package services

import (
	"sort"
	"testing"

	"github.com/udistrital/agora_mid/clients"
	"github.com/udistrital/agora_mid/config"
	"github.com/udistrital/agora_mid/helper"
)

// urlsCobertura son URLs ficticias: la prueba solo necesita que el árbol real cargue
// y que cada related_service tenga su resolvedor.
func urlsCobertura() config.ServiceURLs {
	return config.ServiceURLs{
		"terceros":              "http://terceros.test",
		"parametros":            "http://parametros.test",
		"gestion_personas_crud": "http://gestion.test",
		"ubicaciones":           "http://ubicaciones.test",
	}
}

// servicesDelArbol recorre las relaciones del árbol y devuelve, ordenados, los
// related_service de cada relación; si write es true, solo las que tienen bloque write.
func servicesDelArbol(rels []config.RelationConfig, write bool) []string {
	var nombres []string
	var recorrer func([]config.RelationConfig)
	recorrer = func(rs []config.RelationConfig) {
		for _, r := range rs {
			if !write || r.Write != nil {
				nombres = append(nombres, r.RelatedService)
			}
			recorrer(r.NestedRelations)
		}
	}
	recorrer(rels)
	sort.Strings(nombres)
	return nombres
}

// Cada related_service del árbol real del proveedor tiene un resolvedor de lectura.
func TestNucleoResolversCubrenElArbolDeLectura(t *testing.T) {
	relaciones, err := config.LoadRelaciones("../config", urlsCobertura())
	if err != nil {
		t.Fatalf("la configuración real debe cargar: %v", err)
	}
	registro := NewResolversNucleo(helper.NewHTTPResolver(nil, urlsCobertura()))
	for _, servicio := range servicesDelArbol(relaciones["proveedor"].Relations, false) {
		if _, ok := registro[servicio]; !ok {
			t.Errorf("falta el resolvedor de lectura para el related_service %q", servicio)
		}
	}
}

// Cada relación con bloque write del árbol real tiene un write-resolver de hijo.
func TestNucleoWriteResolversCubrenElArbolDeEscritura(t *testing.T) {
	relaciones, err := config.LoadRelaciones("../config", urlsCobertura())
	if err != nil {
		t.Fatalf("la configuración real debe cargar: %v", err)
	}
	_, hijos := NewWriteResolversNucleo(clients.HTTPClient{}, urlsCobertura())
	for _, servicio := range servicesDelArbol(relaciones["proveedor"].Relations, true) {
		if _, ok := hijos[servicio]; !ok {
			t.Errorf("falta el write-resolver para el related_service %q", servicio)
		}
	}
}
