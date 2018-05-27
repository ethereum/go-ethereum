// Copied and modified from sigar_linux.go.

package gosigar

import (
	"io/ioutil"
	"runtime"
	"strconv"
	"strings"
	"unsafe"
)

/*
#include <sys/param.h>
#include <sys/mount.h>
#include <sys/ucred.h>
#include <sys/types.h>
#include <sys/sysctl.h>
#include <stdlib.h>
#include <stdint.h>
#include <unistd.h>
#include <time.h>
*/
import "C"

func init() {
	system.ticks = uint64(C.sysconf(C._SC_CLK_TCK))

	Procd = "/compat/linux/proc"

	getLinuxBootTime()
}

func getMountTableFileName() string {
	return Procd + "/mtab"
}

func (self *Uptime) Get() error {
	ts := C.struct_timespec{}

	if _, err := C.clock_gettime(C.CLOCK_UPTIME, &ts); err != nil {
		return err
	}

	self.Length = float64(ts.tv_sec) + 1e-9*float64(ts.tv_nsec)

	return nil
}

func (self *FDUsage) Get() error {
	val := C.uint32_t(0)
	sc := C.size_t(4)

	name := C.CString("kern.openfiles")
	_, err := C.sysctlbyname(name, unsafe.Pointer(&val), &sc, nil, 0)
	C.free(unsafe.Pointer(name))
	if err != nil {
		return err
	}
	self.Open = uint64(val)

	name = C.CString("kern.maxfiles")
	_, err = C.sysctlbyname(name, unsafe.Pointer(&val), &sc, nil, 0)
	C.free(unsafe.Pointer(name))
	if err != nil {
		return err
	}
	self.Max = uint64(val)

	self.Unused = self.Max - self.Open

	return nil
}

func (self *ProcFDUsage) Get(pid int) error {
	err := readFile("/proc/"+strconv.Itoa(pid)+"/rlimit", func(line string) bool {
		if strings.HasPrefix(line, "nofile") {
			fields := strings.Fields(line)
			if len(fields) == 3 {
				self.SoftLimit, _ = strconv.ParseUint(fields[1], 10, 64)
				self.HardLimit, _ = strconv.ParseUint(fields[2], 10, 64)
			}
			return false
		}
		return true
	})
	if err != nil {
		return err
	}

	// linprocfs only provides this information for this process (self).
	fds, err := ioutil.ReadDir(procFileName(pid, "fd"))
	if err != nil {
		return err
	}
	self.Open = uint64(len(fds))

	return nil
}

func (self *HugeTLBPages) Get() error {
	return ErrNotImplemented{runtime.GOOS}
}

func parseCpuStat(self *Cpu, line string) error {
	fields := strings.Fields(line)

	self.User, _ = strtoull(fields[1])
	self.Nice, _ = strtoull(fields[2])
	self.Sys, _ = strtoull(fields[3])
	self.Idle, _ = strtoull(fields[4])
	return nil
}
