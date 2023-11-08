package main

import (
	"encoding/json"
	"net/http"
	"sync"

	"go.uber.org/zap"
)

// APIHandler defines the API handler.
type APIHandler struct {
	logger      *zap.Logger
	config      *Config
	stats       *Statistics
	mode        *Maintenance
	clock       Clocker
	idsHandler  UIDHandler
	bookService BookServiceProvider
}

// NewAPIHandler provides a new instance of APIHandler.
func NewAPIHandler(logger *zap.Logger, config *Config, stats *Statistics, ck Clocker, idsHandler UIDHandler, bs BookServiceProvider) *APIHandler {
	m := &Maintenance{}
	m.enabled.Store(false)
	stats.status = make(map[int]uint64)
	stats.mu = &sync.RWMutex{}
	return &APIHandler{logger: logger, config: config, stats: stats, mode: m, clock: ck, idsHandler: idsHandler, bookService: bs}
}

// NotFound is a custom handler used to serve inexistant requested routes.
func (api *APIHandler) NotFound() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := api.idsHandler.Generate(RequestIDPrefix)
		logger := api.logger.With(
			zap.String("request.id", requestID),
			zap.String("request.method", r.Method),
			zap.String("request.path", r.URL.Path),
			zap.String("request.ip", GetRequestSourceIP(r)),
			zap.String("request.agent", r.UserAgent()),
			zap.String("request.referer", r.Referer()),
		)
		logger.Warn("unknown route")
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusNotFound)
		if err := json.NewEncoder(w).Encode(
			map[string]string{
				"requestid": requestID,
				"message":   "route does not exist",
				"path":      r.Method + " " + r.URL.Path,
			},
		); err != nil {
			api.logger.Error("failed to send response", zap.String("request.id", requestID))
		}
	})
}
