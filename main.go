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

	// Cargar y validar la lista negra global de campos de expansión. Aborta el
	// arranque si el archivo es inválido o contiene una entrada inconsistente
	// (por ejemplo ANY/ANY/ANY). Aún no se inyecta en el motor de expansión: esa
	// conexión llega con el recorte por lista negra. Validarla aquí deja el
	// arranque como el punto único donde una configuración incoherente falla
	// temprano.
	config.MustLoadBlacklist("config/global_blacklist.yaml", relaciones)

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
	proveedorWriteService := services.NewProveedorWriteService(
		clients.HTTPClient{},
		urls,
		proveedorConfig,
		tipoNatural,
	)
	proveedorHandler := controllers.NewProveedorHandlerCompleto(
		proveedorService,
		proveedorWriteService,
	)

	// /v1/proveedor expone lectura (GET) y escritura (POST/PATCH) sobre el mismo
	// recurso raíz; el enrutado por método delega en el handler correspondiente.
	beego.Any(
		"/v1/proveedor",
		adaptarHandler(enrutarProveedor(proveedorHandler)),
	)

	// /v1/proveedor/consulta expone la lectura del proveedor por cuerpo JSON. Solo
	// acepta POST; otros métodos responden 405 con Allow: POST (resuelto en el
	// handler Consultar).
	beego.Any(
		"/v1/proveedor/consulta",
		adaptarHandler(http.HandlerFunc(proveedorHandler.Consultar)),
	)

	// Publicar la definición OpenAPI estática en /swagger. El contrato vive en
	// swagger/swagger.yml y queda accesible como GET /swagger/swagger.yml.
	// swagger.json se deriva determinísticamente del YAML en cada arranque para
	// que Swagger UI lo encuentre con su nombre por defecto.
	if err := generarSwaggerJSON("swagger/swagger.yml", "swagger/swagger.json"); err != nil {
		log.Fatalf("generando swagger.json: %v", err)
	}
	beego.SetStaticPath("/swagger", "swagger")

	beego.ErrorController(&customerror.CustomErrorController{})
	apiStatus.Init()
	beego.Run()
}

func adaptarHandler(handler http.Handler) func(*beecontext.Context) {
	return func(ctx *beecontext.Context) {
		handler.ServeHTTP(ctx.ResponseWriter, ctx.Request)
	}
}

// enrutarProveedor delega GET a la lectura del proveedor y POST/PATCH a su
// escritura, sobre el mismo recurso raíz /v1/proveedor.
func enrutarProveedor(h *controllers.ProveedorHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost, http.MethodPatch:
			h.Escribir(w, r)
		default:
			h.Leer(w, r)
		}
	})
}
