package main

import (
	"github.com/julienschmidt/httprouter"
)

// SetupRoutes injects book and ops related endpoints if required.
func (api *APIHandler) SetupRoutes(router *httprouter.Router, m *MiddlewareMap) *httprouter.Router {
	router.RedirectTrailingSlash = true
	api.SetupBookRoutes(router, m)
	if api.config.OpsEndpointsEnable {
		api.SetupOpsRoutes(router, m)
	}
	return router
}
