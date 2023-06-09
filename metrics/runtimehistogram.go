package metrics

import (
	"math"
	"runtime/metrics"
	"sort"
	"sync/atomic"
)

func getOrRegisterRuntimeHistogram(name string, scale float64, r Registry) *runtimeHistogram {
	if r == nil {
		r = DefaultRegistry
	}
	constructor := func() Histogram { return newRuntimeHistogram(scale) }
	return r.GetOrRegister(name, constructor).(*runtimeHistogram)
}

// runtimeHistogram wraps a runtime/metrics histogram.
type runtimeHistogram struct {
	v           atomic.Value
	scaleFactor float64
}

func newRuntimeHistogram(scale float64) *runtimeHistogram {
	h := &runtimeHistogram{scaleFactor: scale}
	h.update(&metrics.Float64Histogram{})
	return h
}

func (h *runtimeHistogram) update(mh *metrics.Float64Histogram) {
	if mh == nil {
		// The update value can be nil if the current Go version doesn't support a
		// requested metric. It's just easier to handle nil here than putting
		// conditionals everywhere.
		return
	}

	s := runtimeHistogramSnapshot{
		Counts:  make([]uint64, len(mh.Counts)),
		Buckets: make([]float64, len(mh.Buckets)),
	}
	copy(s.Counts, mh.Counts)
	copy(s.Buckets, mh.Buckets)
	for i, b := range s.Buckets {
		s.Buckets[i] = b * h.scaleFactor
	}
	h.v.Store(&s)
}

func (h *runtimeHistogram) load() *runtimeHistogramSnapshot {
	return h.v.Load().(*runtimeHistogramSnapshot)
}

func (h *runtimeHistogram) Clear() {
	panic("runtimeHistogram does not support Clear")
}
func (h *runtimeHistogram) Update(int64) {
	panic("runtimeHistogram does not support Update")
}
func (h *runtimeHistogram) Sample() Sample {
	return NilSample{}
}

// Snapshot returns a non-changing cop of the histogram.
func (h *runtimeHistogram) Snapshot() Histogram {
	return h.load()
}

// Count returns the sample count.
func (h *runtimeHistogram) Count() int64 {
	return h.load().Count()
}

// Mean returns an approximation of the mean.
func (h *runtimeHistogram) Mean() float64 {
	return h.load().Mean()
}

// StdDev approximates the standard deviation of the histogram.
func (h *runtimeHistogram) StdDev() float64 {
	return h.load().StdDev()
}

// Variance approximates the variance of the histogram.
func (h *runtimeHistogram) Variance() float64 {
	return h.load().Variance()
}

// Percentile computes the p'th percentile value.
func (h *runtimeHistogram) Percentile(p float64) float64 {
	return h.load().Percentile(p)
}

// Percentiles computes all requested percentile values.
func (h *runtimeHistogram) Percentiles(ps []float64) []float64 {
	return h.load().Percentiles(ps)
}

// Max returns the highest sample value.
func (h *runtimeHistogram) Max() int64 {
	return h.load().Max()
}

// Min returns the lowest sample value.
func (h *runtimeHistogram) Min() int64 {
	return h.load().Min()
}

// Sum returns the sum of all sample values.
func (h *runtimeHistogram) Sum() int64 {
	return h.load().Sum()
}

type runtimeHistogramSnapshot metrics.Float64Histogram

func (h *runtimeHistogramSnapshot) Clear() {
	panic("runtimeHistogram does not support Clear")
}
func (h *runtimeHistogramSnapshot) Update(int64) {
	panic("runtimeHistogram does not support Update")
}
func (h *runtimeHistogramSnapshot) Sample() Sample {
	return NilSample{}
}

func (h *runtimeHistogramSnapshot) Snapshot() Histogram {
	return h
}

// Count returns the sample count.
func (h *runtimeHistogramSnapshot) Count() int64 {
	var count int64
	for _, c := range h.Counts {
		count += int64(c)
	}
	return count
}

// Mean returns an approximation of the mean.
func (h *runtimeHistogramSnapshot) Mean() float64 {
	if len(h.Counts) == 0 {
		return 0
	}
	mean, _ := h.mean()
	return mean
}

// mean computes the mean and also the total sample count.
func (h *runtimeHistogramSnapshot) mean() (mean, totalCount float64) {
	var sum float64
	for i, c := range h.Counts {
		midpoint := h.midpoint(i)
		sum += midpoint * float64(c)
		totalCount += float64(c)
	}
	return sum / totalCount, totalCount
}

