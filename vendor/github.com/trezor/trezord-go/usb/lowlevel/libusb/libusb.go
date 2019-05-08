// +build linux,cgo freebsd,cgo darwin,!ios,cgo windows,cgo

//-----------------------------------------------------------------------------
/*

Golang wrapper for libusb-1.0

Copyright (c) 2017 Jason T. Harris

*/
//-----------------------------------------------------------------------------

// Package libusb provides go wrappers for libusb-1.0
package libusb

/*

extern void goLibusbLog(const char *s);

#define ENABLE_LOGGING 1
#define ENABLE_DEBUG_LOGGING 1
#define ENUM_DEBUG

#cgo CFLAGS: -I./c

#cgo linux CFLAGS: -DDEFAULT_VISIBILITY="" -DOS_LINUX -D_GNU_SOURCE -DPOLL_NFDS_TYPE=int
#cgo linux,!android LDFLAGS: -lrt
#cgo freebsd CFLAGS: -DOS_FREEBSD
#cgo freebsd LDFLAGS: -lusb
#cgo darwin CFLAGS: -DOS_DARWIN -DDEFAULT_VISIBILITY="" -DPOLL_NFDS_TYPE="unsigned int"
#cgo darwin LDFLAGS: -framework CoreFoundation -framework IOKit -lobjc
#cgo windows CFLAGS: -DOS_WINDOWS -DDEFAULT_VISIBILITY="" -DPOLL_NFDS_TYPE="unsigned int"
#cgo windows LDFLAGS: -lsetupapi


#ifdef OS_LINUX
	#include <sys/poll.h>

	#include "os/threads_posix.c"
	#include "os/poll_posix.c"
	#include "os/linux_usbfs.c"
	#include "os/linux_netlink.c"
#elif OS_FREEBSD
	#include <stdlib.h>
#elif OS_DARWIN
	#include <sys/poll.h>

	#include "os/threads_posix.c"
	#include "os/poll_posix.c"
	#include "os/darwin_usb.c"
#elif OS_WINDOWS
	#define HARDCODED_LIBUSB_DEVICE_FILTER "VID_1209"

	#include <oledlg.h>

	#include "os/poll_windows.c"
	#include "os/threads_windows.c"
#endif

#ifndef OS_FREEBSD
	#include "core.c"
	#include "descriptor.c"
	#include "hotplug.c"
	#include "io.c"
	#include "strerror.c"
	#include "sync.c"
#else
	#include <libusb.h>
#endif

#ifdef OS_WINDOWS
	#include "os/windows_nt_common.c"
	#include "os/windows_winusb.c"
#endif

#cgo freebsd LDFLAGS: -lusb

#ifndef __FreeBSD__
#include "libusb.h"
#else
#include <libusb.h>

// "fake" function so freebsd builds
void libusb_cancel_sync_transfers_on_device(struct libusb_device_handle *dev_handle) {
}
#endif

// When a C struct ends with a zero-sized field, but the struct itself is not zero-sized,
// Go code can no longer refer to the zero-sized field. Any such references will have to be rewritten.
// https://golang.org/doc/go1.5#cgo
// https://github.com/golang/go/issues/11925

static uint8_t *dev_capability_data_ptr(struct libusb_bos_dev_capability_descriptor *x) {
  return &x->dev_capability_data[0];
}

static struct libusb_bos_dev_capability_descriptor **dev_capability_ptr(struct libusb_bos_descriptor *x) {
  return &x->dev_capability[0];
}

*/
import "C"

import (
	"fmt"
	"reflect"
	"strings"
	"unsafe"
)

//-----------------------------------------------------------------------------
// utilities

func bcd2str(x uint16) string {
	if (x>>12)&15 != 0 {
		return fmt.Sprintf("%d%d.%d%d", (x>>12)&15, (x>>8)&15, (x>>4)&15, (x>>0)&15)
	} else {
		return fmt.Sprintf("%d.%d%d", (x>>8)&15, (x>>4)&15, (x>>0)&15)
	}
}

func indent(s string) string {
	x := strings.Split(s, "\n")
	for i, _ := range x {
		x[i] = fmt.Sprintf("%s%s", "  ", x[i])
	}
	return strings.Join(x, "\n")
}

// return a string for the extra buffer
func Extra_str(x []byte) string {
	s := make([]string, len(x))
	for i, v := range x {
		s[i] = fmt.Sprintf("%02x", v)
	}
	return fmt.Sprintf("[%s]", strings.Join(s, " "))
}

//-----------------------------------------------------------------------------

// libusb API version.
const API_VERSION = C.LIBUSB_API_VERSION

// Device and/or Interface Class codes.
const (
	CLASS_PER_INTERFACE       = C.LIBUSB_CLASS_PER_INTERFACE
	CLASS_AUDIO               = C.LIBUSB_CLASS_AUDIO
	CLASS_COMM                = C.LIBUSB_CLASS_COMM
	CLASS_HID                 = C.LIBUSB_CLASS_HID
	CLASS_PHYSICAL            = C.LIBUSB_CLASS_PHYSICAL
	CLASS_PRINTER             = C.LIBUSB_CLASS_PRINTER
	CLASS_PTP                 = C.LIBUSB_CLASS_PTP
	CLASS_IMAGE               = C.LIBUSB_CLASS_IMAGE
	CLASS_MASS_STORAGE        = C.LIBUSB_CLASS_MASS_STORAGE
	CLASS_HUB                 = C.LIBUSB_CLASS_HUB
	CLASS_DATA                = C.LIBUSB_CLASS_DATA
	CLASS_SMART_CARD          = C.LIBUSB_CLASS_SMART_CARD
	CLASS_CONTENT_SECURITY    = C.LIBUSB_CLASS_CONTENT_SECURITY
	CLASS_VIDEO               = C.LIBUSB_CLASS_VIDEO
	CLASS_PERSONAL_HEALTHCARE = C.LIBUSB_CLASS_PERSONAL_HEALTHCARE
	CLASS_DIAGNOSTIC_DEVICE   = C.LIBUSB_CLASS_DIAGNOSTIC_DEVICE
	CLASS_WIRELESS            = C.LIBUSB_CLASS_WIRELESS
	CLASS_APPLICATION         = C.LIBUSB_CLASS_APPLICATION
	CLASS_VENDOR_SPEC         = C.LIBUSB_CLASS_VENDOR_SPEC
)

// Descriptor types as defined by the USB specification.
const (
	DT_DEVICE                = C.LIBUSB_DT_DEVICE
	DT_CONFIG                = C.LIBUSB_DT_CONFIG
	DT_STRING                = C.LIBUSB_DT_STRING
	DT_INTERFACE             = C.LIBUSB_DT_INTERFACE
	DT_ENDPOINT              = C.LIBUSB_DT_ENDPOINT
	DT_BOS                   = C.LIBUSB_DT_BOS
	DT_DEVICE_CAPABILITY     = C.LIBUSB_DT_DEVICE_CAPABILITY
	DT_HID                   = C.LIBUSB_DT_HID
	DT_REPORT                = C.LIBUSB_DT_REPORT
	DT_PHYSICAL              = C.LIBUSB_DT_PHYSICAL
	DT_HUB                   = C.LIBUSB_DT_HUB
	DT_SUPERSPEED_HUB        = C.LIBUSB_DT_SUPERSPEED_HUB
	DT_SS_ENDPOINT_COMPANION = C.LIBUSB_DT_SS_ENDPOINT_COMPANION
)

// Descriptor sizes per descriptor type.
const DT_DEVICE_SIZE = C.LIBUSB_DT_DEVICE_SIZE
const DT_CONFIG_SIZE = C.LIBUSB_DT_CONFIG_SIZE
const DT_INTERFACE_SIZE = C.LIBUSB_DT_INTERFACE_SIZE
const DT_ENDPOINT_SIZE = C.LIBUSB_DT_ENDPOINT_SIZE
const DT_ENDPOINT_AUDIO_SIZE = C.LIBUSB_DT_ENDPOINT_AUDIO_SIZE
const DT_HUB_NONVAR_SIZE = C.LIBUSB_DT_HUB_NONVAR_SIZE
const DT_SS_ENDPOINT_COMPANION_SIZE = C.LIBUSB_DT_SS_ENDPOINT_COMPANION_SIZE
const DT_BOS_SIZE = C.LIBUSB_DT_BOS_SIZE
const DT_DEVICE_CAPABILITY_SIZE = C.LIBUSB_DT_DEVICE_CAPABILITY_SIZE

