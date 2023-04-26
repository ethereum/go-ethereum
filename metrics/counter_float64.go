package metrics

import (
	"math"
	"sync/atomic"
)

// CounterFloat64 holds a float64 value that can be incremented and decremented.
type CounterFloat64 interface {
	Clear()
	Count() float64
	Dec(float64)
	Inc(float64)
	Snapshot() CounterFloat64
}

// GetOrRegisterCounterFloat64 returns an existing CounterFloat64 or constructs and registers
// a new StandardCounterFloat64.
func GetOrRegisterCounterFloat64(name string, r Registry) CounterFloat64 {
	if nil == r {
		r = DefaultRegistry
	}
	return r.GetOrRegister(name, NewCounterFloat64).(CounterFloat64)
}

// GetOrRegisterCounterFloat64Forced returns an existing CounterFloat64 or constructs and registers a
// new CounterFloat64 no matter the global switch is enabled or not.
// Be sure to unregister the counter from the registry once it is of no use to
// allow for garbage collection.
func GetOrRegisterCounterFloat64Forced(name string, r Registry) CounterFloat64 {
	if nil == r {
		r = DefaultRegistry
	}
	return r.GetOrRegister(name, NewCounterFloat64Forced).(CounterFloat64)
}

// NewCounterFloat64 constructs a new StandardCounterFloat64.
func NewCounterFloat64() CounterFloat64 {
	if !Enabled {
		return NilCounterFloat64{}
	}
	return &StandardCounterFloat64{}
}

// NewCounterFloat64Forced constructs a new StandardCounterFloat64 and returns it no matter if
// the global switch is enabled or not.
func NewCounterFloat64Forced() CounterFloat64 {
	return &StandardCounterFloat64{}
}

// NewRegisteredCounterFloat64 constructs and registers a new StandardCounterFloat64.
func NewRegisteredCounterFloat64(name string, r Registry) CounterFloat64 {
	c := NewCounterFloat64()
	if nil == r {
		r = DefaultRegistry
	}
	r.Register(name, c)
	return c
}

// NewRegisteredCounterFloat64Forced constructs and registers a new StandardCounterFloat64
// and launches a goroutine no matter the global switch is enabled or not.
// Be sure to unregister the counter from the registry once it is of no use to
// allow for garbage collection.
func NewRegisteredCounterFloat64Forced(name string, r Registry) CounterFloat64 {
	c := NewCounterFloat64Forced()
	if nil == r {
		r = DefaultRegistry
	}
	r.Register(name, c)
	return c
}

// CounterFloat64Snapshot is a read-only copy of another CounterFloat64.
type CounterFloat64Snapshot float64

// Clear panics.
func (CounterFloat64Snapshot) Clear() {
	panic("Clear called on a CounterFloat64Snapshot")
}

// Count returns the value at the time the snapshot was taken.
func (c CounterFloat64Snapshot) Count() float64 { return float64(c) }

// Dec panics.
func (CounterFloat64Snapshot) Dec(float64) {
	panic("Dec called on a CounterFloat64Snapshot")
}

// Inc panics.
func (CounterFloat64Snapshot) Inc(float64) {
	panic("Inc called on a CounterFloat64Snapshot")
}

// Snapshot returns the snapshot.
func (c CounterFloat64Snapshot) Snapshot() CounterFloat64 { return c }

// NilCounterFloat64 is a no-op CounterFloat64.
type NilCounterFloat64 struct{}

// Clear is a no-op.
func (NilCounterFloat64) Clear() {}

// Count is a no-op.
func (NilCounterFloat64) Count() float64 { return 0.0 }

// Dec is a no-op.
func (NilCounterFloat64) Dec(i float64) {}

// Inc is a no-op.
func (NilCounterFloat64) Inc(i float64) {}

// Snapshot is a no-op.
func (NilCounterFloat64) Snapshot() CounterFloat64 { return NilCounterFloat64{} }

// StandardCounterFloat64 is the standard implementation of a CounterFloat64 and uses the
// atomic to manage a single float64 value.
type StandardCounterFloat64 struct {
	floatBits atomic.Uint64
}

// Clear sets the counter to zero.
func (c *StandardCounterFloat64) Clear() {
	c.floatBits.Store(0)
}

// Count returns the current value.
func (c *StandardCounterFloat64) Count() float64 {
	return math.Float64frombits(c.floatBits.Load())
}

// Dec decrements the counter by the given amount.
func (c *StandardCounterFloat64) Dec(v float64) {
	atomicAddFloat(&c.floatBits, -v)
}

// Inc increments the counter by the given amount.
func (c *StandardCounterFloat64) Inc(v float64) {
	atomicAddFloat(&c.floatBits, v)
}

// Snapshot returns a read-only copy of the counter.
func (c *StandardCounterFloat64) Snapshot() CounterFloat64 {
	return CounterFloat64Snapshot(c.Count())
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
