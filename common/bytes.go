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

// Package common contains various helper functions.
package common

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"
	"regexp"
	"strings"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

// BytesToHex converts a byte slice to hexadecimal notation, prefixed with 0x.
func BytesToHex(blob []byte) string {
	return "0x" + hex.EncodeToString(blob)
}

// NumberToHex converts a byte slice to hexadecimal notation, prefixed with 0x,
// with the added rule that the empty slice is encoded as the zero value.
func NumberToHex(blob []byte) string {
	if len(blob) == 0 {
		return "0x0"
	}
	return "0x" + hex.EncodeToString(blob)
}

// FromHex parses a hex string into a byte slice.
func FromHex(s string) []byte {
	// Cut off and optional 0x or 0X prefix
	if len(s) > 2 && (s[:2] == "0x" || s[:2] == "0X") {
		s = s[2:]
	}
	// Pad odd length strings to even ones
	if len(s)%2 == 1 {
		s = "0" + s
	}
	return Hex2Bytes(s)
}

// Number to bytes
//
// Returns the number in bytes with the specified base
func NumberToBytes(num interface{}, bits int) []byte {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, num)
	if err != nil {
		fmt.Println("NumberToBytes failed:", err)
	}

	return buf.Bytes()[buf.Len()-(bits/8):]
}

// Bytes to number
//
// Attempts to cast a byte slice to a unsigned integer
func BytesToNumber(b []byte) uint64 {
	var number uint64

	// Make sure the buffer is 64bits
	data := make([]byte, 8)
	data = append(data[:len(b)], b...)

	buf := bytes.NewReader(data)
	err := binary.Read(buf, binary.BigEndian, &number)
	if err != nil {
		fmt.Println("BytesToNumber failed:", err)
	}

	return number
}

// Read variable int
//
// Read a variable length number in big endian byte order
func ReadVarInt(buff []byte) (ret uint64) {
	switch l := len(buff); {
	case l > 4:
		d := LeftPadBytes(buff, 8)
		binary.Read(bytes.NewReader(d), binary.BigEndian, &ret)
	case l > 2:
		var num uint32
		d := LeftPadBytes(buff, 4)
		binary.Read(bytes.NewReader(d), binary.BigEndian, &num)
		ret = uint64(num)
	case l > 1:
		var num uint16
		d := LeftPadBytes(buff, 2)
		binary.Read(bytes.NewReader(d), binary.BigEndian, &num)
		ret = uint64(num)
	default:
		var num uint8
		binary.Read(bytes.NewReader(buff), binary.BigEndian, &num)
		ret = uint64(num)
	}

	return
}

// Copy bytes
//
// Returns an exact copy of the provided bytes
func CopyBytes(b []byte) (copiedBytes []byte) {
	copiedBytes = make([]byte, len(b))
	copy(copiedBytes, b)

	return
}

// isHexPattern is a regular expression to verify whether a string represents an
// abirarilly long hex number with or without the "0x" prefix.
var isHexPattern = regexp.MustCompile("^0(x|X)([0-9a-fA-F]{2})+$")

// IsHex checks whether a string is a valid hexadecimal number, consisting of
// only hex digits and prefixed with 0x or 0X.
func IsHex(str string) bool {
	return isHexPattern.MatchString(str)
}

func Hex2Bytes(str string) []byte {
	h, err := hex.DecodeString(str)
	if err != nil {
		glog.V(logger.Error).Infof("Invalid hex string to decode: %s: %v", str, err)
	}
	return h
}

func Hex2BytesFixed(str string, flen int) []byte {
	h, err := hex.DecodeString(str)
	if err != nil {
		glog.V(logger.Error).Infof("Invalid hex string to decode: %s: %v", str, err)
	}
	if len(h) == flen {
		return h
	} else {
		if len(h) > flen {
			return h[len(h)-flen : len(h)]
		} else {
			hh := make([]byte, flen)
			copy(hh[flen-len(h):flen], h[:])
			return hh
		}
	}
}

func StringToByteFunc(str string, cb func(str string) []byte) (ret []byte) {
	if len(str) > 1 && (str[:2] == "0x" || str[:2] == "0X") && !strings.Contains(str, "\n") {
		ret = Hex2Bytes(str[2:])
	} else {
		ret = cb(str)
	}

	return
}

func FormatData(data string) []byte {
	if len(data) == 0 {
		return nil
	}
	// Simple stupid
	d := new(big.Int)
	if data[0:1] == "\"" && data[len(data)-1:] == "\"" {
		return RightPadBytes([]byte(data[1:len(data)-1]), 32)
	} else if len(data) > 1 && (data[:2] == "0x" || data[:2] == "0X") {
		d.SetBytes(Hex2Bytes(data[2:]))
	} else {
		d.SetString(data, 0)
	}
	return BigToBytes(d, 256)
}

func ParseData(data ...interface{}) (ret []byte) {
	for _, item := range data {
		switch t := item.(type) {
		case string:
			var str []byte
			if IsHex(t) {
				str = Hex2Bytes(t[2:])
			} else {
				str = []byte(t)
			}

			ret = append(ret, RightPadBytes(str, 32)...)
		case []byte:
			ret = append(ret, LeftPadBytes(t, 32)...)
		}
	}

	return
}

func RightPadBytes(slice []byte, l int) []byte {
	if l < len(slice) {
		return slice
	}

	padded := make([]byte, l)
	copy(padded[0:len(slice)], slice)

	return padded
}

func LeftPadBytes(slice []byte, l int) []byte {
	if l < len(slice) {
		return slice
	}

	padded := make([]byte, l)
	copy(padded[l-len(slice):], slice)

	return padded
}

func LeftPadString(str string, l int) string {
	if l < len(str) {
		return str
	}
	return hex.EncodeToString(make([]byte, (l-len(str))/2)) + str
}

func RightPadString(str string, l int) string {
	if l < len(str) {
		return str
	}
	return str + hex.EncodeToString(make([]byte, (l-len(str))/2))
}

func ToAddress(slice []byte) (addr []byte) {
	if len(slice) < 20 {
		addr = LeftPadBytes(slice, 20)
	} else if len(slice) > 20 {
		addr = slice[len(slice)-20:]
	} else {
		addr = slice
	}

	addr = CopyBytes(addr)

	return
}

func ByteSliceToInterface(slice [][]byte) (ret []interface{}) {
	for _, i := range slice {
		ret = append(ret, i)
	}

	return
}