// BOS descriptor sizes.
const BT_USB_2_0_EXTENSION_SIZE = C.LIBUSB_BT_USB_2_0_EXTENSION_SIZE
const BT_SS_USB_DEVICE_CAPABILITY_SIZE = C.LIBUSB_BT_SS_USB_DEVICE_CAPABILITY_SIZE
const BT_CONTAINER_ID_SIZE = C.LIBUSB_BT_CONTAINER_ID_SIZE
const DT_BOS_MAX_SIZE = C.LIBUSB_DT_BOS_MAX_SIZE
const ENDPOINT_ADDRESS_MASK = C.LIBUSB_ENDPOINT_ADDRESS_MASK
const ENDPOINT_DIR_MASK = C.LIBUSB_ENDPOINT_DIR_MASK

// Endpoint direction. Values for bit 7 of Endpoint_Descriptor.BEndpointAddress.
const (
	ENDPOINT_IN  = C.LIBUSB_ENDPOINT_IN  // In: device-to-host.
	ENDPOINT_OUT = C.LIBUSB_ENDPOINT_OUT // Out: host-to-device.
)

// in BmAttributes
const TRANSFER_TYPE_MASK = C.LIBUSB_TRANSFER_TYPE_MASK

// Endpoint transfer type. Values for bits 0:1 of Endpoint_Descriptor.BmAttributes.
const (
	TRANSFER_TYPE_CONTROL     = C.LIBUSB_TRANSFER_TYPE_CONTROL
	TRANSFER_TYPE_ISOCHRONOUS = C.LIBUSB_TRANSFER_TYPE_ISOCHRONOUS
	TRANSFER_TYPE_BULK        = C.LIBUSB_TRANSFER_TYPE_BULK
	TRANSFER_TYPE_INTERRUPT   = C.LIBUSB_TRANSFER_TYPE_INTERRUPT
	TRANSFER_TYPE_BULK_STREAM = C.LIBUSB_TRANSFER_TYPE_BULK_STREAM
)

// Standard requests, as defined in table 9-5 of the USB 3.0 specifications.
const (
	REQUEST_GET_STATUS        = C.LIBUSB_REQUEST_GET_STATUS
	REQUEST_CLEAR_FEATURE     = C.LIBUSB_REQUEST_CLEAR_FEATURE
	REQUEST_SET_FEATURE       = C.LIBUSB_REQUEST_SET_FEATURE
	REQUEST_SET_ADDRESS       = C.LIBUSB_REQUEST_SET_ADDRESS
	REQUEST_GET_DESCRIPTOR    = C.LIBUSB_REQUEST_GET_DESCRIPTOR
	REQUEST_SET_DESCRIPTOR    = C.LIBUSB_REQUEST_SET_DESCRIPTOR
	REQUEST_GET_CONFIGURATION = C.LIBUSB_REQUEST_GET_CONFIGURATION
	REQUEST_SET_CONFIGURATION = C.LIBUSB_REQUEST_SET_CONFIGURATION
	REQUEST_GET_INTERFACE     = C.LIBUSB_REQUEST_GET_INTERFACE
	REQUEST_SET_INTERFACE     = C.LIBUSB_REQUEST_SET_INTERFACE
	REQUEST_SYNCH_FRAME       = C.LIBUSB_REQUEST_SYNCH_FRAME
	REQUEST_SET_SEL           = C.LIBUSB_REQUEST_SET_SEL
	SET_ISOCH_DELAY           = C.LIBUSB_SET_ISOCH_DELAY
)

// Request type bits of Control_Setup.BmRequestType.
const (
	REQUEST_TYPE_STANDARD = C.LIBUSB_REQUEST_TYPE_STANDARD
	REQUEST_TYPE_CLASS    = C.LIBUSB_REQUEST_TYPE_CLASS
	REQUEST_TYPE_VENDOR   = C.LIBUSB_REQUEST_TYPE_VENDOR
	REQUEST_TYPE_RESERVED = C.LIBUSB_REQUEST_TYPE_RESERVED
)

// Recipient bits of Control_Setup.BmRequestType in control transfers.
// Values 4 through 31 are reserved.
const (
	RECIPIENT_DEVICE    = C.LIBUSB_RECIPIENT_DEVICE
	RECIPIENT_INTERFACE = C.LIBUSB_RECIPIENT_INTERFACE
	RECIPIENT_ENDPOINT  = C.LIBUSB_RECIPIENT_ENDPOINT
	RECIPIENT_OTHER     = C.LIBUSB_RECIPIENT_OTHER
)

const ISO_SYNC_TYPE_MASK = C.LIBUSB_ISO_SYNC_TYPE_MASK

// Synchronization type for isochronous endpoints.
// Values for bits 2:3 of Endpoint_Descriptor.BmAttributes.
const (
	ISO_SYNC_TYPE_NONE     = C.LIBUSB_ISO_SYNC_TYPE_NONE
	ISO_SYNC_TYPE_ASYNC    = C.LIBUSB_ISO_SYNC_TYPE_ASYNC
	ISO_SYNC_TYPE_ADAPTIVE = C.LIBUSB_ISO_SYNC_TYPE_ADAPTIVE
	ISO_SYNC_TYPE_SYNC     = C.LIBUSB_ISO_SYNC_TYPE_SYNC
)

const ISO_USAGE_TYPE_MASK = C.LIBUSB_ISO_USAGE_TYPE_MASK

// Usage type for isochronous endpoints.
// Values for bits 4:5 of Endpoint_Descriptor.BmAttributes.
const (
	ISO_USAGE_TYPE_DATA     = C.LIBUSB_ISO_USAGE_TYPE_DATA
	ISO_USAGE_TYPE_FEEDBACK = C.LIBUSB_ISO_USAGE_TYPE_FEEDBACK
	ISO_USAGE_TYPE_IMPLICIT = C.LIBUSB_ISO_USAGE_TYPE_IMPLICIT
)

const CONTROL_SETUP_SIZE = C.LIBUSB_CONTROL_SETUP_SIZE

// Speed codes. Indicates the speed at which the device is operating.
const (
	SPEED_UNKNOWN = C.LIBUSB_SPEED_UNKNOWN
	SPEED_LOW     = C.LIBUSB_SPEED_LOW
	SPEED_FULL    = C.LIBUSB_SPEED_FULL
	SPEED_HIGH    = C.LIBUSB_SPEED_HIGH
	SPEED_SUPER   = C.LIBUSB_SPEED_SUPER
)

// Supported speeds (WSpeedSupported) bitfield. Indicates what speeds the device supports.
const (
	LOW_SPEED_OPERATION   = C.LIBUSB_LOW_SPEED_OPERATION
	FULL_SPEED_OPERATION  = C.LIBUSB_FULL_SPEED_OPERATION
	HIGH_SPEED_OPERATION  = C.LIBUSB_HIGH_SPEED_OPERATION
	SUPER_SPEED_OPERATION = C.LIBUSB_SUPER_SPEED_OPERATION
)

// Bitmasks for USB_2_0_Extension_Descriptor.BmAttributes.
const (
	BM_LPM_SUPPORT = C.LIBUSB_BM_LPM_SUPPORT
)

// Bitmasks for SS_USB_Device_Capability_Descriptor.BmAttributes.
const (
	BM_LTM_SUPPORT = C.LIBUSB_BM_LTM_SUPPORT
)

