package metrics

import "sync/atomic"

// GaugeSnapshot is a read-only copy of a Gauge.
type GaugeSnapshot int64

// Value returns the value at the time the snapshot was taken.
func (g GaugeSnapshot) Value() int64 { return int64(g) }

// GetOrRegisterGauge returns an existing Gauge or constructs and registers a
// new Gauge.
func GetOrRegisterGauge(name string, r Registry) *Gauge {
	if r == nil {
		r = DefaultRegistry
	}
	return r.GetOrRegister(name, NewGauge).(*Gauge)
}

// NewGauge constructs a new Gauge.
func NewGauge() *Gauge {
	return &Gauge{}
}

// NewRegisteredGauge constructs and registers a new Gauge.
func NewRegisteredGauge(name string, r Registry) *Gauge {
	c := NewGauge()
	if r == nil {
		r = DefaultRegistry
	}
	r.Register(name, c)
	return c
}

// Gauge holds an int64 value that can be set arbitrarily.
type Gauge atomic.Int64

// Snapshot returns a read-only copy of the gauge.
func (g *Gauge) Snapshot() GaugeSnapshot {
	return GaugeSnapshot((*atomic.Int64)(g).Load())
}

// Update updates the gauge's value.
func (g *Gauge) Update(v int64) {
	(*atomic.Int64)(g).Store(v)
}

// UpdateIfGt updates the gauge's value if v is larger then the current value.
func (g *Gauge) UpdateIfGt(v int64) {
	value := (*atomic.Int64)(g)
	for {
		exist := value.Load()
		if exist >= v {
			break
		}
		if value.CompareAndSwap(exist, v) {
			break
		}
	}
}

// Dec decrements the gauge's current value by the given amount.
func (g *Gauge) Dec(i int64) {
	(*atomic.Int64)(g).Add(-i)
}

// Inc increments the gauge's current value by the given amount.
func (g *Gauge) Inc(i int64) {
	(*atomic.Int64)(g).Add(i)
}
