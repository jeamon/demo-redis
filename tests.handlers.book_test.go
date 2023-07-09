package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

// TestStatusHandler ensures api handler can provides its status.
func TestStatusHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	w := httptest.NewRecorder()
	api := NewAPIHandler(zap.NewNop(), nil, &Statistics{started: NewMockClocker().Now()}, NewMockClocker(), nil, nil)
	api.Status(w, req, httprouter.Params{})
	res := w.Result()
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "application/json; charset=UTF-8", res.Header.Get("Content-Type"))
	m := make(map[string]interface{})
	err = json.Unmarshal(data, &m)
	require.NoError(t, err)

	_, ok := m["requestid"]
	require.True(t, ok)

	v, ok := m["status"]
	require.True(t, ok)
	assert.Equal(t, "up & running since 0 mins", v)

	v, ok = m["message"]
	require.True(t, ok)
	assert.Equal(t, v, "Hello. Books store api is available. Enjoy :)")
}

// TestCreateBookHandler ensures api handler can create a book.
func TestCreateBookHandler(t *testing.T) {
	mockRepo := &MockBookStorage{
		AddFunc: func(ctx context.Context, id string, book Book) error {
			return nil
		},
	}
	bs := NewBookService(zap.NewNop(), nil, NewMockClocker(), mockRepo)
	api := NewAPIHandler(zap.NewNop(), nil, &Statistics{started: NewMockClocker().Now()}, NewMockClocker(), NewMockUIDHandler("abc", true), bs)

	t.Run("should pass: valid payload", func(t *testing.T) {
		book := Book{
			Title:       "Test book title",
			Description: "Test book description",
			Author:      "Jerome Amon",
			Price:       "10$",
		}
		payload, err := json.Marshal(book)
		assert.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, "/v1/books", bytes.NewBuffer(payload))
		w := httptest.NewRecorder()
		api.CreateBook(w, req, httprouter.Params{})
		res := w.Result()
		defer res.Body.Close()
		data, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, res.StatusCode)
		assert.Equal(t, "application/json; charset=UTF-8", res.Header.Get("Content-Type"))

		resultMap := make(map[string]interface{})
		err = json.Unmarshal(data, &resultMap)
		require.NoError(t, err)

		_, ok := resultMap["requestid"]
		assert.True(t, ok)

		v, ok := resultMap["status"]
		require.True(t, ok)
		assert.Equal(t, float64(http.StatusCreated), v)

		v, ok = resultMap["message"]
		require.True(t, ok)
		assert.Equal(t, "Book created successfully.", v)

		v, ok = resultMap["data"]
		assert.True(t, ok)

		bookMap, ok := v.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "b:abc", bookMap["id"])
		assert.Equal(t, "Test book title", bookMap["title"])
		assert.Equal(t, "Test book description", bookMap["description"])
		assert.Equal(t, "Jerome Amon", bookMap["author"])
		assert.Equal(t, "10$", bookMap["price"])
		assert.Equal(t, "2023-07-02 00:00:00 +0000 UTC", bookMap["createdAt"])
		assert.Equal(t, "2023-07-02 00:00:00 +0000 UTC", bookMap["updatedAt"])
	})

	t.Run("should fail: storage insertion failure", func(t *testing.T) {
		mockRepo := &MockBookStorage{
			AddFunc: func(ctx context.Context, id string, book Book) error {
				return errors.New("storage failure")
			},
		}
		observedZapCore, observedLogs := observer.New(zap.ErrorLevel)
		observedLogger := zap.New(observedZapCore)
		bs = NewBookService(zap.NewNop(), nil, NewMockClocker(), mockRepo)
		api = NewAPIHandler(observedLogger, nil, &Statistics{started: NewMockClocker().Now()}, NewMockClocker(), NewMockUIDHandler("", false), bs)

		payload := `{"title":"Test book title", "description":"Test book description", "author":"Jerome Amon", "price":"10$"}`
		req := httptest.NewRequest(http.MethodPost, "/v1/books", bytes.NewBuffer([]byte(payload)))
		w := httptest.NewRecorder()
		api.CreateBook(w, req, httprouter.Params{})

		require.Equal(t, 1, observedLogs.Len())
		log := observedLogs.All()[0]
		assert.Equal(t, "failed to create book", log.Message)
		assert.ElementsMatch(t, []zap.Field{
			zap.String("request.id", ""),
			zap.Error(errors.New("storage failure")),
		}, log.Context)

		res := w.Result()
		defer res.Body.Close()
		assert.Equal(t, http.StatusInternalServerError, res.StatusCode)
		assert.Equal(t, "application/json; charset=UTF-8", res.Header.Get("Content-Type"))
		data, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		assert.Contains(t, string(data), `{"requestid":"","status":500,"message":"failed to create the book"`)
		resultMap := make(map[string]interface{})
		err = json.Unmarshal(data, &resultMap)
		require.NoError(t, err)
		v, ok := resultMap["data"]
		require.True(t, ok)
		bookMap, ok := v.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "Test book title", bookMap["title"])
		assert.Equal(t, "Test book description", bookMap["description"])
		assert.Equal(t, "Jerome Amon", bookMap["author"])
		assert.Equal(t, "10$", bookMap["price"])
		assert.Equal(t, "2023-07-02 00:00:00 +0000 UTC", bookMap["createdAt"])
		assert.Equal(t, "2023-07-02 00:00:00 +0000 UTC", bookMap["updatedAt"])
	})

	t.Run("should fail: invalid payload", func(t *testing.T) {
		jsonStringPayload := `{"title":1, "description":"Test book description", "author":"Jerome Amon", "price":"10$"}`
		req := httptest.NewRequest(http.MethodPost, "/v1/books", bytes.NewBuffer([]byte(jsonStringPayload)))
		w := httptest.NewRecorder()
		api.CreateBook(w, req, httprouter.Params{})
		res := w.Result()
		defer res.Body.Close()
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		assert.Equal(t, "application/json; charset=UTF-8", res.Header.Get("Content-Type"))
		data, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		expected := `{"requestid":"", "status":400, "message":"failed to create the book",
		"data":{"id":"", "title":"", "description":"Test book description", "author":"Jerome Amon", "price":"10$", "createdAt":"", "updatedAt":""}}`
		assert.JSONEq(t, expected, string(data))
	})

	t.Run("should fail: required field in payload", func(t *testing.T) {
		testCases := []struct {
			name     string
			payload  []byte
			status   int
			expected string
		}{
			{
				name:     "empty",
				payload:  []byte(`{"title":"", "description":"Test book description", "author":"Jerome Amon", "price":"10$"}`),
				status:   http.StatusBadRequest,
				expected: `{"requestid":"", "status":400, "message":"failed to create the book", "data":"title is required"}`,
			},
			{
				name:     "missing",
				payload:  []byte(`{"description":"Test book description", "author":"Jerome Amon", "price":"10$"}`),
				status:   http.StatusBadRequest,
				expected: `{"requestid":"", "status":400, "message":"failed to create the book", "data":"title is required"}`,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				observedZapCore, observedLogs := observer.New(zap.ErrorLevel)
				observedLogger := zap.New(observedZapCore)
				api = NewAPIHandler(observedLogger, nil, &Statistics{started: NewMockClocker().Now()}, NewMockClocker(), nil, bs)
				req := httptest.NewRequest(http.MethodPost, "/v1/books", bytes.NewBuffer(tc.payload))
				w := httptest.NewRecorder()
				api.CreateBook(w, req, httprouter.Params{})

				require.Equal(t, 1, observedLogs.Len())
				log := observedLogs.All()[0]
				assert.Equal(t, "failed to create book", log.Message)
				assert.ElementsMatch(t, []zap.Field{
					zap.String("request.id", ""),
					zap.Error(missingFieldError("title")),
				}, log.Context)

				res := w.Result()
				defer res.Body.Close()
				assert.Equal(t, tc.status, res.StatusCode)
				assert.Equal(t, "application/json; charset=UTF-8", res.Header.Get("Content-Type"))
				data, err := io.ReadAll(res.Body)
				require.NoError(t, err)
				assert.JSONEq(t, tc.expected, string(data))
			})
		}
	})
}

