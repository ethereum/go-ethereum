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

package memlimit

import (
	"os"
	"path"
	"strconv"
	"strings"
)

// cgroupV1UnlimitedThreshold detects the v1 "no limit" sentinel, which
// is LONG_MAX rounded down to the kernel page size. Anything above 1<<62
// is treated as unlimited regardless of page size.
const cgroupV1UnlimitedThreshold = uint64(1) << 62

// fileReader abstracts os.ReadFile for testing.
type fileReader func(path string) ([]byte, error)

func platformLimit() (uint64, Source, bool) {
	if v, ok := cgroupV2Limit(os.ReadFile); ok {
		return v, SourceCgroupV2, true
	}
	if v, ok := cgroupV1Limit(os.ReadFile); ok {
		return v, SourceCgroupV1, true
	}
	return 0, "", false
}

// cgroupV2Limit reads the cgroup v2 memory.max for the current process.
// It probes /sys/fs/cgroup directly first (the effective root inside a
// cgroup-namespaced container), then the path from /proc/self/cgroup
// for the bare-metal case where the limit sits on a systemd slice.
func cgroupV2Limit(read fileReader) (uint64, bool) {
	if v, ok := readCgroupV2At("/sys/fs/cgroup", "/", read); ok {
		return v, true
	}
	procPath, ok := readProcSelfCgroupV2(read)
	if !ok || procPath == "/" {
		return 0, false
	}
	return readCgroupV2At("/sys/fs/cgroup", procPath, read)
}

// readCgroupV2At reads memory.max under root+rel, walking up parents
// until a numeric value is found or the path bottoms out.
func readCgroupV2At(root, rel string, read fileReader) (uint64, bool) {
	// cgroup.controllers exists only on v2; if absent, v2 is not mounted here.
	if _, err := read(path.Join(root, "cgroup.controllers")); err != nil {
		return 0, false
	}
	for {
		raw, err := read(path.Join(root, rel, "memory.max"))
		if err == nil {
			s := strings.TrimSpace(string(raw))
			if s != "max" {
				// Zero is legal to write but degenerate; treat it like
				// "max" and keep walking up.
				if n, err := strconv.ParseUint(s, 10, 64); err == nil && n != 0 {
					return n, true
				}
			}
		}
		if rel == "/" || rel == "" {
			return 0, false
		}
		rel = path.Dir(rel)
	}
}

// readProcSelfCgroupV2 returns the cgroup path from the v2 line
// ("0::<path>") of /proc/self/cgroup.
func readProcSelfCgroupV2(read fileReader) (string, bool) {
	raw, err := read("/proc/self/cgroup")
	if err != nil {
		return "", false
	}
	for line := range strings.SplitSeq(strings.TrimSpace(string(raw)), "\n") {
		// v2 unified line: "0::<path>"
		if strings.HasPrefix(line, "0::") {
			return strings.TrimPrefix(line, "0::"), true
		}
	}
	return "", false
}

// cgroupV1Limit reads memory.limit_in_bytes from the v1 memory
// controller, walking up parents when a node reports the unlimited
// sentinel.
func cgroupV1Limit(read fileReader) (uint64, bool) {
	rel, ok := readProcSelfCgroupV1Memory(read)
	if !ok {
		return 0, false
	}
	root := "/sys/fs/cgroup/memory"
	if _, err := read(path.Join(root, "memory.limit_in_bytes")); err != nil {
		return 0, false
	}
	for {
		raw, err := read(path.Join(root, rel, "memory.limit_in_bytes"))
		if err == nil {
			if n, err := strconv.ParseUint(strings.TrimSpace(string(raw)), 10, 64); err == nil {
				if n != 0 && n < cgroupV1UnlimitedThreshold {
					return n, true
				}
			}
		}
		if rel == "/" || rel == "" {
			return 0, false
		}
		rel = path.Dir(rel)
	}
}

// readProcSelfCgroupV1Memory parses /proc/self/cgroup for the v1 memory
// controller line ("<id>:memory:<path>" or "<id>:...,memory,...:<path>").
func readProcSelfCgroupV1Memory(read fileReader) (string, bool) {
	raw, err := read("/proc/self/cgroup")
	if err != nil {
		return "", false
	}
	for line := range strings.SplitSeq(strings.TrimSpace(string(raw)), "\n") {
		// Format: "<hierarchy-id>:<comma-separated-controllers>:<path>"
		parts := strings.SplitN(line, ":", 3)
		if len(parts) != 3 {
			continue
		}
		for ctrl := range strings.SplitSeq(parts[1], ",") {
			if ctrl == "memory" {
				return parts[2], true
			}
		}
	}
	return "", false
}
