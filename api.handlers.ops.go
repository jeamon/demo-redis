package main

import (
	"encoding/json"
	"expvar"
	"fmt"
	"net/http"
	"net/http/pprof"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/julienschmidt/httprouter"
	"go.uber.org/zap"
)

// export goroutines to be used by expvar handler.
var goroutines = expvar.NewInt("goroutines")

// Statistics holds app stats for ops.
type Statistics struct {
	version   string
	container bool
	runtime   string
	platform  string
	called    uint64
	started   time.Time
	status    map[int]uint64
	mu        *sync.RWMutex
}

// Maintenance holds app maintenance mode infos.
type Maintenance struct {
	enabled atomic.Bool
	reason  string
	started time.Time
}

// OpsHandlerWrapper takes an http.Handler function and provides httprouter.Handle.
func (api *APIHandler) OpsHandlerWrapper(h http.Handler) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		h.ServeHTTP(w, r)
	}
}

// GetMemStats returns memory statistics with number of goroutines in json.
func GetMemStats(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	goroutines.Set(int64(runtime.NumGoroutine()))
	expvar.Handler().ServeHTTP(w, r)
}

// RunGC forces the run of the garbage collector asynchronously.
func (api *APIHandler) RunGC(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	requestID := GetValueFromContext(r.Context(), ContextRequestID)
	go runtime.GC()
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(w).Encode(
		map[string]string{
			"called": "go runtime.GC()",
		},
	); err != nil {
		api.logger.Error("failed to send run gc response", zap.String("request.id", requestID), zap.Error(err))
	}
}

// FreeOSMemory forces the garbage collector to and tries to returns the memory
// back to the operating system in an asynchronous fashion.
func (api *APIHandler) FreeOSMemory(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	requestID := GetValueFromContext(r.Context(), ContextRequestID)
	go debug.FreeOSMemory()
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(w).Encode(
		map[string]string{
			"called": "go debug.FreeOSMemory()",
		},
	); err != nil {
		api.logger.Error("failed to send free os memory response", zap.String("request.id", requestID), zap.Error(err))
	}
}

// GetStatistics provides useful details about the application to the internal ops users.
// The stats returns by this handler do not contain the ops request which triggered that.
// That is why we remove 1 from the called field value in order to match the status stats.
func (api *APIHandler) GetStatistics(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	requestID := GetValueFromContext(r.Context(), ContextRequestID)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	api.stats.mu.RLock()
	maintenanceModeStartedTime := api.mode.started.String()
	if api.mode.started == (time.Time{}.UTC()) {
		maintenanceModeStartedTime = ""
	}
	err := json.NewEncoder(w).Encode(
		map[string]interface{}{
			"requestid":     requestID,
			"app.version":   api.stats.version,
			"app.container": api.stats.container,
			"app.platform":  api.stats.platform,
			"go.version":    api.stats.runtime,
			"called":        atomic.LoadUint64(&api.stats.called) - 1,
			"started":       api.stats.started.Format(time.RFC1123),
			"uptime":        fmt.Sprintf("%.0f mins", api.clock.Now().Sub(api.stats.started).Minutes()),
			"maintenance": map[string]interface{}{
				"enabled": api.mode.enabled.Load(),
				"started": maintenanceModeStartedTime,
				"reason":  api.mode.reason,
			},
			"status": api.stats.status,
		},
	)
	api.stats.mu.RUnlock()
	if err != nil {
		api.logger.Error("failed to send statistics response", zap.String("request.id", requestID), zap.Error(err))
	}
}

// GetConfigs serves current in-use configurations/settings.
func (api *APIHandler) GetConfigs(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	requestID := GetValueFromContext(r.Context(), ContextRequestID)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(w).Encode(
		map[string]interface{}{
			"configs": api.config,
		},
	); err != nil {
		api.logger.Error("failed to send settings response", zap.String("request.id", requestID), zap.Error(err))
	}
}

// Maintenance handles request to enable or disable the maintenance mode of the service and respond
// to client requests with predefined message when the service is in maintenance mode.
// Enable the maintenance mode : /ops/maintenance?status=enable&msg=message-to-be-displayed-to-users
// Disable the maintenance mode: /ops/maintenance?status=disable
func (api *APIHandler) Maintenance(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	requestID := GetValueFromContext(r.Context(), ContextRequestID)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	var response map[string]interface{}
	var logger *zap.Logger

	q := r.URL.Query()
	mstatus := "show"
	if ps.ByName("status") != mstatus {
		mstatus = q.Get("status")
	}

	switch mstatus {
	case "enable":
		api.mode.reason = q.Get("msg")
		api.mode.started = api.clock.Now().UTC()
		api.mode.enabled.Store(true)
		response = map[string]interface{}{
			"requestid":           requestID,
			"maintenance.started": api.mode.started.Format(time.RFC1123),
			"maintenance.reason":  api.mode.reason,
			"message":             "Maintenance mode enabled successfully.",
		}
		logger = api.logger.With(zap.String("request.id", requestID))

	case "disable":
		api.mode.enabled.Store(false)
		api.mode.started = time.Time{}.UTC()
		api.mode.reason = ""
		response = map[string]interface{}{
			"requestid": requestID,
			"message":   "Maintenance mode disabled successfully.",
		}
		logger = api.logger.With(zap.String("request.id", requestID))

	case "show":
		response = map[string]interface{}{
			"requestid": requestID,
			"message":   "service currently unvailable.",
			"reason":    api.mode.reason,
			"since":     api.mode.started.Format(time.RFC1123),
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Error("failed to send maintenance response",
			zap.String("request.maintenance", mstatus),
			zap.Error(err),
		)
	}
}

// GetProfilerIndexPage displays pprof index page.
// func (api *APIHandler) GetProfilerIndexPage(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
//	pprof.Index(w, r)
// }

// GetCPUProfile returns a snapshot of the pprof-formatted CPU profile.
func (api *APIHandler) GetCPUProfile(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	pprof.Profile(w, r)
}

// GetTraceProfile returns the execution trace.
func (api *APIHandler) GetTraceProfile(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	pprof.Trace(w, r)
}

// GetSymbol returns the program symbol from the pprof package.
func (api *APIHandler) GetSymbol(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	pprof.Symbol(w, r)
}

// GetCmdLine returns the program command lines arguments.
func (api *APIHandler) GetCmdLine(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	pprof.Cmdline(w, r)
}
