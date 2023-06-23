package main

import (
	"net/http"
	"net/http/pprof"

	"github.com/julienschmidt/httprouter"
)

// SetupOpsRoutes injects internal operations related endpoints.
func (api *APIHandler) SetupOpsRoutes(router *httprouter.Router, m *MiddlewareMap) *httprouter.Router {
	router.RedirectTrailingSlash = true
	router.GET("/ops/configs", m.ops(api.GetConfigs))
	router.GET("/ops/stats", m.ops(api.GetStatistics))
	router.GET("/ops/maintenance", m.ops(api.Maintenance))
	router.GET("/ops/debug/vars", m.ops(GetMemStats))
	router.GET("/ops/debug/gc", m.ops(api.RunGC))
	router.GET("/ops/debug/fos", m.ops(api.FreeOSMemory))

	if api.config.ProfilerEnable {
		router.GET("/ops/debug/pprof/", m.ops(api.OpsHandlerWrapper(http.HandlerFunc(pprof.Index))))
		router.GET("/ops/debug/pprof/profile", m.ops(api.GetCPUProfile))
		router.GET("/ops/debug/pprof/trace", m.ops(api.GetTraceProfile))
		router.GET("/ops/debug/pprof/symbol", m.ops(api.GetSymbol))
		router.GET("/ops/debug/pprof/cmdline", m.ops(api.GetCmdLine))
		router.GET("/ops/debug/pprof/heap", m.ops(api.OpsHandlerWrapper(pprof.Handler("heap"))))
		router.GET("/ops/debug/pprof/allocs", m.ops(api.OpsHandlerWrapper(pprof.Handler("allocs"))))
		router.GET("/ops/debug/pprof/goroutine", m.ops(api.OpsHandlerWrapper(pprof.Handler("goroutine"))))
		router.GET("/ops/debug/pprof/threadcreate", m.ops(api.OpsHandlerWrapper(pprof.Handler("threadcreate"))))
		router.GET("/ops/debug/pprof/block", m.ops(api.OpsHandlerWrapper(pprof.Handler("block"))))
		router.GET("/ops/debug/pprof/mutex", m.ops(api.OpsHandlerWrapper(pprof.Handler("mutex"))))
	}

	return router
}
