package metrics

import (
	"math"
	"sync/atomic"
)

// GetOrRegisterCounterFloat64 returns an existing *CounterFloat64 or constructs and registers
// a new CounterFloat64.
func GetOrRegisterCounterFloat64(name string, r Registry) *CounterFloat64 {
	if nil == r {
		r = DefaultRegistry
	}
	return r.GetOrRegister(name, NewCounterFloat64).(*CounterFloat64)
}

// NewCounterFloat64 constructs a new CounterFloat64.
func NewCounterFloat64() *CounterFloat64 {
	return new(CounterFloat64)
}

// NewRegisteredCounterFloat64 constructs and registers a new CounterFloat64.
func NewRegisteredCounterFloat64(name string, r Registry) *CounterFloat64 {
	c := NewCounterFloat64()
	if r == nil {
		r = DefaultRegistry
	}
	r.Register(name, c)
	return c
}

// CounterFloat64Snapshot is a read-only copy of a float64 counter.
type CounterFloat64Snapshot float64

// Count returns the value at the time the snapshot was taken.
func (c CounterFloat64Snapshot) Count() float64 { return float64(c) }

// CounterFloat64 holds a float64 value that can be incremented and decremented.
type CounterFloat64 atomic.Uint64

// Clear sets the counter to zero.
func (c *CounterFloat64) Clear() {
	(*atomic.Uint64)(c).Store(0)
}

// Dec decrements the counter by the given amount.
func (c *CounterFloat64) Dec(v float64) {
	atomicAddFloat((*atomic.Uint64)(c), -v)
}

// Inc increments the counter by the given amount.
func (c *CounterFloat64) Inc(v float64) {
	atomicAddFloat((*atomic.Uint64)(c), v)
}

// Snapshot returns a read-only copy of the counter.
func (c *CounterFloat64) Snapshot() CounterFloat64Snapshot {
	return CounterFloat64Snapshot(math.Float64frombits((*atomic.Uint64)(c).Load()))
}

func atomicAddFloat(fbits *atomic.Uint64, v float64) {
	for {
		loadedBits := fbits.Load()
		newBits := math.Float64bits(math.Float64frombits(loadedBits) + v)
		if fbits.CompareAndSwap(loadedBits, newBits) {
			break
		}
	}
}
