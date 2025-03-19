package metrics

import (
	"testing"
	"time"
)

func BenchmarkMeter(b *testing.B) {
	m := NewMeter()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Mark(1)
	}
}
func TestMeter(t *testing.T) {
	m := NewMeter()
	m.Mark(47)
	if v := m.Snapshot().Count(); v != 47 {
		t.Fatalf("have %d want %d", v, 47)
	}
}
func TestGetOrRegisterMeter(t *testing.T) {
	r := NewRegistry()
	NewRegisteredMeter("foo", r).Mark(47)
	if m := GetOrRegisterMeter("foo", r).Snapshot(); m.Count() != 47 {
		t.Fatal(m.Count())
	}
}

func TestMeterDecay(t *testing.T) {
	m := newMeter()
	m.Mark(1)
	m.tick()
	rateMean := m.Snapshot().RateMean()
	time.Sleep(100 * time.Millisecond)
	m.tick()
	if m.Snapshot().RateMean() >= rateMean {
		t.Error("m.RateMean() didn't decrease")
	}
}

func TestMeterNonzero(t *testing.T) {
	m := NewMeter()
	m.Mark(3)
	if count := m.Snapshot().Count(); count != 3 {
		t.Errorf("m.Count(): 3 != %v\n", count)
	}
}

func TestMeterStop(t *testing.T) {
	l := len(arbiter.meters)
	m := NewMeter()
	if l+1 != len(arbiter.meters) {
		t.Errorf("arbiter.meters: %d != %d\n", l+1, len(arbiter.meters))
	}
	m.Stop()
	if l != len(arbiter.meters) {
		t.Errorf("arbiter.meters: %d != %d\n", l, len(arbiter.meters))
	}
}

func TestMeterZero(t *testing.T) {
	m := NewMeter().Snapshot()
	if count := m.Count(); count != 0 {
		t.Errorf("m.Count(): 0 != %v\n", count)
	}
}

func TestMeterRepeat(t *testing.T) {
	m := NewMeter()
	for i := 0; i < 101; i++ {
		m.Mark(int64(i))
	}
	if count := m.Snapshot().Count(); count != 5050 {
		t.Errorf("m.Count(): 5050 != %v\n", count)
	}
	for i := 0; i < 101; i++ {
		m.Mark(int64(i))
	}
	if count := m.Snapshot().Count(); count != 10100 {
		t.Errorf("m.Count(): 10100 != %v\n", count)
	}
}

func TestMeterTickerLifecycle(t *testing.T) {
	// Make sure metrics are disabled initially
	oldEnabled := metricsEnabled
	metricsEnabled = false
	defer func() {
		metricsEnabled = oldEnabled
	}()

	// Reset the arbiter
	oldArbiter := arbiter
	arbiter = newMeterTicker()
	defer func() {
		arbiter = oldArbiter
	}()

	// Create a new meter but metrics are disabled
	// This should not start the ticker
	m := NewMeter()
	if arbiter.started || arbiter.running {
		t.Error("Ticker started when metrics are disabled")
	}

	// Enable metrics, which should start the ticker
	metricsEnabled = true
	EnableMetricsTicking()

	// Check if the ticker is started
	arbiter.mu.RLock()
	started := arbiter.started
	running := arbiter.running
	arbiter.mu.RUnlock()

	if !started || !running {
		t.Error("Ticker not started when metrics are enabled")
	}

	// Stop the meter
	m.Stop()

	// Ticker should be stopped as there are no more meters
	arbiter.mu.RLock()
	running = arbiter.running
	arbiter.mu.RUnlock()

	if running {
		t.Error("Ticker still running after all meters are stopped")
	}

	// Disable metrics, which should stop the ticker
	metricsEnabled = false
	Disable()

	// Checking for any goroutine leaks would require a more complex test
	// with runtime.NumGoroutine() checks
}
