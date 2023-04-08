package main

import (
	"context"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/julienschmidt/httprouter"
	"go.uber.org/zap"
)

// MiddlewareFunc is a custom type for ease of use.
type MiddlewareFunc func(httprouter.Handle) httprouter.Handle

// Middlewares is a custom type to represent a stack of
// middleware functions used to build a single chain.
type Middlewares []MiddlewareFunc

// CoreMiddleware is used to intercept incoming HTTP calls then attach a unique id to each request
// and log each request details and finally apply cors headers on it.
func (api *APIHandler) CoreMiddleware(next httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		atomic.AddUint64(&api.stats.called, 1)
		start := time.Now()
		requestID := GenerateRequestID()
		ctx, cancel := context.WithTimeout(r.Context(), time.Duration(api.config.Server.RequestTimeout)*time.Second)
		defer cancel()
		ctx = context.WithValue(ctx, ContextRequestID, requestID)
		r = r.WithContext(ctx)
		api.logger.Info(
			"request",
			zap.String("requestid", requestID),
			zap.String("method", r.Method),
			zap.String("url", r.URL.Path),
			zap.String("ip", GetRequestSourceIP(r)),
			zap.String("agent", r.UserAgent()),
			zap.String("referer", r.Referer()),
		)
		setupCORS(&w)
		next(w, r, ps)
		api.logger.Info(
			"request",
			zap.String("requestid", requestID),
			zap.String("method", r.Method),
			zap.String("url", r.URL.Path),
			zap.Duration("duration", time.Since(start)),
		)
	}
}

// PanicRecoveryMiddleware catches any panic during the request lifecycle and produces
// an error log for further analysis. It sends a failure response to the client with 500.
func (api *APIHandler) PanicRecoveryMiddleware(next httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		recovery := func() {
			if err := recover(); err != nil {
				requestID := r.Context().Value(ContextRequestID).(string)
				api.logger.Error("panic occurred", zap.String("requestid", requestID), zap.Any("error", err))
				errResp := NewAPIError(requestID, http.StatusInternalServerError, "failed to process the request.", EmptyData)
				if err := WriteErrorResponse(w, errResp); err != nil {
					api.logger.Error("failed to send error response", zap.String("requestid", requestID), zap.Error(err))
				}
			}
		}
		defer recovery()
		next(w, r, ps)
	}
}

// Chain wraps a given httprouter.Handle with a list of middlewares.
// It does by starting from the last middleware from the list.
func (m Middlewares) Chain(h httprouter.Handle) httprouter.Handle {
	if len(m) == 0 {
		return h
	}
	lg := len(m)
	handle := m[lg-1](h)

	for i := lg - 2; i >= 0; i-- {
		handle = m[i](handle)
	}

	return handle
}
