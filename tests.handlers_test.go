package main

import (
	"encoding/json"
	"io/ioutil"
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
	data, err := ioutil.ReadAll(res.Body)
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
