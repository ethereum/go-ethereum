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

package usb

// #include "libusb.h"
import "C"

import (
	"fmt"
	"reflect"
	"time"
	"unsafe"
)

type Endpoint interface {
	Read(b []byte) (int, error)
	Write(b []byte) (int, error)
	Interface() InterfaceSetup
	Info() EndpointInfo
}

type endpoint struct {
	*Device
	InterfaceSetup
	EndpointInfo
	xfer func(*endpoint, []byte, time.Duration) (int, error)
}

func (e *endpoint) Read(buf []byte) (int, error) {
	if EndpointDirection(e.Address)&ENDPOINT_DIR_MASK != ENDPOINT_DIR_IN {
		return 0, fmt.Errorf("usb: read: not an IN endpoint")
	}

	return e.xfer(e, buf, e.ReadTimeout)
}

func (e *endpoint) Write(buf []byte) (int, error) {
	if EndpointDirection(e.Address)&ENDPOINT_DIR_MASK != ENDPOINT_DIR_OUT {
		return 0, fmt.Errorf("usb: write: not an OUT endpoint")
	}

	return e.xfer(e, buf, e.WriteTimeout)
}

func (e *endpoint) Interface() InterfaceSetup { return e.InterfaceSetup }
func (e *endpoint) Info() EndpointInfo        { return e.EndpointInfo }

// TODO(kevlar): (*Endpoint).Close

func bulk_xfer(e *endpoint, buf []byte, timeout time.Duration) (int, error) {
	if len(buf) == 0 {
		return 0, nil
	}

	data := (*reflect.SliceHeader)(unsafe.Pointer(&buf)).Data

	var cnt C.int
	if errno := C.libusb_bulk_transfer(
		e.handle,
		C.uchar(e.Address),
		(*C.uchar)(unsafe.Pointer(data)),
		C.int(len(buf)),
		&cnt,
		C.uint(timeout/time.Millisecond)); errno < 0 {
		return 0, usbError(errno)
	}
	return int(cnt), nil
}

func interrupt_xfer(e *endpoint, buf []byte, timeout time.Duration) (int, error) {
	if len(buf) == 0 {
		return 0, nil
	}

	data := (*reflect.SliceHeader)(unsafe.Pointer(&buf)).Data

	var cnt C.int
	if errno := C.libusb_interrupt_transfer(
		e.handle,
		C.uchar(e.Address),
		(*C.uchar)(unsafe.Pointer(data)),
		C.int(len(buf)),
		&cnt,
		C.uint(timeout/time.Millisecond)); errno < 0 {
		return 0, usbError(errno)
	}
	return int(cnt), nil
}
