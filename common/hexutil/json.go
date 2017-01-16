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

package hexutil

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strconv"
)

var (
	jsonNull          = []byte("null")
	jsonZero          = []byte(`"0x0"`)
	errNonString      = errors.New("cannot unmarshal non-string as hex data")
	errNegativeBigInt = errors.New("hexutil.Big: can't marshal negative integer")
)

// Bytes marshals/unmarshals as a JSON string with 0x prefix.
// The empty slice marshals as "0x".
type Bytes []byte

// MarshalJSON implements json.Marshaler.
func (b Bytes) MarshalJSON() ([]byte, error) {
	result := make([]byte, len(b)*2+4)
	copy(result, `"0x`)
	hex.Encode(result[3:], b)
	result[len(result)-1] = '"'
	return result, nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (b *Bytes) UnmarshalJSON(input []byte) error {
	raw, err := checkJSON(input)
	if err != nil {
		return err
	}
	dec := make([]byte, len(raw)/2)
	if _, err = hex.Decode(dec, raw); err != nil {
		err = mapError(err)
	} else {
		*b = dec
	}
	return err
}

// String returns the hex encoding of b.
func (b Bytes) String() string {
	return Encode(b)
}

// UnmarshalJSON decodes input as a JSON string with 0x prefix. The length of out
// determines the required input length. This function is commonly used to implement the
// UnmarshalJSON method for fixed-size types:
//
//     type Foo [8]byte
//
//     func (f *Foo) UnmarshalJSON(input []byte) error {
//         return hexutil.UnmarshalJSON("Foo", input, f[:])
//     }
func UnmarshalJSON(typname string, input, out []byte) error {
	raw, err := checkJSON(input)
	if err != nil {
		return err
	}
	if len(raw)/2 != len(out) {
		return fmt.Errorf("hex string has length %d, want %d for %s", len(raw), len(out)*2, typname)
	}
	// Pre-verify syntax before modifying out.
	for _, b := range raw {
		if decodeNibble(b) == badNibble {
			return ErrSyntax
		}
	}
	hex.Decode(out, raw)
	return nil
}

// Big marshals/unmarshals as a JSON string with 0x prefix. The zero value marshals as
// "0x0". Negative integers are not supported at this time. Attempting to marshal them
// will return an error.
type Big big.Int

// MarshalJSON implements json.Marshaler.
func (b *Big) MarshalJSON() ([]byte, error) {
	if b == nil {
		return jsonNull, nil
	}
	bigint := (*big.Int)(b)
	if bigint.Sign() == -1 {
		return nil, errNegativeBigInt
	}
	nbits := bigint.BitLen()
	if nbits == 0 {
		return jsonZero, nil
	}
	enc := make([]byte, 3, (nbits/8)*2+4)
	copy(enc, `"0x`)
	for i := len(bigint.Bits()) - 1; i >= 0; i-- {
		enc = strconv.AppendUint(enc, uint64(bigint.Bits()[i]), 16)
	}
	enc = append(enc, '"')
	return enc, nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (b *Big) UnmarshalJSON(input []byte) error {
	raw, err := checkNumberJSON(input)
	if err != nil {
		return err
	}
	words := make([]big.Word, len(raw)/bigWordNibbles+1)
	end := len(raw)
	for i := range words {
		start := end - bigWordNibbles
		if start < 0 {
			start = 0
		}
		for ri := start; ri < end; ri++ {
			nib := decodeNibble(raw[ri])
			if nib == badNibble {
				return ErrSyntax
			}
			words[i] *= 16
			words[i] += big.Word(nib)
		}
		end = start
	}
	var dec big.Int
	dec.SetBits(words)
	*b = (Big)(dec)
	return nil
}

// ToInt converts b to a big.Int.
func (b *Big) ToInt() *big.Int {
	return (*big.Int)(b)
}

// String returns the hex encoding of b.
func (b *Big) String() string {
	return EncodeBig(b.ToInt())
}

// Uint64 marshals/unmarshals as a JSON string with 0x prefix.
// The zero value marshals as "0x0".
type Uint64 uint64

// MarshalJSON implements json.Marshaler.
func (b Uint64) MarshalJSON() ([]byte, error) {
	buf := make([]byte, 3, 12)
	copy(buf, `"0x`)
	buf = strconv.AppendUint(buf, uint64(b), 16)
	buf = append(buf, '"')
	return buf, nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (b *Uint64) UnmarshalJSON(input []byte) error {
	raw, err := checkNumberJSON(input)
	if err != nil {
		return err
	}
	if len(raw) > 16 {
		return ErrUint64Range
	}
	var dec uint64
	for _, byte := range raw {
		nib := decodeNibble(byte)
		if nib == badNibble {
			return ErrSyntax
		}
		dec *= 16
		dec += uint64(nib)
	}
	*b = Uint64(dec)
	return nil
}

// String returns the hex encoding of b.
func (b Uint64) String() string {
	return EncodeUint64(uint64(b))
}

// Uint marshals/unmarshals as a JSON string with 0x prefix.
// The zero value marshals as "0x0".
type Uint uint

// MarshalJSON implements json.Marshaler.
func (b Uint) MarshalJSON() ([]byte, error) {
	return Uint64(b).MarshalJSON()
}

// UnmarshalJSON implements json.Unmarshaler.
func (b *Uint) UnmarshalJSON(input []byte) error {
	var u64 Uint64
	err := u64.UnmarshalJSON(input)
	if err != nil {
		return err
	} else if u64 > Uint64(^uint(0)) {
		return ErrUintRange
	}
	*b = Uint(u64)
	return nil
}

// String returns the hex encoding of b.
func (b Uint) String() string {
	return EncodeUint64(uint64(b))
}

func isString(input []byte) bool {
	return len(input) >= 2 && input[0] == '"' && input[len(input)-1] == '"'
}

func bytesHave0xPrefix(input []byte) bool {
	return len(input) >= 2 && input[0] == '0' && (input[1] == 'x' || input[1] == 'X')
}

func checkJSON(input []byte) (raw []byte, err error) {
	if !isString(input) {
		return nil, errNonString
	}
	if len(input) == 2 {
		return nil, nil // empty strings are allowed
	}
	if !bytesHave0xPrefix(input[1:]) {
		return nil, ErrMissingPrefix
	}
	input = input[3 : len(input)-1]
	if len(input)%2 != 0 {
		return nil, ErrOddLength
	}
	return input, nil
}

func checkNumberJSON(input []byte) (raw []byte, err error) {
	if !isString(input) {
		return nil, errNonString
	}
	input = input[1 : len(input)-1]
	if len(input) == 0 {
		return nil, nil // empty strings are allowed
	}
	if !bytesHave0xPrefix(input) {
		return nil, ErrMissingPrefix
	}
	input = input[2:]
	if len(input) == 0 {
		return nil, ErrEmptyNumber
	}
	if len(input) > 1 && input[0] == '0' {
		return nil, ErrLeadingZero
	}
	return input, nil
}
