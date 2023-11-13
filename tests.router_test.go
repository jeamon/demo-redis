package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestSetupBookRoutes ensures all expected book endpoints are implemented.
func TestSetupBookRoutes(t *testing.T) {
	testCases := []struct {
		name        string
		request     *http.Request
		implemented bool
	}{
		{
			"index endpoint",
			httptest.NewRequest(http.MethodGet, "/", nil),
			true,
		},
		{
			"status endpoint",
			httptest.NewRequest(http.MethodGet, "/status", nil),
			true,
		},
		{
			"create book endpoint",
			httptest.NewRequest(http.MethodPost, "/v1/books", nil),
			true,
		},
		{
			"fetch all books endpoint",
			httptest.NewRequest(http.MethodGet, "/v1/books", nil),
			true,
		},
		{
			"fetch all books endpoint with slash",
			httptest.NewRequest(http.MethodGet, "/v1/books/", nil),
			true,
		},
		{
			"fetch single book endpoint",
			httptest.NewRequest(http.MethodGet, "/v1/books/b:cb8f2136-fae4-4200-85d9-3533c7f8c70d", nil),
			true,
		},
		{
			"update book endpoint",
			httptest.NewRequest(http.MethodPut, "/v1/books/b:cb8f2136-fae4-4200-85d9-3533c7f8c70d", nil),
			true,
		},
		{
			"delete book endpoint",
			httptest.NewRequest(http.MethodDelete, "/v1/books/b:cb8f2136-fae4-4200-85d9-3533c7f8c70d", nil),
			true,
		},
		{
			"invalid api endpoint",
			httptest.NewRequest(http.MethodGet, "/v1", nil),
			false,
		},
		{
			"invalid books endpoint",
			httptest.NewRequest(http.MethodGet, "/books", nil),
			false,
		},
	}

	mockRepo := &MockBookStorage{
		AddFunc: func(ctx context.Context, id string, book Book) error {
			return nil
		},
		GetOneFunc: func(ctx context.Context, id string) (Book, error) {
			return Book{}, nil
		},
		DeleteFunc: func(ctx context.Context, id string) error {
			return nil
		},
		UpdateFunc: func(ctx context.Context, id string, book Book) (Book, error) {
			return Book{}, nil
		},
		GetAllFunc: func(ctx context.Context) ([]Book, error) {
			return []Book{}, nil
		},
	}
	mockQueue := &MockQueuer{
		PushFunc: func(ctx context.Context, qid string, book Book) error {
			return nil
		},
	}

	bs := NewBookService(zap.NewNop(), nil, NewMockClocker(), mockRepo, mockRepo, mockQueue)
	api := NewAPIHandler(zap.NewNop(), &Config{}, &Statistics{started: NewMockClocker().Now()}, NewMockClocker(), NewMockUIDHandler("", true), bs)
	api.config.Server.LongRequestWriteTimeout = time.Second
	router := httprouter.New()
	m := &MiddlewareMap{public: (&Middlewares{}).Chain, ops: (&Middlewares{}).Chain}
	api.SetupBookRoutes(router, m)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, tc.request)
			if tc.implemented {
				assert.NotEqual(t, 404, w.Code)
			} else {
				assert.Equal(t, 404, w.Code)
			}
		})
	}
}

// TestSetupOpsRoutes ensures all expected operations endpoints are implemented.
func TestSetupOpsRoutes(t *testing.T) {
	testCases := []struct {
		name        string
		request     *http.Request
		implemented bool
	}{
		{
			"fetch configs endpoint",
			httptest.NewRequest(http.MethodGet, "/ops/configs", nil),
			true,
		},
		{
			"ops endpoint with slash",
			httptest.NewRequest(http.MethodGet, "/ops/configs", nil),
			true,
		},
		{
			"fetch stats endpoint",
			httptest.NewRequest(http.MethodGet, "/ops/stats", nil),
			true,
		},
		{
			"maintenance mode endpoint",
			httptest.NewRequest(http.MethodGet, "/ops/maintenance", nil),
			true,
		},
		{
			"invalid ops endpoint",
			httptest.NewRequest(http.MethodGet, "/ops", nil),
			false,
		},
		{
			"unknown ops endpoint",
			httptest.NewRequest(http.MethodGet, "/ops/unknown", nil),
			false,
		},
		{
			"disabled profiler endpoint",
			httptest.NewRequest(http.MethodGet, "/ops/debug/pprof/", nil),
			false,
		},
	}

	config := &Config{ProfilerEndpointsEnable: false}
	bs := NewBookService(zap.NewNop(), config, NewMockClocker(), nil, nil, nil)
	api := NewAPIHandler(zap.NewNop(), config, &Statistics{started: NewMockClocker().Now()}, NewMockClocker(), nil, bs)
	router := httprouter.New()
	m := &MiddlewareMap{public: (&Middlewares{}).Chain, ops: (&Middlewares{}).Chain}
	api.SetupOpsRoutes(router, m)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, tc.request)
			if tc.implemented {
				assert.NotEqual(t, 404, w.Code)
			} else {
				assert.Equal(t, 404, w.Code)
			}
		})
	}
}

