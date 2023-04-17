package main

import (
	"github.com/julienschmidt/httprouter"
)

// SetupRoutes enforces the api routes.
func (api *APIHandler) SetupRoutes(router *httprouter.Router, m *Middlewares) *httprouter.Router {
	router.RedirectTrailingSlash = true
	router.GET("/", m.Chain(api.Index))
	router.GET("/status", m.Chain(api.Status))
	router.GET("/ops/configs", m.Chain(api.GetConfigs))
	router.GET("/ops/stats", m.Chain(api.GetStatistics))
	router.GET("/ops/maintenance", m.Chain(api.Maintenance))
	router.POST("/v1/books", m.Chain(api.CreateBook))
	router.GET("/v1/books", m.Chain(api.GetAllBooks))
	router.GET("/v1/books/:id", m.Chain(api.GetOneBook))
	router.PUT("/v1/books/:id", m.Chain(api.UpdateBook))
	router.DELETE("/v1/books/:id", m.Chain(api.DeleteOneBook))
	return router
}
