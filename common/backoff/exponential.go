package backoff

import (
	"math"
	"math/rand"
	"time"
)

// Exponential is a backoff strategy that increases the delay between retries exponentially.
type Exponential struct {
	attempt int

	maxJitter time.Duration

	min time.Duration
	max time.Duration
}

func NewExponential(minimum, maximum, maxJitter time.Duration) *Exponential {
	return &Exponential{
		min:       minimum,
		max:       maximum,
		maxJitter: maxJitter,
	}
}

func (e *Exponential) NextDuration() time.Duration {
	var jitter time.Duration
	if e.maxJitter > 0 {
		jitter = time.Duration(rand.Int63n(e.maxJitter.Nanoseconds()))
	}

	minFloat := float64(e.min)
	duration := math.Pow(2, float64(e.attempt)) * minFloat

	// limit at configured maximum
	if duration > float64(e.max) {
		duration = float64(e.max)
	}

	e.attempt++
	return time.Duration(duration) + jitter
}

func (e *Exponential) Reset() {
	e.attempt = 0
}

func (e *Exponential) Attempt() int {
	return e.attempt
}
