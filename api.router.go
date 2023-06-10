package main

import (
	"net/http"
	"net/http/pprof"

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

	router.GET("/ops/debug/vars", m.ops(GetMemStats))
	router.GET("/ops/debug/gc", m.ops(api.RunGC))
	router.GET("/ops/debug/fos", m.ops(api.FreeOSMemory))

	if api.config.ProfilerEnable {
		router.GET("/ops/debug/pprof/", m.ops(api.GetProfilerIndexPage))
		router.GET("/ops/debug/pprof/profile", m.ops(api.GetCPUProfile))
		router.GET("/ops/debug/pprof/trace", m.ops(api.GetTraceProfile))
		router.GET("/ops/debug/pprof/symbol", m.ops(api.GetSymbol))
		router.GET("/ops/debug/pprof/cmdline", m.ops(api.GetCmdLine))
		router.Handler(http.MethodGet, "/ops/debug/pprof/heap", pprof.Handler("heap"))
		router.Handler(http.MethodGet, "/ops/debug/pprof/allocs", pprof.Handler("allocs"))
		router.Handler(http.MethodGet, "/ops/debug/pprof/goroutine", pprof.Handler("goroutine"))
		router.Handler(http.MethodGet, "/ops/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
		router.Handler(http.MethodGet, "/ops/debug/pprof/block", pprof.Handler("block"))
		router.Handler(http.MethodGet, "/ops/debug/pprof/mutex", pprof.Handler("mutex"))
	}

	return router
}
