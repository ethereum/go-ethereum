// hid - Gopher Interface Devices (USB HID)
// Copyright (c) 2017 Péter Szilágyi. All rights reserved.
//
// This file is released under the 3-clause BSD license. Note however that Linux
// support depends on libusb, released under LGNU GPL 2.1 or later.

// +build freebsd,cgo linux,cgo darwin,!ios,cgo windows,cgo

package hid

/*
#cgo CFLAGS: -I./hidapi/hidapi

#cgo linux CFLAGS: -I./libusb/libusb -DDEFAULT_VISIBILITY="" -DOS_LINUX -D_GNU_SOURCE -DPOLL_NFDS_TYPE=int
#cgo linux,!android LDFLAGS: -lrt
#cgo darwin CFLAGS: -DOS_DARWIN -I./libusb/libusb
#cgo darwin LDFLAGS: -framework CoreFoundation -framework IOKit -lusb-1.0.0
#cgo windows CFLAGS: -DOS_WINDOWS
#cgo windows LDFLAGS: -lsetupapi
#cgo freebsd CFLAGS: -DOS_FREEBSD
#cgo freebsd LDFLAGS: -lusb

#ifdef OS_LINUX
	#include <poll.h>
	#include "os/threads_posix.c"
	#include "os/poll_posix.c"

	#include "os/linux_usbfs.c"
	#include "os/linux_netlink.c"

	#include "core.c"
	#include "descriptor.c"
	#include "hotplug.c"
	#include "io.c"
	#include "strerror.c"
	#include "sync.c"

	#include "hidapi/libusb/hid.c"
#elif OS_DARWIN
	#include <libusb.h>
	#include "hidapi/mac/hid.c"
#elif OS_WINDOWS
	#include "hidapi/windows/hid.c"
#elif OS_FREEBSD
    #include <stdlib.h>
	#include <libusb.h>
	#include "hidapi/libusb/hid.c"
#endif

#if defined(OS_LINUX) || defined(OS_WINDOWS)
	void copy_device_list_to_slice(struct libusb_device **data, struct libusb_device **list, int count)
	{
		int i;
		struct libusb_device *current = *list;
		for (i=0; i<count; i++)
		{
			 data[i] = current;
			 current = list_entry(current->list.next, struct libusb_device, list);
		}
	}
#elif defined(OS_DARWIN) || defined(OS_FREEBSD)
	void copy_device_list_to_slice(struct libusb_device **data, struct libusb_device **list, int count)
	{
		int i;
		// No memcopy because the struct size isn't available for a sizeof()
		for (i=0; i<count; i++)
		{
			data[i] = list[i];
		}
	}
#endif

const char *usb_strerror(int err)
{
	return libusb_strerror(err);
}
*/
import "C"

import (
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"unsafe"
)

// enumerateLock is a mutex serializing access to USB device enumeration needed
// by the macOS USB HID system calls, which require 2 consecutive method calls
// for enumeration, causing crashes if called concurrently.
//
// For more details, see:
//   https://developer.apple.com/documentation/iokit/1438371-iohidmanagersetdevicematching
//   > "subsequent calls will cause the hid manager to release previously enumerated devices"
var enumerateLock sync.Mutex

// Supported returns whether this platform is supported by the HID library or not.
// The goal of this method is to allow programatically handling platforms that do
// not support USB HID and not having to fall back to build constraints.
func Supported() bool {
	return true
}

