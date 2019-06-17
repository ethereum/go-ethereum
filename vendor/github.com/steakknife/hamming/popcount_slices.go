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

// CountBitsInt8s count 1's in b
func CountBitsInt8s(b []int8) int {
	c := 0
	for _, x := range b {
		c += CountBitsInt8(x)
	}
	return c
}

// CountBitsInt16s count 1's in b
func CountBitsInt16s(b []int16) int {
	c := 0
	for _, x := range b {
		c += CountBitsInt16(x)
	}
	return c
}

// CountBitsInt32s count 1's in b
func CountBitsInt32s(b []int32) int {
	c := 0
	for _, x := range b {
		c += CountBitsInt32(x)
	}
	return c
}

// CountBitsInt64s count 1's in b
func CountBitsInt64s(b []int64) int {
	c := 0
	for _, x := range b {
		c += CountBitsInt64(x)
	}
	return c
}

// CountBitsInts count 1's in b
func CountBitsInts(b []int) int {
	c := 0
	for _, x := range b {
		c += CountBitsInt(x)
	}
	return c
}

// CountBitsUint8s count 1's in b
func CountBitsUint8s(b []uint8) int {
	c := 0
	for _, x := range b {
		c += CountBitsUint8(x)
	}
	return c
}

// CountBitsUint16s count 1's in b
func CountBitsUint16s(b []uint16) int {
	c := 0
	for _, x := range b {
		c += CountBitsUint16(x)
	}
	return c
}

// CountBitsUint32s count 1's in b
func CountBitsUint32s(b []uint32) int {
	c := 0
	for _, x := range b {
		c += CountBitsUint32(x)
	}
	return c
}

// CountBitsUint64s count 1's in b
func CountBitsUint64s(b []uint64) int {
	c := 0
	for _, x := range b {
		c += CountBitsUint64(x)
	}
	return c
}

// CountBitsUints count 1's in b
func CountBitsUints(b []uint) int {
	c := 0
	for _, x := range b {
		c += CountBitsUint(x)
	}
	return c
}

// CountBitsBytes count 1's in b
func CountBitsBytes(b []byte) int {
	c := 0
	for _, x := range b {
		c += CountBitsByte(x)
	}
	return c
}

// CountBitsRunes count 1's in b
func CountBitsRunes(b []rune) int {
	c := 0
	for _, x := range b {
		c += CountBitsRune(x)
	}
	return c
}

// CountBitsString count 1's in s
func CountBitsString(s string) int {
	return CountBitsBytes([]byte(s))
}
