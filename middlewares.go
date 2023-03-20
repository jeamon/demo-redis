package main

import (
	"context"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/julienschmidt/httprouter"
	"go.uber.org/zap"
)

// middleware is used to intercept incoming HTTP calls then attach a unique id to each request
// and log each request details and finally apply cors headers on it.
func (api *APIHandler) middleware(next httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		atomic.AddUint64(&api.stats.called, 1)
		start := time.Now()
		requestID := GenerateRequestID()
		defer func() {
			if err := recover(); err != nil {
				api.logger.Error("panic occurred", zap.String("requestid", requestID), zap.Any("error", err))
				errResp := NewAPIError(requestID, http.StatusInternalServerError, "failed to process the request.", EmptyData)
				if err := WriteErrorResponse(w, errResp); err != nil {
					api.logger.Error("failed to send error response", zap.String("requestid", requestID), zap.Error(err))
				}
			}
		}()

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
