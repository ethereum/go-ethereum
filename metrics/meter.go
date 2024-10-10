package metrics

import (
	"sync"
	"time"
)

type MeterSnapshot interface {
	Count() int64
	Rate1() float64
	Rate5() float64
	Rate15() float64
	RateMean() float64
}

// Meters count events to produce exponentially-weighted moving average rates
// at one-, five-, and fifteen-minutes and a mean rate.
type Meter interface {
	Mark(int64)
	Snapshot() MeterSnapshot
}

// GetOrRegisterMeter returns an existing Meter or constructs and registers a
// new StandardMeter.
func GetOrRegisterMeter(name string, r Registry) Meter {
	if nil == r {
		r = DefaultRegistry
	}
	return r.GetOrRegister(name, NewMeter).(Meter)
}

// NewMeter constructs a new StandardMeter and launches a goroutine.
func NewMeter() Meter {
	if !Enabled {
		return NilMeter{}
	}
	return newStandardMeter()
}

// NewRegisteredMeter constructs and registers a new StandardMeter
// and launches a goroutine.
func NewRegisteredMeter(name string, r Registry) Meter {
	return GetOrRegisterMeter(name, r)
}

// meterSnapshot is a read-only copy of the meter's internal values.
type meterSnapshot struct {
	count                          int64
	rate1, rate5, rate15, rateMean float64
}

// Count returns the count of events at the time the snapshot was taken.
func (m *meterSnapshot) Count() int64 { return m.count }

// Rate1 returns the one-minute moving average rate of events per second at the
// time the snapshot was taken.
func (m *meterSnapshot) Rate1() float64 { return m.rate1 }

// Rate5 returns the five-minute moving average rate of events per second at
// the time the snapshot was taken.
func (m *meterSnapshot) Rate5() float64 { return m.rate5 }

// Rate15 returns the fifteen-minute moving average rate of events per second
// at the time the snapshot was taken.
func (m *meterSnapshot) Rate15() float64 { return m.rate15 }

// RateMean returns the meter's mean rate of events per second at the time the
// snapshot was taken.
func (m *meterSnapshot) RateMean() float64 { return m.rateMean }

// NilMeter is a no-op Meter.
type NilMeter struct{}

func (NilMeter) Count() int64            { return 0 }
func (NilMeter) Mark(n int64)            {}
func (NilMeter) Snapshot() MeterSnapshot { return (*emptySnapshot)(nil) }
func (NilMeter) Stop()                   {}

// StandardMeter is the standard implementation of a Meter.
type StandardMeter struct {
	count       int64
	uncounted   int64
	a1, a5, a15 EWMA
	startTime   time.Time
	lastMark    time.Time
	mutex       sync.Mutex
}

func newStandardMeter() *StandardMeter {
	return &StandardMeter{
		a1:        NewEWMA1(),
		a5:        NewEWMA5(),
		a15:       NewEWMA15(),
		startTime: time.Now(),
		lastMark:  time.Now(),
	}
}

// Mark records the occurrence of n events.
func (m *StandardMeter) Mark(n int64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Synchronize rateMean so that it's only updated after
	// a sampling period elapses.
	if elapsed := time.Since(m.lastMark); elapsed >= 5*time.Second {
		m.lastMark = m.lastMark.Add(elapsed)
		m.count += m.uncounted
		m.uncounted = 0
	}

	m.uncounted += n
	m.a1.Update(n)
	m.a5.Update(n)
	m.a15.Update(n)
}

// Snapshot returns a read-only copy of the meter.
func (m *StandardMeter) Snapshot() MeterSnapshot {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if elapsed := time.Since(m.lastMark); elapsed >= 5*time.Second {
		m.lastMark = m.lastMark.Add(elapsed)
		m.count += m.uncounted
		m.uncounted = 0
	}
	rateMean := float64(m.count) / (1 + time.Since(m.startTime).Seconds())
	return &meterSnapshot{
		count:    m.count + m.uncounted,
		rate1:    m.a1.Snapshot().Rate(),
		rate5:    m.a5.Snapshot().Rate(),
		rate15:   m.a15.Snapshot().Rate(),
		rateMean: rateMean,
	}
}

// used to elapse time in unit tests.
func (m *StandardMeter) addToTimestamp(d time.Duration) {
	m.startTime = m.startTime.Add(d)
	m.lastMark = m.lastMark.Add(d)
	a1, _ := m.a1.(*StandardEWMA)
	a1.addToTimestamp(d)

	a5, _ := m.a5.(*StandardEWMA)
	a5.addToTimestamp(d)

	a15, _ := m.a15.(*StandardEWMA)
	a15.addToTimestamp(d)
}
