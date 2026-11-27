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
	if definidas, err := config.CargarDotEnv(".env"); err != nil {
		log.Printf("config: no se pudo cargar .env: %v", err)
	} else if definidas > 0 {
		log.Printf("config: .env cargado (%d variables)", definidas)
	}

	// Beego lee conf/app.conf durante la inicialización del paquete, antes de
	// main. Se recarga aquí para que las variables cargadas desde .env también
	// apliquen a httpport, runmode y a los hosts de los CRUD.
	if err := beego.LoadAppConfig("ini", "conf/app.conf"); err != nil {
		log.Printf("config: no se pudo recargar conf/app.conf: %v", err)
	}

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

	ids, err := config.CargarIDsCatalogo(context.Background(), os.Getenv, tercerosService)
	if err != nil {
		log.Fatalf("resolviendo los catálogos del dominio 1: %v", err)
	}
	log.Printf(
		"catálogos: P_NATURAL=%s, P_JURIDICA=%s, NIT=%s",
		ids.TipoContribuyenteNatural,
		ids.TipoContribuyenteJuridica,
		ids.TipoDocumentoNIT,
	)

	catalogos := services.NewCatalogosService(
		clients.HTTPClient{},
		urls["terceros"],
		urls["ubicaciones"],
	)
	overridesInfoComplementaria, err := config.CargarOverridesInfoComplementaria(os.Getenv)
	if err != nil {
		log.Fatalf("resolviendo overrides de info_complementaria: %v", err)
	}
	catalogos.InfoComplementariaOverrides = overridesInfoComplementaria
	if len(overridesInfoComplementaria) > 0 {
		log.Printf("info_complementaria: %d overrides por entorno", len(overridesInfoComplementaria))
	}
	if err := catalogos.PrecargarInfoComplementaria(context.Background()); err != nil {
		log.Printf("info_complementaria: no se pudo precargar: %v", err)
	} else {
		log.Printf("info_complementaria: catálogo precargado")
	}
	parametros := services.NewParametrosService(
		clients.HTTPClient{},
		urls["parametros"],
	)

	terceroWriteService := services.NewTerceroWriteService(
		clients.HTTPClient{},
		urls,
		terceroConfig,
		ids.TipoContribuyenteNatural.Valor,
		ids.TipoContribuyenteJuridica.Valor,
		ids.TipoDocumentoNIT.Valor,
		catalogos,
	)
	controllers.ConfigurarTerceroWriteService(terceroWriteService)
	controllers.ConfigurarTerceroJuridicoWriteService(terceroWriteService)
	controllers.ConfigurarCuentaBancariaWriter(services.CuentaBancariaService{})

	caracterizacionNatural := services.NewCaracterizacionNaturalService(
		catalogos,
		parametros,
		ids.TipoContribuyenteNatural.Valor,
		enteroEnvODefecto("MID_TIPO_PROVEEDOR_NATURAL", 1),
		os.Getenv("MID_TIPO_PARAMETRO_CIIU"),
		os.Getenv("MID_TIPO_PARAMETRO_CANAL_CONTACTO"),
	)
	controllers.ConfigurarCaracterizacionNaturalWriteService(caracterizacionNatural)

	caracterizacionJuridica := services.NewCaracterizacionJuridicaService(
		catalogos,
		catalogos,
		parametros,
		ids.TipoContribuyenteJuridica.Valor,
		ids.TipoContribuyenteNatural.Valor,
		enteroEnvODefecto("MID_TIPO_PROVEEDOR_JURIDICO", 2),
		enteroEnvODefecto("MID_TIPO_PROVEEDOR_NATURAL", 1),
		os.Getenv("MID_TIPO_PARAMETRO_CIIU"),
		os.Getenv("MID_TIPO_PARAMETRO_CANAL_CONTACTO"),
		os.Getenv("MID_TIPO_PARAMETRO_REGIMEN_CONTRIBUTIVO"),
		os.Getenv("MID_TIPO_PARAMETRO_TIPO_CONFORMACION"),
		os.Getenv("MID_GRUPO_PROCEDENCIA"),
		os.Getenv("MID_GRUPO_CARGO"),
		os.Getenv("MID_GRUPO_PROVEEDOR_JURIDICO"),
	)
	controllers.ConfigurarCaracterizacionJuridicaWriteService(caracterizacionJuridica)

	if beego.BConfig.RunMode == "dev" {
		beego.BConfig.WebConfig.DirectoryIndex = true
		beego.BConfig.WebConfig.StaticDir["/swagger"] = "swagger"
	}

	apistatus.Init()
	security.SetSecurityHeaders()
	beego.ErrorController(&customerror.CustomErrorController{})
	beego.Run()
}

// enteroEnvODefecto lee un entero positivo desde el entorno. Si la variable no
// está definida usa el valor por defecto; si está definida y es inválida, el
// arranque falla para no operar con IDs incorrectos.
func enteroEnvODefecto(nombre string, defecto int) int {
	valor := strings.TrimSpace(os.Getenv(nombre))
	if valor == "" {
		return defecto
	}
	id, err := strconv.Atoi(valor)
	if err != nil || id <= 0 {
		log.Fatalf("%s debe ser un entero positivo: %q", nombre, valor)
	}
	return id
}
