package main

import "github.com/julienschmidt/httprouter"

// SetupRoutes enforces the api routes.
func (api *APIHandler) SetupRoutes(router *httprouter.Router, m Middlewares) *httprouter.Router {
	router.RedirectTrailingSlash = true
	router.GET("/", m.Chain(api.Index))
	router.GET("/status", m.Chain(api.Index))
	// router.GET("/ops/stats", api.GetStats)
	router.GET("/ops/configs", m.Chain(api.GetConfigs))
	router.POST("/v1/books", m.Chain(api.CreateBook))
	router.GET("/v1/books", m.Chain(api.GetAllBooks))
	router.GET("/v1/books/:id", m.Chain(api.GetOneBook))
	router.PUT("/v1/books/:id", m.Chain(api.UpdateBook))
	router.DELETE("/v1/books/:id", m.Chain(api.DeleteOneBook))
	return router
}
