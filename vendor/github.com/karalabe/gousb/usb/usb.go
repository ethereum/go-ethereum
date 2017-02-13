// Copyright 2013 Google Inc.  All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package usb provides a wrapper around libusb-1.0.
package usb

/*
#cgo CFLAGS: -I../internal/libusb/libusb
#cgo CFLAGS: -DDEFAULT_VISIBILITY=""
#cgo linux CFLAGS: -DOS_LINUX -D_GNU_SOURCE -DPOLL_NFDS_TYPE=int
#cgo darwin CFLAGS: -DOS_DARWIN -DPOLL_NFDS_TYPE=int
#cgo darwin LDFLAGS: -framework CoreFoundation -framework IOKit -lobjc
#cgo openbsd CFLAGS: -DOS_OPENBSD -DPOLL_NFDS_TYPE=int
#cgo windows CFLAGS: -DOS_WINDOWS -DUSE_USBDK -DPOLL_NFDS_TYPE=int

#if defined(OS_LINUX) || defined(OS_DARWIN) || defined(OS_OPENBSD)
	#include <sys/poll.h>
	#include "os/threads_posix.c"
	#include "os/poll_posix.c"
#elif defined(OS_WINDOWS)
	#include "os/threads_windows.c"
	#include "os/poll_windows.c"
#endif

#ifdef OS_LINUX
	#include "os/linux_usbfs.c"
	#include "os/linux_netlink.c"
#elif OS_DARWIN
	#include "os/darwin_usb.c"
#elif OS_OPENBSD
	#include "os/openbsd_usb.c"
#elif OS_WINDOWS
	#include "os/windows_nt_common.c"
	#include "os/windows_usbdk.c"
#endif

#include "core.c"
#include "descriptor.c"
#include "hotplug.c"
#include "io.c"
#include "strerror.c"
#include "sync.c"
*/
import "C"

import (
	"log"
	"reflect"
	"unsafe"
)

type Context struct {
	ctx  *C.libusb_context
	done chan struct{}
}

func (c *Context) Debug(level int) {
	C.libusb_set_debug(c.ctx, C.int(level))
}

func NewContext() (*Context, error) {
	c := &Context{
		done: make(chan struct{}),
	}

	if errno := C.libusb_init(&c.ctx); errno != 0 {
		return nil, usbError(errno)
	}

	go func() {
		tv := C.struct_timeval{
			tv_sec:  0,
			tv_usec: 100000,
		}
		for {
			select {
			case <-c.done:
				return
			default:
			}
			if errno := C.libusb_handle_events_timeout_completed(c.ctx, &tv, nil); errno < 0 {
				log.Printf("handle_events: error: %s", usbError(errno))
				continue
			}
			//log.Printf("handle_events returned")
		}
	}()

	return c, nil
}

// ListDevices calls each with each enumerated device.
// If the function returns true, the device is opened and a Device is returned if the operation succeeds.
// Every Device returned (whether an error is also returned or not) must be closed.
// If there are any errors enumerating the devices,
// the final one is returned along with any successfully opened devices.
func (c *Context) ListDevices(each func(desc *Descriptor) bool) ([]*Device, error) {
	var list **C.libusb_device
	cnt := C.libusb_get_device_list(c.ctx, &list)
	if cnt < 0 {
		return nil, usbError(cnt)
	}
	defer C.libusb_free_device_list(list, 1)

	var slice []*C.libusb_device
	*(*reflect.SliceHeader)(unsafe.Pointer(&slice)) = reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(list)),
		Len:  int(cnt),
		Cap:  int(cnt),
	}

	var reterr error
	var ret []*Device
	for _, dev := range slice {
		desc, err := newDescriptor(dev)
		if err != nil {
			reterr = err
			continue
		}

		if each(desc) {
			var handle *C.libusb_device_handle
			if errno := C.libusb_open(dev, &handle); errno != 0 {
				reterr = usbError(errno)
				continue
			}
			if dev, err := newDevice(handle, desc); err != nil {
				reterr = err
			} else {
				ret = append(ret, dev)
			}
		}
	}
	return ret, reterr
}

// OpenDeviceWithVidPid opens Device from specific VendorId and ProductId.
// If there are any errors, it'll returns at second value.
func (c *Context) OpenDeviceWithVidPid(vid, pid int) (*Device, error) {

	handle := C.libusb_open_device_with_vid_pid(c.ctx, (C.uint16_t)(vid), (C.uint16_t)(pid))
	if handle == nil {
		return nil, ERROR_NOT_FOUND
	}

	dev := C.libusb_get_device(handle)
	if dev == nil {
		return nil, ERROR_NO_DEVICE
	}

	desc, err := newDescriptor(dev)

	// return an error from nil-handle and nil-device
	if err != nil {
		return nil, err
	}
	return newDevice(handle, desc)
}

func (c *Context) Close() error {
	close(c.done)
	if c.ctx != nil {
		C.libusb_exit(c.ctx)
	}
	c.ctx = nil
	return nil
}
