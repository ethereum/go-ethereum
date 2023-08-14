package metrics

import (
	"math"
	"math/rand"
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

func testEWMA(t *testing.T, alpha float64) {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	a := NewEWMA(alpha)

	// Base case.

	if rate := a.Rate(); 0 != rate {
		t.Errorf("(A) Base Case a.Rate(): 0 != %v\n", rate)
	}
	a.Update(10)
	if rate := a.Rate(); 10 != rate {
		t.Errorf("(B) Base Case a.Rate(): 10 != %v\n", rate)
	}

	// Recursive case.

	for i := 0; i < 100; i++ {
		rnd := r.Int63n(1000)
		td, _ := NewEWMA(alpha).(*StandardEWMA)
		td.Update(10)
		td.addToTimestamp(-(rnd * 1e9))
		expect := math.Pow(1-alpha, float64(rnd)) * 10.00
		if rate := td.Rate(); expect != rate {
			t.Fatalf("(A) Recursive Case a.Rate(): %v != %v\n", expect, rate)
		}

		expect = alpha*25 + (1-alpha)*expect
		td.Update(25)
		td.addToTimestamp(-1e9)

		if rate := td.Rate(); rate != expect {
			t.Fatalf("(B) Recursive Case a.Rate(): %v != %v\n", expect, rate)
		}
	}
}

func TestEWMA1(t *testing.T) {
	// 1-minute moving average.
	testEWMA(t, 1-math.Exp(-1.0/60.0/1))
}

func TestEWMA5(t *testing.T) {
	// 5-minute moving average.
	testEWMA(t, 1-math.Exp(-1.0/60.0/5))
}

func TestEWMA15(t *testing.T) {
	// 15-minute moving average.
	testEWMA(t, 1-math.Exp(-1.0/60.0/15))
}
