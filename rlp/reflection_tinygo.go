// Copyright 2025 The go-ethereum Authors
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

//go:build tinygo

package rlp

import "reflect"

var customEncodedTypes = make(map[reflect.Type]bool)

func implementsInterface(t reflect.Type, i reflect.Type) bool {
	// reflect implementation of tinygo cannot handle this automatically
	// at runtime. So we need custom encoded types to be registered manually.
	if i == decoderInterface || i == encoderInterface {
		return customEncodedTypes[t]
	}
	return false
}

// RegisterCustomEncodedType manually registers a type as an implementation of
// Decoder and Encoder interfaces
func RegisterCustomEncodedType(t reflect.Type) {
	customEncodedTypes[t] = true
}
