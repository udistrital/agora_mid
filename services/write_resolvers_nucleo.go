package services

import (
	"fmt"
	"strings"

	"github.com/udistrital/agora_mid/clients"
	"github.com/udistrital/agora_mid/config"
	"github.com/udistrital/agora_mid/helper"
)

// NewWriteResolversNucleo construye los write-resolvers del subárbol del
// proveedor, cada uno con su endpoint real y su estrategia de idempotencia,
// reutilizando el mismo transporte HTTP. Devuelve el resolver de la entidad raíz
// (proveedor) y el mapeo related_service→WriteResolver de las entidades hijas,
// listos para armar el registro que se inyecta al motor de escritura.
//
// Idempotencia por entidad (según el modelo):
//   - proveedor (raíz): por tercero_id (columna unique).
//   - datos_identificacion: por numero + tipo_documento_id.
//   - tercero: sin clave natural propia; se reutiliza por el Id resuelto por el
//     subárbol de identificación o por el Id del cuerpo.
//   - proveedor_natural / proveedor_juridico: por PK compartida (Id propagado).
//   - contacto / cuenta_bancaria: por el Id del cuerpo (sin clave natural).
//
// Los catálogos (tipo_contribuyente, perfil) no tienen write-resolver: el cliente
// envía su Id y el motor nunca los materializa.
func NewWriteResolversNucleo(
	client clients.HTTPClient,
	urls config.ServiceURLs,
) (helper.WriteResolver, map[string]helper.WriteResolver) {
	raiz := NewEntityWriteResolver(
		client, urls, "gestion_personas_crud", endpointProveedor,
		queryPorTerceroId,
	)

	hijos := map[string]helper.WriteResolver{
		ServicioTercero: NewEntityWriteResolver(
			client, urls, "terceros", endpointTercero, nil,
		),
		ServicioDatosIdentificacion: NewEntityWriteResolver(
			client, urls, "terceros", endpointDatosIdentificacion,
			queryPorNumeroYTipoDocumento,
		),
		ServicioProveedorNatural: NewEntityWriteResolver(
			client, urls, "gestion_personas_crud", endpointProveedorNatural, nil,
		),
		ServicioProveedorJuridico: NewEntityWriteResolver(
			client, urls, "gestion_personas_crud", endpointProveedorJuridico, nil,
		),
		ServicioContacto: NewEntityWriteResolver(
			client, urls, "gestion_personas_crud", endpointContacto, nil,
		),
		ServicioCuentaBancaria: NewEntityWriteResolver(
			client, urls, "gestion_personas_crud", endpointCuentaBancaria, nil,
		),
	}

	return raiz, hijos
}

// endpointProveedor es el endpoint real de la entidad raíz proveedor en su CRUD
// dueño.
const endpointProveedor = "/v1/proveedor"

// queryPorTerceroId construye el filtro de búsqueda del proveedor por su
// tercero_id (unique). keys[0] es el valor de TerceroId.
func queryPorTerceroId(keys []string) (string, bool) {
	if len(keys) != 1 || strings.TrimSpace(keys[0]) == "" {
		return "", false
	}
	return "TerceroId:" + keys[0] + ",Activo:true", true
}

// queryPorNumeroYTipoDocumento construye el filtro de búsqueda de los datos de
// identificación por su clave natural numero + tipo_documento_id. keys sigue el
// orden declarado en natural_key (Numero, TipoDocumentoId).
func queryPorNumeroYTipoDocumento(keys []string) (string, bool) {
	if len(keys) != 2 {
		return "", false
	}
	numero := strings.TrimSpace(keys[0])
	tipoDoc := strings.TrimSpace(keys[1])
	if numero == "" || tipoDoc == "" {
		return "", false
	}
	// Rechazar valores con los separadores del filtro para no romper la consulta.
	if strings.ContainsAny(numero, ",:") || strings.ContainsAny(tipoDoc, ",:") {
		return "", false
	}
	return fmt.Sprintf("Numero:%s,TipoDocumentoId:%s,Activo:true", numero, tipoDoc), true
}
