package metrics

import (
	"sync"
	"testing"
)

func BenchmarkCounter(b *testing.B) {
	c := NewCounter()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Inc(1)
	}
}

func BenchmarkCounterParallel(b *testing.B) {
	c := NewCounter()
	b.ResetTimer()
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			for i := 0; i < b.N; i++ {
				c.Inc(1)
			}
			wg.Done()
		}()
		wg.Add(1)
		go func() {
			for i := 0; i < b.N; i++ {
				c.Dec(1)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	if have, want := c.Snapshot().Count(), int64(0); have != want {
		b.Fatalf("have %d want %d", have, want)
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

func TestCounterDec1(t *testing.T) {
	c := NewCounter()
	c.Dec(1)
	if count := c.Snapshot().Count(); count != -1 {
		t.Errorf("c.Count(): -1 != %v\n", count)
	}
}

func TestCounterDec2(t *testing.T) {
	c := NewCounter()
	c.Dec(2)
	if count := c.Snapshot().Count(); count != -2 {
		t.Errorf("c.Count(): -2 != %v\n", count)
	}
}

func TestCounterInc1(t *testing.T) {
	c := NewCounter()
	c.Inc(1)
	if count := c.Snapshot().Count(); count != 1 {
		t.Errorf("c.Count(): 1 != %v\n", count)
	}
}

func TestCounterInc2(t *testing.T) {
	c := NewCounter()
	c.Inc(2)
	if count := c.Snapshot().Count(); count != 2 {
		t.Errorf("c.Count(): 2 != %v\n", count)
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

func TestCounterZero(t *testing.T) {
	c := NewCounter()
	if count := c.Snapshot().Count(); count != 0 {
		t.Errorf("c.Count(): 0 != %v\n", count)
	}
}

func TestGetOrRegisterCounter(t *testing.T) {
	r := NewRegistry()
	NewRegisteredCounter("foo", r).Inc(47)
	if c := GetOrRegisterCounter("foo", r).Snapshot(); c.Count() != 47 {
		t.Fatal(c)
	}
}
