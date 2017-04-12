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

/*
#ifndef OS_WINDOWS
	#include "os/threads_posix.h"
#endif
#include "libusbi.h"
#include "libusb.h"
*/
import "C"

import (
	"fmt"
	"reflect"
	"sync"
	"time"
	"unsafe"
)

var DefaultReadTimeout = 1 * time.Second
var DefaultWriteTimeout = 1 * time.Second
var DefaultControlTimeout = 250 * time.Millisecond //5 * time.Second

type Device struct {
	handle *C.libusb_device_handle

	// Embed the device information for easy access
	*Descriptor

	// Timeouts
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	ControlTimeout time.Duration

	// Claimed interfaces
	lock    *sync.Mutex
	claimed map[uint8]int

	// Detached kernel interfaces
	detached map[uint8]int
}

func newDevice(handle *C.libusb_device_handle, desc *Descriptor) (*Device, error) {
	ifaces := 0
	d := &Device{
		handle:         handle,
		Descriptor:     desc,
		ReadTimeout:    DefaultReadTimeout,
		WriteTimeout:   DefaultWriteTimeout,
		ControlTimeout: DefaultControlTimeout,
		lock:           new(sync.Mutex),
		claimed:        make(map[uint8]int, ifaces),
		detached:       make(map[uint8]int),
	}

	if err := d.detachKernelDriver(); err != nil {
		d.Close()
		return nil, err
	}

	return d, nil
}

// detachKernelDriver detaches any active kernel drivers, if supported by the platform.
// If there are any errors, like Context.ListDevices, only the final one will be returned.
func (d *Device) detachKernelDriver() (err error) {
	for _, cfg := range d.Configs {
		for _, iface := range cfg.Interfaces {
			switch activeErr := C.libusb_kernel_driver_active(d.handle, C.int(iface.Number)); activeErr {
			case C.LIBUSB_ERROR_NOT_SUPPORTED:
				// no need to do any futher checking, no platform support
				return
			case 0:
				continue
			case 1:
				switch detachErr := C.libusb_detach_kernel_driver(d.handle, C.int(iface.Number)); detachErr {
				case C.LIBUSB_ERROR_NOT_SUPPORTED:
					// shouldn't ever get here, should be caught by the outer switch
					return
				case 0:
					d.detached[iface.Number]++
				case C.LIBUSB_ERROR_NOT_FOUND:
					// this status is returned if libusb's driver is already attached to the device
					d.detached[iface.Number]++
				default:
					err = fmt.Errorf("usb: detach kernel driver: %s", usbError(detachErr))
				}
			default:
				err = fmt.Errorf("usb: active kernel driver check: %s", usbError(activeErr))
			}
		}
	}

	return
}

// attachKernelDriver re-attaches kernel drivers to any previously detached interfaces, if supported by the platform.
// If there are any errors, like Context.ListDevices, only the final one will be returned.
func (d *Device) attachKernelDriver() (err error) {
	for iface := range d.detached {
		switch attachErr := C.libusb_attach_kernel_driver(d.handle, C.int(iface)); attachErr {
		case C.LIBUSB_ERROR_NOT_SUPPORTED:
			// no need to do any futher checking, no platform support
			return
		case 0:
			continue
		default:
			err = fmt.Errorf("usb: attach kernel driver: %s", usbError(attachErr))
		}
	}

	return
}

func (d *Device) Reset() error {
	if errno := C.libusb_reset_device(d.handle); errno != 0 {
		return usbError(errno)
	}
	return nil
}

func (d *Device) Control(rType, request uint8, val, idx uint16, data []byte) (int, error) {
	//log.Printf("control xfer: %d:%d/%d:%d %x", idx, rType, request, val, string(data))
	dataSlice := (*reflect.SliceHeader)(unsafe.Pointer(&data))
	n := C.libusb_control_transfer(
		d.handle,
		C.uint8_t(rType),
		C.uint8_t(request),
		C.uint16_t(val),
		C.uint16_t(idx),
		(*C.uchar)(unsafe.Pointer(dataSlice.Data)),
		C.uint16_t(len(data)),
		C.uint(d.ControlTimeout/time.Millisecond))
	if n < 0 {
		return int(n), usbError(n)
	}
	return int(n), nil
}

// ActiveConfig returns the config id (not the index) of the active configuration.
// This corresponds to the ConfigInfo.Config field.
func (d *Device) ActiveConfig() (uint8, error) {
	var cfg C.int
	if errno := C.libusb_get_configuration(d.handle, &cfg); errno < 0 {
		return 0, usbError(errno)
	}
	return uint8(cfg), nil
}