// USB capability types.
const (
	BT_WIRELESS_USB_DEVICE_CAPABILITY = C.LIBUSB_BT_WIRELESS_USB_DEVICE_CAPABILITY
	BT_USB_2_0_EXTENSION              = C.LIBUSB_BT_USB_2_0_EXTENSION
	BT_SS_USB_DEVICE_CAPABILITY       = C.LIBUSB_BT_SS_USB_DEVICE_CAPABILITY
	BT_CONTAINER_ID                   = C.LIBUSB_BT_CONTAINER_ID
)

// Error codes.
const (
	SUCCESS             = C.LIBUSB_SUCCESS
	ERROR_IO            = C.LIBUSB_ERROR_IO
	ERROR_INVALID_PARAM = C.LIBUSB_ERROR_INVALID_PARAM
	ERROR_ACCESS        = C.LIBUSB_ERROR_ACCESS
	ERROR_NO_DEVICE     = C.LIBUSB_ERROR_NO_DEVICE
	ERROR_NOT_FOUND     = C.LIBUSB_ERROR_NOT_FOUND
	ERROR_BUSY          = C.LIBUSB_ERROR_BUSY
	ERROR_TIMEOUT       = C.LIBUSB_ERROR_TIMEOUT
	ERROR_OVERFLOW      = C.LIBUSB_ERROR_OVERFLOW
	ERROR_PIPE          = C.LIBUSB_ERROR_PIPE
	ERROR_INTERRUPTED   = C.LIBUSB_ERROR_INTERRUPTED
	ERROR_NO_MEM        = C.LIBUSB_ERROR_NO_MEM
	ERROR_NOT_SUPPORTED = C.LIBUSB_ERROR_NOT_SUPPORTED
	ERROR_OTHER         = C.LIBUSB_ERROR_OTHER
)

// Total number of error codes.
const ERROR_COUNT = C.LIBUSB_ERROR_COUNT

// Transfer status codes.
const (
	TRANSFER_COMPLETED = C.LIBUSB_TRANSFER_COMPLETED
	TRANSFER_ERROR     = C.LIBUSB_TRANSFER_ERROR
	TRANSFER_TIMED_OUT = C.LIBUSB_TRANSFER_TIMED_OUT
	TRANSFER_CANCELLED = C.LIBUSB_TRANSFER_CANCELLED
	TRANSFER_STALL     = C.LIBUSB_TRANSFER_STALL
	TRANSFER_NO_DEVICE = C.LIBUSB_TRANSFER_NO_DEVICE
	TRANSFER_OVERFLOW  = C.LIBUSB_TRANSFER_OVERFLOW
)

// Transfer.Flags values.
const (
	TRANSFER_SHORT_NOT_OK    = C.LIBUSB_TRANSFER_SHORT_NOT_OK
	TRANSFER_FREE_BUFFER     = C.LIBUSB_TRANSFER_FREE_BUFFER
	TRANSFER_FREE_TRANSFER   = C.LIBUSB_TRANSFER_FREE_TRANSFER
	TRANSFER_ADD_ZERO_PACKET = C.LIBUSB_TRANSFER_ADD_ZERO_PACKET
)

// Capabilities supported by an instance of libusb on the current running platform.
// Test if the loaded library supports a given capability by calling Has_Capability().
const (
	CAP_HAS_CAPABILITY                = C.LIBUSB_CAP_HAS_CAPABILITY
	CAP_HAS_HOTPLUG                   = C.LIBUSB_CAP_HAS_HOTPLUG
	CAP_HAS_HID_ACCESS                = C.LIBUSB_CAP_HAS_HID_ACCESS
	CAP_SUPPORTS_DETACH_KERNEL_DRIVER = C.LIBUSB_CAP_SUPPORTS_DETACH_KERNEL_DRIVER
)

// Log message levels.
const (
	LOG_LEVEL_NONE    = C.LIBUSB_LOG_LEVEL_NONE
	LOG_LEVEL_ERROR   = C.LIBUSB_LOG_LEVEL_ERROR
	LOG_LEVEL_WARNING = C.LIBUSB_LOG_LEVEL_WARNING
	LOG_LEVEL_INFO    = C.LIBUSB_LOG_LEVEL_INFO
	LOG_LEVEL_DEBUG   = C.LIBUSB_LOG_LEVEL_DEBUG
)

// Flags for hotplug events.
const (
	//HOTPLUG_NO_FLAGS  = C.LIBUSB_HOTPLUG_NO_FLAGS
	HOTPLUG_ENUMERATE = C.LIBUSB_HOTPLUG_ENUMERATE
)

// Hotplug events.
const (
	HOTPLUG_EVENT_DEVICE_ARRIVED = C.LIBUSB_HOTPLUG_EVENT_DEVICE_ARRIVED
	HOTPLUG_EVENT_DEVICE_LEFT    = C.LIBUSB_HOTPLUG_EVENT_DEVICE_LEFT
)

// Wildcard matching for hotplug events.
const HOTPLUG_MATCH_ANY = C.LIBUSB_HOTPLUG_MATCH_ANY

//-----------------------------------------------------------------------------

// A structure representing the standard USB endpoint descriptor.
// This descriptor is documented in section 9.6.6 of the USB 3.0 specification.
// All multiple-byte fields are represented in host-endian format.
type Endpoint_Descriptor struct {
	ptr              *C.struct_libusb_endpoint_descriptor
	BLength          uint8
	BDescriptorType  uint8
	BEndpointAddress uint8
	BmAttributes     uint8
	WMaxPacketSize   uint16
	BInterval        uint8
	BRefresh         uint8
	BSynchAddress    uint8
	Extra            []byte
}

func (x *C.struct_libusb_endpoint_descriptor) c2go() *Endpoint_Descriptor {
	return &Endpoint_Descriptor{
		ptr:              x,
		BLength:          uint8(x.bLength),
		BDescriptorType:  uint8(x.bDescriptorType),
		BEndpointAddress: uint8(x.bEndpointAddress),
		BmAttributes:     uint8(x.bmAttributes),
		WMaxPacketSize:   uint16(x.wMaxPacketSize),
		BInterval:        uint8(x.bInterval),
		BRefresh:         uint8(x.bRefresh),
		BSynchAddress:    uint8(x.bSynchAddress),
		Extra:            C.GoBytes(unsafe.Pointer(x.extra), x.extra_length),
	}
}

// return a string for an Endpoint_Descriptor
func (x *Endpoint_Descriptor) String() string {
	s := make([]string, 0, 16)
	s = append(s, fmt.Sprintf("bLength %d", x.BLength))
	s = append(s, fmt.Sprintf("bDescriptorType %d", x.BDescriptorType))
	s = append(s, fmt.Sprintf("bEndpointAddress 0x%02x", x.BEndpointAddress))
	s = append(s, fmt.Sprintf("bmAttributes %d", x.BmAttributes))
	s = append(s, fmt.Sprintf("wMaxPacketSize %d", x.WMaxPacketSize))
	s = append(s, fmt.Sprintf("bInterval %d", x.BInterval))
	s = append(s, fmt.Sprintf("bRefresh %d", x.BRefresh))
	s = append(s, fmt.Sprintf("bSynchAddress %d", x.BSynchAddress))
	s = append(s, fmt.Sprintf("extra %s", Extra_str(x.Extra)))
	return strings.Join(s, "\n")
}

//-----------------------------------------------------------------------------

// A structure representing the standard USB interface descriptor.
// This descriptor is documented in section 9.6.5 of the USB 3.0 specification.
// All multiple-byte fields are represented in host-endian format.
type Interface_Descriptor struct {
	ptr                *C.struct_libusb_interface_descriptor
	BLength            uint8
	BDescriptorType    uint8
	BInterfaceNumber   uint8
	BAlternateSetting  uint8
	BNumEndpoints      uint8
	BInterfaceClass    uint8
	BInterfaceSubClass uint8
	BInterfaceProtocol uint8
	IInterface         uint8
	Endpoint           []*Endpoint_Descriptor
	Extra              []byte
}

