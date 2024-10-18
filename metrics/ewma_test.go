package metrics

import (
	"math"
	"math/rand"
	"sync"
	"testing"
	"time"
)

func BenchmarkEWMA(b *testing.B) {
	a := NewEWMA1()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.Update(1)
	}
}

func BenchmarkEWMAParallel(b *testing.B) {
	a := NewEWMA1()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			a.Update(1)
		}
	})
}

// exercise race detector.
func TestEWMAConcurrency(t *testing.T) {
	a := NewEWMA1()
	wg := &sync.WaitGroup{}
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(ewma EWMA, wg *sync.WaitGroup) {
			a.Update(rand.Int63())
			wg.Done()
		}(a, wg)
	}
	wg.Wait()
}

func testEWMA(t *testing.T, alpha float64) {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	a := NewEWMA(alpha, time.Second)

	// Base case.

	if rate := a.Snapshot().Rate(); rate != 0 {
		t.Errorf("(A) Base Case a.rate(): 0 != %v\n", rate)
	}
	a.Update(10)
	if rate := a.Snapshot().Rate(); rate != 0 {
		t.Errorf("(B) Base Case a.rate(): 0 != %v\n", rate)
	}

	// Recursive case.

	for i := 0; i < 100; i++ {
		rnd := r.Int63n(1000) + 1
		td, _ := NewEWMA(alpha, time.Second).(*StandardEWMA)

		td.Update(10)
		td.addToTimestamp(-time.Duration(rnd) * time.Second)
		expect := math.Pow(1-alpha, float64(rnd-1)) * 10.00
		if rate := td.rate(); math.Abs(rate-expect) > 0.001 {
			t.Fatalf("(A) Recursive Case a.rate(): %v != %v\n",
				expect, rate)
		}

		expect = alpha*25 + (1-alpha)*expect
		td.Update(25)
		td.addToTimestamp(-1e9)

		if rate := td.Snapshot().Rate(); math.Abs(rate-expect) > 0.001 {
			t.Fatalf("(B) Recursive Case a.rate(): %v != %v\n",
				expect, rate)
		}
	}
}

func TestEWMA1(t *testing.T) {
	// 1-minute moving average.
	testEWMA(t, 1-math.Exp(-5.0/60.0/1))
}

func TestEWMA5(t *testing.T) {
	// 5-minute moving average.
	testEWMA(t, 1-math.Exp(-5.0/60.0/5))
}

func TestEWMA15(t *testing.T) {
	// 15-minute moving average.
	testEWMA(t, 1-math.Exp(-5.0/60.0/15))
}