func (h *runtimeHistogramSnapshot) midpoint(bucket int) float64 {
	high := h.Buckets[bucket+1]
	low := h.Buckets[bucket]
	if math.IsInf(high, 1) {
		// The edge of the highest bucket can be +Inf, and it's supposed to mean that this
		// bucket contains all remaining samples > low. We can't get the middle of an
		// infinite range, so just return the lower bound of this bucket instead.
		return low
	}
	if math.IsInf(low, -1) {
		// Similarly, we can get -Inf in the left edge of the lowest bucket,
		// and it means the bucket contains all remaining values < high.
		return high
	}
	return (low + high) / 2
}

// StdDev approximates the standard deviation of the histogram.
func (h *runtimeHistogramSnapshot) StdDev() float64 {
	return math.Sqrt(h.Variance())
}

// Variance approximates the variance of the histogram.
func (h *runtimeHistogramSnapshot) Variance() float64 {
	if len(h.Counts) == 0 {
		return 0
	}

	mean, totalCount := h.mean()
	if totalCount <= 1 {
		// There is no variance when there are zero or one items.
		return 0
	}

	var sum float64
	for i, c := range h.Counts {
		midpoint := h.midpoint(i)
		d := midpoint - mean
		sum += float64(c) * (d * d)
	}
	return sum / (totalCount - 1)
}

// Percentile computes the p'th percentile value.
func (h *runtimeHistogramSnapshot) Percentile(p float64) float64 {
	threshold := float64(h.Count()) * p
	values := [1]float64{threshold}
	h.computePercentiles(values[:])
	return values[0]
}

// Percentiles computes all requested percentile values.
func (h *runtimeHistogramSnapshot) Percentiles(ps []float64) []float64 {
	// Compute threshold values. We need these to be sorted
	// for the percentile computation, but restore the original
	// order later, so keep the indexes as well.
	count := float64(h.Count())
	thresholds := make([]float64, len(ps))
	indexes := make([]int, len(ps))
	for i, percentile := range ps {
		thresholds[i] = count * math.Max(0, math.Min(1.0, percentile))
		indexes[i] = i
	}
	sort.Sort(floatsAscendingKeepingIndex{thresholds, indexes})

	// Now compute. The result is stored back into the thresholds slice.
	h.computePercentiles(thresholds)

	// Put the result back into the requested order.
	sort.Sort(floatsByIndex{thresholds, indexes})
	return thresholds
}

func (h *runtimeHistogramSnapshot) computePercentiles(thresh []float64) {
	var totalCount float64
	for i, count := range h.Counts {
		totalCount += float64(count)

		for len(thresh) > 0 && thresh[0] < totalCount {
			thresh[0] = h.Buckets[i]
			thresh = thresh[1:]
		}
		if len(thresh) == 0 {
			return
		}
	}
}

// Note: runtime/metrics.Float64Histogram is a collection of float64s, but the methods
// below need to return int64 to satisfy the interface. The histogram provided by runtime
// also doesn't keep track of individual samples, so results are approximated.

// Max returns the highest sample value.
func (h *runtimeHistogramSnapshot) Max() int64 {
	for i := len(h.Counts) - 1; i >= 0; i-- {
		count := h.Counts[i]
		if count > 0 {
			edge := h.Buckets[i+1]
			if math.IsInf(edge, 1) {
				edge = h.Buckets[i]
			}
			return int64(math.Ceil(edge))
		}
	}
	return 0
}

// Min returns the lowest sample value.
func (h *runtimeHistogramSnapshot) Min() int64 {
	for i, count := range h.Counts {
		if count > 0 {
			return int64(math.Floor(h.Buckets[i]))
		}
	}
	return 0
}

// Sum returns the sum of all sample values.
func (h *runtimeHistogramSnapshot) Sum() int64 {
	var sum float64
	for i := range h.Counts {
		sum += h.Buckets[i] * float64(h.Counts[i])
	}
	return int64(math.Ceil(sum))
}

type floatsAscendingKeepingIndex struct {
	values  []float64
	indexes []int
}

func (s floatsAscendingKeepingIndex) Len() int {
	return len(s.values)
}

func (s floatsAscendingKeepingIndex) Less(i, j int) bool {
	return s.values[i] < s.values[j]
}

func (s floatsAscendingKeepingIndex) Swap(i, j int) {
	s.values[i], s.values[j] = s.values[j], s.values[i]
	s.indexes[i], s.indexes[j] = s.indexes[j], s.indexes[i]
}

type floatsByIndex struct {
	values  []float64
	indexes []int
}

func (s floatsByIndex) Len() int {
	return len(s.values)
}

func (s floatsByIndex) Less(i, j int) bool {
	return s.indexes[i] < s.indexes[j]
}

func (s floatsByIndex) Swap(i, j int) {
	s.values[i], s.values[j] = s.values[j], s.values[i]
	s.indexes[i], s.indexes[j] = s.indexes[j], s.indexes[i]
}
