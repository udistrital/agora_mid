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
	tipoNatural := 0
	if valor := strings.TrimSpace(os.Getenv("MID_TIPO_CONTRIBUYENTE_NATURAL")); valor != "" {
		id, err := strconv.Atoi(valor)
		if err != nil || id <= 0 {
			log.Fatalf("MID_TIPO_CONTRIBUYENTE_NATURAL debe ser un entero positivo: %q", valor)
		}
		tipoNatural = id
	}
	if tipoNatural == 0 {
		var err error
		tipoNatural, err = tercerosService.ResolverTipoContribuyente(
			context.Background(),
			"P_NATURAL",
		)
		if err != nil {
			log.Fatalf("resolviendo el tipo de contribuyente P_NATURAL: %v", err)
		}
	}

	terceroWriteService := services.NewTerceroWriteService(
		clients.HTTPClient{},
		urls,
		terceroConfig,
		tipoNatural,
	)
	controllers.ConfigurarTerceroWriteService(terceroWriteService)

	if beego.BConfig.RunMode == "dev" {
		beego.BConfig.WebConfig.DirectoryIndex = true
		beego.BConfig.WebConfig.StaticDir["/swagger"] = "swagger"
	}

	apistatus.Init()
	security.SetSecurityHeaders()
	beego.ErrorController(&customerror.CustomErrorController{})
	beego.Run()
}
