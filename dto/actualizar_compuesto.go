package dto

import (
	"encoding/json"
	"errors"
	"fmt"
)

func mapearCamposContratoActualizacion(
	destino *map[string]json.RawMessage,
	origen map[string]json.RawMessage,
	mapeo map[string]string,
) {
	for campoPublico, campoInterno := range mapeo {
		if valor, existe := origen[campoPublico]; existe {
			if *destino == nil {
				*destino = make(map[string]json.RawMessage)
			}
			(*destino)[campoInterno] = valor
		}
	}
}

func validarCamposContratoActualizacion(
	obj map[string]json.RawMessage,
	permitidos map[string]bool,
	contexto string,
) error {
	for campo := range obj {
		if !permitidos[campo] {
			return fmt.Errorf("campo no permitido en %s: %s", contexto, campo)
		}
	}
	return nil
}

func (r *ActualizarPersonaRequest) UnmarshalJSON(data []byte) error {
	var raiz map[string]json.RawMessage
	if err := json.Unmarshal(data, &raiz); err != nil {
		return err
	}

	if err := validarCamposContratoActualizacion(raiz, map[string]bool{
		"Id": true, "Tercero": true, "DescripcionPortafolio": true,
		"PerfilDeclarado": true, "ExperienciaLaboralMeses": true,
		"ExperienciaProfesionalMeses": true, "NombreComercial": true,
		"MatriculaMercantil": true, "FechaConstitucion": true,
		"FechaRenovacion": true, "ReportaBeneficiosFinales": true,
		"CotizaBolsa": true, "RequiereRevisorFiscal": true,
	}, "proveedor"); err != nil {
		return err
	}

	var req ActualizarPersonaRequest

	if rawID, existe := raiz["Id"]; existe {
		if err := json.Unmarshal(rawID, &req.ProveedorID); err != nil ||
			req.ProveedorID <= 0 {
			return errors.New("Id del proveedor debe ser un entero positivo")
		}
	}

	if rawTercero, existe := raiz["Tercero"]; existe &&
		string(rawTercero) != "null" {
		var tercero map[string]json.RawMessage
		if err := json.Unmarshal(rawTercero, &tercero); err != nil {
			return fmt.Errorf("Tercero debe ser un objeto: %w", err)
		}

		if err := validarCamposContratoActualizacion(tercero, map[string]bool{
			"Id": true, "NombreCompleto": true, "PrimerNombre": true,
			"SegundoNombre": true, "PrimerApellido": true,
			"SegundoApellido": true, "FechaNacimiento": true,
			"LugarOrigen": true, "DatosIdentificacion": true,
		}, "Tercero"); err != nil {
			return err
		}

		if rawID, existe := tercero["Id"]; existe {
			if err := json.Unmarshal(rawID, &req.TerceroID); err != nil {
				return errors.New("Tercero.Id debe ser un entero")
			}
		}

		mapearCamposContratoActualizacion(&req.Terceros, tercero,
			map[string]string{
				"NombreCompleto":  "nombre_completo",
				"PrimerNombre":    "primer_nombre",
				"SegundoNombre":   "segundo_nombre",
				"PrimerApellido":  "primer_apellido",
				"SegundoApellido": "segundo_apellido",
				"FechaNacimiento": "fecha_nacimiento",
				"LugarOrigen":     "lugar_origen",
			})

		if rawIDs, existe := tercero["DatosIdentificacion"]; existe &&
			string(rawIDs) != "null" {
			var identificaciones []map[string]json.RawMessage
			if err := json.Unmarshal(rawIDs, &identificaciones); err != nil {
				return fmt.Errorf(
					"Tercero.DatosIdentificacion debe ser una lista: %w", err,
				)
			}
			if len(identificaciones) > 1 {
				return errors.New(
					"solo se puede actualizar una identificación por solicitud",
				)
			}

			if len(identificaciones) == 1 {
				identificacion := identificaciones[0]

				if _, cambiaNumero := identificacion["Numero"]; cambiaNumero {
					if _, existe := identificacion["TipoDocumentoId"]; !existe {
						return errors.New(
							"TipoDocumentoId.CodigoAbreviacion es obligatorio al cambiar Numero",
						)
					}
				}
				if err := validarCamposContratoActualizacion(
					identificacion,
					map[string]bool{
						"Id": true, "Numero": true, "TipoDocumentoId": true,
						"DigitoVerificacion": true, "CiudadExpedicion": true,
						"FechaExpedicion": true,
					},
					"Tercero.DatosIdentificacion",
				); err != nil {
					return err
				}

				mapearCamposContratoActualizacion(
					&req.DatosIdentificacion,
					identificacion,
					map[string]string{
						"Id":                 "id",
						"Numero":             "numero_documento",
						"DigitoVerificacion": "digito_verificacion",
						"CiudadExpedicion":   "ciudad_expedicion",
						"FechaExpedicion":    "fecha_expedicion",
					},
				)

				if rawTipo, existe := identificacion["TipoDocumentoId"]; existe {
					var tipoDocumento map[string]json.RawMessage
					if err := json.Unmarshal(rawTipo, &tipoDocumento); err != nil {
						return fmt.Errorf(
							"TipoDocumentoId debe ser un objeto: %w", err,
						)
					}

					if err := validarCamposContratoActualizacion(
						tipoDocumento,
						map[string]bool{
							"Id": true, "CodigoAbreviacion": true,
						},
						"Tercero.DatosIdentificacion.TipoDocumentoId",
					); err != nil {
						return err
					}

					if codigo, existe := tipoDocumento["CodigoAbreviacion"]; existe {
						if req.DatosIdentificacion == nil {
							req.DatosIdentificacion =
								make(map[string]json.RawMessage)
						}
						req.DatosIdentificacion["codigo_tipo_documento"] = codigo
					} else if _, cambiaNumero := identificacion["Numero"]; cambiaNumero {
						return errors.New(
							"TipoDocumentoId.CodigoAbreviacion es obligatorio al cambiar Numero",
						)
					}
				}
			}
		}
	}

	mapearCamposContratoActualizacion(&req.Proveedor, raiz,
		map[string]string{
			"DescripcionPortafolio": "descripcion_portafolio",
		})

	mapearCamposContratoActualizacion(&req.ProveedorNatural, raiz,
		map[string]string{
			"ExperienciaLaboralMeses":     "experiencia_laboral_meses",
			"ExperienciaProfesionalMeses": "experiencia_profesional_meses",
		})

	if rawPerfil, existe := raiz["PerfilDeclarado"]; existe &&
		string(rawPerfil) != "null" {
		var perfil map[string]json.RawMessage
		if err := json.Unmarshal(rawPerfil, &perfil); err != nil {
			return fmt.Errorf("PerfilDeclarado debe ser un objeto: %w", err)
		}
		if err := validarCamposContratoActualizacion(
			perfil, map[string]bool{"Id": true}, "PerfilDeclarado",
		); err != nil {
			return err
		}
		rawID, existe := perfil["Id"]
		if !existe {
			return errors.New("PerfilDeclarado.Id es obligatorio")
		}
		if req.ProveedorNatural == nil {
			req.ProveedorNatural = make(map[string]json.RawMessage)
		}
		req.ProveedorNatural["perfil_declarado_id"] = rawID
	}

	mapearCamposContratoActualizacion(&req.ProveedorJuridico, raiz,
		map[string]string{
			"NombreComercial":          "nombre_comercial",
			"MatriculaMercantil":       "matricula_mercantil",
			"FechaConstitucion":        "fecha_constitucion",
			"FechaRenovacion":          "fecha_renovacion",
			"ReportaBeneficiosFinales": "reporta_beneficios_finales",
			"CotizaBolsa":              "cotiza_bolsa",
			"RequiereRevisorFiscal":    "requiere_revisor_fiscal",
		})

	*r = req
	return nil
}
