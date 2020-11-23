// Copyright 2017 The go-ethereum Authors
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

// Package pot see doc.go
package pot

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

var (
	zerosBin = Address{}.Bin()
)

// Address is an alias for common.Hash
type Address common.Hash

// NewAddressFromBytes constructs an Address from a byte slice
func NewAddressFromBytes(b []byte) Address {
	h := common.Hash{}
	copy(h[:], b)
	return Address(h)
}

func (a Address) IsZero() bool {
	return a.Bin() == zerosBin
}

func (a Address) String() string {
	return fmt.Sprintf("%x", a[:])
}

// MarshalJSON Address serialisation
func (a *Address) MarshalJSON() (out []byte, err error) {
	return []byte(`"` + a.String() + `"`), nil
}

// UnmarshalJSON Address deserialisation
func (a *Address) UnmarshalJSON(value []byte) error {
	*a = Address(common.HexToHash(string(value[1 : len(value)-1])))
	return nil
}

// Bin returns the string form of the binary representation of an address (only first 8 bits)
func (a Address) Bin() string {
	return ToBin(a[:])
}

// ToBin converts a byteslice to the string binary representation
func ToBin(a []byte) string {
	var bs []string
	for _, b := range a {
		bs = append(bs, fmt.Sprintf("%08b", b))
	}
	return strings.Join(bs, "")
}

// Bytes returns the Address as a byte slice
func (a Address) Bytes() []byte {
	return a[:]
}

/*
Proximity(x, y) returns the proximity order of the MSB distance between x and y

The distance metric MSB(x, y) of two equal length byte sequences x an y is the
value of the binary integer cast of the x^y, ie., x and y bitwise xor-ed.
the binary cast is big endian: most significant bit first (=MSB).

Proximity(x, y) is a discrete logarithmic scaling of the MSB distance.
It is defined as the reverse rank of the integer part of the base 2
logarithm of the distance.
It is calculated by counting the number of common leading zeros in the (MSB)
binary representation of the x^y.

(0 farthest, 255 closest, 256 self)
*/
func proximity(one, other Address) (ret int, eq bool) {
	return posProximity(one, other, 0)
}

// posProximity(a, b, pos) returns proximity order of b wrt a (symmetric) pretending
// the first pos bits match, checking only bits index >= pos
func posProximity(one, other Address, pos int) (ret int, eq bool) {
	for i := pos / 8; i < len(one); i++ {
		if one[i] == other[i] {
			continue
		}
		oxo := one[i] ^ other[i]
		start := 0
		if i == pos/8 {
			start = pos % 8
		}
		for j := start; j < 8; j++ {
			if (oxo>>uint8(7-j))&0x01 != 0 {
				return i*8 + j, false
			}
		}
	}
	return len(one) * 8, true
}

// ProxCmp compares the distances a->target and b->target.
// Returns -1 if a is closer to target, 1 if b is closer to target
// and 0 if they are equal.
func ProxCmp(a, x, y interface{}) int {
	return proxCmp(ToBytes(a), ToBytes(x), ToBytes(y))
}

func proxCmp(a, x, y []byte) int {
	for i := range a {
		dx := x[i] ^ a[i]
		dy := y[i] ^ a[i]
		if dx > dy {
			return 1
		} else if dx < dy {
			return -1
		}
	}
	return 0
}

// RandomAddressAt (address, prox) generates a random address
// at proximity order prox relative to address
// if prox is negative a random address is generated
func RandomAddressAt(self Address, prox int) (addr Address) {
	addr = self
	pos := -1
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

// RandomAddress generates a random address
func RandomAddress() Address {
	return RandomAddressAt(Address{}, -1)
}

// NewAddressFromString creates a byte slice from a string in binary representation
func NewAddressFromString(s string) []byte {
	ha := [32]byte{}

	t := s + zerosBin[:len(zerosBin)-len(s)]
	for i := 0; i < 4; i++ {
		n, err := strconv.ParseUint(t[i*64:(i+1)*64], 2, 64)
		if err != nil {
			panic("wrong format: " + err.Error())
		}
		binary.BigEndian.PutUint64(ha[i*8:(i+1)*8], n)
	}
	return ha[:]
}

// BytesAddress is an interface for elements addressable by a byte slice
type BytesAddress interface {
	Address() []byte
}

// ToBytes turns the Val into bytes
func ToBytes(v Val) []byte {
	if v == nil {
		return nil
	}
	b, ok := v.([]byte)
	if !ok {
		ba, ok := v.(BytesAddress)
		if !ok {
			panic(fmt.Sprintf("unsupported value type %T", v))
		}
		b = ba.Address()
	}
	return b
}

// DefaultPof returns a proximity order comparison operator function
// where all
func DefaultPof(max int) func(one, other Val, pos int) (int, bool) {
	return func(one, other Val, pos int) (int, bool) {
		po, eq := proximityOrder(ToBytes(one), ToBytes(other), pos)
		if po >= max {
			eq = true
			po = max
		}
		return po, eq
	}
}

func proximityOrder(one, other []byte, pos int) (int, bool) {
	for i := pos / 8; i < len(one); i++ {
		if one[i] == other[i] {
			continue
		}
		oxo := one[i] ^ other[i]
		start := 0
		if i == pos/8 {
			start = pos % 8
		}
		for j := start; j < 8; j++ {
			if (oxo>>uint8(7-j))&0x01 != 0 {
				return i*8 + j, false
			}
		}
	}
	return len(one) * 8, true
}

// Label displays the node's key in binary format
func Label(v Val) string {
	if v == nil {
		return "<nil>"
	}
	if s, ok := v.(fmt.Stringer); ok {
		return s.String()
	}
	if b, ok := v.([]byte); ok {
		return ToBin(b)
	}
	panic(fmt.Sprintf("unsupported value type %T", v))
}
