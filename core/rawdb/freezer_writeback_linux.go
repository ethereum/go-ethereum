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

package rawdb

import (
	"os"

	"golang.org/x/sys/unix"
)

// writeback asks the kernel to start writing the given range of the file out
// to disk, without waiting for it to complete.
//
// Left alone, the kernel keeps freshly written data dirty in the page cache
// until it ages out or the dirty set grows past the background threshold, both
// of which are far beyond what a freezer flush interval accumulates. The next
// fsync then has to write everything out at once, blocking the writer for as
// long as that takes. Kicking off writeback as the data is written keeps the
// dirty set down to the last few buffers and turns the fsync into a cheap
// journal commit plus device flush, at a steady disk load instead of bursts.
//
// This is a hint only: errors are ignored, the data is still made durable by
// the eventual fsync.
func writeback(file *os.File, offset, size int64) {
	unix.SyncFileRange(int(file.Fd()), offset, size, unix.SYNC_FILE_RANGE_WRITE)
}
