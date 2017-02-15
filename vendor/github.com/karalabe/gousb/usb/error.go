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

import (
	"fmt"
)

// #include "libusb.h"
import "C"

type usbError C.int

func (e usbError) Error() string {
	return fmt.Sprintf("libusb: %s [code %d]", usbErrorString[e], int(e))
}

const (
	SUCCESS             usbError = C.LIBUSB_SUCCESS
	ERROR_IO            usbError = C.LIBUSB_ERROR_IO
	ERROR_INVALID_PARAM usbError = C.LIBUSB_ERROR_INVALID_PARAM
	ERROR_ACCESS        usbError = C.LIBUSB_ERROR_ACCESS
	ERROR_NO_DEVICE     usbError = C.LIBUSB_ERROR_NO_DEVICE
	ERROR_NOT_FOUND     usbError = C.LIBUSB_ERROR_NOT_FOUND
	ERROR_BUSY          usbError = C.LIBUSB_ERROR_BUSY
	ERROR_TIMEOUT       usbError = C.LIBUSB_ERROR_TIMEOUT
	ERROR_OVERFLOW      usbError = C.LIBUSB_ERROR_OVERFLOW
	ERROR_PIPE          usbError = C.LIBUSB_ERROR_PIPE
	ERROR_INTERRUPTED   usbError = C.LIBUSB_ERROR_INTERRUPTED
	ERROR_NO_MEM        usbError = C.LIBUSB_ERROR_NO_MEM
	ERROR_NOT_SUPPORTED usbError = C.LIBUSB_ERROR_NOT_SUPPORTED
	ERROR_OTHER         usbError = C.LIBUSB_ERROR_OTHER
)

var usbErrorString = map[usbError]string{
	C.LIBUSB_SUCCESS:             "success",
	C.LIBUSB_ERROR_IO:            "i/o error",
	C.LIBUSB_ERROR_INVALID_PARAM: "invalid param",
	C.LIBUSB_ERROR_ACCESS:        "bad access",
	C.LIBUSB_ERROR_NO_DEVICE:     "no device",
	C.LIBUSB_ERROR_NOT_FOUND:     "not found",
	C.LIBUSB_ERROR_BUSY:          "device or resource busy",
	C.LIBUSB_ERROR_TIMEOUT:       "timeout",
	C.LIBUSB_ERROR_OVERFLOW:      "overflow",
	C.LIBUSB_ERROR_PIPE:          "pipe error",
	C.LIBUSB_ERROR_INTERRUPTED:   "interrupted",
	C.LIBUSB_ERROR_NO_MEM:        "out of memory",
	C.LIBUSB_ERROR_NOT_SUPPORTED: "not supported",
	C.LIBUSB_ERROR_OTHER:         "unknown error",
}

type TransferStatus uint8

const (
	LIBUSB_TRANSFER_COMPLETED TransferStatus = C.LIBUSB_TRANSFER_COMPLETED
	LIBUSB_TRANSFER_ERROR     TransferStatus = C.LIBUSB_TRANSFER_ERROR
	LIBUSB_TRANSFER_TIMED_OUT TransferStatus = C.LIBUSB_TRANSFER_TIMED_OUT
	LIBUSB_TRANSFER_CANCELLED TransferStatus = C.LIBUSB_TRANSFER_CANCELLED
	LIBUSB_TRANSFER_STALL     TransferStatus = C.LIBUSB_TRANSFER_STALL
	LIBUSB_TRANSFER_NO_DEVICE TransferStatus = C.LIBUSB_TRANSFER_NO_DEVICE
	LIBUSB_TRANSFER_OVERFLOW  TransferStatus = C.LIBUSB_TRANSFER_OVERFLOW
)

var transferStatusDescription = map[TransferStatus]string{
	LIBUSB_TRANSFER_COMPLETED: "transfer completed without error",
	LIBUSB_TRANSFER_ERROR:     "transfer failed",
	LIBUSB_TRANSFER_TIMED_OUT: "transfer timed out",
	LIBUSB_TRANSFER_CANCELLED: "transfer was cancelled",
	LIBUSB_TRANSFER_STALL:     "halt condition detected (endpoint stalled) or control request not supported",
	LIBUSB_TRANSFER_NO_DEVICE: "device was disconnected",
	LIBUSB_TRANSFER_OVERFLOW:  "device sent more data than requested",
}

func (ts TransferStatus) String() string {
	return transferStatusDescription[ts]
}

func (ts TransferStatus) Error() string {
	return "libusb: " + ts.String()
}
