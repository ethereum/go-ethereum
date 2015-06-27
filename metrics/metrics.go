// Package metrics provides general system and process level metrics collection.
package metrics

import (
	"runtime"
	"time"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/rcrowley/go-metrics"
)

// CollectProcessMetrics periodically collects various metrics about the running
// process.
func CollectProcessMetrics(refresh time.Duration) {
	// Create the various data collectors
	memstats := make([]*runtime.MemStats, 2)
	diskstats := make([]*DiskStats, 2)
	for i := 0; i < len(memstats); i++ {
		memstats[i] = new(runtime.MemStats)
		diskstats[i] = new(DiskStats)
	}
	// Define the various metrics to collect
	memAllocs := metrics.GetOrRegisterMeter("system/memory/allocs", metrics.DefaultRegistry)
	memFrees := metrics.GetOrRegisterMeter("system/memory/frees", metrics.DefaultRegistry)
	memInuse := metrics.GetOrRegisterMeter("system/memory/inuse", metrics.DefaultRegistry)
	memPauses := metrics.GetOrRegisterMeter("system/memory/pauses", metrics.DefaultRegistry)

	var diskReads, diskReadBytes, diskWrites, diskWriteBytes metrics.Meter
	if err := ReadDiskStats(diskstats[0]); err == nil {
		diskReads = metrics.GetOrRegisterMeter("system/disk/readcount", metrics.DefaultRegistry)
		diskReadBytes = metrics.GetOrRegisterMeter("system/disk/readdata", metrics.DefaultRegistry)
		diskWrites = metrics.GetOrRegisterMeter("system/disk/writecount", metrics.DefaultRegistry)
		diskWriteBytes = metrics.GetOrRegisterMeter("system/disk/writedata", metrics.DefaultRegistry)
	} else {
		glog.V(logger.Debug).Infof("failed to read disk metrics: %v", err)
	}
	// Iterate loading the different stats and updating the meters
	for i := 1; ; i++ {
		runtime.ReadMemStats(memstats[i%2])
		memAllocs.Mark(int64(memstats[i%2].Mallocs - memstats[(i-1)%2].Mallocs))
		memFrees.Mark(int64(memstats[i%2].Frees - memstats[(i-1)%2].Frees))
		memInuse.Mark(int64(memstats[i%2].Alloc - memstats[(i-1)%2].Alloc))
		memPauses.Mark(int64(memstats[i%2].PauseTotalNs - memstats[(i-1)%2].PauseTotalNs))

		if ReadDiskStats(diskstats[i%2]) == nil {
			diskReads.Mark(int64(diskstats[i%2].ReadCount - diskstats[(i-1)%2].ReadCount))
			diskReadBytes.Mark(int64(diskstats[i%2].ReadBytes - diskstats[(i-1)%2].ReadBytes))
			diskWrites.Mark(int64(diskstats[i%2].WriteCount - diskstats[(i-1)%2].WriteCount))
			diskWriteBytes.Mark(int64(diskstats[i%2].WriteBytes - diskstats[(i-1)%2].WriteBytes))
		}
		time.Sleep(refresh)
	}
}
