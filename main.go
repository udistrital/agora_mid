package main

import (
	"net/http"

	beego "github.com/beego/beego/v2/server/web"
	beecontext "github.com/beego/beego/v2/server/web/context"
	apiStatus "github.com/udistrital/utils_oas/v2/apiStatusLib"

	"github.com/udistrital/agora_mid/config"
	"github.com/udistrital/agora_mid/controllers"
)

func main() {
	apiStatus.Init()

	cfg := config.Load()
	handler := controllers.NewRegistroPersonaHandler(cfg)

	// Beego.Handler no sirve aquí porque el registro de método no quedó abierto
	// a GET/PUT en la prueba local. Any delega todos los métodos al handler,
	// que aplica la validación y distribución según r.Method.
	beego.Any(
		"/v1/registro/persona-natural",
		adaptarHandler(http.HandlerFunc(handler.RegistrarNatural)),
	)
	beego.Any(
		"/v1/registro/persona-juridica",
		adaptarHandler(http.HandlerFunc(handler.RegistrarJuridica)),
	)

	beego.Run()
}

func adaptarHandler(handler http.Handler) func(*beecontext.Context) {
	return func(ctx *beecontext.Context) {
		handler.ServeHTTP(ctx.ResponseWriter, ctx.Request)
	}
}
