package metrics

import (
	"sync"
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
	return r.GetOrRegister(name, NewLockFreeMeterForced).(Meter)
}

// NewMeter constructs a new lockFreeMeter and launches a goroutine.
// Be sure to call Stop() once the meter is of no use to allow for garbage collection.
func NewMeter() Meter {
	if !Enabled {
		return NilMeter{}
	}
	return newLockFreeMeter()
}

// NewRegisteredMeter constructs and registers a new lockFreeMeter
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
	return NewLockFreeMeterForced(name, r)
}

// MeterSnapshot is a read-only copy of another Meter.
type MeterSnapshot struct {
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

// NewLockFreeRegisteredMeter constructs and registers a new StandardMeter
// and launches a goroutine.
// Be sure to unregister the meter from the registry once it is of no use to
// allow for garbage collection.
func NewLockFreeRegisteredMeter(name string, r Registry) Meter {
	c := NewLockFreeMeter()
	if nil == r {
		r = DefaultRegistry
	}
	r.Register(name, c)
	return c
}

// NewLockFreeMeter constructs a new LockFreeMeter and launches a goroutine.
func NewLockFreeMeter() Meter {
	if !Enabled {
		return NilMeter{}
	}
	return newLockFreeMeter()
}

// NewLockFreeMeterForced constructs and registers a new StandardMeter
// and launches a goroutine no matter the global switch is enabled or not.
// Be sure to unregister the meter from the registry once it is of no use to
// allow for garbage collection.
func NewLockFreeMeterForced(name string, r Registry) Meter {
	c := newLockFreeMeter()
	if nil == r {
		r = DefaultRegistry
	}
	r.Register(name, c)
	return c
}

func NewMeterForced() Meter {
	return newLockFreeMeter()
}

// LockFreeMeter is a lock free implementation of a Meter.
type LockFreeMeter struct {
	mu          sync.RWMutex
	snapshot    *MeterSnapshot
	a1, a5, a15 EWMA
	startTime   time.Time
	dataChan    chan int64
	stopChan    chan interface{}
	ticker      *time.Ticker
}

func newLockFreeMeter() *LockFreeMeter {
	meter := &LockFreeMeter{
		snapshot:  &MeterSnapshot{},
		a1:        NewEWMA1(),
		a5:        NewEWMA5(),
		a15:       NewEWMA15(),
		startTime: time.Now(),
		dataChan:  make(chan int64, 10),
		stopChan:  make(chan interface{}),
		ticker:    time.NewTicker(5e9),
	}
	go func() {
		for {
			select {
			case n := <-meter.dataChan:
				meter.mu.Lock()
				meter.snapshot.count += n
				meter.a1.Update(n)
				meter.a5.Update(n)
				meter.a15.Update(n)
				meter.updateSnapshot()
				meter.mu.Unlock()
			case <-meter.ticker.C:
				meter.tick()
			case <-meter.stopChan:
				close(meter.dataChan)
				meter.ticker.Stop()
				return
			}
		}
	}()
	return meter
}

// Stop stops the meter. If stop was called, Mark becomes a no-op
func (m *LockFreeMeter) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	close(m.stopChan)
}

// Count returns the number of events recorded.
func (m *LockFreeMeter) Count() int64 {
	m.mu.RLock()
	count := m.snapshot.count
	m.mu.RUnlock()
	return count
}

// Mark records the occurrence of n events.
func (m *LockFreeMeter) Mark(n int64) {
	select {
	case <-m.stopChan:
		return
	default:
		m.dataChan <- n
	}
}

// Rate1 returns the one-minute moving average rate of events per second.
func (m *LockFreeMeter) Rate1() float64 {
	m.mu.RLock()
	rate1 := m.snapshot.rate1
	m.mu.RUnlock()
	return rate1
}

// Rate5 returns the five-minute moving average rate of events per second.
func (m *LockFreeMeter) Rate5() float64 {
	m.mu.RLock()
	rate5 := m.snapshot.rate5
	m.mu.RUnlock()
	return rate5
}

// Rate15 returns the fifteen-minute moving average rate of events per second.
func (m *LockFreeMeter) Rate15() float64 {
	m.mu.RLock()
	rate15 := m.snapshot.rate15
	m.mu.RUnlock()
	return rate15
}

// RateMean returns the meter's mean rate of events per second.
func (m *LockFreeMeter) RateMean() float64 {
	m.mu.RLock()
	rateMean := m.snapshot.rateMean
	m.mu.RUnlock()
	return rateMean
}

// Snapshot returns a read-only copy of the meter.
func (m *LockFreeMeter) Snapshot() Meter {
	m.mu.RLock()
	snapshot := *m.snapshot
	m.mu.RUnlock()
	return &snapshot
}

func (m *LockFreeMeter) updateSnapshot() {
	// should run with write lock held on m.lock
	snapshot := m.snapshot
	snapshot.rate1 = m.a1.Rate()
	snapshot.rate5 = m.a5.Rate()
	snapshot.rate15 = m.a15.Rate()
	snapshot.rateMean = float64(snapshot.count) / time.Since(m.startTime).Seconds()
}

func (m *LockFreeMeter) tick() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.a1.Tick()
	m.a5.Tick()
	m.a15.Tick()
	m.updateSnapshot()
}