// TestSetupRoutes ensures all expected endpoints are implemented.
func TestSetupRoutes(t *testing.T) {
	testCases := []struct {
		name               string
		OpsEndpointsEnable bool
		request            *http.Request
		implemented        bool
	}{
		{
			"ops disable:fetch configs endpoint",
			false,
			httptest.NewRequest(http.MethodGet, "/ops/configs", nil),
			false,
		},
		{
			"ops enable:fetch configs endpoint",
			true,
			httptest.NewRequest(http.MethodGet, "/ops/configs", nil),
			true,
		},
		{
			"ops disable:disabled profiler endpoint",
			false,
			httptest.NewRequest(http.MethodGet, "/ops/debug/pprof/", nil),
			false,
		},
		{
			"ops enable:disabled profiler endpoint",
			true,
			httptest.NewRequest(http.MethodGet, "/ops/debug/pprof/", nil),
			false,
		},
		{
			"ops disable:create book endpoint",
			false,
			httptest.NewRequest(http.MethodPost, "/v1/books", nil),
			true,
		},
		{
			"ops enable:create book endpoint",
			true,
			httptest.NewRequest(http.MethodPost, "/v1/books", nil),
			true,
		},
		{
			"invalid ops endpoint",
			false,
			httptest.NewRequest(http.MethodGet, "/ops/", nil),
			false,
		},
		{
			"invalid book endpoint",
			false,
			httptest.NewRequest(http.MethodGet, "/books/", nil),
			false,
		},
	}

	config := &Config{OpsEndpointsEnable: false, ProfilerEndpointsEnable: false}
	bs := NewBookService(zap.NewNop(), config, NewMockClocker(), nil, nil, nil)
	api := NewAPIHandler(zap.NewNop(), config, &Statistics{started: NewMockClocker().Now()}, NewMockClocker(), NewMockUIDHandler("abc", true), bs)
	m := &MiddlewareMap{public: (&Middlewares{}).Chain, ops: (&Middlewares{}).Chain}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			router := httprouter.New()
			if tc.OpsEndpointsEnable {
				config.OpsEndpointsEnable = true
				api.SetupRoutes(router, m)
			} else {
				config.OpsEndpointsEnable = false
				api.SetupRoutes(router, m)
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, tc.request)
			if tc.implemented {
				assert.NotEqual(t, 404, w.Code)
			} else {
				assert.Equal(t, 404, w.Code)
			}
		})
	}
}

// TestSetupRoutes_NotFound ensures exact status code and json response body when a user requests an inexistant route.
func TestSetupRoutes_NotFound(t *testing.T) {
	m := &MiddlewareMap{public: (&Middlewares{}).Chain, ops: (&Middlewares{}).Chain}
	api := NewAPIHandler(zap.NewNop(), &Config{}, &Statistics{started: NewMockClocker().Now()}, NewMockClocker(), NewMockUIDHandler("abc", true), nil)
	router := httprouter.New()
	api.SetupRoutes(router, m)
	r := httptest.NewRequest(http.MethodGet, "/x/books/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	res := w.Result()
	defer res.Body.Close()
	assert.Equal(t, http.StatusNotFound, res.StatusCode)
	assert.Equal(t, "application/json; charset=UTF-8", res.Header.Get("Content-Type"))
	data, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	expected := `{"requestid":"r:abc", "message":"route does not exist", "path":"GET /x/books/"}`
	assert.JSONEq(t, expected, string(data))
}
