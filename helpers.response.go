package main

import (
	"encoding/json"
	"net/http"
)

// CustomResponseWriter is a wrapper for http.ResponseWriter.
// It is used to record response attributes like statusCode.
type CustomResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

// NewCustomResponseWriter provides CustomResponseWriter with 200 as status code.
func NewCustomResponseWriter(rw http.ResponseWriter) *CustomResponseWriter {
	return &CustomResponseWriter{
		ResponseWriter: rw,
		statusCode:     200,
	}
}

// Header implements http.Header interface.
func (rw *CustomResponseWriter) Header() http.Header {
	return rw.ResponseWriter.Header()
}

// WriteHeader implements http.WriteHeader interface.
func (rw *CustomResponseWriter) WriteHeader(statusCode int) {
	rw.ResponseWriter.WriteHeader(statusCode)
	rw.statusCode = statusCode
}

// Write implements http.Write interface.
func (rw *CustomResponseWriter) Write(bytes []byte) (int, error) {
	return rw.ResponseWriter.Write(bytes)
}

func (rw *CustomResponseWriter) Status() int {
	return rw.statusCode
}

// APIError is the data model sent when an error occurred during request processing.
type APIError struct {
	RequestID string      `json:"requestid"`
	Status    int         `json:"status"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data"`
}

// APIResponse is the data model sent when a request succeed.
// We use the omitempty flag on the `total` field. This helps
// set the value for `GetAllBook` calls only.
type APIResponse struct {
	RequestID string      `json:"requestid"`
	Status    int         `json:"status"`
	Message   string      `json:"message"`
	Total     *int        `json:"total,omitempty"`
	Data      interface{} `json:"data"`
}

func NewAPIError(requestid string, status int, message string, data interface{}) *APIError {
	return &APIError{
		RequestID: requestid,
		Status:    status,
		Message:   message,
		Data:      data,
	}
}

func GenericResponse(requestid string, status int, message string, total *int, data interface{}) *APIResponse {
	return &APIResponse{
		RequestID: requestid,
		Status:    status,
		Message:   message,
		Total:     total,
		Data:      data,
	}
}

func WriteErrorResponse(w http.ResponseWriter, errResp *APIError) error {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(errResp.Status)
	return json.NewEncoder(w).Encode(errResp)
}

func WriteResponse(w http.ResponseWriter, resp *APIResponse) error {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(resp.Status)
	return json.NewEncoder(w).Encode(resp)
}
