package metrics

import (
	"sync"
	"time"
)

// GetOrRegisterTimer returns an existing Timer or constructs and registers a
// new Timer.
// Be sure to unregister the meter from the registry once it is of no use to
// allow for garbage collection.
func GetOrRegisterTimer(name string, r Registry) *Timer {
	return getOrRegister(name, NewTimer, r)
}

// NewCustomTimer constructs a new Timer from a Histogram and a Meter.
// Be sure to call Stop() once the timer is of no use to allow for garbage collection.
func NewCustomTimer(h Histogram, m *Meter) *Timer {
	return &Timer{
		histogram: h,
		meter:     m,
	}
}

// NewRegisteredTimer constructs and registers a new Timer.
// Be sure to unregister the meter from the registry once it is of no use to
// allow for garbage collection.
func NewRegisteredTimer(name string, r Registry) *Timer {
	c := NewTimer()
	if nil == r {
		r = DefaultRegistry
	}
	r.Register(name, c)
	return c
}

// NewTimer constructs a new Timer using an exponentially-decaying
// sample with the same reservoir size and alpha as UNIX load averages.
// Be sure to call Stop() once the timer is of no use to allow for garbage collection.
func NewTimer() *Timer {
	return &Timer{
		histogram: NewHistogram(NewExpDecaySample(1028, 0.015)),
		meter:     NewMeter(),
	}
}

// Timer captures the duration and rate of events, using a Histogram and a Meter.
type Timer struct {
	histogram Histogram
	meter     *Meter
	mutex     sync.Mutex
}

// Snapshot returns a read-only copy of the timer.
func (t *Timer) Snapshot() *TimerSnapshot {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return &TimerSnapshot{
		histogram: t.histogram.Snapshot(),
		meter:     t.meter.Snapshot(),
	}
}

// Stop stops the meter.
func (t *Timer) Stop() {
	t.meter.Stop()
}

// Time record the duration of the execution of the given function.
func (t *Timer) Time(f func()) {
	ts := time.Now()
	f()
	t.Update(time.Since(ts))
}

// Update the duration of an event, in nanoseconds.
func (t *Timer) Update(d time.Duration) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.histogram.Update(d.Nanoseconds())
	t.meter.Mark(1)
}

// UpdateSince update the duration of an event that started at a time and ends now.
// The record uses nanoseconds.
func (t *Timer) UpdateSince(ts time.Time) {
	t.Update(time.Since(ts))
}

// TimerSnapshot is a read-only copy of another Timer.
type TimerSnapshot struct {
	histogram HistogramSnapshot
	meter     *MeterSnapshot
}

// Count returns the number of events recorded at the time the snapshot was
// taken.
func (t *TimerSnapshot) Count() int64 { return t.histogram.Count() }

// Max returns the maximum value at the time the snapshot was taken.
func (t *TimerSnapshot) Max() int64 { return t.histogram.Max() }

// Size returns the size of the sample at the time the snapshot was taken.
func (t *TimerSnapshot) Size() int { return t.histogram.Size() }

// Mean returns the mean value at the time the snapshot was taken.
func (t *TimerSnapshot) Mean() float64 { return t.histogram.Mean() }

// Min returns the minimum value at the time the snapshot was taken.
func (t *TimerSnapshot) Min() int64 { return t.histogram.Min() }

// Percentile returns an arbitrary percentile of sampled values at the time the
// snapshot was taken.
func (t *TimerSnapshot) Percentile(p float64) float64 {
	return t.histogram.Percentile(p)
}

// Percentiles returns a slice of arbitrary percentiles of sampled values at
// the time the snapshot was taken.
func (t *TimerSnapshot) Percentiles(ps []float64) []float64 {
	return t.histogram.Percentiles(ps)
}

// Rate1 returns the one-minute moving average rate of events per second at the
// time the snapshot was taken.
func (t *TimerSnapshot) Rate1() float64 { return t.meter.Rate1() }

// Rate5 returns the five-minute moving average rate of events per second at
// the time the snapshot was taken.
func (t *TimerSnapshot) Rate5() float64 { return t.meter.Rate5() }

// Rate15 returns the fifteen-minute moving average rate of events per second
// at the time the snapshot was taken.
func (t *TimerSnapshot) Rate15() float64 { return t.meter.Rate15() }

// RateMean returns the meter's mean rate of events per second at the time the
// snapshot was taken.
func (t *TimerSnapshot) RateMean() float64 { return t.meter.RateMean() }

// StdDev returns the standard deviation of the values at the time the snapshot
// was taken.
func (t *TimerSnapshot) StdDev() float64 { return t.histogram.StdDev() }

// Sum returns the sum at the time the snapshot was taken.
func (t *TimerSnapshot) Sum() int64 { return t.histogram.Sum() }

// Variance returns the variance of the values at the time the snapshot was
// taken.
func (t *TimerSnapshot) Variance() float64 { return t.histogram.Variance() }
