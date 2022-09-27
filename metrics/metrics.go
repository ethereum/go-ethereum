// Go port of Coda Hale's Metrics library
//
// <https://github.com/rcrowley/go-metrics>
//
// Coda Hale's original work: <https://github.com/codahale/metrics>
package metrics

import (
	"os"
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
	GCAllocBytes uint64
	GCFreedBytes uint64

	MemTotal     uint64
	HeapFree     uint64
	HeapReleased uint64
	HeapUnused   uint64

	Goroutines uint64
}

var runtimeSamples = []metrics.Sample{
	{Name: "/gc/pauses:seconds"}, // Histogram
	{Name: "/gc/heap/allocs:bytes"},
	{Name: "/gc/heap/frees:bytes"},
	{Name: "/memory/classes/total:bytes"},
	{Name: "/memory/classes/heap/free:bytes"},
	{Name: "/memory/classes/heap/released:bytes"},
	{Name: "/memory/classes/heap/unused:bytes"},
	{Name: "/sched/goroutines:goroutines"},
}

func readRuntimeMetrics(v *runtimeValues) {
	metrics.Read(runtimeSamples)
	for _, s := range runtimeSamples {
		switch s.Name {
		case "/gc/pauses:seconds":
			v.GCPauses = medianBucket(s.Value.Float64Histogram())
		case "/gc/heap/allocs:bytes":
			if s.Value.Kind() == metrics.KindUint64 {
				v.GCAllocBytes = s.Value.Uint64()
			}
		case "/gc/heap/frees:bytes":
			if s.Value.Kind() == metrics.KindUint64 {
				v.GCFreedBytes = s.Value.Uint64()
			}
		case "/memory/classes/total:bytes":
			v.MemTotal = s.Value.Uint64()
		case "/memory/classes/heap/free:bytes":
			v.HeapFree = s.Value.Uint64()
		case "/memory/classes/heap/released:bytes":
			v.HeapReleased = s.Value.Uint64()
		case "/memory/classes/heap/unused:bytes":
			v.HeapUnused = s.Value.Uint64()
		case "/sched/goroutines:goroutines":
			v.Goroutines = s.Value.Uint64()
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

		memPauses = GetOrRegisterGaugeFloat64("system/memory/pauses", DefaultRegistry)
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
		now := i % 2
		prev := (i - 1) % 2

		ReadCPUStats(&cpuStats[now])
		cpuSysLoad.Update((cpuStats[now].GlobalTime - cpuStats[prev].GlobalTime) / refreshFreq)
		cpuSysWait.Update((cpuStats[now].GlobalWait - cpuStats[prev].GlobalWait) / refreshFreq)
		cpuProcLoad.Update((cpuStats[now].LocalTime - cpuStats[prev].LocalTime) / refreshFreq)

		cpuThreads.Update(int64(threadCreateProfile.Count()))

		readRuntimeMetrics(&memstats[now])
		cpuGoroutines.Update(int64(memstats[now].Goroutines))
		memAllocs.Mark(int64(memstats[now].GCAllocBytes - memstats[prev].GCAllocBytes))
		memFrees.Mark(int64(memstats[now].GCFreedBytes - memstats[prev].GCFreedBytes))
		memPauses.Update(memstats[now].GCPauses)
		memUsed.Update(int64(memstats[now].MemTotal - memstats[now].HeapFree - memstats[now].HeapReleased))
		memHeld.Update(int64(memstats[now].MemTotal))

		if ReadDiskStats(&diskstats[now]) == nil {
			diskReads.Mark(diskstats[now].ReadCount - diskstats[prev].ReadCount)
			diskReadBytes.Mark(diskstats[now].ReadBytes - diskstats[prev].ReadBytes)
			diskWrites.Mark(diskstats[now].WriteCount - diskstats[prev].WriteCount)
			diskWriteBytes.Mark(diskstats[now].WriteBytes - diskstats[prev].WriteBytes)

			diskReadBytesCounter.Inc(diskstats[now].ReadBytes - diskstats[prev].ReadBytes)
			diskWriteBytesCounter.Inc(diskstats[now].WriteBytes - diskstats[prev].WriteBytes)
		}
		time.Sleep(refresh)
	}
}
