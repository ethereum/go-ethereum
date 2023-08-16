package metrics

import (
	"math/rand"
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
		t.Fatal(m.Count())
	}
}

func TestMeterDecay(t *testing.T) {
	m := newStandardMeter()
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

func TestMeterSnapshot(t *testing.T) {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	m := NewMeter()
	m.Mark(r.Int63())

	// RateMean() updates every millisecond, so we test Count().
	if snapshot := m.Snapshot(); m.Count() != snapshot.Count() {
		t.Fatal(snapshot)
	}
}

func TestMeterZero(t *testing.T) {
	m := NewMeter()
	if count := m.Count(); count != 0 {
		t.Errorf("m.Count(): 0 != %v\n", count)
	}
}

func TestMeterRepeat(t *testing.T) {
	m := NewMeter()
	for i := 0; i < 101; i++ {
		m.Mark(int64(i))
	}
	if count := m.Count(); count != 5050 {
		t.Errorf("m.Count(): 5050 != %v\n", count)
	}
	for i := 0; i < 101; i++ {
		m.Mark(int64(i))
	}
	if count := m.Count(); count != 10100 {
		t.Errorf("m.Count(): 10100 != %v\n", count)
	}
}
