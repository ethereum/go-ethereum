package metrics

type HistogramSnapshot interface {
	Count() int64
	Max() int64
	Mean() float64
	Min() int64
	Percentile(float64) float64
	Percentiles([]float64) []float64
	Size() int
	StdDev() float64
	Sum() int64
	Variance() float64
}

// Histogram calculates distribution statistics from a series of int64 values.
type Histogram interface {
	Clear()
	Update(int64)
	Snapshot() HistogramSnapshot
}

// GetOrRegisterHistogram returns an existing Histogram or constructs and
// registers a new StandardHistogram.
func GetOrRegisterHistogram(name string, r Registry, s Sample) Histogram {
	return getOrRegister(name, func() Histogram { return NewHistogram(s) }, r)
}

// GetOrRegisterHistogramLazy returns an existing Histogram or constructs and
// registers a new StandardHistogram.
func GetOrRegisterHistogramLazy(name string, r Registry, s func() Sample) Histogram {
	return getOrRegister(name, func() Histogram { return NewHistogram(s()) }, r)
}

// NewHistogram constructs a new StandardHistogram from a Sample.
func NewHistogram(s Sample) Histogram {
	return &StandardHistogram{s}
}

// NewRegisteredHistogram constructs and registers a new StandardHistogram from
// a Sample.
func NewRegisteredHistogram(name string, r Registry, s Sample) Histogram {
	c := NewHistogram(s)
	if nil == r {
		r = DefaultRegistry
	}
	r.Register(name, c)
	return c
}

// StandardHistogram is the standard implementation of a Histogram and uses a
// Sample to bound its memory use.
type StandardHistogram struct {
	sample Sample
}

// Clear clears the histogram and its sample.
func (h *StandardHistogram) Clear() { h.sample.Clear() }

// Snapshot returns a read-only copy of the histogram.
func (h *StandardHistogram) Snapshot() HistogramSnapshot {
	return h.sample.Snapshot()
}

// Update samples a new value.
func (h *StandardHistogram) Update(v int64) { h.sample.Update(v) }
