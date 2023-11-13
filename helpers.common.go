package main

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"strings"
)

var ErrBookNotFound = errors.New("book not found")

type (
	ContextKey        string
	missingFieldError string
)

const (
	BookIDPrefix            string     = "b"
	RequestIDPrefix         string     = "r"
	RequestIDContextKey     ContextKey = "request.id"
	RequestNumberContextKey ContextKey = "request.number"
	ConnContextKey          ContextKey = "http-conn"
)

func (m missingFieldError) Error() string {
	return string(m) + " is required"
}

// GetValueFromContext returns the value of a given key in the context
// if this key is not available, it returns an empty string.
func GetValueFromContext(ctx context.Context, contextKey ContextKey) string {
	if val := ctx.Value(contextKey); val != nil {
		return val.(string)
	}
	return ""
}

// GetRequestNumberFromContext returns the request number set in
// the context. if not previously set then it returns 0.
func GetRequestNumberFromContext(ctx context.Context) uint64 {
	if val := ctx.Value(RequestNumberContextKey); val != nil {
		return val.(uint64)
	}
	return 0
}

// DecodeCreateOrUpdateBookRequestBody is a helper function to read the content of a book creation or update request.
func DecodeCreateOrUpdateBookRequestBody(r *http.Request, book *Book) error {
	if r.Body == nil {
		return errors.New("invalid create book request body")
	}
	return json.NewDecoder(r.Body).Decode(book)
}

// ValidateCreateBookRequestBody is a helper function to check if the content of a book creation request is valid.
func ValidateCreateBookRequestBody(book *Book) error {
	if len(book.Title) == 0 {
		return missingFieldError("title")
	}

	if len(book.Description) == 0 {
		return missingFieldError("description")
	}

	if len(book.Author) == 0 {
		return missingFieldError("author")
	}

	if len(book.Price) == 0 {
		return missingFieldError("price")
	}

	return nil
}

// ValidateUpdateBookRequestBody is a helper function to check if the content of a book update request is valid.
func ValidateUpdateBookRequestBody(book *Book) error {
	if err := ValidateCreateBookRequestBody(book); err != nil {
		return err
	}

	if len(book.ID) == 0 {
		return missingFieldError("id")
	}

	if len(book.CreatedAt) == 0 {
		return missingFieldError("created_at")
	}

	return nil
}

// GetRequestSourceIP helps find the source IP of the caller.
func GetRequestSourceIP(r *http.Request) string {
	// Get IP from the X-REAL-IP header
	ip := r.Header.Get("X-REAL-IP")
	netIP := net.ParseIP(ip)
	if netIP != nil {
		return ip
	}

	// Get IP from X-FORWARDED-FOR header
	ips := r.Header.Get("X-FORWARDED-FOR")
	splitIps := strings.Split(ips, ",")
	for _, ip := range splitIps {
		netIP = net.ParseIP(ip)
		if netIP != nil {
			return ip
		}
	}

	// Get IP from RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return ""
	}
	netIP = net.ParseIP(ip)
	if netIP != nil {
		return ip
	}
	return ""
}

// IsAppRunningInDocker checks the existence of the .dockerenv
// file at the root directory and returns a boolean result. This
// helps know if the App is running in a docker container or not.
func IsAppRunningInDocker() bool {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	return false
}

// SaveConnInContext is the hook used by the server under ConnContext.
// It sets the underlying connection into the request context for later
// use by ReadDeadline or WriteDeadline method on *CustomResponseWriter.
func SaveConnInContext(ctx context.Context, c net.Conn) context.Context {
	return context.WithValue(ctx, ConnContextKey, c)
}

// GetConnFromContext returns the connection saved into the context.
func GetConnFromContext(ctx context.Context) net.Conn {
	return ctx.Value(ConnContextKey).(net.Conn)
}
