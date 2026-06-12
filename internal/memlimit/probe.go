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

// Package memlimit detects the effective memory limit of the current
// process. On Linux the cgroup limit is consulted first, with total
// system memory as the fallback on all platforms.
package memlimit

import (
	gopsutil "github.com/shirou/gopsutil/mem"
)

// Source identifies which mechanism produced the limit value.
type Source int

const (
	SourceUnknown Source = iota
	SourceCgroupV2
	SourceCgroupV1
	SourceSystem
)

func (s Source) String() string {
	switch s {
	case SourceCgroupV2:
		return "cgroup-v2"
	case SourceCgroupV1:
		return "cgroup-v1"
	case SourceSystem:
		return "system"
	default:
		return "unknown"
	}
}

// Limit returns the memory limit visible to this process in bytes and
// the source that produced it.
func Limit() (bytes uint64, source Source) {
	if v, src, ok := platformLimit(); ok {
		return v, src
	}
	if mem, err := gopsutil.VirtualMemory(); err == nil {
		return mem.Total, SourceSystem
	}
	return 0, SourceUnknown
}
