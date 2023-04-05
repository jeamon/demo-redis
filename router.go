package main

import "github.com/julienschmidt/httprouter"

// SetupRoutes enforces the api routes.
func (api *APIHandler) SetupRoutes(router *httprouter.Router) *httprouter.Router {
	router.RedirectTrailingSlash = true
	router.GET("/", api.middleware(api.Index))
	router.GET("/status", api.middleware(api.Index))
	// router.GET("/ops/stats", api.middleware(api.GetStats))
	router.GET("/ops/configs", api.middleware(api.GetConfigs))
	router.POST("/v1/books", api.middleware(api.CreateBook))
	router.GET("/v1/books", api.middleware(api.GetAllBooks))
	router.GET("/v1/books/:id", api.middleware(api.GetOneBook))
	router.PUT("/v1/books/:id", api.middleware(api.UpdateBook))
	router.DELETE("/v1/books/:id", api.middleware(api.DeleteOneBook))
	return router
}