func TestDeleteOneBook_MissingBook(t *testing.T) {
	helper := func(t *testing.T, repo BookStorage) *http.Response {
		t.Helper()
		bs := NewBookService(zap.NewNop(), nil, NewMockClocker(), repo)
		api := NewAPIHandler(zap.NewNop(), nil, &Statistics{started: time.Now()}, NewMockClocker(), NewMockUIDHandler("", true), bs)
		missingBookID := "b:cb8f2136-fae4-4200-85d9-3533c7f8c70d"
		req := httptest.NewRequest(http.MethodDelete, "/v1/books/"+missingBookID, nil)
		w := httptest.NewRecorder()
		api.DeleteOneBook(w, req, httprouter.Params{})
		return w.Result()
	}

	testCases := []struct {
		name string
		repo *MockBookStorage
	}{
		{
			"during checking",
			&MockBookStorage{
				GetOneFunc: func(ctx context.Context, id string) (Book, error) { return Book{}, ErrBookNotFound },
			},
		},
		{
			"during deletion",
			&MockBookStorage{
				GetOneFunc: func(ctx context.Context, id string) (Book, error) { return Book{}, nil },
				DeleteFunc: func(ctx context.Context, id string) error { return ErrBookNotFound },
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := helper(t, tc.repo)
			defer res.Body.Close()
			assert.Equal(t, http.StatusNotFound, res.StatusCode)
			assert.Equal(t, "application/json; charset=UTF-8", res.Header.Get("Content-Type"))
			data, err := io.ReadAll(res.Body)
			require.NoError(t, err)
			expected := `{"requestid":"", "status":404, "message":"book does not exist",
				"data":{"id":"", "title":"", "description":"", "author":"", "price":"", "createdAt":"", "updatedAt":""}}`
			assert.JSONEq(t, expected, string(data))
		})
	}
}