func (x *C.struct_libusb_interface_descriptor) c2go() *Interface_Descriptor {
	var list []C.struct_libusb_endpoint_descriptor
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&list))
	hdr.Cap = int(x.bNumEndpoints)
	hdr.Len = int(x.bNumEndpoints)
	hdr.Data = uintptr(unsafe.Pointer(x.endpoint))
	endpoints := make([]*Endpoint_Descriptor, x.bNumEndpoints)
	for i, _ := range endpoints {
		endpoints[i] = (&list[i]).c2go()
	}
	return &Interface_Descriptor{
		ptr:                x,
		BLength:            uint8(x.bLength),
		BDescriptorType:    uint8(x.bDescriptorType),
		BInterfaceNumber:   uint8(x.bInterfaceNumber),
		BAlternateSetting:  uint8(x.bAlternateSetting),
		BNumEndpoints:      uint8(x.bNumEndpoints),
		BInterfaceClass:    uint8(x.bInterfaceClass),
		BInterfaceSubClass: uint8(x.bInterfaceSubClass),
		BInterfaceProtocol: uint8(x.bInterfaceProtocol),
		IInterface:         uint8(x.iInterface),
		Endpoint:           endpoints,
		Extra:              C.GoBytes(unsafe.Pointer(x.extra), x.extra_length),
	}
}

// return a string for an Interface_Descriptor
func (x *Interface_Descriptor) String() string {
	s := make([]string, 0, 16)
	s = append(s, fmt.Sprintf("bLength %d", x.BLength))
	s = append(s, fmt.Sprintf("bDescriptorType %d", x.BDescriptorType))
	s = append(s, fmt.Sprintf("bInterfaceNumber %d", x.BInterfaceNumber))
	s = append(s, fmt.Sprintf("bAlternateSetting %d", x.BAlternateSetting))
	s = append(s, fmt.Sprintf("bNumEndpoints %d", x.BNumEndpoints))
	s = append(s, fmt.Sprintf("bInterfaceClass %d", x.BInterfaceClass))
	s = append(s, fmt.Sprintf("bInterfaceSubClass %d", x.BInterfaceSubClass))
	s = append(s, fmt.Sprintf("bInterfaceProtocol %d", x.BInterfaceProtocol))
	s = append(s, fmt.Sprintf("iInterface %d", x.IInterface))
	for i, v := range x.Endpoint {
		s = append(s, fmt.Sprintf("Endpoint %d:", i))
		s = append(s, indent(v.String()))
	}
	s = append(s, fmt.Sprintf("extra %s", Extra_str(x.Extra)))
	return strings.Join(s, "\n")
}

//-----------------------------------------------------------------------------

// A collection of alternate settings for a particular USB interface.
type Interface struct {
	ptr            *C.struct_libusb_interface
	Num_altsetting int
	Altsetting     []*Interface_Descriptor
}

func (x *C.struct_libusb_interface) c2go() *Interface {
	var list []C.struct_libusb_interface_descriptor
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&list))
	hdr.Cap = int(x.num_altsetting)
	hdr.Len = int(x.num_altsetting)
	hdr.Data = uintptr(unsafe.Pointer(x.altsetting))
	altsetting := make([]*Interface_Descriptor, x.num_altsetting)
	for i, _ := range altsetting {
		altsetting[i] = (&list[i]).c2go()
	}
	return &Interface{
		ptr:            x,
		Num_altsetting: int(x.num_altsetting),
		Altsetting:     altsetting,
	}
}

// return a string for an Interface
func Interface_str(x *Interface) string {
	s := make([]string, 0, 1)
	s = append(s, fmt.Sprintf("num_altsetting %d", x.Num_altsetting))
	for i, v := range x.Altsetting {
		s = append(s, fmt.Sprintf("Interface Descriptor %d:", i))
		s = append(s, indent(v.String()))
	}
	return strings.Join(s, "\n")
}

//-----------------------------------------------------------------------------

// A structure representing the standard USB configuration descriptor.
// This descriptor is documented in section 9.6.3 of the USB 3.0 specification.
// All multiple-byte fields are represented in host-endian format.
type Config_Descriptor struct {
	ptr                 *C.struct_libusb_config_descriptor
	BLength             uint8
	BDescriptorType     uint8
	WTotalLength        uint16
	BNumInterfaces      uint8
	BConfigurationValue uint8
	IConfiguration      uint8
	BmAttributes        uint8
	MaxPower            uint8
	Interface           []*Interface
	Extra               []byte
}

func (x *C.struct_libusb_config_descriptor) c2go() *Config_Descriptor {
	var list []C.struct_libusb_interface
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&list))
	hdr.Cap = int(x.bNumInterfaces)
	hdr.Len = int(x.bNumInterfaces)
	hdr.Data = uintptr(unsafe.Pointer(x._interface))
	interfaces := make([]*Interface, x.bNumInterfaces)
	for i, _ := range interfaces {
		interfaces[i] = (&list[i]).c2go()
	}
	return &Config_Descriptor{
		ptr:                 x,
		BLength:             uint8(x.bLength),
		BDescriptorType:     uint8(x.bDescriptorType),
		WTotalLength:        uint16(x.wTotalLength),
		BNumInterfaces:      uint8(x.bNumInterfaces),
		BConfigurationValue: uint8(x.bConfigurationValue),
		IConfiguration:      uint8(x.iConfiguration),
		BmAttributes:        uint8(x.bmAttributes),
		MaxPower:            uint8(x.MaxPower),
		Interface:           interfaces,
		Extra:               C.GoBytes(unsafe.Pointer(x.extra), x.extra_length),
	}
}

// return a string for a Config_Descriptor
func (x *Config_Descriptor) String() string {
	s := make([]string, 0, 16)
	s = append(s, fmt.Sprintf("bLength %d", x.BLength))
	s = append(s, fmt.Sprintf("bDescriptorType %d", x.BDescriptorType))
	s = append(s, fmt.Sprintf("wTotalLength %d", x.WTotalLength))
	s = append(s, fmt.Sprintf("bNumInterfaces %d", x.BNumInterfaces))
	s = append(s, fmt.Sprintf("bConfigurationValue %d", x.BConfigurationValue))
	s = append(s, fmt.Sprintf("iConfiguration %d", x.IConfiguration))
	s = append(s, fmt.Sprintf("bmAttributes %d", x.BmAttributes))
	s = append(s, fmt.Sprintf("MaxPower %d", x.MaxPower))
	for i, v := range x.Interface {
		s = append(s, fmt.Sprintf("Interface %d:", i))
		s = append(s, indent(fmt.Sprintf(Interface_str(v))))
	}
	s = append(s, fmt.Sprintf("extra %s", Extra_str(x.Extra)))
	return strings.Join(s, "\n")
}

//-----------------------------------------------------------------------------

// A structure representing the superspeed endpoint companion descriptor.
// This descriptor is documented in section 9.6.7 of the USB 3.0 specification.
// All multiple-byte fields are represented in host-endian format.
type SS_Endpoint_Companion_Descriptor struct {
	ptr               *C.struct_libusb_ss_endpoint_companion_descriptor
	BLength           uint8
	BDescriptorType   uint8
	BMaxBurst         uint8
	BmAttributes      uint8
	WBytesPerInterval uint16
}

func (x *C.struct_libusb_ss_endpoint_companion_descriptor) c2go() *SS_Endpoint_Companion_Descriptor {
	return &SS_Endpoint_Companion_Descriptor{
		ptr:               x,
		BLength:           uint8(x.bLength),
		BDescriptorType:   uint8(x.bDescriptorType),
		BMaxBurst:         uint8(x.bMaxBurst),
		BmAttributes:      uint8(x.bmAttributes),
		WBytesPerInterval: uint16(x.wBytesPerInterval),
	}
}

//-----------------------------------------------------------------------------

// A generic representation of a BOS Device Capability descriptor.
// It is advised to check BDevCapabilityType and call the matching
// Get_*_Descriptor function to get a structure fully matching the type.
type BOS_Dev_Capability_Descriptor struct {
	ptr                 *C.struct_libusb_bos_dev_capability_descriptor
	BLength             uint8
	BDescriptorType     uint8
	BDevCapabilityType  uint8
	Dev_capability_data []byte
}

