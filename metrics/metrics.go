// Go port of Coda Hale's Metrics library
//
// <https://github.com/rcrowley/go-metrics>
//
// Coda Hale's original work: <https://github.com/codahale/metrics>
package metrics

import (
	"flag"
	"io"
	"os"
	"runtime/metrics"
	"runtime/pprof"
	"strconv"
	"syscall"
	"time"

	"github.com/naoina/toml"
)

// Enabled is checked by the constructor functions for all of the
// standard metrics. If it is true, the metric returned is a stub.
//
// This global kill-switch helps quantify the observer effect and makes
// for less cluttered pprof profiles.
var Enabled = false

// init enables or disables the metrics system. Since we need this to run before
// any other code gets to create meters and timers, we'll actually do an ugly hack
// and peek into the command line args for the metrics flag.
func init() {
	if val, found := syscall.Getenv("GETH_METRICS"); found && !Enabled {
		if enable, _ := strconv.ParseBool(val); enable { // ignore error, flag parser will choke on it later
			Enabled = true
		}
	}

	fs := flag.NewFlagSet("", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Bool("metrics", false, "")
	fs.String("config", "", "")

	// The flag package will quit parsing immediately if it encounters a flag that
	// was not declared to it ahead of time. We could be fancy and try to look
	// through the args to find the ones we care about, but then we'd need to
	// handle the various ways that flags can be defined -- which was the point of
	// using the flags package in the first place! So instead, let's chop off the
	// first element in the args list each time a parse fails. This way we will
	// eventually parse every arg and get the ones we care about.
	for i := range os.Args[1:] {
		if err := fs.Parse(os.Args[i+1:]); err == nil {
			break
		}
	}

	// Now visit the flags we defined which are present in the args and see if we
	// should enable metrics.
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "metrics":
			Enabled = true
		case "config":
			data, err := os.ReadFile(f.Value.String())
			if err != nil {
				return
			}
			var cfg map[string]map[string]any
			if err := toml.Unmarshal(data, &cfg); err != nil {
				return
			}
			// A config file is definitely being used and was parsed correct. Let's
			// try to peek inside and see if metrics are listed as enabled.
			if m, ok := cfg["Metrics"]; ok {
				if v, ok := m["Enabled"].(bool); ok && v {
					Enabled = true
				}
			}
		}
	})
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

// CollectProcessMetrics periodically collects various metrics about the running process.
func CollectProcessMetrics(refresh time.Duration) {
	// Short circuit if the metrics system is disabled
	if !Enabled {
		return
	}

	// Create the various data collectors
	var (
		cpustats  = make([]CPUStats, 2)
		diskstats = make([]DiskStats, 2)
		rstats    = make([]runtimeStats, 2)
	)

	// This scale factor is used for the runtime's time metrics. It's useful to convert to
	// ns here because the runtime gives times in float seconds, but runtimeHistogram can
	// only provide integers for the minimum and maximum values.
	const secondsToNs = float64(time.Second)

	// Define the various metrics to collect
	var (
		cpuSysLoad            = GetOrRegisterGauge("system/cpu/sysload", DefaultRegistry)
		cpuSysWait            = GetOrRegisterGauge("system/cpu/syswait", DefaultRegistry)
		cpuProcLoad           = GetOrRegisterGauge("system/cpu/procload", DefaultRegistry)
		cpuSysLoadTotal       = GetOrRegisterCounterFloat64("system/cpu/sysload/total", DefaultRegistry)
		cpuSysWaitTotal       = GetOrRegisterCounterFloat64("system/cpu/syswait/total", DefaultRegistry)
		cpuProcLoadTotal      = GetOrRegisterCounterFloat64("system/cpu/procload/total", DefaultRegistry)
		cpuThreads            = GetOrRegisterGauge("system/cpu/threads", DefaultRegistry)
		cpuGoroutines         = GetOrRegisterGauge("system/cpu/goroutines", DefaultRegistry)
		cpuSchedLatency       = getOrRegisterRuntimeHistogram("system/cpu/schedlatency", secondsToNs, nil)
		memPauses             = getOrRegisterRuntimeHistogram("system/memory/pauses", secondsToNs, nil)
		memAllocs             = GetOrRegisterMeter("system/memory/allocs", DefaultRegistry)
		memFrees              = GetOrRegisterMeter("system/memory/frees", DefaultRegistry)
		memTotal              = GetOrRegisterGauge("system/memory/held", DefaultRegistry)
		heapUsed              = GetOrRegisterGauge("system/memory/used", DefaultRegistry)
		heapObjects           = GetOrRegisterGauge("system/memory/objects", DefaultRegistry)
		diskReads             = GetOrRegisterMeter("system/disk/readcount", DefaultRegistry)
		diskReadBytes         = GetOrRegisterMeter("system/disk/readdata", DefaultRegistry)
		diskReadBytesCounter  = GetOrRegisterCounter("system/disk/readbytes", DefaultRegistry)
		diskWrites            = GetOrRegisterMeter("system/disk/writecount", DefaultRegistry)
		diskWriteBytes        = GetOrRegisterMeter("system/disk/writedata", DefaultRegistry)
		diskWriteBytesCounter = GetOrRegisterCounter("system/disk/writebytes", DefaultRegistry)
	)

	var lastCollectTime time.Time

	// Iterate loading the different stats and updating the meters.
	now, prev := 0, 1
	for ; ; now, prev = prev, now {
		// Gather CPU times.
		ReadCPUStats(&cpustats[now])
		collectTime := time.Now()
		secondsSinceLastCollect := collectTime.Sub(lastCollectTime).Seconds()
		lastCollectTime = collectTime
		if secondsSinceLastCollect > 0 {
			sysLoad := cpustats[now].GlobalTime - cpustats[prev].GlobalTime
			sysWait := cpustats[now].GlobalWait - cpustats[prev].GlobalWait
			procLoad := cpustats[now].LocalTime - cpustats[prev].LocalTime
			// Convert to integer percentage.
			cpuSysLoad.Update(int64(sysLoad / secondsSinceLastCollect * 100))
			cpuSysWait.Update(int64(sysWait / secondsSinceLastCollect * 100))
			cpuProcLoad.Update(int64(procLoad / secondsSinceLastCollect * 100))
			// increment counters (ms)
			cpuSysLoadTotal.Inc(sysLoad)
			cpuSysWaitTotal.Inc(sysWait)
			cpuProcLoadTotal.Inc(procLoad)
		}

		// Threads
		cpuThreads.Update(int64(threadCreateProfile.Count()))

		// Go runtime metrics
		readRuntimeStats(&rstats[now])

		cpuGoroutines.Update(int64(rstats[now].Goroutines))
		cpuSchedLatency.update(rstats[now].SchedLatency)
		memPauses.update(rstats[now].GCPauses)

		memAllocs.Mark(int64(rstats[now].GCAllocBytes - rstats[prev].GCAllocBytes))
		memFrees.Mark(int64(rstats[now].GCFreedBytes - rstats[prev].GCFreedBytes))

		memTotal.Update(int64(rstats[now].MemTotal))
		heapUsed.Update(int64(rstats[now].MemTotal - rstats[now].HeapUnused - rstats[now].HeapFree - rstats[now].HeapReleased))
		heapObjects.Update(int64(rstats[now].HeapObjects))

		// Disk
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
