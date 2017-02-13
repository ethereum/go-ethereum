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
	"unsafe"
)

type EndpointInfo struct {
	Address       uint8
	Attributes    uint8
	MaxPacketSize uint16
	MaxIsoPacket  uint32
	PollInterval  uint8
	RefreshRate   uint8
	SynchAddress  uint8
}

func (e EndpointInfo) Number() int {
	return int(e.Address) & ENDPOINT_NUM_MASK
}

func (e EndpointInfo) Direction() EndpointDirection {
	return EndpointDirection(e.Address) & ENDPOINT_DIR_MASK
}

func (e EndpointInfo) String() string {
	return fmt.Sprintf("Endpoint %d %-3s %s - %s %s [%d %d]",
		e.Number(), e.Direction(),
		TransferType(e.Attributes)&TRANSFER_TYPE_MASK,
		IsoSyncType(e.Attributes)&ISO_SYNC_TYPE_MASK,
		IsoUsageType(e.Attributes)&ISO_USAGE_TYPE_MASK,
		e.MaxPacketSize, e.MaxIsoPacket,
	)
}

type InterfaceInfo struct {
	Number uint8
	Setups []InterfaceSetup
}

func (i InterfaceInfo) String() string {
	return fmt.Sprintf("Interface %02x (%d setups)", i.Number, len(i.Setups))
}

type InterfaceSetup struct {
	Number     uint8
	Alternate  uint8
	IfClass    uint8
	IfSubClass uint8
	IfProtocol uint8
	Endpoints  []EndpointInfo
}

func (a InterfaceSetup) String() string {
	return fmt.Sprintf("Interface %02x Setup %02x", a.Number, a.Alternate)
}

type ConfigInfo struct {
	Config     uint8
	Attributes uint8
	MaxPower   uint8
	Interfaces []InterfaceInfo
}

func (c ConfigInfo) String() string {
	return fmt.Sprintf("Config %02x", c.Config)
}

func newConfig(dev *C.libusb_device, cfg *C.struct_libusb_config_descriptor) ConfigInfo {
	c := ConfigInfo{
		Config:     uint8(cfg.bConfigurationValue),
		Attributes: uint8(cfg.bmAttributes),
		MaxPower:   uint8(cfg.MaxPower),
	}

	var ifaces []C.struct_libusb_interface
	*(*reflect.SliceHeader)(unsafe.Pointer(&ifaces)) = reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(cfg._interface)),
		Len:  int(cfg.bNumInterfaces),
		Cap:  int(cfg.bNumInterfaces),
	}
	c.Interfaces = make([]InterfaceInfo, 0, len(ifaces))
	for _, iface := range ifaces {
		if iface.num_altsetting == 0 {
			continue
		}

		var alts []C.struct_libusb_interface_descriptor
		*(*reflect.SliceHeader)(unsafe.Pointer(&alts)) = reflect.SliceHeader{
			Data: uintptr(unsafe.Pointer(iface.altsetting)),
			Len:  int(iface.num_altsetting),
			Cap:  int(iface.num_altsetting),
		}
		descs := make([]InterfaceSetup, 0, len(alts))
		for _, alt := range alts {
			i := InterfaceSetup{
				Number:     uint8(alt.bInterfaceNumber),
				Alternate:  uint8(alt.bAlternateSetting),
				IfClass:    uint8(alt.bInterfaceClass),
				IfSubClass: uint8(alt.bInterfaceSubClass),
				IfProtocol: uint8(alt.bInterfaceProtocol),
			}
			var ends []C.struct_libusb_endpoint_descriptor
			*(*reflect.SliceHeader)(unsafe.Pointer(&ends)) = reflect.SliceHeader{
				Data: uintptr(unsafe.Pointer(alt.endpoint)),
				Len:  int(alt.bNumEndpoints),
				Cap:  int(alt.bNumEndpoints),
			}
			i.Endpoints = make([]EndpointInfo, 0, len(ends))
			for _, end := range ends {
				i.Endpoints = append(i.Endpoints, EndpointInfo{
					Address:       uint8(end.bEndpointAddress),
					Attributes:    uint8(end.bmAttributes),
					MaxPacketSize: uint16(end.wMaxPacketSize),
					//MaxIsoPacket:  uint32(C.libusb_get_max_iso_packet_size(dev, C.uchar(end.bEndpointAddress))),
					PollInterval: uint8(end.bInterval),
					RefreshRate:  uint8(end.bRefresh),
					SynchAddress: uint8(end.bSynchAddress),
				})
			}
			descs = append(descs, i)
		}
		c.Interfaces = append(c.Interfaces, InterfaceInfo{
			Number: descs[0].Number,
			Setups: descs,
		})
	}
	return c
}
