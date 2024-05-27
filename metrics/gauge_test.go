package metrics

import (
	"sync"
	"testing"
)

func BenchmarkGauge(b *testing.B) {
	g := NewGauge()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.Update(int64(i))
	}
}

func BenchmarkGaugeIncDecParallel(b *testing.B) {
	g := NewGauge()
	b.ResetTimer()
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			for i := 0; i < b.N; i++ {
				g.Inc(1)
			}
			wg.Done()
		}()
		wg.Add(1)
		go func() {
			for i := 0; i < b.N; i++ {
				g.Dec(1)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	if have, want := g.Snapshot().Value(), int64(0); have != want {
		b.Fatalf("have %d want %d", have, want)
	}
}

func TestGaugeUpdateIfGt(t *testing.T) {
	g := NewGauge()
	g.Update(int64(47))
	g.UpdateIfGt(int64(0))
	if v := g.Snapshot().Value(); v != 47 {
		t.Errorf("g.Value(): 47 != %v\n", v)
	}
	g.UpdateIfGt(int64(58))
	if v := g.Snapshot().Value(); v != 58 {
		t.Errorf("g.Value(): 58 != %v\n", v)
	}
}

func TestGaugeUpdateIfGtParallel(t *testing.T) {
	g := NewGauge()
	g.Update(int64(45))
	if v := g.Snapshot().Value(); v != 45 {
		t.Errorf("g.Value(): 45 != %v\n", v)
	}
	var wg sync.WaitGroup
	for i := 50; i >= 40; i-- {
		wg.Add(1)
		go func(i int) {
			g.UpdateIfGt(int64(i))
			wg.Done()
		}(i)
	}
	wg.Wait()
	if v := g.Snapshot().Value(); v != 50 {
		t.Errorf("g.Value(): 50 != %v\n", v)
	}
}

func TestGaugeSnapshot(t *testing.T) {
	g := NewGauge()
	g.Update(int64(47))
	snapshot := g.Snapshot()
	g.Update(int64(0))
	if v := snapshot.Value(); v != 47 {
		t.Errorf("g.Value(): 47 != %v\n", v)
	}
}

func TestGetOrRegisterGauge(t *testing.T) {
	r := NewRegistry()
	NewRegisteredGauge("foo", r).Update(47)
	if g := GetOrRegisterGauge("foo", r); g.Snapshot().Value() != 47 {
		t.Fatal(g)
	}
}
