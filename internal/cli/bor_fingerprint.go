package cli

import (
	"fmt"
	"math"

	"github.com/ethereum/go-ethereum/params"
	"github.com/mitchellh/cli"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
)

// VersionCommand is the command to show the version of the agent
type FingerprintCommand struct {
	UI cli.Ui
}

// Help implements the cli.Command interface
func (c *FingerprintCommand) Help() string {
	return `Usage: bor fingerprint

  Display the system fingerprint`
}

// Synopsis implements the cli.Command interface
func (c *FingerprintCommand) Synopsis() string {
	return "Display the system fingerprint"
}

func getCoresCount(cp []cpu.InfoStat) int {
	cores := 0
	for i := 0; i < len(cp); i++ {
		cores += int(cp[i].Cores)
	}
	return cores
}

// Run implements the cli.Command interface
func (c *FingerprintCommand) Run(args []string) int {
	v, _ := mem.VirtualMemory()
	h, _ := host.Info()
	cp, _ := cpu.Info()
	d, _ := disk.Usage("/")

	osName := h.OS
	osVer := h.Platform + " - " + h.PlatformVersion + " - " + h.KernelArch
	totalMem := math.Floor(float64(v.Total)/(1024*1024*1024)*100) / 100
	availableMem := math.Floor(float64(v.Available)/(1024*1024*1024)*100) / 100
	usedMem := math.Floor(float64(v.Used)/(1024*1024*1024)*100) / 100
	totalDisk := math.Floor(float64(d.Total)/(1024*1024*1024)*100) / 100
	availableDisk := math.Floor(float64(d.Free)/(1024*1024*1024)*100) / 100
	usedDisk := math.Floor(float64(d.Used)/(1024*1024*1024)*100) / 100

	borDetails := fmt.Sprintf("Bor Version : %s", params.VersionWithMeta)
	cpuDetails := fmt.Sprintf("CPU : %d cores", getCoresCount(cp))
	osDetails := fmt.Sprintf("OS : %s %s ", osName, osVer)
	memDetails := fmt.Sprintf("RAM :: total : %v GB, free : %v GB, used : %v GB", totalMem, availableMem, usedMem)
	diskDetails := fmt.Sprintf("STORAGE :: total : %v GB, free : %v GB, used : %v GB", totalDisk, availableDisk, usedDisk)

	c.UI.Output(borDetails)
	c.UI.Output(cpuDetails)
	c.UI.Output(osDetails)
	c.UI.Output(memDetails)
	c.UI.Output(diskDetails)
	return 0
}
