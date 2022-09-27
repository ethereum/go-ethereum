// Go port of Coda Hale's Metrics library
//
// <https://github.com/rcrowley/go-metrics>
//
// Coda Hale's original work: <https://github.com/codahale/metrics>
package metrics

import (
	"os"
	"runtime"
	"runtime/metrics"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

// Enabled is checked by the constructor functions for all of the
// standard metrics. If it is true, the metric returned is a stub.
//
// This global kill-switch helps quantify the observer effect and makes
// for less cluttered pprof profiles.
var Enabled = false

// EnabledExpensive is a soft-flag meant for external packages to check if costly
// metrics gathering is allowed or not. The goal is to separate standard metrics
// for health monitoring and debug metrics that might impact runtime performance.
var EnabledExpensive = false

// enablerFlags is the CLI flag names to use to enable metrics collections.
var enablerFlags = []string{"metrics"}

// expensiveEnablerFlags is the CLI flag names to use to enable metrics collections.
var expensiveEnablerFlags = []string{"metrics.expensive"}

// Init enables or disables the metrics system. Since we need this to run before
// any other code gets to create meters and timers, we'll actually do an ugly hack
// and peek into the command line args for the metrics flag.
func init() {
	for _, arg := range os.Args {
		flag := strings.TrimLeft(arg, "-")

		for _, enabler := range enablerFlags {
			if !Enabled && flag == enabler {
				log.Info("Enabling metrics collection")
				Enabled = true
			}
		}
		for _, enabler := range expensiveEnablerFlags {
			if !EnabledExpensive && flag == enabler {
				log.Info("Enabling expensive metrics collection")
				EnabledExpensive = true
			}
		}
	}
}

var threadCreateProfile = pprof.Lookup("threadcreate")

type runtimeValues struct {
	GCPauses     float64
	GCAllocs     float64
	GCFrees      float64
	MemTotal     uint64
	HeapFree     uint64
	HeapReleased uint64
	HeapUnused   uint64
}

var runtimeSamples = []metrics.Sample{
	{Name: "/gc/pauses:seconds"},            // Histogram
	{Name: "/gc/heap/allocs-by-size:bytes"}, // Histogram
	{Name: "/gc/heap/frees-by-size:bytes"},  // Histogram
	{Name: "/memory/classes/total:bytes"},
	{Name: "/memory/classes/heap/free:bytes"},
	{Name: "/memory/classes/heap/released:bytes"},
	{Name: "/memory/classes/heap/unused:bytes"},
}

func readRuntimeMetrics(v *runtimeValues) {
	metrics.Read(runtimeSamples)
	for _, s := range runtimeSamples {
		switch s.Name {
		case "/gc/pauses:seconds":
			v.GCPauses = medianBucket(s.Value.Float64Histogram())
		case "/gc/heap/allocs-by-size:bytes":
			v.GCAllocs = medianBucket(s.Value.Float64Histogram())
		case "/gc/heap/frees-by-size:bytes":
			v.GCFrees = medianBucket(s.Value.Float64Histogram())
		case "/memory/classes/total:bytes":
			v.MemTotal = s.Value.Uint64()
		case "/memory/classes/heap/free:bytes":
			v.HeapFree = s.Value.Uint64()
		case "/memory/classes/heap/released:bytes":
			v.HeapReleased = s.Value.Uint64()
		case "/memory/classes/heap/unused:bytes":
			v.HeapUnused = s.Value.Uint64()
		}
	}
}

// medianBucket gives the median of a histogram.
// This is taken from the runtime/metrics example code.
func medianBucket(h *metrics.Float64Histogram) float64 {
	total := uint64(0)
	for _, count := range h.Counts {
		total += count
	}
	thresh := total / 2
	total = 0
	for i, count := range h.Counts {
		total += count
		if total >= thresh {
			return h.Buckets[i]
		}
	}
	panic("should not happen")
}

// CollectProcessMetrics periodically collects various metrics about the running process.
func CollectProcessMetrics(refresh time.Duration) {
	// Short circuit if the metrics system is disabled
	if !Enabled {
		return
	}
	refreshFreq := int64(refresh / time.Second)

	// Create the various data collectors
	var (
		cpuStats  = make([]CPUStats, 2)
		diskstats = make([]DiskStats, 2)
		memstats  = make([]runtimeValues, 2)
	)

	// Define the various metrics to collect
	var (
		cpuSysLoad    = GetOrRegisterGauge("system/cpu/sysload", DefaultRegistry)
		cpuSysWait    = GetOrRegisterGauge("system/cpu/syswait", DefaultRegistry)
		cpuProcLoad   = GetOrRegisterGauge("system/cpu/procload", DefaultRegistry)
		cpuThreads    = GetOrRegisterGauge("system/cpu/threads", DefaultRegistry)
		cpuGoroutines = GetOrRegisterGauge("system/cpu/goroutines", DefaultRegistry)

		memPauses = GetOrRegisterMeter("system/memory/pauses", DefaultRegistry)
		memAllocs = GetOrRegisterMeter("system/memory/allocs", DefaultRegistry)
		memFrees  = GetOrRegisterMeter("system/memory/frees", DefaultRegistry)
		memHeld   = GetOrRegisterGauge("system/memory/held", DefaultRegistry)
		memUsed   = GetOrRegisterGauge("system/memory/used", DefaultRegistry)

		diskReads             = GetOrRegisterMeter("system/disk/readcount", DefaultRegistry)
		diskReadBytes         = GetOrRegisterMeter("system/disk/readdata", DefaultRegistry)
		diskReadBytesCounter  = GetOrRegisterCounter("system/disk/readbytes", DefaultRegistry)
		diskWrites            = GetOrRegisterMeter("system/disk/writecount", DefaultRegistry)
		diskWriteBytes        = GetOrRegisterMeter("system/disk/writedata", DefaultRegistry)
		diskWriteBytesCounter = GetOrRegisterCounter("system/disk/writebytes", DefaultRegistry)
	)

	// Iterate loading the different stats and updating the meters
	for i := 1; ; i++ {
		location1 := i % 2
		location2 := (i - 1) % 2

		ReadCPUStats(&cpuStats[location1])

		cpuSysLoad.Update((cpuStats[location1].GlobalTime - cpuStats[location2].GlobalTime) / refreshFreq)
		cpuSysWait.Update((cpuStats[location1].GlobalWait - cpuStats[location2].GlobalWait) / refreshFreq)
		cpuProcLoad.Update((cpuStats[location1].LocalTime - cpuStats[location2].LocalTime) / refreshFreq)

		cpuThreads.Update(int64(threadCreateProfile.Count()))
		cpuGoroutines.Update(int64(runtime.NumGoroutine()))

		readRuntimeMetrics(&memstats[location1])

		memPauses.Mark(int64(memstats[location1].GCPauses - memstats[location2].GCPauses))
		memAllocs.Mark(int64(memstats[location1].GCAllocs - memstats[location2].GCAllocs))
		memFrees.Mark(int64(memstats[location1].GCFrees - memstats[location2].GCFrees))

		memHeld.Update(int64(memstats[location1].MemTotal - memstats[location1].HeapFree - memstats[location1].HeapReleased))
		memUsed.Update(int64(memstats[location1].MemTotal))

		if ReadDiskStats(&diskstats[location1]) == nil {
			diskReads.Mark(diskstats[location1].ReadCount - diskstats[location2].ReadCount)
			diskReadBytes.Mark(diskstats[location1].ReadBytes - diskstats[location2].ReadBytes)
			diskWrites.Mark(diskstats[location1].WriteCount - diskstats[location2].WriteCount)
			diskWriteBytes.Mark(diskstats[location1].WriteBytes - diskstats[location2].WriteBytes)

			diskReadBytesCounter.Inc(diskstats[location1].ReadBytes - diskstats[location2].ReadBytes)
			diskWriteBytesCounter.Inc(diskstats[location1].WriteBytes - diskstats[location2].WriteBytes)
		}
		time.Sleep(refresh)
	}
}
