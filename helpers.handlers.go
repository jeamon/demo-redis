package main

import (
	"time"

	"github.com/gofrs/uuid"
)

var (
	_ Clocker      = (*Clock)(nil)             // ensure Clock implements Clocker.
	_ UIDGenerator = (*ObjectIDGenerator)(nil) // ensure ObjectIDGenerator implements UIDGenerator.
)

// Clocker is an interface for getting current real time.
type Clocker interface {
	Now() time.Time
}

// UIDGenerator is an interface for getting a uid.
type UIDGenerator interface {
	Generate(prefix string) string
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

// ObjectIDGenerator implements the UIDGenerator interface.
type ObjectIDGenerator struct{}

// NewObjectIDGenerator returns a ready to use ObjectIDGenerator.
func NewObjectIDGenerator() *ObjectIDGenerator {
	return &ObjectIDGenerator{}
}

// Generate provides a random unique identifier.
func (g *ObjectIDGenerator) Generate(prefix string) string {
	id, _ := uuid.NewV4()
	return prefix + ":" + id.String()
}
