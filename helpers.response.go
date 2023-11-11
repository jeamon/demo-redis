package main

import (
	"encoding/json"
	"net/http"
)

// CustomResponseWriter is a wrapper for http.ResponseWriter. It is
// used to record response details like status code and body size.
type CustomResponseWriter struct {
	http.ResponseWriter
	code  int
	bytes int
	wrote bool
}

// NewCustomResponseWriter provides CustomResponseWriter with 200 as status code.
func NewCustomResponseWriter(rw http.ResponseWriter) *CustomResponseWriter {
	return &CustomResponseWriter{
		ResponseWriter: rw,
		code:           200,
	}
}

// Header implements http.Header interface.
func (cw *CustomResponseWriter) Header() http.Header {
	return cw.ResponseWriter.Header()
}

// WriteHeader implements http.WriteHeader interface.
func (cw *CustomResponseWriter) WriteHeader(code int) {
	if cw.Header().Get("X-Timeout-Reached") != "" {
		cw.code = http.StatusGatewayTimeout
		cw.wrote = true
		return
	}

	if !cw.wrote {
		cw.code = code
		cw.wrote = true
		cw.ResponseWriter.WriteHeader(code)
	}
}

// Write implements http.Write interface. If the header X-Timeout-Reached is present
// that means the timeout middleware was already triggered so we do not send anything.
func (cw *CustomResponseWriter) Write(bytes []byte) (int, error) {
	if cw.Header().Get("X-Timeout-Reached") != "" {
		return 0, http.ErrHandlerTimeout
	}

	if !cw.wrote {
		cw.WriteHeader(cw.code)
	}

	n, err := cw.ResponseWriter.Write(bytes)
	cw.bytes += n
	return n, err
}

// Status returns the written status code.
func (cw *CustomResponseWriter) Status() int {
	return cw.code
}

// Bytes returns bytes written as response body.
func (cw *CustomResponseWriter) Bytes() int {
	return cw.bytes
}

// Unwrap returns native response writer and used by
// the http.ResponseController during its operation.
func (cw *CustomResponseWriter) Unwrap() http.ResponseWriter {
	return cw.ResponseWriter
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
