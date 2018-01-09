package metrics

import (
	"sync"
	"time"
)

// Initial slice capacity for the values stored in a ResettingTimer
const InitialResettingTimerSliceCap = 10

// ResettingTimer is used for storing aggregated values for timers, which are reset on every flush interval.
type ResettingTimer interface {
	Values() []int64
	Snapshot() ResettingTimer
	Time(func())
	Update(time.Duration)
	UpdateSince(time.Time)
}

// GetOrRegisterResettingTimer returns an existing ResettingTimer or constructs and registers a
// new StandardResettingTimer.
func GetOrRegisterResettingTimer(name string, r Registry) ResettingTimer {
	if nil == r {
		r = DefaultRegistry
	}
	return r.GetOrRegister(name, NewResettingTimer).(ResettingTimer)
}

// NewRegisteredResettingTimer constructs and registers a new StandardResettingTimer.
func NewRegisteredResettingTimer(name string, r Registry) ResettingTimer {
	c := NewResettingTimer()
	if nil == r {
		r = DefaultRegistry
	}
	r.Register(name, c)
	return c
}

// NewResettingTimer constructs a new StandardResettingTimer
func NewResettingTimer() ResettingTimer {
	if UseNilMetrics {
		return NilResettingTimer{}
	}
	return &StandardResettingTimer{
		values: make([]int64, 0, InitialResettingTimerSliceCap),
	}
}

// NilResettingTimer is a no-op ResettingTimer.
type NilResettingTimer struct {
}

// Values is a no-op.
func (NilResettingTimer) Values() []int64 { return nil }

// Snapshot is a no-op.
func (NilResettingTimer) Snapshot() ResettingTimer { return NilResettingTimer{} }

// Time is a no-op.
func (NilResettingTimer) Time(func()) {}

// Update is a no-op.
func (NilResettingTimer) Update(time.Duration) {}

// UpdateSince is a no-op.
func (NilResettingTimer) UpdateSince(time.Time) {}

// StandardResettingTimer is the standard implementation of a ResettingTimer.
// and Meter.
type StandardResettingTimer struct {
	values []int64
	mutex  sync.Mutex
}

// Values returns a slice with all measurements.
func (t *StandardResettingTimer) Values() []int64 {
	return t.values
}

// Snapshot resets the timer and returns a read-only copy of its contents.
func (t *StandardResettingTimer) Snapshot() ResettingTimer {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	currentValues := t.values
	t.values = make([]int64, 0, InitialResettingTimerSliceCap)

	return &ResettingTimerSnapshot{
		values: currentValues,
	}
}

// Record the duration of the execution of the given function.
func (t *StandardResettingTimer) Time(f func()) {
	ts := time.Now()
	f()
	t.Update(time.Since(ts))
}

// Record the duration of an event.
func (t *StandardResettingTimer) Update(d time.Duration) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.values = append(t.values, int64(d))
}

// Record the duration of an event that started at a time and ends now.
func (t *StandardResettingTimer) UpdateSince(ts time.Time) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.values = append(t.values, int64(time.Since(ts)))
}

// ResettingTimerSnapshot is a read-only copy of another ResettingTimer.
type ResettingTimerSnapshot struct {
	values []int64
}

// Snapshot returns the snapshot.
func (t *ResettingTimerSnapshot) Snapshot() ResettingTimer { return t }

// Time panics.
func (*ResettingTimerSnapshot) Time(func()) {
	panic("Time called on a ResettingTimerSnapshot")
}

// Update panics.
func (*ResettingTimerSnapshot) Update(time.Duration) {
	panic("Update called on a ResettingTimerSnapshot")
}

// UpdateSince panics.
func (*ResettingTimerSnapshot) UpdateSince(time.Time) {
	panic("UpdateSince called on a ResettingTimerSnapshot")
}

// UpdateSince panics.
func (t *ResettingTimerSnapshot) Values() []int64 {
	return t.values
}
