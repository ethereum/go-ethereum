package metrics

import (
	"sync"
	"sync/atomic"
	"time"
)

// Meters count events to produce exponentially-weighted moving average rates
// at one-, five-, and fifteen-minutes and a mean rate.
type Meter interface {
	Count() int64
	Mark(int64)
	Rate1() float64
	Rate5() float64
	Rate15() float64
	RateMean() float64
	Snapshot() Meter
	Stop()
}

// GetOrRegisterMeter returns an existing Meter or constructs and registers a
// new StandardMeter.
// Be sure to unregister the meter from the registry once it is of no use to
// allow for garbage collection.
func GetOrRegisterMeter(name string, r Registry) Meter {
	if nil == r {
		r = DefaultRegistry
	}
	return r.GetOrRegister(name, NewMeter).(Meter)
}

// GetOrRegisterMeterForced returns an existing Meter or constructs and registers a
// new StandardMeter no matter the global switch is enabled or not.
// Be sure to unregister the meter from the registry once it is of no use to
// allow for garbage collection.
func GetOrRegisterMeterForced(name string, r Registry) Meter {
	if nil == r {
		r = DefaultRegistry
	}
	return r.GetOrRegister(name, NewMeterForced).(Meter)
}

// NewMeter constructs a new StandardMeter and launches a goroutine.
// Be sure to call Stop() once the meter is of no use to allow for garbage collection.
func NewMeter() Meter {
	if !Enabled {
		return NilMeter{}
	}
	m := newStandardMeter()
	arbiter.Lock()
	defer arbiter.Unlock()
	arbiter.meters[m] = struct{}{}
	if !arbiter.started {
		arbiter.started = true
		go arbiter.tick()
	}
	return m
}

// NewMeterForced constructs a new StandardMeter and launches a goroutine no matter
// the global switch is enabled or not.
// Be sure to call Stop() once the meter is of no use to allow for garbage collection.
func NewMeterForced() Meter {
	m := newStandardMeter()
	arbiter.Lock()
	defer arbiter.Unlock()
	arbiter.meters[m] = struct{}{}
	if !arbiter.started {
		arbiter.started = true
		go arbiter.tick()
	}
	return m
}

// NewRegisteredMeter constructs and registers a new StandardMeter
// and launches a goroutine.
// Be sure to unregister the meter from the registry once it is of no use to
// allow for garbage collection.
func NewRegisteredMeter(name string, r Registry) Meter {
	c := NewMeter()
	if nil == r {
		r = DefaultRegistry
	}
	r.Register(name, c)
	return c
}

// NewRegisteredMeterForced constructs and registers a new StandardMeter
// and launches a goroutine no matter the global switch is enabled or not.
// Be sure to unregister the meter from the registry once it is of no use to
// allow for garbage collection.
func NewRegisteredMeterForced(name string, r Registry) Meter {
	c := NewMeterForced()
	if nil == r {
		r = DefaultRegistry
	}
	r.Register(name, c)
	return c
}

// MeterSnapshot is a read-only copy of another Meter.
type MeterSnapshot struct {
	temp                           atomic.Int64
	count                          int64
	rate1, rate5, rate15, rateMean float64
}

// Count returns the count of events at the time the snapshot was taken.
func (m *MeterSnapshot) Count() int64 { return m.count }

// Mark panics.
func (*MeterSnapshot) Mark(n int64) {
	panic("Mark called on a MeterSnapshot")
}

// Rate1 returns the one-minute moving average rate of events per second at the
// time the snapshot was taken.
func (m *MeterSnapshot) Rate1() float64 { return m.rate1 }

// Rate5 returns the five-minute moving average rate of events per second at
// the time the snapshot was taken.
func (m *MeterSnapshot) Rate5() float64 { return m.rate5 }

// Rate15 returns the fifteen-minute moving average rate of events per second
// at the time the snapshot was taken.
func (m *MeterSnapshot) Rate15() float64 { return m.rate15 }

// RateMean returns the meter's mean rate of events per second at the time the
// snapshot was taken.
func (m *MeterSnapshot) RateMean() float64 { return m.rateMean }

// Snapshot returns the snapshot.
func (m *MeterSnapshot) Snapshot() Meter { return m }

// Stop is a no-op.
func (m *MeterSnapshot) Stop() {}

// NilMeter is a no-op Meter.
type NilMeter struct{}

// Count is a no-op.
func (NilMeter) Count() int64 { return 0 }

