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

//go:build freebsd || openbsd

package metrics

import (
	"encoding/binary"
	"errors"
	"runtime"
	"strconv"

	"golang.org/x/sys/unix"
)

// clockTicksPerSecond is the fixed CLK_TCK value of the platform, i.e. the
// unit of the kern.cp_time counters as reported by sysconf(_SC_CLK_TCK).
var clockTicksPerSecond = func() float64 {
	if runtime.GOOS == "freebsd" {
		return 128
	}
	return 100 // openbsd
}()

// readCPUTimes returns the aggregated user+nice+system time of all CPUs, in
// seconds since boot. The BSDs have no iowait accounting, so globalWait is 0.
func readCPUTimes() (globalTime, globalWait float64, err error) {
	// kern.cp_time is a long[CPUSTATES] array. CPUSTATES differs between the
	// systems (5 on FreeBSD, 5 or 6 on OpenBSD depending on version), but the
	// first three entries are user, nice and system on all of them.
	buf, err := unix.SysctlRaw("kern.cp_time")
	if err != nil {
		return 0, 0, err
	}
	const longSize = strconv.IntSize / 8
	if len(buf) < 3*longSize {
		return 0, 0, errors.New("malformed kern.cp_time")
	}
	var ticks uint64 // user + nice + system
	for i := range 3 {
		if longSize == 8 {
			ticks += binary.NativeEndian.Uint64(buf[i*longSize:])
		} else {
			ticks += uint64(binary.NativeEndian.Uint32(buf[i*longSize:]))
		}
	}
	return float64(ticks) / clockTicksPerSecond, 0, nil
}