// genericEnumerate performs generic USB device enumeration
func genericEnumerate(vendorID uint16, productID uint16) ([]DeviceInfo, error) {
	var infos []DeviceInfo
	var ctx *C.struct_libusb_context
	errCode := int(C.libusb_init((**C.struct_libusb_context)(&ctx)))
	if errCode < 0 {
		return nil, fmt.Errorf("Error while initializing libusb: %d", errCode)
	}

	var deviceListPtr **C.struct_libusb_device
	count := C.libusb_get_device_list(ctx, (***C.struct_libusb_device)(&deviceListPtr))
	if count < 0 {
		return nil, fmt.Errorf("Error code listing devices: %d", count)
	}
	defer C.libusb_free_device_list(deviceListPtr, C.int(count))

	deviceList := make([]*C.struct_libusb_device, count)
	dlhdr := (*reflect.SliceHeader)(unsafe.Pointer(&deviceList))
	C.copy_device_list_to_slice((**C.struct_libusb_device)(unsafe.Pointer(dlhdr.Data)), deviceListPtr, C.int(count))

	for devnum, dev := range deviceList {
		var desc C.struct_libusb_device_descriptor
		errCode := int(C.libusb_get_device_descriptor(dev, &desc))
		if errCode < 0 {
			return nil, fmt.Errorf("Error getting device descriptor for generic device %d: %d", devnum, errCode)
		}

		// Start by checking the vendor id and the product id if necessary
		if uint16(desc.idVendor) != vendorID || !(productID == 0 || uint16(desc.idProduct) == productID) {
			continue
		}

		// Skip HID devices, they will be handled later
		switch desc.bDeviceClass {
		case 0:
			/* Device class is specified at interface level */
			for cfgnum := 0; cfgnum < int(desc.bNumConfigurations); cfgnum++ {
				var cfgdesc *C.struct_libusb_config_descriptor
				errCode = int(C.libusb_get_config_descriptor(dev, C.uint8_t(cfgnum), &cfgdesc))
				if errCode != 0 {
					return nil, fmt.Errorf("Error getting device configuration #%d for generic device %d: %d", cfgnum, devnum, errCode)
				}

				var ifs []C.struct_libusb_interface
				ifshdr := (*reflect.SliceHeader)(unsafe.Pointer(&ifs))
				ifshdr.Cap = int(cfgdesc.bNumInterfaces)
				ifshdr.Len = int(cfgdesc.bNumInterfaces)
				ifshdr.Data = uintptr(unsafe.Pointer(cfgdesc._interface))

				for ifnum, ifc := range ifs {
					var ifdescs []C.struct_libusb_interface_descriptor
					ifdshdr := (*reflect.SliceHeader)(unsafe.Pointer(&ifdescs))
					ifdshdr.Cap = int(ifc.num_altsetting)
					ifdshdr.Len = int(ifc.num_altsetting)
					ifdshdr.Data = uintptr(unsafe.Pointer(ifc.altsetting))

					for _, alt := range ifdescs {
						if alt.bInterfaceClass != 3 {
							// Device isn't a HID interface, add them to the device list.

							var endps []C.struct_libusb_endpoint_descriptor
							endpshdr := (*reflect.SliceHeader)(unsafe.Pointer(&endps))
							endpshdr.Cap = int(alt.bNumEndpoints)
							endpshdr.Len = int(alt.bNumEndpoints)
							endpshdr.Data = uintptr(unsafe.Pointer(alt.endpoint))

							endpoints := make([]GenericEndpoint, alt.bNumEndpoints)

							for ne, endpoint := range endps {
								endpoints[ne] = GenericEndpoint{
									Direction:  GenericEndpointDirection(endpoint.bEndpointAddress) & GenericEndpointDirectionIn,
									Address:    uint8(endpoint.bEndpointAddress),
									Attributes: uint8(endpoint.bmAttributes),
								}
							}

							info := &GenericDeviceInfo{
								Path:      fmt.Sprintf("%x:%x:%d", vendorID, uint16(desc.idProduct), uint8(C.libusb_get_port_number(dev))),
								VendorID:  uint16(desc.idVendor),
								ProductID: uint16(desc.idProduct),
								device: &GenericDevice{
									device: dev,
								},
								Endpoints: endpoints,
								Interface: ifnum,
							}
							info.device.GenericDeviceInfo = info
							infos = append(infos, info)
						}
					}
				}
			}
		case 3:
			// Device class is HID, skip it
			continue
		}
	}

	return infos, nil
}

