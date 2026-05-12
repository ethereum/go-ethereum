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

// Package memlimit detects the effective memory limit visible to the
// current process.
//
// On Linux, /proc/meminfo always reports the host's RAM, even inside
// a container, so it cannot be used as a budget on its own. Limit
// consults the kernel-enforced cgroup limit first (v2 memory.max or
// v1 memory.limit_in_bytes), walking parent cgroups if a leaf node
// reports no limit. If no container-style limit is in effect, or on
// platforms without cgroups, it falls back to total system memory.
package memlimit

import (
	gopsutil "github.com/shirou/gopsutil/mem"
)

// Source identifies which mechanism produced the limit value.
type Source string

const (
	SourceCgroupV2 Source = "cgroup-v2"
	SourceCgroupV1 Source = "cgroup-v1"
	SourceSystem   Source = "system"
	SourceUnknown  Source = "unknown"
)

// Limit returns the memory limit visible to this process in bytes and
// the source that produced it. The returned value is the tightest
// budget detectable, falling back to system total memory if no
// container-style limit is in effect.
func Limit() (bytes uint64, source Source) {
	if v, src, ok := platformLimit(); ok {
		return v, src
	}
	if mem, err := gopsutil.VirtualMemory(); err == nil {
		return mem.Total, SourceSystem
	}
	return 0, SourceUnknown
}
