package services

import (
	"fmt"
	"strings"

	"github.com/udistrital/agora_mid/clients"
	"github.com/udistrital/agora_mid/config"
	"github.com/udistrital/agora_mid/helper"
)

const (
	// ServicioTercero es la llave lógica del objeto raíz Tercero.
	ServicioTercero = "tercero"
	// ServicioDatosIdentificacion es la llave lógica de la identificación.
	ServicioDatosIdentificacion = "datos_identificacion"

	endpointTercero             = "/v1/tercero"
	endpointDatosIdentificacion = "/v1/datos_identificacion"
)

// NewTerceroWriteResolvers construye los resolvers del dominio 1 reutilizando
// el mismo EntityWriteResolver y el mismo cliente HTTP del motor base.
//
// El resolver raíz es Tercero. El único hijo escribible del dominio es
// DatosIdentificacion, que se resuelve por Numero + TipoDocumentoId.
func NewTerceroWriteResolvers(
	client clients.HTTPClient,
	urls config.ServiceURLs,
) (helper.WriteResolver, map[string]helper.WriteResolver) {
	raiz := NewEntityWriteResolverConFKs(
		client, urls, "terceros", endpointTercero, nil,
		[]string{"TipoContribuyenteId"},
	)

	hijos := map[string]helper.WriteResolver{
		ServicioDatosIdentificacion: NewEntityWriteResolverConFKs(
			client, urls, "terceros", endpointDatosIdentificacion,
			queryPorNumeroYTipoDocumento,
			[]string{"TipoDocumentoId", "TerceroId"},
		),
	}

	return raiz, hijos
}

// queryPorNumeroYTipoDocumento construye el filtro de búsqueda de la
// identificación por su clave natural Numero + TipoDocumentoId.
func queryPorNumeroYTipoDocumento(keys []string) (string, bool) {
	if len(keys) != 2 {
		return "", false
	}
	numero := strings.TrimSpace(keys[0])
	tipoDoc := strings.TrimSpace(keys[1])
	if numero == "" || tipoDoc == "" {
		return "", false
	}
	if strings.ContainsAny(numero, ",:") || strings.ContainsAny(tipoDoc, ",:") {
		return "", false
	}
	return fmt.Sprintf("Numero:%s,TipoDocumentoId:%s,Activo:true", numero, tipoDoc), true
}
