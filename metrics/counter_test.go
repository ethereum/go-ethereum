package metrics

import "testing"

func BenchmarkCounter(b *testing.B) {
	c := NewCounter()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Inc(1)
	}
}

func TestCounterClear(t *testing.T) {
	c := NewCounter()
	c.Inc(1)
	c.Clear()
	if count := c.Snapshot().Count(); count != 0 {
		t.Errorf("c.Count(): 0 != %v\n", count)
	}
}

func TestCounter(t *testing.T) {
	c := NewCounter()
	if count := c.Snapshot().Count(); count != 0 {
		t.Errorf("wrong count: %v", count)
	}
	c.Dec(1)
	if count := c.Snapshot().Count(); count != -1 {
		t.Errorf("wrong count: %v", count)
	}
	c.Dec(2)
	if count := c.Snapshot().Count(); count != -3 {
		t.Errorf("wrong count: %v", count)
	}
	c.Inc(1)
	if count := c.Snapshot().Count(); count != -2 {
		t.Errorf("wrong count: %v", count)
	}
	c.Inc(2)
	if count := c.Snapshot().Count(); count != 0 {
		t.Errorf("wrong count: %v", count)
	}
}

func TestCounterSnapshot(t *testing.T) {
	c := NewCounter()
	c.Inc(1)
	snapshot := c.Snapshot()
	c.Inc(1)
	if count := snapshot.Count(); count != 1 {
		t.Errorf("c.Count(): 1 != %v\n", count)
	}
}

func TestGetOrRegisterCounter(t *testing.T) {
	r := NewRegistry()
	NewRegisteredCounter("foo", r).Inc(47)
	if c := GetOrRegisterCounter("foo", r).Snapshot(); c.Count() != 47 {
		t.Fatal(c)
	}
}
