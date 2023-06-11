package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// This file contains unit tests for each api handler.

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
func TestCreateBookHandler(t *testing.T) {
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

	mockRepo := &MockBookStorage{
		AddFunc: func(ctx context.Context, id string, book Book) error {
			return nil
		},
	}
	bs := NewBookService(zap.NewNop(), nil, mockRepo)
	api := NewAPIHandler(zap.NewNop(), nil, &Statistics{started: time.Now()}, bs)

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

	v, ok := resultMap["message"]
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
}
