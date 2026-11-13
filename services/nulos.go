package services

import (
	"sort"
	"strconv"
	"strings"

	"github.com/udistrital/agora_mid/helper"
)

// validarNulos recorre el cuerpo de escritura y rechaza con *helper.ErrorEscritura
// todo null en un campo escalar que la tabla no declara nulable. El null de una FK
// (sufijo Id), de Id y de una relación anidada se deja al motor, que lo trata como
// ausente o como llave. Se ejecuta antes de cualquier lectura o escritura, para que
// un 400 no deje escrituras parciales.
//
// Las tablas se infieren del nombre de la clave en PascalCase (por ejemplo
// EstadoProveedor -> estado_proveedor). Es una aproximación: TODO: derivar la tabla de
// la configuración de relaciones en lugar de la convención de nombres.
func validarNulos(entidad map[string]any, tabla, ruta string) error {
	claves := make([]string, 0, len(entidad))
	for clave := range entidad {
		claves = append(claves, clave)
	}
	sort.Strings(claves)

	for _, clave := range claves {
		valor := entidad[clave]
		rutaCampo := clave
		if ruta != "" {
			rutaCampo = ruta + "." + clave
		}
		switch v := valor.(type) {
		case map[string]any:
			if err := validarNulos(v, snakeDeClave(clave), rutaCampo); err != nil {
				return err
			}
		case []any:
			for i, elem := range v {
				if m, ok := elem.(map[string]any); ok {
					if err := validarNulos(m, snakeDeClave(clave), fmtRutaElem(rutaCampo, i)); err != nil {
						return err
					}
				}
			}
		case nil:
			if esRelacionAnidada(clave) || clave == "Id" || strings.HasSuffix(clave, "Id") {
				continue
			}
			if !camposNulables[tabla][clave] {
				return &helper.ErrorEscritura{Path: rutaCampo, Motivo: "no admite null"}
			}
		}
	}
	return nil
}

// esRelacionAnidada informa si la clave nombra una tabla del modelo, es decir una
// relación anidada, y no un campo escalar.
func esRelacionAnidada(clave string) bool {
	_, ok := camposNulables[snakeDeClave(clave)]
	return ok
}

// snakeDeClave convierte una clave PascalCase en el nombre de tabla snake_case que
// usa camposNulables (por ejemplo DatosIdentificacion -> datos_identificacion).
func snakeDeClave(clave string) string {
	var b strings.Builder
	for i, r := range clave {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteByte('_')
		}
		b.WriteRune(r)
	}
	return strings.ToLower(b.String())
}

// fmtRutaElem añade el índice de un elemento de colección a una ruta lógica.
func fmtRutaElem(ruta string, i int) string {
	return ruta + "[" + strconv.Itoa(i) + "]"
}