// Enumerate returns a list of all the HID devices attached to the system which
// match the vendor and product id:
//  - If the vendor id is set to 0 then any vendor matches.
//  - If the product id is set to 0 then any product matches.
//  - If the vendor and product id are both 0, all HID devices are returned.
func Enumerate(vendorID uint16, productID uint16) ([]DeviceInfo, error) {
	enumerateLock.Lock()
	defer enumerateLock.Unlock()

	infos, err := genericEnumerate(vendorID, productID)

	if err != nil {
		return nil, err
	}

	// Gather all device infos and ensure they are freed before returning
	head := C.hid_enumerate(C.ushort(vendorID), C.ushort(productID))
	if head == nil {
		return nil, nil
	}
	defer C.hid_free_enumeration(head)

	// Iterate the list and retrieve the device details
	for ; head != nil; head = head.next {
		info := &HidDeviceInfo{
			Path:      C.GoString(head.path),
			VendorID:  uint16(head.vendor_id),
			ProductID: uint16(head.product_id),
			Release:   uint16(head.release_number),
			UsagePage: uint16(head.usage_page),
			Usage:     uint16(head.usage),
			Interface: int(head.interface_number),
		}
		if head.serial_number != nil {
			info.Serial, _ = wcharTToString(head.serial_number)
		}
		if head.product_string != nil {
			info.Product, _ = wcharTToString(head.product_string)
		}
		if head.manufacturer_string != nil {
			info.Manufacturer, _ = wcharTToString(head.manufacturer_string)
		}
		infos = append(infos, info)
	}
	return infos, nil
}

// Open connects to an HID device by its path name.
func (info *HidDeviceInfo) Open() (Device, error) {
	enumerateLock.Lock()
	defer enumerateLock.Unlock()

	path := C.CString(info.Path)
	defer C.free(unsafe.Pointer(path))

	device := C.hid_open_path(path)
	if device == nil {
		return nil, errors.New("hidapi: failed to open device")
	}
	return &HidDevice{
		DeviceInfo: info,
		device:     device,
	}, nil
}

// HidDevice is a live HID USB connected device handle.
type HidDevice struct {
	DeviceInfo // Embed the infos for easier access

	device *C.hid_device // Low level HID device to communicate through
	lock   sync.Mutex
}

// Close releases the HID USB device handle.
func (dev *HidDevice) Close() error {
	dev.lock.Lock()
	defer dev.lock.Unlock()

	if dev.device != nil {
		C.hid_close(dev.device)
		dev.device = nil
	}
	return nil
}

// Write sends an output report to a HID device.
//
// Write will send the data on the first OUT endpoint, if one exists. If it does
// not, it will send the data through the Control Endpoint (Endpoint 0).
func (dev *HidDevice) Write(b []byte) (int, error) {
	// Abort if nothing to write
	if len(b) == 0 {
		return 0, nil
	}
	// Abort if device closed in between
	dev.lock.Lock()
	device := dev.device
	dev.lock.Unlock()

	if device == nil {
		return 0, ErrDeviceClosed
	}
	// Prepend a HID report ID on Windows, other OSes don't need it
	var report []byte
	if runtime.GOOS == "windows" {
		report = append([]byte{0x00}, b...)
	} else {
		report = b
	}
	// Execute the write operation
	written := int(C.hid_write(device, (*C.uchar)(&report[0]), C.size_t(len(report))))
	if written == -1 {
		// If the write failed, verify if closed or other error
		dev.lock.Lock()
		device = dev.device
		dev.lock.Unlock()

		if device == nil {
			return 0, ErrDeviceClosed
		}
		// Device not closed, some other error occurred
		message := C.hid_error(device)
		if message == nil {
			return 0, errors.New("hidapi: unknown failure")
		}
		failure, _ := wcharTToString(message)
		return 0, errors.New("hidapi: " + failure)
	}
	return written, nil
}

