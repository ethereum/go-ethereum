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

// Int8 hamming distance of two int8's
func Int8(x, y int8) int {
	return CountBitsInt8(x ^ y)
}

// Int16 hamming distance of two int16's
func Int16(x, y int16) int {
	return CountBitsInt16(x ^ y)
}

// Int32 hamming distance of two int32's
func Int32(x, y int32) int {
	return CountBitsInt32(x ^ y)
}

// Int64 hamming distance of two int64's
func Int64(x, y int64) int {
	return CountBitsInt64(x ^ y)
}

// Int hamming distance of two ints
func Int(x, y int) int {
	return CountBitsInt(x ^ y)
}

// Uint8 hamming distance of two uint8's
func Uint8(x, y uint8) int {
	return CountBitsUint8(x ^ y)
}

// Uint16 hamming distance of two uint16's
func Uint16(x, y uint16) int {
	return CountBitsUint16(x ^ y)
}

// Uint32 hamming distance of two uint32's
func Uint32(x, y uint32) int {
	return CountBitsUint32(x ^ y)
}

// Uint64 hamming distance of two uint64's
func Uint64(x, y uint64) int {
	return CountBitsUint64(x ^ y)
}

// Uint hamming distance of two uint's
func Uint(x, y uint) int {
	return CountBitsUint(x ^ y)
}

// Byte hamming distance of two bytes
func Byte(x, y byte) int {
	return CountBitsByte(x ^ y)
}

// Rune hamming distance of two runes
func Rune(x, y rune) int {
	return CountBitsRune(x ^ y)
}
