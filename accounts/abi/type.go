// Copyright 2015 The go-ethereum Authors
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

import (
	"github.com/ethereum/go-ethereum/abi"
)

// Type enumerator
const (
	IntTy        = abi.IntTy
	UintTy       = abi.UintTy
	BoolTy       = abi.BoolTy
	StringTy     = abi.StringTy
	SliceTy      = abi.SliceTy
	ArrayTy      = abi.ArrayTy
	TupleTy      = abi.TupleTy
	AddressTy    = abi.AddressTy
	FixedBytesTy = abi.FixedBytesTy
	BytesTy      = abi.BytesTy
	HashTy       = abi.HashTy
	FixedPointTy = abi.FixedPointTy
	FunctionTy   = abi.FunctionTy
)

// Type is the reflection of the supported argument type.
type Type = abi.Type

// NewType creates a new reflection type of abi type given in t.
func NewType(t string, internalType string, components []ArgumentMarshaling) (typ Type, err error) {
	return abi.NewType(t, internalType, components)
}
