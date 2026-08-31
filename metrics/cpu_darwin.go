// Copyright 2026 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

//go:build darwin && !ios && cgo

package metrics

/*
#include <mach/mach_host.h>
#include <mach/mach_init.h>
#include <unistd.h>
*/
import "C"

import (
	"errors"
	"unsafe"
)

// readCPUTimes returns the aggregated user+nice+system time of all CPUs, in
// seconds since boot. Darwin has no iowait accounting, so globalWait is 0.
func readCPUTimes() (globalTime, globalWait float64, err error) {
	var (
		loadInfo C.host_cpu_load_info_data_t
		count    = C.mach_msg_type_number_t(C.HOST_CPU_LOAD_INFO_COUNT)
	)
	status := C.host_statistics(C.host_t(C.mach_host_self()), C.HOST_CPU_LOAD_INFO,
		C.host_info_t(unsafe.Pointer(&loadInfo)), &count)
	if status != C.KERN_SUCCESS {
		return 0, 0, errors.New("host_statistics failed")
	}
	clockTicks := float64(C.sysconf(C._SC_CLK_TCK))
	if clockTicks <= 0 {
		return 0, 0, errors.New("invalid clock ticks per second")
	}
	user := float64(loadInfo.cpu_ticks[C.CPU_STATE_USER])
	nice := float64(loadInfo.cpu_ticks[C.CPU_STATE_NICE])
	system := float64(loadInfo.cpu_ticks[C.CPU_STATE_SYSTEM])
	return (user + nice + system) / clockTicks, 0, nil
}
