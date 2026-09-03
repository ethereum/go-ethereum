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
// return os.ErrNotExist.
type fakeFS map[string]string

func (f fakeFS) read(path string) ([]byte, error) {
	v, ok := f[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return []byte(v), nil
}

func TestCgroupV2Container(t *testing.T) {
	// Namespaced container (Docker default since 20.10): memory.max
	// sits directly at the cgroup root.
	fs := fakeFS{
		"/sys/fs/cgroup/cgroup.controllers": "memory cpu io",
		"/sys/fs/cgroup/memory.max":         "536870912",
		"/proc/self/cgroup":                 "0::/",
	}
	bytes, ok := cgroupV2Limit(fs.read)
	if !ok || bytes != 536870912 {
		t.Errorf("got (%d, %v), want (536870912, true)", bytes, ok)
	}
}

func TestCgroupV2Unlimited(t *testing.T) {
	fs := fakeFS{
		"/sys/fs/cgroup/cgroup.controllers": "memory cpu io",
		"/sys/fs/cgroup/memory.max":         "max",
		"/proc/self/cgroup":                 "0::/",
	}
	if _, ok := cgroupV2Limit(fs.read); ok {
		t.Errorf("expected ok=false for fully unlimited v2 hierarchy")
	}
}

func TestCgroupV2LimitOnAncestor(t *testing.T) {
	// Bare-metal systemd: the leaf has no limit but the containing
	// slice does.
	fs := fakeFS{
		"/sys/fs/cgroup/cgroup.controllers":                   "memory cpu io",
		"/sys/fs/cgroup/memory.max":                           "max",
		"/sys/fs/cgroup/system.slice/memory.max":              "8589934592",
		"/sys/fs/cgroup/system.slice/geth.service/memory.max": "max",
		"/proc/self/cgroup":                                   "0::/system.slice/geth.service",
	}
	bytes, ok := cgroupV2Limit(fs.read)
	if !ok || bytes != 8589934592 {
		t.Errorf("got (%d, %v), want (8589934592, true)", bytes, ok)
	}
}

func TestCgroupV2PrefersDirectRoot(t *testing.T) {
	// In a namespaced container /proc/self/cgroup may show a host-side
	// path; the direct probe at the root must win.
	fs := fakeFS{
		"/sys/fs/cgroup/cgroup.controllers":      "memory cpu io",
		"/sys/fs/cgroup/memory.max":              "536870912",
		"/sys/fs/cgroup/system.slice/memory.max": "max",
		"/proc/self/cgroup":                      "0::/system.slice/docker-abc.scope",
	}
	bytes, ok := cgroupV2Limit(fs.read)
	if !ok || bytes != 536870912 {
		t.Errorf("got (%d, ok=%v), want (536870912, true)", bytes, ok)
	}
}

func TestCgroupV2ZeroWalksUp(t *testing.T) {
	// memory.max="0" is legal but degenerate; walk up like "max".
	fs := fakeFS{
		"/sys/fs/cgroup/cgroup.controllers":                   "memory cpu io",
		"/sys/fs/cgroup/memory.max":                           "max",
		"/sys/fs/cgroup/system.slice/memory.max":              "536870912",
		"/sys/fs/cgroup/system.slice/geth.service/memory.max": "0",
		"/proc/self/cgroup":                                   "0::/system.slice/geth.service",
	}
	bytes, ok := cgroupV2Limit(fs.read)
	if !ok || bytes != 536870912 {
		t.Errorf("got (%d, ok=%v), want (536870912, true)", bytes, ok)
	}
}

func TestCgroupV1(t *testing.T) {
	fs := fakeFS{
		// no v2 hallmark file
		"/sys/fs/cgroup/memory/memory.limit_in_bytes":            "9223372036854771712",
		"/sys/fs/cgroup/memory/docker/abc/memory.limit_in_bytes": "1073741824",
		"/proc/self/cgroup": "12:memory:/docker/abc\n11:cpu:/docker/abc",
	}
	if _, ok := cgroupV2Limit(fs.read); ok {
		t.Errorf("expected v2 probe to fail on a v1-only host")
	}
	bytes, ok := cgroupV1Limit(fs.read)
	if !ok || bytes != 1073741824 {
		t.Errorf("got (%d, %v), want (1073741824, true)", bytes, ok)
	}
}

func TestCgroupV1Unlimited(t *testing.T) {
	fs := fakeFS{
		"/sys/fs/cgroup/memory/memory.limit_in_bytes":            "9223372036854771712",
		"/sys/fs/cgroup/memory/docker/abc/memory.limit_in_bytes": "9223372036854771712",
		"/proc/self/cgroup": "12:memory:/docker/abc",
	}
	if _, ok := cgroupV1Limit(fs.read); ok {
		t.Errorf("expected ok=false when v1 reports the unlimited sentinel everywhere")
	}
}

func TestCgroupV1NonDefaultPageSize(t *testing.T) {
	// The unlimited sentinel is LONG_MAX aligned to the local page
	// size, so it differs on 16 KiB and 64 KiB page kernels.
	fs := fakeFS{
		"/sys/fs/cgroup/memory/memory.limit_in_bytes":     "9223372036854710272", // 64 KiB-page sentinel
		"/sys/fs/cgroup/memory/foo/memory.limit_in_bytes": "9223372036854767616", // 16 KiB-page sentinel
		"/proc/self/cgroup":                               "12:memory:/foo",
	}
	if _, ok := cgroupV1Limit(fs.read); ok {
		t.Errorf("expected ok=false: both values are page-aligned LONG_MAX")
	}
}

func TestCgroupV1CombinedControllers(t *testing.T) {
	// Some kernels list multiple controllers per line.
	fs := fakeFS{
		"/sys/fs/cgroup/memory/memory.limit_in_bytes":     "9223372036854771712",
		"/sys/fs/cgroup/memory/foo/memory.limit_in_bytes": "2147483648",
		"/proc/self/cgroup":                               "8:cpu,memory,blkio:/foo",
	}
	bytes, ok := cgroupV1Limit(fs.read)
	if !ok || bytes != 2147483648 {
		t.Errorf("got (%d, ok=%v), want (2147483648, true)", bytes, ok)
	}
}

func TestNoCgroup(t *testing.T) {
	fs := fakeFS{
		"/proc/self/cgroup": "0::/",
	}
	if _, ok := cgroupV2Limit(fs.read); ok {
		t.Errorf("expected v2 ok=false when v2 is not mounted")
	}
	if _, ok := cgroupV1Limit(fs.read); ok {
		t.Errorf("expected v1 ok=false when v1 is not mounted")
	}
}
