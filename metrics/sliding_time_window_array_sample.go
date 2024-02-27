package metrics

import (
	"math"
	"sync"
	"time"
)

// SlidingTimeWindowArraySample is ported from Coda Hale's dropwizard library
// <https://github.com/dropwizard/metrics/pull/1139>
// A reservoir implementation backed by a sliding window that stores only the
// measurements made in the last given window of time
type SlidingTimeWindowArraySample struct {
	startTick    int64
	measurements *ChunkedAssociativeArray
	window       int64
	count        int64
	lastTick     int64
	mutex        sync.Mutex
}

const (
	// SlidingTimeWindowCollisionBuffer allow this many duplicate ticks
	// before overwriting measurements
	SlidingTimeWindowCollisionBuffer = 256

	// SlidingTimeWindowTrimThreshold is number of updates between trimming data
	SlidingTimeWindowTrimThreshold = 256

	// SlidingTimeWindowClearBufferTicks is the number of ticks to keep past the
	// requested trim
	SlidingTimeWindowClearBufferTicks = int64(time.Hour/time.Nanosecond) *
		SlidingTimeWindowCollisionBuffer
)

// NewSlidingTimeWindowArraySample creates new object with given window of time
func NewSlidingTimeWindowArraySample(window time.Duration) Sample {
	if !Enabled {
		return NilSample{}
	}
	return &SlidingTimeWindowArraySample{
		startTick:    time.Now().UnixNano(),
		measurements: NewChunkedAssociativeArray(ChunkedAssociativeArrayDefaultChunkSize),
		window:       window.Nanoseconds() * SlidingTimeWindowCollisionBuffer,
	}
}

// Clear clears all samples.
func (s *SlidingTimeWindowArraySample) Clear() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.count = 0
	s.measurements.Clear()
}

// trim requires s.mutex to already be acquired
func (s *SlidingTimeWindowArraySample) trim() {
	now := s.getTick()
	windowStart := now - s.window
	windowEnd := now + SlidingTimeWindowClearBufferTicks
	if windowStart < windowEnd {
		s.measurements.Trim(windowStart, windowEnd)
	} else {
		// long overflow handling that can only happen 1 year after class loading
		s.measurements.Clear()
	}
}

// getTick requires s.mutex to already be acquired
func (s *SlidingTimeWindowArraySample) getTick() int64 {
	oldTick := s.lastTick
	tick := (time.Now().UnixNano() - s.startTick) * SlidingTimeWindowCollisionBuffer
	var newTick int64
	if tick-oldTick > 0 {
		newTick = tick
	} else {
		newTick = oldTick + 1
	}
	s.lastTick = newTick
	return newTick
}

// Snapshot returns a read-only copy of the sample.
func (s *SlidingTimeWindowArraySample) Snapshot() SampleSnapshot {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.trim()
	var (
		samples       = s.measurements.Values()
		values        = make([]int64, len(samples))
		max     int64 = math.MinInt64
		min     int64 = math.MaxInt64
		sum     int64
	)
	for i, v := range samples {
		values[i] = v
		sum += v
		if v > max {
			max = v
		}
		if v < min {
			min = v
		}
	}
	return newSampleSnapshotPrecalculated(s.count, values, min, max, sum)
}

// Update samples a new value.
func (s *SlidingTimeWindowArraySample) Update(v int64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	var newTick int64
	s.count += 1
	if s.count%SlidingTimeWindowTrimThreshold == 0 {
		s.trim()
	}
	newTick = s.getTick()
	longOverflow := newTick < s.lastTick
	if longOverflow {
		s.measurements.Clear()
	}
	s.measurements.Put(newTick, v)
}
