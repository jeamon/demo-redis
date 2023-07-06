package main

import "time"

// ensure Clock implements Clocker.
var _ Clocker = (*Clock)(nil)

// Clocker is an interface for getting current real time.
type Clocker interface {
	Now() time.Time
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
