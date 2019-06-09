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
	#include "./libusb/libusb/libusb.h"

	// ctx is a global libusb context to interact with devices through.
	libusb_context* ctx;
*/
import "C"

import (
	"fmt"
	"reflect"
	"sync"
	"unsafe"
)

// enumerateRaw returns a list of all the USB devices attached to the system which
// match the vendor and product id:
//  - If the vendor id is set to 0 then any vendor matches.
//  - If the product id is set to 0 then any product matches.
//  - If the vendor and product id are both 0, all USB devices are returned.
func enumerateRaw(vendorID uint16, productID uint16) ([]DeviceInfo, error) {
	// Enumerate the devices, and free all the matching refcounts (we'll reopen any
	// explicitly requested).
	infos, err := enumerateRawWithRef(vendorID, productID)
	for _, info := range infos {
		C.libusb_unref_device(info.rawDevice.(*C.libusb_device))
	}
	// If enumeration failed, don't return anything, otherwise everything
	if err != nil {
		return nil, err
	}
	return infos, nil
}

// enumerateRawWithRef is the internal device enumerator that retains 1 reference
// to every matched device so they may selectively be opened on request.
func enumerateRawWithRef(vendorID uint16, productID uint16) ([]DeviceInfo, error) {
	// Ensure we have a libusb context to interact through. The enumerate call is
	// protexted by a mutex outside, so it's fine to do the below check and init.
	if C.ctx == nil {
		if err := fromRawErrno(C.libusb_init((**C.libusb_context)(&C.ctx))); err != nil {
			return nil, fmt.Errorf("failed to initialize libusb: %v", err)
		}
	}
	// Retrieve all the available USB devices and wrap them in Go
	var deviceList **C.libusb_device
	count := C.libusb_get_device_list(C.ctx, &deviceList)
	if count < 0 {
		return nil, rawError(count)
	}
	defer C.libusb_free_device_list(deviceList, 1)

	var devices []*C.libusb_device
	*(*reflect.SliceHeader)(unsafe.Pointer(&devices)) = reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(deviceList)),
		Len:  int(count),
		Cap:  int(count),
	}
	//
	var infos []DeviceInfo
	for devnum, dev := range devices {
		// Retrieve the libusb device descriptor and skip non-queried ones
		var desc C.struct_libusb_device_descriptor
		if err := fromRawErrno(C.libusb_get_device_descriptor(dev, &desc)); err != nil {
			return infos, fmt.Errorf("failed to get device %d descriptor: %v", devnum, err)
		}
		if (vendorID > 0 && uint16(desc.idVendor) != vendorID) || (productID > 0 && uint16(desc.idProduct) != productID) {
			continue
		}
		// Skip HID devices, they are handled directly by OS libraries
		if desc.bDeviceClass == C.LIBUSB_CLASS_HID {
			continue
		}
		// Iterate over all the configurations and find raw interfaces
		for cfgnum := 0; cfgnum < int(desc.bNumConfigurations); cfgnum++ {
			// Retrieve the all the possible USB configurations of the device
			var cfg *C.struct_libusb_config_descriptor
			if err := fromRawErrno(C.libusb_get_config_descriptor(dev, C.uint8_t(cfgnum), &cfg)); err != nil {
				return infos, fmt.Errorf("failed to get device %d config %d: %v", devnum, cfgnum, err)
			}
			var ifaces []C.struct_libusb_interface
			*(*reflect.SliceHeader)(unsafe.Pointer(&ifaces)) = reflect.SliceHeader{
				Data: uintptr(unsafe.Pointer(cfg._interface)),
				Len:  int(cfg.bNumInterfaces),
				Cap:  int(cfg.bNumInterfaces),
			}
			// Drill down into each advertised interface
			for ifacenum, iface := range ifaces {
				if iface.num_altsetting == 0 {
					continue
				}
				var alts []C.struct_libusb_interface_descriptor
				*(*reflect.SliceHeader)(unsafe.Pointer(&alts)) = reflect.SliceHeader{
					Data: uintptr(unsafe.Pointer(iface.altsetting)),
					Len:  int(iface.num_altsetting),
					Cap:  int(iface.num_altsetting),
				}
				for _, alt := range alts {
					// Skip HID interfaces, they are handled directly by OS libraries
					if alt.bInterfaceClass == C.LIBUSB_CLASS_HID {
						continue
					}
					// Find the endpoints that can speak libusb interrupts
					var ends []C.struct_libusb_endpoint_descriptor
					*(*reflect.SliceHeader)(unsafe.Pointer(&ends)) = reflect.SliceHeader{
						Data: uintptr(unsafe.Pointer(alt.endpoint)),
						Len:  int(alt.bNumEndpoints),
						Cap:  int(alt.bNumEndpoints),
					}
					var reader, writer *uint8
					for _, end := range ends {
						// Skip any non-interrupt endpoints
						if end.bmAttributes != C.LIBUSB_TRANSFER_TYPE_INTERRUPT {
							continue
						}
						if end.bEndpointAddress&C.LIBUSB_ENDPOINT_IN == C.LIBUSB_ENDPOINT_IN {
							reader = new(uint8)
							*reader = uint8(end.bEndpointAddress)
						} else {
							writer = new(uint8)
							*writer = uint8(end.bEndpointAddress)
						}
					}
					// If both in and out interrupts are available, match the device
					if reader != nil && writer != nil {
						// Enumeration matched, bump the device refcount to avoid cleaning it up
						C.libusb_ref_device(dev)

						port := uint8(C.libusb_get_port_number(dev))
						infos = append(infos, DeviceInfo{
							Path:      fmt.Sprintf("%04x:%04x:%02d", vendorID, uint16(desc.idProduct), port),
							VendorID:  uint16(desc.idVendor),
							ProductID: uint16(desc.idProduct),
							Interface: ifacenum,
							rawDevice: dev,
							rawPort:   &port,
							rawReader: reader,
							rawWriter: writer,
						})
					}
				}
			}
		}
	}
	return infos, nil
}

