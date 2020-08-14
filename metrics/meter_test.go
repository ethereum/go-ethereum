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

func TestGetOrRegisterMeter(t *testing.T) {
	r := NewRegistry()
	NewRegisteredMeter("foo", r).Mark(47)
	if m := GetOrRegisterMeter("foo", r); m.Count() != 47 {
		t.Fatal(m)
	}
}

func TestMeterDecay(t *testing.T) {
	ma := meterArbiter{
		ticker: time.NewTicker(time.Millisecond),
		meters: make(map[*StandardMeter]struct{}),
	}
	defer ma.ticker.Stop()
	m := newStandardMeter()
	ma.meters[m] = struct{}{}
	go ma.tick()
	m.Mark(1)
	rateMean := m.RateMean()
	time.Sleep(100 * time.Millisecond)
	if m.RateMean() >= rateMean {
		t.Error("m.RateMean() didn't decrease")
	}
}

func TestMeterNonzero(t *testing.T) {
	m := NewMeter()
	m.Mark(3)
	if count := m.Count(); count != 3 {
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

func TestMeterSnapshot(t *testing.T) {
	m := NewMeter()
	m.Mark(1)
	if snapshot := m.Snapshot(); m.RateMean() != snapshot.RateMean() {
		t.Fatal(snapshot)
	}
}

func TestMeterZero(t *testing.T) {
	m := NewMeter()
	if count := m.Count(); count != 0 {
		t.Errorf("m.Count(): 0 != %v\n", count)
	}
}

func TestLockFreeMeterNonzero(t *testing.T) {
	m := NewLockFreeMeter()
	m.Mark(3)
	time.Sleep(100 * time.Millisecond)
	if count := m.Count(); count != 3 {
		t.Errorf("m.Count(): 3 != %v\n", count)
	}
}

func TestLockFreeMeterDecay(t *testing.T) {
	m := newLockFreeMeter()
	m.Mark(1)
	time.Sleep(100 * time.Millisecond)
	rateMean := m.RateMean()
	time.Sleep(5*time.Second + 1)
	if m.RateMean() >= rateMean {
		t.Errorf("m.RateMean() didn't decrease: %v %v", rateMean, m.RateMean())
	}
}

func TestLockFreeMeterStop(t *testing.T) {
	m := newLockFreeMeter()
	m.Stop()
	if out := <-m.dataChan; out != 0 {
		t.Error("m.Stop() not properly closes channel")
	}
}
