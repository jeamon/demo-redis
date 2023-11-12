package main

import (
	"time"
)

var (
	_ Clocker       = (*Clock)(nil)     // ensure Clock implements Clocker
	_ TickerClocker = (*TickClock)(nil) // ensure TickClock implements TickerClocker
)

// TickerClocker is an interface which can provides the current time and a ticker.
type TickerClocker interface {
	Clocker
	NewTicker(time.Duration) *time.Ticker
}

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

type TickClock struct {
	tz *time.Location
}

func NewTickClock(isProd bool) *TickClock {
	if isProd {
		return &TickClock{time.UTC}
	}
	return &TickClock{time.Local}
}

func (tc *TickClock) Now() time.Time {
	return time.Now().In(tc.tz)
}

func (tc *TickClock) NewTicker(d time.Duration) *time.Ticker {
	return time.NewTicker(d)
}
