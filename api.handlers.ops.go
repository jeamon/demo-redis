package main

import (
	"net/http"
	"net/http/pprof"

	"github.com/julienschmidt/httprouter"
)

func (api *APIHandler) OpsHandlerWrapper(h http.Handler) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		h.ServeHTTP(w, r)
	}
}

func (api *APIHandler) GetProfilerIndexPage(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	pprof.Index(w, r)
}

func (api *APIHandler) GetCPUProfile(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	pprof.Profile(w, r)
}

func (api *APIHandler) GetTraceProfile(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	pprof.Trace(w, r)
}

func (api *APIHandler) GetSymbol(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	pprof.Symbol(w, r)
}

func (api *APIHandler) GetCmdLine(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	pprof.Cmdline(w, r)
}
