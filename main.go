package main

import (
	"context"
	"log"
	"net/http"

	beego "github.com/beego/beego/v2/server/web"
	beecontext "github.com/beego/beego/v2/server/web/context"
	apiStatus "github.com/udistrital/utils_oas/v2/apiStatusLib"
	"github.com/udistrital/utils_oas/v2/customerror"

	"github.com/udistrital/agora_mid/clients"
	"github.com/udistrital/agora_mid/config"
	"github.com/udistrital/agora_mid/controllers"
	"github.com/udistrital/agora_mid/services"
)

func main() {
	cfg := config.Load()

	// Cargar y validar las relaciones declaradas en config/relaciones_*.yaml.
	urls := cfg.URLs()
	relaciones := config.MustLoadRelaciones("config", urls)

	proveedorConfig, ok := relaciones["proveedor"]
	if !ok {
		log.Fatal("no existe la configuración de relaciones del servicio proveedor")
	}

	tercerosService := services.TercerosService{
		BaseURL: urls["terceros"],
		Client:  clients.HTTPClient{},
	}

	tipoNatural, err := tercerosService.ResolverTipoContribuyente(
		context.Background(),
		"P_NATURAL",
	)
	if err != nil {
		log.Fatalf("resolviendo el tipo de registro P_NATURAL: %v", err)
	}

	proveedorService := services.NewProveedorService(
		clients.HTTPClient{},
		urls,
		proveedorConfig,
		tipoNatural,
		nil,
	)
	proveedorHandler := controllers.NewProveedorHandler(proveedorService)

	beego.Any(
		"/v1/proveedor",
		adaptarHandler(http.HandlerFunc(proveedorHandler.Leer)),
	)

	beego.ErrorController(&customerror.CustomErrorController{})
	apiStatus.Init()
	beego.Run()
}

func adaptarHandler(handler http.Handler) func(*beecontext.Context) {
	return func(ctx *beecontext.Context) {
		handler.ServeHTTP(ctx.ResponseWriter, ctx.Request)
	}
}
