// usb - Self contained USB and HID library for Go
// Copyright 2019 The library Authors
//
// This library is free software: you can redistribute it and/or modify it under
// the terms of the GNU Lesser General Public License as published by the Free
// Software Foundation, either version 3 of the License, or (at your option) any
// later version.
//
// The library is distributed in the hope that it will be useful, but WITHOUT ANY
// WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR
// A PARTICULAR PURPOSE. See the GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License along
// with the library. If not, see <http://www.gnu.org/licenses/>.

// +build freebsd,cgo linux,cgo darwin,!ios,cgo windows,cgo

package usb

/*
#cgo CFLAGS: -I./hidapi/hidapi
#cgo CFLAGS: -I./libusb/libusb
#cgo CFLAGS: -DDEFAULT_VISIBILITY=""
#cgo CFLAGS: -DPOLL_NFDS_TYPE=int

#cgo linux CFLAGS: -DOS_LINUX -D_GNU_SOURCE -DHAVE_SYS_TIME_H
#cgo linux,!android LDFLAGS: -lrt
#cgo darwin CFLAGS: -DOS_DARWIN -DHAVE_SYS_TIME_H
#cgo darwin LDFLAGS: -framework CoreFoundation -framework IOKit -lobjc
#cgo windows CFLAGS: -DOS_WINDOWS
#cgo windows LDFLAGS: -lsetupapi
#cgo freebsd CFLAGS: -DOS_FREEBSD
#cgo freebsd LDFLAGS: -lusb
#cgo openbsd CFLAGS: -DOS_OPENBSD

#if defined(OS_LINUX) || defined(OS_DARWIN) || defined(DOS_FREEBSD) || defined(OS_OPENBSD)
	#include <poll.h>
	#include "os/threads_posix.c"
	#include "os/poll_posix.c"
#elif defined(OS_WINDOWS)
	#include "os/poll_windows.c"
	#include "os/threads_windows.c"
#endif

#ifdef OS_LINUX
	#include "os/linux_usbfs.c"
	#include "os/linux_netlink.c"
	#include "hidapi/libusb/hid.c"
#elif OS_DARWIN
	#include "os/darwin_usb.c"
	#include "hidapi/mac/hid.c"
#elif OS_WINDOWS
	#include "os/windows_nt_common.c"
	#include "os/windows_usbdk.c"
	#include "os/windows_winusb.c"
	#include "hidapi/windows/hid.c"
#elif OS_FREEBSD
	#include <libusb.h>
	#include "hidapi/libusb/hid.c"
#elif DOS_OPENBSD
	#include "os/openbsd_usb.c"
	#include "hidapi/libusb/hid.c"
#endif

#ifndef OS_FREEBSD
	#include "core.c"
	#include "descriptor.c"
	#include "hotplug.c"
	#include "io.c"
	#include "strerror.c"
	#include "sync.c"
#endif
*/
import "C"
