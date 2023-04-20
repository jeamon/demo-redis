package main

import (
	"github.com/julienschmidt/httprouter"
)

// MiddlewareMap contains middlwares chain to
// use for public-facing and ops requests.
type MiddlewareMap struct {
	public *Middlewares
	ops    *Middlewares
}

// SetupRoutes enforces the api routes.
func (api *APIHandler) SetupRoutes(router *httprouter.Router, m *MiddlewareMap) *httprouter.Router {
	router.RedirectTrailingSlash = true
	router.GET("/", m.public.Chain(api.Index))
	router.GET("/status", m.public.Chain(api.Status))

	router.POST("/v1/books", m.public.Chain(api.CreateBook))
	router.GET("/v1/books", m.public.Chain(api.GetAllBooks))
	router.GET("/v1/books/:id", m.public.Chain(api.GetOneBook))
	router.PUT("/v1/books/:id", m.public.Chain(api.UpdateBook))
	router.DELETE("/v1/books/:id", m.public.Chain(api.DeleteOneBook))

	router.GET("/ops/configs", m.ops.Chain(api.GetConfigs))
	router.GET("/ops/stats", m.ops.Chain(api.GetStatistics))
	router.GET("/ops/maintenance", m.ops.Chain(api.Maintenance))
	return router
}
