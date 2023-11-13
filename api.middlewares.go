package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

// MiddlewareMap contains middlwares chain to
// use for public-facing and ops requests.
type MiddlewareMap struct {
	public MiddlewareFunc
	ops    MiddlewareFunc
}

// StatsMiddleware is a middleware that logs the duration it takes to handle each request,
// then update the number of http status codes returned for internal ops statistics purposes.
func (api *APIHandler) StatsMiddleware(next httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		logger := api.GetLoggerFromContext(r.Context())
		conn := GetConnFromContext(r.Context())
		nw := NewCustomResponseWriter(w, conn)
		start := api.clock.Now()
		next(nw, r, ps)
		logger.Info(
			"stats",
			zap.Int("request.status", nw.Status()),
			zap.Int("bytes.sent", nw.Bytes()),
			zap.Duration("request.duration", api.clock.Now().Sub(start)),
		)
		api.stats.mu.Lock()
		if num, found := api.stats.status[nw.code]; !found {
			api.stats.status[nw.code] = 1
		} else {
			api.stats.status[nw.code] = num + 1
		}
		api.stats.mu.Unlock()
	}
}

// AddLoggerMiddleware creates a logger with pre-populated fields for each request.
func (api *APIHandler) AddLoggerMiddleware(next httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		requestID := GetValueFromContext(r.Context(), ContextRequestID)
		requestNum := GetRequestNumberFromContext(r.Context())
		logger := api.logger.With(
			zap.String("request.id", requestID),
			zap.Uint64("request.number", requestNum),
			zap.String("request.method", r.Method),
			zap.String("request.path", r.URL.Path),
			// zap.String("request.ip", GetRequestSourceIP(r)),
			zap.String("request.agent", r.UserAgent()),
			zap.String("request.referer", r.Referer()),
		)

		ctx := context.WithValue(r.Context(), ContextRequestLogger, logger)
		r = r.WithContext(ctx)
		next(w, r, ps)
	}
}

// MaintenanceModeMiddleware responds to client with maintenance message along with 503 code
// when the app field `Mode.enabled` is set to true. Otherwise it forwards the request.
func (api *APIHandler) MaintenanceModeMiddleware(next httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		if api.mode.enabled.Load() {
			api.Maintenance(w, r, httprouter.Params{
				httprouter.Param{
					Key:   "status",
					Value: "show",
				},
			})
			return
		}
		next(w, r, ps)
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
		requestID := api.idsHandler.Generate(RequestIDPrefix)
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
				errResp := NewAPIError(requestID, http.StatusInternalServerError, "failed to process the request.", struct{}{})
				if err := WriteErrorResponse(r.Context(), w, errResp); err != nil {
					api.logger.Error("failed to send error response", zap.String("request.id", requestID), zap.Error(err))
				}
			}
		}
		defer recovery()
		next(w, r, ps)
	}
}

// TimeoutMiddleware returns a Handler which sets X-Timeout-Reached header to instruct the final handler to not
// respond to client because timeout response was already sent. Similarly it sets X-Request-Cancelled into the
// header to notify the final handler to not perform any action towards the client.
func (api *APIHandler) TimeoutMiddleware(next httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		requestID := GetValueFromContext(r.Context(), ContextRequestID)
		logger := api.GetLoggerFromContext(r.Context())
		timeout := api.GetTimeout(r)
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()
		r = r.WithContext(ctx)
		done := make(chan struct{})
		go func() {
			next(w, r, ps)
			close(done)
		}()

		select {
		case <-done:
		case <-ctx.Done():
			if cerr := ctx.Err(); errors.Is(cerr, context.Canceled) {
				w.Header().Set("X-DRAP-ABORTED", "C")
				w.WriteHeader(499)
			} else if errors.Is(cerr, context.DeadlineExceeded) {
				w.Header().Set("X-DRAP-ABORTED", "T")
				w.Header().Set("Content-Type", "application/json; charset=UTF-8")
				w.WriteHeader(http.StatusGatewayTimeout)
				if err := json.NewEncoder(w).Encode(map[string]interface{}{
					"requestid": requestID,
					"message":   "request handling timed out",
					"timeout":   fmt.Sprintf("%.0f secs", timeout.Seconds()),
				}); err != nil {
					logger.Error("failed to send timeout response", zap.String("request.id", requestID), zap.Error(err))
				}
			}
		}
	}
}

// GetTimeout returns the processing timeout to use to update
// a given request context deadline based on path and method.
func (api *APIHandler) GetTimeout(r *http.Request) time.Duration {
	switch {
	case r.Method == "GET" && r.URL.Path == "/v1/books":
		return api.config.Server.LongRequestProcessingTimeout
	default:
		return api.config.Server.RequestTimeout
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

// MiddlewaresStacks builds the map of middlewares stack.
func (api *APIHandler) MiddlewaresStacks() (*Middlewares, *Middlewares) {
	middlewaresPublic := Middlewares{
		api.PanicRecoveryMiddleware,
		api.RequestIDMiddleware,
		api.MaintenanceModeMiddleware,
		api.RequestsCounterMiddleware,
		api.AddLoggerMiddleware,
		CORSMiddleware,
		api.TimeoutMiddleware,
		api.StatsMiddleware,
	}

	middlewaresOps := Middlewares{
		api.PanicRecoveryMiddleware,
		api.RequestIDMiddleware,
		api.RequestsCounterMiddleware,
		api.AddLoggerMiddleware,
		CORSMiddleware,
		api.TimeoutMiddleware,
		api.StatsMiddleware,
	}
	return &middlewaresPublic, &middlewaresOps
}
