// Copyright 2009 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package reuseport

import (
	"golang.org/x/sys/unix"
	"os"
)

func boolint(b bool) int {
	if b {
		return 1
	}
	return 0
}

func setNoDelay(fd int, noDelay bool) error {
	return os.NewSyscallError("setsockopt", unix.SetsockoptInt(fd, unix.IPPROTO_TCP, unix.TCP_NODELAY, boolint(noDelay)))
}

func setLinger(fd int, sec int) error {
	var l unix.Linger
	if sec >= 0 {
		l.Onoff = 1
		l.Linger = int32(sec)
	} else {
		l.Onoff = 0
		l.Linger = 0
	}
	return os.NewSyscallError("setsockopt", unix.SetsockoptLinger(fd, unix.SOL_SOCKET, unix.SO_LINGER, &l))
}
