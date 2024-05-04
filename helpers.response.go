package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

// CustomResponseWriter is a wrapper for http.ResponseWriter. It is
// used to record response details like status code and body size.
// The underlying network connection is tracked for dynamic read/write
// deadline setup.
type CustomResponseWriter struct {
	http.ResponseWriter
	conn  net.Conn
	code  int
	bytes int
	wrote bool
}

// NewCustomResponseWriter provides CustomResponseWriter with 200 as status code.
func NewCustomResponseWriter(rw http.ResponseWriter, c net.Conn) *CustomResponseWriter {
	return &CustomResponseWriter{
		ResponseWriter: rw,
		conn:           c,
		code:           200,
	}
}

// Header implements http.Header interface.
func (cw *CustomResponseWriter) Header() http.Header {
	return cw.ResponseWriter.Header()
}

// WriteHeader implements http.WriteHeader interface.
func (cw *CustomResponseWriter) WriteHeader(code int) {
	if cw.Header().Get("X-DRAP-ABORTED") != "" {
		cw.code = code
		cw.wrote = true
		return
	}

	if !cw.wrote {
		cw.code = code
		cw.wrote = true
		cw.ResponseWriter.WriteHeader(code)
	}
}

// Write implements http.Write interface. If the header X-DRAP-ABORTED was set
// that means the timeout middleware was already triggered so the final handler
// should not send any response to client.
func (cw *CustomResponseWriter) Write(bytes []byte) (int, error) {
	if cw.Header().Get("X-DRAP-ABORTED") != "" {
		return 0, fmt.Errorf("handler: request timed out or cancelled")
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

// SetWriteDeadline rewrites the underlying connection write deadline.
// This is called by http.ResponseController SetWriteDeadline method.
func (cw *CustomResponseWriter) SetWriteDeadline(t time.Time) error {
	return cw.conn.SetWriteDeadline(t)
}

// SetReadDeadline rewrites the underlying connection read deadline.
// This is called by http.ResponseController SetReadDeadline method.
func (cw *CustomResponseWriter) SetReadDeadline(t time.Time) error {
	return cw.conn.SetReadDeadline(t)
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

// WriteErrorResponse is used to send error response to client. In case the client closes the request,
// it logs the stats with the Nginx non standard status code 499 (Client Closed Request). This means
// the timeout middleware already kicked-in and did send the response. In case of request processing
// timeout we set the status code to 504 which will be used to log the stats. Here also, the middleware
// already kicked-in and sent a json message to client.
func WriteErrorResponse(ctx context.Context, w http.ResponseWriter, errResp *APIError) error {
	if err := ctx.Err(); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			w.WriteHeader(http.StatusGatewayTimeout)
		} else {
			w.WriteHeader(499)
		}
		return ctx.Err()
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(errResp.Status)
	return json.NewEncoder(w).Encode(errResp)
}

// WriteResponse is used to send success api response to client. It sets the status code to 499
// in case client cancelled the request, and to 504 if the request processing timed out.
func WriteResponse(ctx context.Context, w http.ResponseWriter, resp *APIResponse) error {
	if err := ctx.Err(); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			w.WriteHeader(http.StatusGatewayTimeout)
		} else {
			w.WriteHeader(499)
		}
		return ctx.Err()
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(resp.Status)
	return json.NewEncoder(w).Encode(resp)
}

// StatusResponse is the data model sent when status endpoint is called.
type StatusResponse struct {
	RequestID string `json:"requestid"`
	Status    string `json:"status"`
	Message   string `json:"message"`
}
