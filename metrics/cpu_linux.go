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

//go:build linux

package metrics

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

const clockTicksPerSecond = 100

// readCPUTimes returns the aggregated user+nice+system and iowait times of all
// CPUs, in seconds since boot.
func readCPUTimes() (globalTime, globalWait float64, err error) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, 0, err
	}
	line := string(data)
	if i := strings.IndexByte(line, '\n'); i >= 0 {
		line = line[:i]
	}
	fields := strings.Fields(line)
	if len(fields) < 5 || fields[0] != "cpu" {
		return 0, 0, errors.New("malformed /proc/stat")
	}
	var ticks [5]uint64 // user, nice, system, idle, iowait
	for i := range ticks {
		if i+1 >= len(fields) {
			break // missing iowait on ancient kernels, leave as 0
		}
		ticks[i], err = strconv.ParseUint(fields[i+1], 10, 64)
		if err != nil {
			return 0, 0, err
		}
	}
	globalTime = float64(ticks[0]+ticks[1]+ticks[2]) / clockTicksPerSecond
	globalWait = float64(ticks[4]) / clockTicksPerSecond
	return globalTime, globalWait, nil
}
