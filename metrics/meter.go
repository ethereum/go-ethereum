package metrics

import (
	"math"
	"sync"
	"sync/atomic"
	"time"
)

// GetOrRegisterMeter returns an existing Meter or constructs and registers a
// new Meter.
// Be sure to unregister the meter from the registry once it is of no use to
// allow for garbage collection.
func GetOrRegisterMeter(name string, r Registry) *Meter {
	if r == nil {
		r = DefaultRegistry
	}
	return r.GetOrRegister(name, NewMeter).(*Meter)
}

// NewMeter constructs a new Meter and launches a goroutine.
// Be sure to call Stop() once the meter is of no use to allow for garbage collection.
func NewMeter() *Meter {
	m := newMeter()
	arbiter.add(m)
	return m
}

// NewInactiveMeter returns a meter but does not start any goroutines. This
// method is mainly intended for testing.
func NewInactiveMeter() *Meter {
	return newMeter()
}

// NewRegisteredMeter constructs and registers a new Meter
// and launches a goroutine.
// Be sure to unregister the meter from the registry once it is of no use to
// allow for garbage collection.
func NewRegisteredMeter(name string, r Registry) *Meter {
	return GetOrRegisterMeter(name, r)
}

// MeterSnapshot is a read-only copy of the meter's internal values.
type MeterSnapshot struct {
	count                          int64
	rate1, rate5, rate15, rateMean float64
}

// Count returns the count of events at the time the snapshot was taken.
func (m *MeterSnapshot) Count() int64 { return m.count }

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

// Meter count events to produce exponentially-weighted moving average rates
// at one-, five-, and fifteen-minutes and a mean rate.
type Meter struct {
	count     atomic.Int64
	uncounted atomic.Int64 // not yet added to the EWMAs
	rateMean  atomic.Uint64

	a1, a5, a15 *EWMA
	startTime   time.Time
	stopped     atomic.Bool
}

func newMeter() *Meter {
	return &Meter{
		a1:        NewEWMA1(),
		a5:        NewEWMA5(),
		a15:       NewEWMA15(),
		startTime: time.Now(),
	}
}

// Stop stops the meter, Mark() will be a no-op if you use it after being stopped.
func (m *Meter) Stop() {
	if stopped := m.stopped.Swap(true); !stopped {
		arbiter.remove(m)
	}
}

// Mark records the occurrence of n events.
func (m *Meter) Mark(n int64) {
	m.uncounted.Add(n)
}

// Snapshot returns a read-only copy of the meter.
func (m *Meter) Snapshot() *MeterSnapshot {
	return &MeterSnapshot{
		count:    m.count.Load() + m.uncounted.Load(),
		rate1:    m.a1.Snapshot().Rate(),
		rate5:    m.a5.Snapshot().Rate(),
		rate15:   m.a15.Snapshot().Rate(),
		rateMean: math.Float64frombits(m.rateMean.Load()),
	}
}

func (m *Meter) tick() {
	// Take the uncounted values, add to count
	n := m.uncounted.Swap(0)
	count := m.count.Add(n)
	m.rateMean.Store(math.Float64bits(float64(count) / time.Since(m.startTime).Seconds()))
	// Update the EWMA's internal state
	m.a1.Update(n)
	m.a5.Update(n)
	m.a15.Update(n)
	// And trigger them to calculate the rates
	m.a1.tick()
	m.a5.tick()
	m.a15.tick()
}

var arbiter = meterTicker{meters: make(map[*Meter]struct{})}

// meterTicker ticks meters every 5s from a single goroutine.
// meters are references in a set for future stopping.
type meterTicker struct {
	mu sync.RWMutex

	once   sync.Once
	meters map[*Meter]struct{}
}

// add a *Meter to the arbiter
func (ma *meterTicker) add(m *Meter) {
	ma.mu.Lock()
	defer ma.mu.Unlock()
	ma.meters[m] = struct{}{}
}

// remove removes a meter from the set of ticked meters.
func (ma *meterTicker) remove(m *Meter) {
	ma.mu.Lock()
	delete(ma.meters, m)
	ma.mu.Unlock()
}

// loop ticks meters on a 5-second interval.
func (ma *meterTicker) loop() {
	ticker := time.NewTicker(5 * time.Second)
	for range ticker.C {
		if !metricsEnabled {
			continue
		}
		ma.mu.RLock()
		for meter := range ma.meters {
			meter.tick()
		}
		ma.mu.RUnlock()
	}
}

// startMeterTickerLoop will start the arbiter ticker.
func startMeterTickerLoop() {
	arbiter.once.Do(func() { go arbiter.loop() })
}