// openRaw connects to a low level libusb device by its path name.
func openRaw(info DeviceInfo) (*rawDevice, error) {
	// Enumerate all the devices matching this particular info
	matches, err := enumerateRawWithRef(info.VendorID, info.ProductID)
	if err != nil {
		// Enumeration failed, make sure any subresults are released
		for _, match := range matches {
			C.libusb_unref_device(match.rawDevice.(*C.libusb_device))
		}
		return nil, err
	}
	// Find the specific endpoint we're interested in
	var device *C.libusb_device
	for _, match := range matches {
		// Keep the matching device reference, release anything else
		if device == nil && *match.rawPort == *info.rawPort && match.Interface == info.Interface {
			device = match.rawDevice.(*C.libusb_device)
		} else {
			C.libusb_unref_device(match.rawDevice.(*C.libusb_device))
		}
	}
	if device == nil {
		return nil, fmt.Errorf("failed to open device: not found")
	}
	// Open the mathcing device
	info.rawDevice = device

	var handle *C.struct_libusb_device_handle
	if err := fromRawErrno(C.libusb_open(info.rawDevice.(*C.libusb_device), (**C.struct_libusb_device_handle)(&handle))); err != nil {
		return nil, fmt.Errorf("failed to open device: %v", err)
	}
	if err := fromRawErrno(C.libusb_claim_interface(handle, (C.int)(info.Interface))); err != nil {
		C.libusb_close(handle)
		return nil, fmt.Errorf("failed to claim interface: %v", err)
	}
	return &rawDevice{
		DeviceInfo: info,
		handle:     handle,
	}, nil
}

// rawDevice is a live low level USB connected device handle.
type rawDevice struct {
	DeviceInfo // Embed the infos for easier access

	handle *C.struct_libusb_device_handle // Low level USB device to communicate through
	lock   sync.Mutex
}

// Close releases the raw USB device handle.
func (dev *rawDevice) Close() error {
	dev.lock.Lock()
	defer dev.lock.Unlock()

	if dev.handle != nil {
		C.libusb_release_interface(dev.handle, (C.int)(dev.Interface))
		C.libusb_close(dev.handle)
		dev.handle = nil
	}
	C.libusb_unref_device(dev.rawDevice.(*C.libusb_device))

	return nil
}

// Write sends a binary blob to a low level USB device.
func (dev *rawDevice) Write(b []byte) (int, error) {
	dev.lock.Lock()
	defer dev.lock.Unlock()

	var transferred C.int
	if err := fromRawErrno(C.libusb_interrupt_transfer(dev.handle, (C.uchar)(*dev.rawWriter), (*C.uchar)(&b[0]), (C.int)(len(b)), &transferred, (C.uint)(0))); err != nil {
		return 0, fmt.Errorf("failed to write to device: %v", err)
	}
	return int(transferred), nil
}

// Read retrieves a binary blob from a low level USB device.
func (dev *rawDevice) Read(b []byte) (int, error) {
	dev.lock.Lock()
	defer dev.lock.Unlock()

	var transferred C.int
	if err := fromRawErrno(C.libusb_interrupt_transfer(dev.handle, (C.uchar)(*dev.rawReader), (*C.uchar)(&b[0]), (C.int)(len(b)), &transferred, (C.uint)(0))); err != nil {
		return 0, fmt.Errorf("failed to read from device: %v", err)
	}
	return int(transferred), nil
}
