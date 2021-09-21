// Copyright (c) 2012 VMware, Inc.

package gosigar

import (
	"io/ioutil"
	"strconv"
	"strings"
	"syscall"
)

func init() {
	system.ticks = 100 // C.sysconf(C._SC_CLK_TCK)

	Procd = "/proc"

	getLinuxBootTime()
}

func getMountTableFileName() string {
	return "/etc/mtab"
}

func (self *Uptime) Get() error {
	sysinfo := syscall.Sysinfo_t{}

	if err := syscall.Sysinfo(&sysinfo); err != nil {
		return err
	}

	self.Length = float64(sysinfo.Uptime)

	return nil
}

func (self *FDUsage) Get() error {
	return readFile(Procd+"/sys/fs/file-nr", func(line string) bool {
		fields := strings.Fields(line)
		if len(fields) == 3 {
			self.Open, _ = strconv.ParseUint(fields[0], 10, 64)
			self.Unused, _ = strconv.ParseUint(fields[1], 10, 64)
			self.Max, _ = strconv.ParseUint(fields[2], 10, 64)
		}
		return false
	})
}

func (self *ProcFDUsage) Get(pid int) error {
	err := readFile(procFileName(pid, "limits"), func(line string) bool {
		if strings.HasPrefix(line, "Max open files") {
			fields := strings.Fields(line)
			if len(fields) == 6 {
				self.SoftLimit, _ = strconv.ParseUint(fields[3], 10, 64)
				self.HardLimit, _ = strconv.ParseUint(fields[4], 10, 64)
			}
			return false
		}
		return true
	})
	if err != nil {
		return err
	}
	fds, err := ioutil.ReadDir(procFileName(pid, "fd"))
	if err != nil {
		return err
	}
	self.Open = uint64(len(fds))
	return nil
}

func parseCpuStat(self *Cpu, line string) error {
	fields := strings.Fields(line)

	self.User, _ = strtoull(fields[1])
	self.Nice, _ = strtoull(fields[2])
	self.Sys, _ = strtoull(fields[3])
	self.Idle, _ = strtoull(fields[4])
	self.Wait, _ = strtoull(fields[5])
	self.Irq, _ = strtoull(fields[6])
	self.SoftIrq, _ = strtoull(fields[7])
	self.Stolen, _ = strtoull(fields[8])

	return nil
}
