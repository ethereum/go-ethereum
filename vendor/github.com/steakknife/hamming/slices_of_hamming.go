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

// Int8s hamming distance of two int8 buffers, of which the size of b0
// is used for both (panics if b1 < b0, does not compare b1 beyond length of b0)
func Int8s(b0, b1 []int8) int {
	d := 0
	for i, x := range b0 {
		d += Int8(x, b1[i])
	}
	return d
}

// Int16s hamming distance of two int16 buffers, of which the size of b0
// is used for both (panics if b1 < b0, does not compare b1 beyond length of b0)
func Int16s(b0, b1 []int16) int {
	d := 0
	for i, x := range b0 {
		d += Int16(x, b1[i])
	}
	return d
}

// Int32s hamming distance of two int32 buffers, of which the size of b0
// is used for both (panics if b1 < b0, does not compare b1 beyond length of b0)
func Int32s(b0, b1 []int32) int {
	d := 0
	for i, x := range b0 {
		d += Int32(x, b1[i])
	}
	return d
}

// Int64s hamming distance of two int64 buffers, of which the size of b0
// is used for both (panics if b1 < b0, does not compare b1 beyond length of b0)
func Int64s(b0, b1 []int64) int {
	d := 0
	for i, x := range b0 {
		d += Int64(x, b1[i])
	}
	return d
}

// Ints hamming distance of two int buffers, of which the size of b0
// is used for both (panics if b1 < b0, does not compare b1 beyond length of b0)
func Ints(b0, b1 []int) int {
	d := 0
	for i, x := range b0 {
		d += Int(x, b1[i])
	}
	return d
}

// Uint8s hamming distance of two uint8 buffers, of which the size of b0
// is used for both (panics if b1 < b0, does not compare b1 beyond length of b0)
func Uint8s(b0, b1 []uint8) int {
	d := 0
	for i, x := range b0 {
		d += Uint8(x, b1[i])
	}
	return d
}

// Uint16s hamming distance of two uint16 buffers, of which the size of b0
// is used for both (panics if b1 < b0, does not compare b1 beyond length of b0)
func Uint16s(b0, b1 []uint16) int {
	d := 0
	for i, x := range b0 {
		d += Uint16(x, b1[i])
	}
	return d
}

// Uint32s hamming distance of two uint32 buffers, of which the size of b0
// is used for both (panics if b1 < b0, does not compare b1 beyond length of b0)
func Uint32s(b0, b1 []uint32) int {
	d := 0
	for i, x := range b0 {
		d += Uint32(x, b1[i])
	}
	return d
}

// Uint64s hamming distance of two uint64 buffers, of which the size of b0
// is used for both (panics if b1 < b0, does not compare b1 beyond length of b0)
func Uint64s(b0, b1 []uint64) int {
	d := 0
	for i, x := range b0 {
		d += Uint64(x, b1[i])
	}
	return d
}

// Uints hamming distance of two uint buffers, of which the size of b0
// is used for both (panics if b1 < b0, does not compare b1 beyond length of b0)
func Uints(b0, b1 []uint) int {
	d := 0
	for i, x := range b0 {
		d += Uint(x, b1[i])
	}
	return d
}

// Bytes hamming distance of two byte buffers, of which the size of b0
// is used for both (panics if b1 < b0, does not compare b1 beyond length of b0)
func Bytes(b0, b1 []byte) int {
	d := 0
	for i, x := range b0 {
		d += Byte(x, b1[i])
	}
	return d
}

// Runes hamming distance of two rune buffers, of which the size of b0
// is used for both (panics if b1 < b0, does not compare b1 beyond length of b0)
func Runes(b0, b1 []rune) int {
	d := 0
	for i, x := range b0 {
		d += Rune(x, b1[i])
	}
	return d
}

// Strings hamming distance of two strings, of which the size of b0
// is used for both (panics if b1 < b0, does not compare b1 beyond length of b0)
func Strings(b0, b1 string) int {
	return Runes(runes(b0), runes(b1))
}

// runize string
func runes(s string) (r []rune) {
	for _, ch := range s {
		r = append(r, ch)
	}
	return
}
