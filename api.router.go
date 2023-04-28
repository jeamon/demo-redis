package main

import (
	"github.com/julienschmidt/httprouter"
)

// SetupRoutes enforces the api routes.
func (api *APIHandler) SetupRoutes(router *httprouter.Router, m *MiddlewareMap) *httprouter.Router {
	router.RedirectTrailingSlash = true
	router.GET("/", m.public(api.Index))
	router.GET("/status", m.public(api.Status))

	router.POST("/v1/books", m.public(api.CreateBook))
	router.GET("/v1/books", m.public(api.GetAllBooks))
	router.GET("/v1/books/:id", m.public(api.GetOneBook))
	router.PUT("/v1/books/:id", m.public(api.UpdateBook))
	router.DELETE("/v1/books/:id", m.public(api.DeleteOneBook))

	router.GET("/ops/configs", m.ops(api.GetConfigs))
	router.GET("/ops/stats", m.ops(api.GetStatistics))
	router.GET("/ops/maintenance", m.ops(api.Maintenance))
	router.GET("/ops/debug/vars", m.ops(GetVars))
	return router
}
