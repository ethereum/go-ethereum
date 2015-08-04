// Copyright 2015 The go-ethereum Authors
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

// +build darwin

package fdtrack

import (
	"os"
	"syscall"
	"unsafe"
)

// #cgo CFLAGS: -lproc
// #include <libproc.h>
// #include <stdlib.h>
import "C"

func fdlimit() int {
	var nofile syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &nofile); err != nil {
		return 0
	}
	return int(nofile.Cur)
}

func fdusage() (int, error) {
	pid := C.int(os.Getpid())
	// Query for a rough estimate on the amout of data that
	// proc_pidinfo will return.
	rlen, err := C.proc_pidinfo(pid, C.PROC_PIDLISTFDS, 0, nil, 0)
	if rlen <= 0 {
		return 0, err
	}
	// Load the list of file descriptors. We don't actually care about
	// the content, only about the size. Since the number of fds can
	// change while we're reading them, the loop enlarges the buffer
	// until proc_pidinfo says the result fitted.
	var buf unsafe.Pointer
	defer func() {
		if buf != nil {
			C.free(buf)
		}
	}()
	for buflen := rlen; ; buflen *= 2 {
		buf, err = C.reallocf(buf, C.size_t(buflen))
		if buf == nil {
			return 0, err
		}
		rlen, err = C.proc_pidinfo(pid, C.PROC_PIDLISTFDS, 0, buf, buflen)
		if rlen <= 0 {
			return 0, err
		} else if rlen == buflen {
			continue
		}
		return int(rlen / C.PROC_PIDLISTFD_SIZE), nil
	}
	panic("unreachable")
}
