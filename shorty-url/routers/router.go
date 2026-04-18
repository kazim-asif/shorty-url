// @APIVersion 1.0.0
// @Title URL Shortener API
// @Description Professional URL shortener service with analytics
// @Contact dev@urlshortener.com
// @TermsOfServiceUrl http://localhost:8080/terms
// @License MIT
// @LicenseUrl https://opensource.org/licenses/MIT
package routers

import (
	"shorty-url/controllers"

	beego "github.com/beego/beego/v2/server/web"
)

func init() {
	ns := beego.NewNamespace("/api/v1",
		beego.NSNamespace("/urls",
			beego.NSRouter("/", &controllers.URLController{}, "post:Post"),
			beego.NSRouter("/list", &controllers.URLController{}, "get:GetAll"),
		),
	)
	beego.AddNamespace(ns)

	beego.Router("/:shortCode", &controllers.URLController{}, "get:Get;delete:Delete")
	beego.Router("/:shortCode/stats", &controllers.URLController{}, "get:GetStats")
	beego.Router("/:shortCode/analytics", &controllers.URLController{}, "get:GetAnalytics")
}
