// @APIVersion 1.0.0
// @Title beego Test API
// @Description beego has a very cool tools to autogenerate documents for your API
// @Contact astaxie@gmail.com
// @TermsOfServiceUrl http://beego.me/
// @License Apache 2.0
// @LicenseUrl http://www.apache.org/licenses/LICENSE-2.0.html
package routers

import (
	"github.com/udistrital/agora_mid/controllers"

	beego "github.com/beego/beego/v2/server/web"
)

func init() {
	ns := beego.NewNamespace("/v1",
		beego.NSNamespace("/proveedores",
			beego.NSRouter("/natural", &controllers.ProveedorNaturalController{}, "post:Post"),
			beego.NSRouter("/natural/:terceroId", &controllers.ProveedorNaturalController{}, "put:Put"),
		),
	)
	beego.AddNamespace(ns)
}
