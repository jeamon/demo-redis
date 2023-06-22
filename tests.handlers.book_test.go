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
		b := struct {
			Title       int
			Description string
			Author      string
			Price       string
		}{
			Title:       1,
			Description: "Test book description",
			Author:      "Jerome Amon",
			Price:       "10$",
		}
		payload, err := json.Marshal(b)
		assert.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, "/v1/books", bytes.NewBuffer(payload))
		w := httptest.NewRecorder()
		api.CreateBook(w, req, httprouter.Params{})
		res := w.Result()
		defer res.Body.Close()
		data, err := io.ReadAll(res.Body)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		assert.Equal(t, "application/json; charset=UTF-8", res.Header.Get("Content-Type"))

		resultMap := make(map[string]interface{})
		err = json.Unmarshal(data, &resultMap)
		assert.NoError(t, err)

		_, ok := resultMap["requestid"]
		assert.True(t, ok)

		v, ok := resultMap["status"]
		assert.True(t, ok)
		assert.Equal(t, float64(http.StatusBadRequest), v)

		v, ok = resultMap["message"]
		assert.True(t, ok)
		assert.Equal(t, "failed to create the book", v)

		v, ok = resultMap["data"]
		assert.True(t, ok)
		bookMap, ok := v.(map[string]interface{})
		assert.True(t, ok)
		assert.Empty(t, bookMap["title"])
		assert.Equal(t, "Test book description", bookMap["description"])
		assert.Equal(t, "Jerome Amon", bookMap["author"])
		assert.Equal(t, "10$", bookMap["price"])
		assert.Empty(t, bookMap["createdAt"])
		assert.Empty(t, bookMap["updatedAt"])
	})

	t.Run("should fail: required field in payload", func(t *testing.T) {
		testCases := map[string]Book{
			"empty": {
				Title:       "",
				Description: "Test book description",
				Author:      "Jerome Amon",
				Price:       "10$",
			},
			"missing": {
				Description: "Test book description",
				Author:      "Jerome Amon",
				Price:       "10$",
			},
		}

		for k := range testCases {
			tc := testCases[k]
			t.Run(k, func(t *testing.T) {
				payload, err := json.Marshal(tc)
				assert.NoError(t, err)
				req := httptest.NewRequest(http.MethodPost, "/v1/books", bytes.NewBuffer(payload))
				w := httptest.NewRecorder()
				api.CreateBook(w, req, httprouter.Params{})
				res := w.Result()
				defer res.Body.Close()
				data, err := io.ReadAll(res.Body)
				assert.NoError(t, err)
				assert.Equal(t, http.StatusBadRequest, res.StatusCode)
				assert.Equal(t, "application/json; charset=UTF-8", res.Header.Get("Content-Type"))

				resultMap := make(map[string]interface{})
				err = json.Unmarshal(data, &resultMap)
				assert.NoError(t, err)

				_, ok := resultMap["requestid"]
				assert.True(t, ok)

				v, ok := resultMap["status"]
				assert.True(t, ok)
				assert.Equal(t, float64(http.StatusBadRequest), v)

				v, ok = resultMap["message"]
				assert.True(t, ok)
				assert.Equal(t, "failed to create the book", v)

				v, ok = resultMap["data"]
				assert.True(t, ok)
				errMsg, ok := v.(string)
				assert.True(t, ok)
				assert.Equal(t, "title is required", errMsg)
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
	data, err := io.ReadAll(res.Body)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, res.StatusCode)
	assert.Equal(t, "application/json; charset=UTF-8", res.Header.Get("Content-Type"))

	resultMap := make(map[string]interface{})
	err = json.Unmarshal(data, &resultMap)
	assert.NoError(t, err)

	_, ok := resultMap["requestid"]
	assert.True(t, ok)

	v, ok := resultMap["status"]
	assert.True(t, ok)
	assert.Equal(t, float64(http.StatusNotFound), v)

	v, ok = resultMap["message"]
	assert.True(t, ok)
	assert.Equal(t, "book does not exist", v)

	v, ok = resultMap["data"]
	assert.True(t, ok)

	bookMap, ok := v.(map[string]interface{})
	assert.True(t, ok)
	bookJSONString, err := json.Marshal(bookMap)
	assert.NoError(t, err)
	expected := `{"id":"", "title":"", "description":"", "author":"", "price":"", "createdAt":"", "updatedAt":""}`
	assert.JSONEq(t, expected, string(bookJSONString))
}
