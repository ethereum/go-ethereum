package metrics

import (
	"fmt"
	"math"
	"reflect"
	"runtime/metrics"
	"testing"
)

var _ Histogram = (*runtimeHistogram)(nil)

type runtimeHistogramTest struct {
	h metrics.Float64Histogram

	Count       int64
	Min         int64
	Max         int64
	Sum         int64
	Mean        float64
	Variance    float64
	StdDev      float64
	Percentiles []float64 // .5 .8 .9 .99 .995
}

// This test checks the results of statistical functions implemented
// by runtimeHistogramSnapshot.
func TestRuntimeHistogramStats(t *testing.T) {
	tests := []runtimeHistogramTest{
		0: {
			h: metrics.Float64Histogram{
				Counts:  []uint64{},
				Buckets: []float64{},
			},
			Count:       0,
			Max:         0,
			Min:         0,
			Sum:         0,
			Mean:        0,
			Variance:    0,
			StdDev:      0,
			Percentiles: []float64{0, 0, 0, 0, 0},
		},
		1: {
			// This checks the case where the highest bucket is +Inf.
			h: metrics.Float64Histogram{
				Counts:  []uint64{0, 1, 2},
				Buckets: []float64{0, 0.5, 1, math.Inf(1)},
			},
			Count:       3,
			Max:         1,
			Min:         0,
			Sum:         3,
			Mean:        0.9166666,
			Percentiles: []float64{1, 1, 1, 1, 1},
			Variance:    0.020833,
			StdDev:      0.144433,
		},
		2: {
			h: metrics.Float64Histogram{
				Counts:  []uint64{8, 6, 3, 1},
				Buckets: []float64{12, 16, 18, 24, 25},
			},
			Count:       18,
			Max:         25,
			Min:         12,
			Sum:         270,
			Mean:        16.75,
			Variance:    10.3015,
			StdDev:      3.2096,
			Percentiles: []float64{16, 18, 18, 24, 24},
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			s := runtimeHistogramSnapshot(test.h)

			if v := s.Count(); v != test.Count {
				t.Errorf("Count() = %v, want %v", v, test.Count)
			}
			if v := s.Min(); v != test.Min {
				t.Errorf("Min() = %v, want %v", v, test.Min)
			}
			if v := s.Max(); v != test.Max {
				t.Errorf("Max() = %v, want %v", v, test.Max)
			}
			if v := s.Sum(); v != test.Sum {
				t.Errorf("Sum() = %v, want %v", v, test.Sum)
			}
			if v := s.Mean(); !approxEqual(v, test.Mean, 0.0001) {
				t.Errorf("Mean() = %v, want %v", v, test.Mean)
			}
			if v := s.Variance(); !approxEqual(v, test.Variance, 0.0001) {
				t.Errorf("Variance() = %v, want %v", v, test.Variance)
			}
			if v := s.StdDev(); !approxEqual(v, test.StdDev, 0.0001) {
				t.Errorf("StdDev() = %v, want %v", v, test.StdDev)
			}
			ps := []float64{.5, .8, .9, .99, .995}
			if v := s.Percentiles(ps); !reflect.DeepEqual(v, test.Percentiles) {
				t.Errorf("Percentiles(%v) = %v, want %v", ps, v, test.Percentiles)
			}
		})
	}
}

func approxEqual(x, y, ε float64) bool {
	if math.IsInf(x, -1) && math.IsInf(y, -1) {
		return true
	}
	if math.IsInf(x, 1) && math.IsInf(y, 1) {
		return true
	}
	if math.IsNaN(x) && math.IsNaN(y) {
		return true
	}
	return math.Abs(x-y) < ε
}

// This test verifies that requesting Percentiles in unsorted order
// returns them in the requested order.
func TestRuntimeHistogramStatsPercentileOrder(t *testing.T) {
	p := runtimeHistogramSnapshot{
		Counts:  []uint64{1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
		Buckets: []float64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
	}
	result := p.Percentiles([]float64{1, 0.2, 0.5, 0.1, 0.2})
	expected := []float64{10, 2, 5, 1, 2}
	if !reflect.DeepEqual(result, expected) {
		t.Fatal("wrong result:", result)
	}
}
