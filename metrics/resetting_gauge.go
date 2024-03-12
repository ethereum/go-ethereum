package metrics

import "sync/atomic"

// ResettingGaugeSnapshot contains a readonly int64.
type ResettingGaugeSnapshot interface {
	Value() int64
}

// ResettingGauge holds an int64 value that can be set arbitrarily.
type ResettingGauge interface {
	Snapshot() ResettingGaugeSnapshot
	Update(int64)
	UpdateIfGt(int64)
	Dec(int64)
	Inc(int64)
}

// GetOrRegisterResettingGauge returns an existing Gauge or constructs and registers a
// new ResettingGauge.
func GetOrRegisterResettingGauge(name string, r Registry) ResettingGauge {
	if nil == r {
		r = DefaultRegistry
	}
	return r.GetOrRegister(name, NewResettingGauge).(ResettingGauge)
}

// NewResettingGauge constructs a new StandardResettingGauge.
func NewResettingGauge() ResettingGauge {
	if !Enabled {
		return NilResettingGauge{}
	}
	return &StandardResettingGauge{}
}

// NewRegisteredResettingGauge constructs and registers a new StandardResettingGauge.
func NewRegisteredResettingGauge(name string, r Registry) ResettingGauge {
	c := NewResettingGauge()
	if nil == r {
		r = DefaultRegistry
	}
	r.Register(name, c)
	return c
}

// resettingGaugeSnapshot is a read-only copy of another Gauge.
type resettingGaugeSnapshot int64

// Value returns the value at the time the snapshot was taken.
func (g resettingGaugeSnapshot) Value() int64 { return int64(g) }

// NilResettingGauge is a no-op Gauge.
type NilResettingGauge struct{}

func (NilResettingGauge) Snapshot() ResettingGaugeSnapshot { return (*emptySnapshot)(nil) }
func (NilResettingGauge) Update(v int64)                   {}
func (NilResettingGauge) UpdateIfGt(v int64)               {}
func (NilResettingGauge) Dec(i int64)                      {}
func (NilResettingGauge) Inc(i int64)                      {}

// StandardResettingGauge is the resetting implementation of a Gauge and uses the
// sync/atomic package to manage a single int64 value.
type StandardResettingGauge struct {
	value atomic.Int64
}

// Snapshot returns a read-only copy of the gauge.
func (g *StandardResettingGauge) Snapshot() ResettingGaugeSnapshot {
	snapshot := resettingGaugeSnapshot(g.value.Load())
	g.value.Store(0)
	return snapshot
}

// Update updates the gauge's value.
func (g *StandardResettingGauge) Update(v int64) {
	g.value.Store(v)
}

// Update updates the gauge's value if v is larger then the current value.
func (g *StandardResettingGauge) UpdateIfGt(v int64) {
	for {
		exist := g.value.Load()
		if exist >= v {
			break
		}
		if g.value.CompareAndSwap(exist, v) {
			break
		}
	}
}

// Dec decrements the gauge's current value by the given amount.
func (g *StandardResettingGauge) Dec(i int64) {
	g.value.Add(-i)
}

// Inc increments the gauge's current value by the given amount.
func (g *StandardResettingGauge) Inc(i int64) {
	g.value.Add(i)
}
