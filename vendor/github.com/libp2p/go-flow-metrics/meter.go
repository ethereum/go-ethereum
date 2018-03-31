package flow

import (
	"fmt"
	"sync/atomic"
)

// Snapshot is a rate/total snapshot.
type Snapshot struct {
	Rate  float64
	Total uint64
}

func (s Snapshot) String() string {
	return fmt.Sprintf("%d (%f/s)", s.Total, s.Rate)
}

// Meter is a meter for monitoring a flow.
type Meter struct {
	accumulator uint64

	// Take lock.
	snapshot Snapshot
}

// Mark updates the total.
func (m *Meter) Mark(count uint64) {
	if count > 0 && atomic.AddUint64(&m.accumulator, count) == count {
		// I'm the first one to bump this above 0.
		// Register it.
		globalSweeper.Register(m)
	}
}

// Snapshot gets a consistent snapshot of the total and rate.
func (m *Meter) Snapshot() Snapshot {
	globalSweeper.mutex.RLock()
	defer globalSweeper.mutex.RUnlock()
	return m.snapshot
}

func (m *Meter) String() string {
	return m.Snapshot().String()
}
