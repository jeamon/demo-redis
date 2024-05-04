package main

import (
	_ "github.com/jeamon/demo-redis/docs"
	"github.com/julienschmidt/httprouter"
	httpswagger "github.com/swaggo/http-swagger/v2"
)

// SetupRoutes injects book and ops related endpoints if required.
func (api *APIHandler) SetupRoutes(router *httprouter.Router, m *MiddlewareMap) *httprouter.Router {
	router.RedirectTrailingSlash = true
	router.NotFound = api.NotFound()
	api.SetupBookRoutes(router, m)
	if api.config.OpsEndpointsEnable {
		api.SetupOpsRoutes(router, m)
	}
	router.GET("/swagger/", m.public(api.OpsHandlerWrapper(httpswagger.WrapHandler)))
	return router
}
