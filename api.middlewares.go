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

// CoreMiddleware setup the duration measurement for each request and logs its result.
func (api *APIHandler) CoreMiddleware(next httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		start := time.Now()
		requestID := GetValueFromContext(r.Context(), ContextRequestID)

		api.logger.Info(
			"request",
			zap.String("request.id", requestID),
			zap.String("request.method", r.Method),
			zap.String("request.path", r.URL.Path),
			zap.String("request.ip", GetRequestSourceIP(r)),
			zap.String("request.agent", r.UserAgent()),
			zap.String("request.referer", r.Referer()),
		)

		next(w, r, ps)
		api.logger.Info(
			"request",
			zap.String("request.id", requestID),
			zap.String("request.method", r.Method),
			zap.String("request.path", r.URL.Path),
			zap.Duration("request.duration", time.Since(start)),
		)
	}
}

// RequestsCounterMiddleware increments the number of received requests statistics and add this
// new value to the request context to be used during logging as `request.num` field.
func (api *APIHandler) RequestsCounterMiddleware(next httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := context.WithValue(r.Context(), ContextRequestNumber, atomic.AddUint64(&api.stats.called, 1))
		r = r.WithContext(ctx)
		next(w, r, ps)
	}
}

// RequestIDMiddleware generates and add a unique id to the request context.
func (api *APIHandler) RequestIDMiddleware(next httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		requestID := GenerateID(RequestIDPrefix)
		ctx := context.WithValue(r.Context(), ContextRequestID, requestID)
		r = r.WithContext(ctx)
		next(w, r, ps)
	}
}

// CORSMiddleware intercepts each incoming HTTP calls then apply cors headers on it.
func CORSMiddleware(next httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, UPDATE, PATCH, HEAD")
		w.Header().Set("Access-Control-Allow-Headers", "Origin, Access-Control-Request-Method, Access-Control-Request-Headers, Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, User-Agent, Accept-Language, Referer, DNT, Connection, Pragma, Cache-Control, TE")
		next(w, r, ps)
	}
}

// PanicRecoveryMiddleware catches any panic during the request lifecycle and produces
// an error log for further analysis. It sends a failure response to the client with 500.
func (api *APIHandler) PanicRecoveryMiddleware(next httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		recovery := func() {
			if err := recover(); err != nil {
				requestID := GetValueFromContext(r.Context(), ContextRequestID)
				api.logger.Error("panic occurred", zap.String("request.id", requestID), zap.Any("error", err))
				errResp := NewAPIError(requestID, http.StatusInternalServerError, "failed to process the request.", EmptyData)
				if err := WriteErrorResponse(w, errResp); err != nil {
					api.logger.Error("failed to send error response", zap.String("request.id", requestID), zap.Error(err))
				}
			}
		}
		defer recovery()
		next(w, r, ps)
	}
}

// Chain wraps a given httprouter.Handle with a list of middlewares.
// It does by starting from the last middleware from the list.
func (m *Middlewares) Chain(h httprouter.Handle) httprouter.Handle {
	if len(*m) == 0 {
		return h
	}
	lg := len(*m)
	handle := (*m)[lg-1](h)

	for i := lg - 2; i >= 0; i-- {
		handle = (*m)[i](handle)
	}

	return handle
}
