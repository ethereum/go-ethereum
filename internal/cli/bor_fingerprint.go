package cli

import (
	"fmt"
	"math"
	"os/exec"
	"strings"

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

// MarkDown implements cli.MarkDown interface
func (c *FingerprintCommand) MarkDown() string {
	items := []string{
		"# Fingerprint",
		"Display the system fingerprint",
	}

	return strings.Join(items, "\n\n")
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

type MemoryDetails struct {
	TotalMem float64 `json:"totalMem"`
	FreeMem  float64 `json:"freeMem"`
	UsedMem  float64 `json:"usedMem"`
}

type DiskDetails struct {
	TotalDisk float64 `json:"totalDisk"`
	FreeDisk  float64 `json:"freeDisk"`
	UsedDisk  float64 `json:"usedDisk"`
}

type BorFingerprint struct {
	CoresCount    int            `json:"coresCount"`
	OsName        string         `json:"osName"`
	OsVer         string         `json:"osVer"`
	DiskDetails   *DiskDetails   `json:"diskDetails"`
	MemoryDetails *MemoryDetails `json:"memoryDetails"`
}

func formatFingerprint(borFingerprint *BorFingerprint) string {
	base := formatKV([]string{
		fmt.Sprintf("Bor Version : %s", params.VersionWithMeta),
		fmt.Sprintf("CPU : %d cores", borFingerprint.CoresCount),
		fmt.Sprintf("OS : %s %s ", borFingerprint.OsName, borFingerprint.OsVer),
		fmt.Sprintf("RAM :: total : %v GB, free : %v GB, used : %v GB", borFingerprint.MemoryDetails.TotalMem, borFingerprint.MemoryDetails.FreeMem, borFingerprint.MemoryDetails.UsedMem),
		fmt.Sprintf("STORAGE :: total : %v GB, free : %v GB, used : %v GB", borFingerprint.DiskDetails.TotalDisk, borFingerprint.DiskDetails.FreeDisk, borFingerprint.DiskDetails.UsedDisk),
	})

	return base
}

func convertBytesToGB(bytesValue uint64) float64 {
	return math.Floor(float64(bytesValue)/(1024*1024*1024)*100) / 100
}

// Checks if fio exists on the node
func (c *FingerprintCommand) checkFio() error {
	cmd := exec.Command("/bin/sh", "-c", "fio -v")

	_, err := cmd.CombinedOutput()
	if err != nil {
		message := "\nFio package not installed. Install Fio for IOPS Benchmarking :\n\nDebianOS  :  'sudo apt-get update && sudo apt-get install fio -y'\nAWS AMI/CentOS  :  'sudo yum install fio -y'\nOracle LinuxOS  :  'sudo dnf install fio -y'\n"
		c.UI.Output(message)

		return err
	}

	return nil
}

// Run the IOPS benchmark for the node
func (c *FingerprintCommand) benchmark() error {
	var b []byte

	err := c.checkFio()

	if err != nil {
		// Missing Fio is not a fatal error. A message will be logged in console when it is missing in "checkFio()".
		return nil //nolint:nilerr
	}

	c.UI.Output("\nRunning a 10 second test...\n")

	cmd := exec.Command("/bin/sh", "-c", "sudo fio --filename=/file --size=2GB --direct=1 --rw=randrw --bs=64k --ioengine=libaio --iodepth=64 --runtime=10 --numjobs=4 --time_based --group_reporting --name=throughput-test-job --eta-newline=1 | grep -e 'read:' -e 'write:' | awk '{print $1,$2}' ")

	b, err = cmd.CombinedOutput()
	if err != nil {
		return err
	}

	out := string(b)
	c.UI.Output(out)

	return nil
}

// Run implements the cli.Command interface
func (c *FingerprintCommand) Run(args []string) int {
	v, err := mem.VirtualMemory()
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	h, err := host.Info()
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	cp, err := cpu.Info()
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	d, err := disk.Usage("/")
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	diskDetails := &DiskDetails{
		TotalDisk: convertBytesToGB(d.Total),
		FreeDisk:  convertBytesToGB(d.Free),
		UsedDisk:  convertBytesToGB(d.Used),
	}

	memoryDetails := &MemoryDetails{
		TotalMem: convertBytesToGB(v.Total),
		FreeMem:  convertBytesToGB(v.Available),
		UsedMem:  convertBytesToGB(v.Used),
	}

	borFingerprint := &BorFingerprint{
		CoresCount:    getCoresCount(cp),
		OsName:        h.OS,
		OsVer:         h.Platform + " - " + h.PlatformVersion + " - " + h.KernelArch,
		DiskDetails:   diskDetails,
		MemoryDetails: memoryDetails,
	}

	c.UI.Output(formatFingerprint(borFingerprint))

	if borFingerprint.OsName == "linux" {
		err = c.benchmark()
		if err != nil {
			c.UI.Error(err.Error())
			return 1
		}
	}

	return 0
}
