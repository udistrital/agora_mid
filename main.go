package main

import (
	"net/http"

	beego "github.com/beego/beego/v2/server/web"
	apiStatus "github.com/udistrital/utils_oas/v2/apiStatusLib"

	"github.com/udistrital/agora_mid/config"
	"github.com/udistrital/agora_mid/controllers"
)

func main() {
	apiStatus.Init()

	cfg := config.Load()
	handler := controllers.NewRegistroPersonaHandler(cfg)

	beego.Handler(
		"/v1/registro/persona-natural",
		http.HandlerFunc(handler.RegistrarNatural),
	)
	beego.Handler(
		"/v1/registro/persona-juridica",
		http.HandlerFunc(handler.RegistrarJuridica),
	)

	beego.Run()
}
