package metrics

import "testing"

func BenchmarkCounterFloat64(b *testing.B) {
	c := NewCounterFloat64()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Inc(1.0)
	}
}

func TestCounterFloat64Clear(t *testing.T) {
	c := NewCounterFloat64()
	c.Inc(1.0)
	c.Clear()
	if count := c.Count(); count != 0 {
		t.Errorf("c.Count(): 0 != %v\n", count)
	}
}

func TestCounterFloat64Dec1(t *testing.T) {
	c := NewCounterFloat64()
	c.Dec(1.0)
	if count := c.Count(); count != -1.0 {
		t.Errorf("c.Count(): -1.0 != %v\n", count)
	}
}

func TestCounterFloat64Dec2(t *testing.T) {
	c := NewCounterFloat64()
	c.Dec(2.0)
	if count := c.Count(); count != -2.0 {
		t.Errorf("c.Count(): -2.0 != %v\n", count)
	}
}

func TestCounterFloat64Inc1(t *testing.T) {
	c := NewCounterFloat64()
	c.Inc(1.0)
	if count := c.Count(); count != 1.0 {
		t.Errorf("c.Count(): 1.0 != %v\n", count)
	}
}

func TestCounterFloat64Inc2(t *testing.T) {
	c := NewCounterFloat64()
	c.Inc(2.0)
	if count := c.Count(); count != 2.0 {
		t.Errorf("c.Count(): 2.0 != %v\n", count)
	}
}

func TestCounterFloat64Snapshot(t *testing.T) {
	c := NewCounterFloat64()
	c.Inc(1.0)
	snapshot := c.Snapshot()
	c.Inc(1.0)
	if count := snapshot.Count(); count != 1.0 {
		t.Errorf("c.Count(): 1.0 != %v\n", count)
	}
}

func TestCounterFloat64Zero(t *testing.T) {
	c := NewCounterFloat64()
	if count := c.Count(); count != 0 {
		t.Errorf("c.Count(): 0 != %v\n", count)
	}
}

func TestGetOrRegisterCounterFloat64(t *testing.T) {
	r := NewRegistry()
	NewRegisteredCounterFloat64("foo", r).Inc(47.0)
	if c := GetOrRegisterCounterFloat64("foo", r); c.Count() != 47.0 {
		t.Fatal(c)
	}
}
