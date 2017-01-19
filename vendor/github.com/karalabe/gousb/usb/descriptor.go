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

type Descriptor struct {
	// Bus information
	Bus     uint8 // The bus on which the device was detected
	Address uint8 // The address of the device on the bus

	// Version information
	Spec   BCD // USB Specification Release Number
	Device BCD // The device version

	// Product information
	Vendor  ID // The Vendor identifer
	Product ID // The Product identifier

	// Protocol information
	Class    uint8 // The class of this device
	SubClass uint8 // The sub-class (within the class) of this device
	Protocol uint8 // The protocol (within the sub-class) of this device

	// Configuration information
	Configs []ConfigInfo
}

func newDescriptor(dev *C.libusb_device) (*Descriptor, error) {
	var desc C.struct_libusb_device_descriptor
	if errno := C.libusb_get_device_descriptor(dev, &desc); errno < 0 {
		return nil, usbError(errno)
	}

	// Enumerate configurations
	var cfgs []ConfigInfo
	for i := 0; i < int(desc.bNumConfigurations); i++ {
		var cfg *C.struct_libusb_config_descriptor
		if errno := C.libusb_get_config_descriptor(dev, C.uint8_t(i), &cfg); errno < 0 {
			return nil, usbError(errno)
		}
		cfgs = append(cfgs, newConfig(dev, cfg))
		C.libusb_free_config_descriptor(cfg)
	}

	return &Descriptor{
		Bus:      uint8(C.libusb_get_bus_number(dev)),
		Address:  uint8(C.libusb_get_device_address(dev)),
		Spec:     BCD(desc.bcdUSB),
		Device:   BCD(desc.bcdDevice),
		Vendor:   ID(desc.idVendor),
		Product:  ID(desc.idProduct),
		Class:    uint8(desc.bDeviceClass),
		SubClass: uint8(desc.bDeviceSubClass),
		Protocol: uint8(desc.bDeviceProtocol),
		Configs:  cfgs,
	}, nil
}
