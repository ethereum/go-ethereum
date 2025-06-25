package metrics

import (
	"sync"
	"time"
)

// GetOrRegisterResettingTimer returns an existing ResettingTimer or constructs and registers a
// new ResettingTimer.
func GetOrRegisterResettingTimer(name string, r Registry) *ResettingTimer {
	if nil == r {
		r = DefaultRegistry
	}
	return r.GetOrRegister(name, NewResettingTimer).(*ResettingTimer)
}

// NewRegisteredResettingTimer constructs and registers a new ResettingTimer.
func NewRegisteredResettingTimer(name string, r Registry) *ResettingTimer {
	c := NewResettingTimer()
	if nil == r {
		r = DefaultRegistry
	}
	r.Register(name, c)
	return c
}

// NewResettingTimer constructs a new ResettingTimer
func NewResettingTimer() *ResettingTimer {
	return &ResettingTimer{
		values: make([]int64, 0, 10),
	}
}

// ResettingTimer is used for storing aggregated values for timers, which are reset on every flush interval.
type ResettingTimer struct {
	values []int64
	sum    int64 // sum is a running count of the total sum, used later to calculate mean

	mutex sync.Mutex
}

// Snapshot resets the timer and returns a read-only copy of its contents.
func (t *ResettingTimer) Snapshot() *ResettingTimerSnapshot {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	snapshot := &ResettingTimerSnapshot{}
	if len(t.values) > 0 {
		snapshot.mean = float64(t.sum) / float64(len(t.values))
		snapshot.values = t.values
		t.values = make([]int64, 0, 10)
	}
	t.sum = 0
	return snapshot
}

// Time records the duration of the execution of the given function.
func (t *ResettingTimer) Time(f func()) {
	ts := time.Now()
	f()
	t.Update(time.Since(ts))
}

// Update records the duration of an event.
func (t *ResettingTimer) Update(d time.Duration) {
	if !metricsEnabled {
		return
	}
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.values = append(t.values, int64(d))
	t.sum += int64(d)
}

// UpdateSince records the duration of an event that started at a time and ends now.
func (t *ResettingTimer) UpdateSince(ts time.Time) {
	t.Update(time.Since(ts))
}

// ResettingTimerSnapshot is a point-in-time copy of another ResettingTimer.
type ResettingTimerSnapshot struct {
	values              []int64
	mean                float64
	max                 int64
	min                 int64
	thresholdBoundaries []float64
	calculated          bool
}

// Count return the length of the values from snapshot.
func (t *ResettingTimerSnapshot) Count() int {
	return len(t.values)
}

// Percentiles returns the boundaries for the input percentiles.
// note: this method is not thread safe
func (t *ResettingTimerSnapshot) Percentiles(percentiles []float64) []float64 {
	t.calc(percentiles)
	return t.thresholdBoundaries
}

// Mean returns the mean of the snapshotted values
// note: this method is not thread safe
func (t *ResettingTimerSnapshot) Mean() float64 {
	if !t.calculated {
		t.calc(nil)
	}

	return t.mean
}

// Max returns the max of the snapshotted values
// note: this method is not thread safe
func (t *ResettingTimerSnapshot) Max() int64 {
	if !t.calculated {
		t.calc(nil)
	}
	return t.max
}

// Min returns the min of the snapshotted values
// note: this method is not thread safe
func (t *ResettingTimerSnapshot) Min() int64 {
	if !t.calculated {
		t.calc(nil)
	}
	return t.min
}

func (t *ResettingTimerSnapshot) calc(percentiles []float64) {
	scores := CalculatePercentiles(t.values, percentiles)
	t.thresholdBoundaries = scores
	if len(t.values) == 0 {
		return
	}
	t.min = t.values[0]
	t.max = t.values[len(t.values)-1]
}
