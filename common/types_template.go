// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// go-ethereum is free software: you can redistribute it and/or modify
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

// +build none
//sed -e 's/_N_/Hash/g' -e 's/_S_/32/g' -e '1d' types_template.go | gofmt -w hash.go

package common

import "math/big"

type _N_ [_S_]byte

func BytesTo_N_(b []byte) _N_ {
	var h _N_
	h.SetBytes(b)
	return h
}
func StringTo_N_(s string) _N_ { return BytesTo_N_([]byte(s)) }
func BigTo_N_(b *big.Int) _N_  { return BytesTo_N_(b.Bytes()) }
func HexTo_N_(s string) _N_    { return BytesTo_N_(FromHex(s)) }

// Don't use the default 'String' method in case we want to overwrite

// Get the string representation of the underlying hash
func (h _N_) Str() string   { return string(h[:]) }
func (h _N_) Bytes() []byte { return h[:] }
func (h _N_) Big() *big.Int { return Bytes2Big(h[:]) }
func (h _N_) Hex() string   { return "0x" + Bytes2Hex(h[:]) }

// Sets the hash to the value of b. If b is larger than len(h) it will panic
func (h *_N_) SetBytes(b []byte) {
	// Use the right most bytes
	if len(b) > len(h) {
		b = b[len(b)-_S_:]
	}

	// Reverse the loop
	for i := len(b) - 1; i >= 0; i-- {
		h[_S_-len(b)+i] = b[i]
	}
}

// Set string `s` to h. If s is larger than len(h) it will panic
func (h *_N_) SetString(s string) { h.SetBytes([]byte(s)) }

// Sets h to other
func (h *_N_) Set(other _N_) {
	for i, v := range other {
		h[i] = v
	}
}