func (x *C.struct_libusb_bos_dev_capability_descriptor) c2go() *BOS_Dev_Capability_Descriptor {
	return &BOS_Dev_Capability_Descriptor{
		ptr:                 x,
		BLength:             uint8(x.bLength),
		BDescriptorType:     uint8(x.bDescriptorType),
		BDevCapabilityType:  uint8(x.bDevCapabilityType),
		Dev_capability_data: C.GoBytes(unsafe.Pointer(C.dev_capability_data_ptr(x)), C.int(x.bLength-3)),
	}
}

//-----------------------------------------------------------------------------

// A structure representing the Binary Device Object Store (BOS) descriptor.
// This descriptor is documented in section 9.6.2 of the USB 3.0 specification.
// All multiple-byte fields are represented in host-endian format.
type BOS_Descriptor struct {
	ptr             *C.struct_libusb_bos_descriptor
	BLength         uint8
	BDescriptorType uint8
	WTotalLength    uint16
	Dev_capability  []*BOS_Dev_Capability_Descriptor
}

func (x *C.struct_libusb_bos_descriptor) c2go() *BOS_Descriptor {
	var list []*C.struct_libusb_bos_dev_capability_descriptor
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&list))
	hdr.Cap = int(x.bNumDeviceCaps)
	hdr.Len = int(x.bNumDeviceCaps)
	hdr.Data = uintptr(unsafe.Pointer(C.dev_capability_ptr(x)))
	dev_capability := make([]*BOS_Dev_Capability_Descriptor, x.bNumDeviceCaps)
	for i, _ := range dev_capability {
		dev_capability[i] = list[i].c2go()
	}
	return &BOS_Descriptor{
		ptr:             x,
		BLength:         uint8(x.bLength),
		BDescriptorType: uint8(x.bDescriptorType),
		WTotalLength:    uint16(x.wTotalLength),
		Dev_capability:  dev_capability,
	}
}

//-----------------------------------------------------------------------------

// A structure representing the USB 2.0 Extension descriptor
// This descriptor is documented in section 9.6.2.1 of the USB 3.0 specification.
// All multiple-byte fields are represented in host-endian format.
type USB_2_0_Extension_Descriptor struct {
	ptr                *C.struct_libusb_usb_2_0_extension_descriptor
	BLength            uint8
	BDescriptorType    uint8
	BDevCapabilityType uint8
	BmAttributes       uint32
}

func (x *C.struct_libusb_usb_2_0_extension_descriptor) c2go() *USB_2_0_Extension_Descriptor {
	return &USB_2_0_Extension_Descriptor{
		ptr:                x,
		BLength:            uint8(x.bLength),
		BDescriptorType:    uint8(x.bDescriptorType),
		BDevCapabilityType: uint8(x.bDevCapabilityType),
		BmAttributes:       uint32(x.bmAttributes),
	}
}

//-----------------------------------------------------------------------------

// A structure representing the SuperSpeed USB Device Capability descriptor
// This descriptor is documented in section 9.6.2.2 of the USB 3.0 specification.
// All multiple-byte fields are represented in host-endian format.
type SS_USB_Device_Capability_Descriptor struct {
	ptr                   *C.struct_libusb_ss_usb_device_capability_descriptor
	BLength               uint8
	BDescriptorType       uint8
	BDevCapabilityType    uint8
	BmAttributes          uint8
	WSpeedSupported       uint16
	BFunctionalitySupport uint8
	BU1DevExitLat         uint8
	BU2DevExitLat         uint16
}

func (x *C.struct_libusb_ss_usb_device_capability_descriptor) c2go() *SS_USB_Device_Capability_Descriptor {
	return &SS_USB_Device_Capability_Descriptor{
		ptr:                   x,
		BLength:               uint8(x.bLength),
		BDescriptorType:       uint8(x.bDescriptorType),
		BDevCapabilityType:    uint8(x.bDevCapabilityType),
		BmAttributes:          uint8(x.bmAttributes),
		WSpeedSupported:       uint16(x.wSpeedSupported),
		BFunctionalitySupport: uint8(x.bFunctionalitySupport),
		BU1DevExitLat:         uint8(x.bU1DevExitLat),
		BU2DevExitLat:         uint16(x.bU2DevExitLat),
	}
}

//-----------------------------------------------------------------------------

// A structure representing the Container ID descriptor.
// This descriptor is documented in section 9.6.2.3 of the USB 3.0 specification.
// All multiple-byte fields, except UUIDs, are represented in host-endian format.
type Container_ID_Descriptor struct {
	ptr                *C.struct_libusb_container_id_descriptor
	BLength            uint8
	BDescriptorType    uint8
	BDevCapabilityType uint8
	BReserved          uint8
	ContainerID        []byte
}

func (x *C.struct_libusb_container_id_descriptor) c2go() *Container_ID_Descriptor {
	return &Container_ID_Descriptor{
		ptr:                x,
		BLength:            uint8(x.bLength),
		BDescriptorType:    uint8(x.bDescriptorType),
		BDevCapabilityType: uint8(x.bDevCapabilityType),
		BReserved:          uint8(x.bReserved),
		ContainerID:        C.GoBytes(unsafe.Pointer(&x.ContainerID[0]), 16),
	}
}

//-----------------------------------------------------------------------------

/*
// Setup packet for control transfers.
struct libusb_control_setup {
	uint8_t  bmRequestType;
	uint8_t  bRequest;
	uint16_t wValue;
	uint16_t wIndex;
	uint16_t wLength;
};
*/

//-----------------------------------------------------------------------------

// A structure representing the standard USB device descriptor.
// This descriptor is documented in section 9.6.1 of the USB 3.0 specification.
// All multiple-byte fields are represented in host-endian format.
type Device_Descriptor struct {
	ptr                *C.struct_libusb_device_descriptor
	BLength            uint8
	BDescriptorType    uint8
	BcdUSB             uint16
	BDeviceClass       uint8
	BDeviceSubClass    uint8
	BDeviceProtocol    uint8
	BMaxPacketSize0    uint8
	IdVendor           uint16
	IdProduct          uint16
	BcdDevice          uint16
	IManufacturer      uint8
	IProduct           uint8
	ISerialNumber      uint8
	BNumConfigurations uint8
}

func (x *C.struct_libusb_device_descriptor) c2go() *Device_Descriptor {
	return &Device_Descriptor{
		ptr:                x,
		BLength:            uint8(x.bLength),
		BDescriptorType:    uint8(x.bDescriptorType),
		BcdUSB:             uint16(x.bcdUSB),
		BDeviceClass:       uint8(x.bDeviceClass),
		BDeviceSubClass:    uint8(x.bDeviceSubClass),
		BDeviceProtocol:    uint8(x.bDeviceProtocol),
		BMaxPacketSize0:    uint8(x.bMaxPacketSize0),
		IdVendor:           uint16(x.idVendor),
		IdProduct:          uint16(x.idProduct),
		BcdDevice:          uint16(x.bcdDevice),
		IManufacturer:      uint8(x.iManufacturer),
		IProduct:           uint8(x.iProduct),
		ISerialNumber:      uint8(x.iSerialNumber),
		BNumConfigurations: uint8(x.bNumConfigurations),
	}
}

// return a string for a Device_Descriptor
func (x *Device_Descriptor) String() string {
	s := make([]string, 0, 16)
	s = append(s, fmt.Sprintf("bLength %d", x.BLength))
	s = append(s, fmt.Sprintf("bDescriptorType %d", x.BDescriptorType))
	s = append(s, fmt.Sprintf("bcdUSB %s", bcd2str(x.BcdUSB)))
	s = append(s, fmt.Sprintf("bDeviceClass %d", x.BDeviceClass))
	s = append(s, fmt.Sprintf("bDeviceSubClass %d", x.BDeviceSubClass))
	s = append(s, fmt.Sprintf("bDeviceProtocol %d", x.BDeviceProtocol))
	s = append(s, fmt.Sprintf("bMaxPacketSize0 %d", x.BMaxPacketSize0))
	s = append(s, fmt.Sprintf("idVendor 0x%04x", x.IdVendor))
	s = append(s, fmt.Sprintf("idProduct 0x%04x", x.IdProduct))
	s = append(s, fmt.Sprintf("bcdDevice %s", bcd2str(x.BcdDevice)))
	s = append(s, fmt.Sprintf("iManufacturer %d", x.IManufacturer))
	s = append(s, fmt.Sprintf("iProduct %d", x.IProduct))
	s = append(s, fmt.Sprintf("iSerialNumber %d", x.ISerialNumber))
	s = append(s, fmt.Sprintf("bNumConfigurations %d", x.BNumConfigurations))
	return strings.Join(s, "\n")
}

