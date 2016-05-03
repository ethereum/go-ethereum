// Copyright 2016 The go-ethereum Authors
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

package abi

import "reflect"

// indirect recursively dereferences the value until it either gets the value
// or finds a big.Int
func indirect(v reflect.Value) reflect.Value {
	if v.Kind() == reflect.Ptr && v.Elem().Type() != big_t {
		return indirect(v.Elem())
	}
	return v
}

// reflectIntKind returns the reflect using the given size and
// unsignedness.
func reflectIntKind(unsigned bool, size int) reflect.Kind {
	switch size {
	case 8:
		if unsigned {
			return reflect.Uint8
		}
		return reflect.Int8
	case 16:
		if unsigned {
			return reflect.Uint16
		}
		return reflect.Int16
	case 32:
		if unsigned {
			return reflect.Uint32
		}
		return reflect.Int32
	case 64:
		if unsigned {
			return reflect.Uint64
		}
		return reflect.Int64
	}
	return reflect.Ptr
}

// mustArrayToBytesSlice creates a new byte slice with the exact same size as value
// and copies the bytes in value to the new slice.
func mustArrayToByteSlice(value reflect.Value) reflect.Value {
	slice := reflect.MakeSlice(reflect.TypeOf([]byte{}), value.Len(), value.Len())
	reflect.Copy(slice, value)
	return slice
}
