package usecase

import "time"

// Clock provides the current time to application scenarios.
type Clock interface {
	Now() time.Time
}

// SystemClock is the production clock implementation.
type SystemClock struct{}

// Now returns the current UTC time.
func (SystemClock) Now() time.Time {
	return time.Now().UTC()
}
