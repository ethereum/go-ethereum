// Go port of Coda Hale's Metrics library
//
// <https://github.com/rcrowley/go-metrics>
//
// Coda Hale's original work: <https://github.com/codahale/metrics>

package metrics

import (
	"runtime/metrics"
	"runtime/pprof"
)

var (
	metricsEnabled = false
)

// Enabled is checked by functions that are deemed 'expensive', e.g. if a
// meter-type does locking and/or non-trivial math operations during update.
func Enabled() bool {
	return metricsEnabled
}

// Enable enables the metrics system.
// The Enabled-flag is expected to be set, once, during startup, but toggling off and on
// is not supported.
//
// Enable is not safe to call concurrently. You need to call this as early as possible in
// the program, before any metrics collection will happen.
func Enable() {
	metricsEnabled = true
	startMeterTickerLoop()
}

var threadCreateProfile = pprof.Lookup("threadcreate")

type runtimeStats struct {
	GCPauses     *metrics.Float64Histogram
	GCAllocBytes uint64
	GCFreedBytes uint64

	MemTotal     uint64
	HeapObjects  uint64
	HeapFree     uint64
	HeapReleased uint64
	HeapUnused   uint64

	Goroutines   uint64
	SchedLatency *metrics.Float64Histogram
}

var runtimeSamples = []metrics.Sample{
	{Name: "/gc/pauses:seconds"}, // histogram
	{Name: "/gc/heap/allocs:bytes"},
	{Name: "/gc/heap/frees:bytes"},
	{Name: "/memory/classes/total:bytes"},
	{Name: "/memory/classes/heap/objects:bytes"},
	{Name: "/memory/classes/heap/free:bytes"},
	{Name: "/memory/classes/heap/released:bytes"},
	{Name: "/memory/classes/heap/unused:bytes"},
	{Name: "/sched/goroutines:goroutines"},
	{Name: "/sched/latencies:seconds"}, // histogram
}

func ReadRuntimeStats() *runtimeStats {
	r := new(runtimeStats)
	readRuntimeStats(r)
	return r
}

func readRuntimeStats(v *runtimeStats) {
	metrics.Read(runtimeSamples)
	for _, s := range runtimeSamples {
		// Skip invalid/unknown metrics. This is needed because some metrics
		// are unavailable in older Go versions, and attempting to read a 'bad'
		// metric panics.
		if s.Value.Kind() == metrics.KindBad {
			continue
		}

		switch s.Name {
		case "/gc/pauses:seconds":
			v.GCPauses = s.Value.Float64Histogram()
		case "/gc/heap/allocs:bytes":
			v.GCAllocBytes = s.Value.Uint64()
		case "/gc/heap/frees:bytes":
			v.GCFreedBytes = s.Value.Uint64()
		case "/memory/classes/total:bytes":
			v.MemTotal = s.Value.Uint64()
		case "/memory/classes/heap/objects:bytes":
			v.HeapObjects = s.Value.Uint64()
		case "/memory/classes/heap/free:bytes":
			v.HeapFree = s.Value.Uint64()
		case "/memory/classes/heap/released:bytes":
			v.HeapReleased = s.Value.Uint64()
		case "/memory/classes/heap/unused:bytes":
			v.HeapUnused = s.Value.Uint64()
		case "/sched/goroutines:goroutines":
			v.Goroutines = s.Value.Uint64()
		case "/sched/latencies:seconds":
			v.SchedLatency = s.Value.Float64Histogram()
		}
	}
}
