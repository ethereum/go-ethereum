// hid - Gopher Interface Devices (USB HID)
// Copyright (c) 2017 Péter Szilágyi. All rights reserved.
//
// This file is released under the 3-clause BSD license. Note however that Linux
// support depends on libusb, released under GNU LGPL 2.1 or later.

// +build !freebsd,!linux,!darwin,!windows ios !cgo

package hid

// Supported returns whether this platform is supported by the HID library or not.
// The goal of this method is to allow programatically handling platforms that do
// not support USB HID and not having to fall back to build constraints.
func Supported() bool {
	return false
}

// Enumerate returns a list of all the HID devices attached to the system which
// match the vendor and product id. On platforms that this file implements the
// function is a noop and returns an empty list always.
func Enumerate(vendorID uint16, productID uint16) []DeviceInfo {
	return nil
}

// HidDevice is a live HID USB connected device handle. On platforms that this file
// implements the type lacks the actual HID device and all methods are noop.
type HidDevice struct {
	HidDeviceInfo // Embed the infos for easier access
}

// Open connects to an HID device by its path name. On platforms that this file
// implements the method just returns an error.
func (info HidDeviceInfo) Open() (*Device, error) {
	return nil, ErrUnsupportedPlatform
}

// Close releases the HID USB device handle. On platforms that this file implements
// the method is just a noop.
func (dev *HidDevice) Close() error { return ErrUnsupportedPlatform }

// Write sends an output report to a HID device. On platforms that this file
// implements the method just returns an error.
func (dev *HidDevice) Write(b []byte) (int, error) {
	return 0, ErrUnsupportedPlatform
}

// Read retrieves an input report from a HID device. On platforms that this file
// implements the method just returns an error.
func (dev *HidDevice) Read(b []byte) (int, error) {
	return 0, ErrUnsupportedPlatform
}

// Open tries to open the USB device represented by the current DeviceInfo
func (gdi *GenericDeviceInfo) Open() (Device, error) {
	return nil, ErrUnsupportedPlatform
}

// GenericDevice represents a generic USB device
type GenericDevice struct {
	*GenericDeviceInfo // Embed the infos for easier access
}

// Write implements io.ReaderWriter
func (gd *GenericDevice) Write(b []byte) (int, error) {
	return 0, ErrUnsupportedPlatform
}

// Read implements io.ReaderWriter
func (gd *GenericDevice) Read(b []byte) (int, error) {
	return 0, ErrUnsupportedPlatform
}

// Close a previously opened generic USB device
func (gd *GenericDevice) Close() error {
	return ErrUnsupportedPlatform
}
