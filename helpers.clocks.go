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
// Refactoring: remove Zero from this interface. The create
// ZeroClocker which embeds Clock and Zero() method. Then
// TickerClocker can use Clocker without being forced to
// implement Zero() method.
type Clocker interface {
	Now() time.Time
	Zero() time.Time
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

// Zero returns zero time.
func (ck *Clock) Zero() time.Time {
	return time.Time{}
}

type TickClock struct {
	clock Clocker
}

func NewTickClock(ck Clocker) *TickClock {
	return &TickClock{ck}
}

func (tc *TickClock) Now() time.Time {
	return tc.clock.Now()
}

func (tc *TickClock) Zero() time.Time {
	return tc.clock.Zero()
}

func (tc *TickClock) NewTicker(d time.Duration) *time.Ticker {
	return time.NewTicker(d)
}
