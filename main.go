package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"

	beego "github.com/beego/beego/v2/server/web"
	apistatus "github.com/udistrital/utils_oas/v2/apiStatusLib"
	"github.com/udistrital/utils_oas/v2/customerror"
	"github.com/udistrital/utils_oas/v2/security"

	"github.com/udistrital/agora_mid/clients"
	"github.com/udistrital/agora_mid/config"
	"github.com/udistrital/agora_mid/controllers"
	_ "github.com/udistrital/agora_mid/routers"
	"github.com/udistrital/agora_mid/services"
)

func main() {
	cfg := config.Load()
	urls := cfg.URLs()

	relaciones := config.MustLoadRelaciones("config", urls)
	terceroConfig, ok := relaciones["tercero"]
	if !ok {
		log.Fatal("no existe la configuración de relaciones del servicio tercero")
	}

	tercerosService := services.TercerosService{
		BaseURL: urls["terceros"],
		Client:  clients.HTTPClient{},
	}

	tipoNatural := resolverTipoContribuyente(
		tercerosService, "MID_TIPO_CONTRIBUYENTE_NATURAL", "P_NATURAL",
	)
	tipoJuridica := resolverTipoContribuyente(
		tercerosService, "MID_TIPO_CONTRIBUYENTE_JURIDICA", "P_JURIDICA",
	)
	tipoDocumentoNIT := resolverTipoDocumento(
		tercerosService, "MID_TIPO_DOCUMENTO_NIT", "NIT",
	)

	catalogos := services.NewCatalogosService(
		clients.HTTPClient{},
		urls["terceros"],
		urls["ubicaciones"],
	)

	terceroWriteService := services.NewTerceroWriteService(
		clients.HTTPClient{},
		urls,
		terceroConfig,
		tipoNatural,
		tipoJuridica,
		tipoDocumentoNIT,
		catalogos,
	)
	controllers.ConfigurarTerceroWriteService(terceroWriteService)
	controllers.ConfigurarTerceroJuridicoWriteService(terceroWriteService)

	if beego.BConfig.RunMode == "dev" {
		beego.BConfig.WebConfig.DirectoryIndex = true
		beego.BConfig.WebConfig.StaticDir["/swagger"] = "swagger"
	}

	apistatus.Init()
	security.SetSecurityHeaders()
	beego.ErrorController(&customerror.CustomErrorController{})
	beego.Run()
}

// resolverTipoContribuyente obtiene el Id del tipo de contribuyente desde la
// variable de entorno indicada o, si no está configurada, desde el catálogo de
// terceros_crud.
func resolverTipoContribuyente(
	servicio services.TercerosService,
	variable string,
	codigo string,
) int {
	if valor := strings.TrimSpace(os.Getenv(variable)); valor != "" {
		id, err := strconv.Atoi(valor)
		if err != nil || id <= 0 {
			log.Fatalf("%s debe ser un entero positivo: %q", variable, valor)
		}
		return id
	}

	id, err := servicio.ResolverTipoContribuyente(context.Background(), codigo)
	if err != nil {
		log.Fatalf("resolviendo el tipo de contribuyente %s: %v", codigo, err)
	}
	return id
}

// resolverTipoDocumento obtiene el Id de un tipo de documento desde la variable
// de entorno indicada o, si no está configurada, desde el catálogo.
func resolverTipoDocumento(
	servicio services.TercerosService,
	variable string,
	codigo string,
) int {
	if valor := strings.TrimSpace(os.Getenv(variable)); valor != "" {
		id, err := strconv.Atoi(valor)
		if err != nil || id <= 0 {
			log.Fatalf("%s debe ser un entero positivo: %q", variable, valor)
		}
		return id
	}

	id, err := servicio.ResolverTipoDocumento(context.Background(), codigo)
	if err != nil {
		log.Fatalf("resolviendo el tipo de documento %s: %v", codigo, err)
	}
	return id
}