//-----------------------------------------------------------------------------

/*

struct libusb_transfer {
	libusb_device_handle *dev_handle;
	uint8_t flags;
	unsigned char endpoint;
	unsigned char type;
	unsigned int timeout;
	enum libusb_transfer_status status;
	int length;
	int actual_length;
	libusb_transfer_cb_fn callback;
	void *user_data;
	unsigned char *buffer;
	int num_iso_packets;
	struct libusb_iso_packet_descriptor iso_packet_desc[];
};

*/

// The generic USB transfer structure. The user populates this structure and
// then submits it in order to request a transfer. After the transfer has
// completed, the library populates the transfer with the results and passes
// it back to the user.
type Transfer struct {
	ptr *C.struct_libusb_transfer
}

func (x *C.struct_libusb_transfer) c2go() *Transfer {
	return &Transfer{
		ptr: x,
	}
}

func (x *Transfer) go2c() *C.struct_libusb_transfer {
	return x.ptr
}

// return a string for a Device_Descriptor
func (x *Transfer) String() string {
	s := make([]string, 0, 1)
	return strings.Join(s, "\n")
}

//-----------------------------------------------------------------------------

// Structure providing the version of the libusb runtime.
type Version struct {
	ptr      *C.struct_libusb_version
	Major    uint16
	Minor    uint16
	Micro    uint16
	Nano     uint16
	Rc       string
	Describe string
}

func (x *C.struct_libusb_version) c2go() *Version {
	return &Version{
		ptr:      x,
		Major:    uint16(x.major),
		Minor:    uint16(x.minor),
		Micro:    uint16(x.micro),
		Nano:     uint16(x.nano),
		Rc:       C.GoString(x.rc),
		Describe: C.GoString(x.describe),
	}
}

//-----------------------------------------------------------------------------

// Structure representing a libusb session.
type Context *C.struct_libusb_context

// Structure representing a USB device detected on the system.
type Device *C.struct_libusb_device

// Structure representing a handle on a USB device.
type Device_Handle *C.struct_libusb_device_handle

//type Hotplug_Callback *C.struct_libusb_hotplug_callback

//-----------------------------------------------------------------------------
// errors

type libusb_error struct {
	Code int
}

func (e *libusb_error) Error() string {
	return Error_Name(e.Code)
}

//-----------------------------------------------------------------------------
// Library initialization/deinitialization

func Set_Debug(ctx Context, level int) {
	C.libusb_set_debug(ctx, C.int(level))
}

func Init(ctx *Context) error {
	rc := int(C.libusb_init((**C.struct_libusb_context)(ctx)))
	if rc != 0 {
		return &libusb_error{rc}
	}
	return nil
}

func Exit(ctx Context) {
	C.libusb_exit(ctx)
}

//-----------------------------------------------------------------------------
// Device handling and enumeration

func Get_Device_List(ctx Context) ([]Device, error) {
	var hdl **C.struct_libusb_device
	rc := int(C.libusb_get_device_list(ctx, (***C.struct_libusb_device)(&hdl)))
	if rc < 0 {
		return nil, &libusb_error{rc}
	}
	// turn the c array into a slice of device pointers
	var list []Device
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&list))
	hdr.Cap = rc
	hdr.Len = rc
	hdr.Data = uintptr(unsafe.Pointer(hdl))
	return list, nil
}

func Free_Device_List(list []Device, unref_devices int) {
	if list == nil {
		return
	}
	if len(list) == 0 {
		return
	}
	C.libusb_free_device_list((**C.struct_libusb_device)(&list[0]), C.int(unref_devices))
}

func Get_Bus_Number(dev Device) uint8 {
	return uint8(C.libusb_get_bus_number(dev))
}

func Get_Port_Number(dev Device) uint8 {
	return uint8(C.libusb_get_port_number(dev))
}

func Get_Port_Numbers(dev Device, ports []byte) ([]byte, error) {
	rc := int(C.libusb_get_port_numbers(dev, (*C.uint8_t)(&ports[0]), (C.int)(len(ports))))
	if rc < 0 {
		return nil, &libusb_error{rc}
	}
	return ports[:rc], nil
}

/*
func Get_Parent(dev Device) Device {
	return C.libusb_get_parent(dev)
}
*/

func Get_Device_Address(dev Device) uint8 {
	return uint8(C.libusb_get_device_address(dev))
}

func Get_Device_Speed(dev Device) int {
	return int(C.libusb_get_device_speed(dev))
}

func Get_Max_Packet_Size(dev Device, endpoint uint8) int {
	return int(C.libusb_get_max_packet_size(dev, (C.uchar)(endpoint)))
}

func Get_Max_ISO_Packet_Size(dev Device, endpoint uint8) int {
	return int(C.libusb_get_max_iso_packet_size(dev, (C.uchar)(endpoint)))
}

func Ref_Device(dev Device) Device {
	return C.libusb_ref_device(dev)
}

func Unref_Device(dev Device) {
	C.libusb_unref_device(dev)
}

func Open(dev Device) (Device_Handle, error) {
	var hdl Device_Handle
	rc := int(C.libusb_open(dev, (**C.struct_libusb_device_handle)(&hdl)))
	if rc < 0 {
		return nil, &libusb_error{rc}
	}
	return hdl, nil
}

func Open_Device_With_VID_PID(ctx Context, vendor_id uint16, product_id uint16) Device_Handle {
	return C.libusb_open_device_with_vid_pid(ctx, (C.uint16_t)(vendor_id), (C.uint16_t)(product_id))
}

func Close(hdl Device_Handle) {
	C.libusb_close(hdl)
}

func Get_Device(hdl Device_Handle) Device {
	return C.libusb_get_device(hdl)
}

func Get_Configuration(hdl Device_Handle) (int, error) {
	var config C.int
	rc := int(C.libusb_get_configuration(hdl, &config))
	if rc < 0 {
		return 0, &libusb_error{rc}
	}
	return int(config), nil
}

func Set_Configuration(hdl Device_Handle, configuration int) error {
	rc := int(C.libusb_set_configuration(hdl, (C.int)(configuration)))
	if rc < 0 {
		return &libusb_error{rc}
	}
	return nil
}

func Claim_Interface(hdl Device_Handle, interface_number int) error {
	rc := int(C.libusb_claim_interface(hdl, (C.int)(interface_number)))
	if rc < 0 {
		return &libusb_error{rc}
	}
	return nil
}

func Release_Interface(hdl Device_Handle, interface_number int) error {
	rc := int(C.libusb_release_interface(hdl, (C.int)(interface_number)))
	if rc < 0 {
		return &libusb_error{rc}
	}
	return nil
}

func Set_Interface_Alt_Setting(hdl Device_Handle, interface_number int, alternate_setting int) error {
	rc := int(C.libusb_set_interface_alt_setting(hdl, (C.int)(interface_number), (C.int)(alternate_setting)))
	if rc < 0 {
		return &libusb_error{rc}
	}
	return nil
}

func Clear_Halt(hdl Device_Handle, endpoint uint8) error {
	rc := int(C.libusb_clear_halt(hdl, (C.uchar)(endpoint)))
	if rc < 0 {
		return &libusb_error{rc}
	}
	return nil
}

func Reset_Device(hdl Device_Handle) error {
	rc := int(C.libusb_reset_device(hdl))
	if rc < 0 {
		return &libusb_error{rc}
	}
	return nil
}

