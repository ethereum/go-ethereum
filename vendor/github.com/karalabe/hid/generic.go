// hid - Gopher Interface Devices (USB HID)
// Copyright (c) 2019 Péter Szilágyi, Guillaume Ballet. All rights reserved.

package hid

import (
	"C"
)

type GenericEndpointDirection uint8

// List of endpoint direction types
const (
	GenericEndpointDirectionOut = 0x00
	GenericEndpointDirectionIn  = 0x80
)

// List of endpoint attributes
const (
	GenericEndpointAttributeInterrupt = 3
)

// GenericEndpoint represents a USB endpoint
type GenericEndpoint struct {
	Address    uint8
	Direction  GenericEndpointDirection
	Attributes uint8
}

type GenericDeviceInfo struct {
	Path      string // Platform-specific device path
	VendorID  uint16 // Device Vendor ID
	ProductID uint16 // Device Product ID

	device *GenericDevice

	Interface int

	Endpoints []GenericEndpoint
}

func (gdi *GenericDeviceInfo) Type() DeviceType {
	return DeviceTypeGeneric
}

// Platform-specific device path
func (gdi *GenericDeviceInfo) GetPath() string {
	return gdi.Path
}

// IDs returns the vendor and product IDs for the device
func (gdi *GenericDeviceInfo) IDs() (uint16, uint16, int, uint16) {
	return gdi.VendorID, gdi.ProductID, gdi.Interface, 0
}
