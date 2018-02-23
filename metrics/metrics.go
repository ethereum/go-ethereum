// Go port of Coda Hale's Metrics library
//
// <https://github.com/rcrowley/go-metrics>
//
// Coda Hale's original work: <https://github.com/codahale/metrics>
package metrics

import (
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

// Enabled is checked by the constructor functions for all of the
// standard metrics.  If it is true, the metric returned is a stub.
//
// This global kill-switch helps quantify the observer effect and makes
// for less cluttered pprof profiles.
var Enabled bool = false

// MetricsEnabledFlag is the CLI flag name to use to enable metrics collections.
const MetricsEnabledFlag = "metrics"
const DashboardEnabledFlag = "dashboard"

// Init enables or disables the metrics system. Since we need this to run before
// any other code gets to create meters and timers, we'll actually do an ugly hack
// and peek into the command line args for the metrics flag.
func init() {
	for _, arg := range os.Args {
		if flag := strings.TrimLeft(arg, "-"); flag == MetricsEnabledFlag || flag == DashboardEnabledFlag {
			log.Info("Enabling metrics collection")
			Enabled = true
		}
	}
}

// CollectProcessMetrics periodically collects various metrics about the running
// process.
func CollectProcessMetrics(refresh time.Duration) {
	// Short circuit if the metrics system is disabled
	if !Enabled {
		return
	}
	// Create the various data collectors
	memstats := make([]*runtime.MemStats, 2)
	diskstats := make([]*DiskStats, 2)
	for i := 0; i < len(memstats); i++ {
		memstats[i] = new(runtime.MemStats)
		diskstats[i] = new(DiskStats)
	}
	// Define the various metrics to collect
	memAllocs := GetOrRegisterMeter("system/memory/allocs", DefaultRegistry)
	memFrees := GetOrRegisterMeter("system/memory/frees", DefaultRegistry)
	memInuse := GetOrRegisterMeter("system/memory/inuse", DefaultRegistry)
	memPauses := GetOrRegisterMeter("system/memory/pauses", DefaultRegistry)

	var diskReads, diskReadBytes, diskWrites, diskWriteBytes Meter
	if err := ReadDiskStats(diskstats[0]); err == nil {
		diskReads = GetOrRegisterMeter("system/disk/readcount", DefaultRegistry)
		diskReadBytes = GetOrRegisterMeter("system/disk/readdata", DefaultRegistry)
		diskWrites = GetOrRegisterMeter("system/disk/writecount", DefaultRegistry)
		diskWriteBytes = GetOrRegisterMeter("system/disk/writedata", DefaultRegistry)
	} else {
		log.Debug("Failed to read disk metrics", "err", err)
	}
	// Iterate loading the different stats and updating the meters
	for i := 1; ; i++ {
		runtime.ReadMemStats(memstats[i%2])
		memAllocs.Mark(int64(memstats[i%2].Mallocs - memstats[(i-1)%2].Mallocs))
		memFrees.Mark(int64(memstats[i%2].Frees - memstats[(i-1)%2].Frees))
		memInuse.Mark(int64(memstats[i%2].Alloc - memstats[(i-1)%2].Alloc))
		memPauses.Mark(int64(memstats[i%2].PauseTotalNs - memstats[(i-1)%2].PauseTotalNs))

		if ReadDiskStats(diskstats[i%2]) == nil {
			diskReads.Mark(diskstats[i%2].ReadCount - diskstats[(i-1)%2].ReadCount)
			diskReadBytes.Mark(diskstats[i%2].ReadBytes - diskstats[(i-1)%2].ReadBytes)
			diskWrites.Mark(diskstats[i%2].WriteCount - diskstats[(i-1)%2].WriteCount)
			diskWriteBytes.Mark(diskstats[i%2].WriteBytes - diskstats[(i-1)%2].WriteBytes)
		}
		time.Sleep(refresh)
	}
}
