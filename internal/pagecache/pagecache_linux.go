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
	"io"
	"os"

	"golang.org/x/sys/unix"
)

// window is how much a Writer leaves dirty, or a Reader cached, before it
// writes or drops it.
const window = 8 << 20

// Writer appends to a file, sending what it writes to disk a window at a
// time and dropping it from the page cache.
type Writer struct {
	f       *os.File
	off     int64 // bytes written
	started int64 // bytes whose writeback has been started
	synced  int64 // bytes on disk and dropped
}

// NewWriter returns a Writer appending to f from its current offset.
func NewWriter(f *os.File) *Writer {
	off, _ := f.Seek(0, io.SeekCurrent)
	return &Writer{f: f, off: off, started: off, synced: off}
}

func (w *Writer) Write(p []byte) (int, error) {
	n, err := w.f.Write(p)
	w.off += int64(n)
	if w.off-w.started >= window {
		// Start the new window's writeback, then wait for the one before and
		// drop it, so at most two windows are ever dirty.
		control(w.f, func(fd int) {
			unix.SyncFileRange(fd, w.started, w.off-w.started, unix.SYNC_FILE_RANGE_WRITE)
			if w.started > w.synced {
				unix.SyncFileRange(fd, w.synced, w.started-w.synced,
					unix.SYNC_FILE_RANGE_WAIT_BEFORE|unix.SYNC_FILE_RANGE_WRITE|unix.SYNC_FILE_RANGE_WAIT_AFTER)
				unix.Fadvise(fd, w.synced, w.started-w.synced, unix.FADV_DONTNEED)
				w.synced = w.started
			}
		})
		w.started = w.off
	}
	return n, err
}

// Reader reads a file, dropping what it has read from the page cache a
// window at a time, and with consume, freeing it on disk too.
type Reader struct {
	f            *os.File
	off, dropped int64
	consume      bool
}

// NewReader returns a Reader of f from its current offset.
func NewReader(f *os.File) *Reader {
	off, _ := f.Seek(0, io.SeekCurrent)
	return &Reader{f: f, off: off, dropped: off}
}

// NewConsumingReader returns a Reader of f from its current offset that also
// punches what it has read out of the file, a window at a time, so a file
// read once gives its disk space back as it goes. f must be open for writing;
// where the file system can't punch holes, or f is read-only, the space
// comes back only once the file is removed or truncated.
func NewConsumingReader(f *os.File) *Reader {
	r := NewReader(f)
	r.consume = true
	return r
}

func (r *Reader) Read(p []byte) (int, error) {
	n, err := r.f.Read(p)
	r.off += int64(n)
	if r.off-r.dropped >= window || (err != nil && r.off > r.dropped) {
		control(r.f, func(fd int) {
			unix.Fadvise(fd, r.dropped, r.off-r.dropped, unix.FADV_DONTNEED)
			if r.consume {
				unix.Fallocate(fd, unix.FALLOC_FL_PUNCH_HOLE|unix.FALLOC_FL_KEEP_SIZE, r.dropped, r.off-r.dropped)
			}
		})
		r.dropped = r.off
	}
	return n, err
}

// control runs fn on f's descriptor; a file without one is left alone.
func control(f *os.File, fn func(fd int)) {
	rc, err := f.SyscallConn()
	if err != nil {
		return
	}
	rc.Control(func(fd uintptr) { fn(int(fd)) })
}
