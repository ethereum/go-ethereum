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

type Class uint8

const (
	CLASS_PER_INTERFACE Class = C.LIBUSB_CLASS_PER_INTERFACE
	CLASS_AUDIO         Class = C.LIBUSB_CLASS_AUDIO
	CLASS_COMM          Class = C.LIBUSB_CLASS_COMM
	CLASS_HID           Class = C.LIBUSB_CLASS_HID
	CLASS_PRINTER       Class = C.LIBUSB_CLASS_PRINTER
	CLASS_PTP           Class = C.LIBUSB_CLASS_PTP
	CLASS_MASS_STORAGE  Class = C.LIBUSB_CLASS_MASS_STORAGE
	CLASS_HUB           Class = C.LIBUSB_CLASS_HUB
	CLASS_DATA          Class = C.LIBUSB_CLASS_DATA
	CLASS_WIRELESS      Class = C.LIBUSB_CLASS_WIRELESS
	CLASS_APPLICATION   Class = C.LIBUSB_CLASS_APPLICATION
	CLASS_VENDOR_SPEC   Class = C.LIBUSB_CLASS_VENDOR_SPEC
)

var classDescription = map[Class]string{
	CLASS_PER_INTERFACE: "per-interface",
	CLASS_AUDIO:         "audio",
	CLASS_COMM:          "communications",
	CLASS_HID:           "human interface device",
	CLASS_PRINTER:       "printer dclass",
	CLASS_PTP:           "picture transfer protocol",
	CLASS_MASS_STORAGE:  "mass storage",
	CLASS_HUB:           "hub",
	CLASS_DATA:          "data",
	CLASS_WIRELESS:      "wireless",
	CLASS_APPLICATION:   "application",
	CLASS_VENDOR_SPEC:   "vendor-specific",
}

func (c Class) String() string {
	return classDescription[c]
}

type DescriptorType uint8

const (
	DT_DEVICE    DescriptorType = C.LIBUSB_DT_DEVICE
	DT_CONFIG    DescriptorType = C.LIBUSB_DT_CONFIG
	DT_STRING    DescriptorType = C.LIBUSB_DT_STRING
	DT_INTERFACE DescriptorType = C.LIBUSB_DT_INTERFACE
	DT_ENDPOINT  DescriptorType = C.LIBUSB_DT_ENDPOINT
	DT_HID       DescriptorType = C.LIBUSB_DT_HID
	DT_REPORT    DescriptorType = C.LIBUSB_DT_REPORT
	DT_PHYSICAL  DescriptorType = C.LIBUSB_DT_PHYSICAL
	DT_HUB       DescriptorType = C.LIBUSB_DT_HUB
)

var descriptorTypeDescription = map[DescriptorType]string{
	DT_DEVICE:    "device",
	DT_CONFIG:    "configuration",
	DT_STRING:    "string",
	DT_INTERFACE: "interface",
	DT_ENDPOINT:  "endpoint",
	DT_HID:       "HID",
	DT_REPORT:    "HID report",
	DT_PHYSICAL:  "physical",
	DT_HUB:       "hub",
}

func (dt DescriptorType) String() string {
	return descriptorTypeDescription[dt]
}

type EndpointDirection uint8

const (
	ENDPOINT_NUM_MASK                   = 0x03
	ENDPOINT_DIR_IN   EndpointDirection = C.LIBUSB_ENDPOINT_IN
	ENDPOINT_DIR_OUT  EndpointDirection = C.LIBUSB_ENDPOINT_OUT
	ENDPOINT_DIR_MASK EndpointDirection = 0x80
)

var endpointDirectionDescription = map[EndpointDirection]string{
	ENDPOINT_DIR_IN:  "IN",
	ENDPOINT_DIR_OUT: "OUT",
}

func (ed EndpointDirection) String() string {
	return endpointDirectionDescription[ed]
}

type TransferType uint8

