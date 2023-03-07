package metrics

import (
	"sync/atomic"
)

// Counters hold an int64 value that can be incremented and decremented.
type Counter interface {
	Clear()
	Count() int64
	Dec(int64)
	Inc(int64)
	Snapshot() Counter
}

// GetOrRegisterCounter returns an existing Counter or constructs and registers
// a new StandardCounter.
func GetOrRegisterCounter(name string, r Registry) Counter {
	if nil == r {
		r = DefaultRegistry
	}
	return r.GetOrRegister(name, NewCounter).(Counter)
}

// GetOrRegisterCounterForced returns an existing Counter or constructs and registers a
// new Counter no matter the global switch is enabled or not.
// Be sure to unregister the counter from the registry once it is of no use to
// allow for garbage collection.
func GetOrRegisterCounterForced(name string, r Registry) Counter {
	if nil == r {
		r = DefaultRegistry
	}
	return r.GetOrRegister(name, NewCounterForced).(Counter)
}

// NewCounter constructs a new StandardCounter.
func NewCounter() Counter {
	if !Enabled {
		return NilCounter{}
	}
	return &StandardCounter{}
}

// NewCounterForced constructs a new StandardCounter and returns it no matter if
// the global switch is enabled or not.
func NewCounterForced() Counter {
	return &StandardCounter{}
}

// NewRegisteredCounter constructs and registers a new StandardCounter.
func NewRegisteredCounter(name string, r Registry) Counter {
	c := NewCounter()
	if nil == r {
		r = DefaultRegistry
	}
	r.Register(name, c)
	return c
}

// NewRegisteredCounterForced constructs and registers a new StandardCounter
// and launches a goroutine no matter the global switch is enabled or not.
// Be sure to unregister the counter from the registry once it is of no use to
// allow for garbage collection.
func NewRegisteredCounterForced(name string, r Registry) Counter {
	c := NewCounterForced()
	if nil == r {
		r = DefaultRegistry
	}
	r.Register(name, c)
	return c
}

// CounterSnapshot is a read-only copy of another Counter.
type CounterSnapshot int64

// Clear panics.
func (CounterSnapshot) Clear() {
	panic("Clear called on a CounterSnapshot")
}

// Count returns the count at the time the snapshot was taken.
func (c CounterSnapshot) Count() int64 { return int64(c) }

// Dec panics.
func (CounterSnapshot) Dec(int64) {
	panic("Dec called on a CounterSnapshot")
}

// Inc panics.
func (CounterSnapshot) Inc(int64) {
	panic("Inc called on a CounterSnapshot")
}

// Snapshot returns the snapshot.
func (c CounterSnapshot) Snapshot() Counter { return c }

// NilCounter is a no-op Counter.
type NilCounter struct{}

// Clear is a no-op.
func (NilCounter) Clear() {}

// Count is a no-op.
func (NilCounter) Count() int64 { return 0 }

// Dec is a no-op.
func (NilCounter) Dec(i int64) {}

// Inc is a no-op.
func (NilCounter) Inc(i int64) {}

// Snapshot is a no-op.
func (NilCounter) Snapshot() Counter { return NilCounter{} }

// StandardCounter is the standard implementation of a Counter and uses the
// sync/atomic package to manage a single int64 value.
type StandardCounter struct {
	count atomic.Int64
}

// Clear sets the counter to zero.
func (c *StandardCounter) Clear() {
	c.count.Store(0)
}

// Count returns the current count.
func (c *StandardCounter) Count() int64 {
	return c.count.Load()
}

// Dec decrements the counter by the given amount.
func (c *StandardCounter) Dec(i int64) {
	c.count.Add(-i)
}

// Inc increments the counter by the given amount.
func (c *StandardCounter) Inc(i int64) {
	c.count.Add(i)
}

// Snapshot returns a read-only copy of the counter.
func (c *StandardCounter) Snapshot() Counter {
	return CounterSnapshot(c.Count())
}
