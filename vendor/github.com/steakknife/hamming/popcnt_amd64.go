//
// Package hamming distance calculations in Go
//
// https://github.com/steakknife/hamming
//
// Copyright © 2014, 2015, 2016, 2018 Barry Allard
//
// MIT license
//
package hamming

import "strconv"

// HasPopCnt returns true if *PopCnt functions are callable
func HasPopCnt() (ret bool)

// CountBitsInt8PopCnt count 1's in x
func CountBitsInt8PopCnt(x int8) (ret int)

// CountBitsInt16PopCnt count 1's in x
func CountBitsInt16PopCnt(x int16) (ret int)

// CountBitsInt32PopCnt count 1's in x
func CountBitsInt32PopCnt(x int32) (ret int)

// CountBitsInt64PopCnt count 1's in x
func CountBitsInt64PopCnt(x int64) (ret int)

// CountBitsIntPopCnt count 1's in x
func CountBitsIntPopCnt(x int) int {
	if strconv.IntSize == 64 {
		return CountBitsInt64PopCnt(int64(x))
	} else if strconv.IntSize == 32 {
		return CountBitsInt32PopCnt(int32(x))
	}
	panic("strconv.IntSize must be 32 or 64")
}

// CountBitsUint8PopCnt count 1's in x
func CountBitsUint8PopCnt(x uint8) (ret int)

// CountBitsUint16PopCnt count 1's in x
func CountBitsUint16PopCnt(x uint16) (ret int)

// CountBitsUint32PopCnt count 1's in x
func CountBitsUint32PopCnt(x uint32) (ret int)

// CountBitsUint64PopCnt count 1's in x
func CountBitsUint64PopCnt(x uint64) (ret int)

// CountBitsUintPopCnt count 1's in x
func CountBitsUintPopCnt(x uint) int {
	if strconv.IntSize == 64 {
		return CountBitsUint64PopCnt(uint64(x))
	} else if strconv.IntSize == 32 {
		return CountBitsUint32PopCnt(uint32(x))
	}
	panic("strconv.IntSize must be 32 or 64")
}

// CountBitsBytePopCnt count 1's in x
func CountBitsBytePopCnt(x byte) (ret int)

// CountBitsRunePopCnt count 1's in x
func CountBitsRunePopCnt(x rune) (ret int)
