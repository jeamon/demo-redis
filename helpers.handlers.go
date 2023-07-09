package main

import (
	"strings"
	"time"

	"github.com/gofrs/uuid"
)

var (
	_ Clocker    = (*Clock)(nil)      // ensure Clock implements Clocker.
	_ UIDHandler = (*IDsHandler)(nil) // ensure IDsHandler implements UIDHandler.
)

// Clocker is an interface for getting current real time.
type Clocker interface {
	Now() time.Time
}

// UIDGenerator is an interface for getting a uid.
type UIDHandler interface {
	Generate(prefix string) string
	IsValid(prefix string, id string) bool
}

// Clock implements the Clocker interface.
type Clock struct{}

// NewClock returns a ready to use Clock.
func NewClock() *Clock {
	return &Clock{}
}

// Now provides current clock time.
func (ck *Clock) Now() time.Time {
	return time.Now()
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
