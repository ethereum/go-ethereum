// Copyright 2013 Google Inc.  All rights reserved.
// Copyright 2016 the gousb Authors.  All rights reserved.
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

// #include "./libusb/libusb/libusb.h"
import "C"

// rawError is an error code from libusb.
type rawError C.int

// Error implements the error interface.
func (e rawError) Error() string {
	return fmt.Sprintf("libusb: %s [code %d]", rawErrorString[e], e)
}

// fromRawErrno converts a raw libusb error into a Go type.
func fromRawErrno(errno C.int) error {
	err := rawError(errno)
	if err == errSuccess {
		return nil
	}
	return err
}

const (
	errSuccess      rawError = C.LIBUSB_SUCCESS
	errIO           rawError = C.LIBUSB_ERROR_IO
	errInvalidParam rawError = C.LIBUSB_ERROR_INVALID_PARAM
	errAccess       rawError = C.LIBUSB_ERROR_ACCESS
	errNoDevice     rawError = C.LIBUSB_ERROR_NO_DEVICE
	errNotFound     rawError = C.LIBUSB_ERROR_NOT_FOUND
	errBusy         rawError = C.LIBUSB_ERROR_BUSY
	errTimeout      rawError = C.LIBUSB_ERROR_TIMEOUT
	errOverflow     rawError = C.LIBUSB_ERROR_OVERFLOW
	errPipe         rawError = C.LIBUSB_ERROR_PIPE
	errInterrupted  rawError = C.LIBUSB_ERROR_INTERRUPTED
	errNoMem        rawError = C.LIBUSB_ERROR_NO_MEM
	errNotSupported rawError = C.LIBUSB_ERROR_NOT_SUPPORTED
	errOther        rawError = C.LIBUSB_ERROR_OTHER
)

var rawErrorString = map[rawError]string{
	errSuccess:      "success",
	errIO:           "i/o error",
	errInvalidParam: "invalid param",
	errAccess:       "bad access",
	errNoDevice:     "no device",
	errNotFound:     "not found",
	errBusy:         "device or resource busy",
	errTimeout:      "timeout",
	errOverflow:     "overflow",
	errPipe:         "pipe error",
	errInterrupted:  "interrupted",
	errNoMem:        "out of memory",
	errNotSupported: "not supported",
	errOther:        "unknown error",
}