const (
	TRANSFER_TYPE_CONTROL     TransferType = C.LIBUSB_TRANSFER_TYPE_CONTROL
	TRANSFER_TYPE_ISOCHRONOUS TransferType = C.LIBUSB_TRANSFER_TYPE_ISOCHRONOUS
	TRANSFER_TYPE_BULK        TransferType = C.LIBUSB_TRANSFER_TYPE_BULK
	TRANSFER_TYPE_INTERRUPT   TransferType = C.LIBUSB_TRANSFER_TYPE_INTERRUPT
	TRANSFER_TYPE_MASK        TransferType = 0x03
)

var transferTypeDescription = map[TransferType]string{
	TRANSFER_TYPE_CONTROL:     "control",
	TRANSFER_TYPE_ISOCHRONOUS: "isochronous",
	TRANSFER_TYPE_BULK:        "bulk",
	TRANSFER_TYPE_INTERRUPT:   "interrupt",
}

func (tt TransferType) String() string {
	return transferTypeDescription[tt]
}

type IsoSyncType uint8

const (
	ISO_SYNC_TYPE_NONE     IsoSyncType = C.LIBUSB_ISO_SYNC_TYPE_NONE << 2
	ISO_SYNC_TYPE_ASYNC    IsoSyncType = C.LIBUSB_ISO_SYNC_TYPE_ASYNC << 2
	ISO_SYNC_TYPE_ADAPTIVE IsoSyncType = C.LIBUSB_ISO_SYNC_TYPE_ADAPTIVE << 2
	ISO_SYNC_TYPE_SYNC     IsoSyncType = C.LIBUSB_ISO_SYNC_TYPE_SYNC << 2
	ISO_SYNC_TYPE_MASK     IsoSyncType = 0x0C
)

var isoSyncTypeDescription = map[IsoSyncType]string{
	ISO_SYNC_TYPE_NONE:     "unsynchronized",
	ISO_SYNC_TYPE_ASYNC:    "asynchronous",
	ISO_SYNC_TYPE_ADAPTIVE: "adaptive",
	ISO_SYNC_TYPE_SYNC:     "synchronous",
}

func (ist IsoSyncType) String() string {
	return isoSyncTypeDescription[ist]
}

type IsoUsageType uint8

const (
	ISO_USAGE_TYPE_DATA     IsoUsageType = C.LIBUSB_ISO_USAGE_TYPE_DATA << 4
	ISO_USAGE_TYPE_FEEDBACK IsoUsageType = C.LIBUSB_ISO_USAGE_TYPE_FEEDBACK << 4
	ISO_USAGE_TYPE_IMPLICIT IsoUsageType = C.LIBUSB_ISO_USAGE_TYPE_IMPLICIT << 4
	ISO_USAGE_TYPE_MASK     IsoUsageType = 0x30
)

var isoUsageTypeDescription = map[IsoUsageType]string{
	ISO_USAGE_TYPE_DATA:     "data",
	ISO_USAGE_TYPE_FEEDBACK: "feedback",
	ISO_USAGE_TYPE_IMPLICIT: "implicit data",
}

func (iut IsoUsageType) String() string {
	return isoUsageTypeDescription[iut]
}

type RequestType uint8

const (
	REQUEST_TYPE_STANDARD = C.LIBUSB_REQUEST_TYPE_STANDARD
	REQUEST_TYPE_CLASS    = C.LIBUSB_REQUEST_TYPE_CLASS
	REQUEST_TYPE_VENDOR   = C.LIBUSB_REQUEST_TYPE_VENDOR
	REQUEST_TYPE_RESERVED = C.LIBUSB_REQUEST_TYPE_RESERVED
)

var requestTypeDescription = map[RequestType]string{
	REQUEST_TYPE_STANDARD: "standard",
	REQUEST_TYPE_CLASS:    "class",
	REQUEST_TYPE_VENDOR:   "vendor",
	REQUEST_TYPE_RESERVED: "reserved",
}

func (rt RequestType) String() string {
	return requestTypeDescription[rt]
}
