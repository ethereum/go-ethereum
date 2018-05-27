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

package kademlia

import (
	"math/rand"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func (Address) Generate(rand *rand.Rand, size int) reflect.Value {
	var id Address
	for i := 0; i < len(id); i++ {
		id[i] = byte(uint8(rand.Intn(255)))
	}
	return reflect.ValueOf(id)
}

func TestCommonBitsAddrF(t *testing.T) {
	a := Address(common.HexToHash("0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"))
	b := Address(common.HexToHash("0x8123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"))
	c := Address(common.HexToHash("0x4123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"))
	d := Address(common.HexToHash("0x0023456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"))
	e := Address(common.HexToHash("0x01A3456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"))
	ab := CommonBitsAddrF(a, b, func() byte { return byte(0x00) }, 10)
	expab := Address(common.HexToHash("0x8000000000000000000000000000000000000000000000000000000000000000"))

	if ab != expab {
		t.Fatalf("%v != %v", ab, expab)
	}
	ac := CommonBitsAddrF(a, c, func() byte { return byte(0x00) }, 10)
	expac := Address(common.HexToHash("0x4000000000000000000000000000000000000000000000000000000000000000"))

	if ac != expac {
		t.Fatalf("%v != %v", ac, expac)
	}
	ad := CommonBitsAddrF(a, d, func() byte { return byte(0x00) }, 10)
	expad := Address(common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"))

	if ad != expad {
		t.Fatalf("%v != %v", ad, expad)
	}
	ae := CommonBitsAddrF(a, e, func() byte { return byte(0x00) }, 10)
	expae := Address(common.HexToHash("0x0180000000000000000000000000000000000000000000000000000000000000"))

	if ae != expae {
		t.Fatalf("%v != %v", ae, expae)
	}
	acf := CommonBitsAddrF(a, c, func() byte { return byte(0xff) }, 10)
	expacf := Address(common.HexToHash("0x7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"))

	if acf != expacf {
		t.Fatalf("%v != %v", acf, expacf)
	}
	aeo := CommonBitsAddrF(a, e, func() byte { return byte(0x00) }, 2)
	expaeo := Address(common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"))

	if aeo != expaeo {
		t.Fatalf("%v != %v", aeo, expaeo)
	}
	aep := CommonBitsAddrF(a, e, func() byte { return byte(0xff) }, 2)
	expaep := Address(common.HexToHash("0x3fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"))

	if aep != expaep {
		t.Fatalf("%v != %v", aep, expaep)
	}

}

func TestRandomAddressAt(t *testing.T) {
	var a Address
	for i := 0; i < 100; i++ {
		a = RandomAddress()
		prox := rand.Intn(255)
		b := RandomAddressAt(a, prox)
		if proximity(a, b) != prox {
			t.Fatalf("incorrect address prox(%v, %v) == %v (expected %v)", a, b, proximity(a, b), prox)
		}
	}
}
