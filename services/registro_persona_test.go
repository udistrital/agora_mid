package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/udistrital/agora_mid/clients"
	"github.com/udistrital/agora_mid/dto"
)

func TestRegistroPersonaOrquestador(t *testing.T) {
	tests := []struct {
		name                  string
		tipo                  string
		codigoContribuyente   string
		codigoDocumento       string
		fallarRegistroDetalle bool
	}{
		{
			name:                "registro natural completo",
			tipo:                "natural",
			codigoContribuyente: "P_NATURAL",
			codigoDocumento:     "CC",
		},
		{
			name:                "registro jurídico completo",
			tipo:                "juridica",
			codigoContribuyente: "P_JURIDICA",
			codigoDocumento:     "NIT",
		},
		{
			name:                  "preserva IDs si falla el registro natural específico",
			tipo:                  "natural",
			codigoContribuyente:   "P_NATURAL",
			codigoDocumento:       "CC",
			fallarRegistroDetalle: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			terceroID := 51
			proveedorID := 81

			tipoDocumentoID := 3
			if tt.codigoDocumento == "NIT" {
				tipoDocumentoID = 7
			}

			tipoContribuyenteID := 1
			if tt.codigoContribuyente == "P_JURIDICA" {
				tipoContribuyenteID = 2
			}

			terceroServer := httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")

					switch {
					case r.Method == http.MethodGet && r.URL.Path == "/v1/tipo_documento":
						if !strings.Contains(
							r.URL.Query().Get("query"),
							"CodigoAbreviacion:"+tt.codigoDocumento,
						) {
							t.Errorf("filtro de documento inesperado: %s", r.URL.Query().Get("query"))
						}
						fmt.Fprintf(
							w,
							`{"Success":true,"Data":[{"Id":%d,"CodigoAbreviacion":%q,"Activo":true}]}`,
							tipoDocumentoID,
							tt.codigoDocumento,
						)

					case r.Method == http.MethodGet && r.URL.Path == "/v1/datos_identificacion":
						fmt.Fprint(w, `{"Success":true,"Data":[]}`)

					case r.Method == http.MethodGet && r.URL.Path == "/v1/tipo_contribuyente":
						if !strings.Contains(
							r.URL.Query().Get("query"),
							"CodigoAbreviacion:"+tt.codigoContribuyente,
						) {
							t.Errorf("filtro de contribuyente inesperado: %s", r.URL.Query().Get("query"))
						}
						fmt.Fprintf(
							w,
							`{"Success":true,"Data":[{"Id":%d,"CodigoAbreviacion":%q,"Activo":true}]}`,
							tipoContribuyenteID,
							tt.codigoContribuyente,
						)

					case r.Method == http.MethodPost && r.URL.Path == "/v1/tercero/":
						var body struct {
							NombreCompleto      string `json:"NombreCompleto"`
							TipoContribuyenteID struct {
								Id int `json:"Id"`
							} `json:"TipoContribuyenteId"`
						}
						if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
							t.Errorf("JSON de tercero inválido: %v", err)
						}
						if body.NombreCompleto == "" {
							t.Error("NombreCompleto no debe estar vacío")
						}
						if body.TipoContribuyenteID.Id != tipoContribuyenteID {
							t.Errorf(
								"tipo de contribuyente enviado %d; esperado %d",
								body.TipoContribuyenteID.Id,
								tipoContribuyenteID,
							)
						}
						fmt.Fprintf(w, `{"Success":true,"Data":{"Id":%d}}`, terceroID)

					case r.Method == http.MethodPost && r.URL.Path == "/v1/datos_identificacion/":
						var body struct {
							Numero    string `json:"Numero"`
							TerceroID struct {
								Id int `json:"Id"`
							} `json:"TerceroId"`
							TipoDocumentoID struct {
								Id int `json:"Id"`
							} `json:"TipoDocumentoId"`
						}
						if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
							t.Errorf("JSON de identificación inválido: %v", err)
						}
						if body.TerceroID.Id != terceroID {
							t.Errorf("tercero_id enviado %d; esperado %d", body.TerceroID.Id, terceroID)
						}
						if body.TipoDocumentoID.Id != tipoDocumentoID {
							t.Errorf("tipo documental enviado %d; esperado %d", body.TipoDocumentoID.Id, tipoDocumentoID)
						}
						if body.Numero == "" {
							t.Error("el número de documento no debe estar vacío")
						}
						fmt.Fprint(w, `{"Success":true,"Data":{"Id":100}}`)

					default:
						t.Errorf("petición inesperada a Terceros: %s %s", r.Method, r.URL.String())
						http.NotFound(w, r)
					}
				},
			))
			defer terceroServer.Close()

			proveedorCreado := false

			crudServer := httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")

					switch {
					case r.Method == http.MethodGet && r.URL.Path == "/v1/proveedor/":
						if proveedorCreado {
							fmt.Fprintf(
								w,
								`{"Data":[{"Id":%d,"TerceroId":%d,"TipoRegistroId":%d,"Activo":true}]}`,
								proveedorID,
								terceroID,
								tipoContribuyenteID,
							)
						} else {
							fmt.Fprint(w, `{"Data":[{}]}`)
						}

					case r.Method == http.MethodPost && r.URL.Path == "/v1/proveedor/":
						var body struct {
							TerceroID      int    `json:"TerceroId"`
							TipoRegistroID int    `json:"TipoRegistroId"`
							Descripcion    string `json:"DescripcionPortafolio"`
						}
						if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
							t.Errorf("JSON de proveedor inválido: %v", err)
						}
						if body.TerceroID != terceroID {
							t.Errorf("tercero_id enviado %d; esperado %d", body.TerceroID, terceroID)
						}
						if body.TipoRegistroID != tipoContribuyenteID {
							t.Errorf("tipo_registro_id enviado %d; esperado %d", body.TipoRegistroID, tipoContribuyenteID)
						}

						proveedorCreado = true
						w.WriteHeader(http.StatusCreated)
						fmt.Fprintf(
							w,
							`{"Success":true,"Data":{"Id":%d,"TerceroId":%d,"TipoRegistroId":%d,"Activo":true}}`,
							proveedorID,
							terceroID,
							tipoContribuyenteID,
						)

					case r.Method == http.MethodGet &&
						(r.URL.Path == "/v1/proveedor_natural/81" ||
							r.URL.Path == "/v1/proveedor_juridico/81"):
						http.NotFound(w, r)

					case r.Method == http.MethodPost && r.URL.Path == "/v1/proveedor_natural/":
						if tt.tipo != "natural" {
							t.Errorf("no debía crearse proveedor natural en el caso %s", tt.tipo)
						}

						var body struct {
							Id              int `json:"Id"`
							PerfilDeclarado struct {
								Id int `json:"Id"`
							} `json:"PerfilDeclarado"`
							ExperienciaLaboralMeses     int `json:"ExperienciaLaboralMeses"`
							ExperienciaProfesionalMeses int `json:"ExperienciaProfesionalMeses"`
						}
						if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
							t.Errorf("JSON de proveedor natural inválido: %v", err)
						}
						if body.Id != proveedorID {
							t.Errorf("ID del proveedor natural %d; esperado %d", body.Id, proveedorID)
						}

						if tt.fallarRegistroDetalle {
							w.WriteHeader(http.StatusInternalServerError)
							fmt.Fprint(w, `{"error":"fallo simulado del registro natural"}`)
							return
						}

						fmt.Fprintf(
							w,
							`{"Data":{"Id":%d,"PerfilDeclarado":{"Id":%d},"ExperienciaLaboralMeses":12,"ExperienciaProfesionalMeses":6,"Activo":true}}`,
							proveedorID,
							body.PerfilDeclarado.Id,
						)

					case r.Method == http.MethodPost && r.URL.Path == "/v1/proveedor_juridico/":
						if tt.tipo != "juridica" {
							t.Errorf("no debía crearse proveedor jurídico en el caso %s", tt.tipo)
						}

						var body struct {
							Id                 int    `json:"Id"`
							NombreComercial    string `json:"NombreComercial"`
							MatriculaMercantil int    `json:"MatriculaMercantil"`
						}
						if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
							t.Errorf("JSON de proveedor jurídico inválido: %v", err)
						}
						if body.Id != proveedorID {
							t.Errorf("ID del proveedor jurídico %d; esperado %d", body.Id, proveedorID)
						}

						fmt.Fprintf(
							w,
							`{"Data":{"Id":%d,"NombreComercial":%q,"MatriculaMercantil":%d,"FechaConstitucion":"2020-01-01T00:00:00Z","FechaRenovacion":"2026-01-01T00:00:00Z","ReportaBeneficiosFinales":false,"CotizaBolsa":false,"RequiereRevisorFiscal":false,"Activo":true}}`,
							proveedorID,
							body.NombreComercial,
							body.MatriculaMercantil,
						)

					default:
						t.Errorf("petición inesperada al CRUD: %s %s", r.Method, r.URL.String())
						http.NotFound(w, r)
					}
				},
			))
			defer crudServer.Close()

			orquestador := RegistroPersonaService{
				Terceros: TercerosService{
					BaseURL: terceroServer.URL,
					Client:  clients.HTTPClient{Client: terceroServer.Client()},
				},
				GestionPersonas: GestionPersonasService{
					BaseURL: crudServer.URL,
					Client:  clients.HTTPClient{Client: crudServer.Client()},
				},
			}

			var resultado RegistroPersonaResultado
			var err error

			if tt.tipo == "natural" {
				resultado, err = orquestador.RegistrarPersonaNatural(
					context.Background(),
					dto.RegistroPersonaNaturalRequest{
						DatosTerceros: dto.DatosTercerosDTO{
							NombreCompleto:      "Ana Pérez",
							PrimerNombre:        "Ana",
							PrimerApellido:      "Pérez",
							CodigoTipoDocumento: tt.codigoDocumento,
							NumeroDocumento:     "123456",
						},
						Proveedor: dto.ProveedorDTO{
							DescripcionPortafolio: "Servicios de prueba",
						},
						ProveedorNatural: dto.ProveedorNaturalDTO{
							PerfilDeclaradoID:           7,
							ExperienciaLaboralMeses:     12,
							ExperienciaProfesionalMeses: 6,
						},
					},
				)
			} else {
				resultado, err = orquestador.RegistrarPersonaJuridica(
					context.Background(),
					dto.RegistroPersonaJuridicaRequest{
						DatosTerceros: dto.DatosTercerosDTO{
							NombreCompleto:      "Empresa de prueba S.A.S.",
							CodigoTipoDocumento: tt.codigoDocumento,
							NumeroDocumento:     "900123456",
						},
						Proveedor: dto.ProveedorDTO{
							DescripcionPortafolio: "Servicios empresariales de prueba",
						},
						ProveedorJuridico: dto.ProveedorJuridicoDTO{
							NombreComercial:          "Empresa de prueba",
							MatriculaMercantil:       12345,
							FechaConstitucion:        "2020-01-01",
							FechaRenovacion:          "2026-01-01",
							ReportaBeneficiosFinales: false,
							CotizaBolsa:              false,
							RequiereRevisorFiscal:    false,
						},
					},
				)
			}

			if tt.fallarRegistroDetalle {
				if err == nil {
					t.Fatal("se esperaba error en el registro específico")
				}
				if resultado.TerceroID != terceroID {
					t.Errorf("tercero_id conservado %d; esperado %d", resultado.TerceroID, terceroID)
				}
				if resultado.ProveedorID != proveedorID {
					t.Errorf("proveedor_id conservado %d; esperado %d", resultado.ProveedorID, proveedorID)
				}
				return
			}

			if err != nil {
				t.Fatalf("el registro debía completarse: %v", err)
			}
			if resultado.TerceroID != terceroID {
				t.Errorf("tercero_id %d; esperado %d", resultado.TerceroID, terceroID)
			}
			if resultado.ProveedorID != proveedorID {
				t.Errorf("proveedor_id %d; esperado %d", resultado.ProveedorID, proveedorID)
			}

			if tt.tipo == "natural" {
				if resultado.ProveedorNatural == nil {
					t.Fatal("no se devolvió el registro específico natural")
				}
				if resultado.ProveedorNatural.Id != proveedorID {
					t.Errorf("ID natural %d; esperado %d", resultado.ProveedorNatural.Id, proveedorID)
				}
			} else {
				if resultado.ProveedorJuridico == nil {
					t.Fatal("no se devolvió el registro específico jurídico")
				}
				if resultado.ProveedorJuridico.Id != proveedorID {
					t.Errorf("ID jurídico %d; esperado %d", resultado.ProveedorJuridico.Id, proveedorID)
				}
			}
		})
	}
}
