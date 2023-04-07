package metrics

import (
	"sync"
	"testing"
)

func BenchmarkGaugeFloat64(b *testing.B) {
	g := NewGaugeFloat64()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.Update(float64(i))
	}
}

func BenchmarkGaugeFloat64Parallel(b *testing.B) {
	c := NewGaugeFloat64()
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			for i := 0; i < b.N; i++ {
				c.Update(float64(i))
			}
			wg.Done()
		}()
	}
	wg.Wait()
	if have, want := c.Value(), float64(b.N-1); have != want {
		b.Fatalf("have %f want %f", have, want)
	}
}

func TestGaugeFloat64(t *testing.T) {
	g := NewGaugeFloat64()
	g.Update(47.0)
	if v := g.Value(); 47.0 != v {
		t.Errorf("g.Value(): 47.0 != %v\n", v)
	}
}

func TestGaugeFloat64Snapshot(t *testing.T) {
	g := NewGaugeFloat64()
	g.Update(47.0)
	snapshot := g.Snapshot()
	g.Update(float64(0))
	if v := snapshot.Value(); 47.0 != v {
		t.Errorf("g.Value(): 47.0 != %v\n", v)
	}
}

func TestGetOrRegisterGaugeFloat64(t *testing.T) {
	r := NewRegistry()
	NewRegisteredGaugeFloat64("foo", r).Update(47.0)
	t.Logf("registry: %v", r)
	if g := GetOrRegisterGaugeFloat64("foo", r); 47.0 != g.Value() {
		t.Fatal(g)
	}
}

func TestFunctionalGaugeFloat64(t *testing.T) {
	var counter float64
	fg := NewFunctionalGaugeFloat64(func() float64 {
		counter++
		return counter
	})
	fg.Value()
	fg.Value()
	if counter != 2 {
		t.Error("counter != 2")
	}
}

func TestGetOrRegisterFunctionalGaugeFloat64(t *testing.T) {
	r := NewRegistry()
	NewRegisteredFunctionalGaugeFloat64("foo", r, func() float64 { return 47 })
	if g := GetOrRegisterGaugeFloat64("foo", r); g.Value() != 47 {
		t.Fatal(g)
	}
}
