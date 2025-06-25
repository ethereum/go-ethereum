package metrics

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestReadRuntimeValues(t *testing.T) {
	var v runtimeStats
	readRuntimeStats(&v)
	t.Logf("%+v", v)
}

func BenchmarkMetrics(b *testing.B) {
	var (
		r  = NewRegistry()
		c  = NewRegisteredCounter("counter", r)
		cf = NewRegisteredCounterFloat64("counterfloat64", r)
		g  = NewRegisteredGauge("gauge", r)
		gf = NewRegisteredGaugeFloat64("gaugefloat64", r)
		h  = NewRegisteredHistogram("histogram", r, NewUniformSample(100))
		m  = NewRegisteredMeter("meter", r)
		t  = NewRegisteredTimer("timer", r)
	)
	RegisterDebugGCStats(r)
	b.ResetTimer()
	var wg sync.WaitGroup
	wg.Add(128)
	for i := 0; i < 128; i++ {
		go func() {
			defer wg.Done()
			for i := 0; i < b.N; i++ {
				c.Inc(1)
				cf.Inc(1.0)
				g.Update(int64(i))
				gf.Update(float64(i))
				h.Update(int64(i))
				m.Mark(1)
				t.Update(1)
			}
		}()
	}
	wg.Wait()
}

func Example() {
	c := NewCounter()
	Register("money", c)
	c.Inc(17)

	// Threadsafe registration
	t := GetOrRegisterTimer("db.get.latency", nil)
	t.Time(func() { time.Sleep(10 * time.Millisecond) })
	t.Update(1)

	fmt.Println(c.Snapshot().Count())
	fmt.Println(t.Snapshot().Min())
	// Output: 17
	// 1
}
