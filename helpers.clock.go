package main

import (
	"time"
)

var _ Clocker = (*Clock)(nil) // ensure Clock implements Clocker.

// Clocker is an interface for getting current real time.
type Clocker interface {
	Now() time.Time
}

// Clock implements the Clocker interface.
type Clock struct {
	tz *time.Location
}

// NewClock returns a ready to use Clock with timezone sets
// to UTC in production environment and Local in dev env.
func NewClock(isProd bool) *Clock {
	if isProd {
		return &Clock{time.UTC}
	}
	return &Clock{time.Local}
}

// Now provides current clock time.
func (ck *Clock) Now() time.Time {
	return time.Now().In(ck.tz)
}
