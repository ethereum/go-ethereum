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
	"bytes"
	"math/big"
	"reflect"
	"testing"
)

func TestNumberTypes(t *testing.T) {
	ubytes := make([]byte, 32)
	ubytes[31] = 1
	sbytesmin := []byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}

	unsigned := U256(big.NewInt(1))
	if !bytes.Equal(unsigned, ubytes) {
		t.Error("expected %x got %x", ubytes, unsigned)
	}

	signed := S256(big.NewInt(1))
	if !bytes.Equal(signed, ubytes) {
		t.Error("expected %x got %x", ubytes, unsigned)
	}

	signed = S256(big.NewInt(-1))
	if !bytes.Equal(signed, sbytesmin) {
		t.Error("expected %x got %x", ubytes, unsigned)
	}
}

func TestPackNumber(t *testing.T) {
	ubytes := make([]byte, 32)
	ubytes[31] = 1
	sbytesmin := []byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	maxunsigned := []byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255}

	packed := packNum(reflect.ValueOf(1), IntTy)
	if !bytes.Equal(packed, ubytes) {
		t.Errorf("expected %x got %x", ubytes, packed)
	}
	packed = packNum(reflect.ValueOf(-1), IntTy)
	if !bytes.Equal(packed, sbytesmin) {
		t.Errorf("expected %x got %x", ubytes, packed)
	}
	packed = packNum(reflect.ValueOf(1), UintTy)
	if !bytes.Equal(packed, ubytes) {
		t.Errorf("expected %x got %x", ubytes, packed)
	}
	packed = packNum(reflect.ValueOf(-1), UintTy)
	if !bytes.Equal(packed, maxunsigned) {
		t.Errorf("expected %x got %x", maxunsigned, packed)
	}

	packed = packNum(reflect.ValueOf("string"), UintTy)
	if packed != nil {
		t.Errorf("expected 'string' to pack to nil. got %x instead", packed)
	}
}

func TestSigned(t *testing.T) {
	if isSigned(reflect.ValueOf(uint(10))) {
		t.Error()
	}

	if !isSigned(reflect.ValueOf(int(10))) {
		t.Error()
	}

	if !isSigned(reflect.ValueOf(big.NewInt(10))) {
		t.Error()
	}
}
