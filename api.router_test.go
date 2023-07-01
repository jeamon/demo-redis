package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
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
	bs := NewBookService(zap.NewNop(), nil, mockRepo)
	api := NewAPIHandler(zap.NewNop(), nil, &Statistics{started: time.Now()}, bs)
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
	}

	config := &Config{ProfilerEndpointsEnable: false}
	bs := NewBookService(zap.NewNop(), config, nil)
	api := NewAPIHandler(zap.NewNop(), config, &Statistics{started: time.Now()}, bs)
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