// SetConfig attempts to change the active configuration.
// The cfg provided is the config id (not the index) of the configuration to set,
// which corresponds to the ConfigInfo.Config field.
func (d *Device) SetConfig(cfg uint8) error {
	if errno := C.libusb_set_configuration(d.handle, C.int(cfg)); errno < 0 {
		return usbError(errno)
	}
	return nil
}

// Close the device.
func (d *Device) Close() error {
	if d.handle == nil {
		return fmt.Errorf("usb: double close on device")
	}
	d.lock.Lock()
	defer d.lock.Unlock()
	for iface := range d.claimed {
		C.libusb_release_interface(d.handle, C.int(iface))
	}
	d.attachKernelDriver()
	C.libusb_close(d.handle)
	d.handle = nil
	return nil
}

func (d *Device) OpenEndpoint(conf, iface, setup, epoint uint8) (Endpoint, error) {
	end := &endpoint{
		Device: d,
	}

	var setAlternate bool
	for _, c := range d.Configs {
		if c.Config != conf {
			continue
		}
		debug.Printf("found conf: %#v\n", c)
		for _, i := range c.Interfaces {
			if i.Number != iface {
				continue
			}
			debug.Printf("found iface: %#v\n", i)
			for i, s := range i.Setups {
				if s.Alternate != setup {
					continue
				}
				setAlternate = i != 0

				debug.Printf("found setup: %#v [default: %v]\n", s, !setAlternate)
				for _, e := range s.Endpoints {
					debug.Printf("ep %02x search: %#v\n", epoint, s)
					if e.Address != epoint {
						continue
					}
					end.InterfaceSetup = s
					end.EndpointInfo = e
					switch tt := TransferType(e.Attributes) & TRANSFER_TYPE_MASK; tt {
					case TRANSFER_TYPE_BULK:
						end.xfer = bulk_xfer
					case TRANSFER_TYPE_INTERRUPT:
						end.xfer = interrupt_xfer
					case TRANSFER_TYPE_ISOCHRONOUS:
						end.xfer = isochronous_xfer
					default:
						return nil, fmt.Errorf("usb: %s transfer is unsupported", tt)
					}
					goto found
				}
				return nil, fmt.Errorf("usb: unknown endpoint %02x", epoint)
			}
			return nil, fmt.Errorf("usb: unknown setup %02x", setup)
		}
		return nil, fmt.Errorf("usb: unknown interface %02x", iface)
	}
	return nil, fmt.Errorf("usb: unknown configuration %02x", conf)

found:

	// Set the configuration
	var activeConf C.int
	if errno := C.libusb_get_configuration(d.handle, &activeConf); errno < 0 {
		return nil, fmt.Errorf("usb: getcfg: %s", usbError(errno))
	}
	if int(activeConf) != int(conf) {
		if errno := C.libusb_set_configuration(d.handle, C.int(conf)); errno < 0 {
			return nil, fmt.Errorf("usb: setcfg: %s", usbError(errno))
		}
	}

	// Claim the interface
	if errno := C.libusb_claim_interface(d.handle, C.int(iface)); errno < 0 {
		return nil, fmt.Errorf("usb: claim: %s", usbError(errno))
	}

	// Increment the claim count
	d.lock.Lock()
	d.claimed[iface]++
	d.lock.Unlock() // unlock immediately because the next calls may block

	// Choose the alternate
	if setAlternate {
		if errno := C.libusb_set_interface_alt_setting(d.handle, C.int(iface), C.int(setup)); errno < 0 {
			debug.Printf("altsetting error: %s", usbError(errno))
			return nil, fmt.Errorf("usb: setalt: %s", usbError(errno))
		}
	}

	return end, nil
}

func (d *Device) GetStringDescriptor(desc_index int) (string, error) {

	// allocate 200-byte array limited the length of string descriptor
	goBuffer := make([]byte, 200)

	// get string descriptor from libusb. if errno < 0 then there are any errors.
	// if errno >= 0; it is a length of result string descriptor
	errno := C.libusb_get_string_descriptor_ascii(
		d.handle,
		C.uint8_t(desc_index),
		(*C.uchar)(unsafe.Pointer(&goBuffer[0])),
		200)

	// if any errors occur
	if errno < 0 {
		return "", fmt.Errorf("usb: getstr: %s", usbError(errno))
	}
	// convert slice of byte to string with limited length from errno
	stringDescriptor := string(goBuffer[:errno])

	return stringDescriptor, nil
}
