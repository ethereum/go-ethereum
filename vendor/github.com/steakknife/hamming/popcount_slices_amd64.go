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

import (
	"strconv"
	"unsafe"
)

// CountBitsInt8sPopCnt count 1's in x
func CountBitsInt8sPopCnt(x []int8) (ret int)

// CountBitsInt16sPopCnt count 1's in x
func CountBitsInt16sPopCnt(x []int16) (ret int)

// CountBitsInt32sPopCnt count 1's in x
func CountBitsInt32sPopCnt(x []int32) (ret int)

// CountBitsInt64sPopCnt count 1's in x
func CountBitsInt64sPopCnt(x []int64) (ret int)

// CountBitsIntsPopCnt count 1's in x
func CountBitsIntsPopCnt(x []int) int {
	if strconv.IntSize == 64 {
		y := (*[]int64)(unsafe.Pointer(&x)) // #nosec G103
		return CountBitsInt64sPopCnt(*y)
	} else if strconv.IntSize == 32 {
		y := (*[]int32)(unsafe.Pointer(&x)) // #nosec G103
		return CountBitsInt32sPopCnt(*y)
	}
	panic("strconv.IntSize must be 32 or 64 bits")
}

// CountBitsUint8sPopCnt count 1's in x
func CountBitsUint8sPopCnt(x []uint8) (ret int)

// CountBitsUint16sPopCnt count 1's in x
func CountBitsUint16sPopCnt(x []uint16) (ret int)

// CountBitsUint32sPopCnt count 1's in x
func CountBitsUint32sPopCnt(x []uint32) (ret int)

// CountBitsUint64sPopCnt count 1's in x
func CountBitsUint64sPopCnt(x []uint64) (ret int)

// CountBitsUintsPopCnt count 1's in x
func CountBitsUintsPopCnt(x []uint) int {
	if strconv.IntSize == 64 {
		y := (*[]uint64)(unsafe.Pointer(&x)) // #nosec G103
		return CountBitsUint64sPopCnt(*y)
	} else if strconv.IntSize == 32 {
		y := (*[]uint32)(unsafe.Pointer(&x)) // #nosec G103
		return CountBitsUint32sPopCnt(*y)
	}
	panic("strconv.IntSize must be 32 or 64 bits")
}

// CountBitsBytesPopCnt count 1's in x
func CountBitsBytesPopCnt(x []byte) (ret int)

// CountBitsRunesPopCnt count 1's in x
func CountBitsRunesPopCnt(x []rune) (ret int)

// CountBitsStringPopCnt count 1's in s
func CountBitsStringPopCnt(s string) (ret int)