func Kernel_Driver_Active(hdl Device_Handle, interface_number int) (bool, error) {
	rc := int(C.libusb_kernel_driver_active(hdl, (C.int)(interface_number)))
	if rc < 0 {
		return false, &libusb_error{rc}
	}
	return rc != 0, nil
}

func Detach_Kernel_Driver(hdl Device_Handle, interface_number int) error {
	rc := int(C.libusb_detach_kernel_driver(hdl, (C.int)(interface_number)))
	if rc < 0 {
		return &libusb_error{rc}
	}
	return nil
}

func Attach_Kernel_Driver(hdl Device_Handle, interface_number int) error {
	rc := int(C.libusb_attach_kernel_driver(hdl, (C.int)(interface_number)))
	if rc < 0 {
		return &libusb_error{rc}
	}
	return nil
}

func Set_Auto_Detach_Kernel_Driver(hdl Device_Handle, enable bool) error {
	enable_int := 0
	if enable {
		enable_int = 1
	}
	rc := int(C.libusb_set_auto_detach_kernel_driver(hdl, (C.int)(enable_int)))
	if rc < 0 {
		return &libusb_error{rc}
	}
	return nil
}

//-----------------------------------------------------------------------------
// Miscellaneous

/*
func Has_Capability(capability uint32) bool {
	rc := int(C.libusb_has_capability((C.uint32_t)(capability)))
	return rc != 0
}
*/

func Error_Name(code int) string {
	return C.GoString(C.libusb_error_name(C.int(code)))
}

func Get_Version() *Version {
	ver := (*C.struct_libusb_version)(unsafe.Pointer(C.libusb_get_version()))
	return ver.c2go()
}

func CPU_To_LE16(x uint16) uint16 {
	return uint16(C.libusb_cpu_to_le16((C.uint16_t)(x)))
}

/*
func Setlocale(locale string) error {
	cstr := C.CString(locale)
	rc := int(C.libusb_setlocale(cstr))
	if rc < 0 {
		return &libusb_error{rc}
	}
	return nil
}
*/

func Strerror(errcode int) string {
	return C.GoString(C.libusb_strerror(int32(errcode)))
}

//-----------------------------------------------------------------------------
// USB descriptors

func Get_Device_Descriptor(dev Device) (*Device_Descriptor, error) {
	var desc C.struct_libusb_device_descriptor
	rc := int(C.libusb_get_device_descriptor(dev, &desc))
	if rc != 0 {
		return nil, &libusb_error{rc}
	}
	return (&desc).c2go(), nil
}

func Get_Active_Config_Descriptor(dev Device) (*Config_Descriptor, error) {
	var desc *C.struct_libusb_config_descriptor
	rc := int(C.libusb_get_active_config_descriptor(dev, &desc))
	if rc != 0 {
		return nil, &libusb_error{rc}
	}
	return desc.c2go(), nil
}

func Get_Config_Descriptor(dev Device, config_index uint8) (*Config_Descriptor, error) {
	var desc *C.struct_libusb_config_descriptor
	rc := int(C.libusb_get_config_descriptor(dev, (C.uint8_t)(config_index), &desc))
	if rc != 0 {
		return nil, &libusb_error{rc}
	}
	return desc.c2go(), nil
}

func Get_Config_Descriptor_By_Value(dev Device, bConfigurationValue uint8) (*Config_Descriptor, error) {
	var desc *C.struct_libusb_config_descriptor
	rc := int(C.libusb_get_config_descriptor_by_value(dev, (C.uint8_t)(bConfigurationValue), &desc))
	if rc != 0 {
		return nil, &libusb_error{rc}
	}
	return desc.c2go(), nil
}

func Free_Config_Descriptor(config *Config_Descriptor) {
	C.libusb_free_config_descriptor(config.ptr)
}

func Get_SS_Endpoint_Companion_Descriptor(ctx Context, endpoint *Endpoint_Descriptor) (*SS_Endpoint_Companion_Descriptor, error) {
	var desc *C.struct_libusb_ss_endpoint_companion_descriptor
	rc := int(C.libusb_get_ss_endpoint_companion_descriptor(ctx, endpoint.ptr, &desc))
	if rc != 0 {
		return nil, &libusb_error{rc}
	}
	return desc.c2go(), nil
}

func Free_SS_Endpoint_Companion_Descriptor(ep_comp *SS_Endpoint_Companion_Descriptor) {
	C.libusb_free_ss_endpoint_companion_descriptor(ep_comp.ptr)
}

func Get_BOS_Descriptor(hdl Device_Handle) (*BOS_Descriptor, error) {
	var desc *C.struct_libusb_bos_descriptor
	rc := int(C.libusb_get_bos_descriptor(hdl, &desc))
	if rc != 0 {
		return nil, &libusb_error{rc}
	}
	return desc.c2go(), nil
}

func Free_BOS_Descriptor(bos *BOS_Descriptor) {
	C.libusb_free_bos_descriptor(bos.ptr)
}

func Get_USB_2_0_Extension_Descriptor(ctx Context, dev_cap *BOS_Dev_Capability_Descriptor) (*USB_2_0_Extension_Descriptor, error) {
	var desc *C.struct_libusb_usb_2_0_extension_descriptor
	rc := int(C.libusb_get_usb_2_0_extension_descriptor(ctx, dev_cap.ptr, &desc))
	if rc != 0 {
		return nil, &libusb_error{rc}
	}
	return desc.c2go(), nil
}

func Free_USB_2_0_Extension_Descriptor(usb_2_0_extension *USB_2_0_Extension_Descriptor) {
	C.libusb_free_usb_2_0_extension_descriptor(usb_2_0_extension.ptr)
}

func Get_SS_USB_Device_Capability_Descriptor(ctx Context, dev_cap *BOS_Dev_Capability_Descriptor) (*SS_USB_Device_Capability_Descriptor, error) {
	var desc *C.struct_libusb_ss_usb_device_capability_descriptor
	rc := int(C.libusb_get_ss_usb_device_capability_descriptor(ctx, dev_cap.ptr, &desc))
	if rc != 0 {
		return nil, &libusb_error{rc}
	}
	return desc.c2go(), nil
}

func Free_SS_USB_Device_Capability_Descriptor(ss_usb_device_cap *SS_USB_Device_Capability_Descriptor) {
	C.libusb_free_ss_usb_device_capability_descriptor(ss_usb_device_cap.ptr)
}

func Get_Container_ID_Descriptor(ctx Context, dev_cap *BOS_Dev_Capability_Descriptor) (*Container_ID_Descriptor, error) {
	var desc *C.struct_libusb_container_id_descriptor
	rc := int(C.libusb_get_container_id_descriptor(ctx, dev_cap.ptr, &desc))
	if rc != 0 {
		return nil, &libusb_error{rc}
	}
	return desc.c2go(), nil
}

func Free_Container_ID_Descriptor(container_id *Container_ID_Descriptor) {
	C.libusb_free_container_id_descriptor(container_id.ptr)
}

func Get_String_Descriptor_ASCII(hdl Device_Handle, desc_index uint8, data []byte) ([]byte, error) {
	rc := int(C.libusb_get_string_descriptor_ascii(hdl, (C.uint8_t)(desc_index), (*C.uchar)(&data[0]), (C.int)(len(data))))
	if rc < 0 {
		return nil, &libusb_error{rc}
	}
	return data[:rc], nil
}

func Get_Descriptor(hdl Device_Handle, desc_type uint8, desc_index uint8, data []byte) ([]byte, error) {
	rc := int(C.libusb_get_descriptor(hdl, (C.uint8_t)(desc_type), (C.uint8_t)(desc_index), (*C.uchar)(&data[0]), (C.int)(len(data))))
	if rc < 0 {
		return nil, &libusb_error{rc}
	}
	return data[:rc], nil
}

func Get_String_Descriptor(hdl Device_Handle, desc_index uint8, langid uint16, data []byte) ([]byte, error) {
	rc := int(C.libusb_get_string_descriptor(hdl, (C.uint8_t)(desc_index), (C.uint16_t)(langid), (*C.uchar)(&data[0]), (C.int)(len(data))))
	if rc < 0 {
		return nil, &libusb_error{rc}
	}
	return data[:rc], nil
}

