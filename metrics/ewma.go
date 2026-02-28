package metrics

import (
	"math"
	"sync"
	"sync/atomic"
	"time"
)

// EWMASnapshot is a read-only copy of an EWMA.
type EWMASnapshot float64

// Rate returns the rate of events per second at the time the snapshot was
// taken.
func (a EWMASnapshot) Rate() float64 { return float64(a) }

// NewEWMA constructs a new EWMA with the given alpha.
func NewEWMA(alpha float64) *EWMA {
	return &EWMA{alpha: alpha}
}

// NewEWMA1 constructs a new EWMA for a one-minute moving average.
func NewEWMA1() *EWMA {
	return NewEWMA(1 - math.Exp(-5.0/60.0/1))
}

// NewEWMA5 constructs a new EWMA for a five-minute moving average.
func NewEWMA5() *EWMA {
	return NewEWMA(1 - math.Exp(-5.0/60.0/5))
}

// NewEWMA15 constructs a new EWMA for a fifteen-minute moving average.
func NewEWMA15() *EWMA {
	return NewEWMA(1 - math.Exp(-5.0/60.0/15))
}

// EWMA continuously calculate an exponentially-weighted moving average
// based on an outside source of clock ticks.
type EWMA struct {
	uncounted atomic.Int64
	alpha     float64
	rate      atomic.Uint64
	init      atomic.Bool
	mutex     sync.Mutex
}

// Snapshot returns a read-only copy of the EWMA.
func (a *EWMA) Snapshot() EWMASnapshot {
	r := math.Float64frombits(a.rate.Load()) * float64(time.Second)
	return EWMASnapshot(r)
}

// tick ticks the clock to update the moving average.  It assumes it is called
// every five seconds.
func (a *EWMA) tick() {
	// Optimization to avoid mutex locking in the hot-path.
	if a.init.Load() {
		a.updateRate(a.fetchInstantRate())
		return
	}
	// Slow-path: this is only needed on the first tick() and preserves transactional updating
	// of init and rate in the else block. The first conditional is needed below because
	// a different thread could have set a.init = 1 between the time of the first atomic load and when
	// the lock was acquired.
	a.mutex.Lock()
	if a.init.Load() {
		// The fetchInstantRate() uses atomic loading, which is unnecessary in this critical section
		// but again, this section is only invoked on the first successful tick() operation.
		a.updateRate(a.fetchInstantRate())
	} else {
		a.init.Store(true)
		a.rate.Store(math.Float64bits(a.fetchInstantRate()))
	}
	a.mutex.Unlock()
}

func (a *EWMA) fetchInstantRate() float64 {
	count := a.uncounted.Swap(0)
	return float64(count) / float64(5*time.Second)
}

func (a *EWMA) updateRate(instantRate float64) {
	currentRate := math.Float64frombits(a.rate.Load())
	currentRate += a.alpha * (instantRate - currentRate)
	a.rate.Store(math.Float64bits(currentRate))
}

// Update adds n uncounted events.
func (a *EWMA) Update(n int64) {
	a.uncounted.Add(n)
}
