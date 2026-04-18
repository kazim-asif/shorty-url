package main

import (
	"shorty-url/middleware"
	_ "shorty-url/routers"

	"github.com/beego/beego/v2/core/logs"
	beego "github.com/beego/beego/v2/server/web"
)

func main() {
	beego.InsertFilter("/*", beego.BeforeRouter, middleware.CORSMiddleware)
	beego.InsertFilter("/*", beego.BeforeRouter, middleware.LoggingMiddleware)
	beego.InsertFilter("/api/*", beego.BeforeRouter, middleware.RateLimitMiddleware)

	if beego.BConfig.RunMode == "dev" {
		beego.BConfig.WebConfig.DirectoryIndex = true
		beego.BConfig.WebConfig.StaticDir["/swagger"] = "swagger"
	}

	logs.Info("URL Shortener API starting...")
	beego.Run()
}