// Mark is a no-op.
func (NilMeter) Mark(n int64) {}

// Rate1 is a no-op.
func (NilMeter) Rate1() float64 { return 0.0 }

// Rate5 is a no-op.
func (NilMeter) Rate5() float64 { return 0.0 }

// Rate15 is a no-op.
func (NilMeter) Rate15() float64 { return 0.0 }

// RateMean is a no-op.
func (NilMeter) RateMean() float64 { return 0.0 }

// Snapshot is a no-op.
func (NilMeter) Snapshot() Meter { return NilMeter{} }

// Stop is a no-op.
func (NilMeter) Stop() {}

// StandardMeter is the standard implementation of a Meter.
type StandardMeter struct {
	lock        sync.RWMutex
	snapshot    *MeterSnapshot
	a1, a5, a15 EWMA
	startTime   time.Time
	stopped     atomic.Bool
}

func newStandardMeter() *StandardMeter {
	return &StandardMeter{
		snapshot:  &MeterSnapshot{},
		a1:        NewEWMA1(),
		a5:        NewEWMA5(),
		a15:       NewEWMA15(),
		startTime: time.Now(),
	}
}

// Stop stops the meter, Mark() will be a no-op if you use it after being stopped.
func (m *StandardMeter) Stop() {
	stopped := m.stopped.Swap(true)
	if !stopped {
		arbiter.Lock()
		delete(arbiter.meters, m)
		arbiter.Unlock()
	}
}

// Count returns the number of events recorded.
// It updates the meter to be as accurate as possible
func (m *StandardMeter) Count() int64 {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.updateMeter()
	return m.snapshot.count
}

// Mark records the occurrence of n events.
func (m *StandardMeter) Mark(n int64) {
	m.snapshot.temp.Add(n)
}

// Rate1 returns the one-minute moving average rate of events per second.
func (m *StandardMeter) Rate1() float64 {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.snapshot.rate1
}

// Rate5 returns the five-minute moving average rate of events per second.
func (m *StandardMeter) Rate5() float64 {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.snapshot.rate5
}

// Rate15 returns the fifteen-minute moving average rate of events per second.
func (m *StandardMeter) Rate15() float64 {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.snapshot.rate15
}

// RateMean returns the meter's mean rate of events per second.
func (m *StandardMeter) RateMean() float64 {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.snapshot.rateMean
}

// Snapshot returns a read-only copy of the meter.
func (m *StandardMeter) Snapshot() Meter {
	m.lock.RLock()
	snapshot := MeterSnapshot{
		count:    m.snapshot.count,
		rate1:    m.snapshot.rate1,
		rate5:    m.snapshot.rate5,
		rate15:   m.snapshot.rate15,
		rateMean: m.snapshot.rateMean,
	}
	snapshot.temp.Store(m.snapshot.temp.Load())
	m.lock.RUnlock()
	return &snapshot
}

func (m *StandardMeter) updateSnapshot() {
	// should run with write lock held on m.lock
	snapshot := m.snapshot
	snapshot.rate1 = m.a1.Rate()
	snapshot.rate5 = m.a5.Rate()
	snapshot.rate15 = m.a15.Rate()
	snapshot.rateMean = float64(snapshot.count) / time.Since(m.startTime).Seconds()
}

func (m *StandardMeter) updateMeter() {
	// should only run with write lock held on m.lock
	n := m.snapshot.temp.Swap(0)
	m.snapshot.count += n
	m.a1.Update(n)
	m.a5.Update(n)
	m.a15.Update(n)
}

func (m *StandardMeter) tick() {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.updateMeter()
	m.a1.Tick()
	m.a5.Tick()
	m.a15.Tick()
	m.updateSnapshot()
}

// meterArbiter ticks meters every 5s from a single goroutine.
// meters are references in a set for future stopping.
type meterArbiter struct {
	sync.RWMutex
	started bool
	meters  map[*StandardMeter]struct{}
	ticker  *time.Ticker
}

var arbiter = meterArbiter{ticker: time.NewTicker(5 * time.Second), meters: make(map[*StandardMeter]struct{})}

// Ticks meters on the scheduled interval
func (ma *meterArbiter) tick() {
	for range ma.ticker.C {
		ma.tickMeters()
	}
}

func (ma *meterArbiter) tickMeters() {
	ma.RLock()
	defer ma.RUnlock()
	for meter := range ma.meters {
		meter.tick()
	}
}
