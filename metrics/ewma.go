package metrics

import (
	"math"
	"sync"
	"time"
)

// EWMAs continuously calculate an exponentially-weighted moving average.
type EWMA interface {
	Rate() float64
	Snapshot() EWMA
	Update(int64)
}

// NewEWMA constructs a new EWMA with the given alpha.
func NewEWMA(alpha float64) EWMA {
	return &StandardEWMA{alpha: alpha}
}

// NewEWMA1 constructs a new EWMA for a one-minute moving average.
func NewEWMA1() EWMA {
	return NewEWMA(1 - math.Exp(-5.0/60.0/1))
}

// NewEWMA5 constructs a new EWMA for a five-minute moving average.
func NewEWMA5() EWMA {
	return NewEWMA(1 - math.Exp(-5.0/60.0/5))
}

// NewEWMA15 constructs a new EWMA for a fifteen-minute moving average.
func NewEWMA15() EWMA {
	return NewEWMA(1 - math.Exp(-5.0/60.0/15))
}

// EWMASnapshot is a read-only copy of another EWMA.
type EWMASnapshot float64

// Rate returns the rate of events per second at the time the snapshot was
// taken.
func (a EWMASnapshot) Rate() float64 { return float64(a) }

// Snapshot returns the snapshot.
func (a EWMASnapshot) Snapshot() EWMA { return a }

// Update panics.
func (EWMASnapshot) Update(int64) {
	panic("Update called on an EWMASnapshot")
}

// NilEWMA is a no-op EWMA.
type NilEWMA struct{}

// Rate is a no-op.
func (NilEWMA) Rate() float64 { return 0.0 }

// Snapshot is a no-op.
func (NilEWMA) Snapshot() EWMA { return NilEWMA{} }

// Tick is a no-op.
func (NilEWMA) Tick() {}

// Update is a no-op.
func (NilEWMA) Update(n int64) {}

// StandardEWMA is the standard implementation of an EWMA, which tracks the
// number of uncounted events and their EWMA.
type StandardEWMA struct {
	uncounted int64
	alpha     float64
	ewma      float64
	init      bool
	timestamp int64
	mutex     sync.Mutex
}

// Rate returns the moving average rate of events per second.
func (a *StandardEWMA) Rate() float64 {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	if !a.init {
		return 0
	}

	// Get number of seconds elapsed.
	now := time.Now().UnixNano()
	elapsed := math.Floor(float64(now-a.timestamp) / 1e9)

	// Wait at least one second for uncounted to accumulate.
	if elapsed >= 1 && a.uncounted != 0 {
		a.ewma = a.alpha*float64(a.uncounted) + (1-a.alpha)*a.ewma

		// a.uncounted may be older than 1 second.
		a.ewma = math.Pow(1-a.alpha, elapsed-1) * a.ewma

		a.timestamp = now
		a.uncounted = 0
		return a.ewma
	}

	a.ewma = math.Pow(1-a.alpha, elapsed) * a.ewma
	a.timestamp = now
	return a.ewma
}

// Snapshot returns a read-only copy of the EWMA.
func (a *StandardEWMA) Snapshot() EWMA {
	return EWMASnapshot(a.Rate())
}

// Used to elapse time in unit tests. Not safe!
func (a *StandardEWMA) addToTimestamp(ns int64) {
	a.timestamp += ns
}

// Update adds n uncounted events.
func (a *StandardEWMA) Update(n int64) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	now := time.Now().UnixNano()
	if !a.init {
		a.ewma = float64(n)
		a.timestamp = now
		a.init = true
		return
	}

	// Wait at least 1s for uncounted events to accumulate.
	elapsed := math.Floor(float64(now-a.timestamp) / 1e9)
	if elapsed < 1 {
		a.uncounted += n
		return
	}

	// Update EWMA with data from previous 1s interval.
	a.ewma = a.alpha*float64(a.uncounted) + (1-a.alpha)*a.ewma
	a.ewma = math.Pow(1-a.alpha, elapsed-1) * a.ewma

	// n starts the next interval.
	a.timestamp = now
	a.uncounted = n
}
