package metrics

import (
	"math"
	"sync/atomic"
)

// GetOrRegisterGaugeFloat64 returns an existing GaugeFloat64 or constructs and registers a
// new GaugeFloat64.
func GetOrRegisterGaugeFloat64(name string, r Registry) *GaugeFloat64 {
	return getOrRegister(name, NewGaugeFloat64, r)
}

// GaugeFloat64Snapshot is a read-only copy of a GaugeFloat64.
type GaugeFloat64Snapshot float64

// Value returns the value at the time the snapshot was taken.
func (g GaugeFloat64Snapshot) Value() float64 { return float64(g) }

// NewGaugeFloat64 constructs a new GaugeFloat64.
func NewGaugeFloat64() *GaugeFloat64 {
	return new(GaugeFloat64)
}

// NewRegisteredGaugeFloat64 constructs and registers a new GaugeFloat64.
func NewRegisteredGaugeFloat64(name string, r Registry) *GaugeFloat64 {
	c := NewGaugeFloat64()
	if nil == r {
		r = DefaultRegistry
	}
	r.Register(name, c)
	return c
}

// GaugeFloat64 hold a float64 value that can be set arbitrarily.
type GaugeFloat64 atomic.Uint64

// Snapshot returns a read-only copy of the gauge.
func (g *GaugeFloat64) Snapshot() GaugeFloat64Snapshot {
	v := math.Float64frombits((*atomic.Uint64)(g).Load())
	return GaugeFloat64Snapshot(v)
}

// Update updates the gauge's value.
func (g *GaugeFloat64) Update(v float64) {
	(*atomic.Uint64)(g).Store(math.Float64bits(v))
}
