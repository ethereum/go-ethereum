// Copyright 2024 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package gmp

// #cgo LDFLAGS: -lgmp
// #include "modexp.h"
import "C"
import (
	"errors"
	"runtime"
	"unsafe"
)

// stripLeadingZeros removes leading zero bytes from a slice
// Note: This has no effect for big-endian integers
func stripLeadingZeros(data []byte) []byte {
	for i, b := range data {
		if b != 0 {
			return data[i:]
		}
	}
	// All zeros, return empty slice
	return []byte{}
}

// ModExp performs modular exponentiation using GMP
// This is thread safe.
func ModExp(base, exp, mod []byte) ([]byte, error) {
	// Handle empty modulus - return empty result (EVM behavior)
	if len(mod) == 0 {
		return []byte{}, nil
	}

	// Special case: zero modulus
	allZero := true
	for _, b := range mod {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		return []byte{}, nil
	}

	// Special case: base == 1
	// If base is 1, the result is always 1 (when mod > 1)
	if len(base) == 1 && base[0] == 1 {
		// For modulus > 1, 1^exp mod modulus = 1
		// For modulus == 1, any number mod 1 = 0
		if len(mod) == 1 && mod[0] == 1 {
			return []byte{}, nil
		}
		return []byte{1}, nil
	}

	// Strip leading zeros for GMP performance
	base = stripLeadingZeros(base)
	exp = stripLeadingZeros(exp)
	modStripped := stripLeadingZeros(mod)

	// Allocate result buffer (size of stripped modulus is the max possible result)
	// Note: We know that the modulus stripped is non-zero because
	// we check for all zeroes.
	result := make([]byte, len(modStripped))
	resultLen := C.size_t(len(result))

	// Handle empty slices - pass a dummy non-nil pointer with length 0
	// This avoids UB when the length is zero.
	dummy := C.uint8_t(0)
	var basePtr, expPtr, modPtr *C.uint8_t = &dummy, &dummy, &dummy

	if len(base) > 0 {
		basePtr = (*C.uint8_t)(unsafe.Pointer(&base[0]))
	}
	if len(exp) > 0 {
		expPtr = (*C.uint8_t)(unsafe.Pointer(&exp[0]))
	}
	if len(modStripped) > 0 {
		modPtr = (*C.uint8_t)(unsafe.Pointer(&modStripped[0]))
	}

	// Call C function
	ret := C.modexp_bytes(
		basePtr, C.size_t(len(base)),
		expPtr, C.size_t(len(exp)),
		modPtr, C.size_t(len(modStripped)),
		(*C.uint8_t)(unsafe.Pointer(&result[0])), &resultLen,
	)

	// Keep the slices alive until after the C call completes
	runtime.KeepAlive(base)
	runtime.KeepAlive(exp)
	runtime.KeepAlive(modStripped)
	runtime.KeepAlive(result)

	// Check for errors
	switch ret {
	case 0:
		// Success - trim result to actual size
		if resultLen == 0 {
			return []byte{}, nil
		}
		return result[:resultLen], nil
	case -1:
		return nil, errors.New("invalid parameter")
	case -2:
		return nil, errors.New("result buffer too small")
	default:
		return nil, errors.New("unknown error")
	}
}
