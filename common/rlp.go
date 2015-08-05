// Copyright 2014 The go-ethereum Authors
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

package common

import (
	"bytes"
	"fmt"
	"math/big"
	"reflect"
)

type RlpEncode interface {
	RlpEncode() []byte
}

type RlpEncodeDecode interface {
	RlpEncode
	RlpValue() []interface{}
}

type RlpEncodable interface {
	RlpData() interface{}
}

func Rlp(encoder RlpEncode) []byte {
	return encoder.RlpEncode()
}

type RlpEncoder struct {
	rlpData []byte
}

func NewRlpEncoder() *RlpEncoder {
	encoder := &RlpEncoder{}

	return encoder
}
func (coder *RlpEncoder) EncodeData(rlpData interface{}) []byte {
	return Encode(rlpData)
}

const (
	RlpEmptyList = 0x80
	RlpEmptyStr  = 0x40
)

const rlpEof = -1

func Char(c []byte) int {
	if len(c) > 0 {
		return int(c[0])
	}

	return rlpEof
}

func DecodeWithReader(reader *bytes.Buffer) interface{} {
	var slice []interface{}

	// Read the next byte
	char := Char(reader.Next(1))
	switch {
	case char <= 0x7f:
		return char

	case char <= 0xb7:
		return reader.Next(int(char - 0x80))

	case char <= 0xbf:
		length := ReadVarInt(reader.Next(int(char - 0xb7)))

		return reader.Next(int(length))

	case char <= 0xf7:
		length := int(char - 0xc0)
		for i := 0; i < length; i++ {
			obj := DecodeWithReader(reader)
			slice = append(slice, obj)
		}

		return slice
	case char <= 0xff:
		length := ReadVarInt(reader.Next(int(char - 0xf7)))
		for i := uint64(0); i < length; i++ {
			obj := DecodeWithReader(reader)
			slice = append(slice, obj)
		}

		return slice
	default:
		panic(fmt.Sprintf("byte not supported: %q", char))
	}

	return slice
}

var (
	directRlp = big.NewInt(0x7f)
	numberRlp = big.NewInt(0xb7)
	zeroRlp   = big.NewInt(0x0)
)

func intlen(i int64) (length int) {
	for i > 0 {
		i = i >> 8
		length++
	}
	return
}

func Encode(object interface{}) []byte {
	var buff bytes.Buffer

	if object != nil {
		switch t := object.(type) {
		case *Value:
			buff.Write(Encode(t.Val))
		case RlpEncodable:
			buff.Write(Encode(t.RlpData()))
		// Code dup :-/
		case int:
			buff.Write(Encode(big.NewInt(int64(t))))
		case uint:
			buff.Write(Encode(big.NewInt(int64(t))))
		case int8:
			buff.Write(Encode(big.NewInt(int64(t))))
		case int16:
			buff.Write(Encode(big.NewInt(int64(t))))
		case int32:
			buff.Write(Encode(big.NewInt(int64(t))))
		case int64:
			buff.Write(Encode(big.NewInt(t)))
		case uint16:
			buff.Write(Encode(big.NewInt(int64(t))))
		case uint32:
			buff.Write(Encode(big.NewInt(int64(t))))
		case uint64:
			buff.Write(Encode(big.NewInt(int64(t))))
		case byte:
			buff.Write(Encode(big.NewInt(int64(t))))
		case *big.Int:
			// Not sure how this is possible while we check for nil
			if t == nil {
				buff.WriteByte(0xc0)
			} else {
				buff.Write(Encode(t.Bytes()))
			}
		case Bytes:
			buff.Write(Encode([]byte(t)))
		case []byte:
			if len(t) == 1 && t[0] <= 0x7f {
				buff.Write(t)
			} else if len(t) < 56 {
				buff.WriteByte(byte(len(t) + 0x80))
				buff.Write(t)
			} else {
				b := big.NewInt(int64(len(t)))
				buff.WriteByte(byte(len(b.Bytes()) + 0xb7))
				buff.Write(b.Bytes())
				buff.Write(t)
			}
		case string:
			buff.Write(Encode([]byte(t)))
		case []interface{}:
			// Inline function for writing the slice header
			WriteSliceHeader := func(length int) {
				if length < 56 {
					buff.WriteByte(byte(length + 0xc0))
				} else {
					b := big.NewInt(int64(length))
					buff.WriteByte(byte(len(b.Bytes()) + 0xf7))
					buff.Write(b.Bytes())
				}
			}

			var b bytes.Buffer
			for _, val := range t {
				b.Write(Encode(val))
			}
			WriteSliceHeader(len(b.Bytes()))
			buff.Write(b.Bytes())
		default:
			// This is how it should have been from the start
			// needs refactoring (@fjl)
			v := reflect.ValueOf(t)
			switch v.Kind() {
			case reflect.Slice:
				var b bytes.Buffer
				for i := 0; i < v.Len(); i++ {
					b.Write(Encode(v.Index(i).Interface()))
				}

				blen := b.Len()
				if blen < 56 {
					buff.WriteByte(byte(blen) + 0xc0)
				} else {
					ilen := byte(intlen(int64(blen)))
					buff.WriteByte(ilen + 0xf7)
					t := make([]byte, ilen)
					for i := byte(0); i < ilen; i++ {
						t[ilen-i-1] = byte(blen >> (i * 8))
					}
					buff.Write(t)
				}
				buff.ReadFrom(&b)
			}
		}
	} else {
		// Empty list for nil
		buff.WriteByte(0xc0)
	}

	return buff.Bytes()
}

// TODO Use a bytes.Buffer instead of a raw byte slice.
// Cleaner code, and use draining instead of seeking the next bytes to read
func Decode(data []byte, pos uint64) (interface{}, uint64) {
	var slice []interface{}
	char := int(data[pos])
	switch {
	case char <= 0x7f:
		return data[pos], pos + 1

	case char <= 0xb7:
		b := uint64(data[pos]) - 0x80

		return data[pos+1 : pos+1+b], pos + 1 + b

	case char <= 0xbf:
		b := uint64(data[pos]) - 0xb7

		b2 := ReadVarInt(data[pos+1 : pos+1+b])

		return data[pos+1+b : pos+1+b+b2], pos + 1 + b + b2

	case char <= 0xf7:
		b := uint64(data[pos]) - 0xc0
		prevPos := pos
		pos++
		for i := uint64(0); i < b; {
			var obj interface{}

			// Get the next item in the data list and append it
			obj, prevPos = Decode(data, pos)
			slice = append(slice, obj)

			// Increment i by the amount bytes read in the previous
			// read
			i += (prevPos - pos)
			pos = prevPos
		}
		return slice, pos

	case char <= 0xff:
		l := uint64(data[pos]) - 0xf7
		b := ReadVarInt(data[pos+1 : pos+1+l])

		pos = pos + l + 1

		prevPos := b
		for i := uint64(0); i < uint64(b); {
			var obj interface{}

			obj, prevPos = Decode(data, pos)
			slice = append(slice, obj)

			i += (prevPos - pos)
			pos = prevPos
		}
		return slice, pos

	default:
		panic(fmt.Sprintf("byte not supported: %q", char))
	}

	return slice, 0
}
