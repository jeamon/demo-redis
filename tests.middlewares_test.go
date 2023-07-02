package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

// TestMiddlewaresStacks ensures we get both public and ops middlewares
// stacks with exact number of elements in those stacks.
func TestMiddlewaresStacks(t *testing.T) {
	api := NewAPIHandler(zap.NewNop(), nil, &Statistics{started: NewMockClocker().Now()}, NewMockClocker(), nil)
	pub, ops := api.MiddlewaresStacks()
	assert.Equal(t, 7, len(*pub))
	assert.Equal(t, 6, len(*ops))
}

// TestChain ensures each middleware in the stack is called as well the handler.
func TestChain(t *testing.T) {
	var ca, cb, cc, ch bool
	queue := make(chan int, 4)

	middlewareA := func(next httprouter.Handle) httprouter.Handle {
		return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
			queue <- 1
			ca = true
			next(w, r, ps)
		}
	}
	middlewareB := func(next httprouter.Handle) httprouter.Handle {
		return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
			queue <- 2
			cb = true
			next(w, r, ps)
		}
	}
	middlewareC := func(next httprouter.Handle) httprouter.Handle {
		return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
			queue <- 3
			cc = true
			next(w, r, ps)
		}
	}
	middlewares := Middlewares{
		middlewareA,
		middlewareB,
		middlewareC,
	}

	handler := func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		queue <- 4
		ch = true
	}

	chained := (&middlewares).Chain(handler)
	req := httptest.NewRequest("GET", "/v1/books", nil)
	w := httptest.NewRecorder()
	chained(w, req, nil)

	t.Run("check calling", func(t *testing.T) {
		assert.Equal(t, true, ca)
		assert.Equal(t, true, cb)
		assert.Equal(t, true, cc)
		assert.Equal(t, true, ch)
	})

	t.Run("check ordering", func(t *testing.T) {
		assert.Equal(t, 1, <-queue)
		assert.Equal(t, 2, <-queue)
		assert.Equal(t, 3, <-queue)
		assert.Equal(t, 4, <-queue)
	})
}

// TestRequestsCounterMiddleware ensures the request counter increment.
func TestRequestsCounterMiddleware(t *testing.T) {
	api := NewAPIHandler(zap.NewNop(), nil, &Statistics{started: NewMockClocker().Now(), called: 0}, NewMockClocker(), nil)
	req := httptest.NewRequest("GET", "/v1/books", nil)
	w := httptest.NewRecorder()
	var called bool
	handler := func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
		called = true
	}
	wrapped := api.RequestsCounterMiddleware(handler)
	wrapped(w, req, nil)
	assert.Equal(t, true, called)
	assert.Equal(t, uint64(1), api.stats.called)
}

// TestMaintenanceModeMiddleware ensures users requests are handled according
// to maintenance mode current settings.
func TestMaintenanceModeMiddleware(t *testing.T) {
	t.Run("maintenance disabled", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/v1/books", nil)
		w := httptest.NewRecorder()
		var called bool
		handler := func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
			called = true
		}
		api := NewAPIHandler(zap.NewNop(), nil, &Statistics{started: NewMockClocker().Now()}, NewMockClocker(), nil)
		wrapped := api.MaintenanceModeMiddleware(handler)
		wrapped(w, req, nil)
		assert.Equal(t, true, called)
	})

	t.Run("maintenance enabled", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		var called bool
		handler := func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
			called = true
		}
		api := NewAPIHandler(zap.NewNop(), nil, &Statistics{started: NewMockClocker().Now()}, NewMockClocker(), nil)
		api.mode.enabled.Store(true)
		ts := NewMockClocker().Now()
		api.mode.started = ts
		api.mode.reason = "ongoing maintenance."
		wrapped := api.MaintenanceModeMiddleware(handler)
		wrapped(w, req, nil)
		// target handler will not be called but the maintenance handler must kick-in.
		assert.Equal(t, false, called)

		res := w.Result()
		defer res.Body.Close()
		data, err := io.ReadAll(res.Body)
		assert.NoError(t, err)
		expected := `{"message":"service currently unvailable.","reason":"ongoing maintenance.", "since":"Sun, 02 Jul 2023 00:00:00 UTC"}`
		assert.JSONEq(t, expected, string(data))
	})
}

// TestRequestIDMiddleware ensures a request id is added to request context.
func TestRequestIDMiddleware(t *testing.T) {
	api := NewAPIHandler(zap.NewNop(), nil, &Statistics{started: NewMockClocker().Now(), called: 0}, NewMockClocker(), nil)
	req := httptest.NewRequest("GET", "/v1/books", nil)
	w := httptest.NewRecorder()
	var called bool
	var id string
	handler := func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
		called = true
		id = GetValueFromContext(req.Context(), ContextRequestID)
	}
	wrapped := api.RequestIDMiddleware(handler)
	wrapped(w, req, nil)
	assert.Equal(t, true, called)
	assert.NotEmpty(t, id)
	assert.Contains(t, id, RequestIDPrefix+":")
}

// TestAddLoggerMiddleware ensures custom logger with exact fields is injected into the request context.
func TestAddLoggerMiddleware(t *testing.T) {
	observedZapCore, observedLogs := observer.New(zap.InfoLevel)
	observedLogger := zap.New(observedZapCore)
	api := NewAPIHandler(observedLogger, nil, &Statistics{started: NewMockClocker().Now(), called: 0}, NewMockClocker(), nil)
	req := httptest.NewRequest("GET", "/v1/books", nil)
	w := httptest.NewRecorder()
	var called bool
	var value interface{}
	handler := func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
		called = true
		value = req.Context().Value(ContextRequestLogger)
	}
	wrapped := api.AddLoggerMiddleware(handler)
	wrapped(w, req, nil)
	assert.Equal(t, true, called)
	logger, ok := value.(*zap.Logger)
	require.Equal(t, true, ok)
	// trigger a logging in order to have the log message
	// and the all fields values saved for later assertion.
	logger.Info("fake log")

	require.Equal(t, 1, observedLogs.Len())
	log := observedLogs.All()[0]
	assert.Equal(t, "fake log", log.Message)
	assert.ElementsMatch(t, []zap.Field{
		zap.String("request.id", ""),
		zap.Uint64("request.number", 0),
		zap.String("request.method", "GET"),
		zap.String("request.path", "/v1/books"),
		zap.String("request.ip", "192.0.2.1"),
		zap.String("request.agent", ""),
		zap.String("request.referer", ""),
	}, log.Context)
}
