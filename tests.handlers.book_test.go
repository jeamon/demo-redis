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
	"go.uber.org/zap"
)

// TestStatusHandler ensures api handler can provides its status.
func TestStatusHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	w := httptest.NewRecorder()
	api := NewAPIHandler(zap.NewNop(), nil, &Statistics{started: time.Now()}, nil)
	api.Status(w, req, httprouter.Params{})
	res := w.Result()
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "application/json; charset=UTF-8", res.Header.Get("Content-Type"))
	m := make(map[string]interface{})
	err = json.Unmarshal(data, &m)
	assert.NoError(t, err)

	_, ok := m["requestid"]
	assert.True(t, ok)

	v, ok := m["status"]
	assert.True(t, ok)
	assert.Equal(t, "up & running since 0 mins", v)

	v, ok = m["message"]
	assert.True(t, ok)
	assert.Equal(t, v, "Hello. Books store api is available. Enjoy :)")
}

// TestCreateBookHandler ensures api handler can create a book.
//
//nolint:funlen
func TestCreateBookHandler(t *testing.T) {
	mockRepo := &MockBookStorage{
		AddFunc: func(ctx context.Context, id string, book Book) error {
			return nil
		},
	}
	bs := NewBookService(zap.NewNop(), nil, mockRepo)
	api := NewAPIHandler(zap.NewNop(), nil, &Statistics{started: time.Now()}, bs)

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
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, res.StatusCode)
		assert.Equal(t, "application/json; charset=UTF-8", res.Header.Get("Content-Type"))

		resultMap := make(map[string]interface{})
		err = json.Unmarshal(data, &resultMap)
		assert.NoError(t, err)

		_, ok := resultMap["requestid"]
		assert.True(t, ok)

		v, ok := resultMap["status"]
		assert.True(t, ok)
		assert.Equal(t, float64(http.StatusCreated), v)

		v, ok = resultMap["message"]
		assert.True(t, ok)
		assert.Equal(t, "Book created successfully.", v)

		v, ok = resultMap["data"]
		assert.True(t, ok)

		bookMap, ok := v.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "Test book title", bookMap["title"])
		assert.Equal(t, "Test book description", bookMap["description"])
		assert.Equal(t, "Jerome Amon", bookMap["author"])
		assert.Equal(t, "10$", bookMap["price"])

		assert.NotEmpty(t, bookMap["createdAt"])
		assert.NotEmpty(t, bookMap["updatedAt"])
	})

	t.Run("should fail: storage insertion failure", func(t *testing.T) {
		mockRepo := &MockBookStorage{
			AddFunc: func(ctx context.Context, id string, book Book) error {
				return errors.New("storage failure")
			},
		}
		bs = NewBookService(zap.NewNop(), nil, mockRepo)
		api = NewAPIHandler(zap.NewNop(), nil, &Statistics{started: time.Now()}, bs)

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
		assert.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, res.StatusCode)
		assert.Equal(t, "application/json; charset=UTF-8", res.Header.Get("Content-Type"))

		resultMap := make(map[string]interface{})
		err = json.Unmarshal(data, &resultMap)
		assert.NoError(t, err)

		_, ok := resultMap["requestid"]
		assert.True(t, ok)

		v, ok := resultMap["status"]
		assert.True(t, ok)
		assert.Equal(t, float64(http.StatusInternalServerError), v)

		v, ok = resultMap["message"]
		assert.True(t, ok)
		assert.Equal(t, "failed to create the book", v)

		v, ok = resultMap["data"]
		assert.True(t, ok)
		bookMap, ok := v.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "Test book title", bookMap["title"])
		assert.Equal(t, "Test book description", bookMap["description"])
		assert.Equal(t, "Jerome Amon", bookMap["author"])
		assert.Equal(t, "10$", bookMap["price"])
		assert.NotEmpty(t, bookMap["createdAt"])
		assert.NotEmpty(t, bookMap["updatedAt"])
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
		assert.NoError(t, err)
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
				req := httptest.NewRequest(http.MethodPost, "/v1/books", bytes.NewBuffer(tc.payload))
				w := httptest.NewRecorder()
				api.CreateBook(w, req, httprouter.Params{})
				res := w.Result()
				defer res.Body.Close()
				assert.Equal(t, tc.status, res.StatusCode)
				assert.Equal(t, "application/json; charset=UTF-8", res.Header.Get("Content-Type"))
				data, err := io.ReadAll(res.Body)
				assert.NoError(t, err)
				assert.JSONEq(t, tc.expected, string(data))
			})
		}
	})
}

func TestDeleteOneBook_MissingBook(t *testing.T) {
	mockRepo := &MockBookStorage{
		GetOneFunc: func(ctx context.Context, id string) (Book, error) {
			return Book{}, ErrBookNotFound
		},
	}

	bs := NewBookService(zap.NewNop(), nil, mockRepo)
	api := NewAPIHandler(zap.NewNop(), nil, &Statistics{started: time.Now()}, bs)

	missingBookID := "b:cb8f2136-fae4-4200-85d9-3533c7f8c70d"
	req := httptest.NewRequest(http.MethodDelete, "/v1/books/"+missingBookID, nil)
	w := httptest.NewRecorder()
	api.DeleteOneBook(w, req, httprouter.Params{})
	res := w.Result()
	defer res.Body.Close()
	assert.Equal(t, http.StatusNotFound, res.StatusCode)
	assert.Equal(t, "application/json; charset=UTF-8", res.Header.Get("Content-Type"))
	data, err := io.ReadAll(res.Body)
	assert.NoError(t, err)
	expected := `{"requestid":"", "status":404, "message":"book does not exist",
		"data":{"id":"", "title":"", "description":"", "author":"", "price":"", "createdAt":"", "updatedAt":""}}`
	assert.JSONEq(t, expected, string(data))
}
