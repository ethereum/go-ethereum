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

import (
	"sync"
)

// enumerateLock is a mutex serializing access to USB device enumeration needed
// by the macOS USB HID system calls, which require 2 consecutive method calls
// for enumeration, causing crashes if called concurrently.
//
// For more details, see:
//   https://developer.apple.com/documentation/iokit/1438371-iohidmanagersetdevicematching
//   > "subsequent calls will cause the hid manager to release previously enumerated devices"
var enumerateLock sync.Mutex

// Supported returns whether this platform is supported by the USB library or not.
// The goal of this method is to allow programatically handling platforms that do
// not support USB and not having to fall back to build constraints.
func Supported() bool {
	return true
}

// Enumerate returns a list of all the USB devices attached to the system which
// match the vendor and product id:
//  - If the vendor id is set to 0 then any vendor matches.
//  - If the product id is set to 0 then any product matches.
//  - If the vendor and product id are both 0, all devices are returned.
//
// For any device that is HID capable, the enumeration will return an interface
// to the HID endpoints. For pure raw USB access, please use EnumerateRaw.
func Enumerate(vendorID uint16, productID uint16) ([]DeviceInfo, error) {
	enumerateLock.Lock()
	defer enumerateLock.Unlock()

	// Enumerate all the raw USB devices and skip the HID ones
	raws, err := enumerateRaw(vendorID, productID)
	if err != nil {
		return nil, err
	}
	// Enumerate all the HID USB devices
	hids, err := enumerateHid(vendorID, productID)
	if err != nil {
		return nil, err
	}
	return append(raws, hids...), nil
}

// EnumerateRaw returns a list of all the USB devices attached to the system which
// match the vendor and product id:
//  - If the vendor id is set to 0 then any vendor matches.
//  - If the product id is set to 0 then any product matches.
//  - If the vendor and product id are both 0, all devices are returned.
func EnumerateRaw(vendorID uint16, productID uint16) ([]DeviceInfo, error) {
	enumerateLock.Lock()
	defer enumerateLock.Unlock()

	return enumerateRaw(vendorID, productID)
}

// EnumerateHid returns a list of all the HID devices attached to the system which
// match the vendor and product id:
//  - If the vendor id is set to 0 then any vendor matches.
//  - If the product id is set to 0 then any product matches.
//  - If the vendor and product id are both 0, all devices are returned.
func EnumerateHid(vendorID uint16, productID uint16) ([]DeviceInfo, error) {
	enumerateLock.Lock()
	defer enumerateLock.Unlock()

	return enumerateHid(vendorID, productID)
}

// Open connects to a previsouly discovered USB device.
func (info DeviceInfo) Open() (Device, error) {
	enumerateLock.Lock()
	defer enumerateLock.Unlock()

	if info.rawDevice == nil {
		return openHid(info)
	}
	return openRaw(info)
}
