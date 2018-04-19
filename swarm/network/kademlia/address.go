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
	"fmt"
	"math/rand"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

type Address common.Hash

func (a Address) String() string {
	return fmt.Sprintf("%x", a[:])
}

func (a *Address) MarshalJSON() (out []byte, err error) {
	return []byte(`"` + a.String() + `"`), nil
}

func (a *Address) UnmarshalJSON(value []byte) error {
	*a = Address(common.HexToHash(string(value[1 : len(value)-1])))
	return nil
}

// the string form of the binary representation of an address (only first 8 bits)
func (a Address) Bin() string {
	var bs []string
	for _, b := range a[:] {
		bs = append(bs, fmt.Sprintf("%08b", b))
	}
	return strings.Join(bs, "")
}

/*
Proximity(x, y) returns the proximity order of the MSB distance between x and y

The distance metric MSB(x, y) of two equal length byte sequences x and y is the
value of the binary integer cast of the x^y, ie., x and y bitwise xor-ed.
the binary cast is big endian: most significant bit first (=MSB).

Proximity(x, y) is a discrete logarithmic scaling of the MSB distance.
It is defined as the reverse rank of the integer part of the base 2
logarithm of the distance.
It is calculated by counting the number of common leading zeros in the (MSB)
binary representation of the x^y.

(0 farthest, 255 closest, 256 self)
*/
func proximity(one, other Address) (ret int) {
	for i := 0; i < len(one); i++ {
		oxo := one[i] ^ other[i]
		for j := 0; j < 8; j++ {
			if (oxo>>uint8(7-j))&0x01 != 0 {
				return i*8 + j
			}
		}
	}
	return len(one) * 8
}

// Address.ProxCmp compares the distances a->target and b->target.
// Returns -1 if a is closer to target, 1 if b is closer to target
// and 0 if they are equal.
func (target Address) ProxCmp(a, b Address) int {
	for i := range target {
		da := a[i] ^ target[i]
		db := b[i] ^ target[i]
		if da > db {
			return 1
		} else if da < db {
			return -1
		}
	}
	return 0
}

// randomAddressAt(address, prox) generates a random address
// at proximity order prox relative to address
// if prox is negative a random address is generated
func RandomAddressAt(self Address, prox int) (addr Address) {
	addr = self
	var pos int
	if prox >= 0 {
		pos = prox / 8
		trans := prox % 8
		transbytea := byte(0)
		for j := 0; j <= trans; j++ {
			transbytea |= 1 << uint8(7-j)
		}
		flipbyte := byte(1 << uint8(7-trans))
		transbyteb := transbytea ^ byte(255)
		randbyte := byte(rand.Intn(255))
		addr[pos] = ((addr[pos] & transbytea) ^ flipbyte) | randbyte&transbyteb
	}
	for i := pos + 1; i < len(addr); i++ {
		addr[i] = byte(rand.Intn(255))
	}

	return
}

// KeyRange(a0, a1, proxLimit) returns the address inclusive address
// range that contain addresses closer to one than other
func KeyRange(one, other Address, proxLimit int) (start, stop Address) {
	prox := proximity(one, other)
	if prox >= proxLimit {
		prox = proxLimit
	}
	start = CommonBitsAddrByte(one, other, byte(0x00), prox)
	stop = CommonBitsAddrByte(one, other, byte(0xff), prox)
	return
}

func CommonBitsAddrF(self, other Address, f func() byte, p int) (addr Address) {
	prox := proximity(self, other)
	var pos int
	if p <= prox {
		prox = p
	}
	pos = prox / 8
	addr = self
	trans := byte(prox % 8)
	var transbytea byte
	if p > prox {
		transbytea = byte(0x7f)
	} else {
		transbytea = byte(0xff)
	}
	transbytea >>= trans
	transbyteb := transbytea ^ byte(0xff)
	addrpos := addr[pos]
	addrpos &= transbyteb
	if p > prox {
		addrpos ^= byte(0x80 >> trans)
	}
	addrpos |= transbytea & f()
	addr[pos] = addrpos
	for i := pos + 1; i < len(addr); i++ {
		addr[i] = f()
	}

	return
}

func CommonBitsAddr(self, other Address, prox int) (addr Address) {
	return CommonBitsAddrF(self, other, func() byte { return byte(rand.Intn(255)) }, prox)
}

func CommonBitsAddrByte(self, other Address, b byte, prox int) (addr Address) {
	return CommonBitsAddrF(self, other, func() byte { return b }, prox)
}

// randomAddressAt() generates a random address
func RandomAddress() Address {
	return RandomAddressAt(Address{}, -1)
}
