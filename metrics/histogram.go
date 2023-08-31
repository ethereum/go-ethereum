package metrics

type HistogramSnapshot interface {
	Count() int64
	Max() int64
	Mean() float64
	Min() int64
	Percentile(float64) float64
	Percentiles([]float64) []float64
	Sample() SampleSnapshot
	StdDev() float64
	Sum() int64
	Variance() float64
}

// Histograms calculate distribution statistics from a series of int64 values.
type Histogram interface {
	Clear()
	Update(int64)
	Snapshot() HistogramSnapshot
}

// GetOrRegisterHistogram returns an existing Histogram or constructs and
// registers a new StandardHistogram.
func GetOrRegisterHistogram(name string, r Registry, s Sample) Histogram {
	if nil == r {
		r = DefaultRegistry
	}
	return r.GetOrRegister(name, func() Histogram { return NewHistogram(s) }).(Histogram)
}

// GetOrRegisterHistogramLazy returns an existing Histogram or constructs and
// registers a new StandardHistogram.
func GetOrRegisterHistogramLazy(name string, r Registry, s func() Sample) Histogram {
	if nil == r {
		r = DefaultRegistry
	}
	return r.GetOrRegister(name, func() Histogram { return NewHistogram(s()) }).(Histogram)
}

// NewHistogram constructs a new StandardHistogram from a Sample.
func NewHistogram(s Sample) Histogram {
	if !Enabled {
		return NilHistogram{}
	}
	return &StandardHistogram{sample: s}
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

// histogramSnapshot is a read-only copy of another Histogram.
type histogramSnapshot struct {
	sample *sampleSnapshot
}

// Count returns the number of samples recorded at the time the snapshot was
// taken.
func (h *histogramSnapshot) Count() int64 { return h.sample.Count() }

// Max returns the maximum value in the sample at the time the snapshot was
// taken.
func (h *histogramSnapshot) Max() int64 { return h.sample.Max() }

// Mean returns the mean of the values in the sample at the time the snapshot
// was taken.
func (h *histogramSnapshot) Mean() float64 { return h.sample.Mean() }

// Min returns the minimum value in the sample at the time the snapshot was
// taken.
func (h *histogramSnapshot) Min() int64 { return h.sample.Min() }

// Percentile returns an arbitrary percentile of values in the sample at the
// time the snapshot was taken.
func (h *histogramSnapshot) Percentile(p float64) float64 {
	return h.sample.Percentile(p)
}

// Percentiles returns a slice of arbitrary percentiles of values in the sample
// at the time the snapshot was taken.
func (h *histogramSnapshot) Percentiles(ps []float64) []float64 {
	return h.sample.Percentiles(ps)
}

// Sample returns the Sample underlying the histogram.
func (h *histogramSnapshot) Sample() SampleSnapshot { return h.sample }

// Snapshot returns the snapshot.
func (h *histogramSnapshot) Snapshot() HistogramSnapshot { return h }

// StdDev returns the standard deviation of the values in the sample at the
// time the snapshot was taken.
func (h *histogramSnapshot) StdDev() float64 { return h.sample.StdDev() }

// Sum returns the sum in the sample at the time the snapshot was taken.
func (h *histogramSnapshot) Sum() int64 { return h.sample.Sum() }

// Variance returns the variance of inputs at the time the snapshot was taken.
func (h *histogramSnapshot) Variance() float64 { return h.sample.Variance() }

// NilHistogram is a no-op Histogram.
type NilHistogram struct{}

// Clear is a no-op.
func (NilHistogram) Clear() {}

// Count is a no-op.
func (NilHistogram) Count() int64 { return 0 }

// Max is a no-op.
func (NilHistogram) Max() int64 { return 0 }

// Mean is a no-op.
func (NilHistogram) Mean() float64 { return 0.0 }

// Min is a no-op.
func (NilHistogram) Min() int64 { return 0 }

// Percentile is a no-op.
func (NilHistogram) Percentile(p float64) float64 { return 0.0 }

// Percentiles is a no-op.
func (NilHistogram) Percentiles(ps []float64) []float64 {
	return make([]float64, len(ps))
}

// Sample is a no-op.
func (NilHistogram) Sample() SampleSnapshot { return NilSample{} }

// Snapshot is a no-op.
func (NilHistogram) Snapshot() HistogramSnapshot { return NilHistogram{} }

// StdDev is a no-op.
func (NilHistogram) StdDev() float64 { return 0.0 }

// Sum is a no-op.
func (NilHistogram) Sum() int64 { return 0 }

// Update is a no-op.
func (NilHistogram) Update(v int64) {}

// Variance is a no-op.
func (NilHistogram) Variance() float64 { return 0.0 }

// StandardHistogram is the standard implementation of a Histogram and uses a
// Sample to bound its memory use.
type StandardHistogram struct {
	sample Sample
}

// Clear clears the histogram and its sample.
func (h *StandardHistogram) Clear() { h.sample.Clear() }

// Snapshot returns a read-only copy of the histogram.
func (h *StandardHistogram) Snapshot() HistogramSnapshot {
	return &histogramSnapshot{sample: h.sample.Snapshot().(*sampleSnapshot)}
}

// Update samples a new value.
func (h *StandardHistogram) Update(v int64) { h.sample.Update(v) }
