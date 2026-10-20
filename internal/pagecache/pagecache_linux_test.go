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

package pagecache

import (
	"bytes"
	"io"
	"math/rand/v2"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"golang.org/x/sys/unix"
)

// TestReaderConsumes: a consuming Reader hands back the file's bytes and
// frees their disk space as it goes; a plain Reader never touches the file.
func TestReaderConsumes(t *testing.T) {
	data := make([]byte, 4*window+12345)
	rand.NewChaCha8([32]byte{7}).Read(data)
	allocated := func(path string) int64 {
		var st syscall.Stat_t
		if err := syscall.Stat(path, &st); err != nil {
			t.Fatal(err)
		}
		return st.Blocks * 512
	}
	read := func(consume bool) (path string, got []byte) {
		path = filepath.Join(t.TempDir(), "file")
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
		f, err := os.OpenFile(path, os.O_RDWR, 0)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		r := NewReader(f)
		if consume {
			r = NewConsumingReader(f)
		}
		if got, err = io.ReadAll(r); err != nil {
			t.Fatal(err)
		}
		return path, got
	}

	path, got := read(false)
	if !bytes.Equal(got, data) {
		t.Fatal("a plain reader returned other bytes")
	}
	if n := allocated(path); n < int64(len(data)) {
		t.Fatalf("a plain reader freed the file: %d bytes allocated, %d written", n, len(data))
	}
	probe, err := os.CreateTemp(t.TempDir(), "probe")
	if err != nil {
		t.Fatal(err)
	}
	defer probe.Close()
	if _, err := probe.Write(data[:window]); err != nil {
		t.Fatal(err)
	}
	if err := unix.Fallocate(int(probe.Fd()), unix.FALLOC_FL_PUNCH_HOLE|unix.FALLOC_FL_KEEP_SIZE, 0, window); err != nil {
		t.Skipf("no hole punching on this file system: %v", err)
	}
	path, got = read(true)
	if !bytes.Equal(got, data) {
		t.Fatal("a consuming reader returned other bytes")
	}
	if n := allocated(path); n > window {
		t.Fatalf("a consuming reader left %d bytes allocated, more than a window", n)
	}
}
