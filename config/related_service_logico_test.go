package config

import "testing"

// Prueba que la llave lógica de servicio se deriva de un nombre PascalCase a
// snake_case, de modo que coincida con las llaves del registro related_service de
// los resolvers (p. ej. Tercero -> tercero, TipoContribuyente -> tipo_contribuyente).
func TestLlaveLogicaServicio_DerivaSnakeCaseDesdePascalCase(t *testing.T) {
	casos := []struct {
		nombre   string
		esperado string
	}{
		{"Tercero", "tercero"},
		{"TipoContribuyente", "tipo_contribuyente"},
		{"DatosIdentificacion", "datos_identificacion"},
		{"TipoDocumento", "tipo_documento"},
		{"ProveedorNatural", "proveedor_natural"},
		{"CuentaBancaria", "cuenta_bancaria"},
		{"tercero", "tercero"},
		{"", ""},
	}
	for _, c := range casos {
		if got := aLlaveLogicaServicio(c.nombre); got != c.esperado {
			t.Errorf("aLlaveLogicaServicio(%q) = %q, se esperaba %q", c.nombre, got, c.esperado)
		}
	}
}

// Prueba que una secuencia de mayúsculas consecutivas se trata como una sola sigla
// y el separador solo se inserta en la frontera sigla↔palabra, no entre cada letra.
// Así HTTPServer deriva http_server (no h_t_t_p_server), de modo que la llave lógica
// de servicios con siglas coincida con el registro related_service de los resolvers.
func TestLlaveLogicaServicio_TrataSiglasComoUnaSolaPalabra(t *testing.T) {
	casos := []struct {
		nombre   string
		esperado string
	}{
		{"HTTPServer", "http_server"},
		{"HTTP", "http"},
		{"ServerHTTP", "server_http"},
		{"APIKey", "api_key"},
		{"UserID", "user_id"},
		{"TerceroHTTPCliente", "tercero_http_cliente"},
		{"XMLHTTPRequest", "xmlhttp_request"},
	}
	for _, c := range casos {
		if got := aLlaveLogicaServicio(c.nombre); got != c.esperado {
			t.Errorf("aLlaveLogicaServicio(%q) = %q, se esperaba %q", c.nombre, got, c.esperado)
		}
	}
}

// Prueba que al componer un objeto sin related_service explícito, el nodo resultante
// lleva la llave lógica derivada del nombre (no el nombre tal cual), tanto en la
// raíz como en las relaciones anidadas, para que el registro related_service de los
// resolvers encuentre su resolver.
func TestCargarComposicion_RelatedServiceDerivadoDesdeNombre(t *testing.T) {
	comp, err := escribirComposicion(t, map[string]string{
		"objetos_tercero.yaml":              objetoTercero,
		"objetos_datos_identificacion.yaml": objetoDatosIdentificacion,
		"objetos_tipo_contribuyente.yaml":   objetoTipoContribuyente,
		"relaciones_proveedor.yaml":         servicioProveedorComposicion,
	})
	if err != nil {
		t.Fatalf("composición válida rechazada: %v", err)
	}

	rel := comp.Servicios["proveedor"].Relations[0]
	if rel.RelatedService != "tercero" {
		t.Errorf("RelatedService raíz = %q, se esperaba la llave lógica %q", rel.RelatedService, "tercero")
	}

	porNombre := make(map[string]string, len(rel.NestedRelations))
	for _, n := range rel.NestedRelations {
		porNombre[n.Name] = n.RelatedService
	}
	if porNombre["DatosIdentificacion"] != "datos_identificacion" {
		t.Errorf("RelatedService de DatosIdentificacion = %q, se esperaba %q",
			porNombre["DatosIdentificacion"], "datos_identificacion")
	}
	if porNombre["TipoContribuyente"] != "tipo_contribuyente" {
		t.Errorf("RelatedService de TipoContribuyente = %q, se esperaba %q",
			porNombre["TipoContribuyente"], "tipo_contribuyente")
	}
}

// Prueba que un related_service explícito en la definición del objeto gana sobre la
// derivación del nombre, para los casos en que la llave lógica no es el snake_case
// del nombre (p. ej. TipoDocumento cuyo servicio lógico es "parametro").
func TestCargarComposicion_RelatedServiceExplicitoGana(t *testing.T) {
	tipoDocumento := "name: TipoDocumento\nrelated_service: parametro\nrelated_api: parametros\nrelated_table: tipo_documento\nendpoint: /v1/tipo_documento\ncampos:\n  - name: Nombre\n    type: string\n    readable: true\n"
	servicio := "service: proveedor\nmax_depth: 4\non_relation_error: notify\nmax_related_ids: 100\nreferencias:\n  - objeto: TipoDocumento\n    direction: local_to_related\n    local_field: TipoDocumentoId\n    related_field: Id\n    cardinality: one\n    depth: 1\n    mapped_name: TipoDocumento\n"

	comp, err := escribirComposicion(t, map[string]string{
		"objetos_tipo_documento.yaml": tipoDocumento,
		"relaciones_proveedor.yaml":   servicio,
	})
	if err != nil {
		t.Fatalf("composición válida rechazada: %v", err)
	}

	rel := comp.Servicios["proveedor"].Relations[0]
	if rel.RelatedService != "parametro" {
		t.Errorf("RelatedService = %q, se esperaba el explícito %q (no la derivación del nombre)",
			rel.RelatedService, "parametro")
	}
}

// Prueba que la definición del objeto parsea el campo related_service explícito.
func TestDecodificarObjetoDireccionable_ParseaRelatedServiceExplicito(t *testing.T) {
	contenido := []byte("name: TipoDocumento\nrelated_service: parametro\nrelated_api: parametros\nrelated_table: tipo_documento\nendpoint: /v1/tipo_documento\ncampos:\n  - name: Nombre\n    type: string\n    readable: true\n")
	obj, err := DecodificarObjetoDireccionable(contenido)
	if err != nil {
		t.Fatalf("objeto válido rechazado: %v", err)
	}
	if obj.RelatedService != "parametro" {
		t.Errorf("RelatedService = %q, se esperaba %q", obj.RelatedService, "parametro")
	}
}
