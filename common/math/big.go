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

// Package math provides integer math utilities.
package math

import (
	"fmt"
	"math/big"
)

// Various big integer limit values.
var (
	tt256     = BigPow(2, 256)
	tt256m1   = new(big.Int).Sub(tt256, big.NewInt(1))
	MaxBig256 = new(big.Int).Set(tt256m1)
)

const (
	// number of bits in a big.Word
	wordBits = 32 << (uint64(^big.Word(0)) >> 63)
	// number of bytes in a big.Word
	wordBytes = wordBits / 8
)

// HexOrDecimal256 marshals big.Int as hex or decimal.
type HexOrDecimal256 big.Int

// NewHexOrDecimal256 creates a new HexOrDecimal256
func NewHexOrDecimal256(x int64) *HexOrDecimal256 {
	b := big.NewInt(x)
	h := HexOrDecimal256(*b)
	return &h
}

// UnmarshalJSON implements json.Unmarshaler.
//
// It is similar to UnmarshalText, but allows parsing real decimals too, not just
// quoted decimal strings.
func (i *HexOrDecimal256) UnmarshalJSON(input []byte) error {
	if len(input) > 1 && input[0] == '"' {
		input = input[1 : len(input)-1]
	}
	return i.UnmarshalText(input)
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (i *HexOrDecimal256) UnmarshalText(input []byte) error {
	bigint, ok := ParseBig256(string(input))
	if !ok {
		return fmt.Errorf("invalid hex or decimal integer %q", input)
	}
	*i = HexOrDecimal256(*bigint)
	return nil
}

// MarshalText implements encoding.TextMarshaler.
func (i *HexOrDecimal256) MarshalText() ([]byte, error) {
	if i == nil {
		return []byte("0x0"), nil
	}
	return fmt.Appendf(nil, "%#x", (*big.Int)(i)), nil
}

// Decimal256 unmarshals big.Int as a decimal string. When unmarshalling,
// it however accepts either "0x"-prefixed (hex encoded) or non-prefixed (decimal)
type Decimal256 big.Int

// NewDecimal256 creates a new Decimal256
func NewDecimal256(x int64) *Decimal256 {
	b := big.NewInt(x)
	d := Decimal256(*b)
	return &d
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (i *Decimal256) UnmarshalText(input []byte) error {
	bigint, ok := ParseBig256(string(input))
	if !ok {
		return fmt.Errorf("invalid hex or decimal integer %q", input)
	}
	*i = Decimal256(*bigint)
	return nil
}

// MarshalText implements encoding.TextMarshaler.
func (i *Decimal256) MarshalText() ([]byte, error) {
	return []byte(i.String()), nil
}

// String implements Stringer.
func (i *Decimal256) String() string {
	if i == nil {
		return "0"
	}
	return fmt.Sprintf("%#d", (*big.Int)(i))
}

// ParseBig256 parses s as a 256 bit integer in decimal or hexadecimal syntax.
// Leading zeros are accepted. The empty string parses as zero.
func ParseBig256(s string) (*big.Int, bool) {
	if s == "" {
		return new(big.Int), true
	}
	var bigint *big.Int
	var ok bool
	if len(s) >= 2 && (s[:2] == "0x" || s[:2] == "0X") {
		bigint, ok = new(big.Int).SetString(s[2:], 16)
	} else {
		bigint, ok = new(big.Int).SetString(s, 10)
	}
	if ok && bigint.BitLen() > 256 {
		bigint, ok = nil, false
	}
	return bigint, ok
}

// MustParseBig256 parses s as a 256 bit big integer and panics if the string is invalid.
func MustParseBig256(s string) *big.Int {
	v, ok := ParseBig256(s)
	if !ok {
		panic("invalid 256 bit integer: " + s)
	}
	return v
}

// BigPow returns a ** b as a big integer.
func BigPow(a, b int64) *big.Int {
	r := big.NewInt(a)
	return r.Exp(r, big.NewInt(b), nil)
}

// PaddedBigBytes encodes a big integer as a big-endian byte slice. The length
// of the slice is at least n bytes.
func PaddedBigBytes(bigint *big.Int, n int) []byte {
	if bigint.BitLen()/8 >= n {
		return bigint.Bytes()
	}
	ret := make([]byte, n)
	ReadBits(bigint, ret)
	return ret
}

// ReadBits encodes the absolute value of bigint as big-endian bytes. Callers must ensure
// that buf has enough space. If buf is too short the result will be incomplete.
func ReadBits(bigint *big.Int, buf []byte) {
	i := len(buf)
	for _, d := range bigint.Bits() {
		for j := 0; j < wordBytes && i > 0; j++ {
			i--
			buf[i] = byte(d)
			d >>= 8
		}
	}
}

// U256 encodes x as a 256 bit two's complement number. This operation is destructive.
func U256(x *big.Int) *big.Int {
	return x.And(x, tt256m1)
}

// U256Bytes converts a big Int into a 256bit EVM number.
// This operation is destructive.
func U256Bytes(n *big.Int) []byte {
	return PaddedBigBytes(U256(n), 32)
}
