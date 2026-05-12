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

// cgroupV1UnlimitedThreshold separates real limits from the kernel's
// "no limit" sentinel. The cgroup v1 memory controller stores limits
// in pages and returns LONG_MAX/PAGE_SIZE*PAGE_SIZE for unlimited,
// so the exact sentinel depends on the kernel's page size (4 KiB on
// most architectures, 16 KiB and 64 KiB also seen on arm64 and
// ppc64le). Treating any value above 1<<62 as unlimited covers every
// page size while staying well above any plausible real limit.
const cgroupV1UnlimitedThreshold = uint64(1) << 62

// fileReader reads the contents of a path. Injected for testing; the
// production reader is os.ReadFile.
type fileReader func(path string) ([]byte, error)

func platformLimit() (uint64, Source, bool) {
	return detectLinuxLimit(os.ReadFile)
}

func detectLinuxLimit(read fileReader) (uint64, Source, bool) {
	if v, ok := cgroupV2Limit(read); ok {
		return v, SourceCgroupV2, true
	}
	if v, ok := cgroupV1Limit(read); ok {
		return v, SourceCgroupV1, true
	}
	return 0, "", false
}

// cgroupV2Limit reads the cgroup v2 memory.max for the current process.
//
// Two paths are considered, in order:
//
//  1. /sys/fs/cgroup/memory.max: what a process running in its own
//     cgroup namespace (Docker default since 20.10, all modern k8s)
//     sees as its effective root. This catches the common case
//     without parsing /proc/self/cgroup.
//
//  2. /sys/fs/cgroup<path>/memory.max where <path> comes from
//     /proc/self/cgroup. This handles bare-metal Linux where the
//     limit is set on a systemd slice or other ancestor cgroup.
//
// In both cases we walk up parent cgroups whenever a node reports
// "max", because the limit may be set on an ancestor.
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

// readCgroupV2At reads memory.max under root+rel, walking up through
// parents until a numeric value is found or the path bottoms out at
// the cgroup root. Returns the first non-"max" value encountered.
func readCgroupV2At(root, rel string, read fileReader) (uint64, bool) {
	// Detect that v2 is mounted at root by checking for any of the v2
	// hallmark files. cgroup.controllers exists only on v2.
	if _, err := read(path.Join(root, "cgroup.controllers")); err != nil {
		return 0, false
	}
	for {
		raw, err := read(path.Join(root, rel, "memory.max"))
		if err == nil {
			s := strings.TrimSpace(string(raw))
			if s != "max" {
				// A numeric zero is degenerate (the kernel would
				// kill anything that allocates) but legal to write;
				// treat it the same as v1 treats a zero leaf, walking
				// up looking for a meaningful ancestor limit.
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

// readProcSelfCgroupV2 returns the cgroup path for the current process
// from a v2-format /proc/self/cgroup line ("0::<path>"). Returns ok=false
// for v1-only systems or parse failure.
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
// controller for this process. Walks parent cgroups when a node
// reports the unlimited sentinel.
func cgroupV1Limit(read fileReader) (uint64, bool) {
	rel, ok := readProcSelfCgroupV1Memory(read)
	if !ok {
		return 0, false
	}
	root := "/sys/fs/cgroup/memory"
	// Sanity-check that v1 is mounted; if not, give up.
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