// Read retrieves an input report from a HID device.
func (dev *HidDevice) Read(b []byte) (int, error) {
	// Aborth if nothing to read
	if len(b) == 0 {
		return 0, nil
	}
	// Abort if device closed in between
	dev.lock.Lock()
	device := dev.device
	dev.lock.Unlock()

	if device == nil {
		return 0, ErrDeviceClosed
	}
	// Execute the read operation
	read := int(C.hid_read(device, (*C.uchar)(&b[0]), C.size_t(len(b))))
	if read == -1 {
		// If the read failed, verify if closed or other error
		dev.lock.Lock()
		device = dev.device
		dev.lock.Unlock()

		if device == nil {
			return 0, ErrDeviceClosed
		}
		// Device not closed, some other error occurred
		message := C.hid_error(device)
		if message == nil {
			return 0, errors.New("hidapi: unknown failure")
		}
		failure, _ := wcharTToString(message)
		return 0, errors.New("hidapi: " + failure)
	}
	return read, nil
}

// Type identify the device as a HID device
func (dev *HidDevice) Type() DeviceType {
	return dev.DeviceInfo.Type()
}

// Open tries to open the USB device represented by the current DeviceInfo
func (gdi *GenericDeviceInfo) Open() (Device, error) {
	var handle *C.struct_libusb_device_handle
	errCode := int(C.libusb_open(gdi.device.device, (**C.struct_libusb_device_handle)(&handle)))
	if errCode < 0 {
		return nil, fmt.Errorf("Error opening generic USB device %v, code %d", gdi.device.handle, errCode)
	}

	gdi.device.handle = handle
	// QUESTION: ai-je deja initialie le GDI ?
	// 	GenericDeviceInfo: gdi,
	// 	handle:            handle,
	// }

	for _, endpoint := range gdi.Endpoints {
		switch {
		case endpoint.Direction == GenericEndpointDirectionOut && endpoint.Attributes == GenericEndpointAttributeInterrupt:
			gdi.device.WEndpoint = endpoint.Address
		case endpoint.Direction == GenericEndpointDirectionIn && endpoint.Attributes == GenericEndpointAttributeInterrupt:
			gdi.device.REndpoint = endpoint.Address
		}
	}

	if gdi.device.REndpoint == 0 || gdi.device.WEndpoint == 0 {
		return nil, fmt.Errorf("Missing endpoint in device %#x:%#x:%d", gdi.VendorID, gdi.ProductID, gdi.Interface)
	}

	return gdi.device, nil
}

// GenericDevice represents a generic USB device
type GenericDevice struct {
	*GenericDeviceInfo // Embed the infos for easier access

	REndpoint uint8
	WEndpoint uint8

	device *C.struct_libusb_device
	handle *C.struct_libusb_device_handle
	lock   sync.Mutex
}

// Write implements io.ReaderWriter
func (gd *GenericDevice) Write(b []byte) (int, error) {
	gd.lock.Lock()
	defer gd.lock.Unlock()

	out, err := interruptTransfer(gd.handle, gd.WEndpoint, b)
	return len(out), err
}

// Read implements io.ReaderWriter
func (gd *GenericDevice) Read(b []byte) (int, error) {
	gd.lock.Lock()
	defer gd.lock.Unlock()

	out, err := interruptTransfer(gd.handle, gd.REndpoint, b)
	return len(out), err
}

// Close a previously opened generic USB device
func (gd *GenericDevice) Close() error {
	gd.lock.Lock()
	defer gd.lock.Unlock()

	if gd.handle != nil {
		C.libusb_close(gd.handle)
		gd.handle = nil
	}

	return nil
}

// interruptTransfer is a helpler function for libusb's interrupt transfer function
func interruptTransfer(handle *C.struct_libusb_device_handle, endpoint uint8, data []byte) ([]byte, error) {
	var transferred C.int
	errCode := int(C.libusb_interrupt_transfer(handle, (C.uchar)(endpoint), (*C.uchar)(&data[0]), (C.int)(len(data)), &transferred, (C.uint)(0)))
	if errCode != 0 {
		return nil, fmt.Errorf("Interrupt transfer error: %s", C.GoString(C.usb_strerror(C.int(errCode))))
	}
	return data[:int(transferred)], nil
}