//-----------------------------------------------------------------------------
// Device hotplug event notification

//int 	libusb_hotplug_register_callback (libusb_context *ctx, libusb_hotplug_event events, libusb_hotplug_flag flags, int vendor_id, int product_id, int dev_class, libusb_hotplug_callback_fn cb_fn, void *user_data, libusb_hotplug_callback_handle *handle)
//void 	libusb_hotplug_deregister_callback (libusb_context *ctx, libusb_hotplug_callback_handle handle)

//-----------------------------------------------------------------------------
//Asynchronous device I/O

func Alloc_Streams(dev Device_Handle, num_streams uint32, endpoints []byte) (int, error) {
	rc := int(C.libusb_alloc_streams(dev, (C.uint32_t)(num_streams), (*C.uchar)(&endpoints[0]), (C.int)(len(endpoints))))
	if rc < 0 {
		return 0, &libusb_error{rc}
	}
	return rc, nil
}

func Free_Streams(dev Device_Handle, endpoints []byte) error {
	rc := int(C.libusb_free_streams(dev, (*C.uchar)(&endpoints[0]), (C.int)(len(endpoints))))
	if rc != 0 {
		return &libusb_error{rc}
	}
	return nil
}

func Alloc_Transfer(iso_packets int) (*Transfer, error) {
	ptr := C.libusb_alloc_transfer((C.int)(iso_packets))
	if ptr == nil {
		return nil, &libusb_error{ERROR_OTHER}
	}
	return ptr.c2go(), nil
}

func Free_Transfer(transfer *Transfer) {
	C.libusb_free_transfer(transfer.ptr)
}

func Submit_Transfer(transfer *Transfer) error {
	rc := int(C.libusb_submit_transfer(transfer.go2c()))
	if rc != 0 {
		return &libusb_error{rc}
	}
	return nil
}

func Cancel_Transfer(transfer *Transfer) error {
	rc := int(C.libusb_cancel_transfer(transfer.go2c()))
	if rc != 0 {
		return &libusb_error{rc}
	}
	return nil
}

func Transfer_Set_Stream_ID(transfer *Transfer, stream_id uint32) {
	C.libusb_transfer_set_stream_id(transfer.go2c(), (C.uint32_t)(stream_id))
}

func Transfer_Get_Stream_ID(transfer *Transfer) uint32 {
	return uint32(C.libusb_transfer_get_stream_id(transfer.go2c()))
}

func Control_Transfer_Get_Data(transfer *Transfer) *byte {
	// should this return a slice? - what's the length?
	return (*byte)(C.libusb_control_transfer_get_data(transfer.go2c()))
}

// static struct libusb_control_setup * 	libusb_control_transfer_get_setup (struct libusb_transfer *transfer)
// static void 	libusb_fill_control_setup (unsigned char *buffer, uint8_t bmRequestType, uint8_t bRequest, uint16_t wValue, uint16_t wIndex, uint16_t wLength)
// static void 	libusb_fill_control_transfer (struct libusb_transfer *transfer, libusb_device_handle *dev_handle, unsigned char *buffer, libusb_transfer_cb_fn callback, void *user_data, unsigned int timeout)
// static void 	libusb_fill_bulk_transfer (struct libusb_transfer *transfer, libusb_device_handle *dev_handle, unsigned char endpoint, unsigned char *buffer, int length, libusb_transfer_cb_fn callback, void *user_data, unsigned int timeout)
// static void 	libusb_fill_bulk_stream_transfer (struct libusb_transfer *transfer, libusb_device_handle *dev_handle, unsigned char endpoint, uint32_t stream_id, unsigned char *buffer, int length, libusb_transfer_cb_fn callback, void *user_data, unsigned int timeout)
// static void 	libusb_fill_interrupt_transfer (struct libusb_transfer *transfer, libusb_device_handle *dev_handle, unsigned char endpoint, unsigned char *buffer, int length, libusb_transfer_cb_fn callback, void *user_data, unsigned int timeout)
// static void 	libusb_fill_iso_transfer (struct libusb_transfer *transfer, libusb_device_handle *dev_handle, unsigned char endpoint, unsigned char *buffer, int length, int num_iso_packets, libusb_transfer_cb_fn callback, void *user_data, unsigned int timeout)
// static void 	libusb_set_iso_packet_lengths (struct libusb_transfer *transfer, unsigned int length)
// static unsigned char * 	libusb_get_iso_packet_buffer (struct libusb_transfer *transfer, unsigned int packet)
// static unsigned char * 	libusb_get_iso_packet_buffer_simple (struct libusb_transfer *transfer, unsigned int packet)

//-----------------------------------------------------------------------------
// Polling and timing

// int 	libusb_try_lock_events (libusb_context *ctx)
// void 	libusb_lock_events (libusb_context *ctx)
// void 	libusb_unlock_events (libusb_context *ctx)
// int 	libusb_event_handling_ok (libusb_context *ctx)
// int 	libusb_event_handler_active (libusb_context *ctx)
// void 	libusb_lock_event_waiters (libusb_context *ctx)
// void 	libusb_unlock_event_waiters (libusb_context *ctx)
// int 	libusb_wait_for_event (libusb_context *ctx, struct timeval *tv)
// int 	libusb_handle_events_timeout_completed (libusb_context *ctx, struct timeval *tv, int *completed)
// int 	libusb_handle_events_timeout (libusb_context *ctx, struct timeval *tv)
// int 	libusb_handle_events (libusb_context *ctx)
// int 	libusb_handle_events_completed (libusb_context *ctx, int *completed)
// int 	libusb_handle_events_locked (libusb_context *ctx, struct timeval *tv)
// int 	libusb_pollfds_handle_timeouts (libusb_context *ctx)
// int 	libusb_get_next_timeout (libusb_context *ctx, struct timeval *tv)
// void 	libusb_set_pollfd_notifiers (libusb_context *ctx, libusb_pollfd_added_cb added_cb, libusb_pollfd_removed_cb removed_cb, void *user_data)
// const struct libusb_pollfd ** 	libusb_get_pollfds (libusb_context *ctx)
// void 	libusb_free_pollfds (const struct libusb_pollfd **pollfds)

//-----------------------------------------------------------------------------
// Synchronous device I/O

func Control_Transfer(hdl Device_Handle, bmRequestType uint8, bRequest uint8, wValue uint16, wIndex uint16, data []byte, timeout uint) ([]byte, error) {
	rc := int(C.libusb_control_transfer(hdl, (C.uint8_t)(bmRequestType), (C.uint8_t)(bRequest), (C.uint16_t)(wValue), (C.uint16_t)(wIndex),
		(*C.uchar)(&data[0]), (C.uint16_t)(len(data)), (C.uint)(timeout)))
	if rc < 0 {
		return nil, &libusb_error{rc}
	}
	return data[:rc], nil
}

func Bulk_Transfer(hdl Device_Handle, endpoint uint8, data []byte, timeout uint) ([]byte, error) {
	var transferred C.int
	rc := int(C.libusb_bulk_transfer(hdl, (C.uchar)(endpoint), (*C.uchar)(&data[0]), (C.int)(len(data)), &transferred, (C.uint)(timeout)))
	if rc != 0 {
		return nil, &libusb_error{rc}
	}
	return data[:int(transferred)], nil
}

func Interrupt_Transfer(hdl Device_Handle, endpoint uint8, data []byte, timeout uint) ([]byte, error) {
	var transferred C.int
	rc := int(C.libusb_interrupt_transfer(hdl, (C.uchar)(endpoint), (*C.uchar)(&data[0]), (C.int)(len(data)), &transferred, (C.uint)(timeout)))
	if rc != 0 {
		return nil, &libusb_error{rc}
	}
	return data[:int(transferred)], nil
}

// libusb_cancel_sync_transfers_on_device(struct libusb_device_handle *dev_handle) {
func Cancel_Sync_Transfers_On_Device(hdl Device_Handle) {
	C.libusb_cancel_sync_transfers_on_device(hdl)
}

//-----------------------------------------------------------------------------
