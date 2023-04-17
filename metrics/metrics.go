// Go port of Coda Hale's Metrics library
//
// <https://github.com/rcrowley/go-metrics>
//
// Coda Hale's original work: <https://github.com/codahale/metrics>
package metrics

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/BurntSushi/toml"
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

// configFlag is the CLI flag name to use to start node by providing a toml based config
var configFlag = "config"

// Init enables or disables the metrics system. Since we need this to run before
// any other code gets to create meters and timers, we'll actually do an ugly hack
// and peek into the command line args for the metrics flag.
func init() {
	var configFile string

	for i := 0; i < len(os.Args); i++ {
		arg := os.Args[i]

		flag := strings.TrimLeft(arg, "-")

		// check for existence of `config` flag
		if flag == configFlag && i < len(os.Args)-1 {
			configFile = strings.TrimLeft(os.Args[i+1], "-") // find the value of flag
		} else if len(flag) > 6 && flag[:6] == configFlag {
			// Checks for `=` separated flag (e.g. config=path)
			configFile = strings.TrimLeft(flag[6:], "=")
		}

		for _, enabler := range enablerFlags {
			if !Enabled && flag == enabler {
				Enabled = true
			}
		}

		for _, enabler := range expensiveEnablerFlags {
			if !EnabledExpensive && flag == enabler {
				EnabledExpensive = true
			}
		}
	}

	// Update the global metrics value, if they're provided in the config file
	updateMetricsFromConfig(configFile)
}

func updateMetricsFromConfig(path string) {
	// Don't act upon any errors here. They're already taken into
	// consideration when the toml config file will be parsed in the cli.
	canonicalPath, err := common.VerifyPath(path)
	if err != nil {
		fmt.Println("path not verified: " + err.Error())
		return
	}

	data, err := os.ReadFile(canonicalPath)
	tomlData := string(data)

	if err != nil {
		return
	}

	// Create a minimal config to decode
	type TelemetryConfig struct {
		Enabled   bool `hcl:"metrics,optional" toml:"metrics,optional"`
		Expensive bool `hcl:"expensive,optional" toml:"expensive,optional"`
	}

	type CliConfig struct {
		Telemetry *TelemetryConfig `hcl:"telemetry,block" toml:"telemetry,block"`
	}

	conf := &CliConfig{}

	_, err = toml.Decode(tomlData, &conf)
	if err != nil || conf == nil || conf.Telemetry == nil {
		return
	}

	// We have the values now, update them
	Enabled = conf.Telemetry.Enabled
	EnabledExpensive = conf.Telemetry.Expensive
}

// CollectProcessMetrics periodically collects various metrics about the running
// process.
func CollectProcessMetrics(refresh time.Duration) {
	// Short circuit if the metrics system is disabled
	if !Enabled {
		return
	}
	refreshFreq := int64(refresh / time.Second)

	// Create the various data collectors
	cpuStats := make([]*CPUStats, 2)
	memstats := make([]*runtime.MemStats, 2)
	diskstats := make([]*DiskStats, 2)
	for i := 0; i < len(memstats); i++ {
		cpuStats[i] = new(CPUStats)
		memstats[i] = new(runtime.MemStats)
		diskstats[i] = new(DiskStats)
	}
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

		ReadCPUStats(cpuStats[location1])
		cpuSysLoad.Update((cpuStats[location1].GlobalTime - cpuStats[location2].GlobalTime) / refreshFreq)
		cpuSysWait.Update((cpuStats[location1].GlobalWait - cpuStats[location2].GlobalWait) / refreshFreq)
		cpuProcLoad.Update((cpuStats[location1].LocalTime - cpuStats[location2].LocalTime) / refreshFreq)
		cpuThreads.Update(int64(threadCreateProfile.Count()))
		cpuGoroutines.Update(int64(runtime.NumGoroutine()))

		runtime.ReadMemStats(memstats[location1])
		memPauses.Mark(int64(memstats[location1].PauseTotalNs - memstats[location2].PauseTotalNs))
		memAllocs.Mark(int64(memstats[location1].Mallocs - memstats[location2].Mallocs))
		memFrees.Mark(int64(memstats[location1].Frees - memstats[location2].Frees))
		memHeld.Update(int64(memstats[location1].HeapSys - memstats[location1].HeapReleased))
		memUsed.Update(int64(memstats[location1].Alloc))

		if ReadDiskStats(diskstats[location1]) == nil {
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
