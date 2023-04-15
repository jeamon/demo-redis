package main

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"

	"github.com/gofrs/uuid"
)

const (
	BookIDPrefix         string     = "b"
	RequestIDPrefix      string     = "r"
	ContextRequestID     ContextKey = "request.id"
	ContextRequestNumber ContextKey = "request.number"
)

var (
	ErrNotFoundBook = errors.New("book not found")
)

type (
	ContextKey        string
	missingFieldError string
)

func (m missingFieldError) Error() string {
	return string(m) + " is required"
}

// GenerateID provides a random uid.
func GenerateID(prefix string) string {
	id, _ := uuid.NewV4()
	return prefix + ":" + id.String()
}

// GetValueFromContext returns the value of a given key in the context
// if this key is not available, it returns an empty string.
func GetValueFromContext(ctx context.Context, contextKey ContextKey) string {
	if val := ctx.Value(contextKey); val != nil {
		return val.(string)
	}
	return ""
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
