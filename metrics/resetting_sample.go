package metrics

import "sync"

// ResettingSample converts an ordinary sample into one that resets whenever its
// snapshot is retrieved. This will break for multi-monitor systems, but when only
// a single metric is being pushed out, this ensure that low-frequency events don't
// skew th charts indefinitely.
func ResettingSample(sample Sample) Sample {
	return &resettingSample{
		Sample: sample,
	}
}

// resettingSample is a simple wrapper around a sample that resets it upon the
// snapshot retrieval.
type resettingSample struct {
	Sample
	mutex      sync.Mutex
	totalCount int64 // cumulative count across all snapshots
	totalSum   int64 // cumulative sum across all snapshots
}

// Snapshot returns a read-only copy of the sample with the original reset.
// The returned snapshot has cumulative count and sum values that monotonically
// increase across resets, as required by the Prometheus counter convention.
func (rs *resettingSample) Snapshot() *sampleSnapshot {
	rs.mutex.Lock()
	defer rs.mutex.Unlock()

	s := rs.Sample.Snapshot()
	rs.Sample.Clear()

	// Accumulate cumulative totals from this snapshot's window.
	rs.totalCount += s.count
	rs.totalSum += s.sum

	// Override count and sum with cumulative values so that Prometheus
	// _count and _sum are monotonically increasing counters.
	s.count = rs.totalCount
	s.sum = rs.totalSum

	return s
}
