package main

import (
	"strings"

	"github.com/gofrs/uuid"
)

var _ UIDHandler = (*IDsHandler)(nil) // ensure IDsHandler implements UIDHandler.

// UIDGenerator is an interface for getting a uid.
type UIDHandler interface {
	Generate(prefix string) string
	IsValid(prefix string, id string) bool
}

// IDsHandler implements the UIDHandler interface.
type IDsHandler struct{}

// NewIDsHandler returns a ready to use IDsHandler.
func NewIDsHandler() *IDsHandler {
	return &IDsHandler{}
}

// Generate provides a random unique identifier.
func (idh *IDsHandler) Generate(prefix string) string {
	id, _ := uuid.NewV4()
	return prefix + ":" + id.String()
}

// IsValid checks if a given string is a valid uuid after removal of custom prefix.
func (idh *IDsHandler) IsValid(id, prefix string) bool {
	if u := uuid.FromStringOrNil(strings.TrimPrefix(id, prefix+":")); u != uuid.Nil {
		return true
	}
	return false
}
