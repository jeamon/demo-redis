package main

import (
	"github.com/julienschmidt/httprouter"
)

// SetupBookRoutes injects book related the api endpoints.
func (api *APIHandler) SetupBookRoutes(router *httprouter.Router, m *MiddlewareMap) *httprouter.Router {
	router.RedirectTrailingSlash = true
	router.GET("/", m.public(api.Index))
	router.GET("/status", m.public(api.Status))
	router.POST("/v1/books", m.public(api.CreateBook))
	router.GET("/v1/books", m.public(api.GetAllBooks))
	router.GET("/v1/books/:id", m.public(api.GetOneBook))
	router.PUT("/v1/books/:id", m.public(api.UpdateBook))
	router.DELETE("/v1/books/:id", m.public(api.DeleteOneBook))
	return router
}
