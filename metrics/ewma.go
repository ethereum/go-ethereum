package metrics

import (
	"math"
	"sync"
	"time"
)

type EWMASnapshot interface {
	Rate() float64
}

// EWMAs calculate an exponentially-weighted moving average.
type EWMA interface {
	Snapshot() EWMASnapshot
	Update(int64)
}

// NewEWMA constructs a new EWMA with the given alpha and sampling period.
func NewEWMA(alpha float64, period time.Duration) EWMA {
	return &StandardEWMA{alpha: alpha, period: period, ts: time.Now()}
}

// NewEWMA1 constructs a new EWMA for a one-minute moving average.
func NewEWMA1() EWMA {
	return NewEWMA(1-math.Exp(-5.0/60.0/1), 5*time.Second)
}

// NewEWMA5 constructs a new EWMA for a five-minute moving average.
func NewEWMA5() EWMA {
	return NewEWMA(1-math.Exp(-5.0/60.0/5), 5*time.Second)
}

// NewEWMA15 constructs a new EWMA for a fifteen-minute moving average.
func NewEWMA15() EWMA {
	return NewEWMA(1-math.Exp(-5.0/60.0/15), 5*time.Second)
}

// ewmaSnapshot is a read-only copy of another EWMA.
type ewmaSnapshot float64

// Rate returns the rate of events per second at the time the snapshot was
// taken.
func (a ewmaSnapshot) Rate() float64 { return float64(a) }

// NilEWMA is a no-op EWMA.
type NilEWMA struct{}

func (NilEWMA) Snapshot() EWMASnapshot { return (*emptySnapshot)(nil) }
func (NilEWMA) Update(n int64)         {}

// StandardEWMA is the standard implementation of an EWMA.
type StandardEWMA struct {
	uncounted int64
	alpha     float64
	period    time.Duration
	ewma      float64
	ts        time.Time
	init      bool
	mutex     sync.Mutex
}

// Snapshot returns a read-only copy of the EWMA.
func (a *StandardEWMA) Snapshot() EWMASnapshot {
	return ewmaSnapshot(a.rate())
}

// rate returns the moving average rate of events per second.
func (a *StandardEWMA) rate() float64 {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	if time.Since(a.ts)/a.period < 1 {
		return a.ewma * float64(time.Second)
	}
	a.updateRate()
	return a.ewma * float64(time.Second)
}

func (a *StandardEWMA) updateRate() {
	periods := time.Since(a.ts) / a.period
	rate := float64(a.uncounted) / float64(a.period)

	a.ewma = a.alpha*(rate) + (1-a.alpha)*a.ewma
	a.ts = a.ts.Add(a.period)
	a.uncounted = 0
	periods -= 1

	if !a.init {
		a.ewma = rate
		a.init = true
	}

	a.ewma = math.Pow(1-a.alpha, float64(periods)) * a.ewma
	a.ts = a.ts.Add(periods * a.period) //nolint:durationcheck
}

// Update adds n uncounted events.
func (a *StandardEWMA) Update(n int64) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	if time.Since(a.ts)/a.period < 1 {
		a.uncounted += n
		return
	}
	a.updateRate()
}

// used to elapse time in unit tests.
func (a *StandardEWMA) addToTimestamp(d time.Duration) {
	a.ts = a.ts.Add(d)
}
