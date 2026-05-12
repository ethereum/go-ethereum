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
	"testing"
)

// fakeFS is a fileReader backed by an in-memory map. Missing keys
// return os.ErrNotExist so the production code paths see the same
// errors they would on a real filesystem.
type fakeFS map[string]string

func (f fakeFS) read(path string) ([]byte, error) {
	v, ok := f[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return []byte(v), nil
}

func TestDetectLinuxLimitCgroupV2Container(t *testing.T) {
	// Common modern Docker scenario: cgroup namespace makes the
	// container see /sys/fs/cgroup as its own cgroup root, so
	// memory.max sits directly there. /proc/self/cgroup says "/".
	fs := fakeFS{
		"/sys/fs/cgroup/cgroup.controllers": "memory cpu io",
		"/sys/fs/cgroup/memory.max":         "536870912",
		"/proc/self/cgroup":                 "0::/",
	}
	bytes, src, ok := detectLinuxLimit(fs.read)
	if !ok || bytes != 536870912 || src != SourceCgroupV2 {
		t.Errorf("got (%d, %s, %v), want (536870912, cgroup-v2, true)", bytes, src, ok)
	}
}

func TestDetectLinuxLimitCgroupV2Unlimited(t *testing.T) {
	fs := fakeFS{
		"/sys/fs/cgroup/cgroup.controllers": "memory cpu io",
		"/sys/fs/cgroup/memory.max":         "max",
		"/proc/self/cgroup":                 "0::/",
	}
	_, _, ok := detectLinuxLimit(fs.read)
	if ok {
		t.Errorf("expected ok=false for fully unlimited v2 hierarchy")
	}
}

func TestDetectLinuxLimitCgroupV2LimitOnAncestor(t *testing.T) {
	// Bare-metal systemd: leaf cgroup has no limit but the
	// containing slice does. The walk-up must find it.
	fs := fakeFS{
		"/sys/fs/cgroup/cgroup.controllers":                   "memory cpu io",
		"/sys/fs/cgroup/memory.max":                           "max",
		"/sys/fs/cgroup/system.slice/memory.max":              "8589934592",
		"/sys/fs/cgroup/system.slice/geth.service/memory.max": "max",
		"/proc/self/cgroup":                                   "0::/system.slice/geth.service",
	}
	bytes, src, ok := detectLinuxLimit(fs.read)
	if !ok || bytes != 8589934592 || src != SourceCgroupV2 {
		t.Errorf("got (%d, %s, %v), want (8589934592, cgroup-v2, true)", bytes, src, ok)
	}
}

func TestDetectLinuxLimitCgroupV2PrefersDirectRoot(t *testing.T) {
	// The direct probe at /sys/fs/cgroup/memory.max is consulted
	// before any walk derived from /proc/self/cgroup. In a
	// namespaced container that direct read is the right answer
	// even if /proc/self/cgroup happens to show a host-side path
	// whose ancestor cgroups have no limit.
	fs := fakeFS{
		"/sys/fs/cgroup/cgroup.controllers":      "memory cpu io",
		"/sys/fs/cgroup/memory.max":              "536870912",
		"/sys/fs/cgroup/system.slice/memory.max": "max",
		"/proc/self/cgroup":                      "0::/system.slice/docker-abc.scope",
	}
	bytes, _, ok := detectLinuxLimit(fs.read)
	if !ok || bytes != 536870912 {
		t.Errorf("got (%d, ok=%v), want (536870912, true)", bytes, ok)
	}
}

func TestDetectLinuxLimitCgroupV1(t *testing.T) {
	fs := fakeFS{
		// no v2 hallmark file
		"/sys/fs/cgroup/memory/memory.limit_in_bytes":            "9223372036854771712",
		"/sys/fs/cgroup/memory/docker/abc/memory.limit_in_bytes": "1073741824",
		"/proc/self/cgroup": "12:memory:/docker/abc\n11:cpu:/docker/abc",
	}
	bytes, src, ok := detectLinuxLimit(fs.read)
	if !ok || bytes != 1073741824 || src != SourceCgroupV1 {
		t.Errorf("got (%d, %s, %v), want (1073741824, cgroup-v1, true)", bytes, src, ok)
	}
}

func TestDetectLinuxLimitCgroupV1Unlimited(t *testing.T) {
	fs := fakeFS{
		"/sys/fs/cgroup/memory/memory.limit_in_bytes":            "9223372036854771712",
		"/sys/fs/cgroup/memory/docker/abc/memory.limit_in_bytes": "9223372036854771712",
		"/proc/self/cgroup": "12:memory:/docker/abc",
	}
	_, _, ok := detectLinuxLimit(fs.read)
	if ok {
		t.Errorf("expected ok=false when v1 reports the unlimited sentinel everywhere")
	}
}

func TestDetectLinuxLimitCgroupV1NonDefaultPageSize(t *testing.T) {
	// On 16 KiB- and 64 KiB-page kernels (some arm64 distros, ppc64le)
	// the v1 unlimited sentinel is LONG_MAX page-aligned to the local
	// page size, not the 4 KiB value 0x7FFFFFFFFFFFF000. Both must
	// still be treated as unlimited.
	fs := fakeFS{
		"/sys/fs/cgroup/memory/memory.limit_in_bytes":     "9223372036854710272", // 64 KiB-page sentinel
		"/sys/fs/cgroup/memory/foo/memory.limit_in_bytes": "9223372036854767616", // 16 KiB-page sentinel
		"/proc/self/cgroup":                               "12:memory:/foo",
	}
	_, _, ok := detectLinuxLimit(fs.read)
	if ok {
		t.Errorf("expected ok=false: both values are page-aligned LONG_MAX for non-4KiB page sizes")
	}
}

func TestDetectLinuxLimitCgroupV2ZeroWalksUp(t *testing.T) {
	// A leaf with memory.max="0" is degenerate (kernel kills any
	// allocation) but legal. We should walk up the same way we do
	// for "max" and pick up an ancestor's real limit.
	fs := fakeFS{
		"/sys/fs/cgroup/cgroup.controllers":                   "memory cpu io",
		"/sys/fs/cgroup/memory.max":                           "max",
		"/sys/fs/cgroup/system.slice/memory.max":              "536870912",
		"/sys/fs/cgroup/system.slice/geth.service/memory.max": "0",
		"/proc/self/cgroup":                                   "0::/system.slice/geth.service",
	}
	bytes, _, ok := detectLinuxLimit(fs.read)
	if !ok || bytes != 536870912 {
		t.Errorf("got (%d, ok=%v), want (536870912, true)", bytes, ok)
	}
}

func TestDetectLinuxLimitCgroupV1CombinedControllers(t *testing.T) {
	// Some kernels list multiple controllers per line.
	fs := fakeFS{
		"/sys/fs/cgroup/memory/memory.limit_in_bytes":     "9223372036854771712",
		"/sys/fs/cgroup/memory/foo/memory.limit_in_bytes": "2147483648",
		"/proc/self/cgroup":                               "8:cpu,memory,blkio:/foo",
	}
	bytes, _, ok := detectLinuxLimit(fs.read)
	if !ok || bytes != 2147483648 {
		t.Errorf("got (%d, ok=%v), want (2147483648, true)", bytes, ok)
	}
}

func TestDetectLinuxLimitNoCgroup(t *testing.T) {
	fs := fakeFS{
		"/proc/self/cgroup": "0::/",
	}
	_, _, ok := detectLinuxLimit(fs.read)
	if ok {
		t.Errorf("expected ok=false when neither v1 nor v2 is mounted")
	}
}
